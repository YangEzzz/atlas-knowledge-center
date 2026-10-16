<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowUpRight,
  Copy,
  FilePenLine,
  FolderKanban,
  Heart,
  PencilLine,
  Plus,
  RefreshCw,
  FileText,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { getDocuments, type DocumentItem } from "@/api/documents";
import { getProject, type Project } from "@/api/projects";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import ProjectFormDialog from "@/components/project/ProjectFormDialog.vue";
import ProjectIdentity from "@/components/project/ProjectIdentity.vue";
import PersonalAccounts from "@/components/project/PersonalAccounts.vue";
import DetailPageSkeleton from "@/components/ui/DetailPageSkeleton.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import { useDelayedLoading } from "@/composables/use-delayed-loading";
import { usePersonalRecords } from "@/composables/use-personal-records";
import { entryLabel, formatDate, isStale } from "@/utils/project-identity";
const route = useRoute();
const router = useRouter();
const loading = ref(false);
const visibleLoading = useDelayedLoading(loading);
const error = ref("");
const documentError = ref("");
const project = ref<Project | null>(null);
const documents = ref<DocumentItem[]>([]);
const documentTotal = ref(0);
const editOpen = ref(false);
const { addRecentVisit, isFavorite, toggleFavorite } = usePersonalRecords();
const favorite = computed(
  () => project.value && isFavorite("project", project.value.id),
);
const activeTab = ref("overview");
const accountsPanel = ref<InstanceType<typeof PersonalAccounts> | null>(null);
const selectedTab = computed({
  get: () => activeTab.value,
  set: async (value: string) => {
    if (
      activeTab.value === "accounts" &&
      !(await accountsPanel.value?.confirmLeave())
    )
      return;
    activeTab.value = value;
  },
});
const owners = computed(() => [
  { label: "前端负责人", names: project.value?.owners?.frontend },
  { label: "后端负责人", names: project.value?.owners?.backend },
  { label: "测试负责人", names: project.value?.owners?.tester },
]);
const docTypes = [
  { value: "intro", label: "项目介绍" },
  { value: "environment", label: "环境说明" },
  { value: "deploy", label: "部署说明" },
  { value: "api", label: "API 说明" },
  { value: "faq", label: "常见问题" },
];
const missingDocTypes = computed(() =>
  docTypes.filter(
    (type) => !documents.value.some((doc) => doc.type === type.value),
  ),
);
let generation = 0;
watch(() => route.params.id, load, { immediate: true });
async function load() {
  const id = Number(route.params.id);
  const request = ++generation;
  if (!Number.isFinite(id) || id <= 0) {
    error.value = "无效的项目地址，请返回项目库。";
    return;
  }
  loading.value = true;
  error.value = "";
  documentError.value = "";
  const previousId = project.value?.id;
  if (previousId !== id) {
    project.value = null;
    documents.value = [];
    activeTab.value = "overview";
  }
  try {
    const res = await getProject(id);
    if (request !== generation) return;
    project.value = res.data;
    addRecentVisit(record(res.data));
    try {
      const d = await getDocuments({ projectId: id, page: 1, pageSize: 100 });
      if (request !== generation) return;
      documents.value = d.data.list;
      documentTotal.value = d.data.total;
    } catch {
      if (request === generation)
        documentError.value = "项目已加载，文档读取失败，请重试。";
    }
  } catch (cause) {
    if (request === generation)
      error.value =
        cause instanceof Error ? cause.message : "项目读取失败，请重试。";
  } finally {
    if (request === generation) loading.value = false;
  }
}
function record(p: Project) {
  return {
    kind: "project" as const,
    targetId: p.id,
    title: p.name,
    description: p.description,
    path: `/projects/${p.id}`,
    badge: p.typeLabel,
  };
}
function toggle() {
  if (project.value) toggleFavorite(record(project.value));
}
function newDocument(type = "intro") {
  router.push({
    path: "/documents/editor",
    query: { projectId: String(project.value?.id), type },
  });
}
async function copy(url: string) {
  try {
    await navigator.clipboard.writeText(url);
    toast.success("链接已复制");
  } catch {
    toast.error("复制失败，请选择链接手动复制");
  }
}
async function saved(p: Project) {
  project.value = p;
  editOpen.value = false;
  await load();
}
</script>

