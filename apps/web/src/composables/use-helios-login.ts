import type { LoginStatus } from '@/api/auth'
import { onScopeDispose, ref } from 'vue'
import { useRouter } from 'vue-router'
import { cancelSSOLogin, getLoginStatus, startSSOLogin } from '@/api/auth'
import { useAuth } from '@/composables/use-auth'
import { safeReturnPath } from '@/utils/session'

export type { LoginStatus } from '@/api/auth'

export function useHeliosLogin() {
  const router = useRouter()
  const { getUserInfo } = useAuth()
  const loading = ref(false)
  const error = ref('')
  let popup: Window | null = null
  let transaction = ''
  let timer: ReturnType<typeof setTimeout> | undefined
  let disposed = false
  let checking = false
  let startedAt = 0

  function cleanup() {
    clearTimeout(timer)
    window.removeEventListener('message', onMessage)
  }

  async function finish(status: LoginStatus) {
    cleanup()
    if (status.status === 'success') {
      if (await getUserInfo()) {
        popup?.close()
        await router.replace(safeReturnPath(status.return_path))
      }
      else { error.value = '无法确认登录状态，请重新登录' }
    }
    else {
      error.value = status.message || '登录未完成，请重新登录'
      popup?.close()
    }
    loading.value = false
  }

  async function poll() {
    if (disposed || !loading.value || !transaction || checking)
      return
    checking = true
    try {
      const status = await getLoginStatus(transaction)
      if (disposed)
        return
      if (['success', 'failed', 'cancelled'].includes(status.status)) {
        await finish(status)
        return
      }
      if (popup?.closed && status.status === 'pending') {
        await cancel()
        if (!loading.value)
          return
      }
      if (Date.now() - startedAt > 10 * 60 * 1000)
        throw new Error('登录等待超时，请重新登录')
      timer = setTimeout(poll, 1500)
    }
    catch (cause) {
      cleanup()
      loading.value = false
      error.value = cause instanceof Error ? cause.message : '无法确认登录状态，请重试'
      popup?.close()
    }
    finally { checking = false }
  }

  function onMessage(event: MessageEvent) {
    if (event.origin !== window.location.origin || event.source !== popup
      || event.data?.type !== 'atlas:login-complete' || event.data?.transaction !== transaction) {
      return
    }
    clearTimeout(timer)
    timer = setTimeout(poll, 0)
  }

  async function login(fullPage = false) {
    if (loading.value) {
      if (!fullPage)
        return
      await cancel()
      if (loading.value || disposed)
        return
    }
    error.value = ''
    loading.value = true
    startedAt = Date.now()
    popup = fullPage ? null : window.open('/auth/sso/complete?waiting=1', 'atlas-helios-login', 'popup,width=480,height=680')
    const mode = popup ? 'popup' : 'redirect'
    try {
      const result = await startSSOLogin({
        mode,
        return_path: safeReturnPath(router.currentRoute.value.query.redirect),
      })
      if (disposed) {
        popup?.close()
        return
      }
      transaction = result.transaction_id
      if (!popup) {
        window.location.assign(result.authorize_url)
        return
      }
      window.addEventListener('message', onMessage)
      popup.location.href = result.authorize_url
      popup.focus()
      timer = setTimeout(poll, 1000)
    }
    catch (cause) {
      popup?.close()
      loading.value = false
      error.value = cause instanceof Error ? cause.message : '无法打开统一登录，请重试'
    }
  }

  async function cancel() {
    if (!loading.value || !transaction)
      return
    try {
      await cancelSSOLogin(transaction)
      const status = await getLoginStatus(transaction)
      if (status.status === 'processing') {
        error.value = '正在完成登录，请稍候'
        return
      }
      if (status.status !== 'pending')
        await finish(status)
    }
    catch (cause) { error.value = cause instanceof Error ? cause.message : '取消失败，请重试' }
  }

  onScopeDispose(() => {
    disposed = true
    cleanup()
  })
  return { login, cancel, loading, error }
}

