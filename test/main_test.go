// test 包包含 agent-backend 的集成测试，使用 SQLite 内存数据库，无需外部依赖即可运行。
// 测试覆盖：注册、登录、JWT 认证、健康检查、端口启动验证。
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"agent-backend/config"
	"agent-backend/models"
	"agent-backend/router"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	gormio "gorm.io/gorm"
)

// ---- 测试前后置 ----

// TestMain 是整个测试套件的入口，负责初始化和清理。
func TestMain(m *testing.M) {
	// 1. 注入测试配置（无需 config.yaml）
	config.SetTestConfig(&config.Config{
		Server: config.ServerConfig{Port: 8080},
		Database: config.DatabaseConfig{
			Host: "127.0.0.1", Port: 3306,
			User: "test", Password: "test", DBName: "test",
			Charset: "utf8mb4", ParseTime: true, Loc: "Local",
		},
		JWT: config.JWTConfig{
			Secret:      "test-secret-key",
			ExpireHours: 24,
		},
	})

	// 2. 用 SQLite 内存数据库替代 MySQL（无需外部依赖，测试随时可跑）
	db, err := gormio.Open(sqlite.Open(":memory:"), &gormio.Config{})
	if err != nil {
		fmt.Printf("Failed to open SQLite: %v\n", err)
		os.Exit(1)
	}
	models.DB = db

	// 3. 自动迁移
	models.AutoMigrate()

	// 4. 设 Gin 为测试模式（关闭调试日志）
	gin.SetMode(gin.TestMode)

	code := m.Run()
	os.Exit(code)
}

// ---- 辅助函数 ----

// newRequest 创建测试 HTTP 请求并执行，返回响应记录器。
func newRequest(method, path string, body interface{}, token ...string) *httptest.ResponseRecorder {
	// 每次 newRequest 使用独立的 Router 实例
	r := router.SetupRouter()

	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(b)
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseBody 解析 JSON 响应体，返回 map。
func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, w.Body.String())
	}
	return result
}

// ---- 注册测试 ----

func TestRegister_Success(t *testing.T) {
	w := newRequest("POST", "/api/v1/auth/register", map[string]string{
		"username": "testuser",
		"password": "123456",
	})

	body := parseBody(t, w)
	assertCode(t, 200, w.Code, body)
	assertField(t, body, "code", float64(200))
	assertNotEmpty(t, body, "token")
	assertNotEmpty(t, body, "user_id")

	t.Logf("✅ Register success: user_id=%.0f token=%s...",
		body["user_id"], body["token"].(string)[:20])
}

func TestRegister_Duplicate(t *testing.T) {
	r := router.SetupRouter()
	payload := map[string]string{"username": "dupuser", "password": "123456"}
	b, _ := json.Marshal(payload)

	// 第一次注册应成功
	req1, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(b))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("first register should succeed, got %d: %s", w1.Code, w1.Body.String())
	}

	// 第二次注册应返回 409
	req2, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(b))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	body := parseBody(t, w2)
	assertCode(t, 409, w2.Code, body)
	assertField(t, body, "code", float64(409))
	t.Log("✅ Duplicate registration correctly rejected with 409")
}

func TestRegister_InvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]string
	}{
		{"missing password", map[string]string{"username": "foo"}},
		{"missing username", map[string]string{"password": "123456"}},
		{"short username", map[string]string{"username": "ab", "password": "123456"}},
		{"short password", map[string]string{"username": "validuser", "password": "12345"}},
		{"empty body", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRequest("POST", "/api/v1/auth/register", tt.payload)
			if w.Code != 400 {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			} else {
				t.Logf("  ✅ %s → %d", tt.name, w.Code)
			}
		})
	}
}

// ---- 登录测试 ----

func TestLogin_Success(t *testing.T) {
	// 先注册
	w := newRequest("POST", "/api/v1/auth/register", map[string]string{
		"username": "loginuser",
		"password": "mypassword",
	})
	if w.Code != 200 {
		t.Fatalf("register before login failed: %s", w.Body.String())
	}

	// 登录
	w2 := newRequest("POST", "/api/v1/auth/login", map[string]string{
		"username": "loginuser",
		"password": "mypassword",
	})

	body := parseBody(t, w2)
	assertCode(t, 200, w2.Code, body)
	assertNotEmpty(t, body, "token")
	t.Logf("✅ Login success: token=%s...", body["token"].(string)[:20])
}

