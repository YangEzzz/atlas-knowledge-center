import { createRouter, createWebHistory } from 'vue-router'
import { isLoggedIn } from '@/utils/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
          path: 'favorites',
          name: 'Favorites',
          component: () => import('@/views/Favorites/index.vue'),
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

router.beforeEach((to) => {
  if (to.meta.public) {
    return true
  }
  if (!isLoggedIn()) {
    return {
      path: '/auth/sign-in',
      query: { redirect: to.fullPath },
    }
  }
  return true
})

export default router
