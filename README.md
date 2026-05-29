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

后端启动时会自动尝试读取仓库根目录或当前工作目录附近的 `.env`；已经在进程环境中显式设置的变量优先级更高，不会被 `.env` 覆盖。

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

`api-smoke.mjs` 会检查 health/readiness、微信开发登录、首页、课程、待办、家长、沟通记录、AI 草稿、AI 夸夸、疗愈记录、收藏和 AI 审计记录。可通过 `LITTLELIGHT_API_URL` 指向非默认 API 地址。

验证运行中的 H5 前端关键路径：

```bash
cd app
npm run test:h5-smoke
```

`test:h5-smoke` 会启动临时 headless Chrome，点击 H5 开发登录，进入首页和日程页，提交一条真实待办，再通过 API 确认服务端落库。需要本机已有 Chrome、API 在 `http://localhost:8080`、H5 在 `http://localhost:5173`；可通过 `CHROME_PATH`、`LITTLELIGHT_API_URL`、`LITTLELIGHT_H5_URL` 覆盖。

## 当前能力

- `app/` 提供 uni-app + Vue 3 五个 Tab 页面骨架，核心页面已接入后端 API。
- `server/` 提供 Go HTTP API，覆盖 Dashboard、课程、待办提醒、家长档案、沟通记录、疗愈记录、收藏、AI 生成、微信 code 登录和开发模拟登录。
- PostgreSQL 负责持久化业务数据，Redis 负责首页 dashboard 缓存。
- API 启动时会按 `MIGRATIONS_DIR` 自动执行 SQL 迁移；Docker 环境默认使用 `/app/migrations`。
- `APP_ENV=local` 时 PostgreSQL 不可用会降级到内存仓库；`APP_ENV=docker/prod/production` 中数据库或迁移失败会直接退出。
- Docker Compose 可编排 H5 Web、API、PostgreSQL 和 Redis；默认 `APP_ENV=docker` 是本地容器开发环境，关闭 `X-User-ID` 开发鉴权但保留模拟微信登录；API 容器使用 `/readyz` 作为 healthcheck。
- GitHub Actions 包含 `docs`、`server`、`app`、`integration` 和 `docker-build` jobs；`integration` 会启动 PostgreSQL/Redis、Go API 与 H5 dev server 后执行 API smoke 和 H5 smoke。
- LLM 支持 OpenAI-compatible `/v1/chat/completions`，未配置或调用失败时回退到本地 mock，保证主流程可验证。

## 文档入口

- 技术设计：[docs/engineering-technical-design.md](docs/engineering-technical-design.md)
- OpenAPI 契约：[docs/openapi.yaml](docs/openapi.yaml)
- 数据库说明：[docs/database-schema.md](docs/database-schema.md)
- 部署运行手册：[docs/deployment-runbook.md](docs/deployment-runbook.md)
- 工程验证矩阵：[docs/engineering-verification-matrix.md](docs/engineering-verification-matrix.md)
- API 手工检查示例：[docs/api-checks.md](docs/api-checks.md)

## 当前边界

- 后端已提供 `/api/v1/auth/wechat` 做微信 code 换 openid，并按 openid 查找或创建用户；小程序端会优先走真实微信登录，H5/本地调试可通过显式“开发登录”按钮调用 `/api/v1/auth/wechat/mock`。业务接口默认要求 Bearer session，session 会写入服务端 `auth_sessions` 并可通过 `/api/v1/auth/logout` 撤销；仅显式开启 `AUTH_ALLOW_DEV_USER=true` 时可用非空且已存在的 `X-User-ID` 做本地开发鉴权。`APP_ENV=prod/production` 启动时会强制要求 `WECHAT_APP_ID`、`WECHAT_APP_SECRET`、至少 32 字符的 `SESSION_SECRET` 与具体的 `CORS_ALLOWED_ORIGINS`，并拒绝 `AUTH_ALLOW_DEV_USER=true`、`AUTH_ALLOW_MOCK_LOGIN=true` 或通配 CORS；生产前端不要设置 `VITE_ENABLE_MOCK_LOGIN=true`。
- 推送通知、支付、对象存储和真实生产多副本部署尚未进入当前工程阶段。
- 完整 Docker 镜像构建依赖基础镜像可拉取；如果本机 Docker Hub 不可用，可先跑 `verify-all.ps1 -IncludeDockerLogic` 验证 PostgreSQL/Redis/API 主业务逻辑，镜像构建由 CI 兜底。
