// Development-only regression harness. Excluded from the production Vite entry.
import { createApp, h } from "vue";
import { createPinia } from "pinia";
import { createRouter, createMemoryHistory, RouterView } from "vue-router";
import request from "../src/request";
import { useAuthStore } from "../src/store/auth";
import { Toaster } from "../src/components/ui/sonner";
import AppUiProvider from "../src/components/app/AppUiProvider.vue";
import appRouter from "../src/router";
import AppLayout from "../src/layouts/AppLayout.vue";
import Home from "../src/views/Home/index.vue";
import Projects from "../src/views/Projects/index.vue";
import Detail from "../src/views/Projects/Detail.vue";
import Documents from "../src/views/Documents/index.vue";
import DocumentDetail from "../src/views/Documents/Detail.vue";
import DocumentEditor from "../src/views/Documents/Editor.vue";
import PublicLinks from "../src/views/PublicLinks/index.vue";
import Favorites from "../src/views/Favorites/index.vue";
import Recent from "../src/views/RecentVisits/index.vue";
import Manage from "../src/views/Manage/index.vue";
import { createManagementPreview } from './management-preview';
import Accounts from "../src/views/Accounts/index.vue";
import Entries from "../src/views/Entries/index.vue";
import "vue-sonner/style.css";
import "../src/style.css";
const now = "2026-10-15T10:20:00+08:00";
const projects = [
  {
    id: 101,
    name: "Nexus 多语言平台",
    type: "admin",
    typeLabel: "后台系统",
    description: "多语言词条、翻译审核与版本发布管理。",
    keywords: ["翻译", "Nexus"],
    owners: {
      frontend: [{ ownerName: "杨林" }],
      backend: [{ ownerName: "陈晨" }],
      tester: [{ ownerName: "李明" }],
    },
    entries: [
      {
        id: 1,
        type: "production",
        title: "线上环境",
        url: "https://nexus.example.com",
        remark: "仅公司内网访问\n外出时请先连接 VPN",
        sortOrder: 0,
      },
      {
        id: 2,
        type: "staging",
        title: "测试环境",
        url: "https://nexus-test.example.com",
        sortOrder: 1,
      },
      {
        id: 3,
        type: "gitlab",
        title: "GitLab 仓库",
        url: "https://gitlab.example.com/rd/nexus",
        sortOrder: 2,
      },
    ],
  },
  {
    id: 102,
    name: "Helios 认证中心",
    type: "backend",
    typeLabel: "后端服务",
    description: "统一账号、项目授权与单点登录。",
    keywords: ["登录", "Helios"],
    owners: {
      frontend: [],
      backend: [{ ownerName: "陈晨" }],
      tester: [{ ownerName: "李明" }],
    },
    entries: [
      {
        id: 4,
        type: "production",
        title: "线上环境",
        url: "https://helios.example.com",
      },
      {
        id: 5,
        type: "api_doc",
        title: "API 文档",
        url: "https://docs.example.com/helios",
      },
    ],
  },
  {
    id: 103,
    name: "云手机用户端",
    type: "user_client",
    typeLabel: "用户端",
    description: "云端设备与用户服务入口。",
    keywords: ["云手机"],
    owners: {
      frontend: [{ ownerName: "王璐" }],
      backend: [{ ownerName: "李明" }],
      tester: [],
    },
    entries: [
      {
        id: 6,
        type: "production",
        title: "线上环境",
        url: "https://cloud.example.com",
      },
    ],
  },
].map((project) => ({
  ...project,
  sortOrder: 0,
  createdAt: now,
  updatedAt: now,
}));
const documents = [
  {
    id: 201,
    projectId: 101,
    type: "deploy",
    typeLabel: "部署说明",
    title: "生产环境部署指南",
    summary: "构建、发布和回滚说明。",
    content:
      "# 生产环境部署指南\n\n## 环境要求\n\n使用发布平台构建并部署。\n\n## 发布步骤\n\n1. 确认配置与版本。\n2. 创建发布任务。\n3. 检查服务状态。",
  },
  {
    id: 202,
    projectId: 102,
    type: "api",
    typeLabel: "API 说明",
    title: "Helios 单点登录接入规范",
    summary: "登录流程、项目角色与回调地址。",
    content: "# 登录接入\n\n项目登录需要有效的项目成员授权。",
  },
  {
    id: 203,
    projectId: 101,
    type: "environment",
    typeLabel: "环境说明",
    title: "测试环境与账号申请",
    summary: "测试环境地址及账号申请方式。",
    content: "# 测试环境\n\n联系项目负责人申请账号。",
  },
  { id: 204, projectId: null, type: "general", typeLabel: "通用文档", title: "团队协作指南", summary: "通用知识，无需关联项目。", content: "<p>团队共用的指南。</p>" },
].map((doc) => ({
  ...doc,
  status: "published",
  statusLabel: "已发布",
  currentVersionNo: 2,
  createdAt: now,
  updatedAt: now,
}));
const tools = [
  {
    id: 301,
    name: "GitLab",
    type: "gitlab",
    typeLabel: "代码仓库",
    url: "https://gitlab.example.com",
    description: "代码托管与合并请求",
  },
  {
    id: 302,
    name: "发布平台",
    type: "deploy",
    typeLabel: "发布部署",
    url: "https://deploy.example.com",
    description: "应用构建、部署与回滚",
  },
  {
    id: 303,
    name: "监控平台",
    type: "monitor",
    typeLabel: "监控告警",
    url: "https://monitor.example.com",
    description: "服务状态与日志查询",
  },
].map((item) => ({
  ...item,
  isEnabled: true,
  sortOrder: 0,
  createdAt: now,
  updatedAt: now,
}));
const scenario = new URLSearchParams(location.search).get("scenario");
const managementPreview = createManagementPreview(scenario, now);
const originalPreviewFetch = window.fetch.bind(window);
window.fetch = (input, init) => String(input).endsWith('/auth/csrf')
  ? Promise.resolve(Response.json({ code: 200, data: { csrf_token: 'preview-only' } }))
  : originalPreviewFetch(input, init);
