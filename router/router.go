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
		// 公开接口
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
		}

		// 需要认证的接口
		protected := api.Group("/")
		protected.Use(middleware.JWTAuth())
		{
			protected.GET("/profile", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				username, _ := c.Get("username")
				c.JSON(200, gin.H{
					"code":     200,
					"user_id":  userID,
					"username": username,
				})
			})
		}
	}

	return r
}
