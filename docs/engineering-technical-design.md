# 微光老师工程技术文档

版本：V0.1 工程启动版  
日期：2026-05-27  
技术栈：uni-app + Vue 3、Golang、PostgreSQL、Redis、Docker

## 1. 工程目标

本阶段目标是把现有 H5 高保真原型推进到可持续开发的工程骨架，明确前后端边界、核心数据模型、接口协议、部署方式和后续迭代路径。

当前工程不追求一次性完成完整商业产品，而是优先打通 V1 MVP 的主链路：

- 今日工作台：课程、待办、家长跟进、节奏状态。
- 疗愈空间：一分钟呼吸、AI 夸夸、树洞/疗愈记录。
- 家校沟通：家长档案、AI 回复草稿、沟通记录。
- 日程提醒：周课表、待办提醒、完成提醒。
- 我的：教师资料、Pro 权益、素材库、云同步状态。

## 2. 目录结构

```text
LittleLight/
  app/                         # uni-app 客户端工程
    src/
      pages/                   # 5 个 Tab 页面
      api/                     # API 请求封装
      utils/                   # UI 与通用工具
      static/                  # 公共样式与静态资源
      manifest.json            # uni-app 应用配置
      pages.json               # 页面与 TabBar 配置
    package.json               # 前端依赖与脚本
    package-lock.json          # 前端依赖锁
  server/                      # Go 后端工程
    cmd/api/                   # API 服务入口
    internal/config/           # 环境变量配置
    internal/domain/           # 领域模型
    internal/http/             # HTTP 路由和 Handler
    internal/repository/       # 数据访问层接口与实现
    internal/service/          # 业务服务和 AI 编排
    internal/platform/         # PostgreSQL / Redis 连接
    migrations/                # 数据库初始化脚本
    go.mod
  deploy/
    docker-compose.yml         # H5 + API + PostgreSQL + Redis 编排
    server.Dockerfile          # Go API 镜像构建
    web.Dockerfile             # H5 静态站点镜像构建
    nginx/default.conf         # H5 静态资源与 API 反向代理配置
  docs/
    engineering-technical-design.md
    openapi.yaml
    deployment-runbook.md
    glimmer-teacher-product-design-spec.md
    teacher-healing-management-prd.md
  prototype/                   # H5 原型与视觉参考
```

## 3. 总体架构

```mermaid
flowchart TD
  A[uni-app App] --> B[Go HTTP API]
  B --> C[PostgreSQL]
  B --> D[Redis]
  B --> E[AI Provider]
  B --> F[Push Provider]
```

### 3.1 客户端

客户端使用 uni-app + Vue 3，优先支持 H5、App、小程序的同构开发。

客户端职责：

- 页面展示与交互状态管理。
- 低成本本地反馈，例如 Toast、呼吸动画、按钮状态。
- 调用后端 API 获取课程、提醒、家长档案、沟通记录和 AI 结果。
- 当前已接入文件选择，用于 Excel/CSV 课表和班级名单导入；后续接入系统通知权限、音频播放和支付 SDK。

### 3.2 服务端

服务端使用 Go，当前采用标准 HTTP + chi 路由。

服务端职责：

- 用户数据、课表、提醒、家长档案、沟通记录的统一读写。
- AI 生成能力编排，包括 AI 夸夸和家长沟通回复。
- 课表和班级名单文件导入解析，当前支持 `.xlsx` 与 `.csv`。
- Redis 缓存、限流、AI 结果短期缓存和异步任务状态。
- 后续接入登录、支付、推送、对象存储和更完整的导入任务队列。

### 3.3 数据库

PostgreSQL 存储长期业务数据：用户、课表、提醒、家长档案、沟通记录、疗愈记录、AI 生成记录、收藏。

### 3.4 缓存

Redis 用于：

- 今日工作台摘要缓存：当前 API 已通过 DashboardCache 接入，写入提醒或家长档案后会主动失效当天缓存。

