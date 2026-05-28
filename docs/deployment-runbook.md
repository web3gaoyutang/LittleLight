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
curl http://localhost:8080/readyz
curl http://localhost:8081/healthz
```

预期返回：

```json
{ "status": "ok", "time": "2026-05-27T14:00:00+08:00" }
```

`/healthz` 只表示 API 进程可响应；`/readyz` 会检查 PostgreSQL 和 Redis。Docker Compose 中 API 容器使用 `/readyz` 作为 healthcheck。

## 4. 本地 Docker 逻辑验证

PostgreSQL 和 Redis 均由 Docker Compose 提供，本机不需要单独安装数据库或缓存服务。执行：

```powershell
.\scripts\verify-docker.ps1
```

脚本会执行：

- `docker compose -f deploy/docker-compose.yml up -d postgres redis`
- 启动本机 Go API，并连接 Docker 中的 PostgreSQL 与 Redis。
- 等待 `http://localhost:8080/healthz` 与 `http://localhost:8080/readyz` 返回正常。
- 通过 API 写入并读取课程、待办事项、家长档案、沟通记录、AI 草稿、疗愈记录和收藏素材。
- 进入 PostgreSQL 容器确认业务数据已落库。
- 进入 Redis 容器确认首页 dashboard 缓存键已生成。

如需使用真实 LLM，把密钥放到仓库根目录 `.env`，该文件已被 Git 忽略：

```text
LLM_API_KEY=sk-xxx
LLM_BASE_URL=https://llmapi.example.com
LLM_MODEL=gpt-4o-mini
```

未配置 LLM 或 LLM 调用失败时，API 会回落到本地 mock 生成逻辑，保证课程、提醒、沟通、疗愈等主流程仍可验证。

## 5. 数据迁移

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

## 6. 常用运维命令

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

## 7. CI 验证

GitHub Actions 会执行：

- OpenAPI YAML 解析检查。
- Go 依赖下载和 `go test ./...`。
- uni-app H5 依赖锁安装和 `npm run build:h5`。

本机验证建议使用 Docker Compose 拉起 PostgreSQL、Redis、API 和 H5 Web，再执行健康检查与核心业务接口请求；CI 作为提交级回归保障。
