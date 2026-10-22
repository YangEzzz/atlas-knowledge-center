---
version: alpha
name: "Atlas 研发知识中心"
description: "项目导航站：天蓝色品牌、浅蓝画布、白色表面、明确的项目标识。"
colors:
  primary: "#0284c7"
  action: "#0369a1"
  action-hover: "#075985"
  background: "#f6fbff"
  foreground: "#0f172a"
  card: "#ffffff"
  muted: "#475569"
  border: "#d7e8f5"
  destructive: "#dc2626"
typography:
  sans:
    fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
  mono:
    fontFamily: 'ui-monospace, "SFMono-Regular", Consolas, "Liberation Mono", monospace'
rounded:
  sm: "4px"
  control: "8px"
  card: "16px"
  full: "999px"
spacing:
  unit: "4px"
  section-gap: "32px"
  card-padding: "24px"
  page-padding: "32px"
components:
  button: {}
  card: {}
  dialog: {}
  select: {}
---

# Atlas 设计上下文

完整规范见 [design-spec.md](design-spec.md)。用户确认项目导航站布局、balanced 密度和原天蓝色。
面向公司内部研发与产品人员，中文界面、桌面优先，首要任务是搜索项目并打开资源和文档。

## 视觉与运行时映射

颜色、字体、间距、圆角和动效的唯一运行时来源是 `apps/web/src/style.css`。
YAML 记录同一组值，不生成第二套主题；修改设计需同步本文、规范和运行时。

- primary → `--primary`，用于品牌识别；action → `--action`，通过 Tailwind `--color-action` 作用于按钮和小字号链接。
- background / foreground / card → 同名 CSS 变量；muted 文本 → `--muted-foreground`；border → `--border`。
- 占位提示使用独立 `--placeholder-foreground: #94a3b8`，字重 400；输入框、多行框、下拉框、首页搜索和编辑器统一复用。用户要求更浅且不加粗，必要说明放在可见标签和辅助文本中。
- sans / mono → `--font-sans` / `--font-mono`；移除外部字体加载依赖。
- 自有导航卡片继续使用 `--radius-control: 8px` / `--radius-card: 16px`；shadcn-vue 使用 `--radius: 8px` 推导默认圆角（md 减 2px、xl 加 4px），不映射到导航卡片圆角。
- `--space-1` 至 `--space-8` 对应 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64px。
- 自有导航卡片采用白色内容面；shadcn-vue 的 Card、Dialog、Select 等保留组件默认边框、阴影、间距、焦点环和状态样式。
- 明确浅色主题；深色方案仍待设计，不采用旧模板的默认灰阶作为 Atlas 深色主题。

## 已实施的系统行为

页面滚动由独立 #app 视口负责，保留可见滚动条及稳定占位。AppUiProvider 仅配置 Reka 的滚动条补偿，不覆盖 shadcn-vue 的尺寸、样式或焦点/关闭行为；body 浮层继续锁定背景交互，页面滚动条不再随弹窗、Select 开关消失。路由恢复应用视口的历史滚动位置，打印恢复自然文档流。

