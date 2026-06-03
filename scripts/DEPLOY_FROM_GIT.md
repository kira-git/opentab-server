# 从 Git 更新云服务器服务端

## 本地提交

```bash
go test ./...
git add .
git commit -m "update opentab server"
git push
```

## 云服务器首次拉取

```bash
cd /home
git clone <你的 GitHub 仓库地址> server
cd server
chmod +x scripts/*.sh
./scripts/init_postgres_linux.sh
```

如果要清空旧数据库并重新初始化：

```bash
OPENTAB_DB_RESET=true ./scripts/init_postgres_linux.sh
```

然后再启动服务端。服务端启动时会自动建表并写入默认数据。

## 后续更新

```bash
cd /home/server
./scripts/deploy_restart.sh
```

脚本会执行：

```text
git pull
go test ./...
停止旧 server
nohup 启动新 server
curl /health 验证
```

## 演示前一键重置部署

如果需要清掉旧服务、重建数据库、清空测试脏数据、生成脱敏演示数据并启动新服务，使用：

```bash
cd /home/server
chmod +x scripts/*.sh
APP_MODE=postgres \
DATABASE_URL="postgres://opentab:opentab123@localhost:5432/opentab?sslmode=disable" \
HOST=0.0.0.0 \
PORT=8080 \
AI_SERVICE_BASE_URL="http://127.0.0.1:8081" \
./scripts/deploy_demo_reset.sh
```

这个脚本会执行：

```text
git fetch + reset 到 origin/main
停止旧 OpenTab server
重建 opentab 数据库
go test ./...
nohup 启动新服务
执行 reset_demo_data.sql 写入脱敏演示数据
curl /health 验证
```

注意：这个脚本会丢弃云服务器 `server` 目录里的本地改动，适合演示部署，不适合保留云端临时修改。

## 数据库同步

当前阶段数据库结构由服务启动时的 GORM 自动处理：

```text
AutoMigrate：新增表、新增字段
Seed：补齐默认数据
```

所以云服务器拉取新代码并重启服务后，会自动根据 Go model 更新表结构。

注意：

```text
AutoMigrate 适合开发阶段的新增表/字段。
如果后续要删除字段、改字段名、改字段类型，需要引入正式 migration。
```
