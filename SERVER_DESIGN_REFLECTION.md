# OpenTab 服务端设计思路与个人复盘

这份文档从我的个人视角出发，总结 OpenTab 服务端从 mock 联调服务逐步演进到具备真实后端特征的过程。它不是接口文档，而是我对服务端架构、权限、数据隔离、数据库设计、AI 协作开发方式的整理。

## 1. 我一开始面对的问题

项目最开始的目标比较模糊：客户端要做开放式 Tab 容器，AI OnCall 要提供辅助能力，服务端既要支撑客户端联调，又不能一开始就设计得过重。

我当时主要面临几个问题：

- 客户端进度不确定，不能等客户端完全定型后再写服务端。
- 业务 Tab 接入协议和客户端服务端接口文档不是一开始就完全一致。
- mock 服务要够客户端联调，但后续又要能接 PostgreSQL。
- 用户、团队、权限、Tab、审批、日程、公告、AI 会话这些数据之间有明显边界，不能简单平铺 CRUD。
- 我自己也还在学习 Go、Gin、PostgreSQL 和后端分层设计。

所以我的整体策略不是一上来追求完整企业级后端，而是先保证“接口能跑通”，再逐步把 mock 后端改造成更接近真实系统的服务端。

## 2. 整体架构思路

当前服务端主要采用分层结构：

```text
cmd/server
  启动入口，读取配置，初始化数据库和路由

internal/config
  读取 APP_MODE、DATABASE_URL、AI_SERVICE_BASE_URL 等配置

internal/routes
  Gin 路由层，负责解析 HTTP 请求和返回 response

internal/services
  业务服务层，负责权限判断、状态流转、业务规则

internal/policies
  权限策略层，集中判断 admin、团队角色、数据可见性

internal/repositories
  数据仓库层，封装 memory/postgres 两种数据来源

internal/database
  GORM 数据模型、自动建表和种子数据

internal/models
  请求、响应和业务模型

internal/response
  统一响应格式

internal/middleware
  鉴权、RequestID、审计日志中间件
```

我对这个分层的理解是：

- `routes` 不写业务，只负责接请求。
- `services` 写业务规则，比如“谁能审批”“什么状态能撤回”。
- `policies` 写权限判断，比如“是否是 admin”“是否是团队主管”。
- `repositories` 写数据读写，比如从 PostgreSQL 查审批、更新日程。
- `database` 写表结构和初始化。
- `middleware` 处理横切能力，例如鉴权、请求追踪和审计日志。

这个分层不是一开始就完全设计好的，而是在多次修改中逐渐清晰的。最开始我也有很多逻辑直接写在 route 或 repository 里，后来发现这样会导致权限规则分散，于是才逐步把业务规则收回 service，把权限判断抽到 policies。

## 3. 为什么保留 memory 和 PostgreSQL 两套 repository

服务端现在有两种运行模式：

```text
APP_MODE=memory
APP_MODE=postgres
```

这样设计主要是为了降低开发和联调成本。

在早期阶段，客户端同学只需要能调接口，不关心数据库。memory 模式可以快速返回 mock 数据，方便改接口、改字段、跑测试。

后面接 PostgreSQL 后，我没有直接删掉 memory 模式，原因是：

- 单元测试可以不用依赖真实数据库。
- 如果云服务器数据库暂时不可用，服务端仍然可以用 memory 模式验证接口。
- 两套 repository 共用同一套 service，能检验业务逻辑是否真的和存储实现解耦。

这也让我更清楚 repository 的职责：它应该屏蔽数据来源差异，而不是把业务规则写死在具体数据库实现里。

## 4. 鉴权设计思路

鉴权目前经历了几个阶段。

第一阶段是固定 token：

```text
账号登录 -> 返回固定 mock token
```

这种方式适合最早期联调，但问题很明显：

- token 永不过期。
- logout 没有服务端意义。
- token 泄漏后无法吊销。

后来改成了 `auth_sessions` 表：

```text
users
auth_sessions
```

登录时生成随机 token，写入：

- `user_id`
- `token`
- `expires_at`
- `revoked_at`

鉴权中间件会检查：

- token 是否存在
- token 是否过期
- token 是否被 logout 吊销
- 用户是否被禁用

现在的流程是：

