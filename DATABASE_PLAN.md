# OpenTab Server 数据库接入设计文档

本文档用于指导 `server/` 从当前内存 mock 数据演进到 PostgreSQL 持久化服务端。

当前服务端已经具备：

```text
routes -> services -> repositories -> mockdata
```

接数据库时不要把 SQL 写进 routes，也不要让 handler 直接操作 GORM。目标是保持：

```text
routes -> services -> repositories -> PostgreSQL
```

这样客户端接口不变，业务逻辑大部分不变，只替换数据来源。

## 1. 总体阶段

### 阶段 7：接入 PostgreSQL 和 GORM

目标：让服务端具备数据库连接、建表、初始化数据能力。

要做：

```text
1. 引入 GORM 和 PostgreSQL driver
2. 新增 internal/database
3. 定义数据库模型
4. 读取 DATABASE_URL
5. 启动时连接 PostgreSQL
6. AutoMigrate 建表
7. 插入初始账号、权限、Tab、业务数据
8. go test ./... 通过
```

这一阶段只要求“数据库能连、表能建、数据能初始化”，不急着替换全部接口。

### 阶段 8：实现 PostgreSQL repository

目标：让现有 repository 接口拥有 PostgreSQL 实现。

要做：

```text
1. PostgresUserRepository 实现 UserRepository
2. PostgresTabRepository 实现 TabRepository
3. PostgresBusinessRepository 实现 BusinessRepository
4. PostgresOnCallRepository 实现 OnCallRepository
5. PostgresDebugRepository 实现 DebugRepository
6. 保持 service/routes 不改或少改
7. go test ./... 通过
```

这一阶段完成后，接口数据就可以来自数据库。

### 阶段 9：支持 mock/postgres 双模式

目标：开发时可继续用内存 mock，正式运行用 PostgreSQL。

建议配置：

```text
APP_MODE=mock
APP_MODE=postgres
DATABASE_URL=postgres://user:password@host:5432/opentab?sslmode=disable
```

启动逻辑：

```go
if cfg.AppMode == "postgres" {
    db := database.Connect(cfg.DatabaseURL)
    repos := repositories.NewPostgresRepositories(db)
} else {
    repos := repositories.NewMemoryRepositories()
}
```

这样本地学习、云服务器演示、客户端联调可以按场景切换。

## 2. 设计原则

### 2.1 先存核心状态，再追求完整业务

第一版数据库不需要复杂到企业级系统，先保证这些状态不会因为服务端重启而丢失：

```text
账号
登录 token
用户权限
Tab 配置
用户启用哪些 Tab
用户自定义 Tab
AI 会话
AI 消息
审批数据
日程数据
```

### 2.2 接口 DTO 和数据库 Model 分开

当前 `internal/models` 主要是接口返回结构，不建议直接拿来当 GORM 表模型。

建议新增：

```text
internal/database/models.go
```

数据库模型可以叫：

```text
UserRecord
PermissionRecord
TabRecord
UserTabRecord
OnCallSessionRecord
OnCallMessageRecord
ApprovalItemRecord
CalendarEventRecord
```

原因：

```text
接口字段适合 JSON
数据库字段适合查询、索引、关联和迁移
两者变化节奏不同
```

例如接口里 `TabManifest.version` 是对象：

```json
{
  "major": 1,
  "minor": 0,
  "patch": 0
}
```

数据库里更适合拆成：

```text
version_major
version_minor
version_patch
```

### 2.3 JSONB 用在扩展配置，不滥用

TabManifest 有些字段结构灵活：

```text
extension
extraConfig
```

这些可以用 PostgreSQL `JSONB`。

但用户、权限、Tab 启用关系、消息归属这些核心关系不要塞 JSONB，要用表关系表达。

原因：

```text
关系字段需要查询、过滤、唯一约束和索引
JSONB 适合配置扩展，不适合核心关联
```

## 3. 建表设计

## 3.1 users

保存账号基础信息。

字段建议：

```text
id                 uuid / string primary key
account            varchar unique not null
display_name       varchar not null
password_hash      varchar not null
created_at         timestamptz not null
updated_at         timestamptz not null
```

第一版学习阶段可以先继续用明文密码字段，但文档设计上建议用 `password_hash`。

索引：

```text
unique(account)
```

原因：

```text
登录时按 account 查用户
account 不能重复
```

## 3.2 auth_sessions

保存登录 token。

字段建议：

```text
id           uuid / string primary key
user_id      references users(id)
token        varchar unique not null
expires_at   timestamptz
created_at   timestamptz not null
revoked_at   timestamptz
```

索引：

```text
unique(token)
index(user_id)
index(expires_at)
```

原因：

