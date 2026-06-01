# OpenTab Server

OpenTab 服务端第一阶段使用 Go + Gin + 内存 Mock 数据，先支撑 Android 客户端联调。后续在同一套路由和模型基础上接入 GORM + PostgreSQL。

## 运行方式

先安装 Go，然后在本目录执行：

```bash
go mod tidy
go run ./cmd/server
```

默认地址：

```text
http://localhost:8080
```

也可以通过环境变量修改端口：

```bash
PORT=18080 go run ./cmd/server
```

Android 模拟器访问本机服务：

```text
http://10.0.2.2:8080
```

## 当前接口

```text
GET  /health
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

## 当前演示账号

```text
账号：opentab-demo
密码：demo123

账号：opentab-admin
密码：admin123

账号：opentab-guest
密码：guest123
```

## 当前 Mock 规则

- `opentab-demo`：启用审批中心、团队日程、新版实验 Tab。
- `opentab-admin`：启用审批中心、团队日程、财务看板、新版实验 Tab、接入文档。
- `opentab-guest`：启用接入文档 Web Tab。
- `GET /tabs` 会根据 Bearer Token 返回当前账号已启用的 TabManifest。
- `GET /tabs/catalog` 会返回系统 Tab 目录，并用 `enabled` 标记当前账号是否启用。
- native Tab 只下发配置，真实页面由客户端本地实现；web Tab 用于演示接入网页。

## 当前阶段边界

- 数据暂时放在 `internal/mockdata`
- 路由放在 `internal/routes`
- 数据模型放在 `internal/models`
- 后续数据库接入时，优先新增 repository 层，不改客户端接口格式
