# API 手工检查示例

curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/dashboard
curl http://localhost:8080/api/v1/courses?weekday=3
curl http://localhost:8080/api/v1/parents

curl -X POST http://localhost:8080/api/v1/ai/parent-drafts \
  -H "Content-Type: application/json" \
  -d '{"issue":"孩子最近课堂专注度下降","parentStyle":"容易焦虑","tone":"温和"}'

# 指定开发用户

curl -H "X-User-ID: 00000000-0000-0000-0000-000000000001" http://localhost:8080/api/v1/dashboard