<template>
  <div class="project-detail-page">
    <div v-if="error" class="notice notice-error" role="alert">
      {{ error
      }}<Button variant="ghost" :disabled="loading" @click="load">重试</Button>
    </div>
    <DetailPageSkeleton v-if="visibleLoading && !project" />
    <EmptyState
      v-else-if="!loading && !project"
      :icon="FolderKanban"
      title="暂时无法查看项目"
      description="项目可能已删除，或当前账号没有访问权限。"
      ><Button variant="secondary" @click="router.push('/projects')"
        >返回项目库</Button
      ></EmptyState
    >
    <template v-else-if="project">
      <header class="portal-detail-header">
        <ProjectIdentity :id="project.id" :name="project.name" />
        <div class="portal-detail-title">
          <Badge variant="secondary">{{
            project.typeLabel || project.type
          }}</Badge>
          <h1>{{ project.name }}</h1>
          <p>{{ project.description || "项目介绍待补充" }}</p>
        </div>
        <div class="page-actions">
          <Button variant="ghost" size="sm" :disabled="loading" @click="load">
            <RefreshCw
              :class="['button-icon', { 'is-spinning': visibleLoading }]"
            />刷新
          </Button>
          <Button
            variant="ghost"
            :aria-pressed="Boolean(favorite)"
            @click="toggle"
            ><Heart :class="['button-icon', { 'is-filled': favorite }]" />{{
              favorite ? "已收藏" : "收藏"
            }}</Button
          ><Button @click="editOpen = true"
            ><PencilLine class="button-icon" />编辑项目</Button
          >
        </div>
      </header>
      <Tabs v-model="selectedTab" class="project-detail-tabs"
        ><TabsList aria-label="项目资料"
          ><TabsTrigger value="overview">概览</TabsTrigger
          ><TabsTrigger value="resources"
            >资源入口
            <span class="tab-count">{{
              project.entries?.length || 0
            }}</span></TabsTrigger
          ><TabsTrigger value="documents"
            >文档
            <span class="tab-count">{{ documentTotal }}</span></TabsTrigger
          ><TabsTrigger value="accounts">我的账号</TabsTrigger></TabsList
        >
        <TabsContent v-if="activeTab === 'accounts'" value="accounts"
          ><PersonalAccounts ref="accountsPanel" :project-id="project.id"
        /></TabsContent>
        <div v-else class="portal-detail-layout">
          <div class="portal-detail-main">
            <TabsContent
              :key="activeTab"
              :value="activeTab"
              class="portal-tab-content"
            >
              <Card v-if="activeTab !== 'documents'"
                ><CardHeader class="portal-card-heading"
                  ><div>
                    <CardTitle>资源入口</CardTitle>
                    <p>常用环境、代码仓库和接口资料。</p>
                  </div>
                  <Button variant="ghost" size="sm" @click="editOpen = true"
                    >管理入口
                    <ArrowUpRight class="button-icon" /></Button></CardHeader
                ><CardContent
                  ><div
                    v-if="project.entries?.length"
                    class="portal-resource-grid"
                  >
                    <article
                      v-for="entry in project.entries"
                      :key="entry.id || entry.url"
                      class="portal-resource"
                    >
                      <div class="portal-resource-heading">
                        <h3>{{ entry.title || entryLabel(entry.type) }}</h3>
                        <span>{{ entryLabel(entry.type) }}</span>
                      </div>
                      <a
                        class="portal-resource-url"
                        :href="entry.url"
                        :title="entry.url"
                        target="_blank"
                        rel="noopener noreferrer"
                        >{{ entry.url }}</a
                      >
                      <p v-if="entry.remark" class="portal-resource-remark">
                        {{ entry.remark }}
                      </p>
                      <div class="portal-resource-actions">
                        <a
                          :href="entry.url"
                          target="_blank"
                          rel="noopener noreferrer"
                          >打开链接 <ArrowUpRight /></a
                        ><button type="button" @click="copy(entry.url)">
                          <Copy />复制链接
                        </button>
                      </div>
                    </article>
                  </div>
                  <EmptyState
                    v-else
                    :icon="FolderKanban"
                    title="资源入口待补充"
                    description="添加线上、测试、仓库和 API 文档地址，方便团队直接访问。"
                    ><Button variant="secondary" @click="editOpen = true"
                      >添加资源入口</Button
                    ></EmptyState
                  ></CardContent
                ></Card
              >

              <section
                v-if="activeTab === 'overview' || activeTab === 'documents'"
                aria-labelledby="project-documents-title"
                class="portal-documents"
              >
                <Card
                  ><CardHeader class="portal-card-heading"
                    ><div>
                      <CardTitle id="project-documents-title"
                        >项目文档</CardTitle
                      >
                      <p>从接手项目到部署上线。</p>
                    </div>
                    <Button variant="ghost" size="sm" @click="newDocument()"
                      ><Plus class="button-icon" />新建文档</Button
                    ></CardHeader
                  ><CardContent
                    ><p v-if="loading" class="aside-empty" role="status">
                      正在加载项目文档…
                    </p>
                    <div
                      v-if="documentError"
                      class="notice notice-error"
                      role="alert"
                    >
                      {{ documentError
                      }}<Button variant="ghost" @click="load">重试</Button>
                    </div>
                    <div
                      v-for="doc in activeTab === 'overview'
                        ? documents.slice(0, 5)
                        : documents"
                      :key="doc.id"
                      class="portal-document-row"
                    >
                      <FileText class="document-row-icon" />
                      <div>
                        <RouterLink :to="`/documents/${doc.id}`">{{
                          doc.title
                        }}</RouterLink>
                        <p>
                          {{ doc.typeLabel }} ·
                          <time :datetime="doc.updatedAt">{{
                            formatDate(doc.updatedAt)
                          }}</time>
                        </p>
                      </div>
                      <span v-if="isStale(doc.updatedAt)" class="stale-label"
                        >可能过期</span
                      ><Button
                        variant="ghost"
                        size="icon"
                        :aria-label="`编辑${doc.title}`"
                        @click="newDocument(doc.type)"
                        ><FilePenLine
                      /></Button>
                    </div>
                    <EmptyState
                      v-if="!loading && !documents.length && !documentError"
                      :icon="FileText"
                      title="还没有项目文档"
                      description="先补充项目介绍、环境说明或部署指南。"
                    />
                    <div
                      v-if="activeTab === 'overview' && documents.length > 5"
                      class="portal-document-more"
                    >
                      <Button variant="ghost" @click="activeTab = 'documents'"
                        >查看全部 {{ documentTotal }} 篇文档
                        <ArrowUpRight class="button-icon"
                      /></Button>
                    </div>
                    <div
                      v-if="!loading && missingDocTypes.length"
                      class="missing-document-types"
                    >
                      <span>待补充</span
                      ><Button
                        v-for="type in missingDocTypes"
                        :key="type.value"
                        size="sm"
                        variant="secondary"
                        @click="newDocument(type.value)"
                        >{{ type.label }} <Plus class="button-icon"
                      /></Button>
                    </div>
                    <RouterLink
                      v-if="documentTotal > documents.length"
                      :to="{
                        path: '/documents',
                        query: { projectId: project.id },
                      }"
                      class="text-link"
                      >前往文档库查看更多</RouterLink
                    ></CardContent
                  ></Card
                >
              </section>
              <Card v-if="activeTab === 'overview'"
                ><CardHeader><CardTitle>项目简介</CardTitle></CardHeader
                ><CardContent
                  ><p class="overview-copy">
                    {{
                      project.description ||
                      "还没有填写项目简介，可通过编辑项目补充。"
                    }}
                  </p>
                  <div v-if="project.keywords?.length" class="badge-row">
                    <Badge
                      v-for="keyword in project.keywords"
                      :key="keyword"
                      variant="secondary"
                      >{{ keyword }}</Badge
                    >
                  </div></CardContent
                ></Card
              >
            </TabsContent>
          </div>
          <aside class="portal-detail-aside">
            <Card
              ><CardHeader><CardTitle>项目负责人</CardTitle></CardHeader
              ><CardContent
                ><div
                  v-for="owner in owners"
                  :key="owner.label"
                  class="portal-owner"
                >
                  <span>{{ owner.label }}</span
                  ><strong>{{
                    owner.names?.map((item) => item.ownerName).join("、") ||
                    "待补充"
                  }}</strong>
                </div></CardContent
              ></Card
            ><Card
              ><CardHeader><CardTitle>项目资料</CardTitle></CardHeader
              ><CardContent
                ><dl class="portal-properties">
                  <div>
                    <dt>资源入口</dt>
                    <dd>{{ project.entries?.length || 0 }} 项</dd>
                  </div>
                  <div>
                    <dt>项目文档</dt>
                    <dd>{{ documentTotal }} 篇</dd>
                  </div>
                  <div>
                    <dt>最近更新</dt>
                    <dd>{{ formatDate(project.updatedAt) }}</dd>
                  </div>
                </dl></CardContent
              ></Card
            >
          </aside>
        </div>
      </Tabs>
      <ProjectFormDialog
        v-model:open="editOpen"
        :project="project"
        @saved="saved"
      />
    </template>
  </div>
</template>
