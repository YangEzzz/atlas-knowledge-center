import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { getCurrentUser } from '@/api/auth'
import { clearLegacyAuth, safeReturnPath } from '@/utils/session'
import { canManage } from '@/utils/management-access'
import { cancelPageScrollRestoration, restorePageScroll } from '@/utils/page-scroll'
import { toast } from 'vue-sonner'

const pageScrollPositions = new Map<number, { top: number, left: number }>()
let currentHistoryPosition = window.history.state?.position ?? 0

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (_, __, savedPosition) => {
    const position = savedPosition
      ? pageScrollPositions.get(window.history.state?.position) || { top: 0, left: 0 }
      : { top: 0, left: 0 }
    restorePageScroll(position)
    return false
  },
  routes: [
    {
      path: '/auth/sso/complete',
      component: () => import('@/views/Auth/SSOComplete.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/sign-in',
      name: 'SignIn',
      component: () => import('@/views/Auth/SignIn.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      children: [
        {path:'my-accounts',name:'PersonalAccounts',component:()=>import('@/views/Accounts/index.vue')},
        {
          path: '',
          name: 'Home',
          component: () => import('@/views/Home/index.vue'),
        },
        {
          path: 'projects',
          name: 'Projects',
          component: () => import('@/views/Projects/index.vue'),
        },
        {
          path: 'projects/:id',
          name: 'ProjectDetail',
          component: () => import('@/views/Projects/Detail.vue'),
        },
        {
          path: 'documents',
          name: 'Documents',
          component: () => import('@/views/Documents/index.vue'),
        },
        {
          path: 'documents/editor',
          name: 'DocumentEditor',
          component: () => import('@/views/Documents/Editor.vue'),
        },
        {
          path: 'documents/:id',
          name: 'DocumentDetail',
          component: () => import('@/views/Documents/Detail.vue'),
        },
        {
          path: 'public-links',
          name: 'PublicLinks',
          component: () => import('@/views/PublicLinks/index.vue'),
        },
        {
          path: 'favorites', redirect: '/my-entries',
        },
        {
          path: 'my-entries', name: 'PersonalEntries', component: () => import('@/views/Entries/index.vue'),
        },
        {
          path: 'recent-visits',
          name: 'RecentVisits',
          component: () => import('@/views/RecentVisits/index.vue'),
        },
        {
          path: 'manage',
          name: 'Manage',
          component: () => import('@/views/Manage/index.vue'),
        },
      ],
    },
  ],
})

router.beforeEach(() => {
  cancelPageScrollRestoration()
  const viewport = document.getElementById('app')
  if (viewport)
    pageScrollPositions.set(currentHistoryPosition, { top: viewport.scrollTop, left: viewport.scrollLeft })
})
router.afterEach((_, __, failure) => {
  if (!failure)
    currentHistoryPosition = window.history.state?.position ?? 0
})

router.beforeEach(async (to) => {
  document.title = to.path.startsWith('/auth/') ? '登录 · Atlas 研发知识中心' : 'Atlas 研发知识中心'
  if (to.path === '/auth/sso/complete') return true
  const store = useAuthStore()
  if (!store.initialized) {
    clearLegacyAuth()
    try { store.setUser(await getCurrentUser()) } catch { store.setUser(null) }
    store.initialized = true
  }
  if (to.meta.public) {
    return store.isLogin ? safeReturnPath(to.query.redirect) : true
  }
  if (!store.isLogin) {
    return {
      path: '/auth/sign-in',
      query: { redirect: to.fullPath },
    }
  }
  if (to.name === 'Manage' && !canManage(store.user?.roles)) {
    toast.error('仅 Atlas 管理员可使用管理中心')
    return { name: 'Home' }
  }
  return true
})

export default router
