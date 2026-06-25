// handlers 包定义 HTTP 请求处理函数。
//
// 本文件实现作品排行榜与账号列表相关的 API。
package handlers

import (
	"net/http"
	"strconv"

	"agent-backend/models"
	"agent-backend/services"

	"github.com/gin-gonic/gin"
)

// ---- 账号列表 ----

// ListAccounts 返回当前用户已授权的所有平台账号。
// 路由：GET /api/v1/accounts（需 Bearer Token）
func ListAccounts(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var accounts []models.Account
	if err := models.DB.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询账号列表失败",
		})
		return
	}

	// 脱敏：不返回 token
	type accountItem struct {
		ID                uint   `json:"id"`
		Platform          string `json:"platform"`
		PlatformAccountID string `json:"platform_account_id"`
	}

	items := make([]accountItem, 0, len(accounts))
	for _, a := range accounts {
		items = append(items, accountItem{
			ID:                a.ID,
			Platform:          a.Platform,
			PlatformAccountID: a.PlatformAccountID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": items,
	})
}

// ---- 作品排行榜 ----

// TopWorksResponse 作品排行榜 API 响应。
type TopWorksResponse struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Data    *services.TopWorksResult `json:"data,omitempty"`
}

// GetTopWorksByFans 返回作品涨粉排行榜。
// 路由：GET /api/v1/works/top-fans?period=7d&page=1&page_size=20&account_id=1
func GetTopWorksByFans(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 解析参数
	period := c.DefaultQuery("period", "7d")
	if period != "1d" && period != "7d" && period != "30d" {
		c.JSON(http.StatusBadRequest, TopWorksResponse{
			Code:    400,
			Message: "period 参数无效，可选值: 1d, 7d, 30d",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	accountIDStr := c.DefaultQuery("account_id", "0")
	accountID, _ := strconv.ParseUint(accountIDStr, 10, 64)

	// 校验 account_id 是否属于当前用户（如果传了）
	if accountID > 0 {
		var account models.Account
		if err := models.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
			c.JSON(http.StatusForbidden, TopWorksResponse{
				Code:    403,
				Message: "该账号不属于当前用户",
			})
			return
		}
	}

	result, err := services.GetTopWorksByFans(services.TopWorksQuery{
		AccountID: uint(accountID),
		Period:    period,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, TopWorksResponse{
			Code:    500,
			Message: "查询排行榜失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TopWorksResponse{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}