```text
客户端提交账号密码
-> 服务端校验密码
-> 生成随机 token
-> token 写入 auth_sessions
-> 客户端后续带 Bearer Token
-> 中间件解析 token
-> 加载用户、权限、团队身份
-> 进入业务接口
```

我认为这个设计的重点不是“有 token 就行”，而是要能解释登录态的生命周期：

- token 为什么会过期
- logout 为什么不只是客户端清 token
- token 被盗后有什么风险
- 服务端如何吊销 token

## 5. 密码加密设计

早期数据库字段叫 `password_hash`，但实际存的是明文密码。这个问题在真实后端里很明显不合理。

后来我改成 bcrypt：

```text
注册时：明文密码 -> bcrypt hash -> 存数据库
登录时：输入密码 + 数据库 hash -> bcrypt compare
```

为了不破坏已有云服务器数据，我没有直接要求清库，而是做了兼容：

- 如果数据库里已经是 bcrypt hash，就按 bcrypt 校验。
- 如果数据库里还是旧明文，就先按明文校验。
- 明文校验成功后，自动升级为 bcrypt hash。

这个设计让我意识到：后端升级不能只考虑“新代码是正确的”，还要考虑已有数据怎么平滑迁移。

## 6. 权限设计思路

当前权限主要分三类：

### 6.1 全局角色

例如：

```text
globalRole = admin
```

admin 可以做系统级管理，比如团队管理、查看全局数据。

### 6.2 团队角色

用户可以属于团队：

```text
team-product
team-operation
```

团队内角色：

```text
manager
employee
```

团队主管可以管理团队内部分业务，例如审批团队申请、管理团队日程、发布团队公告。

### 6.3 权限码

例如：

```text
tab.approval.read
tab.approval.create
tab.approval.approve
tab.calendar.read
tab.calendar.manage
tab.announcement.write
team.manage
ai.oncall
```

权限码解决的是“这个用户有没有使用某类功能的资格”。团队角色解决的是“他能操作哪个范围的数据”。

所以我现在的理解是：

```text
权限码决定能不能进功能
团队/角色决定能操作哪些数据
```

比如一个用户有 `tab.approval.approve`，但如果他不是对应团队 manager，也不能审批别的团队的审批。

## 7. 数据隔离设计思路

服务端目前最重要的设计之一是：客户端不能传一个 `userId` 就随便查数据。

正确流程是：

```text
客户端带 token
-> 服务端根据 token 查出当前 user
-> 服务端用当前 user.ID / currentTeamId / memberships 查询数据
```

也就是说，数据隔离的依据来自服务端解析 token 后得到的身份，而不是客户端传什么。

不同数据的隔离方式：

```text
AI 会话：user_id
自定义 Tab：owner_user_id / user_tabs
审批：applicant_id / user_id / team_id
日程：creator_id / participant_ids / team_id / visibility
公告：scope / team_id
团队成员：team_id
```

这里我比较重视的一点是：数据隔离不是只在 service 判断一次，还要落实到 repository 查询条件里。

比如查询日程时，普通用户只能看到：

- company 可见日程
- 自己团队的 team 日程
- 自己创建的日程
- 自己参与的日程

这类过滤更适合在 SQL 查询里做，因为它直接影响查询结果和性能。

## 8. 为什么把权限策略抽成 policies

一开始权限判断散落在 service 和 repository 中，例如：

```go
user.GlobalRole == "admin"
membership.TeamRole == "manager"
```

这样的问题是：

- 多处重复。
- 后续要改规则很难统一。
- 很难解释“权限规则到底在哪里”。

后来我抽出了 `internal/policies`：

```go
CanManageTeam()
CanReadTeamMembers()
CanCreateApproval()
CanViewApproval()
CanApproveTeamApproval()
CanViewCalendar()
CanManageCalendar()
CanWriteAnnouncement()
```

这样做之后，我可以更清楚地说：

```text
service 负责调用策略
policies 负责定义策略
repository 负责执行查询
```

这也是我从“把代码写出来”到“让代码可解释”的一个转变。

## 9. Service 和 Repository 职责调整

后来我又做了一次架构优化，把一部分业务规则从 repository 移到了 service。

原因是我发现 repository 里如果同时做这些事：

- 查数据库
- 判断谁能审批
- 判断审批状态是否 pending
- 判断谁能删公告

就会变成“数据层也在决定业务规则”。

