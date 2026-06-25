// router 包负责集中管理所有 Gin 路由的注册，供 main 和 test 复用。
package router

import (
	"agent-backend/handlers"
	"agent-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 创建并配置 Gin 路由（供 main 和 test 复用）。
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		// ---- 公开接口（无需认证）----

		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)

			// 抖音 OAuth 授权跳转 —— 前端 window.open 用 ?token=xxx 传认证
			auth.GET("/douyin", handlers.DouyinAuthRedirect)

			// 抖音 OAuth 回调 —— 由抖音服务器回调，无法携带 JWT，
			// 通过 state 参数编码 user_id 来关联用户。
			auth.GET("/douyin/callback", handlers.DouyinCallback)
		}

		// ---- 需要认证的接口 ----

		protected := api.Group("/")
		protected.Use(middleware.JWTAuth())
		{
			// 用户信息
			protected.GET("/profile", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				username, _ := c.Get("username")
				c.JSON(200, gin.H{
					"code":     200,
					"user_id":  userID,
					"username": username,
				})
			})

			// 已授权账号列表
			protected.GET("/accounts", handlers.ListAccounts)

			// 作品涨粉排行榜
			protected.GET("/works/top-fans", handlers.GetTopWorksByFans)
		}
	}

	return r
}
