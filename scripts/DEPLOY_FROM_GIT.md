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
