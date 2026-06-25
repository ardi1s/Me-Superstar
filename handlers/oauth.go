// handlers 包定义 HTTP 请求处理函数（注册、登录等）。
//
// 本文件实现抖音 OAuth 授权流程：
//   - GET /api/v1/auth/douyin         → 生成授权链接并重定向
//   - GET /api/v1/auth/douyin/callback → 接收 code，换取 token，入库
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"agent-backend/config"
	"agent-backend/models"
	"agent-backend/pkg/douyin"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// stateClaims 用于编码 OAuth state 参数的自定义 JWT 载荷。
// state 中包含发起授权的用户 ID，回调时解码以关联用户。
type stateClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// DouyinScope 抖音 OAuth 默认授权权限范围。
const DouyinScope = "user_info,fans.list,video.list"

// DouyinAuthRedirect 生成抖音 OAuth 授权链接并重定向。
// 支持两种认证方式：
//   - Bearer Token（通过 JWT 中间件，c.Get("user_id")）
//   - URL query 参数 ?token=xxx（供前端 window.open 使用，无法设 Header 时）
//
// 路由：GET /api/v1/auth/douyin
func DouyinAuthRedirect(c *gin.Context) {
	cfg := config.AppConfig
	if cfg.Douyin.ClientKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "抖音开放平台未配置",
		})
		return
	}

	// 优先从 JWT 中间件取 user_id；若无（如未经过中间件），尝试从 ?token 参数解析
	uid, exists := c.Get("user_id")
	if !exists {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "缺少认证信息"})
			return
		}
		claims := &Claims{}
		parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil || !parsed.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token 无效"})
			return
		}
		uid = claims.UserID
	}

	// 将 user_id 编码到 state 参数中，回调时解析以关联用户
	state, err := generateStateToken(uid.(uint), cfg.JWT.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成 state 失败",
		})
		return
	}

	client := douyin.NewClient(cfg.Douyin.ClientKey, cfg.Douyin.ClientSecret, cfg.Douyin.RedirectURI)
	authURL := client.GenerateAuthURL(state, DouyinScope)

	c.Redirect(http.StatusFound, authURL)
}

// DouyinCallback 抖音 OAuth 回调处理。
// 路由：GET /api/v1/auth/douyin/callback（公开，由抖音服务器回调）
//
// 流程：解析 state 获取 user_id → 用 code 换取 access_token → 存入 accounts 表。
func DouyinCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 code 参数",
		})
		return
	}

	// 1. 从 state 中解析出用户 ID
	cfg := config.AppConfig
	userID, err := parseStateToken(state, cfg.JWT.Secret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "state 参数无效或已过期: " + err.Error(),
		})
		return
	}

	// 2. 调用抖音 API 换取 access_token
	client := douyin.NewClient(cfg.Douyin.ClientKey, cfg.Douyin.ClientSecret, cfg.Douyin.RedirectURI)
	tokenData, err := client.GetAccessToken(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "换取 access_token 失败: " + err.Error(),
		})
		return
	}

	// 3. 查找已存在的同平台账号，有则更新，无则创建
	var account models.Account
	err = models.DB.Where("user_id = ? AND platform = ? AND platform_account_id = ?",
		userID, "douyin", tokenData.OpenID).First(&account).Error

	if err == nil {
		// 已存在，更新 token
		account.AccessToken = tokenData.AccessToken
		account.RefreshToken = tokenData.RefreshToken
		account.ExpiresAt = time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second)
		if saveErr := models.DB.Save(&account).Error; saveErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新账号信息失败",
			})
			return
		}
	} else {
		// 不存在，新建
		account = models.Account{
			UserID:            userID,
			Platform:          "douyin",
			PlatformAccountID: tokenData.OpenID,
			AccessToken:       tokenData.AccessToken,
			RefreshToken:      tokenData.RefreshToken,
			ExpiresAt:         time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second),
		}
		if createErr := models.DB.Create(&account).Error; createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "创建账号绑定失败",
			})
			return
		}
	}

	// 4. 返回授权成功页面
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(oauthSuccessHTML))
}

// ---- state 编解码 ----

// generateStateToken 将用户 ID 编码为短期有效的 JWT state token。
func generateStateToken(userID uint, secret string) (string, error) {
	claims := stateClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)), // state 10 分钟有效
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// parseStateToken 解析 state token，返回其中的用户 ID。
func parseStateToken(state, secret string) (uint, error) {
	claims := &stateClaims{}
	token, err := jwt.ParseWithClaims(state, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("无效或已过期的 state")
	}
	return claims.UserID, nil
}

// oauthSuccessHTML 授权成功后的展示页面。
const oauthSuccessHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>授权成功</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .card { background: #fff; border-radius: 12px; padding: 48px 64px; text-align: center; box-shadow: 0 2px 16px rgba(0,0,0,0.08); }
        .icon { font-size: 64px; margin-bottom: 16px; }
        h2 { color: #333; margin: 0 0 8px 0; }
        p { color: #666; margin: 0; font-size: 14px; }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">✅</div>
        <h2>抖音授权成功</h2>
        <p>您的抖音账号已成功绑定，可以关闭此页面。</p>
    </div>
</body>
</html>`
