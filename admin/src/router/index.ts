import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/store/user'
import { canShowSocialReviewMenu } from '@/utils/complianceMode'

const socialReviewRoutes = canShowSocialReviewMenu()
  ? [{
      path: 'social-posts',
      name: 'SocialPosts',
      component: () => import('@/views/social-posts/index.vue'),
      meta: { title: '动态审核', icon: 'ChatDotRound' }
    }]
  : []

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { public: true }
  },
  {
    path: '/',
    name: 'Layout',
    component: () => import('@/layout/index.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '首页', icon: 'HomeFilled' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/users/index.vue'),
        meta: { title: '用户管理', icon: 'UserFilled' }
      },
      {
        path: 'matches',
        name: 'Matches',
        component: () => import('@/views/matches/index.vue'),
        meta: { title: '对局管理', icon: 'Trophy' }
      },
      {
        path: 'event-news',
        name: 'EventNews',
        component: () => import('@/views/event-news/index.vue'),
        meta: { title: '赛事情报', icon: 'Calendar' }
      },
      {
        path: 'venues',
        name: 'Venues',
        component: () => import('@/views/venues/index.vue'),
        meta: { title: '球馆审核', icon: 'OfficeBuilding' }
      }
    // 合规收口：动态审核入口暂时隐藏，页面文件保留以便后续持证后快速恢复。
    ].concat(socialReviewRoutes)
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()
  
  if (!to.meta.public && !userStore.token) {
    next('/login')
  } else {
    next()
  }
})

export default router
