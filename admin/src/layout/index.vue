<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside width="200px" class="sidebar">
      <div class="logo">
        <el-icon size="24"><Trophy /></el-icon>
        <span>追分</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :default-openeds="openMenuGroups"
        router
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <template v-for="item in menuList" :key="item.path">
          <el-sub-menu v-if="item.children" :index="item.path">
            <template #title>
              <el-icon>
                <component :is="item.icon" />
              </el-icon>
              <span>{{ item.title }}</span>
            </template>
            <el-menu-item v-for="child in item.children" :key="child.path" :index="child.path">
              <span>{{ child.title }}</span>
            </el-menu-item>
          </el-sub-menu>
          <el-menu-item v-else :index="item.path">
            <el-icon>
              <component :is="item.icon" />
            </el-icon>
            <span>{{ item.title }}</span>
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>

    <el-container>
      <!-- 顶部导航 -->
      <el-header class="header">
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              {{ userStore.userInfo?.nickname || '管理员' }}
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 主内容区 -->
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { canShowSocialReviewMenu } from '@/utils/complianceMode'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const activeMenu = computed(() => route.path)
const openMenuGroups = computed(() => {
  if (route.path.startsWith('/venues/')) return ['/venues']
  if (route.path.startsWith('/settings/')) return ['/settings']
  return []
})

const menuList = [
  { path: '/dashboard', title: '首页', icon: 'HomeFilled' },
  { path: '/users', title: '用户管理', icon: 'UserFilled' },
  { path: '/matches', title: '对局管理', icon: 'Trophy' },
  { path: '/feedback', title: '投诉举报', icon: 'WarningFilled' },
  { path: '/event-news', title: '赛事情报', icon: 'Calendar' },
  {
    path: '/venues',
    title: '球馆管理',
    icon: 'OfficeBuilding',
    children: [
      { path: '/venues/review', title: '球馆审核' },
      { path: '/venues/reward-records', title: '奖励发放记录' }
    ]
  },
  {
    path: '/settings',
    title: '配置管理',
    icon: 'Setting',
    children: [
      { path: '/settings/member-rewards', title: '会员奖励配置' }
    ]
  },
  // 合规收口：动态审核菜单暂时隐藏，页面文件保留以便后续持证后快速恢复。
  ...(canShowSocialReviewMenu() ? [{ path: '/social-posts', title: '动态审核', icon: 'ChatDotRound' }] : [])
]

const handleCommand = (command: string) => {
  if (command === 'logout') {
    userStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped lang="scss">
.layout-container {
  height: 100vh;
}

.sidebar {
  background-color: #304156;
  
  .logo {
    height: 60px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-size: 18px;
    font-weight: bold;
    border-bottom: 1px solid #1f2d3d;
    
    .el-icon {
      margin-right: 8px;
    }
  }
  
  .el-menu {
    border-right: none;
  }
}

.header {
  background-color: #fff;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  
  .header-right {
    .user-info {
      cursor: pointer;
      color: #606266;
      
      .el-icon {
        margin-left: 4px;
      }
    }
  }
}

.main-content {
  background-color: #f0f2f5;
  padding: 20px;
  overflow-y: auto;
}
</style>
