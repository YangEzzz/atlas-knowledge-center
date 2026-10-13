<template>
  <Card>
    <CardHeader class="ui-card-header">
      <div>
        <CardTitle>{{ title }}</CardTitle>
      </div>
      <CardAction v-if="actionText">
        <Button variant="outline" size="sm">{{ actionText }}</Button>
      </CardAction>
    </CardHeader>
    <CardContent>
      <Table>
        <thead>
          <tr>
            <th v-for="column in columns" :key="column">{{ column }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, rowIndex) in rows" :key="rowIndex">
            <td v-for="column in columns" :key="column">{{ row[column] || '-' }}</td>
          </tr>
        </tbody>
      </Table>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table } from '@/components/ui/table'

defineProps<{
  title: string
  actionText?: string
  columns: string[]
  rows: Array<Record<string, string>>
}>()
</script>
