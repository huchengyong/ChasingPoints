<template>
  <div class="matches-page">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <span>对局管理</span>
          <div class="filter-bar">
            <el-select v-model="filterStatus" placeholder="状态筛选" clearable style="width: 120px; margin-right: 10px;">
              <el-option label="全部" :value="-1" />
              <el-option label="进行中" :value="1" />
              <el-option label="已完成" :value="2" />
              <el-option label="已取消" :value="3" />
            </el-select>
            <el-select v-model="filterGameType" placeholder="球种筛选" clearable style="width: 140px; margin-right: 10px;">
              <el-option label="全部" :value="-1" />
              <el-option label="斯诺克" :value="1" />
              <el-option label="九球追分" :value="2" />
              <el-option label="中式八球" :value="3" />
              <el-option label="美式九球" :value="4" />
            </el-select>
            <el-button type="primary" @click="handleFilter">查询</el-button>
          </div>
        </div>
      </template>

      <el-table :data="matchList" stripe border>
        <el-table-column type="index" width="60" label="序号" :index="(index) => (page - 1) * pageSize + index + 1" />
        <el-table-column prop="game_type_name" label="对局类型" width="100" />
        <el-table-column label="选手">
          <template #default="{ row }">
            {{ row.player1_name }} vs {{ row.player2_name }}
          </template>
        </el-table-column>
        <el-table-column prop="winner_name" label="胜者" width="100" />
        <el-table-column label="比分" width="100">
          <template #default="{ row }">
            {{ row.my_score }} : {{ row.opponent_score }}
          </template>
        </el-table-column>
        <el-table-column prop="status_text" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 2 ? 'success' : row.status === 1 ? 'warning' : 'info'">
              {{ row.status_text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="match_time" label="比赛时间" width="150" />
        <el-table-column prop="created_at" label="创建时间" width="150" />
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getMatchList } from '@/api/match'
import type { Match, MatchListParams } from '@/api/match'

const loading = ref(false)
const matchList = ref<Match[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterStatus = ref(-1)
const filterGameType = ref(-1)

const loadData = async () => {
  loading.value = true
  try {
    const params: MatchListParams = {
      page: page.value,
      page_size: pageSize.value,
      status: filterStatus.value,
      game_type: filterGameType.value
    }
    const res = await getMatchList(params)
    matchList.value = res.list
    total.value = res.total
  } catch (error: any) {
    ElMessage.error(error.message || '获取对局列表失败')
  } finally {
    loading.value = false
  }
}

const handleFilter = () => {
  page.value = 1
  loadData()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  loadData()
}

const handlePageChange = (val: number) => {
  page.value = val
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="scss">
.matches-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-bar {
  display: flex;
  align-items: center;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>