<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  ArrowUpRight,
  BookOpen,
  Clock3,
  FolderKanban,
  FolderPlus,
  Link2,
  Search,
  RefreshCw,
} from "lucide-vue-next";
import { getDocuments, type DocumentItem } from "@/api/documents";
import { getProjects, getProject, type Project } from "@/api/projects";
import { getPublicLinks, type PublicLink } from "@/api/public-links";
import { Button } from "@/components/ui/button";
import EmptyState from "@/components/ui/EmptyState.vue";
import PageHeader from "@/components/ui/PageHeader.vue";
import ProjectCard from "@/components/project/ProjectCard.vue";
import ProjectFormDialog from "@/components/project/ProjectFormDialog.vue";
import { usePersonalRecords } from "@/composables/use-personal-records";
import { openSearch } from "@/composables/use-search-dialog";
import { useDelayedLoading } from "@/composables/use-delayed-loading";
import { formatDate, isStale } from "@/utils/project-identity";
const router = useRouter();
const { favorites, recentVisits, addRecentVisit } = usePersonalRecords();
const projects = ref<Project[]>([]);
const favoriteProjects = ref<Project[]>([]);
const documents = ref<DocumentItem[]>([]);
const tools = ref<PublicLink[]>([]);
const totals = ref({ projects: 0, documents: 0, tools: 0 });
const loading = ref(false);
const loaded = ref(false);
const visibleLoading = useDelayedLoading(loading);
const error = ref("");
const createOpen = ref(false);
const keyword = ref("");
const suggestedProjects = computed(() =>
  projects.value.filter(
    (project) =>
      !favoriteProjects.value.some((favorite) => favorite.id === project.id),
  ),
);
const recentRows = computed(() => recentVisits.value.slice(0, 4));
const otherFavorites = computed(() =>
  favorites.value.filter((item) => item.kind !== "project").slice(0, 4),
);
async function load() {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    const [p, d, t] = await Promise.all([
      getProjects({ page: 1, pageSize: 6 }),
      getDocuments({ page: 1, pageSize: 5 }),
      getPublicLinks({ page: 1, pageSize: 6, enabled: true }),
    ]);
    projects.value = p.data.list;
    documents.value = d.data.list;
    tools.value = t.data.list;
    totals.value = {
      projects: p.data.total,
      documents: d.data.total,
      tools: t.data.total,
    };
    await loadFavorites();
    loaded.value = true;
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : "首页资料加载失败，请重试";
  } finally {
    loading.value = false;
  }
}
async function loadFavorites() {
  const ids = favorites.value
    .filter((item) => item.kind === "project")
    .slice(0, 6)
    .map((item) => item.targetId);
  const result = await Promise.allSettled(ids.map((id) => getProject(id)));
  favoriteProjects.value = result.flatMap((item) =>
    item.status === "fulfilled" ? [item.value.data] : [],
  );
}
function openRecord(path: string) {
  if (/^https?:\/\//i.test(path))
    window.open(path, "_blank", "noopener,noreferrer");
  else router.push(path);
}
function visitTool(tool: PublicLink) {
  addRecentVisit({
    kind: "public_link",
    targetId: tool.id,
    title: tool.name,
    description: tool.description,
    path: tool.url,
    badge: tool.typeLabel,
  });
}
onMounted(load);
</script>

<template>
  <div class="home-page">
    <PageHeader
      eyebrow="Atlas · 研发知识中心"
      title="今天，从这里出发。"
      description="找到常用项目，直达环境、仓库和团队知识。"
    >
      <template #actions
        ><Button @click="createOpen = true"
          ><FolderPlus class="button-icon" />新建项目</Button
        ></template
      >
    </PageHeader>
    <form
      class="home-search"
      role="search"
      novalidate
      @submit.prevent="openSearch(keyword)"
    >
      <Search aria-hidden="true" />
      <label for="home-search" class="sr-only">搜索项目、文档或工具</label>
      <input
        id="home-search"
        v-model="keyword"
        placeholder="搜索项目、环境地址、仓库或文档"
        autocomplete="off"
      />
      <Button type="submit" variant="secondary">搜索</Button>
    </form>
    <p class="search-hint">
      项目、资源与资料，一处查找。<button type="button" @click="openSearch()">
        打开全局搜索 <span class="keyboard-hint">Ctrl / ⌘ K</span>
      </button>
    </p>
    <div v-if="error" class="notice notice-error" role="alert">
      {{ error
      }}<Button variant="ghost" :disabled="loading" @click="load"
        ><RefreshCw class="button-icon" />重新加载</Button
      >
    </div>
    <div class="home-layout">
      <div class="home-main">
        <section aria-labelledby="favorite-projects-title" class="home-section">
          <div class="section-heading">
            <div>
              <h2 id="favorite-projects-title">我的常用项目</h2>
              <p>收藏的项目，随时回来继续。</p>
            </div>
            <RouterLink to="/my-entries">我的入口 <ArrowUpRight /></RouterLink>
          </div>
          <div
            v-if="visibleLoading"
            class="project-card-grid"
            aria-label="正在加载常用项目"
          >
            <div
              v-for="n in 3"
              :key="n"
              class="project-card project-card-skeleton"
            >
              <div class="ui-skeleton skeleton-heading" />
              <div class="ui-skeleton skeleton-line" />
              <div class="ui-skeleton skeleton-line medium" />
            </div>
          </div>
          <div v-else-if="favoriteProjects.length" class="project-card-grid">
            <ProjectCard
              v-for="project in favoriteProjects"
              :key="project.id"
              :project="project"
              @favorite="loadFavorites"
            />
          </div>
          <EmptyState
            v-else-if="!loading && !error"
            :icon="FolderKanban"
            title="把常用项目放在这里"
            description="在项目卡片上点一下收藏，下次就能直接找到。"
            ><Button variant="secondary" @click="router.push('/projects')"
              >浏览项目库 <ArrowUpRight class="button-icon" /></Button
          ></EmptyState>
        </section>
        <section
          v-if="suggestedProjects.length"
          class="home-section"
          aria-labelledby="project-overview-title"
        >
          <div class="section-heading">
            <div>
              <h2 id="project-overview-title">项目速览</h2>
              <p>发现项目，找到负责人和资源入口。</p>
            </div>
            <RouterLink to="/projects">全部项目 <ArrowUpRight /></RouterLink>
          </div>
          <div class="project-card-grid">
            <ProjectCard
              v-for="project in suggestedProjects"
              :key="project.id"
              :project="project"
              @favorite="loadFavorites"
            />
          </div>
        </section>
        <section class="home-section" aria-labelledby="tools-title">
          <div class="section-heading">
            <div>
              <h2 id="tools-title">公司工具</h2>
              <p>跨项目使用的公共系统。</p>
            </div>
            <RouterLink to="/public-links"
              >全部工具 <ArrowUpRight
            /></RouterLink>
          </div>
          <div v-if="tools.length" class="tool-shortcuts">
            <a
              v-for="tool in tools"
              :key="tool.id"
              :href="tool.url"
              target="_blank"
              rel="noopener noreferrer"
              @click="visitTool(tool)"
              ><div class="tool-shortcut-head">
                <Link2 /><strong>{{ tool.name }}</strong
                ><ArrowUpRight />
              </div>
              <p>{{ tool.description || tool.typeLabel }}</p></a
            >
          </div>
          <EmptyState
            v-else-if="!loading && !error"
            :icon="Link2"
            title="暂无公司工具"
            description="添加 GitLab、发布平台或监控等公共入口后，会显示在这里。"
            ><Button variant="secondary" @click="router.push('/public-links')"
              >前往工具导航</Button
            ></EmptyState
          >
        </section>
        <section
          v-if="otherFavorites.length"
          class="home-section"
          aria-labelledby="other-favorites-title"
        >
          <div class="section-heading">
            <h2 id="other-favorites-title">收藏的资料与工具</h2>
            <RouterLink to="/favorites">查看全部 <ArrowUpRight /></RouterLink>
          </div>
          <div class="portal-panel">
            <button
              v-for="item in otherFavorites"
              :key="item.id"
              class="portal-list-row"
              @click="openRecord(item.path)"
            >
              <BookOpen /><span
                ><strong>{{ item.title }}</strong
                ><small>{{ item.projectName || item.description }}</small></span
              ><ArrowUpRight />
            </button>
          </div>
        </section>
      </div>
      <aside class="home-aside">
        <section aria-labelledby="updates-title">
          <div class="section-heading">
            <h2 id="updates-title">最近更新</h2>
            <RouterLink to="/documents">全部 <ArrowUpRight /></RouterLink>
          </div>
          <div v-if="documents.length" class="portal-panel">
            <RouterLink
              v-for="doc in documents"
              :key="doc.id"
              :to="`/documents/${doc.id}`"
              class="update-row"
              ><span class="update-type">{{ doc.typeLabel }}</span
              ><strong>{{ doc.title }}</strong
              ><small
                ><time :datetime="doc.updatedAt">{{
                  formatDate(doc.updatedAt)
                }}</time
                ><span v-if="isStale(doc.updatedAt)" class="stale-label"
                  >可能过期</span
                ></small
              ></RouterLink
            >
          </div>
          <p v-else class="aside-empty">
            {{
              loading
                ? "正在加载文档…"
                : error
                  ? "文档暂时无法加载。"
                  : "文档更新后，会出现在这里。"
            }}
          </p>
        </section>
        <section aria-labelledby="recent-title">
          <div class="section-heading">
            <h2 id="recent-title">最近访问</h2>
            <RouterLink to="/recent-visits">全部 <ArrowUpRight /></RouterLink>
          </div>
          <div v-if="recentRows.length" class="portal-panel">
            <button
              v-for="item in recentRows"
              :key="item.id"
              class="portal-list-row"
              @click="openRecord(item.path)"
            >
              <Clock3 /><span
                ><strong>{{ item.title }}</strong
                ><small>{{ formatDate(item.recordedAt) }}</small></span
              ><ArrowUpRight />
            </button>
          </div>
          <p v-else class="aside-empty">访问项目和文档后，便可从这里继续。</p>
        </section>
      </aside>
    </div>
    <footer v-if="loaded" class="home-totals" aria-live="polite">
      <span>{{ loading ? "正在加载资料…" : `${totals.projects} 个项目` }}</span
      ><span v-if="!loading">{{ totals.documents }} 篇文档</span
      ><span v-if="!loading">{{ totals.tools }} 个公司工具</span>
    </footer>
    <ProjectFormDialog v-model:open="createOpen" @saved="load" />
  </div>
</template>
