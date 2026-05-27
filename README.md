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
# 启动 H5、API、PostgreSQL、Redis
docker compose -f deploy/docker-compose.yml up --build

# 启动 uni-app H5
cd app
npm install
npm run dev:h5
```

完整技术文档见：`docs/engineering-technical-design.md`。

## 当前能力

- `app/` 提供 uni-app + Vue 3 五个 Tab 页面骨架，日程页和沟通页已接入核心 API。
- `server/` 提供 Go HTTP API，包含 Dashboard、课程、提醒、家长档案、沟通记录、疗愈记录和 AI mock 服务。
- PostgreSQL 负责持久化业务数据，Redis 负责首页工作台缓存。
- API 启动时会按 `MIGRATIONS_DIR` 自动执行 SQL 迁移，Docker 环境默认使用 `/app/migrations`。
- `APP_ENV=local` 迁移失败时会降级到内存仓库；Docker/生产环境迁移失败会直接退出。
- `deploy/docker-compose.yml` 可编排 H5 Web、API、PostgreSQL、Redis；H5 默认访问 `http://localhost:8081`。
- `.github/workflows/engineering-checks.yml` 包含 Go 测试和 H5 构建检查。

## 本地环境说明

当前机器未安装 `go`、`npm`、`docker` 时，无法直接运行 `go test ./...`、`npm run build:h5` 或 Docker Compose 联调。CI 或具备这些工具的开发机应作为运行级验证入口。
