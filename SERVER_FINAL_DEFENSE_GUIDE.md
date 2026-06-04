# OpenTab 服务端项目答辩讲解稿

这份文档按“当前版本作为终版”来介绍服务端项目。它不是接口文档，也不是开发日志，而是一份可以直接用于答辩准备的讲稿。

答辩主线建议固定为：

```text
我负责的是开放式 Tab 容器的服务端控制面。

它负责账号登录态、权限隔离、Tab 配置下发、内置业务数据、AI OnCall 会话中转、操作审计和云端演示环境恢复。
```

## 1. 开场：我这个服务端到底是什么

建议说：

```text
我的服务端定位是支撑开放式 Tab 容器和 AI OnCall 的后端控制面。

它使用 Go + Gin 提供 HTTP 接口，使用 PostgreSQL + GORM 做数据持久化，同时保留 memory 模式方便本地测试。它承担的职责包括：控制用户能看到哪些 Tab，控制不同角色能访问哪些业务数据，并把 AI OnCall 接入到服务端会话体系里。
```

现场可以展示：

- 客户端登录页面。
- 云服务器访问地址。
- Navicat 里的 PostgreSQL 表。

可以打开：

```text
cmd/server/main.go
```

讲法：

```text
这里是服务端启动入口。它读取配置，初始化 Gin 路由。如果配置了 DATABASE_URL，就连接 PostgreSQL，执行 AutoMigrate 建表，再执行 Seed 写入默认数据。最后根据 APP_MODE 决定使用 memory repository 还是 postgres repository。
```

重点指出：

- `config.Load()`
- `database.Connect`
- `database.AutoMigrate`
- `database.Seed`
- `repositories.NewPostgresRepositorySet`
- `routes.RegisterWithStatus`

这部分要让老师先知道：这是一个能部署、能连库、能支撑客户端联调和演示的服务端项目。

## 2. 总体架构：我怎么组织服务端代码

建议说：

```text
我把服务端拆成 routes、services、policies、repositories、database、middleware 几层。这样做的目的不是为了堆目录，而是为了让每一层职责清楚。

routes 负责接 HTTP 请求，services 负责业务规则，policies 负责权限策略，repositories 负责数据读写，database 负责表结构和初始化，middleware 负责鉴权、请求追踪和审计日志。
```

可以打开：

```text
internal/routes/router.go
internal/repositories/factory.go
```

讲 `router.go` 时说：

```text
这里能看到所有接口分组。公开接口只有 health、登录、注册等，绝大多数业务接口都挂在 authorized group 下，统一经过 Auth 中间件。

这说明我的鉴权不是每个接口手写，而是在路由层统一收口。
```

重点指出：

- `router.Use(middleware.RequestID())`
- `router.Use(middleware.Audit(handler.audit))`
- `authorized.Use(middleware.Auth(handler.auth))`
- `/tabs`
- `/business`
- `/admin`
- `/oncall`

讲 `factory.go` 时说：

```text
RepositorySet 把数据访问层抽象成统一接口。memory 和 postgres 两套实现可以互换，上层 service 不关心数据来自内存还是数据库。

这个设计让我可以在本地测试时不用依赖数据库，在云服务器运行时切到 PostgreSQL。
```

重点指出：

- `NewMemoryRepositorySet`
- `NewPostgresRepositorySet`
- `Users/Tabs/Business/OnCall/Debug/Audit`

## 3. 亮点一：登录态不是固定 token，而是完整生命周期

建议说：

```text
登录态如果只依赖固定 token，会缺少过期、吊销和多设备管理能力。所以我把登录态设计成了 auth_sessions 表。

用户每次登录都会生成随机 token，服务端保存 token、user_id、expires_at、revoked_at。后续接口必须带 Bearer Token。logout 时不是只让客户端清 token，而是服务端把当前 token 标记为 revoked。
```

可以打开：

```text
internal/services/auth_service.go
internal/middleware/auth.go
internal/database/models.go
```

讲 `auth_service.go` 时重点指出：

- `Login`
- `Register`
- `newAccessToken`
- `tokenExpiresAt`
- `Logout`

讲法：

```text
Login 先校验账号密码，成功后生成随机 token，再调用 CreateSession 写入数据库。token 默认 7 天有效。

Logout 会调用 RevokeToken，所以退出登录后这个 token 在服务端失效。这样 token 泄漏时至少可以通过吊销和过期降低风险。
```

讲 `auth.go` 时重点指出：