- 登录态和 Token 黑名单。
- AI 生成结果短期缓存，降低重复调用成本。
- 提醒/导入任务的临时状态。
- 后续分布式锁、限流和异步队列。

## 4. 运行方式

### 4.1 Docker 一键启动

```bash
cd LittleLight
docker compose -f deploy/docker-compose.yml up --build
```

服务：

- H5: `http://localhost:8081`
- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

健康检查：

```bash
curl http://localhost:8080/healthz
```

API 容器启动后会读取 `MIGRATIONS_DIR=/app/migrations` 并按文件名顺序执行 SQL 迁移。当前迁移脚本保持幂等，因此即使 PostgreSQL 数据卷已存在，新增字段或种子数据也可以随服务启动自动对齐。

H5 容器使用 `deploy/web.Dockerfile` 构建 uni-app H5 产物，并由 nginx 托管静态文件；同一个 nginx 配置会把 `/api/` 和 `/healthz` 代理到 API 容器。

### 4.2 前端 H5 开发

```bash
cd app
npm ci
npm run dev:h5
```

H5 默认端口在 `manifest.json` 中配置为 `5173`，并代理 `/api` 到 `http://localhost:8080`。

### 4.3 后端本地开发

```bash
cd server
go mod download
go run ./cmd/api
```

当前服务启动时会尝试连接 PostgreSQL 和 Redis。如果本地未安装依赖，会退回内存数据仓库，便于先开发 API 形态。

## 5. 环境变量

| 变量 | 示例 | 说明 |
| --- | --- | --- |
| APP_ENV | local / docker / prod | 运行环境 |
| HTTP_ADDR | :8080 | Go API 监听地址 |
| DATABASE_URL | postgres://littlelight:littlelight@postgres:5432/littlelight?sslmode=disable | PostgreSQL DSN |
| REDIS_ADDR | redis:6379 | Redis 地址 |
| REDIS_PASSWORD | 空 | Redis 密码 |
| REDIS_DB | 0 | Redis DB |
| MIGRATIONS_DIR | server/migrations / /app/migrations | SQL 迁移目录；Docker 容器内使用 `/app/migrations` |
| AI_PROVIDER | 空 / mock / llm / openai / qwen | AI 供应商；留空时会根据 LLM 配置自动判断 |
| AI_API_KEY | sk-xxx | 兼容旧配置；未设置 `LLM_API_KEY` 时作为 LLM Key 兜底 |
| LLM_API_KEY | sk-xxx | OpenAI-compatible LLM API Key；只放本地 `.env` 或部署密钥，不提交仓库 |
| LLM_BASE_URL | https://llmapi.example.com | OpenAI-compatible Base URL；服务会请求 `{LLM_BASE_URL}/v1/chat/completions` |
| LLM_MODEL | gpt-4o-mini | LLM 模型名；未设置时使用默认值 |
| GO_IMAGE | golang:1.22-alpine | Go API 构建阶段基础镜像，可替换为镜像代理或内网镜像 |
| ALPINE_IMAGE | alpine:3.20 | Go API 运行阶段基础镜像 |
| NODE_IMAGE | node:20-alpine | H5 构建阶段基础镜像 |
| NGINX_IMAGE | nginx:1.27-alpine | H5 静态站点运行镜像 |
| POSTGRES_IMAGE | postgres:16-alpine | PostgreSQL 服务镜像 |
| REDIS_IMAGE | redis:7-alpine | Redis 服务镜像 |
| GOPROXY | https://goproxy.cn,direct | Docker 构建 Go API 时使用的 Go module 代理 |
| NPM_REGISTRY | https://registry.npmjs.org/ | Docker 构建 H5 时使用的 npm registry |

## 6. 核心领域模型

### 6.1 UserProfile

教师个人资料。

核心字段：

- name：姓名。
- school：学校。
- stage：学段。
- subject：学科。
- is_head_teacher：是否班主任。
- pro_status：free / trial / pro / expired。
- reminder_policy：normal / low_interrupt。

