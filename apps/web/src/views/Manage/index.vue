<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Plus, RefreshCw, X } from "lucide-vue-next";
import {
  getTemplates,
  saveTemplate,
  deleteTemplate,
  getTrash,
  restoreTrash,
  getAudit,
  getAuditDetail,
  documentTypes,
  objectTypes,
  auditActions,
  labelFor,
  managementTime,
  type DocumentTemplate,
  type TrashItem,
  type AuditLog,
  type ManagementQuery,
} from "@/api/management";
import PageHeader from "@/components/ui/PageHeader.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import TableSkeletonRows from "@/components/ui/TableSkeletonRows.vue";
import { useDelayedLoading } from "@/composables/use-delayed-loading";
import { useAuthStore } from "@/store/auth";
import { canManage } from "@/utils/management-access";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Select as UiSelect,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import {
  Table,
  TableHeader,
  TableHead,
  TableRow,
  TableBody,
  TableCell,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import TemplateDialog from "./TemplateDialog.vue";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const permitted = computed(() => canManage(auth.user?.roles));
const tab = computed(() =>
  ["templates", "trash", "audit"].includes(String(route.query.tab))
    ? String(route.query.tab)
    : "templates",
);
const titles: Record<string, string> = {
  templates: "文档模板",
  trash: "回收站",
  audit: "操作审计",
};
const filters = reactive({
  keyword: "",
  type: "all",
  enabled: "all",
  actor: "",
  action: "all",
  from: "",
  to: "",
});
const templates = ref<DocumentTemplate[]>([]);
const trash = ref<TrashItem[]>([]);
const audits = ref<AuditLog[]>([]);
const loading = ref(false);
const visibleLoading = useDelayedLoading(loading);
const error = ref("");
const total = ref(0);
const page = computed(() => {
  const value = Number(route.query.page);
  return Number.isFinite(value)
    ? Math.min(1000000, Math.max(1, Math.floor(value)))
    : 1;
});
const pageSize = 20;
const editing = ref<DocumentTemplate>();
const editorOpen = ref(false);
const templateDialog = ref<InstanceType<typeof TemplateDialog>>();
const composing = ref(false);
const pending = ref<
  | { kind: "delete"; row: DocumentTemplate }
  | { kind: "toggle"; row: DocumentTemplate }
  | { kind: "restore"; row: TrashItem }
>();
const busy = ref(false);
const actionError = ref("");
const detail = ref<AuditLog>();
const detailOpen = ref(false);
const detailLoading = ref(false);
const detailError = ref("");
let detailRequest = 0;
let controller: AbortController | undefined;
let version = 0;
const count = computed(() =>
  tab.value === "templates"
    ? templates.value.length
    : tab.value === "trash"
      ? trash.value.length
      : audits.value.length,
);
const filterTypes = computed(() =>
  tab.value === "templates"
    ? documentTypes
    : objectTypes.filter(
        (o) => tab.value === "audit" || o.value !== "template",
      ),
);
const searchInput = ref<InstanceType<typeof Input>>();

function filterQuery(): ManagementQuery {
  return {
    page: page.value,
    pageSize,
    keyword: String(route.query.keyword || ""),
    type: String(route.query.type || ""),
    enabled:
      tab.value === "templates" ? String(route.query.enabled || "") : undefined,
    actor: tab.value === "audit" ? String(route.query.actor || "") : undefined,
    action:
      tab.value === "audit" ? String(route.query.action || "") : undefined,
    from:
      tab.value === "audit" && route.query.from
        ? `${route.query.from}:00+08:00`
        : undefined,
    to:
      tab.value === "audit" && route.query.to
        ? `${route.query.to}:59+08:00`
        : undefined,
  };
}
function syncFilters() {
  filters.keyword = String(route.query.keyword || "");
  filters.type = String(route.query.type || "all");
  filters.enabled = String(route.query.enabled || "all");
  filters.actor = String(route.query.actor || "");
  filters.action = String(route.query.action || "all");
  filters.from = String(route.query.from || "");
  filters.to = String(route.query.to || "");
}
async function load() {
  controller?.abort();
  controller = new AbortController();
  const request = ++version;
  if (!permitted.value) return;
  loading.value = true;
  error.value = "";
  try {
    const params = filterQuery();
    const signal = controller.signal;
    if (tab.value === "templates") {
      const res = await getTemplates(params, signal);
      if (request !== version) return;
      templates.value = res.data.list;
      total.value = res.data.total;
    } else if (tab.value === "trash") {
      const res = await getTrash(params, signal);
      if (request !== version) return;
      trash.value = res.data.list;
      total.value = res.data.total;
    } else {
      const res = await getAudit(params, signal);
      if (request !== version) return;
      audits.value = res.data.list;
      total.value = res.data.total;
    }
    if (page.value > 1 && !count.value)
      await router.replace({
        query: {
          ...route.query,
          page: String(Math.max(1, Math.ceil(total.value / pageSize))),
        },
      });
  } catch (err) {
    if (request === version && !controller?.signal.aborted)
      error.value = err instanceof Error ? err.message : "加载失败，请重试";
  } finally {
    if (request === version) loading.value = false;
  }
}
watch(
  () => route.query,
  () => {
    syncFilters();
    void load();
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  controller?.abort();
  version++;
  detailRequest++;
});
onBeforeRouteLeave(() =>
  busy.value
    ? false
    : editorOpen.value
      ? templateDialog.value?.confirmLeave()
      : true,
);
function changeTab(value: string | number) {
  if (!editorOpen.value && !busy.value)
    void router.push({ query: { tab: String(value) } });
}
function applyFilters(event?: Event) {
  if (composing.value || (event instanceof KeyboardEvent && event.isComposing))
    return;
  if (filters.from && filters.to && filters.from > filters.to) {
    error.value = "开始时间不能晚于结束时间";
    return;
  }
  const query: Record<string, string> = { tab: tab.value };
  for (const [key, value] of Object.entries(filters))
    if (
      value &&
      value !== "all" &&
      (tab.value === "audit" ||
        !["actor", "action", "from", "to"].includes(key)) &&
      (tab.value === "templates" || key !== "enabled")
    )
      query[key] = value.trim();
  if (JSON.stringify(route.query) === JSON.stringify(query)) void load();
  else void router.push({ query });
}
function clearSearch() {
  filters.keyword = "";
  applyFilters();
  (searchInput.value?.$el as HTMLInputElement)?.focus();
}
function reset() {
  if (Object.keys(route.query).every((k) => k === "tab")) {
    syncFilters();
    void load();
  } else void router.push({ query: { tab: tab.value } });
}
function newTemplate() {
  editing.value = undefined;
  editorOpen.value = true;
}
function edit(row: DocumentTemplate) {
  editing.value = row;
  editorOpen.value = true;
}
function confirm(kind: "delete" | "toggle", row: DocumentTemplate) {
  pending.value = { kind, row };
  actionError.value = "";
}
function confirmRestore(row: TrashItem) {
  pending.value = { kind: "restore", row };
  actionError.value = "";
}
async function perform() {
  if (!pending.value || busy.value) return;
  busy.value = true;
  actionError.value = "";
  try {
    const item = pending.value;
    if (item.kind === "delete") await deleteTemplate(item.row);
    else if (item.kind === "toggle")
      await saveTemplate(item.row.id, {
        ...item.row,
        isEnabled: !item.row.isEnabled,
      });
    else await restoreTrash(item.row);
    toast.success(
      item.kind === "restore"
        ? "已恢复"
        : item.kind === "delete"
          ? "模板已删除"
          : "模板状态已更新",
    );
    pending.value = undefined;
    busy.value = false;
    await load();
  } catch (err) {
    actionError.value = err instanceof Error ? err.message : "操作失败，请重试";
  } finally {
    busy.value = false;
  }
}
async function openDetail(row: AuditLog) {
  detail.value = undefined;
  detailOpen.value = true;
  detailLoading.value = true;
  detailError.value = "";
  const request = ++detailRequest;
  try {
    const res = await getAuditDetail(row.id);
    if (request === detailRequest) detail.value = res.data;
  } catch (err) {
    if (request === detailRequest)
      detailError.value = err instanceof Error ? err.message : "读取详情失败";
  } finally {
    if (request === detailRequest) detailLoading.value = false;
  }
}
const actionLabel = computed(() =>
  pending.value?.kind === "restore"
    ? "恢复"
    : pending.value?.kind === "delete"
      ? "删除模板"
      : pending.value?.row.isEnabled
        ? "停用模板"
        : "启用模板",
);
</script>

<template>
  <PageHeader
    title="管理中心"
    description="维护文档模板，恢复误删资料，追溯公共资料变更。"
  >
    <template #actions>
      <Button v-if="permitted && tab === 'templates'" @click="newTemplate">
        <Plus class="button-icon" />
        新增模板
      </Button>
    </template>
  </PageHeader>
  <EmptyState
    v-if="!permitted"
    title="需要管理员权限"
    description="请联系 Atlas 管理员处理模板、恢复与审计。"
  />
  <template v-else>
    <Tabs :model-value="tab" @update:model-value="changeTab">
      <TabsList aria-label="管理模块">
        <TabsTrigger value="templates">文档模板</TabsTrigger>
        <TabsTrigger value="trash">回收站</TabsTrigger>
        <TabsTrigger value="audit">操作审计</TabsTrigger>
      </TabsList>
      <TabsContent :value="tab">
        <Card class="my-5">
          <CardContent>
            <form
              novalidate
              class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4"
              @submit.prevent="applyFilters"
              @compositionstart="composing = true"
              @compositionend="composing = false"
              @keydown.enter="
                (event) => {
                  if (event.isComposing || composing) event.preventDefault();
                }
              "
            >
              <div class="ui-field">
                <label for="manage-search">
                  {{ tab === "audit" ? "对象 / 摘要 / 对象 ID" : "名称" }}
                </label>
                <div class="relative">
                  <Input
                    id="manage-search"
                    ref="searchInput"
                    v-model="filters.keyword"
                    class="pr-9"
                    placeholder="输入关键词后查询"
                  />
                  <Button
                    v-if="filters.keyword"
                    type="button"
                    variant="ghost"
                    size="icon"
                    class="absolute right-0 top-0 size-9"
                    aria-label="清除关键词"
                    @click="clearSearch"
                  >
                    <X class="size-4" />
                  </Button>
                </div>
              </div>
              <div class="ui-field">
                <label for="manage-type">
                  {{ tab === "templates" ? "文档类型" : "对象类型" }}
                </label>
                <UiSelect v-model="filters.type">
                  <SelectTrigger id="manage-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部类型</SelectItem>
                    <SelectItem
                      v-for="type in filterTypes"
                      :key="type.value"
                      :value="type.value"
                    >
                      {{ type.label }}
                    </SelectItem>
                  </SelectContent>
                </UiSelect>
              </div>
              <div v-if="tab === 'templates'" class="ui-field">
                <label for="manage-enabled">启用状态</label>
                <UiSelect v-model="filters.enabled">
                  <SelectTrigger id="manage-enabled">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部状态</SelectItem>
                    <SelectItem value="true">已启用</SelectItem>
                    <SelectItem value="false">已停用</SelectItem>
                  </SelectContent>
                </UiSelect>
              </div>
              <template v-if="tab === 'audit'">
                <div class="ui-field">
                  <label for="manage-actor">操作人</label>
                  <div class="relative">
                    <Input
                      id="manage-actor"
                      v-model="filters.actor"
                      class="pr-9"
                      placeholder="姓名或用户 ID"
                    />
                    <Button
                      v-if="filters.actor"
                      type="button"
                      variant="ghost"
                      size="icon"
                      class="absolute right-0 top-0 size-9"
                      aria-label="清除操作人"
                      @click="
                        filters.actor = '';
                        applyFilters();
                      "
                    >
                      <X class="size-4" />
                    </Button>
                  </div>
                </div>
                <div class="ui-field">
                  <label for="manage-action">操作类型</label>
                  <UiSelect v-model="filters.action">
                    <SelectTrigger id="manage-action">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部操作</SelectItem>
                      <SelectItem
                        v-for="action in auditActions"
                        :key="action.value"
                        :value="action.value"
                      >
                        {{ action.label }}
                      </SelectItem>
                    </SelectContent>
                  </UiSelect>
                </div>
                <div class="ui-field">
                  <label for="manage-from">开始时间（北京时间）</label>
                  <Input
                    id="manage-from"
                    v-model="filters.from"
                    type="datetime-local"
                  />
                </div>
                <div class="ui-field">
                  <label for="manage-to">结束时间（北京时间）</label>
                  <Input
                    id="manage-to"
                    v-model="filters.to"
                    type="datetime-local"
                  />
                </div>
              </template>
              <div class="flex flex-wrap items-end gap-2">
                <Button type="submit" :disabled="loading">查询</Button>
                <Button type="button" variant="outline" @click="reset">
                  重置
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
        <Card>
          <CardContent>
            <div class="mb-4 flex items-center justify-between gap-3">
              <div>
                <h2 class="font-semibold">{{ titles[tab] }}</h2>
                <p class="mt-1 text-sm text-muted-foreground">
                  {{
                    tab === "templates"
                      ? "同类型最多启用一个模板；仅填入新文档的空正文。"
                      : tab === "trash"
                        ? "恢复保留原有内容和关联；所属项目已删除时，请先恢复项目。"
                        : "记录启用审计后的公共资料变更，不包含个人账号或私人入口。"
                  }}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                :disabled="loading"
                @click="load"
              >
                <RefreshCw
                  :class="['button-icon', { 'is-spinning': visibleLoading }]"
                />
                刷新
              </Button>
            </div>
            <div v-if="error" role="alert" class="notice notice-error">
              {{ error }}
              <Button variant="outline" size="sm" @click="load">重试</Button>
            </div>
            <div class="min-h-64" :aria-busy="loading">
              <Table v-if="loading || (!error && count)" class="min-w-[720px]">
                <TableHeader>
                  <TableRow>
                    <template v-if="tab === 'templates'">
                      <TableHead>模板名称</TableHead>
                      <TableHead>文档类型</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>更新时间</TableHead>
                      <TableHead class="text-right">操作</TableHead>
                    </template>
                    <template v-else-if="tab === 'trash'">
                      <TableHead>名称</TableHead>
                      <TableHead>类型</TableHead>
                      <TableHead>所属项目</TableHead>
                      <TableHead>删除人 / 时间</TableHead>
                      <TableHead class="text-right">操作</TableHead>
                    </template>
                    <template v-else>
                      <TableHead>操作时间 / 人员</TableHead>
                      <TableHead>操作</TableHead>
                      <TableHead>对象</TableHead>
                      <TableHead>变更摘要</TableHead>
                      <TableHead class="text-right">操作</TableHead>
                    </template>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableSkeletonRows
                    v-if="visibleLoading"
                    :rows="5"
                    :columns="5"
                  />
                  <template v-else-if="!loading">
                    <template v-if="tab === 'templates'">
                      <TableRow v-for="row in templates" :key="row.id">
                        <TableCell
                          class="max-w-80 whitespace-normal font-medium"
                        >
                          {{ row.name }}
                        </TableCell>
                        <TableCell>
                          {{ labelFor(documentTypes, row.type) }}
                        </TableCell>
                        <TableCell>
                          <Badge
                            :variant="row.isEnabled ? 'default' : 'secondary'"
                          >
                            {{ row.isEnabled ? "已启用" : "已停用" }}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {{ managementTime(row.updatedAt) }}
                        </TableCell>
                        <TableCell>
                          <div class="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              @click="edit(row)"
                            >
                              编辑
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              @click="confirm('toggle', row)"
                            >
                              {{ row.isEnabled ? "停用" : "启用" }}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              class="text-destructive"
                              @click="confirm('delete', row)"
                            >
                              删除
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    </template>
                    <template v-else-if="tab === 'trash'">
                      <TableRow
                        v-for="row in trash"
                        :key="`${row.objectType}:${row.id}`"
                      >
                        <TableCell
                          class="max-w-72 whitespace-normal font-medium"
                        >
                          {{ row.name }}
                          <p class="text-xs text-muted-foreground">
                            #{{ row.id }}
                          </p>
                        </TableCell>
                        <TableCell>
                          {{ labelFor(objectTypes, row.objectType) }}
                        </TableCell>
                        <TableCell class="max-w-64 whitespace-normal">
                          {{ row.projectName || "—" }}
                          <p
                            v-if="row.parentDeleted"
                            class="text-sm text-destructive"
                          >
                            请先恢复所属项目
                          </p>
                        </TableCell>
                        <TableCell>
                          {{ row.deletedByName }}
                          <p class="text-xs text-muted-foreground">
                            {{ managementTime(row.deletedAt) }}
                          </p>
                        </TableCell>
                        <TableCell class="text-right">
                          <Button
                            variant="outline"
                            size="sm"
                            :disabled="row.parentDeleted"
                            @click="confirmRestore(row)"
                          >
                            恢复
                          </Button>
                        </TableCell>
                      </TableRow>
                    </template>
                    <template v-else>
                      <TableRow v-for="row in audits" :key="row.id">
                        <TableCell>
                          {{ managementTime(row.createdAt) }}
                          <p class="text-xs text-muted-foreground">
                            {{ row.actorName }}
                          </p>
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary">
                            {{ labelFor(auditActions, row.action) }}
                          </Badge>
                        </TableCell>
                        <TableCell class="max-w-72 whitespace-normal">
                          {{ row.objectName }}
                          <p class="text-xs text-muted-foreground">
                            {{ labelFor(objectTypes, row.objectType) }} #{{
                              row.objectId
                            }}
                          </p>
                        </TableCell>
                        <TableCell class="max-w-80 whitespace-normal">
                          {{ row.summary }}
                        </TableCell>
                        <TableCell class="text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            @click="openDetail(row)"
                          >
                            详情
                          </Button>
                        </TableCell>
                      </TableRow>
                    </template>
                  </template>
                </TableBody>
              </Table>
              <EmptyState
                v-else-if="!error"
                :title="
                  Object.keys(route.query).some(
                    (k) => !['tab', 'page'].includes(k),
                  )
                    ? '没有符合条件的记录'
                    : tab === 'templates'
                      ? '还没有文档模板'
                      : tab === 'trash'
                        ? '回收站为空'
                        : '暂无操作记录'
                "
                :description="
                  tab === 'templates'
                    ? '新增模板，帮助团队快速编写同类型文档。'
                    : tab === 'trash'
                      ? '已删除的公共资料会显示在这里。'
                      : '后续资料变更将自动记录；历史操作不会补造。'
                "
              />
            </div>
            <div class="pagination" aria-label="列表分页">
              <span role="status">
                共 {{ total }} 条 · 第 {{ page }} /
                {{ Math.max(1, Math.ceil(total / pageSize)) }} 页
              </span>
              <Button
                variant="outline"
                size="sm"
                :disabled="loading || page <= 1"
                @click="
                  router.push({
                    query: { ...route.query, page: String(page - 1) },
                  })
                "
              >
                上一页
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="loading || page * pageSize >= total"
                @click="
                  router.push({
                    query: { ...route.query, page: String(page + 1) },
                  })
                "
              >
                下一页
              </Button>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </template>
  <TemplateDialog
    v-if="editorOpen"
    ref="templateDialog"
    :row="editing"
    @close="editorOpen = false"
    @saved="
      editorOpen = false;
      load();
    "
  />
  <Dialog
    :open="!!pending"
    @update:open="
      (value) => {
        if (!value && !busy) pending = undefined;
      }
    "
  >
    <DialogContent
      @interact-outside="
        (event) => {
          if (busy) event.preventDefault();
        }
      "
      @escape-key-down="
        (event) => {
          if (busy) event.preventDefault();
        }
      "
    >
      <DialogHeader>
        <DialogTitle>
          {{ actionLabel }}「{{ pending?.row.name }}」？
        </DialogTitle>
        <DialogDescription>
          {{
            pending?.kind === "delete"
              ? "模板将被删除，已保存文档不受影响。此操作不能在回收站恢复。"
              : pending?.kind === "restore"
                ? "恢复后将重新出现在对应页面。恢复项目会重新开放原有效子项，此前单独删除的子项仍保留在回收站。"
                : pending?.row.isEnabled
                  ? "停用后，新文档将不再自动使用此模板。"
                  : "启用后，同类型的其他模板将自动停用。已保存文档不受影响。"
          }}
        </DialogDescription>
      </DialogHeader>
      <p v-if="actionError" role="alert" class="text-sm text-destructive">
        {{ actionError }}
      </p>
      <DialogFooter>
        <Button variant="outline" :disabled="busy" @click="pending = undefined">
          取消
        </Button>
        <Button
          :variant="pending?.kind === 'delete' ? 'destructive' : 'default'"
          :disabled="busy"
          class="min-w-24"
          @click="perform"
        >
          {{ busy ? "处理中…" : actionLabel }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
  <Dialog v-model:open="detailOpen">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>操作详情</DialogTitle>
        <DialogDescription>
          服务器记录的操作信息，时间为北京时间。
        </DialogDescription>
      </DialogHeader>
      <p v-if="detailLoading" role="status">正在读取…</p>
      <p v-else-if="detailError" role="alert" class="text-destructive">
        {{ detailError }}
      </p>
      <dl
        v-else-if="detail"
        class="grid grid-cols-[auto_1fr] gap-x-5 gap-y-3 break-words text-sm"
      >
        <dt>操作人</dt>
        <dd>{{ detail.actorName }}（{{ detail.actorId || "历史记录" }}）</dd>
        <dt>时间</dt>
        <dd>{{ managementTime(detail.createdAt) }}</dd>
        <dt>对象</dt>
        <dd>
          {{ labelFor(objectTypes, detail.objectType) }} ·
          {{ detail.objectName }} #{{ detail.objectId }}
        </dd>
        <dt>操作</dt>
        <dd>{{ labelFor(auditActions, detail.action) }}</dd>
        <dt>摘要</dt>
        <dd>{{ detail.summary }}</dd>
        <dt>变更字段</dt>
        <dd>{{ detail.fields || "—" }}</dd>
        <dt>所属项目 ID</dt>
        <dd>{{ detail.projectId || "—" }}</dd>
      </dl>
    </DialogContent>
  </Dialog>
</template>
