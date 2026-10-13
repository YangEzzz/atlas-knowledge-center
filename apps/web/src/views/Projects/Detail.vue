<template>
  <PageHeader
    :title="project?.name || '项目详情'"
    :description="project?.description || '查看项目基础信息、负责人、核心入口和项目文档分区。'"
  >
    <template #actions>
      <Button variant="outline" @click="goBack">
        <ArrowLeft class="button-icon" />
        返回项目库
      </Button>
      <Button variant="outline" :disabled="loading" @click="loadProject">
        <RefreshCw :class="['button-icon', { 'is-spinning': visibleLoading }]" />
        刷新
      </Button>
      <Button v-if="project" @click="openProjectEditor">
        <PencilLine class="button-icon" />
        编辑项目
      </Button>
      <Button v-if="project" variant="outline" @click="toggleProjectFavorite">
        <Heart :class="['button-icon', { 'is-filled': isCurrentProjectFavorite }]" />
        {{ isCurrentProjectFavorite ? '已收藏' : '收藏' }}
      </Button>
    </template>
  </PageHeader>

  <div v-if="errorMessage" class="notice notice-error">{{ errorMessage }}</div>

  <DetailPageSkeleton v-if="visibleLoading && !project" />

  <EmptyState
    v-else-if="!loading && !project"
    :icon="FolderKanban"
    title="项目不存在"
    description="可能项目已被删除，或当前账号没有访问权限。"
  >
    <Button variant="outline" @click="goBack">
      <ArrowLeft class="button-icon" />
      返回项目库
    </Button>
  </EmptyState>

  <template v-else-if="project">
    <section v-if="project" class="project-command-panel" aria-label="项目概览">
      <div class="project-command-main">
        <div class="document-meta-row">
          <Badge variant="secondary">{{ typeLabel(project.type) }}</Badge>
          <Badge variant="secondary" :class="['ui-badge', badgeClass(projectStatus(project))]">
            {{ projectStatus(project) }}
          </Badge>
          <Badge variant="outline">{{ documentCompletion.done }}/{{ documentCompletion.total }} 文档</Badge>
        </div>
        <p>{{ project.description || '这个项目还没有补充介绍。' }}</p>
        <div class="badge-row">
          <Badge v-for="keyword in project.keywords" :key="keyword" variant="outline">
            {{ keyword }}
          </Badge>
          <span v-if="project.keywords.length === 0" class="muted-inline">暂无关键词</span>
        </div>
      </div>
      <dl class="project-command-meta">
        <div>
          <dt>项目入口</dt>
          <dd>{{ configuredEntryCount }}/{{ entryDisplayItems.length }}</dd>
        </div>
        <div>
          <dt>负责人</dt>
          <dd>{{ ownerCompletion.done }}/{{ ownerCompletion.total }}</dd>
        </div>
        <div>
          <dt>最后更新</dt>
          <dd>{{ formatDate(project.updatedAt) }}</dd>
        </div>
      </dl>
    </section>

    <div class="project-home-layout">
      <main class="project-home-main">
        <Card class="detail-section-card project-entry-section">
          <CardHeader class="ui-card-header">
            <div>
              <CardTitle>核心入口</CardTitle>
              <CardDescription>研发最常用的环境、代码、原型、API 和运维入口。</CardDescription>
            </div>
            <Badge
              variant="secondary"
              :class="['ui-badge', missingEntryCount > 0 ? 'ui-badge--warning' : 'ui-badge--success']"
            >
              {{ missingEntryCount > 0 ? `${missingEntryCount} 项待补` : '已补全' }}
            </Badge>
          </CardHeader>
          <CardContent>
            <div class="project-entry-grid">
              <article
                v-for="entryItem in entryDisplayItems"
                :key="entryItem.key"
                :class="['project-entry-tile', { 'is-empty': !entryItem.entry }]"
              >
                <div class="project-entry-icon">
                  <component :is="entryItem.icon" />
                </div>
                <div class="project-entry-content">
                  <div class="project-entry-title">{{ entryItem.label }}</div>
                  <a
                    v-if="entryItem.entry"
                    class="project-entry-url"
                    :href="entryItem.entry.url"
                    target="_blank"
                    rel="noreferrer"
                  >
                    {{ entryItem.entry.url }}
                  </a>
                  <span v-else class="project-entry-url">暂未配置</span>
                </div>
                <Button
                  v-if="entryItem.entry"
                  as="a"
                  :href="entryItem.entry.url"
                  target="_blank"
                  rel="noreferrer"
                  variant="outline"
                  size="sm"
                >
                  <ExternalLink class="button-icon" />
                  打开
                </Button>
                <Badge v-else variant="secondary" class="ui-badge ui-badge--warning">待补</Badge>
              </article>
            </div>
          </CardContent>
        </Card>

        <Card class="detail-section-card">
          <CardHeader class="ui-card-header">
            <div>
              <CardTitle>项目文档</CardTitle>
              <CardDescription>按研发常用场景组织，快速查看或补全文档。</CardDescription>
            </div>
            <Badge variant="secondary">{{ documentCompletion.done }}/{{ documentCompletion.total }}</Badge>
          </CardHeader>
          <CardContent>
            <div class="project-doc-map">
              <article
                v-for="section in docSections"
                :key="section.value"
                :class="['project-doc-card', { 'is-empty': !documentByType(section.value) }]"
              >
                <div class="project-doc-card-head">
                  <div>
                    <div class="project-doc-title">{{ section.title }}</div>
                    <p>{{ section.description }}</p>
                  </div>
                  <Badge
                    v-if="documentByType(section.value)"
                    variant="secondary"
                    :class="['ui-badge', documentStatusClass(documentByType(section.value)?.status)]"
                  >
                    {{ documentByType(section.value)?.statusLabel }}
                  </Badge>
                  <Badge v-else variant="secondary" class="ui-badge ui-badge--warning">待编写</Badge>
                </div>
                <p class="project-doc-excerpt">
                  {{ documentExcerpt(documentByType(section.value), section.emptyDescription) }}
                </p>
                <div class="project-doc-footer">
                  <span>
                    {{ documentByType(section.value) ? `更新于 ${formatDate(documentByType(section.value)?.updatedAt || '')}` : '暂无更新时间' }}
                  </span>
                  <div class="section-actions">
                    <Button
                      v-if="documentByType(section.value)"
                      variant="secondary"
                      size="sm"
                      @click="goViewDocument(documentByType(section.value)?.id)"
                    >
                      <Eye class="button-icon" />
                      查看
                    </Button>
                    <Button variant="outline" size="sm" @click="goEditDocument(section.value)">
                      <FilePenLine class="button-icon" />
                      {{ documentByType(section.value) ? '编辑' : '新建' }}
                    </Button>
                  </div>
                </div>
              </article>
            </div>
          </CardContent>
        </Card>
      </main>

      <aside class="project-home-aside">
        <Card>
          <CardHeader>
            <CardTitle>负责人</CardTitle>
            <CardDescription>当前维护责任人。</CardDescription>
          </CardHeader>
          <CardContent>
            <div class="owner-detail-grid">
              <div v-for="group in ownerGroups" :key="group.value" class="owner-detail-item">
                <span>{{ group.label }}</span>
                <strong>{{ group.names }}</strong>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>维护清单</CardTitle>
            <CardDescription>补齐后项目页会更可靠。</CardDescription>
          </CardHeader>
          <CardContent>
            <div class="project-checklist">
              <div v-for="item in projectChecklist" :key="item.label" :class="{ 'is-done': item.done }">
                <component :is="item.done ? CheckCircle2 : AlertCircle" class="project-check-icon" />
                <span>{{ item.label }}</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </aside>
    </div>
  </template>

  <ProjectFormDialog
    v-model:open="projectDialogOpen"
    :project="project"
    @saved="handleProjectSaved"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  AlertCircle,
  Activity,
  ArrowLeft,
  Braces,
  CheckCircle2,
  ExternalLink,
  FilePenLine,
  FileText,
  FolderKanban,
  GitBranch,
  Globe2,
  Heart,
  KeyRound,
  Layers3,
  Link2,
  PencilLine,
  RefreshCw,
  Eye,
  Rocket,
  ScrollText,
  Server,
  ShieldCheck,
} from 'lucide-vue-next'
import { getDocuments, type DocumentItem } from '@/api/documents'
import { getProject, type Project, type ProjectEntry, type ProjectOwner } from '@/api/projects'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import ProjectFormDialog from '@/components/project/ProjectFormDialog.vue'
import DetailPageSkeleton from '@/components/ui/DetailPageSkeleton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useDelayedLoading } from '@/composables/use-delayed-loading'
import { usePersonalRecords } from '@/composables/use-personal-records'
import { stripHtml } from '@/utils/markdown'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const visibleLoading = useDelayedLoading(loading)
const errorMessage = ref('')
const project = ref<Project | null>(null)
const documents = ref<DocumentItem[]>([])
const projectDialogOpen = ref(false)
const { addRecentVisit, isFavorite, toggleFavorite } = usePersonalRecords()

