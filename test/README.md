# 测试套件说明

本目录包含 agent-backend 的集成测试，覆盖注册、登录、JWT 认证等核心链路。

## 测试架构

```
test/
├── main_test.go   # 10 个测试用例（见下表）
└── README.md      # 本文件
```

测试使用 **SQLite 内存数据库** 替代 MySQL，无需外部依赖。`TestMain` 负责初始化配置、数据库和 Gin 路由。

## 测试用例清单

| # | 测试函数 | 验证内容 | 预期 |
|---|---------|---------|------|
| 1 | `TestRegister_Success` | 有效用户名+密码注册 | 200 + token + user_id |
| 2 | `TestRegister_Duplicate` | 重复用户名注册 | 409 |
| 3 | `TestRegister_InvalidInput` | 5 种非法输入（缺字段/长度不足） | 400 |
| 4 | `TestLogin_Success` | 注册后登录 | 200 + token |
| 5 | `TestLogin_WrongPassword` | 错误密码登录 | 401 |
| 6 | `TestLogin_NonexistentUser` | 不存在的用户登录 | 401 |
| 7 | `TestProfile_WithValidToken` | 带有效 token 访问 profile | 200 + username |
| 8 | `TestProfile_NoToken` | 无 token 访问 profile | 401 |
| 9 | `TestProfile_InvalidToken` | 伪造 token 访问 profile | 401 |
| 10 | `TestHealthCheck` | 健康检查端点 | 200 + `{"status":"ok"}` |
| 11 | `TestServerStartup` | **真实端口启动**，发送 HTTP 请求 | 200 |

## 运行方式

```bash
# 所有测试
cd agent-backend
go test -v ./test/ -timeout 30s

# 单独跑一个
go test -v ./test/ -run TestRegister_Success

# 只跑端口启动测试
go test -v ./test/ -run TestServerStartup
```

## 关键技术点

- **SQLite :memory:**：在 `TestMain` 中初始化，所有测试共享同一内存数据库，表结构通过 AutoMigrate 自动创建
- **httptest**：大多数测试使用 `httptest.NewRecorder()` 直接测试 Gin handler，不占用真实端口，速度快
- **真实端口测试**：`TestServerStartup` 使用 `net.Listen("tcp", ":0")` 获取空闲端口，在 goroutine 中启动 Gin，通过 `http.Get` / `http.Post` 验证
- **配置注入**：通过 `config.SetTestConfig()` 直接注入测试配置，无需 `config.yaml`

## 添加新测试

1. 在 `test/main_test.go` 中编写 `func TestXxx(t *testing.T)`
2. 使用 `newRequest(method, path, body)` 发送请求
3. 使用 `parseBody(t, w)` 解析 JSON 响应
4. 使用 `assertCode` / `assertField` / `assertNotEmpty` 做断言

```go
func TestSomethingNew(t *testing.T) {
    w := newRequest("POST", "/api/v1/xxx", map[string]string{
        "field": "value",
    })
    body := parseBody(t, w)
    assertCode(t, 200, w.Code, body)
}
```
