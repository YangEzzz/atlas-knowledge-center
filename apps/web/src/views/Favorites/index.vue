<template>
  <PageHeader
    title="我的收藏"
    description="集中查看当前用户收藏的项目、文档和工具，收藏仅对自己生效。"
  />

  <Card>
    <CardHeader class="ui-card-header">
      <div>
        <CardTitle>收藏列表</CardTitle>
        <CardDescription
          >共
          {{
            rows.length
          }}
          条收藏，常用项目、文档和工具会固定在这里。</CardDescription
        >
      </div>
      <CardAction>
        <Button
          variant="outline"
          size="sm"
          :disabled="rows.length === 0"
          @click="clearFavorites"
        >
          <HeartOff class="button-icon" />
          清空收藏
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <EmptyState
        v-if="rows.length === 0"
        :icon="Heart"
        title="暂无收藏"
        description="在项目详情、文档详情或工具导航中点击收藏后，会出现在这里。"
      />

      <div v-else class="personal-table-panel">
        <Table class="personal-record-table">
          <colgroup>
            <col />
            <col style="width: 96px" />
            <col style="width: 180px" />
            <col style="width: 148px" />
            <col style="width: 176px" />
          </colgroup>
          <TableHeader class="bg-sky-50/70">
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>所属项目</TableHead>
              <TableHead>收藏时间</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow
              v-for="row in rows"
              :key="row.id"
              class="hover:bg-sky-50/40"
            >
              <TableCell class="whitespace-normal">
                <div class="min-w-0">
                  <div class="record-title">{{ row.title }}</div>
                  <div class="record-description">
                    {{ row.description || "暂无描述" }}
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <Badge variant="secondary">{{
                  recordKindLabel(row.kind)
                }}</Badge>
              </TableCell>
              <TableCell class="text-muted-foreground truncate">
                {{ row.projectName || "-" }}
              </TableCell>
              <TableCell class="text-muted-foreground">
                {{ formatDate(row.recordedAt) }}
              </TableCell>
              <TableCell>
                <div class="flex justify-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="openRecord(row.path)"
                  >
                    <ExternalLink class="button-icon" />
                    打开
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="removeFavorite(row.id)"
                  >
                    <HeartOff class="button-icon" />
                    取消
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { ExternalLink, Heart, HeartOff } from "lucide-vue-next";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  usePersonalRecords,
  type PersonalRecordKind,
} from "@/composables/use-personal-records";

const router = useRouter();
const {
  favorites: rows,
  clearFavorites,
  removeFavorite,
} = usePersonalRecords();

function openRecord(path: string) {
  if (/^https?:\/\//i.test(path)) {
    window.open(path, "_blank", "noopener,noreferrer");
    return;
  }
  router.push(path);
}

function recordKindLabel(kind: PersonalRecordKind) {
  const labels: Record<PersonalRecordKind, string> = {
    project: "项目",
    document: "文档",
    public_link: "工具",
  };
  return labels[kind];
}

function formatDate(value: string) {
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
