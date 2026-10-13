<template>
  <PageHeader
    eyebrow="文档编辑"
    title="编辑富文本文档"
    description="直接维护真实文档内容，可插入图片、记录变更，并按项目关联文档类型。"
  >
    <template #actions>
      <Button variant="outline">
        <Clock3 class="button-icon" />
        查看记录
      </Button>
      <Button variant="secondary" :disabled="saving" @click="saveDraft">
        <Save class="button-icon" />
        {{ saving ? '保存中...' : '保存草稿' }}
      </Button>
      <Button :disabled="saving" @click="publishDocument">
        <Send class="button-icon" />
        发布
      </Button>
    </template>
  </PageHeader>

  <Card class="filter-card">
    <CardContent>
      <div v-if="message" class="notice">{{ message }}</div>
      <div v-if="errorMessage" class="notice notice-error">{{ errorMessage }}</div>
      <div class="form-grid">
        <label class="ui-field">
          <span>文档标题</span>
          <Input v-model="form.title" class="ui-input" placeholder="例如：云手机用户端环境说明" />
        </label>
        <label class="ui-field">
          <span>所属项目</span>
          <Select v-model="form.projectId">
            <SelectTrigger class="ui-select-trigger">
              <SelectValue placeholder="请选择所属项目" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="item in projectOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="ui-field">
          <span>文档类型</span>
          <Select v-model="form.type">
            <SelectTrigger class="ui-select-trigger">
              <SelectValue placeholder="请选择文档类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="item in docTypeOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </label>
      </div>
    </CardContent>
  </Card>

  <div class="editor-layout">
    <Card>
      <CardHeader class="ui-card-header">
        <div>
          <CardTitle>正文</CardTitle>
          <CardDescription>使用富文本编辑器维护内容，支持标题、表格、链接和图片。</CardDescription>
        </div>
        <CardAction>
          <Badge variant="secondary" class="ui-badge ui-badge--default">富文本</Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
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
              <h2>{{ form.title || '未命名文档' }}</h2>
              <p>{{ selectedProjectName }} / {{ selectedDocTypeLabel }}</p>
            </div>
            <div class="preview-document-body rich-document-body" v-html="previewHtml" />
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
            <Badge variant="secondary" class="ui-badge ui-badge--success">自动保存中</Badge>
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
              <strong>内部文件服务</strong>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@wangeditor/editor/dist/css/style.css'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Clock3, Save, Send } from 'lucide-vue-next'
import { createEditor, createToolbar } from '@wangeditor/editor'
import type { IDomEditor, IEditorConfig, IToolbarConfig, Toolbar as WangToolbar } from '@wangeditor/editor'
import { getDocuments, getProjectDocument, saveDocument, type DocumentItem } from '@/api/documents'
import { getProjects } from '@/api/projects'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import PageHeader from '@/components/ui/PageHeader.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { isHtmlContent, markdownToHtml, stripHtml } from '@/utils/markdown'

const route = useRoute()
const saving = ref(false)
const loading = ref(false)
const message = ref('')
const errorMessage = ref('')
const currentDocument = ref<DocumentItem | null>(null)
const projectOptions = ref<Array<{ label: string; value: string }>>([])
const editorRef = shallowRef<IDomEditor | null>(null)
const toolbarRef = shallowRef<WangToolbar | null>(null)
const toolbarContainerRef = ref<HTMLElement | null>(null)
const editorContainerRef = ref<HTMLElement | null>(null)
const isSyncingEditor = ref(false)

const form = reactive({
  title: '',
  projectId: '',
  type: 'environment',
  summary: '',
  status: 'draft' as 'draft' | 'published',
  content: '',
})

const docTypeOptions = [
  { label: '项目介绍', value: 'intro' },
  { label: '环境说明', value: 'environment' },
  { label: '部署说明', value: 'deploy' },
  { label: 'API 说明', value: 'api' },
  { label: '账号/权限申请说明', value: 'access' },
  { label: '常见问题 FAQ', value: 'faq' },
  { label: '开发规范', value: 'standard' },
]

const toolbarConfig: Partial<IToolbarConfig> = {
  excludeKeys: ['group-video', 'fullScreen'],
}

const editorConfig: Partial<IEditorConfig> = {
  placeholder: '编写项目介绍、环境说明、部署步骤、API 说明、FAQ 等真实内容...',
  scroll: false,
  MENU_CONF: {
    uploadImage: {
      customUpload(file: File, insertFn: (url: string, alt: string, href: string) => void) {
        const reader = new FileReader()
        reader.onload = () => {
          if (typeof reader.result === 'string') {
            insertFn(reader.result, file.name, '')
          }
        }
        reader.readAsDataURL(file)
      },
    },
  },
  onChange(editor: IDomEditor) {
    if (!isSyncingEditor.value) {
      form.content = editor.getHtml()
    }
  },
}

const selectedProjectName = computed(() => {
  return projectOptions.value.find((item) => item.value === form.projectId)?.label || '未选择项目'
})

const selectedDocTypeLabel = computed(() => {
  return docTypeOptions.find((item) => item.value === form.type)?.label || form.type
})

const statusLabel = computed(() => (form.status === 'published' ? '已发布' : '草稿'))

const previewHtml = computed(() => form.content || '<p>正文内容将在这里预览。</p>')

onMounted(async () => {
  await loadProjects()
  await hydrateFromRoute()
  await nextTick()
  mountWangEditor()
})

onBeforeUnmount(() => {
  toolbarRef.value?.destroy()
  editorRef.value?.destroy()
})

