// douyin 包封装抖音开放平台 OAuth 相关 API，包括获取 access_token 和刷新 token。
package douyin

import (
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
	AuthURL         = "https://open.douyin.com/platform/oauth/connect/"     // 用户授权页面
	AccessTokenURL  = "https://open.douyin.com/oauth/access_token/"         // code 换 access_token
	RefreshTokenURL = "https://open.douyin.com/oauth/renew_refresh_token/" // 刷新 access_token
)

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

// Client 抖音开放平台客户端，封装 OAuth 相关 API 调用。
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

// RefreshToken 使用 refresh_token 换取新的 access_token。
// 注意：抖音会在每次刷新后返回新的 refresh_token，旧的立即失效。
func (c *Client) RefreshToken(refreshToken string) (*RefreshData, error) {
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
