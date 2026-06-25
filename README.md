# agent-backend

基于 Gin + GORM 的多平台账号数据管理后端，支持用户注册登录、OAuth 授权、作品数据同步与涨粉排行榜展示。

## 技术栈

- **后端**：Go + Gin + GORM + MySQL
- **前端**：Vue 3 + Vite
- **认证**：JWT + bcrypt
- **定时任务**：robfig/cron

## 已实现功能

- 用户注册 / 登录（bcrypt 密码加密，JWT Token）
- 抖音 OAuth 授权流程（生成授权链接 → 回调换 token → 自动入库）
- 作品数据模型（Works 表 + WorkDailyStats 表，联合唯一索引）
- 作品涨粉排行榜 API（支持 1d / 7d / 30d 时间段 + 分页）
- 账号同步定时任务（每小时拉取作品与数据，token 过期自动刷新）
- Vue 3 管理面板（Apple 风格 UI，登录 / 排行榜 / 账号管理）

## 项目状态

**⚠️ 项目暂时搁置**

抖音开放平台企业开发者认证（企业资质）审核未通过，无法获取 `client_key` / `client_secret`，OAuth 授权流程和数据 API 调用受阻。

当前的解决方案是将数据源从抖音 API 切换为手动录入 —— 用户通过后台管理页录入作品与每日数据，排行榜等已有功能继续正常使用。后续认证通过后可无缝切回 API 模式。

## 运行方式

```bash
# 1. 启动后端
cd agent-backend
cp config.yaml.example config.yaml  # 编辑数据库连接信息
go run main.go                      # 监听 :8080

# 2. 启动前端（开发模式）
cd frontend
npm install
npm run dev                         # 监听 :5173，自动代理 /api 到 :8080

# 3. 运行测试
go test -v ./test/
```

## 项目结构

```
agent-backend/
├── config/            # viper 配置加载
├── models/            # GORM 数据模型（User/Account/Work/WorkDailyStats）
├── handlers/          # HTTP 请求处理（auth/oauth/works）
├── middleware/        # JWT 认证中间件
├── services/          # 业务逻辑层（sync/ranking）
├── pkg/douyin/        # 抖音开放平台客户端封装
├── worker/            # cron 定时调度
├── router/            # Gin 路由注册
├── test/              # 集成测试（MySQL 真实环境）
├── frontend/          # Vue 3 管理前端
├── main.go            # 入口
└── config.yaml        # 配置文件（不纳入版本控制）
```
