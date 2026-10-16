<template>
  <Dialog v-model:open="dialogOpen">
    <DialogContent class="max-h-[calc(100dvh-2rem)] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ isEditing ? "编辑项目" : "新建项目" }}</DialogTitle>
        <DialogDescription>
          {{
            isEditing
              ? "维护项目基础信息、核心入口和负责人，保存后会同步刷新项目页面。"
              : "录入项目基础信息、核心入口和负责人，后续可继续补充项目文档。"
          }}
        </DialogDescription>
      </DialogHeader>

      <form novalidate class="project-form" @submit.prevent="submitProject">
        <div class="form-grid">
          <label class="ui-field">
            <span>项目名称</span>
            <Input
              v-model="form.name"
              class="ui-input"
              placeholder="例如：云手机用户端"
            />
          </label>
          <label class="ui-field">
            <span>项目形态</span>
            <UiSelect v-model="form.type">
              <SelectTrigger class="ui-select-trigger">
                <SelectValue placeholder="请选择项目形态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="type in projectTypes"
                  :key="type.value"
                  :value="type.value"
                >
                  {{ type.label }}
                </SelectItem>
              </SelectContent>
            </UiSelect>
          </label>
          <label class="ui-field">
            <span>关键词</span>
            <Input
              v-model="form.keywords"
              class="ui-input"
              placeholder="云手机,购买,客户入口"
            />
          </label>
        </div>

        <label class="ui-field">
          <span>项目简介</span>
          <Textarea
            v-model="form.description"
            class="ui-textarea resize-none"
            placeholder="说明这个业务系统的用途、范围和核心能力。"
          />
        </label>

        <div class="entry-form-section">
          <div class="entry-form-head">
            <div>
              <div class="section-label">项目入口</div>
              <p>同一类型可添加多个入口，用名称区分测试、线上等不同用途。</p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              @click="addEntryRow()"
            >
              <Plus class="button-icon" />
              新增入口
            </Button>
          </div>

          <div class="entry-form-list">
            <div
              v-for="(entry, index) in form.entries"
              :key="entry.id"
              class="entry-form-row"
            >
              <label class="ui-field entry-type-field">
                <span>类型</span>
                <UiSelect v-model="entry.type">
                  <SelectTrigger class="ui-select-trigger">
                    <SelectValue placeholder="请选择入口类型" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="type in entryTypeOptions"
                      :key="type.value"
                      :value="type.value"
                    >
                      {{ type.label }}
                    </SelectItem>
                  </SelectContent>
                </UiSelect>
              </label>
              <label class="ui-field">
                <span>入口名称（选填）</span>
                <Input
                  v-model="entry.title"
                  maxlength="100"
                  :placeholder="entryTypeLabel(entry.type) + '，例如：测试发布系统'"
                />
              </label>
              <label class="ui-field entry-url-field">
                <span>URL</span>
                <Input
                  v-model="entry.url"
                  class="ui-input"
                  placeholder="https://example.company.local"
                />
              </label>
              <div class="entry-row-actions">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="上移"
                  :disabled="index === 0"
                  @click="moveEntry(index, -1)"
                >
                  <ArrowUp class="button-icon" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="下移"
                  :disabled="index === form.entries.length - 1"
                  @click="moveEntry(index, 1)"
                >
                  <ArrowDown class="button-icon" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="删除"
                  :disabled="form.entries.length <= 1"
                  @click="removeEntry(index)"
                >
                  <Trash2 class="button-icon" />
                </Button>
              </div>
              <label class="ui-field entry-remark-field">
                <span>备注（选填）</span>
                <Textarea
                  v-model="entry.remark"
                  class="ui-textarea resize-none"
                  rows="2"
                  placeholder="例如：仅内网访问、需先连接 VPN，或说明此入口的用途"
                />
              </label>
            </div>
          </div>
        </div>

        <div class="form-grid">
          <label class="ui-field">
            <span>前端负责人</span>
            <Input
              v-model="form.frontendOwners"
              class="ui-input"
              placeholder="多人用逗号分隔"
            />
          </label>
          <label class="ui-field">
            <span>后端负责人</span>
            <Input
              v-model="form.backendOwners"
              class="ui-input"
              placeholder="多人用逗号分隔"
            />
          </label>
          <label class="ui-field">
            <span>测试负责人</span>
            <Input
              v-model="form.testerOwners"
              class="ui-input"
              placeholder="多人用逗号分隔"
            />
          </label>
        </div>

        <div v-if="formError" class="notice notice-error">{{ formError }}</div>

        <DialogFooter>
          <Button variant="outline" type="button" @click="dialogOpen = false"
            >取消</Button
          >
          <Button type="submit" :disabled="saving">
            <Save class="button-icon" />
            {{ saving ? "保存中..." : isEditing ? "保存修改" : "保存项目" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ArrowDown, ArrowUp, Plus, Save, Trash2 } from "lucide-vue-next";
import {
  createProject,
  updateProject,
  type Project,
  type ProjectEntry,
  type ProjectOwner,
} from "@/api/projects";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select as UiSelect,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

type EntryFormRow = {
  id: number;
  type: string;
  title: string;
  url: string;
  remark: string;
};

const props = defineProps<{
  open: boolean;
  project?: Project | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  saved: [project: Project];
}>();

const projectTypes = [
  { label: "用户端", value: "user_client" },
  { label: "后台系统", value: "admin" },
  { label: "前端应用", value: "frontend" },
  { label: "后端服务", value: "backend" },
  { label: "官网", value: "website" },
  { label: "移动端", value: "mobile" },
  { label: "小程序", value: "mini_program" },
  { label: "公共库/组件", value: "shared_lib" },
  { label: "其他", value: "other" },
];

const entryTypeOptions = [
  { label: "线上地址", value: "production" },
  { label: "测试地址", value: "staging" },
  { label: "GitLab", value: "gitlab" },
  { label: "原型地址", value: "prototype" },
  { label: "API 文档", value: "api_doc" },
  { label: "后台地址", value: "admin" },
  { label: "监控地址", value: "monitor" },
  { label: "日志地址", value: "logs" },
  { label: "发布系统", value: "deploy" },
  { label: "账号权限", value: "access" },
  { label: "其他", value: "other" },
];

let entryRowId = 0;

const form = reactive({
  name: "",
  type: "user_client",
  keywords: "",
  description: "",
  entries: [] as EntryFormRow[],
  frontendOwners: "",
  backendOwners: "",
  testerOwners: "",
});

const saving = ref(false);
const formError = ref("");

const dialogOpen = computed({
  get: () => props.open,
  set: (value) => emit("update:open", value),
});

const isEditing = computed(() => Boolean(props.project));

watch(
  () => [props.open, props.project] as const,
  ([open]) => {
    if (!open) return;
    hydrateForm(props.project || null);
    formError.value = "";
  },
  { immediate: true },
);

async function submitProject() {
  formError.value = "";
  if (!form.name.trim()) {
    formError.value = "项目名称不能为空";
    return;
  }
  if (!form.type) {
    formError.value = "请选择项目形态";
    return;
  }

  saving.value = true;
  try {
    const payload = {
      name: form.name.trim(),
      type: form.type,
      description: form.description.trim(),
      keywords: splitText(form.keywords),
      entries: buildEntryPayload(),
      frontendOwners: splitText(form.frontendOwners).map((ownerName) => ({
        ownerName,
      })),
      backendOwners: splitText(form.backendOwners).map((ownerName) => ({
        ownerName,
      })),
      testerOwners: splitText(form.testerOwners).map((ownerName) => ({
        ownerName,
      })),
      sortOrder: props.project?.sortOrder || 0,
    };
    const res = props.project
      ? await updateProject(props.project.id, payload)
      : await createProject(payload);
    emit("saved", res.data);
    dialogOpen.value = false;
  } catch (error) {
    formError.value =
      error instanceof Error
        ? error.message
        : isEditing.value
          ? "更新项目失败"
          : "创建项目失败";
  } finally {
    saving.value = false;
  }
}

function hydrateForm(project: Project | null) {
  if (!project) {
    resetForm();
    return;
  }
  form.name = project.name;
  form.type = project.type;
  form.keywords = project.keywords.join("，");
  form.description = project.description || "";
  form.entries = normalizeEntries(project.entries);
  form.frontendOwners = ownerNamesForForm(project.owners?.frontend);
  form.backendOwners = ownerNamesForForm(project.owners?.backend);
  form.testerOwners = ownerNamesForForm(project.owners?.tester);
}

function resetForm() {
  form.name = "";
  form.type = "user_client";
  form.keywords = "";
  form.description = "";
  form.entries = normalizeEntries([]);
  form.frontendOwners = "";
  form.backendOwners = "";
  form.testerOwners = "";
}

function addEntryRow(type = "other") {
  form.entries.push(createEntryRow(type));
}

function removeEntry(index: number) {
  if (form.entries.length <= 1) return;
  form.entries.splice(index, 1);
}

function moveEntry(index: number, direction: -1 | 1) {
  const nextIndex = index + direction;
  if (nextIndex < 0 || nextIndex >= form.entries.length) return;
  const [entry] = form.entries.splice(index, 1);
  if (!entry) return;
  form.entries.splice(nextIndex, 0, entry);
}

function splitText(value: string) {
  return value
    .split(/[,，、\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function ownerNamesForForm(owners?: ProjectOwner[] | null) {
  return owners?.map((owner) => owner.ownerName).join("，") || "";
}

function normalizeEntries(entries: ProjectEntry[] | undefined) {
  const normalized =
    entries
      ?.slice()
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .map((entry) => createEntryRow(entry.type, entry.url, entry.remark, entry.title)) || [];

  for (const type of ["production", "staging", "gitlab"]) {
    if (!normalized.some((entry) => entry.type === type)) {
      normalized.push(createEntryRow(type));
    }
  }
  return normalized.length
    ? normalized
    : [
        createEntryRow("production"),
        createEntryRow("staging"),
        createEntryRow("gitlab"),
      ];
}

function createEntryRow(type = "other", url = "", remark = "", title = ""): EntryFormRow {
  entryRowId += 1;
  return {
    id: entryRowId,
    type,
    title,
    url,
    remark,
  };
}

function entryTypeLabel(type: string) {
  return (
    entryTypeOptions.find((option) => option.value === type)?.label || "其他"
  );
}

function buildEntryPayload() {
  return form.entries
    .map((entry) => ({ ...entry, url: entry.url.trim() }))
    .filter((entry) => entry.url)
    .map((entry) => ({
      type: entry.type,
      title: entry.title.trim() || entryTypeLabel(entry.type),
      url: entry.url,
      remark: entry.remark.trim(),
    }));
}
</script>
