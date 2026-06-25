// services 包封装业务逻辑层，负责编排数据拉取、转换、入库等操作。
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"agent-backend/config"
	"agent-backend/models"
	"agent-backend/pkg/douyin"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncAccountData 同步指定账号的作品及昨日数据。
//
// 流程：
//  1. 加载账号信息
//  2. 调抖音 GetWorks 拉取最近 50 条作品
//  3. 若 token 过期，用 refresh_token 刷新并重试
//  4. 将作品 upsert 到 works 表
//  5. 对每个作品拉取当前指标，计算昨日增量，写入 work_daily_stats
func SyncAccountData(accountID uint) error {
	// 1. 加载账号
	var account models.Account
	if err := models.DB.First(&account, accountID).Error; err != nil {
		return fmt.Errorf("加载账号 %d 失败: %w", accountID, err)
	}

	cfg := config.AppConfig
	client := douyin.NewClient(cfg.Douyin.ClientKey, cfg.Douyin.ClientSecret, cfg.Douyin.RedirectURI)

	// 2. 拉取作品（含 token 过期刷新重试）
	works, err := fetchWorksWithTokenRefresh(client, &account)
	if err != nil {
		return fmt.Errorf("拉取账号 %d 作品失败: %w", accountID, err)
	}

	log.Printf("[同步] 账号 %d (open_id=%s) 拉取到 %d 件作品", accountID, account.PlatformAccountID, len(works))

	// 3. 作品入库（存在则更新标题/封面等字段）
	for _, item := range works {
		publishTime := time.Unix(item.CreateTime, 0)
		work := models.Work{
			AccountID:   account.ID,
			WorkID:      item.ItemID,
			Title:       item.Title,
			CoverURL:    item.Cover,
			PublishTime: publishTime,
		}

		// Upsert：联合唯一索引匹配则更新
		if err := models.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}, {Name: "work_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"title", "cover_url", "publish_time"}),
		}).Create(&work).Error; err != nil {
			log.Printf("[同步] 作品 %s 入库失败: %v", item.ItemID, err)
			continue
		}

		// 4. 同步昨日数据指标
		if err := syncYesterdayStats(client, &account, item.ItemID); err != nil {
			log.Printf("[同步] 作品 %s 昨日数据同步失败: %v", item.ItemID, err)
		}
	}

	return nil
}

// fetchWorksWithTokenRefresh 调用 GetWorks，token 过期时自动刷新并重试一次。
func fetchWorksWithTokenRefresh(client *douyin.Client, account *models.Account) ([]douyin.WorkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	items, err := client.GetWorks(ctx, account.AccessToken, account.PlatformAccountID)
	if err == nil {
		return items, nil
	}

	// 非 token 过期错误，直接返回
	if _, ok := err.(*douyin.TokenExpiredError); !ok {
		return nil, err
	}

	log.Printf("[同步] 账号 %d token 过期，尝试刷新...", account.ID)

	// 刷新 token
	refreshData, refreshErr := client.RefreshAccessToken(account.RefreshToken)
	if refreshErr != nil {
		return nil, fmt.Errorf("刷新 token 失败: %w", refreshErr)
	}

	// 更新 accounts 表
	models.DB.Model(account).Updates(map[string]interface{}{
		"access_token":  refreshData.AccessToken,
		"refresh_token": refreshData.RefreshToken,
		"expires_at":    time.Now().Add(time.Duration(refreshData.ExpiresIn) * time.Second),
	})

	// 内存中也更新，供后续调用使用
	account.AccessToken = refreshData.AccessToken
	account.RefreshToken = refreshData.RefreshToken

	// 用新 token 重试
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	return client.GetWorks(ctx2, account.AccessToken, account.PlatformAccountID)
}

// syncYesterdayStats 同步指定作品昨天的数据指标。
//
// 抖音作品数据 API 返回的是累计值（如累计播放、累计点赞等），
// 我们需要的是每日增量。因此策略为：
//   - 获取作品当前的累计指标
//   - 查找已存储的最近一天记录，计算 delta = 当天累计 − 前一天累计
//   - 将昨日增量写入 work_daily_stats
func syncYesterdayStats(client *douyin.Client, account *models.Account, workID string) error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 检查是否已存在昨日记录（幂等：已存在则跳过）
	var existing models.WorkDailyStats
	err := models.DB.Where("work_id = ? AND stat_date = ?", workID, yesterday).First(&existing).Error
	if err == nil {
		return nil // 已有，跳过
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询昨日记录失败: %w", err)
	}

	// 获取当前累计指标（含 token 过期重试）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, err := client.GetItemData(ctx, account.AccessToken, account.PlatformAccountID, workID)
	if err != nil {
		// token 过期则刷新重试一次
		if _, ok := err.(*douyin.TokenExpiredError); ok {
			refreshData, refreshErr := client.RefreshAccessToken(account.RefreshToken)
			if refreshErr != nil {
				return fmt.Errorf("刷新 token 失败: %w", refreshErr)
			}
			models.DB.Model(account).Updates(map[string]interface{}{
				"access_token":  refreshData.AccessToken,
				"refresh_token": refreshData.RefreshToken,
				"expires_at":    time.Now().Add(time.Duration(refreshData.ExpiresIn) * time.Second),
			})
			account.AccessToken = refreshData.AccessToken
			account.RefreshToken = refreshData.RefreshToken

			ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel2()
			metrics, err = client.GetItemData(ctx2, account.AccessToken, account.PlatformAccountID, workID)
		}
		if err != nil {
			return fmt.Errorf("获取作品指标失败: %w", err)
		}
	}

	// 查找最近一条历史记录，用于计算增量
	var prevStats models.WorkDailyStats
	prevErr := models.DB.Where("work_id = ?", workID).Order("stat_date DESC").First(&prevStats).Error

	// 计算增量（若无历史记录，则增量为当前累计值本身）
	deltaPlay := int(metrics.PlayCount)
	deltaLike := int(metrics.LikeCount)
	deltaComment := int(metrics.CommentCount)
	deltaShare := int(metrics.ShareCount)
	deltaFans := 0 // 作品维度粉丝增量由外部 GetWorkFansDelta 提供

	if prevErr == nil {
		deltaPlay = deltaPlay - prevStats.PlayCount
		deltaLike = deltaLike - prevStats.LikeCount
		deltaComment = deltaComment - prevStats.CommentCount
		deltaShare = deltaShare - prevStats.ShareCount
		if deltaPlay < 0 {
			deltaPlay = 0
		}
		if deltaLike < 0 {
			deltaLike = 0
		}
		if deltaComment < 0 {
			deltaComment = 0
		}
		if deltaShare < 0 {
			deltaShare = 0
		}
	}

	// 写入昨日增量（非累计值）
	stats := models.WorkDailyStats{
		WorkID:       workID,
		StatDate:     yesterday,
		FansAdded:    deltaFans,
		PlayCount:    deltaPlay,
		LikeCount:    deltaLike,
		CommentCount: deltaComment,
		ShareCount:   deltaShare,
	}

	if err := models.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "work_id"}, {Name: "stat_date"}},
		DoNothing: true,
	}).Create(&stats).Error; err != nil {
		return fmt.Errorf("写入昨日统计失败: %w", err)
	}

	log.Printf("[统计] 作品 %s 昨日(%s) 播放+%d 点赞+%d 评论+%d 分享+%d",
		workID, yesterday, deltaPlay, deltaLike, deltaComment, deltaShare)

	return nil
}
