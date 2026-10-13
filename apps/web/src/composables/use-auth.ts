import { useRouter } from 'vue-router'
import { api } from '@/request'
import { useAuthStore } from '@/store/auth'
import { removeToken, setToken } from '@/utils/auth'

interface LoginResponse {
  token: string
  user: {
    id: string | number
    username?: string
    displayName?: string
    nickname?: string
    avatar?: string
    avatarUrl?: string
    roles?: string[]
    permissions?: string[]
  }
}

export function useAuth() {
  const router = useRouter()
  const authStore = useAuthStore()

  async function login(formData: Record<string, string>) {
    const res = await api.post<LoginResponse>({
      url: '/auth/login',
      data: formData,
    })
    const { token, user } = res.data
    if (token) {
      setToken(token)
    }
    authStore.setUser({
      id: user.id,
      username: user.username,
      displayName: user.displayName || user.nickname || user.username,
      avatarUrl: user.avatarUrl || user.avatar,
      roles: user.roles,
      permissions: user.permissions,
    })
    router.push('/')
  }

  function logout() {
    removeToken()
    authStore.setUser(null)
    router.push('/auth/sign-in')
  }

  return {
    login,
    logout,
  }
}
