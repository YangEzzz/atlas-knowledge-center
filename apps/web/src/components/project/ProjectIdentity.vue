<script setup lang="ts">
import { computed } from "vue";
import { projectIdentity } from "@/utils/project-identity";
const props = defineProps<{
  id: number | string;
  name: string;
  small?: boolean;
}>();
const identity = computed(() => projectIdentity(props.id, props.name));
</script>

<template>
  <span
    :class="[
      'project-identity',
      `project-identity--${identity.tone}`,
      { 'project-identity--small': small },
    ]"
    aria-hidden="true"
    >{{ identity.label }}</span
  >
</template>