### 6.2 Course

课表课程。

核心字段：

- title：课程名。
- class_name：班级或对象。
- location：地点。
- weekday：0 到 6。
- start_time / end_time：课程时间。
- repeat_rule：重复规则。
- note：备注。

### 6.3 Reminder

待办事项和低打扰提醒。

核心字段：

- title：提醒标题。
- category：调课、会议、家长会、教研、值班、活动、回访、个人事项。
- remind_at：提醒时间。
- status：pending / done / snoozed / deleted。
- parent_id：可关联家长档案。
- course_id：可关联课程。

### 6.4 ParentProfile

家长档案。

核心字段：

- student_name：学生姓名。
- class_name：班级。
- parent_name：家长称呼。
- relationship：关系。
- contact：联系方式。
- communication_style：沟通风格。
- risk_level：low / medium / high。
- important_notes：重要备注。
- next_action：下一步动作。

### 6.5 CommunicationRecord

沟通记录。

核心字段：

- parent_id：关联家长。
- student：学生。
- channel：微信、电话、面谈、家长会、短信、其他。
- reason：沟通原因。
- summary：沟通摘要。
- result：沟通结果。
- risk_level：风险等级。
- follow_up_at：下次跟进时间。

### 6.6 HealingEntry

疗愈行为记录。

核心字段：

- type：breath / praise / treehole / sound。
- mood：情绪标签。
- content：用户输入。
- ai_reply：AI 回应。
- visibility：private / group。

### 6.7 AIGeneration

AI 生成记录。

核心字段：

- scenario：parent_drafts / praise。后续可扩展 communication_summary。
- input：生成输入。
- output：生成结果。
- safety_label：安全标签。
- token_usage：Token 用量。

## 7. 数据库设计

迁移脚本：

- `server/migrations/001_init.sql`
- `server/migrations/002_seed.sql`

迁移执行策略：

- API 启动时调用 `database.Migrate`，读取 `MIGRATIONS_DIR` 下所有 `.sql` 文件并按文件名顺序执行。
- Docker 镜像会把 `server/migrations` 复制到 `/app/migrations`，Compose 中显式设置 `MIGRATIONS_DIR=/app/migrations`。
- 本地直接运行 `go run ./cmd/api` 时可使用 `.env.example` 中的 `MIGRATIONS_DIR=server/migrations`，也可以按实际工作目录覆盖；迁移器会依次尝试 `MIGRATIONS_DIR`、`migrations`、`server/migrations`、`/app/migrations`。
- 当前迁移脚本使用 `CREATE TABLE IF NOT EXISTS`、`CREATE INDEX IF NOT EXISTS`、`ON CONFLICT DO NOTHING` 等幂等写法，适合服务启动时重复执行。后续涉及结构变更时，新增迁移必须继续保持可重复执行。
- `APP_ENV=local` 时迁移失败会降级到内存仓库，便于本地先调通 API；`APP_ENV=docker/prod` 时迁移失败会直接退出进程，避免部署环境静默丢失持久化。

当前表：

- users
- courses
- parent_profiles
- reminders
- communication_records
- healing_entries
- ai_generations
- favorites

索引策略：

- 课程按 `user_id + weekday` 查询。
- 提醒按 `user_id + remind_at + status` 查询。
- 家长档案按 `user_id + risk_level` 查询。
- 沟通记录按 `user_id + parent_id + created_at` 查询。
- 疗愈记录和 AI 生成按用户与创建时间倒序查询。

## 8. API 设计

基础路径：`/api/v1`

开发阶段鉴权：当前提供 `POST /api/v1/auth/wechat/mock` 作为微信模拟登录入口，返回模拟 `sessionToken`、`openId` 和当前教师资料；前端保存 `userId` 并在后续请求中携带 `X-User-ID`。未传时后端仍默认使用种子用户 `00000000-0000-0000-0000-000000000001`，便于接口调试。后续接入真实微信登录时，可保持前端调用形态不变，将 mock code 换成微信 code 并由后端换取 openid/session。

