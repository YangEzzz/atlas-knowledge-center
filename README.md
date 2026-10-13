# Atlas 研发知识中心

公司内部研发项目入口与知识门户。

## 目录

```text
apps/api   Go + Gin + GORM 后端
apps/web   Vue 3 + Vite 前端
docs       产品、接口、数据模型和技术文档
```

## 后端本地启动

```bash
cd apps/api
cp .env-example .env
go mod tidy
PORT=8080 DB_ENABLED=false go run main.go
```

## 前端本地启动

```bash
cd apps/web
cp .env.example .env.local
pnpm install
pnpm dev
```
