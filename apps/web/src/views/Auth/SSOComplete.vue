<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { useRoute, useRouter } from 'vue-router'
import { getLoginStatus } from '@/api/auth'
import { useAuth } from '@/composables/use-auth'
import { safeReturnPath } from '@/utils/session'

const route = useRoute()
const router = useRouter()
const { getUserInfo } = useAuth()
const title = ref('正在打开 Helios')
const message = ref('请稍候，登录窗口正在准备中。')
const busy = ref(true)
const failed = ref(false)
const transaction = typeof route.query.transaction === 'string' ? route.query.transaction : ''
const destination = ref('/')

async function enter() {
  if (await getUserInfo())
    await router.replace(destination.value)
  else await router.replace('/auth/sign-in')
}

onMounted(async () => {
  if (route.query.waiting === '1')
    return
  try {
    if (!transaction)
      throw new Error('登录请求已失效，请返回应用重新登录')
    const status = await getLoginStatus(transaction)
    if (status.status !== 'success')
      throw new Error(status.message || '登录未完成，请重新登录')
    title.value = '登录成功'
    message.value = '可以返回原窗口继续工作。此窗口将自动关闭。'
    destination.value = safeReturnPath(status.return_path)
    if (status.mode === 'redirect') {
      await enter()
      return
    }
    window.opener?.postMessage({ type: 'atlas:login-complete', transaction }, window.location.origin)
    window.close()
  }
  catch (cause) {
    failed.value = true
    title.value = '登录未完成'
    message.value = cause instanceof Error ? cause.message : '请返回应用重新登录'
    window.opener?.postMessage({ type: 'atlas:login-complete', transaction }, window.location.origin)
  }
  finally { busy.value = false }
})
</script>

<template>
  <main class="flex min-h-dvh items-center justify-center bg-background p-6">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <CardTitle>{{ title }}</CardTitle>
        <CardDescription aria-live="polite">
          {{ message }}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <p v-if="busy" role="status">正在准备登录...</p>
        <Button v-else class="w-full cursor-pointer" @click="failed ? router.replace('/auth/sign-in') : enter()">
          {{ failed ? '返回登录' : '进入知识中心' }}
        </Button>
      </CardContent>
    </Card>
  </main>
</template>

