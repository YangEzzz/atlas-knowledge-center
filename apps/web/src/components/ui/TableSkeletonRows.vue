<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton'

withDefaults(
  defineProps<{
    rows?: number
    columns?: number
  }>(),
  {
    rows: 4,
    columns: 5,
  },
)
</script>

<template>
  <tr v-for="row in rows" :key="row" class="skeleton-table-row">
    <td v-for="column in columns" :key="column">
      <Skeleton :class="['skeleton-table-cell', column === 1 && 'is-wide']" />
    </td>
  </tr>
</template>
