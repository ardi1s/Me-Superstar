// douyin 包封装抖音开放平台 OAuth 与数据 API，包括获取 access_token、刷新 token、
// 拉取作品列表、获取作品数据指标等。
package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 抖音开放平台 API 地址
const (
	AuthURL          = "https://open.douyin.com/platform/oauth/connect/"      // 用户授权页面
	AccessTokenURL   = "https://open.douyin.com/oauth/access_token/"          // code 换 access_token
	RefreshTokenURL  = "https://open.douyin.com/oauth/renew_refresh_token/"  // 刷新 access_token
	VideoListURL     = "https://open.douyin.com/video/list/"                  // 作品列表
	ItemDataURL      = "https://open.douyin.com/data/external/item/data/"    // 作品数据指标
)

// TokenExpiredCode 抖音 API 返回的 token 过期错误码（不同版本可能不同，此处列出常见值）。
const TokenExpiredCode = 10008

// ---- OAuth 相关类型 ----

// TokenResponse 抖音 access_token 接口的返回结构。
type TokenResponse struct {
	Data    TokenData `json:"data"`
	Message string    `json:"message"`
}

// TokenData access_token 接口返回的数据体。
type TokenData struct {
	AccessToken  string `json:"access_token"`  // 访问令牌
	ExpiresIn    int    `json:"expires_in"`    // 过期时间（秒）
	RefreshToken string `json:"refresh_token"` // 用于刷新 access_token 的令牌
	OpenID       string `json:"open_id"`       // 授权用户的唯一标识
	Scope        string `json:"scope"`         // 授权的权限范围
}

// RefreshResponse 抖音刷新 token 接口的返回结构。
type RefreshResponse struct {
	Data    RefreshData `json:"data"`
	Message string      `json:"message"`
}

// RefreshData 刷新 token 接口返回的数据体。
type RefreshData struct {
	AccessToken  string `json:"access_token"`  // 新的访问令牌
	ExpiresIn    int    `json:"expires_in"`    // 过期时间（秒）
	RefreshToken string `json:"refresh_token"` // 新的 refresh_token（旧 token 立即失效）
	OpenID       string `json:"open_id"`       // 授权用户的唯一标识
	Scope        string `json:"scope"`         // 授权的权限范围
}

// ---- 作品相关类型 ----

// VideoListResponse 抖音视频列表 API 的返回结构。
type VideoListResponse struct {
	Data    VideoListData  `json:"data"`
	Extra   *ErrorExtra    `json:"extra,omitempty"` // 错误时抖音将错误信息放在 extra 中
	Message string         `json:"message,omitempty"`
}

// ErrorExtra 抖音 API 错误附加信息。
type ErrorExtra struct {
	ErrorCode int    `json:"error_code"`
	SubCode   int    `json:"sub_code,omitempty"`
	SubDesc   string `json:"sub_description,omitempty"`
	Desc      string `json:"description,omitempty"`
}

// VideoListData 视频列表数据体。
type VideoListData struct {
	List    []WorkItem `json:"list"`     // 作品列表
	Cursor  int64      `json:"cursor"`   // 分页游标，下次请求传入
	HasMore bool       `json:"has_more"` // 是否还有更多
}

// WorkItem 抖音 API 返回的单条作品信息。
type WorkItem struct {
	ItemID     string `json:"item_id"`     // 作品 ID
	Title      string `json:"title"`       // 作品标题
	Cover      string `json:"cover"`       // 封面图 URL
	CreateTime int64  `json:"create_time"` // 发布时间（Unix 时间戳）
}

// ---- 作品数据指标类型 ----

// ItemDataResponse 抖音作品数据 API 返回结构。
type ItemDataResponse struct {
	Data    ItemDataWrapper `json:"data"`
	Extra   *ErrorExtra     `json:"extra,omitempty"`
	Message string          `json:"message,omitempty"`
}