```text
每次请求都要用 token 找用户
用户退出登录时可撤销 token
后续可以清理过期 token
```

为什么先用 token 表，不直接 JWT：

```text
token 表更容易学习和调试
服务端可以主动让 token 失效
数据库里能看见当前登录状态
```

后续如果熟悉了，可以再换 JWT。

## 3.3 permissions

保存系统权限字典。

字段建议：

```text
code          varchar primary key
description   varchar not null
created_at    timestamptz not null
```

初始数据：

```text
tab.approval.read
tab.calendar.read
tab.finance.read
ai.oncall
```

索引：

```text
primary key(code)
```

原因：

```text
权限 code 是稳定标识
debug/permissions 可直接从这里返回
```

## 3.4 user_permissions

保存用户拥有哪些权限。

字段建议：

```text
user_id          references users(id)
permission_code  references permissions(code)
created_at       timestamptz not null
primary key(user_id, permission_code)
```

索引：

```text
primary key(user_id, permission_code)
index(permission_code)
```

原因：

```text
查当前用户权限时按 user_id
避免同一个用户重复插入同一权限
```

## 3.5 tabs

保存系统内置 Tab 和用户自定义 Tab。

字段建议：

```text
id                     varchar primary key
owner_user_id           references users(id), nullable
display_name            varchar not null
description             text
icon                    varchar
route                   varchar not null
entry_type              varchar not null
entry_uri               text
version_major           int not null
version_minor           int not null
version_patch           int not null
min_container_version   int not null
permissions_json        jsonb not null
extension_json          jsonb
extra_config_json       jsonb
is_system               boolean not null
created_at              timestamptz not null
updated_at              timestamptz not null
```

字段说明：

```text
owner_user_id 为空：系统内置 Tab
owner_user_id 不为空：某个用户创建的自定义 Tab
is_system=true：系统内置 Tab，不允许用户修改/删除
permissions_json：Tab 打开需要的权限列表
extension_json：TitleBar/FAB/BottomPanel 等扩展点
extra_config_json：业务扩展配置
```

索引：

```text
primary key(id)
index(owner_user_id)
index(entry_type)
unique(owner_user_id, route)
```

关于 `unique(owner_user_id, route)`：

```text
用于防止同一个用户创建两个 route 一样的自定义 Tab
```

注意 PostgreSQL 里 `NULL` 的唯一约束行为比较特殊。系统 Tab 的 `owner_user_id` 是 NULL，是否需要全局 route 唯一可以另外加：

```text
unique(route) where owner_user_id is null
```

第一版如果不写部分索引，也可以在 service 层先校验。

## 3.6 user_tabs

保存用户启用了哪些 Tab，以及排序。

字段建议：

```text
user_id      references users(id)
tab_id       references tabs(id)
enabled      boolean not null default true
sort_order   int not null default 0
created_at   timestamptz not null
updated_at   timestamptz not null
primary key(user_id, tab_id)
```

索引：

```text
primary key(user_id, tab_id)
index(user_id, sort_order)
index(tab_id)
```

原因：

```text
GET /tabs 按 user_id 查询已启用 Tab，并按 sort_order 排序
POST /me/tabs 写入或更新 user_tabs
DELETE /me/tabs/{tabId} 删除或 enabled=false
```

建议第一版停用时直接删除 `user_tabs` 记录。

如果后续想保留历史，可以改成 `enabled=false`。

## 3.7 oncall_sessions

保存 AI 会话。

字段建议：

```text
id          varchar primary key
user_id     references users(id)
title       varchar not null
created_at  timestamptz not null
updated_at  timestamptz not null
deleted_at  timestamptz
```

索引：

```text
index(user_id, updated_at desc)
index(deleted_at)
```

原因：

```text
GET /oncall/sessions 需要按当前用户查询，并按更新时间排序
删除会话可以软删除，避免误删聊天记录
```

第一版也可以硬删除，简单一些。

## 3.8 oncall_messages

保存 AI 消息。

字段建议：

```text
id            varchar primary key
session_id    references oncall_sessions(id)
role          varchar not null
content       text not null
content_type  varchar not null default 'text'
created_at    timestamptz not null
```

索引：

```text
index(session_id, created_at)
index(role)
```

原因：

```text
GET /oncall/sessions/{sessionId}/messages 按 session_id 拉消息，并按 created_at 排序
role 用于区分 user / assistant / tool
```

注意：

```text
不要只在客户端保存聊天记录
服务端保存后，换设备/重启服务/重新登录都能恢复
```

## 3.9 approval_items

保存审批演示数据。

字段建议：

