# OpenTab Server 开发路线

本文档记录 `server/` 后续从当前 Go Mock Server 演进到 PostgreSQL 持久化服务端的阶段计划。

当前约定：

```text
mockserver/ 作为稳定联调版本，暂时不动。
server/     作为继续学习和演进的服务端工程。
```

## 1. 当前项目结构

仓库结构：

```text
OpenTab/
  app/                  Android 客户端
  server/               后续继续开发的服务端
  mockserver/           已可联调的稳定 mock server 副本
  docs/                 接口文档、项目文档
```

`server/` 当前结构：

```text
server/
  cmd/
    server/
      main.go           服务启动入口

  internal/
    routes/             HTTP 路由和接口处理
      router.go         注册所有接口
      auth.go           登录、当前用户、鉴权辅助
      tabs.go           TabManifest 相关接口
      oncall.go         AI OnCall SSE
      business.go       审批/日程 mock 业务数据
      debug.go          debug 状态

    models/             请求、响应、数据结构
      auth.go
      tab.go
      error.go
      business.go

    mockdata/           内存 mock 数据
      users.go
      tabs.go
      business.go

  go.mod
  go.sum
  README.md
  DEVELOPMENT_ROADMAP.md
```

当前服务端已经具备：

```text
POST /auth/login
POST /auth/logout
GET  /me
GET  /tabs
GET  /tabs/catalog
POST /me/tabs
DELETE /me/tabs/{tabId}
GET  /tabs/{tabId}
POST /tabs/validate
POST /tabs/{tabId}/actions/{actionId}
GET  /business/approval/summary
GET  /business/calendar/summary
GET  /debug/status
GET  /oncall/stream?message=...
```

当前服务端特点：

```text
优点：
- 已经能跑
- 接口比较完整
- 路由、模型、mock 数据已经分目录
- 支持登录、TabManifest 下发、SSE、业务 mock、debug

不足：
- routes 里同时包含 HTTP 处理、业务判断、数据访问
- mockdata 是全局变量，没有 repository 抽象
- 鉴权还不是 middleware
- 错误响应没有统一 helper
- 配置能力还比较简单
- 没有服务端测试
- 还没接 PostgreSQL
```

## 2. 总体演进路线

建议按以下顺序推进：

```text
1. 整理配置 config
2. 统一错误响应 response
3. 鉴权改成 middleware
4. 拆 service 层
5. 抽 repository 层，并实现 memory repository
6. 补服务端测试
7. 接 PostgreSQL + GORM
8. 实现 postgres repository
9. 支持 mock/postgres 双模式切换
10. 完善 AI OnCall SSE
```

不要直接先接 PostgreSQL。当前代码还没有 service/repository 分层，直接写数据库逻辑会让 routes 变得很乱。

## 3. 阶段 1：整理配置 config

目标：让服务端启动参数更规范，为后续数据库做准备。

要做：

```text
1. 新增 internal/config/config.go
2. 支持 PORT
3. 支持 APP_MODE=mock/postgres
4. 支持 DATABASE_URL
5. main.go 不直接读取零散环境变量
```

建议结构：

```text
internal/
  config/
    config.go
```

建议配置模型：

```go
type Config struct {
    Port        string
    AppMode     string
    DatabaseURL string
}
```

完成后 `main.go` 应类似：

```go
cfg := config.Load()
router.Run(":" + cfg.Port)
```

需要学习：

```text
os.Getenv
默认值处理
Go struct
配置和代码解耦
```

验收标准：

```text
不设置 PORT 时默认 8080
设置 PORT=18080 时监听 18080
go test ./... 通过
```

## 4. 阶段 2：统一错误响应 response

目标：避免每个接口重复手写错误响应。

当前代码里多处存在：

```go
c.JSON(http.StatusBadRequest, models.ErrorResponse{
    Code:    "INVALID_REQUEST",
    Message: "请求格式不正确",
})
```

要做：

```text
1. 新增 internal/response/response.go
2. 封装 Error 方法
3. 可选封装 OK 方法
4. routes 中统一使用 response.Error
```

建议结构：

```text
internal/
  response/
    response.go
```

示例：

```go
func Error(c *gin.Context, status int, code string, message string) {
    c.JSON(status, models.ErrorResponse{
        Code:    code,
        Message: message,
    })
}
```

