<template>
  <div class="document-composer">
    <PageHeader
      title="编辑文档"
      description="直接编写文档，也可以按需关联项目。"
    >
      <template #actions>
        <Button
          variant="outline"
          :disabled="saving"
          @click="router.push('/documents')"
          ><ArrowLeft class="button-icon" />返回文档库</Button
        >
        <Button
          v-if="contextReady"
          variant="outline"
          :disabled="busy"
          @click="previewOpen = true"
          ><Eye class="button-icon" />预览</Button
        >
        <Button
          variant="secondary"
          class="min-w-28"
          :disabled="busy || !contextReady"
          @click="saveDraft"
        >
          <Loader2 v-if="saving" class="button-icon animate-spin" /><Save
            v-else
            class="button-icon"
          />保存草稿
        </Button>
        <Button
          class="min-w-28"
          :disabled="busy || !contextReady"
          @click="publishDocument"
          ><Send class="button-icon" />发布文档</Button
        >
      </template>
    </PageHeader>
    <div v-if="message" class="notice" role="status">{{ message }}</div>
    <div
      v-if="errorMessage"
      id="document-save-error"
      class="notice notice-error"
      role="alert"
    >
      {{ errorMessage }}
    </div>
    <Card :aria-busy="loading">
      <CardContent class="space-y-4">
        <div class="composer-context">
          <p class="text-sm text-muted-foreground">
            {{
              currentDocument
                ? "正在编辑已保存的文档"
                : "新文档 · 项目关联为选填"
            }}
          </p>
          <Badge variant="secondary">{{ statusLabel }}</Badge>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="ui-field">
            <label for="document-project"
              >关联项目
              <span class="font-normal text-muted-foreground"
                >（选填）</span
              ></label
            >
            <UiSelect
              v-model="selectedProject"
              :disabled="busy || projectsLoading"
            >
              <SelectTrigger id="document-project" class="w-full"
                ><SelectValue
                  >{{ selectedProjectName
                  }}<template v-if="!form.projectId"
                    >（通用文档）</template
                  ></SelectValue
                ></SelectTrigger
              >
              <SelectContent>
                <SelectItem value="__none__">不关联项目（通用文档）</SelectItem>
                <SelectItem
                  v-for="item in availableProjectOptions"
                  :key="item.value"
                  :value="item.value"
                  >{{ item.label }}</SelectItem
                >
              </SelectContent>
            </UiSelect>
          </div>
          <div class="ui-field">
            <label for="document-type">文档类型</label>
            <UiSelect v-model="form.type" :disabled="busy">
              <SelectTrigger id="document-type" class="w-full"
                ><SelectValue
              /></SelectTrigger>
              <SelectContent
                ><SelectItem
                  v-for="item in docTypeOptions"
                  :key="item.value"
                  :value="item.value"
                  >{{ item.label }}</SelectItem
                ></SelectContent
              >
            </UiSelect>
          </div>
        </div>
        <p
          v-if="projectsLoading"
          class="text-sm text-muted-foreground"
          role="status"
        >
          正在加载可关联的项目，不影响编写通用文档。
        </p>
        <div
          v-if="projectLoadError"
          class="flex flex-wrap items-center gap-2 text-sm"
          role="alert"
        >
          <span>{{ projectLoadError }}，仍可编写不关联项目的文档。</span
          ><Button variant="outline" size="sm" @click="loadProjects"
            >重试项目列表</Button
          >
        </div>
        <p v-if="loading" role="status" class="text-sm text-muted-foreground">
          正在加载文档资料…
        </p>
        <div
          v-if="loadError"
          class="flex flex-wrap items-center gap-2"
          role="alert"
        >
          <span class="text-sm text-destructive">{{ loadError }}</span
          ><Button variant="outline" size="sm" @click="initialize"
            >重新加载</Button
          >
        </div>
      </CardContent>
    </Card>
    <form
      v-show="contextReady"
      novalidate
      class="space-y-5"
      @submit.prevent="saveDraft"
    >
      <Card>
        <CardContent class="space-y-5">
          <div class="ui-field">
            <label for="document-title"
              >文档标题
              <span class="font-normal text-muted-foreground"
                >（必填）</span
              ></label
            >
            <Input
              id="document-title"
              v-model="form.title"
              :disabled="saving"
              :aria-invalid="invalidTitle"
              :aria-describedby="
                invalidTitle ? 'document-save-error' : undefined
              "
              placeholder="为文档起一个清楚的标题"
            />
          </div>
          <div class="ui-field">
            <label for="document-summary"
              >内容摘要
              <span class="font-normal text-muted-foreground"
                >（选填）</span
              ></label
            >
            <Textarea
              id="document-summary"
              v-model="form.summary"
              class="min-h-20 resize-none"
              :disabled="saving"
              placeholder="简要说明这篇文档的内容；留空时从正文提取"
            />
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <div class="composer-context">
            <CardTitle>正文</CardTitle>
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-xs text-muted-foreground"
                >{{ wordCount }} 字</span
              >
              <input
                ref="imagePickerRef"
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                class="sr-only"
                tabindex="-1"
                aria-label="选择文档图片"
                @change="uploadFromPicker"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                :disabled="busy || !editorRef"
                @click="imagePickerRef?.click()"
                ><ImagePlus class="button-icon" />上传图片</Button
              >
            </div>
          </div>
          <div
            v-if="templateLoading || templateError || templateMessage"
            class="flex flex-wrap items-center gap-2 text-sm"
            aria-live="polite"
          >
            <span v-if="templateLoading" role="status">正在读取文档模板…</span>
            <template v-else-if="templateError"
              ><span role="alert" class="text-destructive">{{
                templateError
              }}</span
              ><Button
                type="button"
                variant="outline"
                size="sm"
                :disabled="busy"
                @click="applyTemplateIfEmpty"
                >重试模板</Button
              ></template
            >
            <span v-else class="text-muted-foreground">{{
              templateMessage
            }}</span>
          </div>
        </CardHeader>
        <CardContent class="space-y-3">
          <div class="wang-editor-shell">
            <div ref="toolbarContainerRef" class="wang-editor-toolbar" />
            <div ref="editorContainerRef" class="wang-editor-body" />
          </div>
          <div
            class="flex flex-wrap items-center gap-2 text-xs leading-6 text-muted-foreground"
            aria-live="polite"
          >
            <span v-if="uploading" role="status"
              >{{ imageStatus }}（{{ uploading }} 张处理中）</span
            >
            <span
              v-else-if="imageError"
              role="alert"
              class="text-destructive"
              >{{ imageError }}</span
            >
            <span v-else-if="imageStatus" role="status">{{ imageStatus }}</span>
            <Button
              v-if="uploading"
              type="button"
              variant="outline"
              size="sm"
              @click="cancelImageUploads"
              >取消上传</Button
            >
            <span
              >图片支持 JPG、PNG、GIF、WebP，最大
              10MB。上传后可公开访问，请勿包含私人资料。</span
            >
          </div>
        </CardContent>
      </Card>
      <div class="composer-footer" aria-live="polite">
        <span
          >{{
            dirty
              ? "有未保存的修改"
              : currentDocument
                ? "修改已保存"
                : "新文档尚未保存"
          }}<template v-if="currentDocument">
            · 版本 {{ currentDocument.currentVersionNo }}</template
          ></span
        >
        <span>保存草稿和发布都会生成版本记录。</span>
      </div>
    </form>
    <Dialog v-model:open="previewOpen">
      <DialogContent
        class="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-4xl"
      >
        <DialogHeader
          ><DialogTitle>文档预览</DialogTitle
          ><DialogDescription
            >查看当前编辑内容，预览不会保存或发布。</DialogDescription
          ></DialogHeader
        >
        <article class="min-w-0 space-y-4 break-words">
          <header class="space-y-2 border-b pb-4">
            <h2 class="text-2xl font-semibold">
              {{ form.title || "请填写文档标题" }}
            </h2>
            <p class="text-sm text-muted-foreground">
              {{ selectedProjectName }} · {{ selectedDocTypeLabel }}
            </p>
            <p v-if="form.summary" class="text-sm text-muted-foreground">
              {{ form.summary }}
            </p>
          </header>
          <div
            class="preview-document-body rich-document-body"
            v-html="previewHtml"
          />
        </article>
      </DialogContent>
    </Dialog>
    <Dialog v-model:open="discardOpen">
      <DialogContent>
        <DialogHeader
          ><DialogTitle>放弃未保存的修改？</DialogTitle
          ><DialogDescription
            >离开文档会丢失本次未保存的内容，并取消正在上传的图片。</DialogDescription
          ></DialogHeader
        >
        <DialogFooter
          ><Button variant="outline" @click="discardOpen = false"
            >继续编辑</Button
          ><Button variant="destructive" @click="discardChanges"
            >放弃更改</Button
          ></DialogFooter
        >
      </DialogContent>
    </Dialog>
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
import {
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
  useRoute,
  useRouter,
} from "vue-router";
import { ApiResponseError } from "@/request";
import { uploadDocumentImage } from "@/api/images";
import { getActiveTemplate, documentTypes } from "@/api/management";
import {
  ArrowLeft,
  Eye,
  ImagePlus,
  Loader2,
  Save,
  Send,
} from "lucide-vue-next";
import { createEditor, createToolbar } from "@wangeditor/editor";
import type {
  IDomEditor,
  IEditorConfig,
  IToolbarConfig,
  Toolbar as WangToolbar,
} from "@wangeditor/editor";
import {
  getDocument,
  getProjectDocument,
  saveDocument,
  type DocumentItem,
} from "@/api/documents";
import { getProjects } from "@/api/projects";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import PageHeader from "@/components/ui/PageHeader.vue";
import {
  Select as UiSelect,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { destroyRichEditor } from "@/utils/destroy-rich-editor";
import { Textarea } from "@/components/ui/textarea";
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

const loading = ref(true);
const contextReady = ref(false);
const loadError = ref("");
const previewOpen = ref(false);
const discardOpen = ref(false);
const invalidTitle = ref(false);
const baseline = ref("");
let pendingLeave: ((allow: boolean) => void) | undefined;
const projectsLoading = ref(false);
const projectLoadError = ref("");
let disposed = false;
let loadVersion = 0;
const snapshot = () =>
  JSON.stringify([
    form.title,
    form.summary,
    form.content,
    form.projectId,
    form.type,
  ]);
const dirty = computed(
  () => contextReady.value && snapshot() !== baseline.value,
);
const busy = computed(
  () => loading.value || saving.value || uploading.value > 0,
);
const wordCount = computed(
  () => stripHtml(form.content).replace(/\s/g, "").length,
);
function beforeUnload(event: BeforeUnloadEvent) {
  if (dirty.value || saving.value || uploading.value) {
    event.preventDefault();
    event.returnValue = "";
  }
}
function confirmLeave() {
  if (saving.value) return false;
  if (!dirty.value && !uploading.value) {
    cancelImageUploads();
    return true;
  }
  discardOpen.value = true;
  return new Promise<boolean>((resolve) => {
    pendingLeave = resolve;
  });
}
onBeforeRouteLeave(confirmLeave);
onBeforeRouteUpdate(confirmLeave);
watch(
  () => [
    route.query.id,
    route.query.projectId,
    route.query.type,
    route.query.mode,
  ],
  () => {
    if (
      currentDocument.value &&
      String(currentDocument.value.id) === route.query.id
    )
      return;
    resetContext();
    void initialize();
  },
);
watch(discardOpen, (open) => {
  if (!open) {
    pendingLeave?.(false);
    pendingLeave = undefined;
  }
});
function resetContext() {
  loadVersion++;
  loading.value = false;
  templateRequest?.abort();
  templateVersion++;
  contextReady.value = false;
  currentDocument.value = null;
  form.title = "";
  form.summary = "";
  form.content = "";
  form.status = "draft";
  templateMessage.value = "";
  templateError.value = "";
  message.value = "";
  errorMessage.value = "";
  invalidTitle.value = false;
  syncEditorHtml();
}
function discardChanges() {
  cancelImageUploads();
  const resolve = pendingLeave;
  pendingLeave = undefined;
  discardOpen.value = false;
  resolve?.(true);
}
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

const form = reactive({
  title: "",
  projectId: "",
  type: "general",
  summary: "",
  status: "draft" as "draft" | "published",
  content: "",
});

const docTypeOptions = documentTypes;
// Preserve the option keyed by ID when its asynchronous label resolves.
const availableProjectOptions = computed(() => {
  if (
    !form.projectId ||
    projectOptions.value.some((item) => item.value === form.projectId)
  )
    return projectOptions.value;
  return [
    { value: form.projectId, label: "项目 #" + form.projectId },
    ...projectOptions.value,
  ];
});
const selectedProject = computed({
  get: () => form.projectId || "__none__",
  set: (value: string) => {
    form.projectId = value === "__none__" ? "" : value;
  },
});

const toolbarConfig: Partial<IToolbarConfig> = {
  toolbarKeys: [
    "headerSelect",
    "|",
    "bold",
    "italic",
    "underline",
    "|",
    "bulletedList",
    "numberedList",
    "blockquote",
    "|",
    "insertLink",
    "insertTable",
    "codeBlock",
    "uploadImage",
    "|",
    "undo",
    "redo",
  ],
};

const editorConfig: Partial<IEditorConfig> = {
  placeholder: "从这里开始写正文…",
  scroll: false,
  MENU_CONF: {
    uploadImage: {
      maxFileSize: 10 * 1024 * 1024,
      allowedFileTypes: ["image/jpeg", "image/png", "image/gif", "image/webp"],
      async customUpload(
        file: File,
        insertFn: (url: string, alt: string, href: string) => void,
      ) {
        if (saving.value || loading.value || !contextReady.value) return;
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
          if (
            controller.signal.aborted ||
            editorRef.value !== editor ||
            route.fullPath !== documentRoute
          )
            return;
          insertFn(result.url, file.name, "");
          imageStatus.value = result.reused
            ? "图片已存在，已复用并插入"
            : "图片已上传并插入";
        } catch (error) {
          if (!controller.signal.aborted) {
            imageError.value =
              error instanceof Error
                ? error.message
                : "图片上传失败，请重新选择图片重试";
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
      form.content = editor.isEmpty() ? "" : editor.getHtml();
    }
  },
};

async function uploadFromPicker(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  const editor = editorRef.value;
  if (!file || !editor) return;
  await editorConfig.MENU_CONF?.uploadImage?.customUpload(
    file,
    (url: string, alt: string) => {
      editor.restoreSelection();
      editor.focus();
      const image = document.createElement("img");
      image.src = url;
      image.alt = alt;
      editor.dangerouslyInsertHtml(image.outerHTML);
    },
  );
}

const selectedProjectName = computed(() => {
  return (
    projectOptions.value.find((item) => item.value === form.projectId)?.label ||
    (form.projectId ? `项目 #${form.projectId}` : "不关联项目")
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
  window.addEventListener("beforeunload", beforeUnload);
  await nextTick();
  if (!disposed) mountWangEditor();
  await initialize();
});
onBeforeUnmount(() => {
  disposed = true;
  window.removeEventListener("beforeunload", beforeUnload);
  pendingLeave?.(false);
  templateRequest?.abort();
  templateVersion++;
  cancelImageUploads();
  destroyRichEditor(editorRef.value, toolbarRef.value);
});
watch([contextReady, busy], () => {
  if (!editorRef.value) return;
  if (contextReady.value && !busy.value) editorRef.value.enable();
  else editorRef.value.disable();
});
watch(
  () => form.type,
  () => {
    if (contextReady.value && !loading.value && !currentDocument.value)
      void applyTemplateIfEmpty();
  },
);
watch(
  () => form.title,
  () => {
    invalidTitle.value = false;
  },
);
async function loadProjects() {
  projectsLoading.value = true;
  projectLoadError.value = "";
  try {
    const options: Array<{ label: string; value: string }> = [];
    let page = 1;
    while (true) {
      const res = await getProjects({ page, pageSize: 100 });
      if (disposed) return;
      options.push(
        ...res.data.list.map((project) => ({
          label: project.name,
          value: String(project.id),
        })),
      );
      if (!res.data.list.length || options.length >= res.data.total) break;
      page++;
    }
    projectOptions.value = options;
  } catch (error) {
    if (!disposed)
      projectLoadError.value =
        error instanceof Error ? error.message : "项目列表加载失败";
  } finally {
    if (!disposed) projectsLoading.value = false;
  }
}
async function initialize() {
  const version = ++loadVersion;
  loading.value = true;
  loadError.value = "";
  if (!projectOptions.value.length && !projectsLoading.value)
    void loadProjects();
  try {
    const id = Number(route.query.id || 0);
    const projectId = Number(route.query.projectId || 0);
    const type = String(route.query.type || "general");
    form.projectId = projectId > 0 ? String(projectId) : "";
    form.type = docTypeOptions.some((t) => t.value === type) ? type : "general";
    let document: DocumentItem | null = null;
    if (route.query.id) {
      if (!Number.isInteger(id) || id <= 0) throw new Error("无效的文档 ID");
      document = (await getDocument(id)).data;
      if (!document) throw new Error("文档不存在");
    } else if (
      projectId > 0 &&
      route.query.type &&
      route.query.mode !== "create"
    ) {
      try {
        document = (await getProjectDocument(projectId, form.type)).data;
      } catch (error) {
        if (
          (error instanceof ApiResponseError
            ? error.code
            : (error as { response?: { status?: number } }).response
                ?.status) !== 404
        )
          throw error;
      }
    }
    if (disposed || version !== loadVersion) return;
    if (document) fillDocument(document);
    baseline.value = snapshot();
    contextReady.value = true;
    syncEditorHtml();
  } catch (error) {
    if (!disposed && version === loadVersion)
      loadError.value =
        error instanceof Error ? error.message : "文档加载失败，请重试";
  } finally {
    if (!disposed && version === loadVersion) loading.value = false;
  }
  if (
    !disposed &&
    version === loadVersion &&
    contextReady.value &&
    !currentDocument.value
  )
    await applyTemplateIfEmpty();
}

function fillDocument(document: DocumentItem) {
  currentDocument.value = document;
  form.title = document.title;
  form.projectId = document.projectId ? String(document.projectId) : "";
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
  const hasContent =
    stripHtml(form.content) ||
    /<(img|table|hr|video|audio)\b/i.test(form.content);
  if (currentDocument.value || hasContent) return;
  const type = form.type;
  const before = form.content;
  const controller = new AbortController();
  templateRequest = controller;
  templateLoading.value = true;
  try {
    const res = await getActiveTemplate(type, controller.signal);
    if (
      version !== templateVersion ||
      controller.signal.aborted ||
      currentDocument.value ||
      form.type !== type ||
      form.content !== before
    )
      return;
    const untouched = snapshot() === baseline.value;
    form.content = res.data?.content || "";

    templateMessage.value = res.data
      ? `已使用模板「${res.data.name}」，可直接修改正文。`
      : "";
    syncEditorHtml();
    if (untouched) baseline.value = snapshot();
  } catch (error) {
    if (version === templateVersion && !controller.signal.aborted)
      templateError.value =
        error instanceof Error
          ? error.message
          : "模板读取失败，可重试或直接编写正文";
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
  if (saving.value || loading.value || !contextReady.value) return;
  if (uploading.value) {
    errorMessage.value = "请等待图片处理完成，或取消上传后再保存";
    return;
  }
  message.value = "";
  errorMessage.value = "";
  if (!form.title.trim()) {
    invalidTitle.value = true;
    errorMessage.value = "请填写文档标题后再保存";
    document.getElementById("document-title")?.focus();
    return;
  }

  templateRequest?.abort();
  templateVersion++;
  templateLoading.value = false;
  saving.value = true;
  try {
    const res = await saveDocument({
      id: currentDocument.value?.id,
      projectId: form.projectId ? Number(form.projectId) : null,
      type: form.type,
      title: form.title.trim(),
      summary: form.summary.trim() || buildSummary(form.content),
      content: form.content,
      status,
    });
    fillDocument(res.data);
    baseline.value = snapshot();
    message.value = status === "published" ? "文档已发布" : "草稿已保存";
    saving.value = false;
    await router.replace({
      path: "/documents/editor",
      query: { id: String(res.data.id) },
    });
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
  if (!contextReady.value || busy.value) editor.disable();
  editorContainerRef.value
    .querySelector("[data-slate-editor]")
    ?.setAttribute("aria-label", "文档正文");
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

<style scoped>
.document-composer {
  display: grid;
  gap: 24px;
  min-width: 0;
  max-width: 1120px;
  margin-inline: auto;
}
:deep(.page-header) {
  margin-bottom: 0;
}
.composer-context {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.composer-picker {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 16px;
  align-items: end;
}
.composer-footer {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 24px;
  padding-bottom: 24px;
  color: var(--muted-foreground);
  font-size: 13px;
}
@media (max-width: 640px) {
  .document-composer {
    gap: 16px;
  }
  .composer-picker {
    grid-template-columns: minmax(0, 1fr);
  }
  .composer-context {
    flex-wrap: wrap;
  }
}
</style>
