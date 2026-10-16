<script setup lang="ts">
import { ShieldCheck, ArrowUpRight } from 'lucide-vue-next'
import AtlasLogo from '@/components/app/AtlasLogo.vue'
import { Button } from '@/components/ui/button'
import { useHeliosLogin } from '@/composables/use-helios-login'
const { login, cancel, loading, error } = useHeliosLogin()
</script>

<template>
  <main class="auth-page">
    <section class="auth-card" aria-labelledby="login-title">
      <div class="auth-brand">
        <div class="brand-mark"><AtlasLogo class="brand-icon" /></div>
        <div>
          <h1 id="login-title">Atlas 研发知识中心</h1>
          <p>登录后查看项目、入口和文档。</p>
        </div>
      </div>
      <p>使用 Helios 账号登录。已登录 Helios 时，可直接使用当前账号。</p>
      <div class="auth-status" aria-live="polite">
        <p v-if="error" class="form-error" role="alert">{{ error }}</p>
        <p v-else-if="loading">请在 Helios 窗口中完成登录，此页面将自动进入知识中心。</p>
      </div>
      <Button size="lg" :disabled="loading" :aria-busy="loading" @click="login()">
        <ShieldCheck class="button-icon" aria-hidden="true" />
        {{ loading ? '等待 Helios 登录...' : '使用 Helios 登录' }}
      </Button>
      <Button variant="secondary" :disabled="loading" @click="login(true)">
        在当前页面登录 <ArrowUpRight class="button-icon" aria-hidden="true" />
      </Button>
      <Button v-if="loading" variant="ghost" @click="cancel">取消登录</Button>
      <p class="auth-help">没有访问权限？请联系管理员将你的账号加入 Atlas 项目。</p>
    </section>
  </main>
</template>