使用方式：

```go
response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token 无效或已过期")
```

需要学习：

```text
函数封装
HTTP 状态码
统一 API 响应格式
```

验收标准：

```text
routes 中不再大量重复构造 ErrorResponse
错误 JSON 格式保持不变
go test ./... 通过
```

## 5. 阶段 3：鉴权改成 middleware

目标：让需要登录的接口不再每个 handler 手动调用 `currentUser(c)`。

当前方式：

```go
user, ok := currentUser(c)
if !ok {
    return
}
```

目标方式：

```go
authorized := router.Group("/")
authorized.Use(middleware.Auth())
authorized.GET("/tabs", listTabs)
```

建议结构：

```text
internal/
  middleware/
    auth.go
```

middleware 要做：

```text
1. 读取 Authorization Header
2. 解析 Bearer Token
3. 根据 token 找用户
4. 找不到则返回 401
5. 找到则 c.Set("currentUser", user)
```

routes 中获取当前用户：

```go
user := middleware.CurrentUser(c)
```

需要学习：

```text
Gin middleware
c.Set
c.Get
Authorization Header
Bearer Token
```

验收标准：

```text
未带 token 访问 /tabs 返回 401
带正确 token 访问 /tabs 正常
handler 中不再重复写 currentUser 判断
go test ./... 通过
```

## 6. 阶段 4：拆 service 层

目标：routes 只负责 HTTP，业务逻辑放到 services。

当前 `routes/tabs.go` 同时处理：

```text
读取用户
查 Tab
检查权限
启用/停用 Tab
校验 Tab
返回 JSON
```

建议拆成：

```text
routes -> services -> repositories/mockdata
```

建议结构：

```text
internal/
  services/
    auth_service.go
    tab_service.go
    oncall_service.go
```

建议职责：

```text
AuthService:
  Login
  GetCurrentUser

TabService:
  ListUserTabs
  ListCatalog
  GetTab
  EnableTab
  DisableTab
  ValidateTab
  ReportAction

OnCallService:
  StreamMockReply
```

routes 中代码应变薄：

```go
resp, err := tabService.ListUserTabs(user.ID)
if err != nil {
    response.Error(...)
    return
}
c.JSON(http.StatusOK, resp)
```

需要学习：

```text
Go struct 方法
依赖传递
业务逻辑分层
```

验收标准：

```text
routes 文件主要处理请求绑定和响应
业务判断移动到 services
接口返回格式不变
go test ./... 通过
```

## 7. 阶段 5：抽 repository 层

目标：把数据来源从 `mockdata` 全局变量里抽出来。

当前：

```go
mockdata.FindUserByToken(token)
mockdata.TabsForUser(user.ID)
```

目标：

```go
userRepo.FindByToken(token)
tabRepo.ListByUser(userID)
```

建议结构：

```text
internal/
  repositories/
    user_repository.go
    tab_repository.go
    memory_user_repository.go
    memory_tab_repository.go
```

接口示例：

```go
type UserRepository interface {
    FindByAccount(account string) (*models.User, error)
    FindByToken(token string) (*models.User, error)
}

type TabRepository interface {
    ListByUser(userID string) ([]models.TabManifest, error)
    ListCatalog(userID string) ([]models.TabManifest, error)
    FindByID(tabID string) (*models.TabManifest, error)
    Enable(userID string, tabID string) error
    Disable(userID string, tabID string) error
}
```

需要学习：

```text
Go interface
依赖反转
内存实现
为什么 repository 可以替换数据源
```

验收标准：

```text
services 不直接依赖 mockdata
services 依赖 repository interface
MemoryRepository 实现当前 mock 行为
接口返回格式不变
go test ./... 通过
```

## 8. 阶段 6：补服务端测试

目标：主接口可自动验证，后续重构不容易改坏。

建议测试：

```text
POST /auth/login 成功
POST /auth/login 失败
GET /tabs 未登录返回 401
GET /tabs demo 返回 3 个 Tab
GET /tabs/catalog 返回所有 Tab
POST /tabs/validate 能发现缺字段
GET /oncall/stream 返回 SSE
```

建议结构：

```text
internal/
  routes/
    router_test.go
```

需要学习：

```text
go test
httptest
构造 HTTP request
检查响应状态码
解析 JSON 响应
```

