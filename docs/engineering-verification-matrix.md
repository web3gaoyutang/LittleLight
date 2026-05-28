# 工程验证矩阵

本文档把当前工程目标拆成可检查条目，并给出对应文件、验证命令和剩余风险。它用于判断工程实现是否仍朝“uni-app + Go + PostgreSQL + Redis + Docker + 完整技术文档”的目标推进。

## 验证总入口

| 场景 | 命令 | 覆盖范围 |
| --- | --- | --- |
| 本地快速回归 | `powershell -ExecutionPolicy Bypass -File .\scripts\verify-all.ps1` | OpenAPI、环境示例、Compose、Web readiness、文档覆盖、Go 测试、前端 API client 测试、H5 构建、密钥检查 |
| 本地业务逻辑回归 | `powershell -ExecutionPolicy Bypass -File .\scripts\verify-all.ps1 -IncludeDockerLogic` | 快速回归 + Docker PostgreSQL/Redis + 本机 API 主业务写读链路 |
| CI 回归 | `.github/workflows/engineering-checks.yml` | OpenAPI、文档覆盖、Compose、Go 测试、前端 API client 测试、H5 构建、Web/API Docker 镜像构建 |

## 目标到证据映射

| 目标要求 | 当前证据 | 验证方式 | 状态 |
| --- | --- | --- | --- |
| App 使用 uni-app | `app/src`、`app/src/pages.json`、`app/vite.config.js`、`app/package.json` | `npm run build:h5`、`npm run test:api` | 已实现工程骨架和核心 API 接入 |
| 后端使用 Golang | `server/go.mod`、`server/cmd/api/main.go`、`server/internal/http/router.go` | `cd server && go test ./...` | 已实现 HTTP API、服务层和 repository |
| 数据库使用 PostgreSQL | `server/internal/repository/postgres.go`、`server/migrations`、`docs/database-schema.md` | `scripts/verify-docker.ps1` 写入并查询 PostgreSQL；`docs/database-schema.md` 记录 schema | 已接入持久化仓库和迁移 |
| 缓存使用 Redis | `server/internal/platform/cache`、`server/internal/repository/*Dashboard*`、`scripts/verify-docker.ps1` | Docker 逻辑验证会确认 dashboard 缓存 key 存在 | 已接入 dashboard 缓存 |
| 部署使用 Docker | `deploy/docker-compose.yml`、`deploy/server.Dockerfile`、`deploy/web.Dockerfile`、`deploy/nginx/default.conf` | `docker compose config --quiet`、CI `docker-build` job | 已有 Compose 编排和镜像构建门禁 |
| 编写完整详细技术文档 | `docs/engineering-technical-design.md`、`docs/openapi.yaml`、`docs/database-schema.md`、`docs/deployment-runbook.md`、本文档 | `python scripts/verify-docs.py` | 已覆盖架构、API、数据库、部署、验证矩阵 |

## 关键验证项

| 验证项 | 证明什么 | 命令或文件 |
| --- | --- | --- |
| OpenAPI 解析 | API 契约可被机器读取 | `.github/workflows/engineering-checks.yml` docs job |
| 路由/OpenAPI 一致性 | 后端路由和契约没有 path/method 漂移 | `server/internal/http/router_test.go` |
| 配置加载测试 | LLM 自动模式、旧 key 兼容、Redis DB 容错正常 | `server/internal/config/config_test.go` |
| 前端 API client 测试 | base URL、鉴权头、微信模拟登录态、上传解析正常 | `app/scripts/test-api-client.mjs` |
| Docker 逻辑验证 | PostgreSQL/Redis/API 主业务链路能本地闭环 | `scripts/verify-docker.ps1` |
| 文档覆盖检查 | 关键文档和环境示例未被漏删 | `scripts/verify-docs.py` |
| 敏感信息检查 | 本地 LLM key/base URL 未进入 Git 跟踪文件 | `scripts/verify-all.ps1` |

## 当前剩余风险

- 本机 Docker Hub 访问可能失败；当前通过镜像变量、CI `docker-build` job 和本地 Docker 逻辑验证降低风险。
- 真实微信 code 换 session 尚未接入；当前是微信模拟登录和 `X-User-ID` 开发鉴权。
- 推送通知、支付、对象存储、真实生产多副本部署尚未进入本阶段实现。

## 完成判定建议

在声明工程阶段完成前，至少需要以下证据同时成立：

1. `scripts/verify-all.ps1 -IncludeDockerLogic` 通过。
2. GitHub Actions 全部 job 通过，尤其是 `docker-build`。
3. `docs/engineering-technical-design.md`、`docs/openapi.yaml`、`docs/database-schema.md`、`docs/deployment-runbook.md` 和本文档均为最新。
4. `rg "真实密钥|LLM_API_KEY实际值|sk-"` 不应在 Git 跟踪文件中命中真实密钥。
