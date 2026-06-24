// middleware 包定义 Gin 中间件，如 JWT 认证中间件。
package middleware

import (
	"net/http"
	"strings"

	"agent-backend/config"
	"agent-backend/handlers"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth 是 Gin 中间件，从 Authorization 头中提取并校验 Bearer JWT Token。
// 校验通过后将 user_id 和 username 注入 context，后续 handler 可直接通过 c.Get 获取。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少 Authorization 头",
			})
			c.Abort()
			return
		}

		// Authorization 头格式：Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization 格式错误，需要 Bearer token",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &handlers.Claims{}

		// 解析并校验 JWT
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Token 无效或已过期",
			})
			c.Abort()
			return
		}

		// 将用户信息注入 context，后续 handler 可直接使用
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
