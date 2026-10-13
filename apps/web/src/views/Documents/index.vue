<template>
  <PageHeader title="文档库" description="跨项目查找富文本文档，支持环境、部署、API、账号、FAQ 和开发规范等内容。">
    <template #actions>
      <Button @click="goEditor">
        <FilePenLine class="button-icon" />
        新建文档
      </Button>
      <Button variant="outline">
        <History class="button-icon" />
        查看记录
      </Button>
    </template>
  </PageHeader>

  <Card class="document-workbench-card">
    <CardContent>
      <div class="document-type-strip" aria-label="文档类型筛选">
        <button
          type="button"
          :class="['document-type-chip', { 'is-active': !filters.type }]"
          @click="filters.type = ''"
        >
          <span>全部</span>
          <strong>{{ rows.length }}</strong>
        </button>
        <button
          v-for="item in docTypeCards"
          :key="item.value"
          type="button"
          :class="['document-type-chip', { 'is-active': filters.type === item.value }]"
          @click="filters.type = filters.type === item.value ? '' : item.value"
        >
          <component :is="item.icon" class="document-type-chip-icon" />
          <span>{{ item.name }}</span>
          <strong>{{ item.count }}</strong>
        </button>
      </div>

      <div class="document-filter-bar">
        <Input v-model="filters.keyword" placeholder="搜索标题、正文、项目、负责人" @keyup.enter="loadDocuments" />
        <Select v-model="selectedDocType">
          <SelectTrigger class="ui-select-trigger" aria-label="文档类型">
            <SelectValue placeholder="全部类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="item in docTypeOptions" :key="item.value" :value="selectValue(item.value)">
              {{ item.label }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Select v-model="selectedStatus">
          <SelectTrigger class="ui-select-trigger" aria-label="文档状态">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="item in statusOptions" :key="item.value" :value="selectValue(item.value)">
              {{ item.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </CardContent>
  </Card>

  <Card class="document-list-card">
    <CardHeader class="ui-card-header">
      <div>
        <CardTitle>文档列表</CardTitle>
        <CardDescription>{{ filteredRows.length }} 篇文档，按最近更新时间排序。</CardDescription>
      </div>
      <CardAction class="document-list-actions">
        <Button variant="outline" size="sm" :disabled="loading" class="document-refresh-button" @click="loadDocuments">
          <RefreshCw :class="['button-icon', { 'is-spinning': visibleLoading }]" />
          刷新
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <div v-if="errorMessage" class="notice notice-error">{{ errorMessage }}</div>
      <EmptyState
        v-if="!loading && filteredRows.length === 0"
        :icon="BookMarked"
        title="暂无文档"
        description="先从项目详情页或文档编辑器创建第一篇富文本文档。"
      >
        <Button @click="goEditor">
          <FilePenLine class="button-icon" />
          新建文档
        </Button>
      </EmptyState>
      <Table v-else class="document-table">
        <thead>
          <tr>
            <th>文档</th>
            <th>所属项目</th>
            <th>维护信息</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <TableSkeletonRows v-if="visibleLoading" :rows="5" :columns="5" />
          <tr v-for="row in filteredRows" v-else :key="row.id" class="document-row">
            <td class="document-main-cell">
              <button type="button" class="document-title-link" @click="goDocument(row.id)">
                <BookOpenText class="document-title-icon" />
                <span>{{ row.title }}</span>
              </button>
              <div class="document-summary">{{ row.summary || '暂无摘要' }}</div>
              <div class="document-chip-row">
                <Badge variant="secondary" class="document-type-badge">{{ row.typeLabel }}</Badge>
                <span>v{{ row.currentVersionNo || 1 }}</span>
              </div>
            </td>
            <td class="document-project-cell">
              <div class="document-project-name">{{ projectName(row.projectId) }}</div>
              <div class="document-project-meta">项目文档</div>
            </td>
            <td class="document-maintain-cell">
              <div>研发用户</div>
              <span>{{ formatDate(row.updatedAt) }}</span>
            </td>
            <td class="document-status-cell">
              <Badge variant="secondary" :class="['ui-badge', badgeClass(row.status)]">
                {{ row.statusLabel }}
              </Badge>
            </td>
            <td class="document-action-cell">
              <div class="table-actions">
                <Button variant="ghost" size="sm" @click="goDocument(row.id)">
                  <Eye class="button-icon" />
                  查看
                </Button>
                <Button variant="ghost" size="sm" @click="goEditor(row)">
                  <FilePenLine class="button-icon" />
                  编辑
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </Table>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  BookMarked,
  BookOpenText,
  Braces,
  CircleHelp,
  Eye,
  FileCog,
  FileKey2,
  FilePenLine,
  History,
  RefreshCw,
  Rocket,
  ServerCog,
} from 'lucide-vue-next'
import { getDocuments, type DocumentItem } from '@/api/documents'
import { getProjects, type Project } from '@/api/projects'
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
import EmptyState from '@/components/ui/EmptyState.vue'
import { Input } from '@/components/ui/input'
import PageHeader from '@/components/ui/PageHeader.vue'
import TableSkeletonRows from '@/components/ui/TableSkeletonRows.vue'
import { useDelayedLoading } from '@/composables/use-delayed-loading'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Table } from '@/components/ui/table'

const EMPTY_SELECT_VALUE = '__empty__'

const router = useRouter()
const loading = ref(false)
const visibleLoading = useDelayedLoading(loading)
const errorMessage = ref('')
const rows = ref<DocumentItem[]>([])
const projects = ref<Project[]>([])

const filters = reactive({
  keyword: '',
  type: '',
  status: '',
})

const selectedDocType = computed({
  get: () => selectValue(filters.type),
  set: (value) => {
    filters.type = restoreSelectValue(String(value))
  },
})

const selectedStatus = computed({
  get: () => selectValue(filters.status),
  set: (value) => {
    filters.status = restoreSelectValue(String(value))
  },
})

const docTypes = [
  { name: '项目介绍', value: 'intro', description: '背景、范围、核心能力', icon: BookMarked },
  { name: '环境说明', value: 'environment', description: '线上、测试、灰度环境', icon: ServerCog },
  { name: '部署说明', value: 'deploy', description: '发布流程和回滚步骤', icon: Rocket },
  { name: 'API 说明', value: 'api', description: '接口平台和联调说明', icon: Braces },
  { name: '账号/权限申请说明', value: 'access', description: '权限入口和申请流程', icon: FileKey2 },
  { name: '常见问题 FAQ', value: 'faq', description: '常见排障和答疑', icon: CircleHelp },
  { name: '开发规范', value: 'standard', description: '代码、分支、评审规范', icon: FileCog },
]

const docTypeCards = computed(() => {
  return docTypes.map((type) => ({
    ...type,
    count: rows.value.filter((row) => row.type === type.value).length,
  }))
})

const docTypeOptions = computed(() => [
  { label: '全部类型', value: '' },
  ...docTypes.map((item) => ({ label: item.name, value: item.value })),
])

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '已发布', value: 'published' },
  { label: '草稿', value: 'draft' },
]

