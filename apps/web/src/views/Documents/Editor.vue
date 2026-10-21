<template>
  <PageHeader
    eyebrow="文档编辑"
    title="编辑文档"
    description="直接维护真实文档内容，可插入图片、记录变更，并按项目关联文档类型。"
  >
    <template #actions>
      <Button
        variant="outline"
        :disabled="!currentDocument"
        @click="
          currentDocument &&
          router.push({
            name: 'DocumentDetail',
            params: { id: currentDocument.id },
          })
        "
      >
        <Clock3 class="button-icon" />
        查看记录
      </Button>
      <Button variant="secondary" :disabled="saving || uploading" @click="saveDraft">
        <Save class="button-icon" />
        {{ saving ? "保存中..." : "保存草稿" }}
      </Button>
      <Button :disabled="saving || uploading" @click="publishDocument">
        <Send class="button-icon" />
        发布
      </Button>
    </template>
  </PageHeader>

  <Card class="filter-card">
    <CardContent>
      <div v-if="message" class="notice">{{ message }}</div>
      <div v-if="errorMessage" class="notice notice-error">
        {{ errorMessage }}
      </div>
      <div class="form-grid">
        <label class="ui-field">
          <span>文档标题</span>
          <Input
            v-model="form.title"
            class="ui-input"
            placeholder="例如：云手机用户端环境说明"
          />
        </label>
        <label class="ui-field">
          <span>所属项目</span>
          <UiSelect v-model="form.projectId">
            <SelectTrigger class="ui-select-trigger">
              <SelectValue placeholder="请选择所属项目" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="item in projectOptions"
                :key="item.value"
                :value="item.value"
              >
                {{ item.label }}
              </SelectItem>
            </SelectContent>
          </UiSelect>
        </label>
        <label class="ui-field">
          <span>文档类型</span>
          <UiSelect v-model="form.type">
            <SelectTrigger class="ui-select-trigger">
              <SelectValue placeholder="请选择文档类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="item in docTypeOptions"
                :key="item.value"
                :value="item.value"
              >
                {{ item.label }}
              </SelectItem>
            </SelectContent>
          </UiSelect>
        </label>
      </div>
    </CardContent>
  </Card>

  <div class="editor-layout">
    <Card>
      <CardHeader class="ui-card-header">
        <div>
          <CardTitle>正文</CardTitle>
        </div>
        <CardAction>
          <input ref="imagePickerRef" type="file" accept="image/jpeg,image/png,image/gif,image/webp" class="sr-only" tabindex="-1" aria-label="选择文档图片" @change="uploadFromPicker" />
          <Button type="button" variant="outline" size="sm" :disabled="saving || !editorRef" @click="imagePickerRef?.click()">上传图片</Button>
          <Badge variant="secondary" class="ui-badge ui-badge--default"
            >富文本</Badge
          >
        </CardAction>
      </CardHeader>
      <CardContent>
        <div class="mb-3 flex min-h-6 flex-wrap items-center gap-2 text-sm" aria-live="polite">
          <span v-if="templateLoading" role="status">正在读取文档模板…</span>
          <template v-else-if="templateError"><span role="alert" class="text-destructive">{{ templateError }}</span><Button type="button" variant="outline" size="sm" @click="applyTemplateIfEmpty">重试模板</Button></template>
          <span v-else-if="templateMessage" class="text-muted-foreground">{{ templateMessage }}</span>
        </div>
        <p class="mb-3 text-sm leading-6 text-muted-foreground">支持标题、表格、链接和图片。图片支持 JPG、PNG、GIF、WebP，最大 10MB，相同内容自动复用。</p>
        <div class="mb-3 flex min-h-9 flex-wrap items-center gap-2 text-sm" aria-live="polite">
          <span v-if="uploading" role="status">{{ imageStatus }}（{{ uploading }} 张处理中）</span>
          <span v-else-if="imageError" role="alert" class="text-destructive">{{ imageError }}</span>
          <span v-else-if="imageStatus" role="status">{{ imageStatus }}</span>
          <span v-else class="text-muted-foreground">上传的文档配图为公共图片，请勿包含私人资料。</span>
          <Button v-if="uploading" type="button" variant="outline" size="sm" @click="cancelImageUploads">取消上传</Button>
        </div>
        <div class="wang-editor-shell">
          <div ref="toolbarContainerRef" class="wang-editor-toolbar" />
          <div ref="editorContainerRef" class="wang-editor-body" />
        </div>
      </CardContent>
    </Card>

    <div class="side-stack">
      <Card>
        <CardHeader class="ui-card-header">
          <div>
            <CardTitle>预览</CardTitle>
            <CardDescription>按阅读页样式展示当前富文本内容。</CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          <div class="rich-preview">
            <div class="preview-document-head">
              <h2>{{ form.title || "未命名文档" }}</h2>
              <p>{{ selectedProjectName }} / {{ selectedDocTypeLabel }}</p>
            </div>
            <div
              class="preview-document-body rich-document-body"
              v-html="previewHtml"
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="ui-card-header">
          <div>
            <CardTitle>属性</CardTitle>
            <CardDescription>用于搜索、过期提醒和权限对接。</CardDescription>
          </div>
          <CardAction>
            <Badge variant="secondary" class="ui-badge ui-badge--success"
              >手动保存</Badge
            >
          </CardAction>
        </CardHeader>
        <CardContent>
          <div class="property-list">
            <div>
              <span>维护人</span>
              <strong>研发用户</strong>
            </div>
            <div>
              <span>文档状态</span>
              <strong>{{ statusLabel }}</strong>
            </div>
            <div>
              <span>可见范围</span>
              <strong>暂不限制</strong>
            </div>
            <div>
              <span>图片能力</span>
              <strong>Helios 公共图片</strong>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script setup lang="ts">
