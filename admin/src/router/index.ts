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
        path: 'feedback',
        name: 'FeedbackTickets',
        component: () => import('@/views/feedback/index.vue'),
        meta: { title: '投诉举报', icon: 'WarningFilled' }
      },
      {
        path: 'reputation-logs',
        name: 'ReputationLogs',
        component: () => import('@/views/reputation-logs/index.vue'),
        meta: { title: '信誉日志', icon: 'Document' }
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
        redirect: '/venues/review',
        meta: { title: '球馆管理', icon: 'OfficeBuilding' }
      },
      {
        path: 'venues/review',
        name: 'VenueReview',
        component: () => import('@/views/venues/review.vue'),
        meta: { title: '球馆审核', parent: '/venues' }
      },
      {
        path: 'venues/reward-records',
        name: 'VenueRewardRecords',
        component: () => import('@/views/venues/reward-records.vue'),
        meta: { title: '奖励发放记录', parent: '/venues' }
      },
      {
        path: 'settings',
        name: 'Settings',
        redirect: '/settings/member-rewards',
        meta: { title: '配置管理', icon: 'Setting' }
      },
      {
        path: 'settings/member-rewards',
        name: 'MemberRewardSettings',
        component: () => import('@/views/config/member-rewards.vue'),
        meta: { title: '会员奖励配置', parent: '/settings' }
      },
      {
        path: 'settings/member-ranking-rights',
        name: 'MemberRankingRightsSettings',
        component: () => import('@/views/config/member-ranking-rights.vue'),
        meta: { title: '会员排位权益配置', parent: '/settings' }
      },
      {
        path: 'settings/reputation',
        name: 'ReputationSettings',
        component: () => import('@/views/config/reputation.vue'),
        meta: { title: '信誉制度配置', parent: '/settings' }
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