onMounted(async () => {
  await Promise.all([loadProjects(), loadDocuments()])
})

const filteredRows = computed(() => {
  return rows.value.filter((row) => {
    const text = `${row.title} ${projectName(row.projectId)} ${row.typeLabel} ${row.summary} ${row.content}`
    const matchKeyword = !filters.keyword || text.includes(filters.keyword)
    const matchType = !filters.type || row.type === filters.type
    const matchStatus = !filters.status || row.status === filters.status
    return matchKeyword && matchType && matchStatus
  })
})

function badgeClass(status: string) {
  if (status === 'published') return 'ui-badge--success'
  return 'ui-badge--default'
}

async function loadProjects() {
  const res = await getProjects({ page: 1, pageSize: 100 })
  projects.value = res.data.list
}

async function loadDocuments() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getDocuments({
      keyword: filters.keyword.trim(),
      type: filters.type,
      status: filters.status,
      page: 1,
      pageSize: 100,
    })
    rows.value = res.data.list
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '获取文档列表失败'
  } finally {
    loading.value = false
  }
}

function goEditor(row?: DocumentItem) {
  if (row) {
    router.push({
      path: '/documents/editor',
      query: {
        projectId: String(row.projectId),
        type: row.type,
      },
    })
    return
  }
  router.push('/documents/editor')
}

function goDocument(id: number) {
  router.push({ name: 'DocumentDetail', params: { id } })
}

function projectName(projectId: number) {
  return projects.value.find((project) => project.id === projectId)?.name || `项目 #${projectId}`
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

function selectValue(value: string) {
  return value || EMPTY_SELECT_VALUE
}

function restoreSelectValue(value: string) {
  return value === EMPTY_SELECT_VALUE ? '' : value
}
</script>
