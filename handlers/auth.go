// handlers 包定义 HTTP 请求处理函数（注册、登录等）。
package handlers

import (
	"net/http"
	"time"

	"agent-backend/config"
	"agent-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ---- 请求 / 响应结构体 ----

// RegisterRequest 注册请求体。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse 认证相关的统一响应体（注册 / 登录共用）。
type AuthResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	UserID  uint   `json:"user_id,omitempty"`
}

// ---- JWT 生成 ----

// Claims 自定义 JWT 载荷，包含用户 ID 和用户名。
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// generateJWT 根据用户信息生成签名的 JWT Token。
func generateJWT(user *models.User) (string, error) {
	cfg := config.AppConfig.JWT
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.ExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ---- 注册 ----

// Register 处理用户注册：校验参数 → 检查重名 → bcrypt 加密密码 → 入库 → 返回 JWT。
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
		})
		return
	}

	// 检查用户名是否已存在
	var exist models.User
	if err := models.DB.Where("username = ?", req.Username).First(&exist).Error; err == nil {
		c.JSON(http.StatusConflict, AuthResponse{
			Code:    409,
			Message: "用户名已存在",
		})
		return
	}

	user := models.User{
		Username: req.Username,
	}
	// 使用 bcrypt 加密密码
	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    500,
			Message: "密码加密失败",
		})
		return
	}

	// 写入数据库
	if err := models.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    500,
			Message: "用户创建失败",
		})
		return
	}

	token, err := generateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    500,
			Message: "Token 生成失败",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Code:    200,
		Message: "注册成功",
		Token:   token,
		UserID:  user.ID,
	})
}

// ---- 登录 ----

// Login 处理用户登录：校验参数 → 查用户 → 校验密码 → 返回 JWT。
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
		})
		return
	}

	var user models.User
	if err := models.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	// bcrypt 校验密码
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	token, err := generateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    500,
			Message: "Token 生成失败",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Code:    200,
		Message: "登录成功",
		Token:   token,
		UserID:  user.ID,
	})
}
