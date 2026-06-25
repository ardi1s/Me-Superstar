// agent-backend 基于 Gin + GORM 的 Web 后端服务。
// 入口文件：加载配置 → 初始化数据库 → 自动迁移 → 启动定时任务 → 注册路由 → 启动 HTTP 服务。
package main

import (
	"fmt"
	"log"

	"agent-backend/config"
	"agent-backend/models"
	"agent-backend/router"
	"agent-backend/worker"
)

func main() {
	// 1. 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化数据库
	models.InitDB()

	// 3. 自动迁移
	models.AutoMigrate()

	// 4. 启动后台定时任务（每小时同步作品及数据）
	worker.StartScheduler()

	// 5. 创建 Gin 引擎并注册路由
	r := router.SetupRouter()

	// 6. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