现在我更倾向于：

```text
service：判断业务规则
repository：执行数据读写
```

例如审批：

```text
service:
  1. 查审批详情
  2. 判断状态是否 pending
  3. 判断用户是否有权审批
  4. 调 repository 更新状态

repository:
  1. 根据 id 查记录
  2. 更新 status/comment
```

这样做的好处是：

- 业务规则集中在 service，更容易读。
- repository 更接近数据库访问层。
- 测试 service 时可以用 fake repository，不依赖真实数据库。

但我也没有把所有过滤都搬出 repository。比如列表查询的数据可见性仍然保留在 repository，因为它和 SQL 查询条件、索引、性能直接相关。

这是一个取舍：不是机械地说 repository 不能出现任何判断，而是区分“业务决策”和“查询过滤”。

## 10. 数据库表设计思路

当前主要表包括：

```text
users
auth_sessions
teams
team_members
permissions
user_permissions
tabs
user_tabs
approval_items
calendar_events
announcements
oncall_sessions
oncall_messages
```

我设计这些表时的基本思路：

### 10.1 用户和登录态分离

`users` 存用户基本信息。

`auth_sessions` 存登录态。

这样一个用户可以多设备登录，每次登录生成一个 session token。logout 只吊销当前 token，不影响其他设备。

### 10.2 用户和团队通过 team_members 关联

没有直接在 users 表里只放一个 team_id，而是使用：

```text
team_members(user_id, team_id, team_role)
```

虽然当前阶段暂时按“一个用户一个团队”来做，但这个表结构可以扩展到多团队。

### 10.3 Tab 和用户启用状态分离

`tabs` 存系统 Tab 或自定义 Tab 定义。

`user_tabs` 存用户启用了哪些 Tab。

这样可以区分：

- Tab 本身是什么
- 某个用户有没有启用它

### 10.4 业务数据保留 user_id/team_id

审批、日程、公告都尽量保留：

```text
user_id
team_id
creator_id
applicant_id
publisher_id
```

这样做是为了以后能回答：

- 这个数据是谁创建的
- 属于哪个团队
- 谁可以看
- 谁可以操作

### 10.5 索引思路

我补了一些组合索引，例如：

```text
approval_items(team_id, status, created_at)
oncall_sessions(user_id, updated_at)
oncall_messages(session_id, created_at)
announcements(scope, team_id, created_at)
```

我的理解是，索引不是越多越好，而是要围绕实际查询：

- 审批经常按团队、状态、时间查。
- AI 会话经常按用户和更新时间查。
- 消息经常按 session 和创建时间查。
- 公告经常按范围和团队查。

## 11. AI OnCall 接入思路

AI OnCall 当前不是由客户端直接调 AI 服务，而是：

```text
客户端 -> 我的服务端 -> AI 服务
```

这样做的原因：

- 客户端不用知道 AI 服务细节。
- 服务端可以保存会话和消息。
- 服务端可以适配 AI 服务 SSE 格式。
- AI 服务异常时，服务端可以降级。

当时有一个实际问题：AI 服务返回的是嵌套 SSE：

```text
data:event: tool
data:data: {"type":"tool","tool":"search"}
```

而我一开始写的解析逻辑只支持：

```text
data: {"type":"content","delta":"xxx"}
```

结果客户端报错。后来我通过本地脚本直接请求 AI 服务，确认它不是没响应，而是返回格式和预期不同，并且曾经出现过 tool 事件后提前 EOF。

这让我学到一点：联调时不能只看客户端截图，要把链路拆开：

```text
客户端 -> 服务端
服务端 -> AI 服务
AI 服务原始响应
服务端转发后的响应
```

只有这样才能判断问题到底在哪一方。

## 12. 我如何使用 AI 写代码

导师知道我们会用 AI，所以我觉得不需要避讳。真正重要的是：我不是把 AI 生成的代码直接当成最终答案，而是不断和它对抗、校验、修正。

我的使用方式大概分几类。

### 12.1 用 AI 快速生成初版

比如最开始的接口文档、mock 数据、基础 route/service/repository，AI 可以很快生成一个能跑的版本。

这对我有帮助，因为我可以先看到一个完整结构，而不是卡在空白文件前。

但问题是，AI 生成的初版经常是“看起来完整”，实际细节不一定合理。

### 12.2 追问 AI 的设计假设

