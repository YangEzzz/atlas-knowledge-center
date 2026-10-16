<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { Search, X, ArrowUpRight } from "lucide-vue-next";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/request";
import type { Project, ProjectListResponse } from "@/api/projects";
import type { DocumentListResponse } from "@/api/documents";
import type { PublicLinkListResponse } from "@/api/public-links";
import { searchKeyword, searchOpen } from "@/composables/use-search-dialog";
import { entryLabel } from "@/utils/project-identity";
const router = useRouter();
const loading = ref(false);
const error = ref("");
type Result = { key: string; title: string; description: string; path: string };
const groups = ref<{ name: string; items: Result[] }[]>([]);
const resultCount = computed(() =>
  groups.value.reduce((sum, group) => sum + group.items.length, 0),
);
let timer: ReturnType<typeof setTimeout> | undefined;
let controller: AbortController | undefined;
let generation = 0;
const composing = ref(false);
function schedule() {
  clearTimeout(timer);
  controller?.abort();
  generation++;
  loading.value = false;
  error.value = "";
  groups.value = [];
  if (!searchOpen.value || !searchKeyword.value.trim() || composing.value)
    return;
  loading.value = true;
  timer = setTimeout(runSearch, 300);
}
watch([searchKeyword, searchOpen], schedule);
function startComposition() {
  composing.value = true;
  schedule();
}
function endComposition() {
  composing.value = false;
  schedule();
}
async function runSearch() {
  clearTimeout(timer);
  if (composing.value) return;
  controller?.abort();
  const id = ++generation;
  const keyword = searchKeyword.value.trim();
  if (!keyword) {
    groups.value = [];
    loading.value = false;
    return;
  }
  controller = new AbortController();
  loading.value = true;
  error.value = "";
  try {
    const options = {
      params: { keyword, page: 1, pageSize: 12 },
      signal: controller.signal,
    };
    const [p, d, t] = await Promise.all([
      api.get<ProjectListResponse>({ url: "/projects", ...options }),
      api.get<DocumentListResponse>({ url: "/documents", ...options }),
      api.get<PublicLinkListResponse>({
        url: "/public-links",
        ...options,
        params: { ...options.params, enabled: true },
      }),
    ]);
    if (id !== generation) return;
    const projectResult = (project: Project): Result => ({
      key: `project-${project.id}`,
      title: project.name,
      description: project.description,
      path: `/projects/${project.id}`,
    });
    groups.value = [
      { name: "项目", items: p.data.list.map(projectResult) },
      {
        name: "资源入口",
        items: p.data.list.flatMap((project) =>
          (project.entries || [])
            .filter((entry) =>
              `${entry.title} ${entry.url}`
                .toLowerCase()
                .includes(keyword.toLowerCase()),
            )
            .map((entry) => ({
              key: `entry-${entry.id}`,
              title: entry.title || entryLabel(entry.type),
              description: `${project.name} · ${entry.url}`,
              path: entry.url,
            })),
        ),
      },
      {
        name: "负责人",
        items: p.data.list.flatMap((project) =>
          Object.values(project.owners || {})
            .flatMap((owners) => owners || [])
            .filter((owner) =>
              owner.ownerName.toLowerCase().includes(keyword.toLowerCase()),
            )
            .map((owner) => ({
              key: `owner-${project.id}-${owner.ownerName}`,
              title: owner.ownerName,
              description: project.name,
              path: `/projects/${project.id}`,
            })),
        ),
      },
      {
        name: "文档",
        items: d.data.list.map((doc) => ({
          key: `document-${doc.id}`,
          title: doc.title,
          description: doc.summary || doc.typeLabel,
          path: `/documents/${doc.id}`,
        })),
      },
      {
        name: "工具",
        items: t.data.list.map((tool) => ({
          key: `tool-${tool.id}`,
          title: tool.name,
          description: tool.description || tool.url,
          path: tool.url,
        })),
      },
    ].filter((group) => group.items.length);
  } catch (cause) {
    if (id === generation) error.value = "搜索未完成，请检查网络后重试。";
  } finally {
    if (id === generation) loading.value = false;
  }
}
function openResult(item: Result) {
  if (/^https?:\/\//i.test(item.path))
    window.open(item.path, "_blank", "noopener,noreferrer");
  else {
    searchOpen.value = false;
    router.push(item.path);
  }
}
onBeforeUnmount(() => {
  clearTimeout(timer);
  generation++;
  controller?.abort();
});
</script>

<template>
  <Dialog v-model:open="searchOpen">
    <DialogContent>
      <DialogHeader
        ><DialogTitle>搜索 Atlas</DialogTitle
        ><DialogDescription
          >查找项目、资源入口、文档和公司工具。</DialogDescription
        ></DialogHeader
      >
      <form
        role="search"
        novalidate
        class="search-dialog-form"
        @submit.prevent="runSearch"
      >
        <label for="global-search-input" class="sr-only">搜索关键词</label
        ><Search aria-hidden="true" /><Input
          id="global-search-input"
          v-model="searchKeyword"
          placeholder="项目名称、环境地址、仓库或文档"
          autocomplete="off"
          @compositionstart="startComposition"
          @compositionend="endComposition"
        /><Button
          v-if="searchKeyword"
          type="button"
          variant="ghost"
          size="icon"
          aria-label="清空搜索"
          @click="searchKeyword = ''"
          ><X
        /></Button>
      </form>
      <div class="search-results" :aria-busy="loading" aria-live="polite">
        <p v-if="error" class="search-message" role="alert">
          {{ error }}
          <Button variant="secondary" @click="runSearch">重试</Button>
        </p>
        <p v-else-if="loading" class="search-message">正在搜索…</p>
        <p v-else-if="!searchKeyword.trim()" class="search-message">
          输入关键词，找到需要的资料。
        </p>
        <p v-else-if="!resultCount" class="search-message">
          没有找到相关资料，试试其他关键词。
        </p>
        <section
          v-for="group in !loading && !error ? groups : []"
          :key="group.name"
        >
          <h3>{{ group.name }}</h3>
          <button
            v-for="item in group.items"
            :key="item.key"
            class="portal-list-row"
            @click="openResult(item)"
          >
            <span
              ><strong>{{ item.title }}</strong
              ><small>{{ item.description }}</small></span
            ><ArrowUpRight aria-hidden="true" />
          </button>
        </section>
      </div>
    </DialogContent>
  </Dialog>
</template>
