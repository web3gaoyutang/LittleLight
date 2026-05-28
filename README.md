# 微光教师工程实现

本仓库已从 H5 原型进入工程化阶段，当前目标是形成可持续开发、可本地验证、可 Docker 部署的 V1 工程底座。

## 技术栈

- App：uni-app + Vue 3
- 后端：Golang HTTP API
- 数据库：PostgreSQL
- 缓存：Redis
- 部署：Docker Compose

## 快速开始

启动完整 Docker 环境：

```bash
docker compose -f deploy/docker-compose.yml up --build
```

默认入口：

- H5 Web：http://localhost:8081
- API：http://localhost:8080
- API 存活检查：http://localhost:8080/healthz
- API 依赖检查：http://localhost:8080/readyz

如果 Docker Hub 拉取基础镜像较慢或不可用，可在本地 `.env` 覆盖 `GO_IMAGE`、`NODE_IMAGE`、`NGINX_IMAGE`、`ALPINE_IMAGE`、`POSTGRES_IMAGE`、`REDIS_IMAGE`、`GOPROXY` 和 `NPM_REGISTRY`。`.env` 已被 Git 忽略，可放本机镜像源和真实 LLM 配置。

## 本地开发

启动后端：

```bash
cd server
go run ./cmd/api
```

启动 uni-app H5：

```bash
cd app
npm ci
npm run dev:h5
```

前端默认请求同源 `/api/v1`；本地 H5 开发时可通过 `app/.env` 覆盖 `VITE_DEV_API_TARGET`，默认代理到 `http://localhost:8080`。

## 验证命令

快速工程回归：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify-all.ps1
```

带 PostgreSQL/Redis 的本地业务闭环：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify-all.ps1 -IncludeDockerLogic
```

只验证 Docker PostgreSQL/Redis + 本机 Go API 主链路：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify-docker.ps1
```

验证运行中的 API：

```bash
node scripts/api-smoke.mjs
```

`api-smoke.mjs` 会检查 health/readiness、微信模拟登录、首页、课程、待办、家长、沟通记录、AI 草稿、AI 夸夸、疗愈记录、收藏和 AI 审计记录。可通过 `LITTLELIGHT_API_URL` 指向非默认 API 地址。

## 当前能力

- `app/` 提供 uni-app + Vue 3 五个 Tab 页面骨架，核心页面已接入后端 API。
- `server/` 提供 Go HTTP API，覆盖 Dashboard、课程、待办提醒、家长档案、沟通记录、疗愈记录、收藏、AI 生成和微信模拟登录。
- PostgreSQL 负责持久化业务数据，Redis 负责首页 dashboard 缓存。
- API 启动时会按 `MIGRATIONS_DIR` 自动执行 SQL 迁移；Docker 环境默认使用 `/app/migrations`。
- `APP_ENV=local` 时 PostgreSQL 不可用会降级到内存仓库；Docker/生产环境中数据库或迁移失败会直接退出。
- Docker Compose 可编排 H5 Web、API、PostgreSQL 和 Redis；API 容器使用 `/readyz` 作为 healthcheck。
- GitHub Actions 包含 `docs`、`server`、`app`、`integration` 和 `docker-build` jobs；`integration` 会启动 PostgreSQL/Redis 与 Go API 后执行 API smoke。
- LLM 支持 OpenAI-compatible `/v1/chat/completions`，未配置或调用失败时回退到本地 mock，保证主流程可验证。

## 文档入口

- 技术设计：[docs/engineering-technical-design.md](docs/engineering-technical-design.md)
- OpenAPI 契约：[docs/openapi.yaml](docs/openapi.yaml)
- 数据库说明：[docs/database-schema.md](docs/database-schema.md)
- 部署运行手册：[docs/deployment-runbook.md](docs/deployment-runbook.md)
- 工程验证矩阵：[docs/engineering-verification-matrix.md](docs/engineering-verification-matrix.md)
- API 手工检查示例：[docs/api-checks.md](docs/api-checks.md)

## 当前边界

- 真实微信 code 换 session 尚未接入；当前使用微信模拟登录和 `X-User-ID` 开发鉴权。
- 推送通知、支付、对象存储和真实生产多副本部署尚未进入当前工程阶段。
- 完整 Docker 镜像构建依赖基础镜像可拉取；如果本机 Docker Hub 不可用，可先跑 `verify-all.ps1 -IncludeDockerLogic` 验证 PostgreSQL/Redis/API 主业务逻辑，镜像构建由 CI 兜底。
