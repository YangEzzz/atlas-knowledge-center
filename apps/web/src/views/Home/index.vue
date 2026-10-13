<template>
  <PageHeader
    eyebrow="Atlas 工作台"
    title="研发知识地图"
    description="统一查找项目、环境地址、GitLab、接口文档、负责人和流程说明。"
  >
    <template #actions>
      <Button>
        <FolderPlus class="button-icon" />
        新建项目
      </Button>
      <Button variant="outline" @click="goEditor">
        <FilePenLine class="button-icon" />
        新建文档
      </Button>
    </template>
  </PageHeader>

  <Card class="search-panel">
    <CardContent>
      <div class="search-panel-row">
        <Input v-model="keyword" placeholder="搜索项目、GitLab、测试地址、线上地址、文档或负责人" />
        <Button size="lg">
          <Search class="button-icon" />
          搜索
        </Button>
      </div>
      <div class="quick-tags">
        <Badge v-for="tag in quickTags" :key="tag" variant="outline">{{ tag }}</Badge>
      </div>
    </CardContent>
  </Card>

  <div class="stats-grid">
    <StatCard label="项目" :value="statsLoading ? '...' : homeStats.projectTotal" description="已沉淀业务系统">
      <template #icon><FolderKanban class="metric-icon" /></template>
    </StatCard>
    <StatCard
      label="资源入口"
      :value="statsLoading ? '...' : homeStats.entryTotal"
      description="项目入口和工具导航"
    >
      <template #icon><Link2 class="metric-icon" /></template>
    </StatCard>
    <StatCard label="文档" :value="statsLoading ? '...' : homeStats.documentTotal" description="富文本文档">
      <template #icon><BookOpen class="metric-icon" /></template>
    </StatCard>
    <StatCard label="可能过期" :value="statsLoading ? '...' : homeStats.staleDocumentTotal" description="超过 90 天未更新">
      <template #icon><TriangleAlert class="metric-icon" /></template>
    </StatCard>
  </div>

  <div v-if="homeError" class="notice notice-error">{{ homeError }}</div>
  <div v-if="pingResult" class="notice">{{ pingResult }}</div>

  <div class="content-grid">
    <Card>
      <CardHeader class="ui-card-header">
        <div>
          <CardTitle>常用入口</CardTitle>
          <CardDescription>研发高频打开的项目、文档和工具。</CardDescription>
        </div>
        <CardAction>
          <Button variant="outline" size="sm" @click="router.push('/favorites')">查看全部</Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        <EmptyState
          v-if="favoriteRows.length === 0"
          :icon="Link2"
          title="暂无常用入口"
          description="收藏项目、文档或工具后，会优先出现在这里。"
        />

        <Table v-else class="home-record-table">
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead class="home-center-cell">类型</TableHead>
              <TableHead class="home-center-cell">所属项目</TableHead>
              <TableHead class="home-center-cell">收藏时间</TableHead>
              <TableHead class="home-center-cell">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in favoriteRows" :key="row.id">
              <TableCell class="whitespace-normal">
                <div class="record-title">{{ row.title }}</div>
                <div class="record-description">{{ row.description || '暂无描述' }}</div>
              </TableCell>
              <TableCell class="home-center-cell">
                <Badge variant="secondary">{{ recordKindLabel(row.kind) }}</Badge>
              </TableCell>
              <TableCell class="home-center-cell text-muted-foreground">{{ row.projectName || '-' }}</TableCell>
              <TableCell class="home-center-cell text-muted-foreground">{{ formatDate(row.recordedAt) }}</TableCell>
              <TableCell class="home-center-cell">
                <Button variant="ghost" size="sm" @click="openRecord(row.path)">
                  <ExternalLink class="button-icon" />
                  打开
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <div class="side-stack">
      <Card>
        <CardHeader class="ui-card-header">
          <div>
            <CardTitle>最近访问</CardTitle>
            <CardDescription>近期打开过的项目、文档和工具。</CardDescription>
          </div>
          <CardAction>
            <Button variant="outline" size="sm" @click="router.push('/recent-visits')">查看全部</Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <EmptyState
            v-if="recentRows.length === 0"
            :icon="ExternalLink"
            title="暂无最近访问"
            description="打开项目、文档或工具后，会自动记录到这里。"
          />

          <div v-else class="activity-list">
            <div v-for="item in recentRows" :key="item.id" class="activity-item">
              <div>
                <div class="item-title">{{ item.title }}</div>
                <div class="item-meta">{{ recordKindLabel(item.kind) }} / {{ item.projectName || '-' }}</div>
              </div>
              <Button variant="ghost" size="sm" @click="openRecord(item.path)">
                <ExternalLink class="button-icon" />
                打开
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="ui-card-header">
          <div>
            <CardTitle>服务状态</CardTitle>
            <CardDescription>本地调试时可快速确认 API。</CardDescription>
          </div>
          <CardAction>
            <Button variant="outline" size="sm" @click="checkPing">检测 API</Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          <EmptyState
            :icon="PlugZap"
            title="接口连通性"
            :description="pingResult || '点击检测 API，确认前后端代理和服务状态。'"
          />
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  BookOpen,
  ExternalLink,
  FilePenLine,
  FolderKanban,
  FolderPlus,
  Link2,
  PlugZap,
  Search,
  TriangleAlert,
} from 'lucide-vue-next'
import { getDocuments, type DocumentItem } from '@/api/documents'
import { getProjects, type Project } from '@/api/projects'
import { getPublicLinks } from '@/api/public-links'
import { ping } from '@/api/system'
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
import StatCard from '@/components/ui/StatCard.vue'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { usePersonalRecords, type PersonalRecordKind } from '@/composables/use-personal-records'