- `Auth`
- `FindUserByToken`
- `ErrTokenExpired`
- `ErrTokenRevoked`
- `ErrUserDisabled`

讲法：

```text
鉴权中间件统一检查 token 是否存在、是否过期、是否被吊销、用户是否被禁用。业务接口不用重复写这些判断。
```

讲 `models.go` 时找到：

```text
AuthSessionRecord
```

讲法：

```text
这里是 auth_sessions 表结构。一个用户可以有多个 session，所以同一账号可以多设备登录。logout 只吊销当前设备的 token，不影响其他设备。
```

现场可以展示：

```sql
SELECT id, user_id, expires_at, revoked_at, created_at
FROM auth_sessions
ORDER BY created_at DESC
LIMIT 10;
```

## 4. 亮点二：密码安全和旧数据兼容

建议说：

```text
服务端不再保存明文密码，而是使用 bcrypt 保存 password_hash。这里我还考虑了旧数据兼容：如果数据库里曾经有明文密码，用户登录成功后会自动升级成 bcrypt hash。
```

可以打开：

```text
internal/security/password.go
internal/services/auth_service.go
internal/database/seed.go
```

讲 `password.go`：

```text
这里封装了 HashPassword、VerifyPassword、IsBcryptHash。service 层不直接操作 bcrypt 细节，而是通过 security 包统一处理。
```

讲 `auth_service.go`：

```text
登录时通过 VerifyPassword 校验。如果发现不是 bcrypt hash，登录成功后会调用 UpdatePasswordHash 自动升级。
```

讲 `seed.go`：

```text
默认账号 seed 时也会写入 bcrypt hash，而不是明文密码。
```

现场可以展示：

```sql
SELECT account, display_name, password_hash
FROM users
ORDER BY account;
```

讲的时候提醒：

```text
这里展示 password_hash，不展示真实密码。默认演示账号密码只是为了答辩和联调方便。
```

## 5. 亮点三：用户数据隔离，不相信客户端传 userId

建议说：

```text
我在服务端做数据隔离时，没有让客户端传 userId 决定查谁的数据。客户端只带 token，服务端从 token 解析当前用户，再根据用户身份、团队、角色和权限查询数据。

这样可以避免客户端伪造 userId 去看别人的审批、日程、AI 会话。
```

可以打开：

```text
internal/middleware/auth.go
internal/services/business_service.go
internal/repositories/postgres_business_repository.go
internal/repositories/postgres_oncall_repository.go
```

讲 `auth.go`：

```text
Auth 中间件查出当前用户后，会把 user 放到 Gin context 里。后续业务接口都是从 context 里拿当前用户，而不是相信请求参数里的 userId。
```

讲 `business_service.go`：

```text
service 层拿到当前 user 后，再判断能不能创建审批、能不能审批、能不能管理日程、能不能发布公告。
```

讲 `postgres_business_repository.go`：

```text
列表查询会结合 user.ID、currentTeamId、memberships 做过滤。比如日程能看到 company、自己团队、自己创建或自己参与的内容。
```

讲 `postgres_oncall_repository.go`：

```text
AI 会话按 user_id 查询，所以不同账号的聊天记录隔离。
```

现场可以展示：

- 用 `product-manager` 登录看产品研发部数据。
- 用 `operation-employee` 登录看运营支持部数据。
- 在 Navicat 中查 `approval_items`、`calendar_events` 的 `user_id/team_id`。

可讲一句：

```text
这里体现的是数据边界：用户看到什么不是客户端决定的，而是服务端根据登录态和权限策略决定的。
```

## 6. 亮点四：权限不是简单 admin，而是权限码 + 团队角色

建议说：

```text
我没有只做一个简单 admin/普通用户，而是设计了三层权限：全局角色、团队角色、权限码。

权限码决定能不能使用某个功能，团队角色决定能操作哪个范围的数据。
```

可以打开：

```text
internal/policies/authorization.go
internal/mockdata/users.go
internal/database/models.go
```

讲 `authorization.go`：

```text
这里集中定义权限策略，比如 CanManageTeam、CanViewApproval、CanApproveTeamApproval、CanManageCalendar、CanWriteAnnouncement。

这样权限规则不是散落在每个接口里，而是集中在 policies 层。
```

讲 `users.go`：

```text
这里是默认演示用户。不同账号有不同权限码，比如 manager 有审批和日程管理权限，employee 更多是查看和创建权限。
```

讲 `models.go`：

```text
teams、team_members、permissions、user_permissions 这些表共同支撑团队和权限模型。
```

