# shadcn-vue 默认样式与交互恢复

2026-10-15 用户明确要求：组件仅做必要的品牌主题适配，其余保留默认样式与交互。

移除全局组件尺寸、阴影、圆角、焦点环、Tab 和表格样式覆盖，移除弹窗宽度强制覆盖；长表单在调用处显式设置高度与滚动。Select 恢复上游默认内容宽度与 viewport；Card 恢复默认边框与阴影。品牌主题和占位文本颜色保留。账号表单的遮罩关闭重新走已有未保存确认。修正工具 Switch 的旧版 v-model:checked 用法。

验证结果：

- `pnpm --config.manage-package-manager-versions=false build`：通过（含 vue-tsc）；依赖注释及已有大包体积提示仍存在。
- `node --test tests/design-contract.test.mjs`（apps/web）：3 项通过。
- `node tests/shadcn-interactions.cjs`（apps/web，环境提供 Playwright）：6 组通过，真实 Edge 浏览器，无页面脚本异常。覆盖默认尺寸与焦点环、Tab 焦点约束、Select 嵌套 Escape、弹窗 Escape/遮罩关闭/焦点恢复、开关指针与键盘切换、Tab 方向键、账号未保存确认、390px 窄屏与减少动态效果、保存失败保留输入与重试。
- 技能 strict audit：0 findings，报告见 `docs/shadcn-audit.json`。

浏览器使用开发专用 `tests/design-preview.html` 和内存适配器，CSRF 请求另行隔离，不写入真实业务数据；未以这次检查替代线上接口或真实账号权限验证。截图和原始浏览器结果位于被忽略的 `.gocache/shadcn-verification/`。

## 2026-10-16 添加账号切换项目

切换项目曾复用 `formLoading`，使整张表单被一行加载提示替换，造成弹窗高度跳变、输入节点重新挂载与焦点丢失。现在使用独立 `projectDetailLoading`，表单始终保留，仅在资源入口选择器内展示局部加载状态，并禁止加载时保存或选择旧资源。既有请求版本号继续阻止过期响应覆盖新选择。

构建、3 项单测与 strict audit 均通过。浏览器回归增至 7 组，新增检查慢请求期间弹窗位置/高度偏差不超过 1px、表单输入节点身份和内容保持不变、项目选择焦点保留、快速切换时旧响应被忽略、加载失败后的表单保留与切换恢复。

## 项目列表标题与容器修正

移除项目网格外的 Card 及其清零内边距的旧覆盖，使用独立 section/header 排列“项目列表”、项目数量和视图/刷新操作。标题与数量保留 6px 间距，标题区与卡片网格保留 24px 间距；表格使用独立内容面，基础 Card 组件不变。

构建通过。真实浏览器在 1440px 与 390px 宽度下检查通过，卡片/表格切换正常、手机页面无横向溢出、无脚本异常。截图：`.gocache/shadcn-verification/project-library-desktop.png` 和 `project-library-mobile.png`。