首页以常用项目、项目速览、公司工具和最近更新为主。统计降级为页尾文字。
项目标识由共享 `ProjectIdentity` 稳定生成，不暗示运行健康度。
项目卡片由 `ProjectCard` 复用；项目库支持卡片 / 表格切换及服务端分页。
项目库使用独立列表 section 承载标题、项目数量和操作，不将网格套进 Card 后再清空 Card 内边距。项目列表标题与数量间距 6px、标题区与内容间距 24px；表格使用自身的内容面。
个人账号由 `PersonalAccounts` 复用，项目内与侧栏汇总页保持一致；账号卡片延续白色内容面，密码遮蔽和显示动作明确区分。
账号卡片标题旁仅显示“线上环境 / 测试环境 / 其他环境”；保存位置放在底部元信息区，明确写“保存于服务器 / 仅存此浏览器”，不将两个维度合并在同一个标签中。
账号卡片以所属项目、工具或私人入口名称作为标题，用户名保留在明细行。旧“账号名称”改为靠后选填的“用途备注”，留空不补造名称；已保存名称作为用途备注展示，不做数据迁移。
2026-10-16 用户确认账号保存位置选择：新账号默认“仅此浏览器”，用户名、密码、账号名称和账号备注整条保存在 IndexedDB，不做本地加密或独立解锁密码；用户可主动选择“服务器同步”。编辑不切换位置，列表同时展示并标明本地/服务器来源，筛选、弹窗均复用默认 shadcn Select。入口资料仍由服务器保存；本地存储限制在表单内说明，不以密码遮蔽暗示加密。
工具导航也通过该组件管理个人账号，工具列表的“我的账号”进入按工具筛选的同一页面；汇总页明确显示项目或工具归属，新增表单以默认 Select 选择关联类型，不另造账号组件或密码交互。
2026-10-16 用户确认“我的入口”方案：我的空间依次展示我的入口、我的账号、最近访问，旧我的收藏路径重定向到我的入口。统一卡片列表收纳公司项目/文档/工具收藏与私人入口，支持名称/网址搜索、来源和分组筛选、置顶及服务端分页。入口标识复用 ProjectIdentity；弹窗、控件和确认交互沿用默认 shadcn-vue 组件，不增加全局覆盖。
私人入口仅名称、网址必填，备注加密保存；可以保存后继续添加账号，也可以在账号表单内就地新建入口并保留草稿。嵌套编辑器在父弹窗之后渲染，保持默认 Portal 遮罩顺序、焦点与 Escape 行为。
项目详情将资源与文档放在主区，负责人放在辅助区；窄屏按顺序重排。
全局搜索由共享 `GlobalSearch` 提供；手机导航使用现有 Dialog 的焦点、Escape 和滚动管理。
不设顶部栏。桌面搜索位于侧栏，手机在正文起始处提供导航按钮，导航弹层中提供搜索；Ctrl/Cmd+K 保留。
项目详情不设页首面包屑或返回导航行，直接展示项目标题；刷新与收藏、编辑放在同一操作区。
既有文档、工具、收藏、最近访问和管理页面沿用共享样式与表单组件。

## 边界与验证

用户明确要求：组件仅通过主题 token 适配品牌配色和字体；不得用全局 CSS 覆盖 shadcn-vue 默认尺寸、圆角、阴影、Tab 状态或焦点/关闭行为。长表单可在调用处显式限制高度并滚动，不扩大为全局默认。未保存确认、提交防重和失败保留输入属于业务保护，继续保留。

管理中心于 2026-10-21 实现文档模板、公共回收站与操作审计，范围见 `docs/管理中心实施.md`。采用默认 Tabs 切换、筛选表单与服务端分页表格；模板复用 wangEditor、确认和详情复用默认 Dialog。系统配置不显示占位入口；通用私有笔记仍未实现。
认证规则继续由 `UX-CONTRACT.md` 和 `docs/Helios登录接入.md` 约束。
浏览器验证使用 `apps/web/tests/design-preview.html` 挂载真实组件，示例数据只在开发测试入口生效。
实施与验证记录见 `docs/ui-implementation-2026-10-15.md`；原型文件不等于正式页面的实现或验收证据。

## 文档编辑页（2026-10-21）

编辑采用单栏写作区，直接编写标题和正文，项目关联为选填，默认“不关联项目（通用文档）”；从项目详情进入时延续已明确的归属。正文不与常驻预览争夺宽度，预览通过默认 Dialog 按需打开。标题不自动拼接，摘要选填，空白新文档可按类型加载真实启用模板；工具栏只保留常用写作功能，避免字体、字号和行高默认值干扰。继续使用主题 token 与默认 shadcn-vue 控件，不改动共享样式。
