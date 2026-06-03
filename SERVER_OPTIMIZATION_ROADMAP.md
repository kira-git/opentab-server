# OpenTab 服务端优化方向与优先级

本文档记录在不改变现有客户端能力、不要求客户端或 AI 服务改接口的前提下，服务端后续可以继续优化的方向。

当前约束：

- 对客户端已有接口保持兼容，不删除、不随意改字段含义。
- AI OnCall 仍调用当前 AI 服务接口，服务端只做转发、适配、降级和记录。
- 优先提升服务端自身质量：安全性、数据隔离、架构清晰度、可维护性、可观测性。

## 总体判断

当前服务端已经具备基础联调能力：登录、注册、token 鉴权、Tab、团队、审批、日程、公告、AI OnCall、PostgreSQL 持久化、mock/memory 模式。

下一步优化不应继续盲目堆接口，而应该把已有能力做得更像一个真实后端系统：

- 用户身份更可信。
- 权限判断更集中。
- 数据隔离更严格。
- 数据库结构更稳定。
- 出错后更容易定位。
- 部署和迁移更可控。

## P0：优先优化，直接影响安全和稳定性

### 1. 密码加密存储

现状：

- 数据库字段叫 `password_hash`，但实际存的是明文密码。

问题：

- 数据库泄漏时账号密码直接泄漏。
- 字段名和实际行为不一致，容易在答辩或代码审查中被问到。

优化方案：

- 使用 bcrypt 存储密码哈希。
- 注册时将明文密码 hash 后入库。
- 登录时使用 bcrypt compare 校验。
- 对现有明文种子数据做兼容：如果发现旧明文密码登录成功，可以自动升级为 hash。

对客户端影响：

- 无影响。
- 客户端仍然提交 `account + password`。

优先级理由：

- 安全收益高。
- 改动范围可控。
- 能体现后端安全意识。

### 2. Token 生命周期和吊销机制完善

现状：

- 已经有 `auth_sessions` 表。
- token 可以过期。
- logout 可以要求鉴权。

可优化点：

- 明确 token 默认有效期，例如 7 天。
- logout 时将当前 token 标记为 revoked，而不是只让客户端删除。
- 鉴权中间件统一检查：
  - token 是否存在
  - 是否过期
  - 是否被吊销
  - 用户是否启用

对客户端影响：

- 基本无影响。
- 客户端只需要继续带 Bearer Token。
- 如果 token 过期，客户端按现有错误提示重新登录。

优先级理由：

- 这是登录态正规化的核心。
- 可以解释“为什么 token 被盗仍然危险，以及如何降低风险”。

### 3. 权限判断集中化

现状：

- 部分权限判断在 service 中。
- 部分数据可见性判断在 repository 中。
- 管理员、团队主管、员工的判断逻辑散落在不同函数里。

问题：

- 后续新增业务时容易漏权限。
- 同一个权限规则可能在多处重复实现。

优化方案：

- 新增 `internal/authorization` 或 `internal/policies` 包。
- 把权限判断统一成策略函数，例如：
  - `CanManageTeam(user, teamID)`
  - `CanReadTeamMembers(user, teamID)`
  - `CanApproveItem(user, item)`
  - `CanReadCalendarEvent(user, event)`
  - `CanManageAnnouncement(user, announcement)`

对客户端影响：

- 无影响。
- 接口和返回格式不变。

优先级理由：

- 直接提升代码可维护性。
- 能体现“权限不是简单 if 判断，而是策略层设计”。

### 4. 数据隔离规则收口

现状：

- 多数业务数据已经通过 `user_id`、`team_id`、`currentTeamId` 做隔离。
- 但不同 repository 里仍有一些各自判断。

问题：

- 一旦某个查询漏了 `user_id/team_id` 条件，就可能跨账号/跨团队读到数据。

优化方案：

- 为 repository 查询建立固定模式：
  - 个人数据必须带 `user_id`
  - 团队数据必须带 `team_id`
  - 管理员读全部数据必须经过 `globalRole=admin` 判断
- 对核心表增加组合索引：
  - `oncall_sessions(user_id, updated_at)`
  - `oncall_messages(session_id, created_at)`
  - `approval_items(team_id, status, created_at)`
  - `calendar_events(team_id, start_time)`
  - `announcements(scope, team_id, created_at)`
  - `auth_sessions(token, expires_at)`

