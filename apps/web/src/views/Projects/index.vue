<template>
  <PageHeader title="项目库" description="浏览、搜索和维护公司内部业务系统，以及每个项目下的仓库、环境和负责人。">
    <template #actions>
      <Button @click="openCreate">
        <FolderPlus class="button-icon" />
        新建项目
      </Button>
      <Button variant="outline">
        <SlidersHorizontal class="button-icon" />
        批量维护
      </Button>
    </template>
  </PageHeader>

  <Card class="filter-card">
    <CardContent>
      <div class="filter-bar">
        <Input v-model="filters.keyword" placeholder="搜索项目、关键词、负责人" @keyup.enter="loadProjects" />
        <Select v-model="selectedType">
          <SelectTrigger class="ui-select-trigger" aria-label="项目形态">
            <SelectValue placeholder="全部形态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="type in projectTypeFilterOptions" :key="type.value" :value="selectValue(type.value)">
              {{ type.label }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Select v-model="selectedStatus">
          <SelectTrigger class="ui-select-trigger" aria-label="维护状态">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="status in statusOptions" :key="status.value" :value="selectValue(status.value)">
              {{ status.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="quick-tags">
        <button
          v-for="type in projectTypes"
          :key="type.value"
          :class="['tag-button', { 'is-active': filters.type === type.value }]"
          type="button"
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
        <CardTitle>项目列表</CardTitle>
        <CardDescription>一个业务系统作为一个项目。当前共 {{ total }} 个项目。</CardDescription>
      </div>
      <CardAction>
        <Button variant="outline" size="sm" :disabled="loading" @click="loadProjects">
          <RefreshCw :class="['button-icon', { 'is-spinning': visibleLoading }]" />
          刷新
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <div v-if="errorMessage" class="notice notice-error">{{ errorMessage }}</div>

      <EmptyState
        v-if="!loading && filteredRows.length === 0"
        :icon="FolderKanban"
        title="暂无项目"
        description="先新建一个项目，后续再为项目补充环境、GitLab、原型和 API 文档地址。"
      >
        <Button @click="openCreate">
          <FolderPlus class="button-icon" />
          新建项目
        </Button>
      </EmptyState>

      <Table v-else>
        <TableHeader>
          <TableRow>
            <TableHead>项目名称</TableHead>
            <TableHead>形态</TableHead>
            <TableHead>简介</TableHead>
            <TableHead>负责人</TableHead>
            <TableHead>项目入口</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>更新时间</TableHead>
            <TableHead class="operation-head">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableSkeletonRows v-if="visibleLoading" :rows="5" :columns="8" />
          <TableRow v-for="row in filteredRows" v-else :key="row.id">
            <TableCell>
              <div class="strong-cell">{{ row.name }}</div>
              <div class="item-meta">{{ row.keywords.length ? row.keywords.join(' / ') : '暂无关键词' }}</div>
            </TableCell>
            <TableCell>
              <Badge variant="secondary">{{ typeLabel(row.type) || row.typeLabel }}</Badge>
            </TableCell>
            <TableCell class="wide-cell">{{ row.description || '暂无简介' }}</TableCell>
            <TableCell>
              <div class="owner-stack">
                <span>前端：{{ ownerNames(row.owners?.frontend) }}</span>
                <span>后端：{{ ownerNames(row.owners?.backend) }}</span>
                <span>测试：{{ ownerNames(row.owners?.tester) }}</span>
              </div>
            </TableCell>
            <TableCell>
              <div class="entry-list">
                <template v-for="entryType in coreEntryTypes" :key="`core-${entryType.value}`">
                  <Button
                    v-if="entryByType(row, entryType.value)"
                    as="a"
                    :href="entryByType(row, entryType.value)?.url"
                    target="_blank"
                    rel="noreferrer"
                    variant="outline"
                    size="sm"
                    class="entry-link"
                  >
                    {{ entryType.shortLabel }}
                  </Button>
                  <Button v-else variant="ghost" size="sm" disabled class="entry-missing">
                    {{ entryType.shortLabel }}
                  </Button>
                </template>
                <Button
                  v-for="entry in visibleExtraEntries(row)"
                  :key="`extra-${entry.id || entry.type}-${entry.url}`"
                  as="a"
                  :href="entry.url"
                  target="_blank"
                  rel="noreferrer"
                  variant="outline"
                  size="sm"
                  class="entry-link entry-link-extra"
                >
                  {{ entryTitle(entry) }}
                </Button>
                <Badge
                  v-if="hiddenExtraEntryCount(row) > 0"
                  variant="secondary"
                  class="entry-overflow-badge"
                >
                  +{{ hiddenExtraEntryCount(row) }}
                </Badge>
              </div>
            </TableCell>
            <TableCell>
              <Badge variant="secondary" :class="['ui-badge', badgeClass(projectStatus(row))]">
                {{ projectStatus(row) }}
              </Badge>
            </TableCell>
            <TableCell>{{ formatDate(row.updatedAt) }}</TableCell>
            <TableCell>
              <div class="row-actions">
                <Button variant="ghost" size="sm" class="row-action-button" @click="goDetail(row.id)">
                  <Eye class="button-icon" />
                  详情
                </Button>
                <Button variant="ghost" size="sm" class="row-action-button" @click="openEdit(row)">
                  <PencilLine class="button-icon" />
                  编辑
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </CardContent>
  </Card>

  <ProjectFormDialog
    v-model:open="projectDialogOpen"
    :project="editingProject"
    @saved="handleProjectSaved"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Eye,
  FolderKanban,
  FolderPlus,
  PencilLine,
  RefreshCw,
  SlidersHorizontal,
} from 'lucide-vue-next'
import {
  getProjects,
  type Project,
  type ProjectEntry,
  type ProjectOwner,
} from '@/api/projects'
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
import ProjectFormDialog from '@/components/project/ProjectFormDialog.vue'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const EMPTY_SELECT_VALUE = '__empty__'
const router = useRouter()

const coreEntryTypes = [
  { label: '线上地址', shortLabel: '线上', value: 'production' },
  { label: '测试地址', shortLabel: '测试', value: 'staging' },
  { label: 'GitLab', shortLabel: 'GitLab', value: 'gitlab' },
]

const entryTypeLabels: Record<string, string> = {
  production: '线上',
  staging: '测试',
  gitlab: 'GitLab',
  prototype: '原型',
  api_doc: 'API',
  admin: '后台',
  monitor: '监控',
  logs: '日志',
  deploy: '发布',
  access: '权限',
  other: '其他',
}

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

const projectTypeFilterOptions = [{ label: '全部形态', value: '' }, ...projectTypes]
const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '正常', value: '正常' },
  { label: '需补介绍', value: '需补介绍' },
  { label: '需补负责人', value: '需补负责人' },
  { label: '待补核心入口', value: '待补核心入口' },
]

