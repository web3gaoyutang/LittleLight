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
# 启动后端依赖和 API
docker compose -f deploy/docker-compose.yml up --build

# 启动 uni-app H5
cd app
npm install
npm run dev:h5
```

完整技术文档见：`docs/engineering-technical-design.md`。