const previewImages = new Map<string, object>();
let editorFailures = 0;
let imageFailures = scenario === 'image-error' ? 1 : 0;
// Disposable examples only. Local-account tests use isolated, disposable browser contexts.
let accountId = 401;
let accountRows: any[] = [
  {
    id: 401,
    projectId: 101,
    projectName: projects[0]!.name,
    name: "测试后台账号",
    username: "atlas-demo",
    environment: "staging",
    resourceUrl: projects[0]!.entries[0]!.url,
    hasPassword: true,
    revision: 1,
    updatedAt: now,
  },
];
let entryId=700;
let entryRows:any[]=[];
const accountSecrets = new Map<number, { password: string; note: string }>([
  [401, { password: "demo-only-password", note: "示例私人备注" }],
]);
const delay = scenario === "slow" ? 3000 : 160;
const list = (items: any[], config: any) => {
  const params = config.params || {};
  let filtered = items.filter(
    (item) =>
      !params.keyword ||
      JSON.stringify(item)
        .toLowerCase()
        .includes(String(params.keyword).toLowerCase()),
  );
  if (params.projectId)
    filtered = filtered.filter(
      (item) => item.projectId === Number(params.projectId),
    );
  if (params.toolId) filtered = filtered.filter(item => item.toolId === Number(params.toolId));
  if(params.entryId)filtered=filtered.filter(item=>item.entryId===Number(params.entryId));
  if (params.type)
    filtered = filtered.filter((item) => item.type === params.type);
  if (config.url?.includes("/trash") || scenario === "empty") filtered = [];
  const size = params.pageSize || 20;
  return {
    list: filtered.slice(
      ((params.page || 1) - 1) * size,
      (params.page || 1) * size,
    ),
    total: filtered.length,
    page: params.page || 1,
    pageSize: size,
  };
};
const previewRequests: { url?: string; method?: string; params?: unknown; data?: unknown }[] = [];
Object.assign(window, { __atlasPreviewRequests: previewRequests });
request.defaults.adapter = async (config) => {
  previewRequests.push({ url: config.url, method: config.method, params: config.params, data: config.data });
  const url = config.url || "";
  const switchingProject = scenario === "project-switch" && /^\/projects\/\d+$/.test(url);
  const requestDelay = url.startsWith('/images/') && scenario === 'image-slow' ? 5000 : switchingProject ? (url.endsWith('/101') ? 1500 : 500) : delay;
  await new Promise((resolve) => setTimeout(resolve, requestDelay));
  if (scenario === "project-list-error" && url === "/projects") throw new Error("项目列表暂时不可用");
  if (scenario === "error") throw new Error("暂时无法读取资料，请稍后重试。");
  if (switchingProject && url.endsWith('/103')) throw new Error('项目暂时无法加载');
  let responseCode = 200;
  let responseMessage = "ok";
  let data: unknown = {};
  if (url.startsWith('/manage/') || url.startsWith('/document-templates/active/')) data = await managementPreview(config);
  else if (url.startsWith('/images/')) {
    if (imageFailures > 0) { imageFailures--; throw new Error('图片服务暂时不可用，请重新选择图片重试'); }
    if (url === '/images/check') {
      const { md5 } = JSON.parse(String(config.data));
      data = previewImages.has(md5) ? { ...previewImages.get(md5), exists: true, reused: true } : { exists: false, md5 };
    } else {
      const md5 = (config.data as FormData).get('md5') as string;
      data = { md5, url: `${location.origin}/tests/image-upload.png`, reused: false };
      previewImages.set(md5, data as object);
    }
  }
  else if(url.startsWith("/my/entries")) {
    const method=config.method||"get";const input=typeof config.data==="string"?JSON.parse(config.data):config.data||{};
    const id=Number(url.split('/')[3]);const row=entryRows.find(item=>item.id===id);
    const hydrate=(entry:any)=>{
      const target=entry.source==='project'?projects.find(p=>p.id===entry.targetId):entry.source==='tool'?tools.find(p=>p.id===entry.targetId):documents.find(p=>p.id===entry.targetId);
      return {...entry,name:entry.source==='private'?entry.name:target?('name' in target?target.name:target.title):entry.name,url:entry.source==='project'?`/projects/${entry.targetId}`:entry.source==='document'?`/documents/${entry.targetId}`:entry.source==='tool'&&target&&'url' in target?target.url:entry.url,available:!entry.deletedAt&&(entry.source==='private'||!!target),accountCount:accountRows.filter(a=>a.entryId===entry.id&&!a.deletedAt).length};
    };
    if(scenario==='entry-save-error'&&(method==='post'||method==='put')&&!url.endsWith('/import'))throw new Error('保存失败');
    if(url.endsWith('/import')) {
      for(const item of input.items||[])if(!entryRows.some(e=>e.source===item.source&&e.targetId===item.targetId))entryRows.push({id:++entryId,source:item.source,targetId:item.targetId,kind:item.source,name:item.name,url:'',groupName:'',pinned:false,note:'',revision:1,updatedAt:now});
    } else if(url.endsWith('/restore')){if(row){delete row.deletedAt;for(const a of accountRows.filter(a=>a.entryId===id&&a.entryCascade)){delete a.deletedAt;a.entryCascade=false}}}
    else if(method==='delete'){if(row?.source==='private'){row.deletedAt=now;for(const a of accountRows.filter(a=>a.entryId===id&&!a.deletedAt)){a.deletedAt=now;a.entryCascade=true}}else entryRows=entryRows.filter(e=>e.id!==id)}
    else if(method==='post'||method==='put'){const updated={...input,id:row?.id||++entryId,revision:(row?.revision||0)+1,updatedAt:now};if(row)Object.assign(row,updated);else entryRows.push(updated);data=hydrate(updated)}
    else if(id){data=hydrate(row)}
    else {let filtered=entryRows.filter(e=>!!e.deletedAt===(config.params?.trash===true));if(config.params?.source)filtered=filtered.filter(e=>config.params.source==='public'?e.source!=='private':e.source===config.params.source);if(config.params?.group)filtered=filtered.filter(e=>e.groupName===config.params.group);filtered.sort((a,b)=>Number(b.pinned)-Number(a.pinned)||b.id-a.id);data=list(filtered.map(hydrate),config)}
  } else if (url.startsWith("/my/accounts")) {
    const id = Number(url.split("/")[3]);
    const row = accountRows.find((item) => item.id === id);
    const method = config.method || "get";
    const input =
      typeof config.data === "string"
        ? JSON.parse(config.data)
        : config.data || {};
    if (scenario === "save-error" && method !== "get")
      throw new Error("保存失败，请稍后重试。");
    if (url.endsWith("/password"))
      data = { password: accountSecrets.get(id)?.password || "" };
    else if (url.endsWith("/restore")) {
      if (row) delete row.deletedAt;
      data = {};
    } else if (method === "delete") {
      if (row) row.deletedAt = now;
      data = {};
    } else if (method === "post" || method === "put") {
      const nextId = row?.id || ++accountId;
      const old = accountSecrets.get(nextId) || { password: "", note: "" };
      accountSecrets.set(nextId, {
        password:
          input.passwordAction === "replace"
            ? input.password
            : input.passwordAction === "clear"
              ? ""
              : old.password,
        note: input.note,
      });
      const updated = {
        id: nextId,
        projectId: input.projectId,
        toolId: input.toolId || 0,
        entryId: input.entryId || 0,
        entryName: entryRows.find(e=>e.id===input.entryId)?.name,
        projectName: projects.find((p) => p.id === input.projectId)?.name,
        toolName: tools.find(tool => tool.id === input.toolId)?.name,
        name: input.name,
        username: input.username,
        environment: input.environment,
        resourceUrl: input.resourceUrl,
        hasPassword: Boolean(accountSecrets.get(nextId)?.password),
        revision: (row?.revision || 0) + 1,
        updatedAt: now,
      };
      if (row) Object.assign(row, updated);
      else accountRows.push(updated);
      data = updated;
    } else if (id) {
      data = { account: row, note: accountSecrets.get(id)?.note || "" };
    } else {
      let items = accountRows.filter(
        (item) => Boolean(item.deletedAt) === (config.params?.trash === true),
      );
      if (config.params?.environment)
        items = items.filter(
          (item) => item.environment === config.params.environment,
        );
      data = list(items, config);
    }
  } else if (url === "/projects" || /^\/projects\/\d+$/.test(url)) {
    const method = config.method || "get";
    if (method === "post" || method === "put") {
      if (scenario === "project-save-error") throw new Error("保存项目失败，请重试");
      const input = typeof config.data === "string" ? JSON.parse(config.data) : config.data;
      const existing = projects.find((p) => p.id === Number(url.split("/").pop()));
      const saved = {
        ...projects[0], ...existing, ...input,
        id: existing?.id || 1000 + projects.length,
        entries: input.entries.map((entry: object, index: number) => ({ ...entry, id: 100 + index, sortOrder: index })),
      };
      if (existing) Object.assign(existing, saved);
      else projects.push(saved);
      data = saved;
    } else data = url === "/projects" ? list(projects, config) : projects.find((p) => p.id === Number(url.split("/").pop()));
  }
  else if (url === "/documents" && config.method === "post") {
    if (scenario === "document-save-error" && editorFailures++ === 0) throw new Error("保存文档失败，请重试");
    if (scenario === "document-save-slow") await new Promise(resolve => setTimeout(resolve, 1200));
    const input = typeof config.data === "string" ? JSON.parse(config.data) : config.data;
    const existing = documents.find(d => d.id === input.id);
    if (input.projectId && documents.some(d => d.id !== input.id && d.projectId === input.projectId && d.type === input.type)) throw new Error("该项目已有同类型文档，请从文档库打开后编辑");
    const saved = { ...input, id: existing?.id || Math.max(...documents.map(d => d.id), 998) + 1, typeLabel: input.type, statusLabel: input.status === "published" ? "已发布" : "草稿", currentVersionNo: (existing?.currentVersionNo || 0) + 1, createdAt: now, updatedAt: now };
    if (existing) Object.assign(existing, saved); else documents.push(saved);
    data = saved;
  }
  else if (url === "/documents" || url === "/documents/trash")
    data = list(documents, config);
  else if (/^\/documents\/\d+$/.test(url)) {
    if (scenario === "document-load-error" && editorFailures++ === 0) throw new Error("读取文档失败，请重试");
    data = documents.find((d) => d.id === Number(url.split("/").pop()));
  }
  else if (url.startsWith("/documents/project/")) {
    if (scenario === "document-load-error" && editorFailures++ === 0) throw new Error("读取文档失败，请重试");
    if (scenario === "document-load-slow") await new Promise(resolve => setTimeout(resolve, 1200));
    data =
      documents.find(
        (d) =>
          d.projectId === Number(url.split("/")[3]) &&
          d.type === url.split("/").pop(),
      ) || null;
    if (!data) { responseCode = 404; responseMessage = "文档不存在"; }
    if (scenario === "document-business-error") { responseCode = 500; responseMessage = "文档读取暂时不可用"; data = null; }
  }
  else if (url.includes("/versions")) data = { list: [] };
  else if (url === "/public-links") data = list(tools, config);
  else if (/^\/public-links\/\d+$/.test(url)) data = tools.find(tool => tool.id === Number(url.split('/').pop()));
  return {
    status: 200,
    statusText: "OK",
    headers: {},
    config,
    data: { code: responseCode, message: responseMessage, data },
  };
};
const pinia = createPinia();
const router = new URLSearchParams(location.search).has("history") ? appRouter : createRouter({
  history: createMemoryHistory(),
  routes: [
    {
      path: "/",
      component: AppLayout,
      children: [
        { path: "my-accounts", component: Accounts },
        {path:"my-entries",component:Entries},
        { path: "", component: Home },
        { path: "projects", name: "Projects", component: Projects },
        { path: "projects/:id", name: "ProjectDetail", component: Detail },
        { path: "documents", component: Documents },
        { path: "documents/editor", component: DocumentEditor },
        {
          path: "documents/:id",
          name: "DocumentDetail",
          component: DocumentDetail,
        },
        { path: "public-links", component: PublicLinks },
        { path: "favorites",redirect:"/my-entries" },
        { path: "recent-visits", component: Recent },
        { path: "manage", component: Manage },
      ],
    },
  ],
});
const app = createApp({
  render: () =>
    h(AppUiProvider, null, () => [
      h(RouterView),
      h(Toaster),
      h(
        "div",
        {
          style:
            "position:fixed;bottom:0;left:0;right:0;padding:4px 12px;background:#0f172a;color:white;font:12px system-ui;text-align:center;z-index:100",
        },
        "页面回归预览 · 示例数据 · 不写入数据库",
      ),
    ]),
});
app.use(pinia);
useAuthStore().setUser({
  id: "ui-preview",
  displayName: "杨林",
  roles: ["admin"],
});
useAuthStore().initialized = true;
localStorage.removeItem("ty-notebook:personal-records:ui-preview:entries-imported");
localStorage.setItem(
  "ty-notebook:personal-records:ui-preview",
  JSON.stringify({
    favorites:
      scenario === "empty"
        ? []
        : projects.map((p) => ({
            id: `project:${p.id}`,
            kind: "project",
            targetId: p.id,
            title: p.name,
            path: `/projects/${p.id}`,
            recordedAt: now,
          })),
    recentVisits: [],
  }),
);
app.use(router);
router
  .push(new URLSearchParams(location.search).get("route") || "/")
  .then(() => router.isReady())
  .then(() => app.mount("#app"));