机器可读契约：`docs/openapi.yaml`。后端路由、前端 `app/src/api/client.js` 和手工检查清单应以该文件保持一致。

### 8.1 健康检查

`GET /healthz`

返回：

```json
{ "status": "ok", "time": "2026-05-27T14:00:00+08:00" }
```

### 8.2 今日工作台

`GET /api/v1/dashboard?day=2026-05-27`

返回：

```json
{
  "todayLabel": "2026-05-27",
  "coursesCount": 2,
  "remindersCount": 2,
  "followUpsCount": 2,
  "nextCourse": {},
  "reminders": [],
  "focusParents": [],
  "rhythm": {
    "code": "steady",
    "title": "温柔但高效",
    "description": "今天事项可控，先处理下一节课，再推进家长跟进。"
  }
}
```

### 8.3 我的资料与收藏

`GET /api/v1/me`

`PUT /api/v1/me`

```json
{
  "name": "林小微",
  "school": "微光实验小学",
  "stage": "小学",
  "subject": "语文",
  "isHeadTeacher": true,
  "proStatus": "trial",
  "reminderPolicy": "low_interrupt"
}
```

`GET /api/v1/me/favorites?type=communication_template`

`POST /api/v1/me/favorites`

```json
{
  "type": "communication_template",
  "title": "先共情再同步",
  "content": "我理解您对孩子状态的担心，我先同步今天观察到的具体表现，再一起看下一步。"
}
```

`DELETE /api/v1/me/favorites/{id}`

### 8.4 课程

`GET /api/v1/courses?weekday=3`

用途：日程页切换日期时读取课程列表。

当前已实现接口：

- `GET /api/v1/courses/{id}`
- `POST /api/v1/courses`
- `PUT /api/v1/courses/{id}`
- `DELETE /api/v1/courses/{id}`

请求示例：

```json
{
  "title": "心理健康",
  "className": "高二(3)班",
  "location": "教学楼 B 座 402 室",
  "weekday": 3,
  "startTime": "09:30",
  "endTime": "10:15",
  "note": "情绪识别与压力调节"
}
```

导入接口：

- `POST /api/v1/courses/imports`，上传 `.xlsx` 或 `.csv` 课表。
- 请求使用 `multipart/form-data`，文件字段名为 `file`。
- 当前支持表头：课程名称/课程/科目、班级、星期/周几、开始时间、结束时间、地点/教室、备注。
- 后端限制单文件 5MB、单次最多解析 300 行，返回导入成功数、跳过数和行级错误。

### 8.5 提醒

`GET /api/v1/reminders?day=2026-05-27`

`POST /api/v1/reminders`

```json
{
  "title": "回访陈子默爸爸",
  "category": "回访",
  "remindAt": "2026-05-27T17:20:00+08:00",
  "note": "同步测试反馈和订正计划"
}
```

`POST /api/v1/reminders/{id}/complete`

当前已实现接口：

- `GET /api/v1/reminders/{id}`
- `PUT /api/v1/reminders/{id}`
- `DELETE /api/v1/reminders/{id}`，软删除为 `status=deleted`
- `POST /api/v1/reminders/{id}/snooze`

延后提醒请求：

```json
{
  "until": "2026-05-27T18:00:00+08:00"
}
```

### 8.6 家长档案

`GET /api/v1/parents`

`POST /api/v1/parents`

```json
{
  "studentName": "林晓晓",
  "className": "高二(5)班",
  "parentName": "林晓晓妈妈",
  "relationship": "母亲",
  "communicationStyle": "比较敏感",
  "riskLevel": "medium",
  "importantNotes": "近期睡眠不足",
  "nextAction": "同步课堂参与中的积极信号"
}
```

当前已实现接口：

