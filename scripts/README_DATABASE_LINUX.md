# OpenTab Linux PostgreSQL 初始化

这个目录里的脚本用于在 Linux 云服务器上创建和本地一致的 PostgreSQL 数据库环境。

## 1. 脚本做什么

`init_postgres_linux.sh` 只做三件事：

1. 创建 PostgreSQL 用户。
2. 创建 `opentab` 数据库。
3. 给用户授权。

业务表不在脚本里手写 SQL 创建。服务端启动时会自动执行：

```text
database.AutoMigrate(db)
database.Seed(db)
```

也就是说，表结构来自 Go 代码里的 GORM model，种子数据来自 `internal/database/seed.go`。

## 2. 使用方式

把整个 server 目录放到 Linux 服务器后执行：

```bash
cd server
chmod +x scripts/init_postgres_linux.sh
./scripts/init_postgres_linux.sh
```

默认会创建：

```text
数据库：opentab
用户：opentab
密码：opentab123
```

如果要自定义：

```bash
OPENTAB_DB_NAME=opentab \
OPENTAB_DB_USER=opentab \
OPENTAB_DB_PASSWORD=你的密码 \
./scripts/init_postgres_linux.sh
```

## 3. 启动服务端

脚本执行成功后，用 PostgreSQL 模式启动服务端：

```bash
APP_MODE=postgres \
DATABASE_URL="postgres://opentab:opentab123@localhost:5432/opentab?sslmode=disable" \
HOST=0.0.0.0 \
PORT=8080 \
go run ./cmd/server
```

第一次启动时，服务端会自动创建这些表：

```text
users
auth_sessions
permissions
user_permissions
tabs
user_tabs
oncall_sessions
oncall_messages
approval_items
calendar_events
```

并写入默认账号、默认 Tab、审批、日程等种子数据。

## 4. 为什么不写完整建表 SQL

当前项目还在开发阶段，表结构会继续变化。直接维护一份手写 SQL 容易和 Go 代码不一致。

现在采用的方式是：

```text
脚本：创建数据库和账号
服务端：自动建表和初始化数据
```

这样本地和云服务器只要运行同一份服务端代码，数据库结构就是一致的。
