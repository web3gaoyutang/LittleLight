# 数据库 Schema 说明

本文档对应 `server/migrations`，用于说明 PostgreSQL 中的核心业务表、关键字段、索引和数据所有权。迁移脚本仍是唯一执行来源；本文档用于评审、联调和运维排障。

## 总体原则

- 所有业务表通过 `user_id` 归属到教师用户；开发种子数据归属 `00000000-0000-0000-0000-000000000001`，但 HTTP 鉴权不会在缺少登录态时自动落到该用户。
- PostgreSQL 是业务事实来源；Redis 只保存 dashboard 缓存和后续临时状态。
- 所有主键使用 UUID，默认由 `uuid_generate_v4()` 生成。
- 关键枚举、课程时间顺序和业务表 `user_id` 归属在数据库层也有约束，避免绕过 API 产生脏数据。
- `created_at` 记录创建时间；有编辑语义的表同时保留 `updated_at`。
- 当前迁移为向前幂等脚本，尚未提供独立 down migration。

## 表结构

### users

教师账号和个人资料。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 用户 ID |
| wechat_open_id | TEXT UNIQUE | 微信 openid，真实/模拟登录的账号映射键 |
| name | TEXT NOT NULL | 教师姓名 |
| avatar_url | TEXT | 微信头像或用户头像 URL |
| school | TEXT | 学校 |
| stage | TEXT | 学段 |
| subject | TEXT | 学科 |
| is_head_teacher | BOOLEAN | 是否班主任 |
| pro_status | TEXT | free / trial / pro / expired |
| reminder_policy | TEXT | low_interrupt / normal |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

索引：`idx_users_wechat_open_id(wechat_open_id)`，仅索引非空 openid。

约束：`pro_status` 仅允许 free / trial / pro / expired；`reminder_policy` 仅允许 low_interrupt / normal。

### auth_sessions

服务端登录态，用于校验和撤销 Bearer session。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | Session ID |
| user_id | UUID FK users(id) | 所属教师 |
| token_hash | TEXT UNIQUE | sessionToken 的 SHA-256 摘要，不保存明文 token |
| expires_at | TIMESTAMPTZ | 过期时间 |
| revoked_at | TIMESTAMPTZ | 撤销时间，非空表示不可再用 |
| created_at | TIMESTAMPTZ | 创建时间 |

索引：`idx_auth_sessions_user_active(user_id, expires_at)`，仅索引未撤销 session。

### courses

周课表课程。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 课程 ID |
| user_id | UUID FK users(id) | 所属教师 |
| title | TEXT NOT NULL | 课程名 |
| class_name | TEXT NOT NULL | 班级或对象 |
| location | TEXT | 地点 |
| weekday | SMALLINT | 0-6，0 表示周日 |
| start_time | TIME | 开始时间 |
| end_time | TIME | 结束时间 |
| repeat_rule | TEXT | 当前默认 weekly |
| color | TEXT | 预留课程颜色 |
| note | TEXT | 备注 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

索引：`idx_courses_user_weekday(user_id, weekday)`。

约束：`user_id` 非空；`weekday` 为 0-6；`end_time` 必须晚于 `start_time`。

### parent_profiles

家长档案和重点关注信息。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 家长档案 ID |
| user_id | UUID FK users(id) | 所属教师 |
| student_name | TEXT NOT NULL | 学生姓名 |
| class_name | TEXT NOT NULL | 班级 |
| parent_name | TEXT NOT NULL | 家长称呼 |
| relationship | TEXT NOT NULL | 关系 |
| contact | TEXT | 联系方式 |
| communication_style | TEXT | 沟通风格 |
| risk_level | TEXT | low / medium / high |
| important_notes | TEXT | 重点观察 |
| next_action | TEXT | 下一步动作 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

索引：`idx_parent_profiles_user_risk(user_id, risk_level)`。

约束：`user_id` 非空；`risk_level` 仅允许 low / medium / high。

### reminders

