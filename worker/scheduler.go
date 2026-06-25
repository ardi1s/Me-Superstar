// worker 包管理后台定时任务，如每小时同步所有已授权账号的作品数据。
package worker

import (
	"log"

	"agent-backend/models"
	"agent-backend/services"

	"github.com/robfig/cron/v3"
)

// StartScheduler 启动 cron 调度器，注册周期性任务。
// 当前注册任务：
//   - 每小时执行一次全量账号作品与数据同步。
func StartScheduler() {
	c := cron.New(cron.WithSeconds()) // 支持秒级表达式

	// 每小时同步一次所有账号
	_, err := c.AddFunc("0 0 * * * *", func() {
		log.Println("[调度] 开始执行账号数据同步任务...")
		syncAllAccounts()
		log.Println("[调度] 本轮账号数据同步任务完成")
	})
	if err != nil {
		log.Fatalf("[调度] 注册定时任务失败: %v", err)
	}

	c.Start()
	log.Println("[调度] Cron 调度器已启动，每小时同步一次账号数据")
}

// syncAllAccounts 遍历所有有效的 accounts，依次调用 SyncAccountData。
func syncAllAccounts() {
	var accounts []models.Account
	if err := models.DB.Find(&accounts).Error; err != nil {
		log.Printf("[调度] 查询账号列表失败: %v", err)
		return
	}

	if len(accounts) == 0 {
		log.Println("[调度] 没有待同步的账号，跳过")
		return
	}

	log.Printf("[调度] 共发现 %d 个账号待同步", len(accounts))

	for _, account := range accounts {
		// 每个账号独立的 goroutine，避免单账号阻塞整体
		go func(accID uint) {
			if err := services.SyncAccountData(accID); err != nil {
				log.Printf("[调度] 账号 %d 同步失败: %v", accID, err)
			}
		}(account.ID)
	}
}