现场可以展示：

```sql
SELECT * FROM team_members;
SELECT * FROM user_permissions ORDER BY user_id, permission_code;
```

可讲一句：

```text
这个设计为后续团队账号、多团队、RBAC 扩展留了空间，但当前阶段没有过度复杂化。
```

## 7. 亮点五：Tab 控制面，服务端下发工作台结构

建议说：

```text
开放式 Tab 容器的重点不是简单跳页面，而是客户端工作台结构可以由服务端控制。

服务端维护 tabs 和 user_tabs：tabs 表示 Tab 本身是什么，user_tabs 表示某个用户启用了哪些 Tab，以及排序。
```

可以打开：

```text
internal/models/tab.go
internal/mockdata/tabs.go
internal/services/tab_service.go
internal/repositories/postgres_tab_repository.go
```

讲 `tab.go`：

```text
TabManifest 描述了一个 Tab 的核心信息，比如 id、displayName、icon、route、entryType、entryURI、version、permissions、extension、extraConfig。
```

讲 `tabs.go`：

```text
这里是系统内置 Tab，比如审批中心、团队日程、公告、接入文档。不同用户默认启用不同 Tab。
```

讲 `tab_service.go`：

```text
service 层处理 Tab 启用、停用、排序、创建自定义 Tab 和校验。
```

讲 `postgres_tab_repository.go`：

```text
这里把 TabManifest 映射到数据库 tabs/user_tabs。这样客户端工作台不是完全写死，而是由服务端下发配置。
```

现场可以展示：

- 客户端工作台。
- `/tabs` 当前用户启用 Tab。
- `/tabs/catalog` 可选 Tab。
- Navicat 查 `tabs` 和 `user_tabs`。

可讲一句：

```text
这部分是我和项目主题最强相关的设计：服务端作为开放式 Tab 容器的控制面。
```

## 8. 亮点六：内置业务 Tab 不是本地假数据，而是服务端业务接口

建议说：

```text
为了让审批、日程、公告这些内置 Tab 真正可联调，我给它们提供了服务端业务接口和数据库表，而不是依赖客户端本地写死数据。
```

可以打开：

```text
internal/models/business.go
internal/routes/business.go
internal/services/business_service.go
internal/repositories/postgres_business_repository.go
```

讲 `business.go`：

```text
这里定义审批、日程、公告的请求和响应模型。
```

讲 `routes/business.go`：

```text
这里是业务接口入口，包括审批列表、详情、创建、通过、拒绝、撤回，日程的增删改查，公告的增删改查。
```

讲 `business_service.go`：

```text
这里是业务规则，比如审批只有 pending 状态能处理，员工不能管理团队日程，只有有权限的人能发布公告。
```

讲 `postgres_business_repository.go`：

```text
repository 负责数据库查询和更新，同时在列表查询中落实数据可见性。
```

现场可以展示：

- 审批中心。
- 团队日程，注意演示数据时间已经错开。
- 公告页面。
- Navicat 查：

```sql
SELECT id, team_id, title, applicant, approver, status FROM approval_items;
SELECT id, team_id, title, start_time, end_time FROM calendar_events ORDER BY start_time;
SELECT id, scope, team_id, title, publisher_name FROM announcements;
```

可讲一句：

```text
这些业务数据不是孤立 CRUD，而是和当前登录用户、团队、权限相关联。
```

## 9. 亮点七：AI OnCall 不是客户端直连 AI，而是服务端中转和会话管理

建议说：

```text
AI OnCall 部分不是让客户端直接请求 AI 服务，而是客户端请求我的服务端。我的服务端创建会话、保存消息、调用 AI 服务、解析流式返回，再返回给客户端。
```

可以打开：

```text
internal/routes/oncall.go
internal/services/oncall_service.go
internal/repositories/postgres_oncall_repository.go
internal/models/oncall.go
```

讲 `routes/oncall.go`：

```text
这里是 AI OnCall 的接口入口，包括创建会话、拉取会话列表、保存消息、拉取消息、流式对话和删除会话。
```

讲 `oncall_service.go`：

```text
这里负责服务端到 AI 服务的调用和 SSE 返回解析。之前联调时 AI 服务返回格式和预期不一致，所以我在服务端做了适配和容错。
```

讲 `postgres_oncall_repository.go`：

```text
oncall_sessions 和 oncall_messages 都按 user_id/session_id 存储，保证不同用户聊天记录隔离。
```

现场可以展示：