const projectId = computed(() => Number(route.params.id))
const isCurrentProjectFavorite = computed(() => {
  return project.value ? isFavorite('project', project.value.id) : false
})

const coreEntryTypes = [
  { label: '线上地址', value: 'production', icon: Globe2 },
  { label: '测试地址', value: 'staging', icon: Server },
  { label: 'GitLab', value: 'gitlab', icon: GitBranch },
]

const entryTypeMeta = [
  ...coreEntryTypes,
  { label: '原型地址', value: 'prototype', icon: Layers3 },
  { label: 'API 文档', value: 'api_doc', icon: Braces },
  { label: '后台地址', value: 'admin', icon: ShieldCheck },
  { label: '监控地址', value: 'monitor', icon: Activity },
  { label: '日志地址', value: 'logs', icon: ScrollText },
  { label: '发布系统', value: 'deploy', icon: Rocket },
  { label: '账号权限', value: 'access', icon: KeyRound },
  { label: '其他', value: 'other', icon: Link2 },
]

const projectTypes = [
  { label: '用户端', value: 'user_client' },
  { label: '后台系统', value: 'admin' },
  { label: '前端应用', value: 'frontend' },
  { label: '后端服务', value: 'backend' },
  { label: '官网', value: 'website' },
  { label: '移动端', value: 'mobile' },
  { label: '小程序', value: 'mini_program' },
  { label: '公共库/组件', value: 'shared_lib' },
  { label: '其他', value: 'other' },
]

