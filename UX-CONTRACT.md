# Atlas 交互约定

## 业务依据

文档图片接入遵循 `docs/图片上传接入.md`：复用现有 Button 和 wangEditor，浏览器计算 MD5 并先查询，命中跳过文件上传；Atlas 后端以会话用户和服务端项目凭据代调 Helios。上传期间展示状态及取消按钮、暂禁保存和发布；失败保留正文并允许同文件重试，离开页面取消请求。图片公开读取的说明位于上传区，项目密钥不进入前端。

| 领域 | 来源 | 界面后果 |
|---|---|---|
| 项目与文档 | docs/PRD.md | 项目资料就地维护 |
| 登录与权限 | docs/Helios登录接入.md、Helios SSO 接口 | 仅允许 Atlas 项目成员登录；服务端验证 |
| 数据生命周期 | docs/数据模型设计.md | 业务对象软删除，可恢复 |

## 认证流程

登录按钮默认打开独立窗口；弹窗被拦截时在当前页面继续，也可主动选择当前页面登录。
等待期间禁止重复发起请求，可取消；成功后获取服务端当前用户并返回原页面。
取消、权限不足、网络异常和超时显示内联可访问错误，保留重试入口。
登录完成页验证事务状态；弹窗发送同源、同事务通知，当前页面模式直接返回目标页。
应用启动以服务端 /auth/me 为准，禁止占位 token 或持久化用户信息绕过验证。
401 清理展示状态并回到登录页；退出成功才清空当前用户，退出失败显示 toast 并允许重试。

## Canonical UI Map

| Capability | Canonical owner | Source of truth | Allowed variants | Verification |
|---|---|---|---|---|
| Auth | api/auth.ts、use-auth.ts、use-helios-login.ts | docs/Helios登录接入.md | popup / redirect | 后端认证测试、前端构建、浏览器检查 |
| Button | components/ui/button | shadcn-vue 默认组件；style.css 仅提供主题 token | default / secondary / ghost / outline / destructive | 键盘与加载状态 |
| Dialog | components/ui/dialog | shadcn-vue / Reka UI | default / 调用处显式限制长表单高度 | Escape、遮罩关闭、Tab 焦点约束与关闭后焦点恢复 |
| Toast | vue-sonner provider | main.ts | error | 退出失败提示 |
| Form | ProjectFormDialog、页面表单与 components/ui/input | design-spec.md §8 | create / edit | 构建、浏览器表单检查 |
| Select/Listbox | components/ui/select | shadcn-vue / Reka UI 默认弹层与键盘行为 | authored | 方向键、Escape、嵌套弹层与窄屏检查 |
| Scrollbar | src/style.css 全局规则 | design-spec.md | standard / forced-colors | 浏览器计算样式 |
| CRUD | 业务 views、api 与 ProjectFormDialog | docs/PRD.md、design-spec.md | page / dialog | tests/design-preview.ts |
| Date | components/ui/input 的 datetime-local | docs/管理中心实施.md | native（接受操作系统日期弹层）；输入与请求明确北京时间 | 管理中心日期筛选回归 |
| Management | views/Manage、api/management、TemplateDialog | docs/管理中心实施.md | templates / trash / audit；共用默认 Tabs、Table、Select、Dialog | 后端事务测试、tests/management.cjs |
| Personal Entries | views/Entries、EntryFormDialog、api/entries、use-personal-records | docs/个人入口实施.md | private / public reference / trash | tests/personal-entries.cjs、后端 entries 事务回滚测试 |
| Account Storage | PersonalAccounts、composables/personal-accounts、utils/local-accounts | docs/本地账号保存.md（用户确认） | local / server | tests/local-accounts.cjs、tests/tool-accounts.cjs |

## 项目导航站

2026-10-15 用户修正：shadcn-vue 基础组件保留默认样式和交互，仅用主题 token 适配 Atlas 配色和字体。全局 CSS 不得替换组件焦点环、尺寸、阴影、关闭按钮和 Tab 选中样式；账号表单的点击遮罩与 Escape 统一走关闭请求，存在未保存内容时再显示业务确认。提交中继续防止关闭与重复提交。