import "@wangeditor/editor/dist/css/style.css";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  shallowRef,
  watch,
} from "vue";
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router";
import { uploadDocumentImage } from "@/api/images";
import { getActiveTemplate, documentTypes } from "@/api/management";
import { Clock3, Save, Send } from "lucide-vue-next";
import { createEditor, createToolbar } from "@wangeditor/editor";
import type {
  IDomEditor,
  IEditorConfig,
  IToolbarConfig,
  Toolbar as WangToolbar,
} from "@wangeditor/editor";
import {
  getDocuments,
  getProjectDocument,
  saveDocument,
  type DocumentItem,
} from "@/api/documents";
import { getProjects } from "@/api/projects";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import PageHeader from "@/components/ui/PageHeader.vue";
import {
  Select as UiSelect,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { isHtmlContent, markdownToHtml, stripHtml } from "@/utils/markdown";

const route = useRoute();
const router = useRouter();
const saving = ref(false);
const uploading = ref(0);
const imageStatus = ref("");
const imageError = ref("");
const imageRequests = new Set<AbortController>();

function cancelImageUploads() {
  for (const controller of imageRequests) controller.abort();
  imageStatus.value = "已取消图片上传，可重新选择图片";
}
onBeforeRouteLeave(() => cancelImageUploads());
const loading = ref(false);
const message = ref("");
const errorMessage = ref("");
const currentDocument = ref<DocumentItem | null>(null);
const projectOptions = ref<Array<{ label: string; value: string }>>([]);
const editorRef = shallowRef<IDomEditor | null>(null);
const imagePickerRef = ref<HTMLInputElement | null>(null);
const toolbarRef = shallowRef<WangToolbar | null>(null);
const toolbarContainerRef = ref<HTMLElement | null>(null);
const editorContainerRef = ref<HTMLElement | null>(null);
const isSyncingEditor = ref(false);
const templateLoading = ref(false);
const templateMessage = ref("");
const templateError = ref("");
let templateRequest: AbortController | undefined;
let templateVersion = 0;
let lastTemplateContent = "";

const form = reactive({
  title: "",
  projectId: "",
  type: "environment",
  summary: "",
  status: "draft" as "draft" | "published",
  content: "",
});

const docTypeOptions = documentTypes;

const toolbarConfig: Partial<IToolbarConfig> = {
  excludeKeys: ["group-video", "fullScreen"],
};

const editorConfig: Partial<IEditorConfig> = {
  placeholder: "编写项目介绍、环境说明、部署步骤、API 说明、FAQ 等真实内容...",
  scroll: false,
  MENU_CONF: {
    uploadImage: {
      maxFileSize: 10 * 1024 * 1024,
      allowedFileTypes: ["image/jpeg", "image/png", "image/gif", "image/webp"],
      async customUpload(
        file: File,
        insertFn: (url: string, alt: string, href: string) => void,
      ) {
        if (saving.value) return;
        const controller = new AbortController();
        const editor = editorRef.value;
        const documentRoute = route.fullPath;
        imageRequests.add(controller);
        uploading.value = imageRequests.size;
        imageError.value = "";
        try {
          const result = await uploadDocumentImage(file, {
            signal: controller.signal,
            onStatus: (status) => {
              if (!controller.signal.aborted) imageStatus.value = status;
            },
          });
          if (controller.signal.aborted || editorRef.value !== editor || route.fullPath !== documentRoute) return;
          insertFn(result.url, file.name, "");
          imageStatus.value = result.reused ? "图片已存在，已复用并插入" : "图片已上传并插入";
        } catch (error) {
          if (!controller.signal.aborted) {
            imageError.value = error instanceof Error ? error.message : "图片上传失败，请重新选择图片重试";
            imageStatus.value = "";
          }
        } finally {
          imageRequests.delete(controller);
          uploading.value = imageRequests.size;
        }
      },
    },
  },
  onChange(editor: IDomEditor) {
    if (!isSyncingEditor.value) {
      form.content = editor.getHtml();
    }
  },
};

async function uploadFromPicker(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  const editor = editorRef.value;
  if (!file || !editor) return;
  await editorConfig.MENU_CONF?.uploadImage?.customUpload(file, (url: string, alt: string) => {
    editor.restoreSelection();
    editor.focus();
    const image = document.createElement("img");
    image.src = url;
    image.alt = alt;
    editor.dangerouslyInsertHtml(image.outerHTML);
  });
}

const selectedProjectName = computed(() => {
  return (
    projectOptions.value.find((item) => item.value === form.projectId)?.label ||
    "未选择项目"
  );
});

const selectedDocTypeLabel = computed(() => {
  return (
    docTypeOptions.find((item) => item.value === form.type)?.label || form.type
  );
});

const statusLabel = computed(() =>
  form.status === "published" ? "已发布" : "草稿",
);

const previewHtml = computed(
  () => form.content || "<p>正文内容将在这里预览。</p>",
);

onMounted(async () => {
  await loadProjects();
  await hydrateFromRoute();
  await nextTick();
  mountWangEditor();
});

onBeforeUnmount(() => {
  templateRequest?.abort();
  templateVersion++;
  cancelImageUploads();
  toolbarRef.value?.destroy();
  editorRef.value?.destroy();
});

watch(
  () => form.type,
  () => {
    if (!currentDocument.value && !loading.value) {
      void applyTemplateIfEmpty();
    }
  },
);

async function loadProjects() {
  const res = await getProjects({ page: 1, pageSize: 100 });
  projectOptions.value = res.data.list.map((project) => ({
    label: project.name,
    value: String(project.id),
  }));
}

async function hydrateFromRoute() {
  loading.value = true;
  errorMessage.value = "";
  try {
    const projectId = String(route.query.projectId || "");
    const type = String(route.query.type || "");
    if (projectId) {
      form.projectId = projectId;
    } else if (projectOptions.value[0]) {
      form.projectId = projectOptions.value[0].value;
    }
    if (type) {
      form.type = type;
    }

    if (form.projectId && form.type) {
      await loadExistingDocument(Number(form.projectId), form.type);
    }
    if (!currentDocument.value) {
      await applyTemplateIfEmpty();
    }
  } finally {
    loading.value = false;
  }
}

async function loadExistingDocument(projectId: number, type: string) {
  try {
    const res = await getProjectDocument(projectId, type);
    fillDocument(res.data);
  } catch {
    const list = await getDocuments({ projectId, type, page: 1, pageSize: 1 });
    const firstDocument = list.data.list[0];
    if (firstDocument) {
      fillDocument(firstDocument);
      return;
    }
    currentDocument.value = null;
  }
}

function fillDocument(document: DocumentItem) {
  currentDocument.value = document;
  form.title = document.title;
  form.projectId = String(document.projectId);
  form.type = document.type;
  form.summary = document.summary;
  form.status = document.status === "published" ? "published" : "draft";
  form.content = isHtmlContent(document.content)
    ? document.content
    : markdownToHtml(document.content);
  syncEditorHtml();
}

async function applyTemplateIfEmpty() {
  templateRequest?.abort();
  const version = ++templateVersion;
  templateLoading.value = false;
  templateError.value = "";
  templateMessage.value = "";
  const hasContent = stripHtml(form.content) || /<(img|table|hr|video|audio)\b/i.test(form.content);
  if (currentDocument.value || (hasContent && form.content !== lastTemplateContent)) return;
  const type = form.type;
  const before = form.content;
  const controller = new AbortController();
  templateRequest = controller;
  templateLoading.value = true;
  try {
    const res = await getActiveTemplate(type, controller.signal);
    if (version !== templateVersion || controller.signal.aborted || currentDocument.value || form.type !== type || form.content !== before) return;
    form.content = res.data?.content || "";
    lastTemplateContent = form.content;
    if (!form.title) form.title = `${selectedProjectName.value}${selectedDocTypeLabel.value}`;
    templateMessage.value = res.data ? `已使用模板「${res.data.name}」，可直接修改正文。` : "此类型暂无启用模板，请直接编写正文。";
    syncEditorHtml();
  } catch (error) {
    if (version === templateVersion && !controller.signal.aborted) templateError.value = error instanceof Error ? error.message : "模板读取失败，可重试或直接编写正文";
  } finally {
    if (version === templateVersion) templateLoading.value = false;
  }
}

async function saveDraft() {
  await save("draft");
}

async function publishDocument() {
  await save("published");
}

async function save(status: "draft" | "published") {
  if (uploading.value) {
    errorMessage.value = "请等待图片处理完成，或取消上传后再保存";
    return;
  }
  message.value = "";
  errorMessage.value = "";
  if (!form.projectId) {
    errorMessage.value = "请选择所属项目";
    return;
  }
  if (!form.title.trim()) {
    errorMessage.value = "文档标题不能为空";
    return;
  }

  saving.value = true;
  try {
    const res = await saveDocument({
      projectId: Number(form.projectId),
      type: form.type,
      title: form.title.trim(),
      summary: form.summary.trim() || buildSummary(form.content),
      content: form.content,
      status,
    });
    fillDocument(res.data);
    message.value = status === "published" ? "文档已发布" : "草稿已保存";
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "保存文档失败";
  } finally {
    saving.value = false;
  }
}

function buildSummary(content: string) {
  const plainText = isHtmlContent(content)
    ? stripHtml(content)
    : content
        .split("\n")
        .map((item) =>
          item
            .replace(/^#+\s*/, "")
            .replace(/^[-*]\s*/, "")
            .trim(),
        )
        .find(Boolean);
  return (plainText || "暂无摘要").slice(0, 120);
}

function mountWangEditor() {
  if (
    !editorContainerRef.value ||
    !toolbarContainerRef.value ||
    editorRef.value
  ) {
    return;
  }

  const editor = createEditor({
    selector: editorContainerRef.value,
    html: form.content || "<p><br></p>",
    config: editorConfig,
    mode: "default",
  });
  const toolbar = createToolbar({
    editor,
    selector: toolbarContainerRef.value,
    config: toolbarConfig,
    mode: "default",
  });

  editorRef.value = editor;
  toolbarRef.value = toolbar;
}

function syncEditorHtml() {
  const editor = editorRef.value;
  if (!editor || editor.isDestroyed || editor.getHtml() === form.content) {
    return;
  }
  isSyncingEditor.value = true;
  editor.setHtml(form.content || "<p><br></p>");
  isSyncingEditor.value = false;
}
</script>
