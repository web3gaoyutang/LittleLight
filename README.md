# 微光老师工程实现

本仓库已从 H5 原型进入工程化阶段。

## 技术栈

- App：uni-app + Vue 3
- 后端：Golang
- 数据库：PostgreSQL
- 缓存：Redis
- 部署：Docker Compose

## 快速开始

```bash
# 启动 H5、API、PostgreSQL、Redis。若 Docker Hub 拉取基础镜像受限，可先用下面的本地逻辑验证。
docker compose -f deploy/docker-compose.yml up --build

# Docker Hub 访问较慢时，可在 .env 中覆盖 GO_IMAGE、NODE_IMAGE、NGINX_IMAGE、
# ALPINE_IMAGE、POSTGRES_IMAGE、REDIS_IMAGE、GOPROXY、NPM_REGISTRY。

# 使用 Docker 中的 PostgreSQL/Redis + 本机 Go API 做逻辑验证
powershell -ExecutionPolicy Bypass -File .\scripts\verify-docker.ps1

# 一键运行本地工程回归；需要连 Docker 业务逻辑时追加 -IncludeDockerLogic
powershell -ExecutionPolicy Bypass -File .\scripts\verify-all.ps1

# 启动 uni-app H5
cd app
npm ci
npm run dev:h5
```

完整技术文档见：`docs/engineering-technical-design.md`。机器可读 API 契约见：`docs/openapi.yaml`。部署运行手册见：`docs/deployment-runbook.md`。

## 当前能力

- `app/` 提供 uni-app + Vue 3 五个 Tab 页面骨架，日程页和沟通页已接入核心 API。
- `server/` 提供 Go HTTP API，包含 Dashboard、课程、提醒、家长档案、沟通记录、疗愈记录、AI 服务和微信模拟登录。
- PostgreSQL 负责持久化业务数据，Redis 负责首页工作台缓存。
- API 启动时会按 `MIGRATIONS_DIR` 自动执行 SQL 迁移，Docker 环境默认使用 `/app/migrations`。
- `APP_ENV=local` 迁移失败时会降级到内存仓库；Docker/生产环境迁移失败会直接退出。
- `deploy/docker-compose.yml` 可编排 H5 Web、API、PostgreSQL、Redis；H5 默认访问 `http://localhost:8081`。
- Docker 基础镜像、Go 代理和 npm registry 支持通过 `.env` 覆盖，便于弱网或内网镜像环境构建。
- API 提供 `/healthz` 进程健康检查和 `/readyz` 依赖就绪检查；Docker API 容器使用 `/readyz` 作为 healthcheck。
- `.github/workflows/engineering-checks.yml` 包含 OpenAPI、Docker Compose 配置、Go 测试和 H5 构建检查。

## 本地环境说明

当前本机已支持 Go、Node 和 Docker 验证。基础回归命令：

```bash
cd server && go test ./...
cd ../app && npm run build:h5
cd .. && docker compose -f deploy/docker-compose.yml config --quiet
powershell -ExecutionPolicy Bypass -File .\scripts\verify-all.ps1 -IncludeDockerLogic
```