const selectedType = computed({
  get: () => selectValue(filters.type),
  set: (value) => {
    filters.type = restoreSelectValue(String(value))
    loadProjects()
  },
})

const selectedStatus = computed({
  get: () => selectValue(filters.status),
  set: (value) => {
    filters.status = restoreSelectValue(String(value))
  },
})

const filters = reactive({
  keyword: '',
  type: '',
  status: '',
})

const rows = ref<Project[]>([])
const total = ref(0)
const loading = ref(false)
const visibleLoading = useDelayedLoading(loading)
const projectDialogOpen = ref(false)
const editingProject = ref<Project | null>(null)
const errorMessage = ref('')

const filteredRows = computed(() => {
  return rows.value.filter((row) => {
    const status = projectStatus(row)
    return !filters.status || status === filters.status
  })
})

onMounted(() => {
  loadProjects()
})

async function loadProjects() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getProjects({
      keyword: filters.keyword.trim(),
      type: filters.type,
      page: 1,
      pageSize: 50,
    })
    rows.value = res.data.list
    total.value = res.data.total
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '获取项目列表失败'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingProject.value = null
  projectDialogOpen.value = true
}

function goDetail(id: number) {
  router.push({ name: 'ProjectDetail', params: { id } })
}

function openEdit(project: Project) {
  editingProject.value = project
  projectDialogOpen.value = true
}

async function handleProjectSaved() {
  editingProject.value = null
  await loadProjects()
}

function toggleType(type: string) {
  filters.type = filters.type === type ? '' : type
  loadProjects()
}

function selectValue(value: string) {
  return value || EMPTY_SELECT_VALUE
}

function restoreSelectValue(value: string) {
  return value === EMPTY_SELECT_VALUE ? '' : value
}

function ownerNames(owners?: ProjectOwner[] | null) {
  if (!owners || owners.length === 0) {
    return '-'
  }
  return owners.map((owner) => owner.ownerName).join('、')
}

function entryByType(project: Project, type: string) {
  return project.entries?.find((entry) => entry.type === type && entry.url)
}

function extraEntries(project: Project) {
  const coreTypes = new Set(coreEntryTypes.map((entryType) => entryType.value))
  return (project.entries || [])
    .filter((entry) => entry.url && !coreTypes.has(entry.type))
    .slice()
    .sort((a, b) => a.sortOrder - b.sortOrder)
}

function visibleExtraEntries(project: Project) {
  return extraEntries(project).slice(0, 2)
}

function hiddenExtraEntryCount(project: Project) {
  return Math.max(extraEntries(project).length - visibleExtraEntries(project).length, 0)
}

function entryTitle(entry: ProjectEntry) {
  return entryTypeLabels[entry.type] || entry.title || entry.type
}

function typeLabel(type: string) {
  return projectTypes.find((item) => item.value === type)?.label || type
}

function projectStatus(project: Project) {
  const hasAnyOwner =
    Boolean(project.owners?.frontend?.length) ||
    Boolean(project.owners?.backend?.length) ||
    Boolean(project.owners?.tester?.length)
  if (!project.description) {
    return '需补介绍'
  }
  if (!hasAnyOwner) {
    return '需补负责人'
  }
  if (coreEntryTypes.some((entryType) => !entryByType(project, entryType.value))) {
    return '待补核心入口'
  }
  return '正常'
}

function badgeClass(status: string) {
  if (status === '正常') return 'ui-badge--success'
  if (status === '待补核心入口') return 'ui-badge--warning'
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
</script>