比如 Tab 接入协议、静态接入、用户启用 Tab、审批 Tab 数据从哪里来，我一开始其实很晕。

我反复问：

- 客户端到底提前写了什么？
- 服务端到底下发什么？
- 用户增加 Tab 是启用配置，还是上传代码？
- 数据是客户端本地 mock，还是服务端提供？

这个过程不是单纯让 AI 写代码，而是用 AI 帮我把概念边界讲清楚。

### 12.3 对 AI 生成代码做反向检查

AI 生成代码后，我会继续问：

- 这个接口有没有真的实现？
- 数据库里到底保存了什么？
- 不同用户是否真的隔离？
- token 是永久的吗？
- logout 有没有真的吊销？
- 服务器重启后客户端状态会怎样？

这些问题暴露出很多初版代码的问题。例如：

- 密码字段叫 hash 但其实存明文。
- debug 接口一开始状态不准确。
- logout 没走鉴权中间件。
- AI SSE 解析过于理想化。
- repository 中混入了太多业务规则。

### 12.4 用测试和命令验证 AI 的结论

我逐渐发现，不能只听 AI 说“应该可以”。必须用命令验证：

```bash
go test ./...
curl ...
lsof -i :8080
tail -n 80 server.err.log
```

比如 AI 服务出问题时，我没有只看客户端错误，而是写脚本直接请求 AI 服务，确认它返回了 200，但没有完整 content/done。

这类验证让我能区分：

- 是客户端没调？
- 是我的服务端转发错？
- 是 AI 服务没完整返回？
- 是端口冲突？
- 是云服务器进程没启动？

### 12.5 和 AI 对抗的收获

我最大的收获是：AI 很适合生成结构和候选方案，但它不天然知道我项目里的真实边界。

比如它可能倾向于：

- 直接加很多接口。
- 直接改客户端。
- 把权限判断写到任何地方。
- 默认某个服务是正常的。
- 忽略云服务器上已有进程或数据库状态。

所以我需要做判断：

- 哪些需求现在要做，哪些先不做。
- 哪些接口不能改，因为客户端已经依赖。
- 哪些问题要通过日志和 curl 验证。
- 哪些设计不能过度复杂。
- 哪些代码虽然能跑，但以后不好维护。

我觉得这就是我和 AI 协作中真正有价值的部分：不是证明代码是不是 AI 写的，而是证明我能不能理解、质疑、验证和改造 AI 生成的代码。

## 13. 我认为目前服务端体现的设计能力

目前服务端不只是简单 mock 接口，已经体现了一些真实后端设计：

- 分层架构：routes / services / policies / repositories / database。
- 双模式数据源：memory 和 postgres。
- 登录态管理：随机 token、过期、logout 吊销。
- 密码安全：bcrypt 和旧明文兼容迁移。
- 权限模型：权限码 + 全局角色 + 团队角色。
- 数据隔离：user_id、team_id、currentTeamId、memberships。
- AI 服务适配：服务端代理、SSE 转换、异常降级。
- 可观测和溯源：RequestID、统一错误 traceId、audit_logs 操作审计。
- 数据库初始化：AutoMigrate + Seed + Linux 初始化脚本。
- 测试覆盖：路由测试、service 测试、middleware 测试、AI SSE 解析测试。

这些点不是一次完成的，而是在联调和问题排查中逐步补起来的。

## 14. 操作审计和可溯源设计

在后续优化中，我又增加了服务端内部审计日志能力。这个功能不要求客户端改接口，也不改变原来的请求参数和响应主体，主要目的是让服务端具备“谁在什么时候做了什么”的追踪能力。

当前审计日志写入数据库表：

```text
audit_logs
```

主要记录字段包括：

```text
request_id
user_id
account
action
method
path
status_code
result
error_code
client_ip
user_agent
duration_ms
created_at
```

我没有记录请求体，尤其是不记录登录密码这类敏感信息。这个取舍很重要：日志是为了定位问题和追踪操作，不应该制造新的敏感数据泄漏风险。

审计日志通过 Gin middleware 统一实现，而不是每个接口手写一遍。这样登录、注册、退出、Tab 操作、审批、日程、公告、AI OnCall、管理员接口等关键路径都会自动记录。

我对这个功能的理解是：

