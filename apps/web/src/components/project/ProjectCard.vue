<script setup lang="ts">
import { Heart, ArrowUpRight } from "lucide-vue-next";
import type { Project } from "@/api/projects";
import { Button } from "@/components/ui/button";
import ProjectIdentity from "./ProjectIdentity.vue";
import { usePersonalRecords } from "@/composables/use-personal-records";
import {
  entryLabel,
  projectOwnerNames,
  formatDate,
} from "@/utils/project-identity";
defineProps<{ project: Project }>();
const emit = defineEmits<{ favorite: [] }>();
const { isFavorite, toggleFavorite } = usePersonalRecords();
function favorite(project: Project) {
  toggleFavorite({
    kind: "project",
    targetId: project.id,
    title: project.name,
    description: project.description,
    path: `/projects/${project.id}`,
    badge: project.typeLabel,
  });
  emit("favorite");
}
</script>

<template>
  <article class="project-card">
    <div class="project-card-top">
      <ProjectIdentity :id="project.id" :name="project.name" />
      <Button
        variant="ghost"
        size="icon"
        :aria-label="`${isFavorite('project', project.id) ? '取消收藏' : '收藏'}${project.name}`"
        :aria-pressed="isFavorite('project', project.id)"
        @click="favorite(project)"
      >
        <Heart
          :class="{ 'is-filled': isFavorite('project', project.id) }"
          aria-hidden="true"
        />
      </Button>
    </div>
    <h3>
      <RouterLink :to="`/projects/${project.id}`">{{
        project.name
      }}</RouterLink>
    </h3>
    <p class="project-card-description">
      {{ project.description || "项目介绍待补充" }}
    </p>
    <div class="project-card-resources">
      <a
        v-for="entry in (project.entries || [])
          .filter((item) => item.url)
          .slice(0, 3)"
        :key="entry.id || entry.url"
        :href="entry.url"
        target="_blank"
        rel="noopener noreferrer"
        :title="`${entry.title || entryLabel(entry.type)}：${entry.url}`"
        >{{ entry.title || entryLabel(entry.type) }}<ArrowUpRight aria-hidden="true"
      /></a>
      <RouterLink
        v-if="!project.entries?.length"
        :to="`/projects/${project.id}`"
        >补充资源入口 <ArrowUpRight aria-hidden="true"
      /></RouterLink>
    </div>
    <div class="project-card-footer">
      <span :title="projectOwnerNames(project)">{{
        projectOwnerNames(project)
      }}</span
      ><span>{{ project.typeLabel || project.type }}</span>
    </div>
    <time class="project-card-date" :datetime="project.updatedAt"
      >更新于 {{ formatDate(project.updatedAt) }}</time
    >
  </article>
</template>