const docSections = [
  {
    value: 'intro',
    title: '项目介绍',
    description: '记录项目定位、业务范围和核心能力。',
    emptyTitle: '项目介绍还没有编写',
    emptyDescription: '后续会接入项目文档编辑器，在这里维护富文本内容。',
  },
  {
    value: 'environment',
    title: '环境说明',
    description: '整理线上、测试、预发、本地开发等环境差异。',
    emptyTitle: '环境说明还没有编写',
    emptyDescription: '核心入口已经可以维护，完整环境说明后续放到这里。',
  },
  {
    value: 'deploy',
    title: '部署说明',
    description: '记录构建、发布、回滚和部署注意事项。',
    emptyTitle: '部署说明还没有编写',
    emptyDescription: '可以先在项目文档库里补充，后续在详情页关联。',
  },
  {
    value: 'api',
    title: 'API 说明',
    description: '关联接口文档、联调方式和鉴权规则。',
    emptyTitle: 'API 说明还没有编写',
    emptyDescription: '后续会支持关联 API 文档入口或直接维护富文本内容。',
  },
  {
    value: 'faq',
    title: 'FAQ',
    description: '沉淀常见问题、排障路径和新人踩坑点。',
    emptyTitle: 'FAQ 还没有编写',
    emptyDescription: '这里会成为新人和跨团队协作时最常看的部分。',
  },
]

const entryDisplayItems = computed(() => {
  const item = project.value
  const extraEntries =
    item?.entries
      ?.filter((entry) => !coreEntryTypes.some((entryType) => entryType.value === entry.type))
      .map((entry, index) => ({
        key: `entry-${entry.id || entry.type}-${index}`,
        label: entry.title || entryTypeLabel(entry.type),
        value: entry.type,
        icon: entryTypeIcon(entry.type),
        entry,
      })) || []

  const coreEntries = coreEntryTypes.map((entryType) => ({
    ...entryType,
    key: `core-${entryType.value}`,
    entry: item ? entryByType(item, entryType.value) : undefined,
  }))
  return [...coreEntries, ...extraEntries]
})

const configuredEntryCount = computed(() => {
  return entryDisplayItems.value.filter((item) => Boolean(item.entry)).length
})

const missingEntryCount = computed(() => {
  return entryDisplayItems.value.length - configuredEntryCount.value
})

const documentCompletion = computed(() => {
  return {
    done: docSections.filter((section) => Boolean(documentByType(section.value))).length,
    total: docSections.length,
  }
})