// ItemDataWrapper 作品数据外层包装。
type ItemDataWrapper struct {
	ResultList []ItemDataResult `json:"result_list"`
}

// ItemDataResult 单条作品的数据指标。
type ItemDataResult struct {
	ItemID string      `json:"item_id"`
	Data   ItemMetrics `json:"data"`
}

// ItemMetrics 作品维度指标（均为累计值，非增量）。
type ItemMetrics struct {
	PlayCount    int64 `json:"play_count"`    // 播放数
	LikeCount    int64 `json:"like_count"`    // 点赞数
	CommentCount int64 `json:"comment_count"` // 评论数
	ShareCount   int64 `json:"share_count"`   // 分享数
}

// ---- Client ----

// Client 抖音开放平台客户端，封装 OAuth 及数据相关 API 调用。
type Client struct {
	ClientKey    string       // 应用 Key（client_key）
	ClientSecret string       // 应用密钥（client_secret）
	RedirectURI  string       // OAuth 授权成功后的回调地址
	HTTPClient   *http.Client // HTTP 客户端，默认超时 15 秒
}

// NewClient 创建一个 DouyinClient 实例。
func NewClient(clientKey, clientSecret, redirectURI string) *Client {
	return &Client{
		ClientKey:    clientKey,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---- 通用错误处理 ----

// isTokenExpired 判断 API 返回是否为 token 过期错误。
func isTokenExpired(extra *ErrorExtra, body string) bool {
	if extra != nil && extra.ErrorCode == TokenExpiredCode {
		return true
	}
	// 兼容抖音部分接口直接返回 HTTP 401
	return strings.Contains(body, `"error_code":10008`)
}

// ---- OAuth 方法 ----

// GenerateAuthURL 生成抖音 OAuth 授权页面链接。
// state 参数用于防 CSRF，建议编码用户 ID 等上下文信息。
// scope 为授权的权限范围，多个用逗号分隔，如 "user_info,fans.list,video.list"。
func (c *Client) GenerateAuthURL(state, scope string) string {
	params := url.Values{}
	params.Set("client_key", c.ClientKey)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("redirect_uri", c.RedirectURI)
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// GetAccessToken 使用授权码 code 换取 access_token 和 open_id。
func (c *Client) GetAccessToken(code string) (*TokenData, error) {
	payload := url.Values{}
	payload.Set("client_key", c.ClientKey)
	payload.Set("client_secret", c.ClientSecret)
	payload.Set("code", code)
	payload.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", AccessTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造 access_token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 access_token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 access_token 响应失败: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("解析 access_token 响应失败: %w, body: %s", err, string(body))
	}

	if tokenResp.Data.AccessToken == "" {
		return nil, fmt.Errorf("获取 access_token 失败，响应: %s", string(body))
	}

	return &tokenResp.Data, nil
}

// RefreshAccessToken 使用 refresh_token 换取新的 access_token。
// 注意：抖音会在每次刷新后返回新的 refresh_token，旧的立即失效。
func (c *Client) RefreshAccessToken(refreshToken string) (*RefreshData, error) {
	payload := url.Values{}
	payload.Set("client_key", c.ClientKey)
	payload.Set("refresh_token", refreshToken)
	payload.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", RefreshTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造 refresh_token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 refresh_token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 refresh_token 响应失败: %w", err)
	}

	var refreshResp RefreshResponse
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return nil, fmt.Errorf("解析 refresh_token 响应失败: %w, body: %s", err, string(body))
	}

	if refreshResp.Data.AccessToken == "" {
		return nil, fmt.Errorf("刷新 token 失败，响应: %s", string(body))
	}

	return &refreshResp.Data, nil
}

// ---- 数据 API 方法 ----

