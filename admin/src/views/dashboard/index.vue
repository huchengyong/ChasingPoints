<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="6" v-for="item in statistics" :key="item.title">
        <el-card class="stat-card">
          <div class="stat-icon" :style="{ backgroundColor: item.color }">
            <el-icon size="24"><component :is="item.icon" /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-title">{{ item.title }}</div>
            <div class="stat-value">{{ item.value }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="12">
        <el-card v-loading="loadingUsers">
          <template #header>最近注册用户</template>
          <el-table :data="recentUsers" stripe>
            <el-table-column prop="nickname" label="昵称" />
            <el-table-column prop="phone" label="手机号" />
            <el-table-column prop="created_at" label="注册时间" />
          </el-table>
        </el-card>
      </el-col>
      
      <el-col :span="12">
        <el-card v-loading="loadingMatches">
          <template #header>最近对局</template>
          <el-table :data="recentMatches" stripe>
            <el-table-column prop="game_type_name" label="类型" />
            <el-table-column label="选手">
              <template #default="{ row }">
                {{ row.player1_name }} vs {{ row.player2_name }}
              </template>
            </el-table-column>
            <el-table-column label="结果">
              <template #default="{ row }">
                {{ row.my_score }} : {{ row.opponent_score }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { UserFilled, User, Trophy, VideoPlay } from '@element-plus/icons-vue'
import { getDashboardStats, getRecentUsers, getRecentMatches } from '@/api/dashboard'
import type { RecentUser, RecentMatch } from '@/api/dashboard'

const loadingUsers = ref(false)
const loadingMatches = ref(false)

const statistics = ref([
  { title: '总用户数', value: '0', icon: UserFilled, color: '#409EFF' },
  { title: '今日新增', value: '0', icon: User, color: '#67C23A' },
  { title: '总对局数', value: '0', icon: Trophy, color: '#E6A23C' },
  { title: '进行中', value: '0', icon: VideoPlay, color: '#F56C6C' }
])

const recentUsers = ref<RecentUser[]>([])
const recentMatches = ref<RecentMatch[]>([])

const loadStats = async () => {
  try {
    const stats = await getDashboardStats()
    statistics.value[0].value = stats.total_users.toLocaleString()
    statistics.value[1].value = stats.today_new_users.toLocaleString()
    statistics.value[2].value = stats.total_matches.toLocaleString()
    statistics.value[3].value = stats.ongoing_matches.toLocaleString()
  } catch (error: any) {
    ElMessage.error('获取统计数据失败')
  }
}

const loadRecentUsers = async () => {
  loadingUsers.value = true
  try {
    recentUsers.value = await getRecentUsers()
  } catch (error: any) {
    ElMessage.error('获取最近用户失败')
  } finally {
    loadingUsers.value = false
  }
}

const loadRecentMatches = async () => {
  loadingMatches.value = true
  try {
    recentMatches.value = await getRecentMatches()
  } catch (error: any) {
    ElMessage.error('获取最近对局失败')
  } finally {
    loadingMatches.value = false
  }
}

onMounted(() => {
  loadStats()
  loadRecentUsers()
  loadRecentMatches()
})
</script>

<style scoped lang="scss">
.dashboard {
  padding: 20px;
}

.stat-card {
  display: flex;
  align-items: center;
  
  :deep(.el-card__body) {
    display: flex;
    align-items: center;
    width: 100%;
  }
  
  .stat-icon {
    width: 60px;
    height: 60px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    margin-right: 16px;
  }
  
  .stat-info {
    .stat-title {
      font-size: 14px;
      color: #909399;
      margin-bottom: 8px;
    }
    
    .stat-value {
      font-size: 24px;
      font-weight: bold;
      color: #303133;
    }
  }
}
</style>
