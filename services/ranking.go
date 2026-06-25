// services 包封装业务逻辑层。
//
// 本文件实现作品排行查询：按时间段聚合 work_daily_stats 的涨粉数据，
// 关联 works 表，返回排序后的作品列表。
package services

import (
	"fmt"
	"time"

	"agent-backend/models"
)

// TopWorkItem 排行榜单条作品信息。
type TopWorkItem struct {
	WorkID         string `json:"work_id"`
	Title          string `json:"title"`
	CoverURL       string `json:"cover_url"`
	PublishTime    string `json:"publish_time"`
	TotalFansAdded int    `json:"total_fans_added"`
	TotalPlayCount int    `json:"total_play_count"`
	TotalLikeCount int    `json:"total_like_count"`
}

// TopWorksQuery 作品排行查询参数。
type TopWorksQuery struct {
	AccountID uint // 限定账号，0 表示不限制
	Period    string
	Page      int
	PageSize  int
}

// TopWorksResult 排行查询结果（含分页信息）。
type TopWorksResult struct {
	Items    []TopWorkItem `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// GetTopWorksByFans 根据时间段聚合涨粉数据，返回作品涨粉排行榜。
//
// period 取值：
//   - "1d"  仅昨天
//   - "7d"  最近 7 天（昨天往回 6 天）
//   - "30d" 最近 30 天
func GetTopWorksByFans(query TopWorksQuery) (*TopWorksResult, error) {
	startDate, endDate := parsePeriod(query.Period)

	// 子查询：按 work_id 聚合指定时间段内的指标增量
	subQuery := models.DB.Table("work_daily_stats").
		Select(`work_id,
			SUM(fans_added)    AS total_fans,
			SUM(play_count)    AS total_play,
			SUM(like_count)    AS total_like,
			SUM(comment_count) AS total_comment,
			SUM(share_count)   AS total_share`).
		Where("stat_date >= ? AND stat_date <= ?", startDate, endDate).
		Group("work_id")

	// 主查询：关联 works 表，按涨粉降序
	baseQuery := models.DB.Table("works w").
		Select(`w.work_id, w.title, w.cover_url, w.publish_time,
			COALESCE(stats.total_fans, 0)    AS total_fans_added,
			COALESCE(stats.total_play, 0)    AS total_play_count,
			COALESCE(stats.total_like, 0)    AS total_like_count`).
		Joins("LEFT JOIN (?) stats ON w.work_id = stats.work_id", subQuery).
		Where("stats.total_fans > 0") // 只展示有涨粉数据的作品

	if query.AccountID > 0 {
		baseQuery = baseQuery.Where("w.account_id = ?", query.AccountID)
	}

	// 总数
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询排行总数失败: %w", err)
	}

	// 分页 + 排序
	var items []TopWorkItem
	offset := (query.Page - 1) * query.PageSize
	if err := baseQuery.
		Order("total_fans_added DESC").
		Limit(query.PageSize).
		Offset(offset).
		Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("查询排行列表失败: %w", err)
	}

	return &TopWorksResult{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// parsePeriod 根据 period 参数返回起止日期（仅日期部分，不含时间）。
func parsePeriod(period string) (start, end string) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	switch period {
	case "1d":
		return yesterday, yesterday
	case "7d":
		return now.AddDate(0, 0, -7).Format("2006-01-02"), yesterday
	case "30d":
		return now.AddDate(0, 0, -30).Format("2006-01-02"), yesterday
	default:
		return now.AddDate(0, 0, -7).Format("2006-01-02"), yesterday
	}
}
