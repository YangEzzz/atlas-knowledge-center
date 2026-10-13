<template>
  <div class="app-shell">
    <button v-if="mobileOpen" class="sidebar-scrim" type="button" aria-label="关闭导航" @click="mobileOpen = false" />

    <aside :class="['sidebar', { 'is-open': mobileOpen }]">
      <div class="brand">
        <div class="brand-mark">
          <AtlasLogo class="brand-icon" />
        </div>
        <div>
          <div class="brand-title">Atlas 研发知识中心</div>
          <div class="brand-subtitle">Engineering Knowledge Hub</div>
        </div>
      </div>

      <nav class="nav-list">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          @click="mobileOpen = false"
        >
          <component :is="item.icon" class="nav-icon" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>
    </aside>

    <div class="main-shell">
      <header class="topbar">
        <Button class="mobile-menu-button" variant="ghost" size="icon" @click="mobileOpen = !mobileOpen">
          <PanelLeftOpen />
        </Button>
        <div class="global-search">
          <Search class="search-icon" />
          <Input class="topbar-search-input" placeholder="搜索项目、GitLab、测试地址、文档或负责人" />
        </div>
        <div class="user-area">
          <span class="user-name">{{ authStore.user?.displayName || '研发用户' }}</span>
          <Button variant="ghost" size="sm" @click="logout">退出</Button>
        </div>
      </header>

      <main class="page-container">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import {
  BookOpen,
  Clock3,
  FolderKanban,
  Heart,
  LayoutDashboard,
  Link2,
  PanelLeftOpen,
  Search,
  Settings,
} from 'lucide-vue-next'
import AtlasLogo from '@/components/app/AtlasLogo.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuth } from '@/composables/use-auth'
import { useAuthStore } from '@/store/auth'

const authStore = useAuthStore()
const { logout } = useAuth()

const navItems = [
  { label: '首页', path: '/', icon: LayoutDashboard },
  { label: '项目库', path: '/projects', icon: FolderKanban },
  { label: '文档库', path: '/documents', icon: BookOpen },
  { label: '工具导航', path: '/public-links', icon: Link2 },
  { label: '我的收藏', path: '/favorites', icon: Heart },
  { label: '最近访问', path: '/recent-visits', icon: Clock3 },
  { label: '管理中心', path: '/manage', icon: Settings },
]

const mobileOpen = ref(false)
</script>
