<template>
  <AppUiProvider>
    <RouterView />
    <Toaster position="top-right" />
  </AppUiProvider>
</template>

<script setup lang="ts">
import AppUiProvider from '@/components/app/AppUiProvider.vue'
import { Toaster } from '@/components/ui/sonner'
</script>