- `GET /api/v1/parents/{id}`
- `PUT /api/v1/parents/{id}`
- `DELETE /api/v1/parents/{id}`

导入接口：

- `POST /api/v1/parents/imports`，上传 `.xlsx` 或 `.csv` 班级名单。
- 请求使用 `multipart/form-data`，文件字段名为 `file`。
- 当前支持表头：学生姓名/学生、班级、家长姓名/家长、关系、联系方式/手机号、沟通风格/家长风格、风险等级、重点备注、下一步。
- 导入时学生姓名和班级必填；家长姓名为空时默认生成为“学生姓名 + 家长”，关系为空时默认为“家长”。

### 8.7 沟通记录

`GET /api/v1/communication-records?parentId=xxx`

`POST /api/v1/communication-records`

```json
{
  "parentId": "parent_chen",
  "student": "陈子默",
  "channel": "微信",
  "reason": "测试反馈",
  "summary": "同步测试问题和订正方向",
  "result": "家长认可三天订正计划",
  "riskLevel": "low",
  "followUpAt": "2026-05-30T17:20:00+08:00"
}
```

当前已实现接口：

- `GET /api/v1/communication-records/{id}`
- `PUT /api/v1/communication-records/{id}`
- `DELETE /api/v1/communication-records/{id}`

### 8.8 AI 家长回复

`POST /api/v1/ai/parent-drafts`

```json
{
  "issue": "孩子最近课堂专注度下降，希望语气温和一点",
  "parentStyle": "容易焦虑",
  "tone": "温和",
  "studentName": "浩宇"
}
```

返回多个版本：温和、正式、简短、坚定但礼貌。

生成成功后写入 `ai_generations`，记录 `scenario=parent_drafts`、输入参数、候选草稿、`teacher_review_required` 安全标记。

安全要求：

- 输出默认需要老师确认。
- 不使用绝对化表达。
- 不给学生贴负面标签。
- 不指责家长。

### 8.9 AI 夸夸

`POST /api/v1/ai/praise`

```json
{
  "persona": "温柔前辈",
  "content": "今天课很多，还处理了家长反馈",
  "mood": "tired"
}
```

生成成功后写入 `ai_generations`，记录 `scenario=praise`、输入参数、生成内容和 `self_care` 安全标记。疗愈页保存 AI 夸夸时会再写入 `healing_entries`，两张表分别服务“AI 质量复盘”和“用户疗愈历史”。

### 8.10 AI 生成审计

`GET /api/v1/ai/generations`

按当前用户返回最近 50 条 AI 生成记录，支持 `scenario=parent_drafts|praise` 过滤。

`GET /api/v1/ai/generations/{id}`

返回单条 AI 生成记录，包含输入、输出、安全标签、token 用量和创建时间。

用途：

- 质量复盘：检查家校沟通草稿是否符合安全策略。
- 成本统计：后续真实模型接入后记录 token 用量。
- 素材沉淀：老师可从生成结果中收藏常用模板或夸夸语。

### 8.11 疗愈记录

`GET /api/v1/healing/entries`

支持按 `type` 过滤：`breath`、`praise`、`treehole`、`sound`。

`POST /api/v1/healing/entries`

用于保存 AI 夸夸、树洞、呼吸、声音播放等疗愈行为。

`GET /api/v1/healing/entries/{id}`

用于进入单条疗愈记录详情，后续可承载更完整的复盘内容。

`DELETE /api/v1/healing/entries/{id}`

用于删除老师主动清理的私密疗愈记录。

## 9. 缓存设计

| Key | TTL | 内容 |
| --- | --- | --- |
| dashboard:{userId}:{date} | 5 分钟 | 今日工作台摘要 |
| ai:parent-draft:{hash} | 24 小时 | 家长回复生成结果 |
| ai:praise:{hash} | 12 小时 | AI 夸夸生成结果 |
| import:{jobId} | 30 分钟 | Excel 导入任务进度 |
| reminder:due:{minute} | 2 分钟 | 待触达提醒集合 |

