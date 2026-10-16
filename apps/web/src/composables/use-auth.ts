import type { CurrentUser } from '@/store/auth'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { getCurrentUser, logoutSession } from '@/api/auth'
import { useAuthStore } from '@/store/auth'
import { clearLegacyAuth } from '@/utils/session'

let profileRequest: Promise<CurrentUser | null> | null = null

export function useAuth() {
  const router = useRouter()
  const store = useAuthStore()
  const loading = ref(false)

  async function logout() {
    if (loading.value)
      return
    loading.value = true
    try {
      await logoutSession()
      store.isLogin = false
      store.user = null
      store.initialized = true
      clearLegacyAuth()
      await router.push('/auth/sign-in')
    }
    catch (error) { toast.error(error instanceof Error ? error.message : '退出失败，请重试') }
    finally { loading.value = false }
  }

  async function getUserInfo() {
    if (!profileRequest) {
      clearLegacyAuth()
      profileRequest = getCurrentUser()
        .then((user) => {
          store.user = user
          store.isLogin = true
          return user
        })
        .catch(() => {
          store.user = null
          store.isLogin = false
          return null
        })
        .finally(() => {
          store.initialized = true
          profileRequest = null
        })
    }
    return profileRequest
  }

  return { loading, logout, getUserInfo }
}
