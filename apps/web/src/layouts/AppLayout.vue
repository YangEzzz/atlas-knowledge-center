<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import {
  BookOpen,
  Clock3,
  FolderKanban,
  KeyRound,
  LayoutDashboard,
  Link2,
  LogOut,
  PanelLeftOpen,
  Search,
  Settings,
} from "lucide-vue-next";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import AtlasLogo from "@/components/app/AtlasLogo.vue";
import GlobalSearch from "@/components/app/GlobalSearch.vue";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/composables/use-auth";
import { useAuthStore } from "@/store/auth";
import { canManage } from "@/utils/management-access";
import { openSearch } from "@/composables/use-search-dialog";
const authStore = useAuthStore();
const { logout, loading: logoutLoading } = useAuth();
const route = useRoute();
const mobileOpen = ref(false);
const groups = [
  {
    label: "工作空间",
    items: [
      { label: "首页", path: "/", icon: LayoutDashboard },
      { label: "项目库", path: "/projects", icon: FolderKanban },
      { label: "文档库", path: "/documents", icon: BookOpen },
      { label: "工具导航", path: "/public-links", icon: Link2 },
    ],
  },
  {
    label: "我的空间",
    items: [
      { label: "我的入口", path: "/my-entries", icon: Link2 },
      { label: "我的账号", path: "/my-accounts", icon: KeyRound },
      { label: "最近访问", path: "/recent-visits", icon: Clock3 },
    ],
  },
];
const userName = computed(
  () => authStore.user?.displayName || authStore.user?.username || "研发用户",
);
const userInitial = computed(() =>
  Array.from(userName.value).slice(0, 2).join(""),
);
function onKey(event: KeyboardEvent) {
  if (event.isComposing) return;
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    openSearch();
  }
}
onMounted(() => window.addEventListener("keydown", onKey));
onBeforeUnmount(() => window.removeEventListener("keydown", onKey));
</script>

<template>
  <div class="app-shell">
    <a href="#main-content" class="skip-link">跳到主要内容</a>
    <aside class="sidebar" aria-label="主导航">
      <RouterLink to="/" class="brand"
        ><AtlasLogo class="brand-icon" /><span
          ><strong class="brand-title">Atlas</strong
          ><span class="brand-subtitle">研发知识中心</span></span
        ></RouterLink
      >
      <button type="button" class="sidebar-search" @click="openSearch()">
        <Search aria-hidden="true" /><span>搜索 Atlas</span><kbd>Ctrl K</kbd>
      </button>
      <nav
        v-for="group in groups"
        :key="group.label"
        class="nav-group"
        :aria-label="group.label"
      >
        <p class="nav-group-label">{{ group.label }}</p>
        <div class="nav-list">
          <RouterLink
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            :class="[
              'nav-item',
              {
                'is-current':
                  item.path === '/'
                    ? route.path === '/'
                    : route.path.startsWith(item.path),
              },
            ]"
            ><component :is="item.icon" class="nav-icon" aria-hidden="true" />{{
              item.label
            }}</RouterLink
          >
        </div>
      </nav>
      <div class="sidebar-footer">
        <RouterLink v-if="canManage(authStore.user?.roles)" to="/manage" class="nav-item"
          ><Settings class="nav-icon" aria-hidden="true" />管理中心</RouterLink
        >
        <div class="sidebar-user">
          <span class="user-avatar" aria-hidden="true">{{ userInitial }}</span>
          <div>
            <strong>{{ userName }}</strong
            ><span>我的工作空间</span>
          </div>
          <Button
            variant="ghost"
            size="icon"
            aria-label="退出登录"
            :disabled="logoutLoading"
            @click="logout"
            ><LogOut
          /></Button>
        </div>
      </div>
    </aside>
    <div class="main-shell">
      <main id="main-content" class="page-container" tabindex="-1">
        <Button
          class="mobile-menu-button mobile-navigation-trigger"
          variant="ghost"
          aria-label="打开导航"
          @click="mobileOpen = true"
        ><PanelLeftOpen />导航</Button>
        <RouterView />
      </main>
    </div>
    <Dialog v-model:open="mobileOpen"
      ><DialogContent class="mobile-navigation"
        ><DialogTitle>Atlas 导航</DialogTitle
        ><DialogDescription>项目、文档与我的工作空间。</DialogDescription>
        <Button variant="secondary" @click="mobileOpen = false; openSearch()">
          <Search aria-hidden="true" />搜索 Atlas
        </Button>
        <nav
          v-for="group in groups"
          :key="group.label"
          :aria-label="group.label"
        >
          <p class="nav-group-label">{{ group.label }}</p>
          <RouterLink
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="nav-item"
            @click="mobileOpen = false"
            ><component :is="item.icon" class="nav-icon" />{{
              item.label
            }}</RouterLink
          >
        </nav>
        <RouterLink v-if="canManage(authStore.user?.roles)" to="/manage" class="nav-item" @click="mobileOpen = false"
          ><Settings class="nav-icon" />管理中心</RouterLink
        ><Button variant="secondary" :disabled="logoutLoading" @click="logout"
          >退出登录</Button
        ></DialogContent
      ></Dialog
    >
    <GlobalSearch />
  </div>
</template>
