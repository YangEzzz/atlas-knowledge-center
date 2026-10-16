import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface CurrentUser {
  id: string | number
  username?: string
  displayName?: string
  avatarUrl?: string
  roles?: string[]
  permissions?: string[]
}

export const useAuthStore = defineStore(
  'auth',
  () => {
    const isLogin = ref(false)
    const initialized = ref(false)
    const user = ref<CurrentUser | null>(null)

    function setUser(nextUser: CurrentUser | null) {
      user.value = nextUser
      isLogin.value = !!nextUser
    }

    return {
      isLogin,
      initialized,
      user,
      setUser,
    }
  },
  {
    persist: false,
  },
)
