<script setup lang="ts">
import { ConfigProvider } from 'reka-ui'

// The app viewport owns the visible page scrollbar. Keep body overlay locking
// and its focus/pointer isolation, without a second scrollbar allowance.
const scrollBody = { padding: 0, margin: 0 }
</script>

<template>
  <ConfigProvider :scroll-body="scrollBody">
    <slot />
  </ConfigProvider>
</template>
