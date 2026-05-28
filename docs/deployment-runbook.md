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

### 2.1 镜像源与弱网构建

默认配置直接使用 Docker Hub 官方基础镜像。如果本机访问 Docker Hub 较慢或公司环境要求走内网镜像，可在仓库根目录 `.env` 中覆盖以下变量：

```text
GO_IMAGE=golang:1.22-alpine
ALPINE_IMAGE=alpine:3.20
NODE_IMAGE=node:20-alpine
NGINX_IMAGE=nginx:1.27-alpine
POSTGRES_IMAGE=postgres:16-alpine
REDIS_IMAGE=redis:7-alpine
GOPROXY=https://goproxy.cn,direct
NPM_REGISTRY=https://registry.npmjs.org/
```

示例：如果已有内网 registry 镜像，可只替换镜像地址，不需要改 Dockerfile：

```text
GO_IMAGE=registry.example.com/library/golang:1.22-alpine
NODE_IMAGE=registry.example.com/library/node:20-alpine
NGINX_IMAGE=registry.example.com/library/nginx:1.27-alpine
ALPINE_IMAGE=registry.example.com/library/alpine:3.20
POSTGRES_IMAGE=registry.example.com/library/postgres:16-alpine
REDIS_IMAGE=registry.example.com/library/redis:7-alpine
NPM_REGISTRY=https://registry.npmmirror.com/
```

`.env` 已被 Git 忽略，可放本机真实镜像源、LLM 密钥和部署私有配置；`.env.example` 只保留安全默认值。

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

`/healthz` 只表示 API 进程可响应；`/readyz` 会检查 PostgreSQL 和 Redis。H5 Web 容器会通过 nginx 代理 `/healthz` 和 `/readyz` 到 API，Compose 中 API 容器使用 `/readyz` 作为 healthcheck，Web 容器等待 API healthy 后再启动并使用 `/healthz` 做自身 healthcheck。

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

## 5. 本地工程回归

提交前建议运行统一验证脚本：

```powershell
.\scripts\verify-all.ps1
```

脚本默认执行：

- OpenAPI 契约关键结构检查。
- H5 API 默认值和后端 `.env.example` 关键变量检查。
- Docker Compose 配置解析。
- Web 网关 readiness 配置检查。
- Go 后端测试。
- uni-app H5 生产构建。
- 本地 `.env` 中的 LLM 配置未被写入 Git 跟踪文件检查。

需要同时验证 PostgreSQL/Redis 业务逻辑闭环时执行：

```powershell
.\scripts\verify-all.ps1 -IncludeDockerLogic
```

## 6. 数据迁移

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

## 7. 发布流程

建议按以下顺序发布：

1. 确认目标分支 CI 通过，尤其是 `docker-build` job。
2. 在发布机器同步最新代码，或拉取已构建好的镜像。
3. 确认 `.env` 中数据库、Redis、LLM、镜像源等配置完整，真实密钥只放部署环境。
4. 先执行配置检查：

```bash
docker compose -f deploy/docker-compose.yml config --quiet
```

5. 启动或滚动更新服务：

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

6. 等待健康检查：

```bash
docker compose -f deploy/docker-compose.yml ps
curl http://localhost:8080/readyz
curl http://localhost:8081/readyz
```

7. 通过 H5 入口执行一次 smoke test：微信模拟登录、首页加载、创建待办、创建家长、生成一条 AI 草稿。

## 8. 数据备份与恢复

PostgreSQL 是业务事实来源；Redis 只做缓存和临时状态，不作为唯一数据来源。

创建 PostgreSQL 备份：

```bash
docker compose -f deploy/docker-compose.yml exec -T postgres pg_dump -U littlelight -d littlelight --clean --if-exists > littlelight-backup.sql
```

恢复 PostgreSQL 备份前，应先停止 API 和 Web，避免恢复过程中产生新写入：

```bash
docker compose -f deploy/docker-compose.yml stop web api
docker compose -f deploy/docker-compose.yml exec -T postgres psql -U littlelight -d littlelight < littlelight-backup.sql
docker compose -f deploy/docker-compose.yml up -d api web
```

Redis 可通过 AOF 卷恢复缓存状态；如果只是缓存异常，优先清空 Redis 并让 API 重新生成 dashboard 缓存：

```bash
docker compose -f deploy/docker-compose.yml exec -T redis redis-cli FLUSHDB
```

## 9. 回滚流程

代码或镜像回滚：

1. 保留当前故障版本的日志。
2. 切回上一个稳定 Git commit 或镜像 tag。
3. 重新执行 `docker compose -f deploy/docker-compose.yml up -d --build`。
4. 检查 `http://localhost:8080/readyz` 和 `http://localhost:8081/readyz`。
5. 如回滚前已执行破坏性迁移，先按备份文件恢复 PostgreSQL，再启动 API/Web。

当前迁移脚本保持向前幂等，但还没有独立 down migration。涉及删字段、改字段类型、批量重写数据的变更必须先创建备份，并在发布说明中写明恢复步骤。

## 10. 常用运维命令

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

## 11. CI 验证

GitHub Actions 会执行：

- OpenAPI YAML 解析检查。
- 文档覆盖检查，避免部署手册、数据库说明和关键环境示例漂移。
- Docker Compose 配置解析检查。
- Go 依赖下载和 `go test ./...`。
- 前端 API client 请求封装测试。
- uni-app H5 依赖锁安装和 `npm run build:h5`。
- Go API 与 H5 Web Docker 镜像构建检查，不推送镜像。

本机验证建议使用 Docker Compose 拉起 PostgreSQL、Redis、API 和 H5 Web，再执行健康检查与核心业务接口请求；CI 作为提交级回归保障。
如果本机无法访问 Docker Hub，可先通过 `.env` 覆盖镜像源，或运行 `.\scripts\verify-all.ps1 -IncludeDockerLogic` 验证 PostgreSQL、Redis 与 API 主业务逻辑；镜像构建由 GitHub Actions 的 `docker-build` job 继续兜底。
