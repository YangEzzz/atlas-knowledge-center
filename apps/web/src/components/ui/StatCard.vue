<template>
  <Card class="stat-card">
    <div class="stat-top">
      <div>
        <div class="stat-label">{{ label }}</div>
        <div class="stat-value">{{ value }}</div>
      </div>
      <div v-if="$slots.icon" class="stat-icon">
        <slot name="icon" />
      </div>
    </div>
    <div v-if="description" class="stat-description">{{ description }}</div>
  </Card>
</template>

<script setup lang="ts">
import { Card } from '@/components/ui/card'

defineProps<{
  label: string
  value: string | number
  description?: string
}>()
</script>