缓存原则：

- 数据写入后需要主动删除相关 dashboard 缓存。
- AI 缓存按输入 hash 命中，避免重复调用。
- 涉及隐私内容的缓存必须设置 TTL，不长期保留。

## 10. AI 服务设计

### 10.1 Provider 抽象

后续应抽象为：

```go
type AIProvider interface {
    GenerateParentDrafts(ctx context.Context, input ParentDraftInput) ([]AIDraft, error)
    GeneratePraise(ctx context.Context, input PraiseInput) (AIDraft, error)
    SummarizeCommunication(ctx context.Context, input RecordInput) (Summary, error)
}
```

### 10.2 安全策略

- 敏感家校沟通输出必须保留 `teacher_review_required` 标签。
- AI 不直接发送消息给家长。
- 所有 AI 生成都写入 `ai_generations`，便于质量复盘和成本统计。
- Prompt 中必须禁止攻击性、污名化、绝对化表达。

### 10.3 成本策略

- 免费版限制每日 AI 生成次数。
- 常见场景优先模板 + 小模型。
- Pro 用户使用高质量模型。
- 对同一输入短期缓存。

## 11. uni-app 页面设计

### 11.1 页面列表

| 页面 | 路径 | 职责 |
| --- | --- | --- |
| 首页 | pages/home/index | 今日工作台、状态、下一项、快捷入口 |
| 疗愈 | pages/heal/index | 呼吸、AI 夸夸、疗愈记录 |
| 沟通 | pages/communication/index | AI 回复、家长关注、草稿复制 |
| 家长档案详情 | pages/communication/parent-detail | 档案编辑、风险等级、重点备注、沟通记录 |
| 日程 | pages/schedule/index | 周日期、课程、待办提醒 |
| 我的 | pages/profile/index | 资料、素材库、Pro、云同步 |

### 11.2 状态管理

当前工程先使用页面内 `ref` 管理状态。进入第二阶段后建议引入 Pinia store：

- useUserStore
- useDashboardStore
- useScheduleStore
- useCommunicationStore
- useHealingStore

### 11.3 API 调用

统一通过 `app/src/api/client.js` 调用，避免页面散落 `uni.request`。

当前已接入接口：

- 首页：`GET /dashboard`
- 日程：`GET/POST/PUT/DELETE /courses`、`GET/POST/PUT/DELETE /reminders`
- 沟通：`GET/POST/PUT/DELETE /parents`、`GET/POST/PUT/DELETE /communication-records`、`POST /ai/parent-drafts`
- 疗愈：`GET/POST/DELETE /healing/entries`、`POST /ai/praise`
- 我的：`GET/PUT /me`、`GET/POST/DELETE /me/favorites`、`GET /ai/generations`

## 12. 部署设计

### 12.1 开发环境

Docker Compose 启动 H5 Web、API、PostgreSQL、Redis。uni-app H5 也可以本地运行，通过 Vite 代理访问 API。

### 12.2 生产环境建议

- API 使用多副本容器部署。
- PostgreSQL 使用云数据库或独立持久化卷。
- Redis 使用托管 Redis 或持久化容器。
- Nginx / API Gateway 负责 TLS、CORS、压缩、限流。
- 日志输出到 stdout，由平台采集。

### 12.3 数据持久化

Docker Compose 中：

- H5 静态产物由 nginx 容器托管，不保存业务状态。
- postgres_data：数据库数据卷。
- redis_data：Redis AOF 数据卷。

### 12.4 备份策略

- PostgreSQL 每日全量备份。
- 关键表启用 PITR 或云数据库备份。
- Redis 不作为唯一事实来源，只做缓存和临时任务状态。

## 13. 安全与隐私