```text
id            varchar primary key
title         varchar not null
applicant     varchar not null
amount        int
reason        text
status        varchar not null
comment       text
created_at    timestamptz not null
updated_at    timestamptz not null
```

索引：

```text
index(status)
index(created_at)
index(status, created_at)
```

原因：

```text
GET /business/approval/items?status=pending 需要按 status 过滤
列表通常按 created_at 排序
```

## 3.10 calendar_events

保存日程演示数据。

字段建议：

```text
id             varchar primary key
title          varchar not null
description    text
start_time      timestamptz not null
end_time        timestamptz not null
location        varchar
participants_json jsonb
created_at      timestamptz not null
updated_at      timestamptz not null
```

索引：

```text
index(start_time)
index(start_time, end_time)
```

原因：

```text
GET /business/calendar/events?date=2026-05-31 需要按日期范围查询
后续查某个时间段日程也会用 start_time / end_time
```

## 4. 推荐实现顺序

### 第一步：只接数据库连接

新增：

```text
internal/database/database.go
internal/database/models.go
```

先做到：

```text
APP_MODE=postgres DATABASE_URL=xxx go run ./cmd/server
```

能连接数据库并打印：

```text
database connected
auto migrate completed
```

### 第二步：AutoMigrate 建表

先建这些核心表：

```text
users
auth_sessions
permissions
user_permissions
tabs
user_tabs
```

原因：

```text
这几张表能支撑登录、鉴权、GET /tabs、POST /me/tabs
先打通主链路
```

### 第三步：初始化基础数据

插入当前 mock 中已有账号：

```text
opentab-demo / demo123
opentab-admin / admin123
opentab-guest / guest123
```

插入当前权限：

```text
tab.approval.read
tab.calendar.read
tab.finance.read
ai.oncall
```

插入当前系统 Tab：

```text
approval
calendar
finance
next
docs
```

插入默认 user_tabs：

```text
demo: approval, calendar, next
admin: approval, calendar, finance, next, docs
guest: docs
```

### 第四步：替换 UserRepository

先让：

```text
POST /auth/login
GET /me
```

从数据库读用户。

验收：

```text
用 demo/admin/guest 都能登录
带 token 能访问 /me
go test ./... 通过
```

### 第五步：替换 TabRepository

让：

```text
GET /tabs
GET /tabs/catalog
POST /me/tabs
DELETE /me/tabs/{tabId}
POST /tabs
PUT /tabs/{tabId}
DELETE /tabs/{tabId}
PUT /me/tabs/order
```

从数据库读写。

验收：

```text
启用 Tab 后重启服务，启用状态仍然存在
创建自定义 Web Tab 后重启服务，Tab 仍然存在
```

### 第六步：替换 OnCallRepository

让：

```text
POST /oncall/sessions
GET /oncall/sessions
POST /oncall/sessions/{sessionId}/messages
GET /oncall/sessions/{sessionId}/messages
DELETE /oncall/sessions/{sessionId}
```

从数据库读写。

验收：

```text
用户 A 的会话不会出现在用户 B
服务重启后聊天记录仍然存在
```

### 第七步：替换 BusinessRepository

让审批和日程接口从数据库读写。

验收：

```text
审批通过/驳回后重启服务，状态仍然存在
新增日程后重启服务，日程仍然存在
```

## 5. GORM 模型建议

GORM 模型建议使用 `gorm.Model` 或显式时间字段。

学习阶段建议显式字段，更容易看懂：

```go
type UserRecord struct {
    ID           string    `gorm:"primaryKey;size:64"`
    Account      string    `gorm:"uniqueIndex;size:64;not null"`
    DisplayName  string    `gorm:"size:128;not null"`
    PasswordHash string    `gorm:"size:255;not null"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

JSONB 字段可以使用：

```go
datatypes.JSON
```

需要引入：

```text
gorm.io/datatypes
```

## 6. 索引策略总结

第一版必须有的索引：

```text
users.account unique
auth_sessions.token unique
user_permissions(user_id, permission_code) primary key
tabs.id primary key
tabs(owner_user_id)
user_tabs(user_id, tab_id) primary key
user_tabs(user_id, sort_order)
oncall_sessions(user_id, updated_at)
oncall_messages(session_id, created_at)
approval_items(status, created_at)
calendar_events(start_time, end_time)
```

为什么这些索引重要：

```text
登录按 account 查
鉴权按 token 查
Tab 列表按 user_id 查
Tab 排序按 sort_order 排
聊天记录按 session_id 查
审批列表按 status 查
日程列表按时间范围查
```

## 7. 数据一致性要注意什么

### 7.1 创建自定义 Tab 要用事务

`POST /tabs` 实际要做两件事：

```text
1. 插入 tabs
2. 插入 user_tabs，默认启用
```