- 客户端 AI OnCall 对话。
- Navicat 查：

```sql
SELECT id, user_id, title, created_at, updated_at
FROM oncall_sessions
ORDER BY updated_at DESC;

SELECT session_id, role, content, created_at
FROM oncall_messages
ORDER BY created_at DESC
LIMIT 20;
```

可讲一句：

```text
这个设计的好处是，AI 服务可以作为后端能力接入系统，而不是客户端的一个孤立聊天窗口。
```

## 10. 亮点八：操作审计日志和 requestId，可溯源

建议说：

```text
为了让服务端具备可追踪能力，我新增了 RequestID 和 Audit 中间件。

每个请求会生成 requestId，错误响应会带 traceId。关键操作会写入 audit_logs 表，包括用户、接口、操作类型、状态码、错误码、IP、耗时等。
```

可以打开：

```text
internal/middleware/request_id.go
internal/middleware/audit.go
internal/database/models.go
internal/repositories/postgres_audit_repository.go
internal/response/response.go
```

讲 `request_id.go`：

```text
这里会为每个请求生成 X-Request-Id。如果客户端已经传了 X-Request-Id，也可以沿用。
```

讲 `audit.go`：

```text
Audit 中间件在请求结束后统一记录操作，不需要每个接口手写日志。它不记录请求体，避免把密码等敏感信息写入日志。
```

讲 `models.go`：

```text
AuditLogRecord 是 audit_logs 表结构。这里记录 request_id、user_id、action、status_code、result、error_code、duration_ms 等。
```

讲 `response.go`：

```text
错误响应会带 traceId。客户端出错时，可以拿 traceId 到服务端查审计日志和运行日志。
```

现场可以展示：

```sql
SELECT request_id, user_id, account, action, method, path, status_code, result, error_code, duration_ms, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT 20;
```

可讲一句：

```text
运行日志回答服务有没有报错，审计日志回答谁做了什么，requestId 把一次请求串起来。这个功能让系统出了问题后可以追踪和复盘。
```

## 11. 亮点九：云端部署和演示环境恢复

建议说：

```text
为了支持客户端同学联调和答辩演示，我把服务端部署到了云服务器，并写了数据库初始化、演示数据重置和部署脚本。

演示数据被污染后，可以重新建库、写入固定演示数据、启动服务，减少临场风险。
```

可以打开：

```text
scripts/init_postgres_linux.sh
scripts/reset_demo_data.sql
scripts/reset_demo_data.sh
scripts/deploy_demo_reset.sh
scripts/DEPLOY_FROM_GIT.md
```

讲 `init_postgres_linux.sh`：

```text
这个脚本创建 PostgreSQL 用户和数据库。如果设置 OPENTAB_DB_RESET=true，会重建数据库。
```

讲 `reset_demo_data.sql`：

```text
这个脚本清理脏数据并写入一组固定演示数据，包括用户显示名、审批、日程、公告等。
```

讲 `deploy_demo_reset.sh`：

```text
这个脚本用于演示前部署：拉取最新代码、停止旧服务、重建数据库、跑测试、启动服务、写入演示数据、检查 health。
```

现场可以展示：

```bash
curl http://127.0.0.1:8080/health
tail -n 80 server.err.log
```

可讲一句：

```text
这部分体现的是我考虑了联调和演示环境的可恢复性，而不只是本地代码能跑。
```

## 12. 我的个人巧思和取舍

这部分适合老师追问“你的思考在哪里”时讲。

### 12.1 没有盲目堆接口，而是先保证控制面

可以说：

```text
这个项目容易变成单纯堆 CRUD。我最后把主线收束到后端控制面：账号、权限、Tab 配置、业务数据、AI 会话都围绕当前登录用户展开。
```

### 12.2 保留 memory 模式，不是多余

可以说：

```text
memory 模式让测试不依赖数据库，postgres 模式用于真实部署。两套 repository 共用同一套 service，能检验业务逻辑和存储实现是否解耦。
```

### 12.3 权限码和团队角色分开

可以说：

```text
权限码解决能不能使用功能，团队角色解决能操作哪个范围的数据。这样比单纯 admin/普通用户更贴近团队工作台场景。
```

### 12.4 审计日志不记录请求体

可以说：

```text
审计日志是为了追踪操作，不是为了保存所有数据。登录请求里有密码，所以我没有记录请求体，避免日志变成新的敏感数据风险。
```

### 12.5 AutoMigrate 适合当前阶段，但我知道它的边界

可以说：

