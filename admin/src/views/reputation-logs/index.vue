<template>
  <div class="reputation-logs-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>信誉日志</span>
          <div class="header-filters">
            <el-input
              v-model="filterUserId"
              placeholder="用户ID"
              clearable
              style="width: 140px"
              @keyup.enter="handleFilterChange"
              @clear="handleFilterChange"
            />
            <el-select
              v-model="filterChangeType"
              placeholder="变更类型"
              clearable
              style="width: 140px"
              @change="handleFilterChange"
            >
              <el-option
                v-for="option in changeTypeOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <el-select
              v-model="filterReasonCode"
              placeholder="原因类型"
              clearable
              style="width: 180px"
              @change="handleFilterChange"
            >
              <el-option
                v-for="option in reasonCodeOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <el-button type="primary" @click="handleFilterChange">查询</el-button>
          </div>
        </div>
      </template>

      <el-table :data="logList" stripe border v-loading="loading">
        <el-table-column
          type="index"
          width="60"
          label="序号"
          :index="(index: number) => (page - 1) * pageSize + index + 1"
        />
        <el-table-column prop="user_id" label="用户ID" width="96" />
        <el-table-column prop="nickname" label="昵称" width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.nickname || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="change_type_text" label="变更类型" width="120">
          <template #default="{ row }">
            <el-tag :type="changeTypeTagType(row.change_type)">
              {{ row.change_type_text || '-' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason_text" label="原因" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.reason_text || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="分值变化" width="110">
          <template #default="{ row }">
            <span :class="scoreClass(row.change_score)">
              {{ formatScore(row.change_score) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="before_score" label="变更前" width="90" />
        <el-table-column prop="after_score" label="变更后" width="90" />
        <el-table-column label="对局ID" width="100">
          <template #default="{ row }">
            {{ row.match_id || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">详情</el-button>
          </template>
        </el-table-column>
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

    <el-dialog v-model="detailVisible" title="信誉日志详情" width="720px">
      <el-descriptions v-if="currentLog" :column="2" border>
        <el-descriptions-item label="用户ID">{{ currentLog.user_id }}</el-descriptions-item>
        <el-descriptions-item label="昵称">{{ currentLog.nickname || '-' }}</el-descriptions-item>
        <el-descriptions-item label="变更类型">{{ currentLog.change_type_text || '-' }}</el-descriptions-item>
        <el-descriptions-item label="原因">{{ currentLog.reason_text || '-' }}</el-descriptions-item>
        <el-descriptions-item label="分值变化">
          <span :class="scoreClass(currentLog.change_score)">
            {{ formatScore(currentLog.change_score) }}
          </span>
        </el-descriptions-item>
        <el-descriptions-item label="对局ID">{{ currentLog.match_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="变更前">{{ currentLog.before_score }}</el-descriptions-item>
        <el-descriptions-item label="变更后">{{ currentLog.after_score }}</el-descriptions-item>
        <el-descriptions-item label="操作人">{{ currentLog.operator_text || '-' }}</el-descriptions-item>
        <el-descriptions-item label="操作人ID">{{ currentLog.operator_admin_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ currentLog.created_at }}</el-descriptions-item>
        <el-descriptions-item label="原因码">{{ currentLog.reason_code || '-' }}</el-descriptions-item>
        <el-descriptions-item label="原因详情" :span="2">
          {{ currentLog.reason_detail || '-' }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getAdminReputationLogList,
  type AdminReputationLogItem,
  type AdminReputationLogListParams
} from '@/api/reputation-log'

const loading = ref(false)
const logList = ref<AdminReputationLogItem[]>([])
const currentLog = ref<AdminReputationLogItem | null>(null)
const detailVisible = ref(false)

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filterUserId = ref('')
const filterChangeType = ref('')
const filterReasonCode = ref('')

const changeTypeOptions = [
  { label: '处罚', value: 'penalty' },
  { label: '恢复', value: 'recovery' },
  { label: '人工调整', value: 'manual_adjust' }
]

const reasonCodeOptions = [
  { label: '时长异常', value: 'duration_abnormal' },
  { label: '同对手高频', value: 'same_opponent_high_frequency' },
  { label: '系统恢复', value: 'system_recovery' },
  { label: '人工调整', value: 'manual_adjust' },
  { label: '异常比赛', value: 'abnormal_match' }
]

const parseUserId = () => {
  const value = filterUserId.value.trim()
  if (!value) {
    return undefined
  }
  if (!/^\d+$/.test(value)) {
    ElMessage.warning('用户ID 只能输入数字')
    return null
  }
  return Number(value)
}

const fetchLogList = async () => {
  const userId = parseUserId()
  if (userId === null) {
    logList.value = []
    total.value = 0
    return
  }

  loading.value = true
  try {
    const params: AdminReputationLogListParams = {
      page: page.value,
      page_size: pageSize.value
    }
    if (typeof userId === 'number') {
      params.user_id = userId
    }
    if (filterChangeType.value) {
      params.change_type = filterChangeType.value
    }
    if (filterReasonCode.value) {
      params.reason_code = filterReasonCode.value
    }

    const res = await getAdminReputationLogList(params)
    if (res.success) {
      logList.value = res.list || []
      total.value = res.total || 0
      return
    }
    logList.value = []
    total.value = 0
    ElMessage.error(res.message || '获取信誉日志失败')
  } catch (error: any) {
    logList.value = []
    total.value = 0
    ElMessage.error(error.message || '获取信誉日志失败')
  } finally {
    loading.value = false
  }
}

const handleFilterChange = () => {
  page.value = 1
  fetchLogList()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  fetchLogList()
}

const handlePageChange = (current: number) => {
  page.value = current
  fetchLogList()
}

const showDetail = (log: AdminReputationLogItem) => {
  currentLog.value = log
  detailVisible.value = true
}

const formatScore = (score: number) => (score > 0 ? `+${score}` : `${score}`)

const scoreClass = (score: number) => {
  if (score > 0) {
    return 'score-positive'
  }
  if (score < 0) {
    return 'score-negative'
  }
  return 'score-neutral'
}

const changeTypeTagType = (changeType: string) => {
  switch (changeType) {
    case 'penalty':
      return 'danger'
    case 'recovery':
      return 'success'
    case 'manual_adjust':
      return 'warning'
    default:
      return 'info'
  }
}

onMounted(() => {
  fetchLogList()
})
</script>

<style scoped lang="scss">
.reputation-logs-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.header-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.score-positive {
  color: #67c23a;
  font-weight: 600;
}

.score-negative {
  color: #f56c6c;
  font-weight: 600;
}

.score-neutral {
  color: #909399;
  font-weight: 600;
}
</style>