待办事项和提醒。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 提醒 ID |
| user_id | UUID FK users(id) | 所属教师 |
| parent_id | UUID FK parent_profiles(id) | 可关联家长 |
| course_id | UUID FK courses(id) | 可关联课程 |
| title | TEXT NOT NULL | 标题 |
| category | TEXT | 分类，默认 personal |
| remind_at | TIMESTAMPTZ | 提醒时间 |
| status | TEXT | pending / done / snoozed / deleted |
| note | TEXT | 备注 |
| done_at | TIMESTAMPTZ | 完成时间 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

索引：`idx_reminders_user_time_status(user_id, remind_at, status)`。

约束：`user_id` 非空；`status` 仅允许 pending / done / snoozed / deleted。

### communication_records

家校沟通记录。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 沟通记录 ID |
| user_id | UUID FK users(id) | 所属教师 |
| parent_id | UUID FK parent_profiles(id) | 关联家长 |
| student | TEXT NOT NULL | 学生姓名快照 |
| channel | TEXT NOT NULL | 微信 / 电话 / 面谈等 |
| reason | TEXT NOT NULL | 沟通原因 |
| summary | TEXT NOT NULL | 沟通摘要 |
| result | TEXT | 沟通结果 |
| risk_level | TEXT | low / medium / high |
| follow_up_at | TIMESTAMPTZ | 后续跟进时间 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

索引：`idx_records_user_parent(user_id, parent_id, created_at DESC)`。

约束：`user_id` 非空；`risk_level` 仅允许 low / medium / high。

### healing_entries

疗愈记录，包括呼吸、AI 夸夸、树洞、声音等。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 记录 ID |
| user_id | UUID FK users(id) | 所属教师 |
| type | TEXT NOT NULL | breath / praise / treehole / sound |
| mood | TEXT | 情绪 |
| content | TEXT | 用户输入或记录内容 |
| ai_reply | TEXT | AI 回复 |
| visibility | TEXT | 默认 private |
| created_at | TIMESTAMPTZ | 创建时间 |

索引：`idx_healing_user_created(user_id, created_at DESC)`。

约束：`user_id` 非空；`type` 仅允许 breath / praise / treehole / sound；`visibility` 当前仅允许 private。

### ai_generations

AI 生成审计记录。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 生成记录 ID |
| user_id | UUID FK users(id) | 所属教师 |
| scenario | TEXT NOT NULL | parent_drafts / praise |
| input | JSONB | 输入快照 |
| output | JSONB | 输出快照 |
| safety_label | TEXT | 安全标记 |
| token_usage | INTEGER | Token 用量，当前可为 0 |
| created_at | TIMESTAMPTZ | 创建时间 |

索引：`idx_ai_generations_user_created(user_id, created_at DESC)`。

约束：`user_id` 非空；`scenario` 仅允许 parent_drafts / praise；`safety_label` 仅允许 teacher_review_required / self_care / crisis_support_required / student_safety_review_required / medical_review_required。

### favorites

收藏素材。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | UUID PK | 收藏 ID |
| user_id | UUID FK users(id) | 所属教师 |
| type | TEXT NOT NULL | communication_template / ai_praise / class_feedback 等 |
| title | TEXT NOT NULL | 标题 |
| content | TEXT NOT NULL | 内容 |
| source_id | UUID | 来源记录，当前不强制外键 |
| created_at | TIMESTAMPTZ | 创建时间 |

约束：`user_id` 非空；`type` 仅允许 communication_template / ai_praise / class_feedback。

## 种子数据

`002_seed.sql` 会写入：

- 默认教师用户。
- 两个家长档案。
- 两条课程。
- 两条收藏素材。
- 两条疗愈记录。
- 一条 AI 夸夸审计记录。

种子数据使用固定 UUID，并通过 `ON CONFLICT (id) DO NOTHING` 保持幂等。

## 迁移约定

- 新增表或字段时必须更新 `server/migrations`、`docs/database-schema.md`、`docs/openapi.yaml` 和相关 repository 测试。
- 删除字段、改字段类型、批量重写数据属于破坏性迁移，发布前必须先执行 PostgreSQL 备份，并在发布说明中写明恢复路径。
- 如果新增 API 路由，`TestRoutesMatchOpenAPIContract` 会要求同步 OpenAPI path/method。
