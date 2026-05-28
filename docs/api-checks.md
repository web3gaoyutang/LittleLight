# API 手工检查示例

以下命令默认服务运行在 `http://localhost:8080`，开发用户通过 `X-User-ID` 指定。完整 API 契约见 `docs/openapi.yaml`。

```bash
export API=http://localhost:8080
export USER_ID=00000000-0000-0000-0000-000000000001

curl "$API/healthz"

curl -X POST "$API/api/v1/auth/wechat/mock" \
  -H "Content-Type: application/json" \
  -d '{"code":"dev-login","nickName":"林小微"}'

curl -H "X-User-ID: $USER_ID" "$API/api/v1/dashboard?day=2026-05-27"
```

## 我的资料与收藏

```bash
curl -H "X-User-ID: $USER_ID" "$API/api/v1/me"

curl -X PUT "$API/api/v1/me" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"name":"林小微","school":"微光实验小学","stage":"小学","subject":"语文","isHeadTeacher":true,"proStatus":"trial","reminderPolicy":"low_interrupt"}'

curl -H "X-User-ID: $USER_ID" "$API/api/v1/me/favorites"

curl -X POST "$API/api/v1/me/favorites" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"type":"communication_template","title":"先共情再同步","content":"我理解您对孩子状态的担心，我先同步今天观察到的具体表现，再一起看下一步。"}'
```

## 课程

```bash
curl -H "X-User-ID: $USER_ID" "$API/api/v1/courses?weekday=3"

curl -X POST "$API/api/v1/courses" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"title":"心理健康","className":"高二(3)班","location":"教学楼 B 座 402 室","weekday":3,"startTime":"09:30","endTime":"10:15","note":"情绪识别与压力调节"}'

curl -X POST "$API/api/v1/courses/imports" \
  -H "X-User-ID: $USER_ID" \
  -F "file=@./courses.csv"

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

curl -X POST "$API/api/v1/parents/imports" \
  -H "X-User-ID: $USER_ID" \
  -F "file=@./parents.csv"

curl -H "X-User-ID: $USER_ID" "$API/api/v1/communication-records?parentId={parentId}"

curl -X POST "$API/api/v1/communication-records" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"parentId":"{parentId}","student":"林晓晓","channel":"微信","reason":"睡眠观察","summary":"同步近期观察","result":"约定明天继续反馈","riskLevel":"medium","followUpAt":"2026-05-28T17:20:00+08:00"}'
```

## AI

```bash
curl -X POST "$API/api/v1/ai/parent-drafts" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{"issue":"孩子最近课堂专注度下降","parentStyle":"容易焦虑","tone":"温和"}'

curl -X POST "$API/api/v1/ai/praise" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{"persona":"温柔前辈","content":"今天课很多，还处理了家长反馈"}'

curl -H "X-User-ID: $USER_ID" "$API/api/v1/ai/generations"
curl -H "X-User-ID: $USER_ID" "$API/api/v1/ai/generations?scenario=praise"
curl -H "X-User-ID: $USER_ID" "$API/api/v1/ai/generations/{id}"
```

## 疗愈记录

```bash
curl -H "X-User-ID: $USER_ID" "$API/api/v1/healing/entries"
curl -H "X-User-ID: $USER_ID" "$API/api/v1/healing/entries?type=praise"

curl -X POST "$API/api/v1/healing/entries" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"type":"praise","mood":"warm","content":"今天课很多，还处理了家长反馈","aiReply":"你已经处理了很多复杂信息，先给自己一点恢复空间。"}'

curl -H "X-User-ID: $USER_ID" "$API/api/v1/healing/entries/{id}"
curl -X DELETE -H "X-User-ID: $USER_ID" "$API/api/v1/healing/entries/{id}"
```
