<template>
  <PageHeader
    title="管理中心"
    description="低频配置、回收站、审计记录和基础字典统一收纳在这里。"
  />

  <div class="management-grid">
    <Card v-for="item in modules" :key="item.title" class="manage-card">
      <div class="manage-card-icon">
        <component :is="item.icon" class="inline-icon" />
      </div>
      <div class="card-title">{{ item.title }}</div>
      <div class="card-meta">{{ item.description }}</div>
      <Button variant="secondary" size="sm" @click="item.action?.()">
        进入
        <ArrowRight class="button-icon" />
      </Button>
    </Card>
  </div>

  <Card ref="trashCardRef">
    <CardHeader class="ui-card-header">
      <div>
        <CardTitle>回收站</CardTitle>
        <CardDescription
          >当前有
          {{
            trashTotal
          }}
          篇已删除文档。恢复后会重新出现在文档库和项目详情中。</CardDescription
        >
      </div>
      <CardAction>
        <Button
          variant="outline"
          size="sm"
          :disabled="trashLoading"
          @click="loadTrash"
        >
          <RefreshCw
            :class="['button-icon', { 'is-spinning': visibleTrashLoading }]"
          />
          刷新
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <div v-if="trashError" class="notice notice-error">{{ trashError }}</div>
      <div v-if="trashMessage" class="notice">{{ trashMessage }}</div>

      <EmptyState
        v-if="!trashLoading && trashRows.length === 0"
        :icon="Trash2"
        title="回收站为空"
        description="删除后的文档会暂存在这里，后续可以恢复。"
      />

      <Table v-else class="manage-trash-table">
        <TableHeader>
          <TableRow>
            <TableHead>文档</TableHead>
            <TableHead>类型</TableHead>
            <TableHead>所属项目</TableHead>
            <TableHead>状态</TableHead>
            <TableHead>删除时间</TableHead>
            <TableHead class="operation-head">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableSkeletonRows
            v-if="visibleTrashLoading"
            :rows="4"
            :columns="6"
          />
          <TableRow v-for="row in trashRows" v-else :key="row.id">
            <TableCell class="wide-cell">
              <div class="strong-cell">{{ row.title }}</div>
              <div class="item-meta">{{ row.summary || "暂无摘要" }}</div>
            </TableCell>
            <TableCell>
              <Badge variant="secondary">{{ row.typeLabel }}</Badge>
            </TableCell>
            <TableCell>{{ projectName(row.projectId) }}</TableCell>
            <TableCell>
              <Badge
                variant="secondary"
                :class="[
                  'ui-badge',
                  row.status === 'published'
                    ? 'ui-badge--success'
                    : 'ui-badge--default',
                ]"
              >
                {{ row.statusLabel }}
              </Badge>
            </TableCell>
            <TableCell>{{
              formatDate(row.deletedAt || row.updatedAt)
            }}</TableCell>
            <TableCell>
              <div class="row-actions">
                <Button
                  variant="ghost"
                  size="sm"
                  :disabled="restoringId === row.id"
                  @click="restoreTrashDocument(row)"
                >
                  <RotateCcw
                    :class="[
                      'button-icon',
                      { 'is-spinning': restoringId === row.id },
                    ]"
                  />
                  {{ restoringId === row.id ? "恢复中" : "恢复" }}
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  ArrowRight,
  FileText,
  History,
  RefreshCw,
  RotateCcw,
  Settings2,
  Trash2,
} from "lucide-vue-next";
import {
  getDeletedDocuments,
  restoreDocument,
  type DocumentItem,
} from "@/api/documents";
import { getProjects, type Project } from "@/api/projects";
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
import EmptyState from "@/components/ui/EmptyState.vue";
import PageHeader from "@/components/ui/PageHeader.vue";
import TableSkeletonRows from "@/components/ui/TableSkeletonRows.vue";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useDelayedLoading } from "@/composables/use-delayed-loading";

const trashRows = ref<DocumentItem[]>([]);
const projects = ref<Project[]>([]);
const trashTotal = ref(0);
const trashLoading = ref(false);
const visibleTrashLoading = useDelayedLoading(trashLoading);
const restoringId = ref<number | null>(null);
const trashError = ref("");
const trashMessage = ref("");
const trashCardRef = ref<InstanceType<typeof Card> | null>(null);

const modules = [
  {
    title: "文档模板管理",
    description: "按文档类型维护富文本模板。",
    icon: FileText,
  },
  {
    title: "回收站",
    description: "恢复已软删除的文档。",
    icon: Trash2,
    action: scrollToTrash,
  },
  {
    title: "操作审计",
    description: "追踪创建、修改、删除和恢复记录。",
    icon: History,
  },
  {
    title: "系统配置",
    description: "项目形态、资源类型、上传限制、过期天数。",
    icon: Settings2,
  },
];

const projectMap = computed(() => {
  return new Map(projects.value.map((project) => [project.id, project.name]));
});

onMounted(async () => {
  await Promise.all([loadProjects(), loadTrash()]);
});

async function loadProjects() {
  try {
    const res = await getProjects({ page: 1, pageSize: 100 });
    projects.value = res.data.list;
  } catch {
    projects.value = [];
  }
}

async function loadTrash() {
  trashLoading.value = true;
  trashError.value = "";
  try {
    const res = await getDeletedDocuments({ page: 1, pageSize: 50 });
    trashRows.value = res.data.list;
    trashTotal.value = res.data.total;
  } catch (error) {
    trashError.value =
      error instanceof Error ? error.message : "获取回收站失败";
  } finally {
    trashLoading.value = false;
  }
}

async function restoreTrashDocument(row: DocumentItem) {
  restoringId.value = row.id;
  trashError.value = "";
  trashMessage.value = "";
  try {
    await restoreDocument(row.id);
    trashMessage.value = `已恢复「${row.title}」`;
    await loadTrash();
  } catch (error) {
    trashError.value = error instanceof Error ? error.message : "恢复文档失败";
  } finally {
    restoringId.value = null;
  }
}

function projectName(projectId: number) {
  return projectMap.value.get(projectId) || `项目 #${projectId}`;
}

function scrollToTrash() {
  trashCardRef.value?.$el?.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
}

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    timeZone: "Asia/Shanghai",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
</script>