```text
运行日志回答“服务有没有报错”
审计日志回答“谁做了什么操作”
requestId 回答“这次请求的完整链路是哪一条”
```

例如客户端出现接口失败时，错误响应会带 `traceId`，服务端可以根据这个 traceId/requestId 去查审计日志和运行日志，从而定位：

- 是哪个用户触发的请求。
- 调的是哪个接口。
- 返回了什么状态码。
- 失败的错误码是什么。
- 请求耗时是否异常。

这个功能的价值不在于多一个表，而在于让后端从“能返回数据”变成“出了问题能解释、能追踪、能复盘”。

## 15. 我接下来想继续优化的方向

如果继续推进，我认为优先级是：

### 15.1 数据库迁移正规化

当前主要依赖 GORM AutoMigrate。开发阶段可以，但后续更正式时应该引入 migration：

```text
migrations/001_init.sql
migrations/002_auth_sessions.sql
migrations/003_team_business.sql
```

这样云服务器数据库结构变化更可追踪。

### 15.2 操作审计日志

关键操作应该记录：

- 登录
- logout
- 审批通过/拒绝/撤回
- 管理员改团队成员
- 管理员改用户角色
- 删除日程/公告

这样出问题时可以追踪是谁操作的。

这一项目前已经实现了基础版本：通过 `Audit` middleware 写入 `audit_logs`。后续还可以继续增强，例如增加资源 ID、资源类型、旧值/新值摘要，以及给管理员提供审计日志查询接口。

### 15.3 并发和幂等

比如两个设备同时撤回同一审批，应该用：

```sql
WHERE id = ? AND status = 'pending'
```

或者增加 version 做乐观锁。

### 15.4 systemd + Nginx 部署

现在服务可以用 nohup 跑，但更正规应该用 systemd 管理：

- 开机自启
- 崩溃自动重启
- 日志集中查看

有域名后再用 Nginx 做 HTTPS。

## 16. 答辩讲解顺序和现场展示脚本

答辩时我不应该从“我写了哪些接口”开始讲，否则容易显得平铺直叙。更合适的主线是：

```text
我做的是开放式 Tab 容器的后端控制面。
它负责账号登录态、权限隔离、Tab 配置下发、业务数据管理、AI OnCall 接入，以及服务端可观测和演示环境恢复。
```

### 16.1 开场定位

建议先用 30 秒说明服务端定位：

```text
我的服务端最初是为了支持客户端联调的 mock server，但后面我把它逐步演进成了一个可部署、可持久化、具备鉴权和权限隔离的后端控制面。

它不是只提供静态 mock 数据，而是负责决定用户能看到哪些 Tab、能操作哪些业务数据，并且把 AI OnCall 接入到后端会话体系里。
```

这个阶段展示：

- 客户端登录页面。
- 云服务器接口能访问。
- Navicat 中的数据库表。

### 16.2 展示账号登录态和安全设计

讲解重点：

```text
我没有继续使用固定 token，而是做了 auth_sessions 登录态表。每次登录生成随机 token，token 有过期时间，logout 会在服务端吊销当前 token。

密码也不再明文保存，而是用 bcrypt 加密。对于旧数据，我做了兼容迁移：旧明文密码登录成功后会自动升级为 bcrypt hash。
```

现场可以展示：

- `users` 表里的 `password_hash`。
- `auth_sessions` 表里的 `expires_at`、`revoked_at`。
- 登录后有新 token。
- logout 后旧 token 调 `/me` 返回 `TOKEN_REVOKED`。

可以说：

```text
这里我想体现的是，后端不是只要能登录就行，还要能解释登录态的生命周期。
```

### 16.3 展示用户、团队、权限和数据隔离

讲解重点：

```text
服务端不会相信客户端传来的 userId，而是从 Bearer Token 解析当前用户，再根据用户的团队、角色和权限码决定能看到什么数据。
```

现场可以展示：

- 用 `product-manager` 登录，看产品研发部审批和日程。
- 用 `operation-employee` 登录，看不到产品研发部不该看的数据。
- Navicat 展示 `teams`、`team_members`、`user_permissions`。

可以说：

```text
权限码决定用户能不能使用某类功能，团队角色决定用户能操作哪个范围的数据。
```

### 16.4 展示 Tab 控制面

讲解重点：

