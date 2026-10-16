# Atlas Helios 登录接入

Atlas 沿用 Nexus 当前的 Helios SSO 实现（授权码 + PKCE + 服务端 Cookie 会话），替代原技术方案中的密码代理登录/JWT 设想。

## 本地配置

- Atlas 前端：http://localhost:5176
- Atlas API：http://127.0.0.1:3458
- Helios 前端：https://helios.team-tool.top
- Helios API：https://helios-api.team-tool.top
- Helios 项目标识与 client ID：atlas-knowledge-center
- 登录返回地址：http://localhost:5176/api/v1/auth/sso/callback
- 数据库：与 Nexus 相同服务器、端口及账号，独立 atlas_db。

密钥仅写入被 Git 忽略的 apps/api/.env。不得复制 Nexus 的 SSO 密钥或把实际数据库密码写入示例文件。

本地开发直接使用已部署的 Helios，`HELIOS_BASE_URL` 和 `HELIOS_INTERNAL_URL` 均配置为 `https://helios-api.team-tool.top`，无需启动本地 Helios 前后端。Atlas 仍使用本地回调地址；线上 Helios 必须已登记上述 client ID、对应密钥及精确返回地址。修改配置后重启 Atlas API。

## 初始化与运行

在 apps/api 执行 `go run ./cmd/setup-auth` 创建独立数据库、Helios 项目与 SSO 配置。重复执行保留现有密钥，不修改已有项目权限，也不自动复制 Nexus 成员。
执行 `go run ./cmd/migrate` 初始化 Atlas 数据表及登录事务/会话表，然后执行 `go run .`。
在 apps/web 执行 `pnpm install --frozen-lockfile`、`pnpm dev`。
首次实际登录前，需在 Helios 的 Atlas 项目成员管理中添加授权账号及项目角色。

## 接口与会话

公开接口：GET /auth/csrf、POST /auth/sso/start、GET /auth/sso/callback、GET /auth/sso/status、POST /auth/sso/cancel、POST /auth/logout；原 POST /auth/login 返回 410。
GET /auth/me 与全部业务接口由 Cookie 会话保护。接口前缀 /api/v1。
Cookie 使用 atlas_session、atlas_csrf、atlas_browser，避免与 Nexus 的 Cookie 冲突。
登录事务有效期 10 分钟，Helios 与 Atlas 的 SSO 会话均为登录后固定 30 天，不随访问续期；相关 Cookie 使用相同有效期。写请求及超过 5 分钟的读取重新验证 Helios 账号和项目角色。
新有效期在对应后端部署或重启后，对重新登录创建的会话生效；既有会话不自动延长。
所有写请求校验 Origin 与 X-CSRF-Token；state 绑定浏览器，PKCE 校验授权码；外部跳转和 /auth/ 返回地址被拒绝。

## 生产配置

配置实际 FRONTEND_URL、HELIOS_BASE_URL、HELIOS_INTERNAL_URL、COOKIE_SECURE=true 及精确注册的返回地址。
默认回调由可信 FRONTEND_URL 推导；API 使用不同公开域名时设置 PUBLIC_API_URL，或显式 HELIOS_REDIRECT_URI。
生产域名未确定，本次只注册本地回调；生产部署前需在 Helios 中追加真实 HTTPS 返回地址。