对客户端影响：

- 无影响。

优先级理由：

- 数据隔离是团队账号设计的基础。
- 比继续加新功能更重要。

## P1：第二优先级，提升工程质量和真实项目感

### 5. 数据库迁移脚本正规化

现状：

- 主要依赖 GORM `AutoMigrate`。
- 有数据库初始化脚本和 seed 逻辑。

问题：

- `AutoMigrate` 适合开发，但正式环境中每次结构变化不够可控。
- 后续表结构改动多时，不容易知道数据库经历了哪些版本。

优化方案：

- 引入迁移目录，例如：
  - `migrations/001_init.sql`
  - `migrations/002_auth_sessions.sql`
  - `migrations/003_team_business.sql`
- 部署时执行迁移。
- seed 数据和 schema 迁移分开。

对客户端影响：

- 无影响。

优先级理由：

- 能解决“本地数据库和云服务器数据库如何同步”的问题。
- 很适合写进阶段总结。

### 6. Service / Repository 职责进一步清晰

现状：

- 已经有 routes、services、repositories 分层。
- 但部分业务规则仍然混在 repository 中。

优化方案：

- routes：只负责参数解析、鉴权用户提取、调用 service。
- services：负责业务规则、权限判断、状态流转。
- repositories：只负责数据库读写，不决定业务权限。

示例：

- 审批是否能撤回，应该在 service 根据当前用户和审批状态判断。
- repository 只执行更新并保证数据条件正确。

对客户端影响：

- 无影响。

优先级理由：

- 能体现服务端架构思考。
- 让后续接真实业务更容易。

### 7. 审批、日程、公告状态模型细化

现状：

- 审批状态已有 `pending/approved/rejected/cancelled`。
- 日程和公告字段基本够用。

可优化点：

- 审批增加状态流转约束：
  - pending -> approved
  - pending -> rejected
  - pending -> cancelled
  - approved/rejected/cancelled 不可重复操作
- 日程增加可见性约束：
  - personal
  - team
  - company
- 公告增加发布状态：
  - draft
  - published
  - archived

对客户端影响：

- 可保持兼容。
- 旧客户端不传新字段时，服务端使用默认值。

优先级理由：

- 让业务闭环更真实。
- 但不如鉴权和数据隔离紧急。

### 8. AI OnCall 服务端侧上下文整理

现状：

- 服务端保存会话和消息。
- 发给 AI 服务的是当前问题 + conversationId。
- 不直接把历史消息数组传给 AI。

在不要求 AI 方改接口的前提下，可以优化：

- 服务端内部整理最近 N 条历史。
- 将历史压缩成一段上下文文本拼进当前 message。
- 仍然调用原来的 AI 接口：

```json
{
  "message": "历史上下文...\n\n当前问题...",
  "conversationId": "sessionId"
}
```

对客户端影响：

- 无影响。

对 AI 方影响：

- 不需要改接口。
- 只是 message 内容变长。

优先级理由：

- 可以提升多轮问答体验。
- 但要控制 token 长度，避免过长影响 AI 服务。

## P2：后续优化，适合答辩展示但不急

### 9. 请求日志和操作审计

现状：

- Gin 有基础日志。
- 业务操作没有完整审计表。

优化方案：

- 增加 `audit_logs` 表。
- 记录关键操作：
  - 登录
  - logout
  - 创建/审批/撤回审批
  - 创建/修改/删除日程
  - 管理员修改团队成员
  - 管理员修改用户角色

建议字段：

- `id`
- `user_id`
- `action`
- `resource_type`
- `resource_id`
- `ip`
- `user_agent`
- `created_at`

对客户端影响：

- 无影响。

价值：

- 能体现“出了问题可以追责和排查”。

### 10. 统一错误码和错误语义

现状：

- 已有统一 response 包。
- 错误码已经比早期规范很多。

可优化点：

- 建立错误码文档：
  - `UNAUTHORIZED`
  - `FORBIDDEN`
  - `RESOURCE_NOT_FOUND`
  - `INVALID_REQUEST`
  - `TOKEN_EXPIRED`
  - `TOKEN_REVOKED`
  - `CONFLICT`
  - `INVALID_STATE`
  - `AI_SERVICE_ERROR`
