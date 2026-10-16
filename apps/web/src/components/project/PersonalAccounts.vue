<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from "vue";
import { onBeforeRouteLeave, onBeforeRouteUpdate } from "vue-router";
import {
  KeyRound,
  Plus,
  Copy,
  Eye,
  EyeOff,
  PencilLine,
  Trash2,
  RotateCcw,
  Search,
  X,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select as UiSelect,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from "@/components/ui/select";
import EmptyState from "@/components/ui/EmptyState.vue";
import { getProject, getProjects, type Project } from "@/api/projects";
import EntryFormDialog from "@/components/project/EntryFormDialog.vue";
import { getEntry, listEntries, type PersonalEntry } from "@/api/entries";
import { getPublicLink, getPublicLinks, type PublicLink } from "@/api/public-links";
import {
  deleteAccount,
  getAccount,
  listAccounts,
  readPassword,
  restoreAccount,
  saveAccount,
  type AccountInput,
  type PersonalAccount,
  type AccountStorage,
} from "@/composables/personal-accounts";
import { formatDate } from "@/utils/project-identity";
import { useAuthStore } from "@/store/auth";

const props = defineProps<{ projectId?: number; toolId?: number; entryId?: number; autoCreate?: boolean }>();
const emit=defineEmits<{createConsumed:[]}>();
const newEntryOpen=ref(false);
const entryEditor=ref<InstanceType<typeof EntryFormDialog>|null>(null);
const projectComposing = ref(false);
async function clearSearch() {
  keyword.value = "";
  await nextTick();
  document
    .querySelector<HTMLInputElement>('input[aria-label="搜索我的账号"]')
    ?.focus();
}
async function clearProjectSearch() {
  projectKeyword.value = "";
  await nextTick();
  document.getElementById("account-project-search")?.focus();
}
const auth = useAuthStore();
const storageFilter = ref("all"), saveStorage = ref<AccountStorage>("local"), warning = ref("");
const formSnapshot = () => JSON.stringify({ ...form, storage: saveStorage.value });
const rows = ref<PersonalAccount[]>([]),
  total = ref(0),
  page = ref(1),
  loading = ref(false),
  error = ref("");
const keyword = ref(""),
  environment = ref("all"),
  trash = ref(false),
  composing = ref(false);
const environments = [
  { value: "production", label: "线上" },
  { value: "staging", label: "测试" },
  { value: "other", label: "其他" },
];
const envLabel = (value: string) =>
  environments.find((e) => e.value === value)?.label || value;
const accountTitle = (item: PersonalAccount) => item.entryId ? item.entryName || "私人入口" : item.toolId ? item.toolName || "所属工具" : item.projectName || selectedProject.value?.name || "所属项目";
const accountLabel = (item: PersonalAccount) => item.name || accountTitle(item);
let generation = 0,
  debounce: ReturnType<typeof setTimeout> | undefined;
async function load() {
  const ticket = ++generation;
  loading.value = true;
  error.value = "";
  hidePassword();
  try {
    const result = await listAccounts({
      projectId: props.projectId,
      toolId: props.toolId,
      entryId: props.entryId,
      keyword: keyword.value,
      environment: environment.value === "all" ? undefined : environment.value,
      trash: trash.value,
      page: page.value,
      pageSize: 12,
      storage: storageFilter.value,
    });
    if (ticket !== generation) return;
    rows.value = result.data.list;
    total.value = result.data.total;
    warning.value = result.data.warning;
    if (page.value > 1 && !rows.value.length) {
      page.value = Math.max(1, Math.ceil(total.value / 12));
      return;
    }
  } catch (e) {
    if (ticket === generation) error.value = message(e);
  } finally {
    if (ticket === generation) loading.value = false;
  }
}
watch([keyword, composing], () => {
  clearTimeout(debounce);
  if (composing.value) return;
  debounce = setTimeout(
    () => {
      if (page.value !== 1) page.value = 1;
      else void load();
    },
    keyword.value ? 300 : 0,
  );
});
watch([trash, environment, storageFilter, () => props.projectId, () => props.toolId, () => props.entryId], () => {
  page.value = 1;
});
watch(
  () => auth.user,
  () => {
    rows.value = [];
    total.value = 0;
    deleting.value = null;
    warning.value = "";
    hidePassword();
    resetForm();
    void load();
  },
);
function message(e: unknown) {
  return e instanceof Error ? e.message : "操作失败，请重试";
}

const shownId = ref<number | null>(null),
  shownPassword = ref(""),
  secretBusy = ref<number | null>(null);
let secretEpoch = 0,
  timer: ReturnType<typeof setTimeout> | undefined;
function hidePassword() {
  secretEpoch++;
  clearTimeout(timer);
  shownId.value = null;
  shownPassword.value = "";
  secretBusy.value = null;
}
function onVisibility() {
  if (document.hidden) hidePassword();
}
window.addEventListener("blur", hidePassword);
document.addEventListener("visibilitychange", onVisibility);
async function password(item: PersonalAccount, copy = false) {
  if (shownId.value === item.id && !copy) {
    hidePassword();
    return;
  }
  hidePassword();
  const ticket = secretEpoch;
  secretBusy.value = item.id;
  try {
    const result = await readPassword(item.id);
    if (ticket !== secretEpoch) return;
    if (copy) {
      await navigator.clipboard.writeText(result.data.password);
      toast.success("密码已复制");
    } else {
      shownId.value = item.id;
      shownPassword.value = result.data.password;
      timer = setTimeout(hidePassword, 15000);
    }
  } catch (e) {
    if (ticket === secretEpoch) toast.error(message(e));
  } finally {
    if (ticket === secretEpoch) secretBusy.value = null;
  }
}
async function copyUsername(value: string) {
  try {
    await navigator.clipboard.writeText(value);
    toast.success("用户名已复制");
  } catch {
    toast.error("复制失败，请手动选择用户名");
  }
}

const editingId = ref<number | null>(null),
  formOpen = ref(false),
  formLoading = ref(false),
  saving = ref(false),
  formError = ref(""),
  showNewPassword = ref(false);
const form = reactive<AccountInput>({
  projectId: 0,
  toolId: 0,
  entryId: 0,
  name: "",
  username: "",
  environment: "staging",
  resourceUrl: "",
  note: "",
  passwordAction: "keep",
  password: "",
  revision: 0,
});
const baseline = ref(""),
  dirty = computed(
    () => formOpen.value && formSnapshot() !== baseline.value,
  );
const targetKind = ref<"project" | "tool" | "entry">("project");
const selectedEntry=ref<PersonalEntry|null>(null);
const entryOptions=ref<PersonalEntry[]>([]);
const selectedTool = ref<PublicLink | null>(null);
const toolOptions = ref<PublicLink[]>([]);
const targetReady = computed(() => targetKind.value === "entry" ? Boolean(selectedEntry.value) : targetKind.value === "tool" ? Boolean(selectedTool.value) : Boolean(selectedProject.value));
const targetOptions = computed(() => targetKind.value === "entry" ? entryOptions.value : targetKind.value === "tool" ? toolOptions.value : projectOptions.value);
const targetLabel = computed(() => targetKind.value === "entry" ? "私人入口" : targetKind.value === "tool" ? "工具" : "项目");
const selectedProject = ref<Project | null>(null),
  projectOptions = ref<Project[]>([]),
  projectKeyword = ref(""),
  projectsLoading = ref(false),
  projectDetailLoading = ref(false),
  projectError = ref("");
let formEpoch = 0,
  projectEpoch = 0,
  projectTimer: ReturnType<typeof setTimeout> | undefined;
const dialogOpen = computed({
  get: () => formOpen.value,
  set: (value) => {
    if (!value) requestClose();
  },
});
const discardOpen = ref(false);
let afterDiscard: (() => void) | null = null;
function resetForm() {
  formEpoch++;
  projectEpoch++;
  formOpen.value = false;
  editingId.value = null;
  saveStorage.value = "local";
  formLoading.value = false;
  projectDetailLoading.value = false;
  Object.assign(form, {
    projectId: 0,
    toolId: 0,
  entryId: 0,
    name: "",
    username: "",
    environment: "staging",
    resourceUrl: "",
    note: "",
    passwordAction: "keep",
    password: "",
    revision: 0,
  });
  showNewPassword.value = false;
  selectedProject.value = null;
  selectedTool.value = null;
  selectedEntry.value = null;
  toolOptions.value = [];
  entryOptions.value = [];
  formError.value = "";
  projectKeyword.value = "";
  projectOptions.value = [];
  hidePassword();
}
function requestClose() {
  if (saving.value) return;
  if (dirty.value) {
    discardOpen.value = true;
    afterDiscard = () => resetForm();
  } else resetForm();
}
function discard() {
  discardOpen.value = false;
  const next = afterDiscard;
  afterDiscard = null;
  resetForm();
  next?.();
}
function stay() {
  discardOpen.value = false;
  afterDiscard = null;
}
async function fetchProjects() {
  const ticket = ++projectEpoch;
  projectsLoading.value = true;
  projectError.value = "";
  try {
    const params = { keyword: projectKeyword.value, page: 1, pageSize: 20 };
    if (targetKind.value === "entry") {
      const result=await listEntries({...params,source:"private"});
      if(ticket===projectEpoch)entryOptions.value=result.data.list;
    } else if (targetKind.value === "tool") {
      const result = await getPublicLinks(params);
      if (ticket === projectEpoch) toolOptions.value = result.data.list;
    } else {
      const result = await getProjects(params);
      if (ticket === projectEpoch) projectOptions.value = result.data.list;
    }
  } catch (e) {
    if (ticket === projectEpoch) projectError.value = message(e);
  } finally {
    if (ticket === projectEpoch) projectsLoading.value = false;
  }
}
watch([projectKeyword, projectComposing], () => {
  clearTimeout(projectTimer);
  if (projectComposing.value) return;
  projectTimer = setTimeout(() => {
    void fetchProjects();
  }, 300);
});
async function selectProject(id: number) {
  const ticket = ++formEpoch;
  projectDetailLoading.value = true;
  selectedProject.value = null;
  projectError.value = "";
  selectedTool.value = null;
  selectedEntry.value = null;
  form.projectId = targetKind.value === "project" ? id : 0;
  form.toolId = targetKind.value === "tool" ? id : 0;
  form.entryId = targetKind.value === "entry" ? id : 0;
  try {
    if(targetKind.value==="entry"){const result=await getEntry(id);if(ticket===formEpoch)selectedEntry.value=result.data;} else if (targetKind.value === "tool") {
      const result = await getPublicLink(id);
      if (ticket === formEpoch) selectedTool.value = result.data;
    } else {
      const result = await getProject(id);
      if (ticket === formEpoch) selectedProject.value = result.data;
    }
  } catch (e) {
    if (ticket === formEpoch) projectError.value = message(e);
  } finally {
    if (ticket === formEpoch) projectDetailLoading.value = false;
  }
}
function changeTargetKind() {
  formEpoch++;
  projectEpoch++;
  projectDetailLoading.value = false;
  selectedProject.value = null;
  selectedTool.value = null;
  selectedEntry.value = null;
  form.projectId = 0;
  form.toolId = 0;
  form.entryId = 0;
  form.resourceUrl = "";
  projectOptions.value = [];
  toolOptions.value = [];
  entryOptions.value = [];
  projectKeyword.value = "";
  void fetchProjects();
}
const projectValue = computed({
  get: () => (form.entryId || form.toolId || form.projectId ? String(form.entryId || form.toolId || form.projectId) : undefined),
  set: (value) => {
    form.resourceUrl = "";
    void selectProject(Number(value));
  },
});
const resourceValue = computed({
  get: () => form.resourceUrl || "none",
  set: (value) => {
    form.resourceUrl = String(value) === "none" ? "" : String(value);
  },
});
async function openForm(item?: PersonalAccount) {
  resetForm();
  targetKind.value = item?.entryId || props.entryId ? "entry" : item?.toolId || props.toolId ? "tool" : "project";
  editingId.value = item?.id || null;
  saveStorage.value = item && item.id > 0 ? "server" : "local";
  formOpen.value = true;
  baseline.value = formSnapshot();
  formLoading.value = true;
  const ticket = ++formEpoch;
  try {
    if (item) {
      const result = await getAccount(item.id);
      if (ticket !== formEpoch) return;
      const a = result.data.account;
      Object.assign(form, {
        projectId: a.projectId,
        toolId: a.toolId || 0,
        entryId: a.entryId || 0,
        name: a.name,
        username: a.username,
        environment: a.environment,
        resourceUrl: a.resourceUrl,
        note: result.data.note,
        passwordAction: "keep",
        password: "",
        revision: a.revision,
      });
    } else { form.projectId = props.projectId || 0; form.toolId = props.toolId || 0; form.entryId=props.entryId||0; }
    baseline.value = formSnapshot();
    if(form.entryId){const result=await getEntry(form.entryId);if(ticket!==formEpoch)return;selectedEntry.value=result.data;} else if (form.toolId) {
      const result = await getPublicLink(form.toolId);
      if (ticket !== formEpoch) return;
      selectedTool.value = result.data;
    } else if (form.projectId) {
      const result = await getProject(form.projectId);
      if (ticket !== formEpoch) return;
      selectedProject.value = result.data;
    } else await fetchProjects();
  } catch (e) {
    if (ticket === formEpoch) formError.value = message(e);
  } finally {
    if (ticket === formEpoch) formLoading.value = false;
  }
}
watch(
  () => form.passwordAction,
  () => {
    form.password = "";
    showNewPassword.value = false;
  },
);
async function submit() {
  if (saving.value || formLoading.value || projectDetailLoading.value) return;
  formError.value = "";
  if (
    !form.username.trim() ||
    !targetReady.value ||
    (form.passwordAction === "replace" && !form.password)
  ) {
    formError.value = "请选择所属项目、工具或私人入口，填写用户名，以及要修改的密码。";
    await nextTick();
    document
      .querySelector<HTMLElement>('[data-account-form] [aria-invalid="true"]')
      ?.focus();
    return;
  }
  saving.value = true;
  const ticket = formEpoch;
  try {
    await saveAccount(editingId.value, {
      ...form,
      resourceUrl: targetKind.value === "entry" ? (selectedEntry.value?.url||"") : targetKind.value === "tool" ? (selectedTool.value?.url || "") : form.resourceUrl,
      password: form.passwordAction === "replace" ? form.password : "",
    }, saveStorage.value, {
      projectName: selectedProject.value?.name,
      toolName: selectedTool.value?.name,
      entryName: selectedEntry.value?.name,
    });
    if (ticket !== formEpoch) return;
    const savedStorage = saveStorage.value;
    resetForm();
    toast.success("账号已保存");
    if (storageFilter.value !== "all" && storageFilter.value !== savedStorage) storageFilter.value = savedStorage;
    if (page.value !== 1) page.value = 1;
    else await load();
  } catch (e) {
    if (ticket === formEpoch) formError.value = message(e);
  } finally {
    saving.value = false;
  }
}
const deleting = ref<PersonalAccount | null>(null),
  deletingBusy = ref(false),
  deleteError = ref(""),
  restoreBusy = ref<number | null>(null);
async function confirmDelete() {
  if (!deleting.value || deletingBusy.value) return;
  deletingBusy.value = true;
  deleteError.value = "";
  hidePassword();
  try {
    await deleteAccount(deleting.value);
    deleting.value = null;
    toast.success("账号已移入回收站");
    await load();
  } catch (e) {
    deleteError.value = message(e);
  } finally {
    deletingBusy.value = false;
  }
}
async function restore(item: PersonalAccount) {
  if (restoreBusy.value) return;
  restoreBusy.value = item.id;
  try {
    await restoreAccount(item.id);
    toast.success("账号已恢复");
    await load();
  } catch (e) {
    toast.error(message(e));
  } finally {
    restoreBusy.value = null;
  }
}
function confirmAccountLeave(): boolean | Promise<boolean> {
  if (saving.value) return false;
  if (!dirty.value) return true;
  return new Promise<boolean>((resolve) => {
    discardOpen.value = true;
    afterDiscard = () => resolve(true);
    const stop = watch(discardOpen, (open) => {
      if (!open) {
        stop();
        if (afterDiscard === null) resolve(false);
      }
    });
  });
}
async function confirmLeave():Promise<boolean>{
 if(newEntryOpen.value){if(!entryEditor.value||!await entryEditor.value.confirmLeave())return false;newEntryOpen.value=false}
 return confirmAccountLeave();
}
onBeforeRouteLeave(confirmLeave);
onBeforeRouteUpdate(confirmLeave);
defineExpose({ confirmLeave });
async function entryCreated(entry:PersonalEntry){
 targetKind.value="entry";form.projectId=0;form.toolId=0;form.entryId=entry.id;form.resourceUrl=entry.url;
 selectedProject.value=null;selectedTool.value=null;selectedEntry.value=entry;entryOptions.value=[entry,...entryOptions.value.filter(item=>item.id!==entry.id)];
}
function beforeUnload(event: BeforeUnloadEvent) {
  if (dirty.value) {
    event.preventDefault();
    event.returnValue = "";
  }
}
window.addEventListener("beforeunload", beforeUnload);
onBeforeUnmount(() => {
  generation++;
  resetForm();
  clearTimeout(debounce);
  clearTimeout(projectTimer);
  window.removeEventListener("blur", hidePassword);
  document.removeEventListener("visibilitychange", onVisibility);
  window.removeEventListener("beforeunload", beforeUnload);
});
watch(
  [page, trash, environment, storageFilter, () => props.projectId, () => props.toolId, () => props.entryId],
  () => {
    void load();
  },
  { immediate: true },
);
watch([() => props.projectId, () => props.toolId, () => props.entryId], () => {
  resetForm();
  rows.value = [];
  hidePassword();
});
watch(()=>props.autoCreate,async value=>{if(value){await openForm();emit("createConsumed")}}, {immediate:true});
</script>

<template>
  <section class="personal-accounts" aria-label="我的账号">
    <div class="account-section-heading">
      <div>
        <h2>
          {{ trash ? "账号回收站" : (props.projectId || props.toolId || props.entryId) ? "我的账号" : "账号列表" }}
        </h2>
        <p>本地账号仅在此浏览器显示；服务器账号可跨设备使用。删除后保留 7 天。</p>
      </div>
      <div class="page-actions">
        <Button variant="ghost" @click="trash = !trash">{{
          trash ? "返回账号" : "回收站"
        }}</Button
        ><Button v-if="!trash" @click="openForm()"
          ><Plus class="button-icon" />添加账号</Button
        >
      </div>
    </div>
    <div class="account-filters">
      <div class="account-search">
        <Search aria-hidden="true" /><Input
          v-model="keyword"
          aria-label="搜索我的账号"
          placeholder="搜索账号、用户名、项目或工具"
          @compositionstart="composing = true"
          @compositionend="composing = false"
        /><Button
          v-if="keyword"
          variant="ghost"
          size="icon"
          aria-label="清空账号搜索"
          @click="clearSearch"
          ><X
        /></Button>
      </div>
      <UiSelect v-model="storageFilter">
        <SelectTrigger aria-label="筛选保存位置"><SelectValue /></SelectTrigger>
        <SelectContent><SelectItem value="all">全部保存位置</SelectItem><SelectItem value="local">仅此浏览器</SelectItem><SelectItem value="server">服务器同步</SelectItem></SelectContent>
      </UiSelect>
      <UiSelect v-model="environment"
        ><SelectTrigger aria-label="筛选账号环境"><SelectValue /></SelectTrigger
        ><SelectContent
          ><SelectItem value="all">全部环境</SelectItem
          ><SelectItem
            v-for="env in environments"
            :key="env.value"
            :value="env.value"
            >{{ env.label }}</SelectItem
          ></SelectContent
        ></UiSelect
      >
    </div>
    <p v-if="warning && !loading" role="status" class="notice">{{ warning }}<Button variant="ghost" @click="load">重试加载</Button></p>
    <div v-if="error" class="notice notice-error" role="alert">
      {{ error }}<Button variant="ghost" @click="load">重试</Button>
    </div>
    <p v-else-if="loading" role="status">正在加载账号…</p>
    <EmptyState
      v-else-if="!rows.length"
      :icon="KeyRound"
      :title="
        trash
          ? '回收站为空'
          : keyword || environment !== 'all' || storageFilter !== 'all'
            ? '没有匹配的账号'
            : '还没有个人账号'
      "
      :description="
        trash
          ? '删除的账号会在这里保留 7 天。'
          : '保存你在项目或工具中的个人账号，方便下次使用。'
      "
    />
    <div v-else class="account-card-grid">
      <article v-for="item in rows" :key="item.id" class="account-card">
        <div class="account-card-heading">
          <h3><RouterLink :to="item.entryId ? '/my-entries' : item.toolId ? '/public-links' : `/projects/${item.projectId}`" class="hover:underline">{{ accountTitle(item) }}</RouterLink></h3>
          <span class="account-environment">{{
            envLabel(item.environment)
          }}环境</span>
        </div>
        <p v-if="item.name" class="mt-2 text-sm text-muted-foreground">用途：{{ item.name }}</p>
        <div class="account-value">
          <span>用户名</span>
          <div>
            <code>{{ item.username }}</code
            ><Button
              v-if="!trash"
              variant="ghost"
              size="icon"
              :aria-label="`复制${accountLabel(item)}用户名`"
              @click="copyUsername(item.username)"
              ><Copy
            /></Button>
          </div>
        </div>
        <div class="account-value">
          <span>密码</span>
          <div>
            <code :class="{ 'account-password-hidden': shownId !== item.id }">{{
              shownId === item.id
                ? shownPassword
                : item.hasPassword
                  ? "••••••••"
                  : "未保存密码"
            }}</code
            ><template v-if="!trash && item.hasPassword"
              ><Button
                variant="ghost"
                size="icon"
                :disabled="secretBusy === item.id"
                :aria-label="`${shownId === item.id ? '隐藏' : '显示'}${accountLabel(item)}密码`"
                @click="password(item)"
                ><EyeOff v-if="shownId === item.id" /><Eye v-else /></Button
              ><Button
                variant="ghost"
                size="icon"
                :disabled="secretBusy === item.id"
                :aria-label="`复制${accountLabel(item)}密码`"
                @click="password(item, true)"
                ><Copy /></Button
            ></template>
          </div>
          <small v-if="shownId === item.id">15 秒后自动隐藏</small>
        </div>
        <div class="account-card-actions">
          <template v-if="!trash"
            ><a
              v-if="item.resourceUrl"
              :href="item.resourceUrl"
              target="_blank"
              rel="noopener noreferrer"
              >打开入口 ↗</a
            ><Button variant="ghost" size="sm" @click="openForm(item)"
              ><PencilLine class="button-icon" />编辑</Button
            ><Button
              variant="ghost"
              size="sm"
              @click="
                deleting = item;
                deleteError = '';
              "
              ><Trash2 class="button-icon" />删除</Button
            ></template
          ><Button
            v-else
            variant="secondary"
            :disabled="restoreBusy !== null"
            @click="restore(item)"
            ><RotateCcw class="button-icon" />{{
              restoreBusy === item.id ? "正在恢复…" : "恢复账号"
            }}</Button
          >
        </div>
        <p class="account-timestamp flex flex-wrap justify-between gap-x-4 gap-y-1">
          <span>{{ item.id < 0 ? "仅存此浏览器" : "保存于服务器" }}</span>
          <span>{{
            item.deletedAt
              ? `删除于 ${formatDate(item.deletedAt)}`
              : `更新于 ${formatDate(item.updatedAt)}`
          }}</span>
        </p>
      </article>
    </div>
    <div v-if="!loading && !error && total" class="account-pagination">
      <span
        >共 {{ total }} 条 · 第 {{ page }} /
        {{ Math.max(1, Math.ceil(total / 12)) }} 页</span
      ><Button variant="outline" :disabled="page <= 1" @click="page--"
        >上一页</Button
      ><Button variant="outline" :disabled="page * 12 >= total" @click="page++"
        >下一页</Button
      >
    </div>
  </section>

  <Dialog v-model:open="dialogOpen"
    ><DialogContent
      class="max-h-[calc(100dvh-2rem)] overflow-y-auto"
      @escape-key-down="saving && $event.preventDefault()"
      ><DialogTitle>{{ editingId ? "编辑账号" : "添加账号" }}</DialogTitle
      ><DialogDescription
        >{{ saveStorage === 'local' ? '用户名、密码、用途和私人备注仅保存在此浏览器，不上传服务器。' : '账号同步到服务器，密码和私人备注加密保存。' }}</DialogDescription
      >
      <p v-if="formLoading" role="status">正在加载账号资料…</p>
      <form v-else data-account-form novalidate @submit.prevent="submit">
        <fieldset :disabled="saving" class="account-form-fields">
          <div
            v-if="formError"
            id="account-form-error"
            class="notice notice-error"
            role="alert"
          >
            {{ formError }}
          </div>
          <div class="ui-field">
            <span id="account-storage-label">保存位置</span>
            <UiSelect v-model="saveStorage" :disabled="editingId !== null">
              <SelectTrigger aria-labelledby="account-storage-label" aria-describedby="account-storage-help"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="local">仅此浏览器</SelectItem><SelectItem value="server">服务器同步</SelectItem></SelectContent>
            </UiSelect>
            <p id="account-storage-help" class="text-sm text-muted-foreground">{{ saveStorage === 'local' ? '本地数据不加密，使用此浏览器的人可能查看。清除网站数据会丢失，换浏览器或网址不会同步。入口名称和网址仍保存在服务器。' : '使用你的 Atlas 账号登录后，可在其他设备查看。' }}{{ editingId !== null ? ' 编辑时不更改保存位置。' : '' }}</p>
          </div>
          <div v-if="!editingId && !props.projectId && !props.toolId && !props.entryId" class="ui-field">
            <span id="account-target-kind-label">关联类型</span>
            <UiSelect v-model="targetKind" @update:model-value="changeTargetKind">
              <SelectTrigger aria-labelledby="account-target-kind-label"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="project">项目</SelectItem><SelectItem value="tool">工具</SelectItem><SelectItem value="entry">私人入口</SelectItem></SelectContent>
            </UiSelect>
          </div>
          <label v-if="editingId || props.projectId || props.toolId || props.entryId" class="ui-field"
            ><span>所属{{ targetLabel }}</span
            ><Input :model-value="selectedEntry?.name || selectedTool?.name || selectedProject?.name || '加载失败'" readonly
          /></label>
          <div v-else class="ui-field">
            <label for="account-project-search">所属{{ targetLabel }} *</label
            ><Input
              id="account-project-search"
              v-model="projectKeyword"
              @compositionstart="projectComposing = true"
              @compositionend="projectComposing = false"
              :placeholder="`搜索${targetLabel}名称`"
            /><Button
              v-if="projectKeyword"
              type="button"
              variant="ghost"
              size="sm"
              :aria-label="`清空${targetLabel}搜索`"
              @click="clearProjectSearch"
              ><X class="button-icon" />清空</Button
            ><UiSelect v-model="projectValue"
              ><SelectTrigger :aria-label="`选择所属${targetLabel}`"
                ><SelectValue :placeholder="`请选择${targetLabel}`" /></SelectTrigger
              ><SelectContent
                ><SelectItem
                  v-for="p in targetOptions"
                  :key="p.id"
                  :value="String(p.id)"
                  >{{ p.name }}</SelectItem
                ></SelectContent
              ></UiSelect
            ><Button v-if="targetKind==='entry'" type="button" variant="outline" @click="newEntryOpen=true">新建私人入口</Button><small class="min-h-5" aria-live="polite">{{ projectsLoading ? `正在搜索${targetLabel}…` : !targetOptions.length ? `暂无匹配${targetLabel}` : "" }}</small>
          </div>
          <div v-if="projectError" class="notice notice-error" role="alert">
            {{ projectError
            }}<Button
              type="button"
              variant="ghost"
              @click="
                (form.entryId || form.toolId || form.projectId) ? selectProject(form.entryId || form.toolId || form.projectId) : fetchProjects()
              "
              >重试加载{{ targetLabel }}</Button
            >
          </div>
          <div class="ui-field">
            <span id="account-environment-label">环境</span
            ><UiSelect v-model="form.environment"
              ><SelectTrigger aria-labelledby="account-environment-label"
                ><SelectValue /></SelectTrigger
              ><SelectContent
                ><SelectItem
                  v-for="env in environments"
                  :key="env.value"
                  :value="env.value"
                  >{{ env.label }}</SelectItem
                ></SelectContent
              ></UiSelect
            >
          </div>
          <label class="ui-field"
            ><span>用户名 *</span
            ><Input
              v-model="form.username"
              :aria-invalid="Boolean(formError && !form.username.trim())"
              :aria-describedby="formError ? 'account-form-error' : undefined"
              placeholder="账号或邮箱"
              autocomplete="off"
              maxlength="300"
          /></label>
          <div class="ui-field">
            <span id="account-password-action-label">密码处理</span
            ><UiSelect v-model="form.passwordAction"
              ><SelectTrigger aria-labelledby="account-password-action-label"
                ><SelectValue /></SelectTrigger
              ><SelectContent
                ><SelectItem value="keep">{{
                  editingId ? "保留原密码" : "暂不保存密码"
                }}</SelectItem
                ><SelectItem value="replace">{{
                  editingId ? "修改密码" : "保存密码"
                }}</SelectItem
                ><SelectItem v-if="editingId" value="clear"
                  >清除已存密码</SelectItem
                ></SelectContent
              ></UiSelect
            >
          </div>
          <div v-if="form.passwordAction === 'replace'" class="ui-field">
            <label for="personal-account-password">新密码 *</label>
            <div class="account-password-input">
              <Input
                id="personal-account-password"
                v-model="form.password"
                :aria-invalid="Boolean(formError && !form.password)"
                :aria-describedby="formError ? 'account-form-error' : undefined"
                :type="showNewPassword ? 'text' : 'password'"
                autocomplete="new-password"
                placeholder="请输入要保存的密码"
                maxlength="4096"
              /><Button
                type="button"
                variant="ghost"
                size="icon"
                :aria-label="showNewPassword ? '隐藏新密码' : '显示新密码'"
                :aria-pressed="showNewPassword"
                @click="showNewPassword = !showNewPassword"
                ><EyeOff v-if="showNewPassword" /><Eye v-else
              /></Button>
            </div>
          </div>
          <p v-if="form.passwordAction === 'clear'" class="notice">
            保存后将清除已存密码，请确认此操作。
          </p>
          <label v-if="targetKind !== 'project'" class="ui-field">
            <span>{{targetKind==='entry'?'私人入口网址':'工具入口'}}</span>
            <Input
              :model-value="selectedEntry?.url || selectedTool?.url || ''"
              :placeholder="projectDetailLoading ? '正在加载工具入口…' : '选择工具后自动带入'"
              :aria-busy="projectDetailLoading"
              readonly
            />
          </label>
          <div v-else class="ui-field">
            <span id="account-resource-label">关联资源入口（选填）</span
            ><UiSelect
              v-model="resourceValue"
              :disabled="projectDetailLoading || !selectedProject"
              ><SelectTrigger
                aria-labelledby="account-resource-label"
                :aria-busy="projectDetailLoading"
                ><span v-if="projectDetailLoading" role="status"
                  >正在加载项目入口…</span
                ><SelectValue v-else /></SelectTrigger
              ><SelectContent
                ><SelectItem value="none">不关联入口</SelectItem
                ><SelectItem
                  v-for="entry in selectedProject?.entries || []"
                  :key="entry.id"
                  :value="entry.url"
                  >{{ entry.title }} · {{ entry.url }}</SelectItem
                ></SelectContent
              ></UiSelect
            >
          </div>
          <label class="ui-field"><span>用途备注（选填）</span><Input v-model="form.name" placeholder="例如：宝塔登录、运营主账号" maxlength="200" /></label>
          <label class="ui-field"
            ><span>私人备注（选填）</span
            ><Textarea
              v-model="form.note"
              class="resize-none"
              rows="3"
              placeholder="仅本人可见的账号说明"
              maxlength="10000"
          /></label>
        </fieldset>
        <DialogFooter
          ><Button
            type="button"
            variant="outline"
            :disabled="saving"
            @click="requestClose"
            >取消</Button
          ><Button type="submit" :disabled="saving || projectDetailLoading || !targetReady">{{
            saving ? "正在保存…" : "保存账号"
          }}</Button></DialogFooter
        >
      </form>
    </DialogContent></Dialog
  >
  <Dialog
    :open="Boolean(deleting)"
    @update:open="
      (value) => {
        if (!value && !deletingBusy) deleting = null;
      }
    "
    ><DialogContent
      ><DialogTitle>删除账号</DialogTitle
      ><DialogDescription
        >将“{{ deleting ? accountTitle(deleting) + (deleting.name ? ' · ' + deleting.name : '') : '' }}”的账号移入回收站，7
        天内可恢复，之后自动清除。</DialogDescription
      >
      <p v-if="deleteError" role="alert" class="notice notice-error">
        {{ deleteError }}
      </p>
      <DialogFooter
        ><Button
          variant="outline"
          :disabled="deletingBusy"
          autofocus
          @click="deleting = null"
          >取消</Button
        ><Button
          variant="destructive"
          :disabled="deletingBusy"
          @click="confirmDelete"
          >{{ deletingBusy ? "正在删除…" : "移入回收站" }}</Button
        ></DialogFooter
      ></DialogContent
    ></Dialog
  >
  <Dialog
    :open="discardOpen"
    @update:open="
      (value) => {
        if (!value) stay();
      }
    "
    ><DialogContent
      ><DialogTitle>放弃未保存的修改？</DialogTitle
      ><DialogDescription>未保存的账号、密码和备注将被清除。</DialogDescription
      ><DialogFooter
        ><Button variant="outline" @click="stay">继续编辑</Button
        ><Button variant="destructive" @click="discard"
          >放弃修改</Button
        ></DialogFooter
      ></DialogContent
    ></Dialog
  >
  <EntryFormDialog ref="entryEditor" v-model:open="newEntryOpen" compact @saved="entryCreated" />
</template>