这两个操作必须一起成功或一起失败。

所以要用事务：

```go
db.Transaction(func(tx *gorm.DB) error {
    // create tab
    // create user_tab
    return nil
})
```

### 7.2 删除自定义 Tab 要检查 owner

用户只能删除自己创建的自定义 Tab。

判断条件：

```text
tabs.owner_user_id == currentUser.ID
tabs.is_system == false
```

系统内置 Tab 不允许删除。

### 7.3 查询用户 Tab 要同时看权限和启用关系

`GET /tabs` 推荐查询逻辑：

```text
1. 根据 currentUser.ID 查询 user_tabs
2. join tabs
3. enabled=true
4. 按 user_tabs.sort_order 排序
5. 返回 TabManifest
```

权限不足的 Tab 是否返回：

```text
当前项目建议：已启用就返回，由客户端显示受限原因
启用新 Tab 时服务端必须检查权限
```

### 7.4 token 失效策略

第一版可以设置较长过期时间，例如：

```text
7 天
```

每次请求：

```text
1. 按 token 查 auth_sessions
2. token 不存在 -> 401
3. revoked_at 不为空 -> 401
4. expires_at 已过期 -> 401
5. 找到 user -> 放入 middleware currentUser
```

## 8. 初始化数据策略

初始化数据要可重复执行。

不要简单 `Create`，否则第二次启动会重复插入或报错。

建议使用：

```go
FirstOrCreate
```

或者：

```go
OnConflict DoNothing / DoUpdates
```

目标：

```text
服务重启时不会重复制造数据
缺少基础数据时能自动补齐
```

## 9. 测试策略

保留当前 HTTP 测试。

新增数据库测试时分两类：

### 9.1 repository 单元/集成测试

测试重点：

```text
PostgresUserRepository.FindByAccount
PostgresUserRepository.FindByToken
PostgresTabRepository.ListByUser
PostgresTabRepository.Enable / Disable
PostgresOnCallRepository.AddMessage / ListMessages
```

### 9.2 HTTP 回归测试

当前已有 `routes/router_test.go`，可以继续跑 mock 模式。

Postgres 模式可以后续加：

```text
go test ./... -tags=integration
```

第一版不强制，因为需要本地或 CI 有 PostgreSQL。

## 10. 常见问题

### 10.1 为什么不直接把 mockdata 写进数据库？

可以把 mockdata 当作初始化数据来源，但运行时不要继续依赖 mockdata。

最终应该是：

```text
mock mode -> mockdata
postgres mode -> database
```

### 10.2 为什么 tabs.permissions 先用 JSONB，不拆 tab_permissions？

当前 Tab 权限只是“打开需要哪些权限”，查询需求很少，主要是返回给客户端和 service 校验。

第一版用 JSONB 更简单。

如果后续需要：

```text
查哪些 Tab 需要某权限
权限变更影响哪些 Tab
复杂权限分析
```

再拆：

```text
tab_permissions(tab_id, permission_code)
```

### 10.3 为什么 user_tabs 单独一张表？

因为 Tab 配置和用户启用关系不是一回事。

同一个系统 Tab：

```text
admin 可以启用
guest 可以不启用
demo 可以排序到第一位
```

所以必须有 `user_tabs`。

### 10.4 服务端重启后客户端登录会怎样？

如果 token 存在 `auth_sessions` 表，重启后仍然能查到 token，客户端不需要重新登录。

如果 token 只存在内存，重启后 token 丢失，客户端会收到 401。

因此正式阶段 token 必须持久化，或者使用 JWT。

## 11. 最小验收标准

数据库接入完成后，至少满足：

```text
1. APP_MODE=postgres 能启动
2. AutoMigrate 能创建表
3. 初始 demo/admin/guest 账号可登录
4. GET /tabs 从数据库返回
5. POST /me/tabs 写入数据库
6. POST /tabs 创建自定义 Web Tab 并持久化
7. POST /oncall/sessions 创建会话并持久化
8. POST /oncall/sessions/{sessionId}/messages 保存消息
9. 审批状态修改后重启仍保留
10. go test ./... 通过
```

## 12. 我的实现建议

接下来不要一次性把所有 repository 都换成 PostgreSQL。

建议顺序：

```text
1. 先接 GORM + AutoMigrate
2. 初始化 users / permissions / tabs / user_tabs
3. 替换 AuthRepository
4. 替换 TabRepository
5. 再替换 OnCallRepository
6. 最后替换 BusinessRepository
```

理由：

```text
登录和 Tab 是客户端主链路
AI 和业务数据可以在主链路稳定后再持久化
每一步都能测试，不容易一次改太多出错
```
