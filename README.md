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
go run ./cmd/migrate
go run .
```

## 前端本地启动

```bash
cd apps/web
cp .env.example .env.local
pnpm install
pnpm dev
```

本地前端 `http://localhost:5176`，后端 `http://127.0.0.1:3458`。
数据库使用独立 `atlas_db`。Helios 统一登录配置、初始化和成员授权见 [登录接入说明](docs/Helios登录接入.md)。

已有数据库升级同类型多入口支持时，在 `apps/api` 执行 `go run ./cmd/migrate --multiple-project-entries`，移除旧的项目入口类型唯一索引。完整迁移 `go run ./cmd/migrate` 也会执行此步骤，可重复运行。资源入口可分别命名为“测试发布系统”“线上发布系统”，类型均选“发布系统”；名称留空时沿用类型名称。