```text
开放式 Tab 容器里，客户端不应该完全写死工作台结构。服务端通过 TabManifest、user_tabs 和权限控制，决定当前用户启用了哪些 Tab、Tab 的顺序和入口信息。
```

现场可以展示：

- `/tabs` 返回当前用户启用的 Tab。
- `/tabs/catalog` 返回可选 Tab。
- 客户端工作台里启用/停用 Tab 后，数据库 `user_tabs` 变化。

可以说：

```text
这部分体现的是服务端控制面能力，而不只是业务 CRUD。
```

### 16.5 展示内置业务 Tab 数据

讲解重点：

```text
为了让审批、日程、公告这些内置 Tab 真正可联调，我给它们补了后端业务接口，而不是只让客户端本地 mock。
```

现场可以展示：

- 审批列表、审批详情、通过/拒绝/撤回。
- 日程列表，演示数据时间错开。
- 公告列表。
- Navicat 里对应的 `approval_items`、`calendar_events`、`announcements`。

可以说：

```text
这些数据都和当前登录用户、团队、权限有关，不同账号看到的结果不完全一样。
```

### 16.6 展示 AI OnCall 接入

讲解重点：

```text
客户端不是直接请求 AI 服务，而是请求我的服务端。服务端保存会话和消息，再转发给 AI 服务，并处理 AI SSE 返回格式。
```

现场可以展示：

- 客户端 AI OnCall 对话。
- 数据库 `oncall_sessions`、`oncall_messages`。
- 如果 AI 服务异常，可以看 `server.err.log`。

可以说：

```text
我这里做的不是单纯接一个聊天接口，而是把 AI 对话纳入服务端会话管理和业务系统中。
```

### 16.7 展示审计日志和 requestId

讲解重点：

```text
为了让系统具备可溯源能力，我增加了服务端审计日志。关键操作都会写入 audit_logs，包括用户、接口、操作类型、状态码、错误码、耗时和 requestId。
```

现场可以展示：

```sql
SELECT *
FROM audit_logs
ORDER BY created_at DESC
LIMIT 20;
```

同时可以展示一次失败请求的响应：

```json
{
  "code": "UNAUTHORIZED",
  "message": "缺少 Bearer Token",
  "traceId": "req-..."
}
```

可以说：

```text
运行日志用于看服务错误，审计日志用于看用户操作，requestId 用于把一次请求串起来。这是我后面补的服务端可观测能力。
```

### 16.8 展示云端部署和演示数据恢复

讲解重点：

```text
为了支持多人联调和答辩演示，我把服务端部署到了云服务器，并写了数据库初始化、演示数据重置和部署脚本。
```

现场可以展示：

- 云服务器服务运行。
- `curl /health`。
- `scripts/reset_demo_data.sql`。
- `scripts/deploy_demo_reset.sh`。

可以说：

```text
演示数据被污染后，可以清库、重建、写入一组固定演示数据，保证演示环境可恢复。
```

### 16.9 最后总结

最后用 30 秒收束：

```text
我这个服务端的重点不是接口数量，而是从 mock 服务逐步演进出真实后端应该考虑的问题：身份可信、权限可解释、数据隔离可证明、AI 接入可管理、出错后可追踪、演示环境可恢复。

这个过程中我也大量使用 AI 辅助生成代码和文档，但我主要做的是不断追问边界、验证结果、修正设计，把 AI 生成的初版代码改造成符合项目约束的服务端。
```

## 17. 总结

这个服务端一开始只是为了让客户端能联调的 mock server，但在不断推进中，我逐渐把它往真实后端方向改：

```text
从固定 token 到 auth_sessions
从明文密码到 bcrypt
从散乱权限判断到 policies
从 repository 混业务到 service 管规则
从本地 mock 到 PostgreSQL 持久化
从简单 AI 转发到 SSE 适配和异常降级
从无操作记录到 requestId + audit_logs 可溯源
```

我觉得这个过程最重要的不是“写了多少接口”，而是我逐渐能回答这些问题：

- 这个数据属于谁？
- 谁能看？
- 谁能改？
- 凭什么能改？
- token 失效后会怎样？
- 数据库重建后如何恢复？
- 云服务器上出问题怎么定位？
- AI 生成代码哪里可能不可靠？

这也是我目前对服务端开发最大的理解：接口只是表面，真正的后端设计要关注身份、权限、数据边界、状态流转、错误处理和可维护性。