watch(
  () => form.type,
  () => {
    if (!currentDocument.value) {
      applyTemplateIfEmpty()
    }
  },
)

async function loadProjects() {
  const res = await getProjects({ page: 1, pageSize: 100 })
  projectOptions.value = res.data.list.map((project) => ({
    label: project.name,
    value: String(project.id),
  }))
}

async function hydrateFromRoute() {
  loading.value = true
  errorMessage.value = ''
  try {
    const projectId = String(route.query.projectId || '')
    const type = String(route.query.type || '')
    if (projectId) {
      form.projectId = projectId
    } else if (projectOptions.value[0]) {
      form.projectId = projectOptions.value[0].value
    }
    if (type) {
      form.type = type
    }

    if (form.projectId && form.type) {
      await loadExistingDocument(Number(form.projectId), form.type)
    }
    if (!currentDocument.value) {
      applyTemplateIfEmpty()
    }
  } finally {
    loading.value = false
  }
}

async function loadExistingDocument(projectId: number, type: string) {
  try {
    const res = await getProjectDocument(projectId, type)
    fillDocument(res.data)
  } catch {
    const list = await getDocuments({ projectId, type, page: 1, pageSize: 1 })
    const firstDocument = list.data.list[0]
    if (firstDocument) {
      fillDocument(firstDocument)
      return
    }
    currentDocument.value = null
  }
}

function fillDocument(document: DocumentItem) {
  currentDocument.value = document
  form.title = document.title
  form.projectId = String(document.projectId)
  form.type = document.type
  form.summary = document.summary
  form.status = document.status === 'published' ? 'published' : 'draft'
  form.content = isHtmlContent(document.content) ? document.content : markdownToHtml(document.content)
  syncEditorHtml()
}

function applyTemplateIfEmpty() {
  if (form.title || form.content) {
    return
  }
  form.title = `${selectedProjectName.value}${selectedDocTypeLabel.value}`
  form.summary = `${selectedDocTypeLabel.value}待补充`
  form.content = documentTemplate(form.type)
}

async function saveDraft() {
  await save('draft')
}

async function publishDocument() {
  await save('published')
}

async function save(status: 'draft' | 'published') {
  message.value = ''
  errorMessage.value = ''
  if (!form.projectId) {
    errorMessage.value = '请选择所属项目'
    return
  }
  if (!form.title.trim()) {
    errorMessage.value = '文档标题不能为空'
    return
  }

  saving.value = true
  try {
    const res = await saveDocument({
      projectId: Number(form.projectId),
      type: form.type,
      title: form.title.trim(),
      summary: form.summary.trim() || buildSummary(form.content),
      content: form.content,
      status,
    })
    fillDocument(res.data)
    message.value = status === 'published' ? '文档已发布' : '草稿已保存'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '保存文档失败'
  } finally {
    saving.value = false
  }
}

function buildSummary(content: string) {
  const plainText = isHtmlContent(content)
    ? stripHtml(content)
    : content
        .split('\n')
        .map((item) => item.replace(/^#+\s*/, '').replace(/^[-*]\s*/, '').trim())
        .find(Boolean)
  return (plainText || '暂无摘要').slice(0, 120)
}

function documentTemplate(type: string) {
  const title = selectedDocTypeLabel.value
  const projectName = selectedProjectName.value
  if (type === 'environment') {
    return `<h1>${title}</h1>
<h2>测试环境</h2>
<ul>
  <li>测试地址：</li>
  <li>测试账号：</li>
  <li>联调注意事项：</li>
</ul>
<h2>线上环境</h2>
<ul>
  <li>线上地址：</li>
  <li>监控入口：</li>
  <li>告警联系人：</li>
</ul>
<h2>常见问题</h2>
<ul>
  <li></li>
</ul>`
  }
  if (type === 'deploy') {
    return `<h1>${title}</h1>
<h2>发布前检查</h2>
<ul>
  <li>分支：</li>
  <li>配置：</li>
  <li>依赖服务：</li>
</ul>
<h2>发布步骤</h2>
<ol>
  <li></li>
  <li></li>
  <li></li>
</ol>
<h2>回滚方案</h2>
<ul>
  <li></li>
</ul>`
  }
  if (type === 'api') {
    return `<h1>${title}</h1>
<h2>接口文档入口</h2>
<ul>
  <li>API 文档：</li>
  <li>鉴权方式：</li>
  <li>联调环境：</li>
</ul>
<h2>联调约定</h2>
<ul>
  <li></li>
</ul>`
  }
  return `<h1>${projectName}${title}</h1>
<h2>背景</h2>
<p></p>
<h2>范围</h2>
<p></p>
<h2>维护说明</h2>
<ul>
  <li></li>
</ul>`
}

function mountWangEditor() {
  if (!editorContainerRef.value || !toolbarContainerRef.value || editorRef.value) {
    return
  }

  const editor = createEditor({
    selector: editorContainerRef.value,
    html: form.content || '<p><br></p>',
    config: editorConfig,
    mode: 'default',
  })
  const toolbar = createToolbar({
    editor,
    selector: toolbarContainerRef.value,
    config: toolbarConfig,
    mode: 'default',
  })

  editorRef.value = editor
  toolbarRef.value = toolbar
}

function syncEditorHtml() {
  const editor = editorRef.value
  if (!editor || editor.isDestroyed || editor.getHtml() === form.content) {
    return
  }
  isSyncingEditor.value = true
  editor.setHtml(form.content || '<p><br></p>')
  isSyncingEditor.value = false
}

</script>