const router = useRouter()
const keyword = ref('')
const pingResult = ref('')
const homeError = ref('')
const statsLoading = ref(false)
const quickTags = ['测试环境', 'GitLab', 'API 文档', '部署说明', '负责人', '账号申请']
const { favorites, recentVisits } = usePersonalRecords()
const favoriteRows = computed(() => favorites.value.slice(0, 5))
const recentRows = computed(() => recentVisits.value.slice(0, 5))
const homeStats = reactive({
  projectTotal: 0,
  entryTotal: 0,
  documentTotal: 0,
  staleDocumentTotal: 0,
})

onMounted(() => {
  loadHomeStats()
})

function goEditor() {
  router.push('/documents/editor')
}

function openRecord(path: string) {
  if (/^https?:\/\//i.test(path)) {
    window.open(path, '_blank', 'noopener,noreferrer')
    return
  }
  router.push(path)
}

function recordKindLabel(kind: PersonalRecordKind) {
  const labels: Record<PersonalRecordKind, string> = {
    project: '项目',
    document: '文档',
    public_link: '工具',
  }
  return labels[kind]
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

async function checkPing() {
  try {
    const res = await ping()
    pingResult.value = `API 正常：${res.data.message} / ${res.data.time}`
  } catch (error) {
    pingResult.value = error instanceof Error ? `API 连接失败：${error.message}` : 'API 连接失败'
  }
}

async function loadHomeStats() {
  statsLoading.value = true
  homeError.value = ''
  try {
    const [projects, documents, publicLinksRes] = await Promise.all([
      fetchAllProjects(),
      fetchAllDocuments(),
      getPublicLinks({ page: 1, pageSize: 1 }),
    ])
    homeStats.projectTotal = projects.total
    homeStats.documentTotal = documents.total
    homeStats.entryTotal = countProjectEntries(projects.list) + publicLinksRes.data.total
    homeStats.staleDocumentTotal = countStaleDocuments(documents.list)
  } catch (error) {
    homeError.value = error instanceof Error ? `首页统计加载失败：${error.message}` : '首页统计加载失败'
  } finally {
    statsLoading.value = false
  }
}

async function fetchAllProjects() {
  const pageSize = 100
  const firstPage = await getProjects({ page: 1, pageSize })
  const list = [...firstPage.data.list]
  const total = firstPage.data.total
  const pageCount = Math.ceil(total / pageSize)
  for (let page = 2; page <= pageCount; page += 1) {
    const res = await getProjects({ page, pageSize })
    list.push(...res.data.list)
  }
  return { list, total }
}

async function fetchAllDocuments() {
  const pageSize = 100
  const firstPage = await getDocuments({ page: 1, pageSize })
  const list = [...firstPage.data.list]
  const total = firstPage.data.total
  const pageCount = Math.ceil(total / pageSize)
  for (let page = 2; page <= pageCount; page += 1) {
    const res = await getDocuments({ page, pageSize })
    list.push(...res.data.list)
  }
  return { list, total }
}

function countProjectEntries(projects: Project[]) {
  return projects.reduce((total, project) => total + (project.entries?.length || 0), 0)
}

function countStaleDocuments(documents: DocumentItem[]) {
  const ninetyDays = 90 * 24 * 60 * 60 * 1000
  return documents.filter((document) => {
    if (!document.updatedAt) return false
    const updatedAt = new Date(document.updatedAt).getTime()
    return Number.isFinite(updatedAt) && Date.now() - updatedAt > ninetyDays
  }).length
}
</script>
