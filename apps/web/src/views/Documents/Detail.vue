<template>
  <PageHeader
    eyebrow="文档详情"
    :title="document?.title || '文档详情'"
    :description="document?.summary || '阅读项目文档，查看目录、属性和关联项目。'"
  >
    <template #actions>
      <Button variant="outline" @click="goDocuments">
        <ArrowLeft class="button-icon" />
        返回文档库
      </Button>
      <Button v-if="document" variant="outline" @click="goProject">
        <FolderKanban class="button-icon" />
        查看项目
      </Button>
      <Button v-if="document" variant="outline" @click="openVersionHistory">
        <History class="button-icon" />
        版本历史
      </Button>
      <Button v-if="document" @click="goEditor">
        <FilePenLine class="button-icon" />
        编辑文档
      </Button>
      <Button v-if="document" variant="outline" @click="toggleDocumentFavorite">
        <Heart :class="['button-icon', { 'is-filled': isCurrentDocumentFavorite }]" />
        {{ isCurrentDocumentFavorite ? '已收藏' : '收藏' }}
      </Button>
      <Button v-if="document" variant="destructive" @click="deleteDialogOpen = true">
        <Trash2 class="button-icon" />
        删除
      </Button>
    </template>
  </PageHeader>

  <div v-if="errorMessage" class="notice notice-error">{{ errorMessage }}</div>

  <DetailPageSkeleton v-if="visibleLoading && !document" />

  <EmptyState
    v-else-if="!loading && !document"
    :icon="BookOpenText"
    title="文档不存在"
    description="可能文档已被删除，或当前账号没有访问权限。"
  >
    <Button variant="outline" @click="goDocuments">
      <ArrowLeft class="button-icon" />
      返回文档库
    </Button>
  </EmptyState>

  <template v-else-if="document">
    <section class="document-summary-strip" aria-label="文档概览">
      <div class="document-summary-main">
        <div class="document-meta-row">
          <Badge variant="secondary">{{ document?.typeLabel || '文档' }}</Badge>
          <Badge v-if="document" variant="secondary" :class="['ui-badge', statusClass]">
            {{ document.statusLabel }}
          </Badge>
          <Badge v-if="isStale" variant="secondary" class="ui-badge ui-badge--warning">
            可能过期
          </Badge>
        </div>
      </div>
      <dl class="document-summary-meta">
        <div>
          <dt>所属项目</dt>
          <dd>{{ project?.name || projectName }}</dd>
        </div>
        <div>
          <dt>更新时间</dt>
          <dd>{{ formatDate(document?.updatedAt || '') }}</dd>
        </div>
      </dl>
    </section>

    <div class="document-detail-layout">
      <article class="document-reader-card">
        <div
          v-if="isDocumentHtml"
          class="preview-document-body document-reader-body rich-document-body"
          v-html="renderedDocumentHtml"
        />
        <div v-else class="preview-document-body document-reader-body">
          <template v-for="(block, index) in contentBlocks" :key="index">
            <h2 v-if="block.type === 'h1'" :id="headingId(index)">{{ block.text }}</h2>
            <h3 v-else-if="block.type === 'h2'" :id="headingId(index)">{{ block.text }}</h3>
            <h4 v-else-if="block.type === 'h3'" :id="headingId(index)">{{ block.text }}</h4>
            <ul v-else-if="block.type === 'list'">
              <li v-for="item in block.items" :key="item">{{ item }}</li>
            </ul>
            <pre v-else-if="block.type === 'code'"><code>{{ block.text }}</code></pre>
            <figure v-else-if="block.type === 'image'" class="preview-image">
              <img :src="block.url" :alt="block.alt" />
              <figcaption>{{ block.alt }}</figcaption>
            </figure>
            <p v-else>{{ block.text }}</p>
          </template>
        </div>
      </article>

      <aside class="document-aside">
        <Card>
          <CardHeader>
            <CardTitle>文档属性</CardTitle>
            <CardDescription>用于确认来源、状态和维护时间。</CardDescription>
          </CardHeader>
          <CardContent>
            <div class="property-list">
              <div>
                <span>所属项目</span>
                <strong>{{ project?.name || projectName }}</strong>
              </div>
              <div>
                <span>文档类型</span>
                <strong>{{ document?.typeLabel || '-' }}</strong>
              </div>
              <div>
                <span>文档状态</span>
                <strong>{{ document?.statusLabel || '-' }}</strong>
              </div>
              <div>
                <span>最后更新</span>
                <strong>{{ formatDate(document?.updatedAt || '') }}</strong>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>目录</CardTitle>
            <CardDescription>根据正文标题自动生成。</CardDescription>
          </CardHeader>
          <CardContent>
            <div v-if="tocItems.length > 0" class="document-toc">
              <a
                v-for="item in tocItems"
                :key="item.id"
                :href="`#${item.id}`"
                :class="`toc-level-${item.level}`"
              >
                {{ item.text }}
              </a>
            </div>
            <div v-else class="doc-placeholder compact">
              <ListTree class="doc-placeholder-icon" />
              <div>
                <strong>暂无目录</strong>
                <p>给正文添加标题后会自动生成目录。</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </aside>
    </div>
  </template>

  <Dialog v-model:open="versionDialogOpen">
    <DialogContent class="version-dialog-content">
      <DialogHeader>
        <DialogTitle>版本历史</DialogTitle>
        <DialogDescription>
          查看这篇文档每次保存时生成的内容快照。
        </DialogDescription>
      </DialogHeader>

      <div v-if="versionError" class="notice notice-error">{{ versionError }}</div>
      <div v-if="versionMessage" class="notice">{{ versionMessage }}</div>

      <div class="version-history-layout">
        <div class="version-list-panel">
          <div v-if="versionsLoading" class="detail-loading">加载版本中...</div>
          <EmptyState
            v-else-if="versions.length === 0"
            :icon="History"
            title="暂无版本记录"
            description="下次保存文档后会生成版本记录。"
          />
          <template v-else>
            <button
              v-for="version in versions"
              :key="version.id"
              type="button"
              :class="['version-item', { 'is-active': selectedVersion?.id === version.id }]"
              @click="selectVersion(version.id)"
            >
              <span>v{{ version.versionNo }}</span>
              <strong>{{ version.changeSummary || '保存文档' }}</strong>
              <small>{{ formatDate(version.createdAt) }}</small>
            </button>
          </template>
        </div>

        <div class="version-preview-panel">
          <div v-if="selectedVersion" class="version-preview-head">
            <div>
              <div class="document-meta-row">
                <Badge variant="secondary">v{{ selectedVersion.versionNo }}</Badge>
                <Badge variant="secondary">{{ selectedVersion.typeLabel }}</Badge>
              </div>
              <h3>{{ selectedVersion.title }}</h3>
              <p>{{ selectedVersion.summary || '这个版本没有摘要。' }}</p>
            </div>
            <div class="version-preview-actions">
              <span>恢复会生成一个新的版本记录</span>
              <Button
                variant="outline"
                size="sm"
                :disabled="restoringVersion || isSelectedCurrentVersion"
                @click="restoreSelectedVersion"
              >
                <History class="button-icon" />
                {{ restoringVersion ? '恢复中...' : isSelectedCurrentVersion ? '当前版本' : '恢复到此版本' }}
              </Button>
            </div>
          </div>
          <div
            v-if="selectedVersion && isSelectedVersionHtml"
            class="preview-document-body compact version-preview-body rich-document-body"
            v-html="selectedVersion.content"
          />
          <div v-else-if="selectedVersion" class="preview-document-body compact version-preview-body">
            <template v-for="(block, index) in selectedVersionBlocks" :key="index">
              <h2 v-if="block.type === 'h1'">{{ block.text }}</h2>
              <h3 v-else-if="block.type === 'h2'">{{ block.text }}</h3>
              <h4 v-else-if="block.type === 'h3'">{{ block.text }}</h4>
              <ul v-else-if="block.type === 'list'">
                <li v-for="item in block.items" :key="item">{{ item }}</li>
              </ul>
              <pre v-else-if="block.type === 'code'"><code>{{ block.text }}</code></pre>
              <figure v-else-if="block.type === 'image'" class="preview-image">
                <img :src="block.url" :alt="block.alt" />
                <figcaption>{{ block.alt }}</figcaption>
              </figure>
              <p v-else>{{ block.text }}</p>
            </template>
          </div>
          <div v-else class="doc-placeholder">
            <History class="doc-placeholder-icon" />
            <div>
              <strong>选择一个版本</strong>
              <p>点击左侧版本后会在这里预览当时的文档内容。</p>
            </div>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>

  <Dialog v-model:open="deleteDialogOpen">
    <DialogContent class="delete-dialog-content">
      <DialogHeader>
        <DialogTitle>删除文档</DialogTitle>
        <DialogDescription>
          删除后文档会进入回收站，普通文档库和项目详情中将不再展示。
        </DialogDescription>
      </DialogHeader>

      <div class="delete-warning-box">
        <Trash2 class="delete-warning-icon" />
        <div>
          <strong>{{ document?.title || '当前文档' }}</strong>
          <p>确认删除这篇文档吗？后续可以在回收站恢复。</p>
        </div>
      </div>

      <div v-if="deleteError" class="notice notice-error">{{ deleteError }}</div>

      <DialogFooter>
        <Button variant="outline" type="button" :disabled="deletingDocument" @click="deleteDialogOpen = false">
          取消
        </Button>
        <Button variant="destructive" type="button" :disabled="deletingDocument" @click="confirmDeleteDocument">
          <Trash2 class="button-icon" />
          {{ deletingDocument ? '删除中...' : '确认删除' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  BookOpenText,
  FilePenLine,
  FolderKanban,
  Heart,
  History,
  ListTree,
  Trash2,
} from 'lucide-vue-next'
import {
  deleteDocument,
  getDocument,
  getDocumentVersion,
  getDocumentVersions,
  restoreDocumentVersion,
  type DocumentItem,
  type DocumentVersionItem,
} from '@/api/documents'
import { getProject, type Project } from '@/api/projects'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import DetailPageSkeleton from '@/components/ui/DetailPageSkeleton.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useDelayedLoading } from '@/composables/use-delayed-loading'
import { usePersonalRecords } from '@/composables/use-personal-records'
import {
  addHeadingIdsToHtml,
  extractHtmlToc,
  extractMarkdownToc,
  isHtmlContent,
  parseMarkdown,
} from '@/utils/markdown'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const visibleLoading = useDelayedLoading(loading)
const errorMessage = ref('')
const document = ref<DocumentItem | null>(null)
const project = ref<Project | null>(null)
const versionDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const versionsLoading = ref(false)
const restoringVersion = ref(false)
const deletingDocument = ref(false)
const versionError = ref('')
const versionMessage = ref('')
const deleteError = ref('')
const versions = ref<DocumentVersionItem[]>([])
const selectedVersion = ref<DocumentVersionItem | null>(null)
const { addRecentVisit, isFavorite, toggleFavorite } = usePersonalRecords()

const documentId = computed(() => Number(route.params.id))
const isCurrentDocumentFavorite = computed(() => {
  return document.value ? isFavorite('document', document.value.id) : false
})
const isDocumentHtml = computed(() => isHtmlContent(document.value?.content || ''))
const renderedDocumentHtml = computed(() => addHeadingIdsToHtml(document.value?.content || ''))
const contentBlocks = computed(() => parseMarkdown(document.value?.content || ''))
const isSelectedVersionHtml = computed(() => isHtmlContent(selectedVersion.value?.content || ''))
const selectedVersionBlocks = computed(() => parseMarkdown(selectedVersion.value?.content || ''))
const isSelectedCurrentVersion = computed(() => {
  if (!document.value || !selectedVersion.value) return false
  return selectedVersion.value.versionNo === document.value.currentVersionNo
})
const projectName = computed(() => {
  if (!document.value) return '-'
  return `项目 #${document.value.projectId}`
})
const statusClass = computed(() => {
  if (document.value?.status === 'published') return 'ui-badge--success'
  return 'ui-badge--default'
})
const isStale = computed(() => {
  if (!document.value?.updatedAt) return false
  const updatedAt = new Date(document.value.updatedAt).getTime()
  if (Number.isNaN(updatedAt)) return false
  const ninetyDays = 90 * 24 * 60 * 60 * 1000
  return Date.now() - updatedAt > ninetyDays
})
const tocItems = computed(() => {
  if (isDocumentHtml.value) {
    return extractHtmlToc(document.value?.content || '')
  }
  return extractMarkdownToc(contentBlocks.value)
})

onMounted(() => {
  loadDocument()
})

async function loadDocument() {
  if (!Number.isFinite(documentId.value) || documentId.value <= 0) {
    errorMessage.value = '无效的文档 ID'
    document.value = null
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getDocument(documentId.value)
    document.value = res.data
    await loadProject(res.data.projectId)
    addRecentVisit(documentRecordPayload(res.data))
  } catch (error) {
    document.value = null
    project.value = null
    errorMessage.value = error instanceof Error ? error.message : '获取文档详情失败'
  } finally {
    loading.value = false
  }
}

async function loadProject(projectId: number) {
  try {
    const res = await getProject(projectId)
    project.value = res.data
  } catch {
    project.value = null
  }
}

function goDocuments() {
  router.push({ name: 'Documents' })
}

function goProject() {
  if (!document.value) return
  router.push({ name: 'ProjectDetail', params: { id: document.value.projectId } })
}

function goEditor() {
  if (!document.value) return
  router.push({
    path: '/documents/editor',
    query: {
      projectId: String(document.value.projectId),
      type: document.value.type,
    },
  })
}

function toggleDocumentFavorite() {
  if (!document.value) return
  toggleFavorite(documentRecordPayload(document.value))
}

async function openVersionHistory() {
  versionDialogOpen.value = true
  await loadVersions()
}

async function loadVersions() {
  if (!document.value) return
  versionsLoading.value = true
  versionError.value = ''
  try {
    const res = await getDocumentVersions(document.value.id)
    versions.value = res.data.list
    if (versions.value[0]) {
      await selectVersion(versions.value[0].id)
    } else {
      selectedVersion.value = null
    }
  } catch (error) {
    versionError.value = error instanceof Error ? error.message : '获取版本历史失败'
  } finally {
    versionsLoading.value = false
  }
}

async function selectVersion(versionId: number) {
  if (!document.value) return
  versionError.value = ''
  try {
    const res = await getDocumentVersion(document.value.id, versionId)
    selectedVersion.value = res.data
  } catch (error) {
    versionError.value = error instanceof Error ? error.message : '获取版本内容失败'
  }
}

async function restoreSelectedVersion() {
  if (!document.value || !selectedVersion.value || isSelectedCurrentVersion.value) return
  restoringVersion.value = true
  versionError.value = ''
  versionMessage.value = ''
  try {
    const versionNo = selectedVersion.value.versionNo
    const res = await restoreDocumentVersion(document.value.id, selectedVersion.value.id, {
      changeSummary: `恢复到 v${versionNo}`,
    })
    document.value = res.data
    versionMessage.value = `已恢复到 v${versionNo}，并生成 v${res.data.currentVersionNo}`
    await loadVersions()
  } catch (error) {
    versionError.value = error instanceof Error ? error.message : '恢复版本失败'
  } finally {
    restoringVersion.value = false
  }
}

async function confirmDeleteDocument() {
  if (!document.value) return
  deletingDocument.value = true
  deleteError.value = ''
  try {
    await deleteDocument(document.value.id)
    deleteDialogOpen.value = false
    router.push({ name: 'Documents' })
  } catch (error) {
    deleteError.value = error instanceof Error ? error.message : '删除文档失败'
  } finally {
    deletingDocument.value = false
  }
}

function headingId(index: number) {
  return `doc-heading-${index}`
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function documentRecordPayload(item: DocumentItem) {
  return {
    kind: 'document' as const,
    targetId: item.id,
    title: item.title,
    projectName: project.value?.name || `项目 #${item.projectId}`,
    description: item.summary || '暂无摘要',
    path: `/documents/${item.id}`,
    badge: item.typeLabel,
  }
}
</script>
