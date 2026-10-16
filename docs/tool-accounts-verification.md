# 工具个人账号实施与验证

2026-10-16：工具导航每行新增“我的账号”，进入 `/my-accounts?toolId=…`；复用 PersonalAccounts 和原有个人账号 API。侧栏统一汇总项目和工具账号，显示归属；新增表单可选择项目或工具，编辑时归属不可迁移。工具账号采用工具公开 URL，不接受其他工具入口。

后端新增 accounts.tool_id 和项目/工具二选一约束。项目账号保持原 AAD，加密兼容测试已通过；工具账号使用独立 AAD 前缀绑定用户、工具与记录 ID。所有查询、修改、密码读取、删除与恢复均继续按当前会话用户限定；公开工具详情不返回任何私人账号。

迁移命令 `go run ./cmd/migrate-tool-accounts` 已在当前配置的 atlas_db 执行，仅新增字段、索引和约束，不修改既有账号或密钥。本地后端已重启，健康接口返回 200。其他环境部署前须执行同一迁移。

验证：

- `ATLAS_AUTH_INTEGRATION=true go test ./internal/accounts -count=1`：6 项通过。真实数据库测试数据在事务中回滚，覆盖 HTTP 工具账号保存/筛选、跨用户隔离、项目加密兼容、工具加密隔离、改密保留、归属不可迁移、URL 归属、删除/恢复及工具删除后的账号不可访问。
- `go test ./internal/accounts ./internal/api/... ./internal/service/...`、后端 `go build`：通过。
- 前端构建及设计契约单测：通过。
- `node tests/tool-accounts.cjs`：真实 Edge 浏览器通过工具入口、筛选列表、创建、密码查看、编辑、删除、恢复、统一汇总、通用工具选择、390px 窄屏及保存失败恢复；使用内存示例数据，不写真实账号。
- `node tests/shadcn-interactions.cjs`：7 组既有项目账号/组件交互回归通过，项目切换期间布局与焦点保持稳定。
- strict audit：0 findings，见 `tool-accounts-audit.json`。

截图位于被忽略的 `.gocache/shadcn-verification/tool-account-aggregate.png` 与 `tool-account-mobile.png`。未以预览测试替代真实登录授权验证；真实权限与加密行为由上面的事务回滚集成测试验证。