- 所有家长联系方式、沟通记录、树洞内容都属于敏感数据。
- 正式环境必须启用 HTTPS。
- 服务端必须做用户身份校验，禁止跨用户读取数据。
- AI 生成内容不能自动发送给家长。
- 树洞默认私密，不自动公开。
- Excel 导入文件需要限制大小、类型和有效期，导入后及时删除原文件。

## 14. 后续工程任务

### P0

- 接入真实 PostgreSQL repository。当前已提供 `PostgresStore`，服务启动时 PostgreSQL 可用则使用持久化仓库，否则 fallback 到内存仓库。
- 完成真实微信登录和用户鉴权；当前已完成微信模拟登录闭环。
- 完成课程、提醒、家长档案、沟通记录 CRUD。
- 完成 AI Provider 抽象和真实模型接入。
- 完成 uni-app 页面接口联调。

### P1

- 完善 Excel/CSV 导入预览、字段映射配置和导入回滚。
- 完善家长档案详情时间线和附件。
- 沟通记录详情页。
- 提醒延后、编辑、删除。
- 收藏列表与素材库。
- Pro 权益页。

### P2

- 推送通知。
- 白噪音真实音频资源。
- 树洞历史与小范围可见。
- AI 沟通风险识别。
- 云同步状态页。

## 15. 当前实现状态

已完成：

- 工程目录拆分：`app/`、`server/`、`deploy/`、`docs/`。
- PostgreSQL repository 已接入，覆盖 Dashboard、课程、提醒、家长档案、沟通记录、疗愈记录。
- 课程、提醒、家长档案、沟通记录已具备列表、详情、新增、编辑、删除等核心 CRUD；提醒额外支持完成和延后。
- 后端单元测试初版已补充，覆盖 HTTP 路由、微信模拟登录、`X-User-ID` 开发鉴权、内存仓库 CRUD 和 AI 服务。
- GitHub Actions 初版已补充，包含 OpenAPI 解析、Docker Compose 配置解析、Go 测试和 uni-app H5 构建。
- Go/Node 依赖锁已补齐：`server/go.sum`、`app/package-lock.json` 已提交，CI 与 Docker Web 镜像使用锁文件进行可复现安装。
- uni-app 五个 Tab 页面骨架，日程页已接入课程/待办增删改查入口，沟通页已接入家长档案、家长详情和沟通记录基础操作，我的页已接入教师资料与收藏素材管理。
- Go API 服务骨架。
- V1 核心领域模型。
- 内存仓库用于本地无数据库演示。
- PostgreSQL 初始化迁移脚本与 API 启动时自动迁移。
- 教师资料与收藏素材 API 已接入 `users`、`favorites` 表。
- H5 Web/API/Redis/PostgreSQL/Docker Compose 配置。
- Redis dashboard 缓存已接入，读取首页时优先查缓存，课程、提醒、家长写入成功后清理缓存。
- API readiness 检查已接入：`/healthz` 表示进程存活，`/readyz` 会检查 PostgreSQL 与 Redis；Docker API 容器使用 `/readyz` 作为健康检查。
- 本地逻辑验证脚本已补充并通过：PostgreSQL 与 Redis 由 Docker Compose 提供，本机 Go API 连接容器完成健康检查、业务写入查询、数据库落库和 Redis 缓存键验证。
- 微信模拟登录已接入，前端“我的”页可发起模拟登录并保存登录态；HTTP 开发鉴权中间件支持 `X-User-ID` 并保留默认种子用户。
- Excel/CSV 课表导入和班级名单导入已接入前端入口与后端解析接口；当前支持 `.xlsx` 与 `.csv`，暂不解析老式二进制 `.xls`。
- 详细技术文档。

待完成：

- 完整 Web/API Docker 镜像构建仍依赖 Docker Hub 基础镜像可拉取；若 `auth.docker.io` 网络不可用，先使用 `scripts/verify-docker.ps1` 验证数据库、缓存和后端业务逻辑。
- 真实微信 code 换 session 尚未接入；当前为微信模拟登录 + `X-User-ID` 开发态。



