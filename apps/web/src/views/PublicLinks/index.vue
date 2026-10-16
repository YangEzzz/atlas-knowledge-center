<template>
  <Dialog
    :open="Boolean(deleteCandidate)"
    @update:open="!$event && !deleting && (deleteCandidate = null)"
    ><DialogContent
      ><DialogHeader
        ><DialogTitle>删除工具入口</DialogTitle
        ><DialogDescription
          >确认删除「{{
            deleteCandidate?.name
          }}」？删除后将从工具导航中隐藏。</DialogDescription
        ></DialogHeader
      >
      <DialogFooter>
        <Button
          variant="secondary"
          :disabled="deleting"
          autofocus
          @click="deleteCandidate = null"
          >取消</Button
        ><Button
          variant="destructive"
          :disabled="deleting"
          @click="deleteCandidate && deleteLink(deleteCandidate)"
          >{{ deleting ? "删除中…" : "删除入口" }}</Button
        >
      </DialogFooter></DialogContent
    ></Dialog
  >
  <PageHeader
    title="工具导航"
    description="维护公司级通用工具入口，例如 GitLab、发布、监控、日志、API 和需求平台。"
  >
    <template #actions>
      <Button @click="openCreate">
        <Plus class="button-icon" />
        新增工具
      </Button>
    </template>
  </PageHeader>

  <Card class="filter-card">
    <CardContent>
      <div class="filter-bar">
        <Input
          v-model="filters.keyword"
          placeholder="搜索工具名称、URL、说明"
          @keyup.enter="loadLinks"
        />
        <UiSelect v-model="selectedType">
          <SelectTrigger class="ui-select-trigger" aria-label="工具类型">
            <SelectValue placeholder="全部类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="type in typeOptionsWithAll"
              :key="type.value"
              :value="selectValue(type.value)"
            >
              {{ type.label }}
            </SelectItem>
          </SelectContent>
        </UiSelect>
        <UiSelect v-model="selectedEnabled">
          <SelectTrigger class="ui-select-trigger" aria-label="启用状态">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="true">启用中</SelectItem>
            <SelectItem value="false">已停用</SelectItem>
          </SelectContent>
        </UiSelect>
      </div>
      <div class="quick-tags">
        <button
          v-for="type in typeOptions"
          :key="type.value"
          type="button"
          :class="['tag-button', { 'is-active': filters.type === type.value }]"
          @click="toggleType(type.value)"
        >
          {{ type.label }}
        </button>
      </div>
    </CardContent>
  </Card>

  <Card>
    <CardHeader class="ui-card-header">
      <div>
        <CardTitle>工具列表</CardTitle>
        <CardDescription
          >当前共
          {{ total }} 个工具入口，按排序和更新时间排列。</CardDescription
        >
      </div>
      <CardAction>
        <Button
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadLinks"
        >
          <RefreshCw
            :class="['button-icon', { 'is-spinning': visibleLoading }]"
          />
          刷新
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <div v-if="errorMessage" class="notice notice-error">
        {{ errorMessage }}
      </div>

      <EmptyState
        v-if="!loading && rows.length === 0"
        :icon="Wrench"
        title="暂无工具入口"
        description="先新增一个公司级工具入口，例如 GitLab、发布平台、监控平台或 API 平台。"
      >
        <Button @click="openCreate">
          <Plus class="button-icon" />
          新增工具
        </Button>
      </EmptyState>

      <Table v-else class="tool-table">
        <colgroup>
          <col class="tool-col-name" />
          <col class="tool-col-type" />
          <col class="tool-col-url" />
          <col class="tool-col-status" />
          <col class="tool-col-date" />
          <col class="tool-col-actions" />
        </colgroup>
        <TableHeader>
          <TableRow>
            <TableHead>名称</TableHead>
            <TableHead class="tool-center-cell">类型</TableHead>
            <TableHead>URL</TableHead>
            <TableHead class="tool-center-cell">状态</TableHead>
            <TableHead class="tool-center-cell">更新时间</TableHead>
            <TableHead class="tool-center-cell">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableSkeletonRows v-if="visibleLoading" :rows="5" :columns="6" />
          <TableRow v-for="row in rows" v-else :key="row.id">
            <TableCell class="wide-cell">
              <div class="strong-cell">{{ row.name }}</div>
              <div class="item-meta">{{ row.description || "暂无说明" }}</div>
            </TableCell>
            <TableCell class="tool-center-cell">
              <Badge variant="secondary">{{ row.typeLabel }}</Badge>
            </TableCell>
            <TableCell class="tool-url-cell">{{ row.url }}</TableCell>
            <TableCell class="tool-center-cell">
              <Badge
                variant="secondary"
                :class="[
                  'ui-badge',
                  row.isEnabled ? 'ui-badge--success' : 'ui-badge--default',
                ]"
              >
                {{ row.isEnabled ? "启用中" : "已停用" }}
              </Badge>
            </TableCell>
            <TableCell class="tool-center-cell">{{
              formatDate(row.updatedAt)
            }}</TableCell>
            <TableCell class="tool-center-cell">
              <div class="row-actions tool-row-actions">
                  <RouterLink :class="buttonVariants({ variant: 'ghost', size: 'sm' })" :to="{ path: '/my-accounts', query: { toolId: row.id } }">
                    <KeyRound class="button-icon" />我的账号
                  </RouterLink>
                <Button
                  variant="ghost"
                  size="sm"
                  :disabled="!row.isEnabled"
                  @click="openLink(row)"
                >
                  <ExternalLink class="button-icon" />
                  打开
                </Button>
                <Button variant="ghost" size="sm" @click="copyLink(row)">
                  <Copy class="button-icon" />
                  复制
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  @click="toggleLinkFavorite(row)"
                >
                  <Heart
                    :class="[
                      'button-icon',
                      { 'is-filled': isLinkFavorite(row) },
                    ]"
                  />
                  {{ isLinkFavorite(row) ? "已收藏" : "收藏" }}
                </Button>
                <Button variant="ghost" size="sm" @click="openEdit(row)">
                  <PencilLine class="button-icon" />
                  编辑
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="danger-action"
                  @click="deleteCandidate = row"
                >
                  <Trash2 class="button-icon" />
                  删除
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </CardContent>
  </Card>

  <Dialog v-model:open="dialogOpen">
    <DialogContent class="max-h-[calc(100dvh-2rem)] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ editingLink ? "编辑工具" : "新增工具" }}</DialogTitle>
        <DialogDescription
          >维护公司级通用工具入口，不绑定到具体项目。</DialogDescription
        >
      </DialogHeader>

      <form novalidate class="project-form" @submit.prevent="submitLink">
        <div class="form-grid">
          <label class="ui-field">
            <span>名称</span>
            <Input
              v-model="form.name"
              class="ui-input"
              placeholder="例如：GitLab"
            />
          </label>
          <label class="ui-field">
            <span>类型</span>
            <UiSelect v-model="form.type">
              <SelectTrigger class="ui-select-trigger">
                <SelectValue placeholder="请选择类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="type in typeOptions"
                  :key="type.value"
                  :value="type.value"
                >
                  {{ type.label }}
                </SelectItem>
              </SelectContent>
            </UiSelect>
          </label>
          <label class="ui-field">
            <span>排序</span>
            <Input
              v-model="form.sortOrder"
              class="ui-input"
              type="number"
              placeholder="数字越小越靠前"
            />
          </label>
        </div>

        <label class="ui-field">
          <span>URL</span>
          <Input
            v-model="form.url"
            class="ui-input"
            placeholder="https://gitlab.company.local"
          />
        </label>

        <label class="ui-field">
          <span>说明</span>
          <Textarea
            v-model="form.description"
            class="ui-textarea compact-textarea resize-none"
            placeholder="说明这个工具的用途或适用场景。"
          />
        </label>

        <div class="tool-enabled-row">
          <div>
            <span>启用状态</span>
            <p>关闭后列表仍可维护，但普通打开操作会被禁用。</p>
          </div>
          <Switch v-model="form.isEnabled" aria-label="启用工具入口" />
        </div>

        <div v-if="formError" class="notice notice-error">{{ formError }}</div>

        <DialogFooter>
          <Button type="button" variant="outline" @click="dialogOpen = false"
            >取消</Button
          >
          <Button type="submit" :disabled="saving">
            <Save class="button-icon" />
            {{ saving ? "保存中..." : "保存工具" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { toast } from "vue-sonner";
import {
  Copy,
  ExternalLink,
  Heart,
  PencilLine,
  Plus,
  RefreshCw,
  Save,
  Trash2,
  Wrench,
  KeyRound,
} from "lucide-vue-next";
import {
  createPublicLink,
  deletePublicLink,
  getPublicLinks,
  updatePublicLink,
  type PublicLink,
  type PublicLinkSavePayload,
} from "@/api/public-links";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import EmptyState from "@/components/ui/EmptyState.vue";
import { Input } from "@/components/ui/input";
import PageHeader from "@/components/ui/PageHeader.vue";
import {
  Select as UiSelect,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import TableSkeletonRows from "@/components/ui/TableSkeletonRows.vue";
import { Textarea } from "@/components/ui/textarea";
import { useDelayedLoading } from "@/composables/use-delayed-loading";
import { usePersonalRecords } from "@/composables/use-personal-records";

const deleteCandidate = ref<PublicLink | null>(null);
const deleting = ref(false);
const EMPTY_SELECT_VALUE = "__empty__";

const typeOptions = [
  { label: "代码", value: "code" },
  { label: "发布", value: "deploy" },
  { label: "监控", value: "monitor" },
  { label: "日志", value: "logs" },
  { label: "接口", value: "api" },
  { label: "设计", value: "design" },
  { label: "需求", value: "require" },
  { label: "测试", value: "test" },
  { label: "运维", value: "ops" },
  { label: "权限", value: "access" },
  { label: "文档", value: "document" },
  { label: "其他", value: "other" },
];

const typeOptionsWithAll = [{ label: "全部类型", value: "" }, ...typeOptions];

const filters = reactive({
  keyword: "",
  type: "",
  enabled: "all",
});

const rows = ref<PublicLink[]>([]);
const total = ref(0);
const loading = ref(false);
const visibleLoading = useDelayedLoading(loading);
const errorMessage = ref("");
const dialogOpen = ref(false);
const editingLink = ref<PublicLink | null>(null);
const saving = ref(false);
const formError = ref("");
const { addRecentVisit, isFavorite, toggleFavorite } = usePersonalRecords();

const form = reactive({
  name: "",
  type: "code",
  url: "",
  description: "",
  isEnabled: true,
  sortOrder: "0",
});

const selectedType = computed({
  get: () => selectValue(filters.type),
  set: (value) => {
    filters.type = restoreSelectValue(String(value));
    loadLinks();
  },
});

const selectedEnabled = computed({
  get: () => filters.enabled,
  set: (value) => {
    filters.enabled = String(value);
    loadLinks();
  },
});

onMounted(() => {
  loadLinks();
});

async function loadLinks() {
  loading.value = true;
  errorMessage.value = "";
  try {
    const enabled =
      filters.enabled === "all" ? undefined : filters.enabled === "true";
    const res = await getPublicLinks({
      keyword: filters.keyword.trim(),
      type: filters.type,
      enabled,
      page: 1,
      pageSize: 100,
    });
    rows.value = res.data.list;
    total.value = res.data.total;
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "获取工具导航失败";
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingLink.value = null;
  hydrateForm(null);
  dialogOpen.value = true;
}

function openEdit(row: PublicLink) {
  editingLink.value = row;
  hydrateForm(row);
  dialogOpen.value = true;
}

function hydrateForm(row: PublicLink | null) {
  form.name = row?.name || "";
  form.type = row?.type || "code";
  form.url = row?.url || "";
  form.description = row?.description || "";
  form.isEnabled = row?.isEnabled ?? true;
  form.sortOrder = String(row?.sortOrder ?? 0);
  formError.value = "";
}

async function submitLink() {
  formError.value = "";
  if (!form.name.trim()) {
    formError.value = "名称不能为空";
    return;
  }
  if (!form.type) {
    formError.value = "请选择类型";
    return;
  }
  if (!form.url.trim()) {
    formError.value = "URL 不能为空";
    return;
  }

  saving.value = true;
  try {
    const payload: PublicLinkSavePayload = {
      name: form.name.trim(),
      type: form.type,
      url: form.url.trim(),
      description: form.description.trim(),
      isEnabled: form.isEnabled,
      sortOrder: Number(form.sortOrder) || 0,
    };
    if (editingLink.value) {
      await updatePublicLink(editingLink.value.id, payload);
      toast.success(`已更新「${payload.name}」`);
    } else {
      await createPublicLink(payload);
      toast.success(`已新增「${payload.name}」`);
    }
    dialogOpen.value = false;
    await loadLinks();
  } catch (error) {
    formError.value = error instanceof Error ? error.message : "保存工具失败";
  } finally {
    saving.value = false;
  }
}

function openLink(row: PublicLink) {
  const url = normalizeUrl(row.url);
  addRecentVisit(publicLinkRecordPayload(row));
  window.open(url, "_blank", "noopener,noreferrer");
}

async function copyLink(row: PublicLink) {
  const url = normalizeUrl(row.url);
  try {
    await copyText(url);
    toast.success(`已复制「${row.name}」链接`);
  } catch {
    toast.error("复制失败，请手动复制链接");
  }
}

function toggleLinkFavorite(row: PublicLink) {
  const wasFavorite = isLinkFavorite(row);
  toggleFavorite(publicLinkRecordPayload(row));
  toast.success(
    wasFavorite ? `已取消收藏「${row.name}」` : `已收藏「${row.name}」`,
  );
}

function isLinkFavorite(row: PublicLink) {
  return isFavorite("public_link", row.id);
}

async function deleteLink(row: PublicLink) {
  if (deleting.value) return;
  deleting.value = true;
  errorMessage.value = "";
  try {
    await deletePublicLink(row.id);
    deleteCandidate.value = null;
    toast.success(`已删除「${row.name}」`);
    await loadLinks();
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "删除工具失败";
    toast.error(errorMessage.value);
  } finally {
    deleting.value = false;
  }
}

function toggleType(type: string) {
  filters.type = filters.type === type ? "" : type;
  loadLinks();
}

function selectValue(value: string) {
  return value || EMPTY_SELECT_VALUE;
}

function restoreSelectValue(value: string) {
  return value === EMPTY_SELECT_VALUE ? "" : value;
}

function normalizeUrl(value: string) {
  const trimmed = value.trim();
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  return `https://${trimmed}`;
}

function publicLinkRecordPayload(row: PublicLink) {
  return {
    kind: "public_link" as const,
    targetId: row.id,
    title: row.name,
    projectName: "-",
    description: row.description || row.url,
    path: normalizeUrl(row.url),
    badge: row.typeLabel,
  };
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      // Fall through to the textarea fallback for restricted browser contexts.
    }
  }

  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "absolute";
  textarea.style.left = "-9999px";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.focus({ preventScroll: true });
  textarea.select();
  textarea.setSelectionRange(0, value.length);

  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error("copy failed");
  }
}

function formatDate(value: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    timeZone: "Asia/Shanghai",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
</script>