func TestLogin_WrongPassword(t *testing.T) {
	newRequest("POST", "/api/v1/auth/register", map[string]string{
		"username": "wrongpwuser",
		"password": "correct",
	})

	w := newRequest("POST", "/api/v1/auth/login", map[string]string{
		"username": "wrongpwuser",
		"password": "wrongpassword",
	})

	body := parseBody(t, w)
	assertCode(t, 401, w.Code, body)
	t.Log("✅ Wrong password correctly rejected with 401")
}

func TestLogin_NonexistentUser(t *testing.T) {
	w := newRequest("POST", "/api/v1/auth/login", map[string]string{
		"username": "nobody",
		"password": "whatever",
	})

	body := parseBody(t, w)
	assertCode(t, 401, w.Code, body)
	t.Log("✅ Nonexistent user correctly rejected with 401")
}

// ---- Profile 认证测试 ----

func TestProfile_WithValidToken(t *testing.T) {
	w := newRequest("POST", "/api/v1/auth/register", map[string]string{
		"username": "profileuser",
		"password": "123456",
	})
	regBody := parseBody(t, w)
	token := regBody["token"].(string)

	w2 := newRequest("GET", "/api/v1/profile", nil, token)
	body := parseBody(t, w2)

	assertCode(t, 200, w2.Code, body)
	if body["username"] != "profileuser" {
		t.Fatalf("expected username 'profileuser', got %v", body["username"])
	}
	t.Logf("✅ Profile accessed with valid token: username=%v user_id=%v",
		body["username"], body["user_id"])
}

func TestProfile_NoToken(t *testing.T) {
	w := newRequest("GET", "/api/v1/profile", nil)
	body := parseBody(t, w)
	assertCode(t, 401, w.Code, body)
	t.Log("✅ Profile without token correctly rejected with 401")
}

func TestProfile_InvalidToken(t *testing.T) {
	w := newRequest("GET", "/api/v1/profile", nil, "this-is-not-a-valid-jwt")
	body := parseBody(t, w)
	assertCode(t, 401, w.Code, body)
	t.Log("✅ Profile with invalid token correctly rejected with 401")
}

// ---- 健康检查 ----

func TestHealthCheck(t *testing.T) {
	w := newRequest("GET", "/health", nil)
	body := parseBody(t, w)
	assertCode(t, 200, w.Code, body)
	if body["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %v", body["status"])
	}
	t.Log("✅ Health check passed")
}

// ---- 端口启动测试 ----
// 验证服务能否在真实端口上启动并接受 HTTP 请求。

func TestServerStartup(t *testing.T) {
	// 获取一个空闲端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	r := router.SetupRouter()

	// 在 goroutine 中启动服务
	go func() {
		_ = r.Run(addr)
	}()

	// 等待服务就绪
	time.Sleep(200 * time.Millisecond)

	// 1) 健康检查
	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("health: expected 200, got %d", resp.StatusCode)
	}

	// 2) 注册端点
	payload := map[string]string{"username": "portuser", "password": "123456"}
	b, _ := json.Marshal(payload)
	resp2, err := http.Post(
		fmt.Sprintf("http://%s/api/v1/auth/register", addr),
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		t.Fatalf("Failed to call register: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp2.Body)
		t.Fatalf("register on real port: expected 200, got %d: %s",
			resp2.StatusCode, string(bodyBytes))
	}

	t.Logf("✅ Server started on %s, health + register passed", addr)
}

// ---- assert 辅助 ----

func assertCode(t *testing.T, want, got int, body map[string]interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("expected HTTP %d, got %d, body: %v", want, got, body)
	}
}

func assertField(t *testing.T, body map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if body[key] != want {
		t.Fatalf("expected body[%q]=%v, got %v", key, want, body[key])
	}
}

func assertNotEmpty(t *testing.T, body map[string]interface{}, key string) {
	t.Helper()
	val, ok := body[key]
	if !ok || val == nil || val == "" {
		t.Fatalf("expected body[%q] to be non-empty, got %v", key, val)
	}
}
