<template>
  <div class="auth-page">
    <form class="auth-card" @submit.prevent="handleSubmit">
      <div class="auth-brand">
        <div class="brand-mark">
          <AtlasLogo class="brand-icon" />
        </div>
        <div>
          <h1>Atlas 研发知识中心</h1>
          <p>登录后查看项目、入口和文档。</p>
        </div>
      </div>

      <label class="ui-field">
        <span>用户名</span>
        <Input v-model="form.username" class="ui-input" placeholder="请输入用户名" />
      </label>
      <label class="ui-field">
        <span>密码</span>
        <Input v-model="form.password" class="ui-input" type="password" placeholder="请输入密码" />
      </label>
      <label class="ui-field">
        <span>项目映射 ID</span>
        <Input v-model="form.project_id_string" class="ui-input" placeholder="按现有登录接口要求填写" />
      </label>

      <Button type="submit" size="lg" :disabled="loading">
        <LogIn class="button-icon" />
        {{ loading ? '登录中...' : '登录' }}
      </Button>

      <Button variant="secondary" type="button" @click="mockLogin">
        本地占位登录
      </Button>

      <p v-if="errorMessage" class="form-error">{{ errorMessage }}</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { LogIn } from 'lucide-vue-next'
import AtlasLogo from '@/components/app/AtlasLogo.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuth } from '@/composables/use-auth'
import { useAuthStore } from '@/store/auth'
import { setToken } from '@/utils/auth'

const router = useRouter()
const authStore = useAuthStore()
const { login } = useAuth()
const loading = ref(false)
const errorMessage = ref('')

const form = reactive({
  username: '',
  password: '',
  project_id_string: '',
})

async function handleSubmit() {
  loading.value = true
  errorMessage.value = ''
  try {
    await login(form)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录失败'
  } finally {
    loading.value = false
  }
}

function mockLogin() {
  setToken('local-dev-token')
  authStore.setUser({
    id: 'local',
    username: 'developer',
    displayName: '研发用户',
    roles: ['editor'],
    permissions: ['*'],
  })
  router.push('/')
}
</script>
