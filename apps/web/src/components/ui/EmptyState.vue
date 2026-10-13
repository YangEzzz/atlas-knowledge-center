<template>
  <div class="ui-empty">
    <component :is="icon" v-if="icon" class="ui-empty-icon" />
    <h3>{{ title }}</h3>
    <p>{{ description }}</p>
    <slot />
  </div>
</template>

<script setup lang="ts">
import type { Component, PropType } from 'vue'

defineProps({
  icon: {
    type: [Object, Function] as PropType<Component>,
    default: undefined,
  },
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    required: true,
  },
})
</script>
