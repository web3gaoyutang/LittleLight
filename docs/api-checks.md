# API 手工检查示例

以下命令默认服务运行在 `http://localhost:8080`，开发用户通过 `X-User-ID` 指定。完整 API 契约见 `docs/openapi.yaml`。

```bash
export API=http://localhost:8080
export USER_ID=00000000-0000-0000-0000-000000000001

curl "$API/healthz"
curl -H "X-User-ID: $USER_ID" "$API/api/v1/dashboard?day=2026-05-27"
```

## 课程

```bash
curl -H "X-User-ID: $USER_ID" "$API/api/v1/courses?weekday=3"

curl -X POST "$API/api/v1/courses" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"title":"心理健康","className":"高二(3)班","location":"教学楼 B 座 402 室","weekday":3,"startTime":"09:30","endTime":"10:15","note":"情绪识别与压力调节"}'

curl -X PUT "$API/api/v1/courses/{id}" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"title":"心理健康","className":"高二(3)班","location":"教学楼 B 座 402 室","weekday":3,"startTime":"09:40","endTime":"10:20","note":"更新后的课程安排"}'

curl -X DELETE -H "X-User-ID: $USER_ID" "$API/api/v1/courses/{id}"
```

## 提醒

```bash
curl -H "X-User-ID: $USER_ID" "$API/api/v1/reminders?day=2026-05-27"

curl -X POST "$API/api/v1/reminders" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"title":"回访陈子默爸爸","category":"回访","remindAt":"2026-05-27T17:20:00+08:00","note":"同步测试反馈和订正计划"}'

curl -X POST -H "X-User-ID: $USER_ID" "$API/api/v1/reminders/{id}/complete"

curl -X POST "$API/api/v1/reminders/{id}/snooze" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"until":"2026-05-27T18:00:00+08:00"}'

curl -X DELETE -H "X-User-ID: $USER_ID" "$API/api/v1/reminders/{id}"
```

## 家长档案与沟通记录

```bash
curl -H "X-User-ID: $USER_ID" "$API/api/v1/parents"

curl -X POST "$API/api/v1/parents" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"studentName":"林晓晓","className":"高二(5)班","parentName":"林晓晓妈妈","relationship":"母亲","contact":"13800000000","communicationStyle":"比较敏感","riskLevel":"medium","importantNotes":"近期睡眠不足","nextAction":"同步课堂参与中的积极信号"}'

curl -H "X-User-ID: $USER_ID" "$API/api/v1/communication-records?parentId={parentId}"

curl -X POST "$API/api/v1/communication-records" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"parentId":"{parentId}","student":"林晓晓","channel":"微信","reason":"睡眠观察","summary":"同步近期观察","result":"约定明天继续反馈","riskLevel":"medium","followUpAt":"2026-05-28T17:20:00+08:00"}'
```

## AI

```bash
curl -X POST "$API/api/v1/ai/parent-drafts" \
  -H "Content-Type: application/json" \
  -d '{"issue":"孩子最近课堂专注度下降","parentStyle":"容易焦虑","tone":"温和"}'

curl -X POST "$API/api/v1/ai/praise" \
  -H "Content-Type: application/json" \
  -d '{"persona":"温柔前辈","content":"今天课很多，还处理了家长反馈"}'
```
