# 部署运行手册

## 1. 目标环境

- Docker Compose 运行 H5 Web、Go API、PostgreSQL、Redis。
- H5 入口：`http://localhost:8081`
- API 入口：`http://localhost:8080`
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`

## 2. 一键启动

```bash
docker compose -f deploy/docker-compose.yml up --build
```

首次启动会构建：

- `deploy/web.Dockerfile`：构建 uni-app H5，并用 nginx 托管 `dist` 静态资源。
- `deploy/server.Dockerfile`：构建 Go API，并复制 `server/migrations` 到 `/app/migrations`。

## 3. 健康检查

```bash
curl http://localhost:8080/healthz
curl http://localhost:8081/healthz
```

预期返回：

```json
{ "status": "ok", "time": "2026-05-27T14:00:00+08:00" }
```

## 4. 数据迁移

API 启动时读取 `MIGRATIONS_DIR` 并按文件名顺序执行 SQL 文件。

Docker 默认值：

```text
MIGRATIONS_DIR=/app/migrations
```

本地开发默认值：

```text
MIGRATIONS_DIR=server/migrations
```

当前迁移脚本保持幂等，服务重复启动不会重复插入种子数据。Docker / 生产环境迁移失败会让 API 退出；`APP_ENV=local` 下迁移失败会降级到内存仓库。

## 5. 常用运维命令

```bash
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml logs -f api
docker compose -f deploy/docker-compose.yml logs -f web
docker compose -f deploy/docker-compose.yml restart api
docker compose -f deploy/docker-compose.yml down
```

清空本地数据库和缓存卷：

```bash
docker compose -f deploy/docker-compose.yml down -v
```

## 6. CI 验证

GitHub Actions 会执行：

- OpenAPI YAML 解析检查。
- Go 依赖下载和 `go test ./...`。
- uni-app H5 依赖安装和 `npm run build:h5`。

本机若没有 `go`、`npm`、`docker`，以 CI 或具备完整工具链的开发机作为运行级验证入口。