```text
当前阶段我用 GORM AutoMigrate 快速同步表结构。它适合开发和联调，但正式环境后续应该引入 migrations，把 schema 版本管理起来。
```

### 12.6 AI 生成代码后，我做了验证和修正

可以说：

```text
我确实使用 AI 辅助生成代码和文档，但我没有把 AI 输出直接当最终答案。比如 token 是否永久、logout 是否真的吊销、不同用户数据是否隔离、AI SSE 是否解析正确，这些都是通过测试、curl、日志和数据库检查不断修正的。
```

## 13. 如果老师问“你的服务端亮点在哪里”

可以回答：

```text
我的服务端亮点不在于接口数量，而在于它覆盖了开放式 Tab 容器服务端需要处理的几个核心问题：

第一，有登录态生命周期，token 可以过期和吊销。

第二，有用户、团队、权限和数据隔离，不同账号看到的数据不同。

第三，有 Tab 控制面，服务端下发用户启用的 Tab 和 TabManifest。

第四，有 PostgreSQL 持久化，审批、日程、公告、AI 会话都能保存。

第五，有 requestId 和 audit_logs，能追踪用户操作。

第六，有云端部署和演示数据恢复脚本，支持真实联调。
```

## 14. 如果老师问“目前还有什么不足”

可以回答：

```text
当前版本已经能支撑联调和演示，但我认为还有几个后续优化方向：

第一，数据库迁移还主要依赖 AutoMigrate，后续应该引入正式 migrations。

第二，审计日志目前是服务端内部记录，还没有给管理员提供查询页面或查询接口。

第三，审批和日程可以进一步增加并发保护，比如乐观锁或条件更新。

第四，部署目前使用 nohup，后续更正规应该使用 systemd 和 Nginx。

第五，AI OnCall 后续可以和 audit_logs、requestId 结合，让 AI 根据服务端错误上下文辅助诊断。
```

这样回答的重点是：承认不足，但不足都是工程演进方向，不是当前版本不能跑。

## 15. 现场演示推荐顺序

建议不要上来就翻代码。先演示效果，再讲代码支撑。

### 第一步：登录

演示账号：

```text
admin / admin123
product-manager / manager123
product-employee / employee123
operation-manager / manager123
operation-employee / employee123
```

展示：

- 登录成功。
- 不同账号显示名不同。
- 工作台 Tab 不完全一样。

然后打开：

```text
internal/services/auth_service.go
internal/middleware/auth.go
```

讲登录态和鉴权。

### 第二步：Tab 工作台

展示：

- 当前用户启用的 Tab。
- Tab 可启用/停用/排序。

然后打开：

```text
internal/models/tab.go
internal/services/tab_service.go
internal/repositories/postgres_tab_repository.go
```

讲服务端控制面。

### 第三步：审批和日程

展示：

- 审批列表。
- 审批处理。
- 日程列表，时间错开。

然后打开：

```text
internal/services/business_service.go
internal/policies/authorization.go
internal/repositories/postgres_business_repository.go
```

讲业务规则和权限隔离。

### 第四步：AI OnCall

展示：

- 发起一次 AI 对话。
- Navicat 查 `oncall_sessions`、`oncall_messages`。

然后打开：

```text
internal/routes/oncall.go
internal/services/oncall_service.go
```

讲服务端中转和会话保存。

### 第五步：审计日志

展示：

```sql
SELECT request_id, account, action, path, status_code, result, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT 20;
```

然后打开：

```text
internal/middleware/audit.go
internal/middleware/request_id.go
internal/response/response.go
```

讲可溯源。

### 第六步：部署脚本

展示：

```text
scripts/deploy_demo_reset.sh
scripts/reset_demo_data.sql
```

讲云端部署和演示数据恢复。

## 16. 最后 30 秒总结

可以直接照着说：

```text
总结一下，我这个服务端的重点不是接口数量，而是围绕开放式 Tab 容器做一个后端控制面。

它解决了几个真实后端问题：用户身份怎么可信，token 怎么过期和吊销，密码怎么安全保存，不同用户和团队的数据怎么隔离，客户端工作台 Tab 怎么由服务端控制，AI OnCall 怎么纳入服务端会话体系，以及出了问题怎么通过 requestId 和 audit_logs 追踪。

我在这个过程中也使用了 AI 辅助开发，但主要工作不是简单复制 AI 代码，而是不断确认需求边界、验证接口行为、检查数据库状态、补测试、重构分层，并把代码调整成我能解释清楚的后端设计。
```