- 统一 401/403/404/409/500 的使用场景。

对客户端影响：

- 不改变现有字段。
- 只让错误更稳定。

### 11. 并发和幂等保护

现状：

- 基础 CRUD 可以跑通。
- 并发场景还没有专门处理。

潜在问题：

- 两台设备同时撤回同一个审批。
- 两个管理员同时修改同一成员角色。
- 重复提交同一个表单。

优化方案：

- 审批状态更新使用条件更新：
  - `WHERE id = ? AND status = 'pending'`
- 需要时增加 `version` 字段做乐观锁。
- 对关键创建接口支持幂等 key。

对客户端影响：

- 可以无影响。
- 幂等 key 可作为后续可选字段。

### 12. 部署稳定性

现状：

- 使用 `nohup` 和脚本启动。

优化方案：

- 使用 systemd 管理服务：
  - 开机自启
  - 自动重启
  - 日志集中查看
- Nginx 反向代理：
  - 对外 80/443
  - Go 服务只监听 127.0.0.1:8080
- 后续有域名后启用 HTTPS。

对客户端影响：

- 如果启用 HTTPS，需要客户端改 baseUrl。
- 其他无影响。

## P3：更深入但当前不急

### 13. Refresh Token

当前可以先用单 token + 过期时间。

后续可以升级为：

- access token：短期，例如 30 分钟。
- refresh token：长期，例如 7 天或 30 天。
- refresh token 可吊销。

对客户端影响：

- 有影响，需要客户端接刷新接口。
- 因此当前不建议马上做。

### 14. 多团队切换

当前设计先保持一个用户一个当前团队。

后续可以做：

- 一个用户加入多个团队。
- `/me` 返回 memberships。
- 新增切换当前团队接口。
- 所有团队数据按 currentTeamId 过滤。

对客户端影响：

- 有影响，需要客户端做团队选择。
- 当前阶段只纳入设计，不急着实现。

### 15. 真实权限模型 RBAC 化

当前权限仍然是用户权限码为主。

后续可以正规化：

- roles
- role_permissions
- user_roles
- team_roles

对客户端影响：

- 无直接影响。
- 但服务端改动较大。

当前阶段可以先保留权限码方案，避免过度设计。

## 推荐实施顺序

### 第一批：安全基础

1. 密码 bcrypt 加密。
2. token 过期、吊销、用户启用状态检查。
3. 统一鉴权中间件错误码。

目标：

- 让账号系统不再像 mock。
- 不影响客户端接口。

### 第二批：权限和隔离

1. 抽出权限策略层。
2. 收口用户/团队数据隔离规则。
3. 为核心表补组合索引。

目标：

- 能明确说明不同用户为什么只能看到自己的数据。
- 降低后续加功能时漏权限的风险。

### 第三批：数据库工程化

1. 增加 migrations。
2. schema 初始化和 seed 数据分开。
3. 云服务器部署脚本支持迁移执行。

目标：

- 本地和云服务器数据库结构同步更稳定。
- 避免每次靠手工重建。

### 第四批：业务状态和审计

1. 审批状态流转约束。
2. 操作审计日志。
3. 并发更新保护。

目标：

- 让审批、团队管理这些业务更像真实系统。

### 第五批：部署和展示

1. systemd 管理 Go 服务。
2. Nginx 反向代理。
3. 有域名后启用 HTTPS。

目标：

- 云服务器运行更稳定。
- 演示时访问方式更正规。

## 可以向导师表达的个人思考

可以这样总结：

> 我没有继续简单堆接口，而是把服务端拆成几个更真实的后端问题：身份可信、权限可解释、数据隔离可证明、状态流转可控、数据库迁移可复现、异常可观测。当前接口已经能支撑客户端联调，后续优化重点是不破坏客户端调用方式的前提下，把 mock 后端逐步升级为有真实工程约束的业务后端。

核心差异点：

- 不是只做 CRUD。
- 关注多用户和团队权限下的数据边界。
- 关注登录态、token 过期和吊销。
- 关注数据库结构演进和云端部署可复现。
- 关注 AI 服务不稳定时的服务端降级能力。