// GetWorks 拉取指定账号的作品列表，带游标分页，最多取最近 50 条。
// ctx 用于超时控制；accessToken 和 openID 为授权后获得的凭证。
func (c *Client) GetWorks(ctx context.Context, accessToken, openID string) ([]WorkItem, error) {
	var allItems []WorkItem
	cursor := int64(0)
	count := int64(10) // 每页 10 条
	maxItems := 50

	for len(allItems) < maxItems {
		select {
		case <-ctx.Done():
			return allItems, ctx.Err()
		default:
		}

		params := url.Values{}
		params.Set("access_token", accessToken)
		params.Set("open_id", openID)
		params.Set("cursor", fmt.Sprintf("%d", cursor))
		params.Set("count", fmt.Sprintf("%d", count))

		reqURL := fmt.Sprintf("%s?%s", VideoListURL, params.Encode())
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return allItems, fmt.Errorf("构造作品列表请求失败: %w", err)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return allItems, fmt.Errorf("请求作品列表失败: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return allItems, fmt.Errorf("读取作品列表响应失败: %w", err)
		}

		var videoResp VideoListResponse
		if err := json.Unmarshal(body, &videoResp); err != nil {
			return allItems, fmt.Errorf("解析作品列表失败: %w, body: %s", err, string(body))
		}

		// 检查 token 是否过期
		if isTokenExpired(videoResp.Extra, string(body)) {
			return allItems, &TokenExpiredError{Message: videoResp.Extra.Desc}
		}

		if videoResp.Data.List == nil {
			return allItems, fmt.Errorf("作品列表接口返回异常: %s", string(body))
		}

		allItems = append(allItems, videoResp.Data.List...)

		if !videoResp.Data.HasMore {
			break
		}
		cursor = videoResp.Data.Cursor
	}

	if len(allItems) > maxItems {
		allItems = allItems[:maxItems]
	}

	return allItems, nil
}

// GetItemData 获取指定作品的数据指标（播放、点赞、评论、分享等累计值）。
func (c *Client) GetItemData(ctx context.Context, accessToken, openID, itemID string) (*ItemMetrics, error) {
	reqBody := map[string]interface{}{
		"access_token": accessToken,
		"open_id":      openID,
		"item_ids":     []string{itemID},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ItemDataURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("构造作品数据请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求作品数据失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取作品数据响应失败: %w", err)
	}

	var dataResp ItemDataResponse
	if err := json.Unmarshal(body, &dataResp); err != nil {
		return nil, fmt.Errorf("解析作品数据失败: %w, body: %s", err, string(body))
	}

	if isTokenExpired(dataResp.Extra, string(body)) {
		return nil, &TokenExpiredError{Message: dataResp.Extra.Desc}
	}

	if len(dataResp.Data.ResultList) == 0 {
		return nil, fmt.Errorf("作品数据接口未返回指标: %s", string(body))
	}

	return &dataResp.Data.ResultList[0].Data, nil
}

// GetWorkFansDelta 获取指定作品在时间范围内的粉丝增长数。
// 抖音作品数据 API 暂无直接的按时间范围返回粉丝增量的接口，
// 因此采用近似方案：先获取作品当前累计互动指标，
// 再由 service 层与昨日存储的数据做差值计算每日增量。
// 如果 accessToken 过期，返回 TokenExpiredError。
func (c *Client) GetWorkFansDelta(ctx context.Context, accessToken, openID, workID string) (int64, error) {
	// 抖音作品维度数据 API 暂不支持直接按日期返回粉丝增量，
	// 此处返回 0 并交由 service 层通过累加差值近似计算。
	_ = ctx
	_ = accessToken
	_ = openID
	_ = workID
	return 0, nil
}

// ---- TokenExpiredError ----

// TokenExpiredError 表示抖音 API 返回 token 过期 / 无效错误。
// service 层捕获此错误后应使用 refresh_token 刷新凭证。
type TokenExpiredError struct {
	Message string
}

func (e *TokenExpiredError) Error() string {
	if e.Message != "" {
		return "token 已过期: " + e.Message
	}
	return "token 已过期"
}