验收标准：

```text
go test ./... 能跑出真实测试
至少覆盖登录、tabs、validate、SSE
```

## 9. 阶段 7：接 PostgreSQL + GORM

目标：把内存数据迁移到数据库。

建议新增：

```text
internal/
  database/
    database.go
    models.go
```

建议先做最少几张表：

```text
users
permissions
user_permissions
tabs
user_tabs
```

`tabs` 建议字段：

```text
id
display_name
description
icon
route
entry_type
entry_uri
version_major
version_minor
version_patch
min_container_version
permissions_json
extension_json
extra_config_json
sort_order
```

`extension_json` 和 `extra_config_json` 可以先用 PostgreSQL `JSONB`。

需要学习：

```text
PostgreSQL 基础
GORM
数据库连接字符串
AutoMigrate
JSONB
一对多/多对多关系
```

验收标准：

```text
服务能连接 PostgreSQL
AutoMigrate 能创建表
能插入初始用户和 Tab 数据
go test ./... 通过
```

## 10. 阶段 8：实现 postgres repository

目标：用 PostgreSQL 替换内存数据源，但保持 service/routes 不变。

建议新增：

```text
internal/
  repositories/
    postgres_user_repository.go
    postgres_tab_repository.go
```

要做：

```text
1. PostgresUserRepository 实现 UserRepository
2. PostgresTabRepository 实现 TabRepository
3. 查询用户、权限、Tab、用户启用关系
4. 启用/停用 Tab 写入 user_tabs
```

需要学习：

```text
GORM 查询
Where / First / Find
事务
错误处理
模型与 API DTO 转换
```

验收标准：

```text
GET /tabs 能从数据库返回数据
POST /me/tabs 能写数据库
服务重启后启用关系仍存在
客户端接口不需要改
```

## 11. 阶段 9：支持 mock/postgres 双模式

目标：同一套服务端可以切换不同数据源。

配置：

```text
APP_MODE=mock
APP_MODE=postgres
```

启动逻辑：

```go
if cfg.AppMode == "mock" {
    userRepo = repositories.NewMemoryUserRepository()
    tabRepo = repositories.NewMemoryTabRepository()
} else {
    db := database.Connect(cfg.DatabaseURL)
    userRepo = repositories.NewPostgresUserRepository(db)
    tabRepo = repositories.NewPostgresTabRepository(db)
}
```

需要学习：

```text
interface 多实现
配置驱动
依赖组装
```

验收标准：

```text
APP_MODE=mock 时使用内存数据
APP_MODE=postgres 时使用 PostgreSQL
两种模式接口返回格式一致
```

## 12. 阶段 10：完善 AI OnCall SSE

目标：让 AI 流式接口更接近真实工程。

当前是固定 mock 回复，后续可扩展：

```text
1. 抽 OnCallService
2. 定义 SseEvent model
3. 根据 message 返回不同 mock 内容
4. 配置类问题返回 validate_tab_config 工具事件
5. 日志类问题返回 analyze_client_log 工具事件
6. 支持 context 取消和客户端断开
```

需要学习：

```text
SSE 格式
Flush
context
连接断开处理
流式响应错误处理
```

验收标准：

```text
SSE 至少支持 delta/tool/done/error
客户端断开时服务端能结束处理
接口格式与客户端约定一致
```

## 13. 推荐学习顺序

```text
1. Go 基础：struct、slice、map、函数、错误处理
2. Gin：路由、JSON、请求绑定、中间件
3. HTTP：GET、POST、Header、状态码
4. SSE：event-stream、Flush、连接断开
5. Go 项目结构：cmd、internal
6. interface：repository 抽象
7. go test 和 httptest
8. PostgreSQL 基础
9. GORM
10. 配置驱动和双数据源模式
```

## 14. 最终答辩表述

服务端最终可以这样描述：

```text
我负责服务端支撑层。
第一阶段使用 Go + Gin 和内存数据实现 Mock Server，支撑客户端登录、TabManifest 下发、权限控制、AI OnCall SSE 和业务 mock 数据联调。
第二阶段通过 service/repository 分层，把数据源从内存替换为 PostgreSQL。
最终服务端支持用户、权限、Tab 启用关系、TabManifest 配置、业务数据和 AI OnCall 流式接口。
```