视觉规范以 design-spec.md 为准。正式页面以真实接口数据展示；tests/design-preview.html 是仅供开发回归的组件测试入口，不随正式应用入口构建。
首页优先展示收藏项目、项目速览、公司工具和最近更新；无收藏时给出浏览项目库入口。
项目标识统一由 ProjectIdentity 生成，收藏与最近访问共用按用户隔离的响应式缓存。
全局搜索由 GlobalSearch 管理，Ctrl/Cmd+K 打开；搜索有 IME 防护、300ms 防抖和过期请求取消。结果区分项目、资源入口、文档、工具。
页面不设顶部栏；桌面全局搜索从侧栏进入，移动导航按钮位于正文起始处，弹层内可打开搜索。
项目库默认卡片，可切换表格；分页由服务端完成。项目详情概览优先展示资源和文档，负责人在辅助栏；小屏辅助内容顺序重排，不隐藏真实记录。
同一项目允许多个相同类型的资源入口，以可编辑的名称区分测试、线上等用途；名称留空使用类型名称，编辑保留已有名称、地址、备注与顺序。
不以无效按钮展示尚无接口的能力。2026-10-21 管理中心新增模板、公共回收站与审计，具体范围见 docs/管理中心实施.md；系统配置不展示占位入口。
个人账号已单独实现，通用私有笔记仍待实现。项目详情与侧栏汇总复用 PersonalAccounts；仅本人可见，密码按需读取并自动隐藏。新增/编辑后回到当前账号列表并刷新，删除进回收站，7 天后失效并自动清理；失败保留输入，离开未保存表单先确认。详情见 docs/个人账户信息流程.md。
个人账号的搜索和筛选仅保存在当前组件内存中，避免把私人用户名等搜索条件写入 URL 或浏览历史。
卡片环境和保存位置分区展示：标题旁为完整环境标签，底部明确描述保存位置，避免“服务器 · 测试”被理解为服务器类型。
账号标题使用归属名称，用户名不承担标题。用途备注选填，位于表单后部；本地和服务器均允许空 name，不自动生成或用用户名填充。已有 name 保留原值，卡片按“用途：…”展示；空用途不占空行。
账号新增默认“仅此浏览器”，既有服务器账号保持服务器存储；账号名称、用户名、密码、账号备注整条存入按当前 Atlas 用户 ID 隔离的 IndexedDB。本地不加密、不设置解锁密码；清除网站数据会丢失，浏览器、用户配置和站点来源之间不共享。此隔离是界面行为，不宣称能防止控制浏览器的人读取数据。退出登录清除显示与草稿但保留本地数据。编辑固定原保存位置，不自动迁移、复制或上传；失败保留表单，不回退服务器。
项目、工具、私人入口复用同一保存位置选择。列表混合排序分页并标记位置，可筛选本地或服务器；搜索在浏览器内匹配，不把本地用户名搜索发给服务器。服务器列表按 50 条分页读取元数据（不包含密码、账号备注）后在内存筛选，避免本地搜索泄漏；任一来源失败时明确标记列表不完整，另一来源仍可使用。本地记录使用事务校验 revision，防止多标签覆盖。删除本地账号后在本地回收站保留 7 天，到期不可恢复，下次访问本地存储时物理清理。
删除服务器入口只联动服务器账号；本地账号独立保留在“我的账号”，避免其他设备删除入口时造成唯一凭据丢失。本地保存关联名称和地址的快照，编辑账号时从所属入口重新读取；所属入口不存在时仍可在账号汇总页查看、复制或删除本地账号。本地模式仍依赖现有 Atlas 登录和入口资料，不承诺离线登录。
工具导航通过“我的账号”进入该工具的私人账号列表，复用 PersonalAccounts，侧栏汇总包含项目与工具账号并标明归属。仅工具 ID 筛选可出现在 URL，私人搜索内容继续仅存内存。新增时项目、工具、私人入口三选一，编辑时归属不可修改；所有权始终由服务端登录会话限定。工具已删除时暂停访问账号，不影响现有项目账号的密文格式。
私人入口同样复用 PersonalAccounts：账号新增时项目、工具、私人入口三选一，归属不可迁移。入口 ID 可以作为账号页面的 URL 筛选，名称、私人备注、搜索和分组筛选不写入 URL。私人入口编辑网址同步账号的打开地址，不重写密码密文。
“我的入口”承接原收藏；浏览器旧收藏在当前用户登录后幂等导入服务器，成功后才保存迁移标记，保留旧数据备份。导入失败在入口页提供重试，收藏操作失败明确反馈。公共名称与地址跟随原内容更新，个人分组、置顶和备注独立维护；原内容删除或停用时显示不可用，不开启未知地址。
删除私人入口时明确确认入口及其当前有效账号一起进入回收站，保留 7 天。恢复仅恢复被本次入口删除联动移除的账号，此前单独删除的账号继续保留在账号回收站；过期入口不能恢复。入口和账号读写、密码读取均由服务端限定当前用户，公共接口不暴露私人内容。
添加账号切换项目只加载该项目的资源入口，保留整张表单、已填内容和焦点，不用整表单加载状态替换内容。加载期间禁用资源入口选择与保存，项目选择保持可用；过期请求不得覆盖最新选择，关闭表单后忽略未完成请求。
工具删除使用应用内确认弹层，取消优先；失败保留对象并允许重试。表单禁用浏览器验证气泡，沿用现有业务验证。