const ownerGroups = computed(() => {
  const owners = project.value?.owners
  return [
    { label: '前端', value: 'frontend', names: ownerNames(owners?.frontend) },
    { label: '后端', value: 'backend', names: ownerNames(owners?.backend) },
    { label: '测试', value: 'tester', names: ownerNames(owners?.tester) },
  ]
})

const ownerCompletion = computed(() => {
  const groups = ownerGroups.value
  return {
    done: groups.filter((group) => group.names !== '-').length,
    total: groups.length,
  }
})

const projectChecklist = computed(() => {
  const item = project.value
  return [
    { label: '项目介绍已填写', done: Boolean(item?.description) },
    { label: '负责人已配置', done: ownerCompletion.value.done === ownerCompletion.value.total },
    { label: '核心入口已配置', done: missingEntryCount.value === 0 },
    { label: '基础文档已维护', done: documentCompletion.value.done >= 3 },
  ]
})

onMounted(() => {
  loadProject()
})

async function loadProject() {
  if (!Number.isFinite(projectId.value) || projectId.value <= 0) {
    errorMessage.value = '无效的项目 ID'
    project.value = null
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getProject(projectId.value)
    project.value = res.data
    await loadDocuments()
    addRecentVisit(projectRecordPayload(res.data))
  } catch (error) {
    project.value = null
    documents.value = []
    errorMessage.value = error instanceof Error ? error.message : '获取项目详情失败'
  } finally {
    loading.value = false
  }
}

async function loadDocuments() {
  const res = await getDocuments({
    projectId: projectId.value,
    page: 1,
    pageSize: 100,
  })
  documents.value = res.data.list
}

function goBack() {
  router.push({ name: 'Projects' })
}

function openProjectEditor() {
  projectDialogOpen.value = true
}

async function handleProjectSaved(savedProject: Project) {
  project.value = savedProject
  projectDialogOpen.value = false
  addRecentVisit(projectRecordPayload(savedProject))
  await loadDocuments()
}

function toggleProjectFavorite() {
  if (!project.value) return
  toggleFavorite(projectRecordPayload(project.value))
}

function goEditDocument(type: string) {
  router.push({
    path: '/documents/editor',
    query: {
      projectId: String(projectId.value),
      type,
    },
  })
}

function goViewDocument(id?: number) {
  if (!id) return
  router.push({ name: 'DocumentDetail', params: { id } })
}

function documentByType(type: string) {
  return documents.value.find((document) => document.type === type)
}

function documentExcerpt(document: DocumentItem | undefined, fallback: string) {
  if (!document) return fallback
  const text = document.summary || stripHtml(document.content || '')
  return (text || '这篇文档还没有摘要。').slice(0, 96)
}

function entryByType(item: Project, type: string) {
  return item.entries?.find((entry: ProjectEntry) => entry.type === type && entry.url)
}

function entryTypeLabel(type: string) {
  return entryTypeMeta.find((entryType) => entryType.value === type)?.label || type
}

function entryTypeIcon(type: string) {
  return entryTypeMeta.find((entryType) => entryType.value === type)?.icon || Link2
}

function ownerNames(owners?: ProjectOwner[] | null) {
  if (!owners || owners.length === 0) {
    return '-'
  }
  return owners.map((owner) => owner.ownerName).join('、')
}

function typeLabel(type: string) {
  return projectTypes.find((item) => item.value === type)?.label || type
}

function projectStatus(item: Project) {
  const hasAnyOwner =
    Boolean(item.owners?.frontend?.length) ||
    Boolean(item.owners?.backend?.length) ||
    Boolean(item.owners?.tester?.length)
  if (!hasAnyOwner || !item.description) {
    return '需补充'
  }
  if (!item.entries?.length) {
    return '待补入口'
  }
  return '正常'
}

function badgeClass(status: string) {
  if (status === '正常') return 'ui-badge--success'
  if (status === '待补入口') return 'ui-badge--warning'
  return 'ui-badge--default'
}

function documentStatusClass(status?: string) {
  if (status === 'published') return 'ui-badge--success'
  return 'ui-badge--default'
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function projectRecordPayload(item: Project) {
  return {
    kind: 'project' as const,
    targetId: item.id,
    title: item.name,
    description: item.description || '暂无项目介绍',
    path: `/projects/${item.id}`,
    badge: typeLabel(item.type),
  }
}

</script>
