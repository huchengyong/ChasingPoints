<template>
  <div class="venues-page">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <span>会员奖励配置</span>
        </div>
      </template>

      <el-form :model="rewardConfig" label-width="120px" class="config-form" v-loading="configLoading">
        <div class="config-section-title">常玩球馆奖励</div>
        <div class="config-grid">
          <el-form-item label="奖励开关">
            <el-switch v-model="rewardConfig.enabled" />
          </el-form-item>
          <el-form-item label="弹窗开关">
            <el-switch v-model="rewardConfig.popup_enabled" />
          </el-form-item>
          <el-form-item label="奖励天数">
            <el-input-number v-model="rewardConfig.reward_days" :min="1" :max="365" />
          </el-form-item>
          <el-form-item label="开始时间">
            <el-date-picker
              v-model="rewardConfig.start_at"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="可选"
            />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-date-picker
              v-model="rewardConfig.end_at"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="可选"
            />
          </el-form-item>
        </div>
        <div class="config-section-title">新用户注册奖励</div>
        <div class="config-grid">
          <el-form-item label="奖励开关">
            <el-switch v-model="rewardConfig.welcome_reward_enabled" />
          </el-form-item>
          <el-form-item label="奖励天数">
            <el-input-number v-model="rewardConfig.welcome_reward_days" :min="1" :max="365" />
          </el-form-item>
        </div>
        <div class="config-actions">
          <el-button type="primary" :loading="configSaving" @click="handleSaveRewardConfig">保存配置</el-button>
        </div>
      </el-form>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>常玩球馆奖励发放记录</span>
        </div>
      </template>

      <el-table :data="rewardRecordList" stripe border v-loading="recordLoading">
        <el-table-column prop="user_nickname" label="用户昵称" min-width="140" />
        <el-table-column prop="user_phone" label="手机号" width="140" />
        <el-table-column prop="venue_name" label="球馆" min-width="160" />
        <el-table-column prop="reward_days" label="奖励时长" width="110" align="center">
          <template #default="{ row }">
            {{ row.reward_days }} 天
          </template>
        </el-table-column>
        <el-table-column prop="member_expires_at_before" label="发放前到期" width="170">
          <template #default="{ row }">
            {{ row.member_expires_at_before || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="member_expires_at_after" label="发放后到期" width="170" />
        <el-table-column prop="granted_at" label="发放时间" width="170" />
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="recordPage"
          v-model:page-size="recordPageSize"
          :total="recordTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="handleRecordSizeChange"
          @current-change="handleRecordPageChange"
        />
      </div>
    </el-card>

    <el-card>
      <template #header>
        <div class="card-header">
          <span>球馆审核管理</span>
          <div class="header-filters">
            <el-select v-model="filterStatus" placeholder="审核状态" clearable @change="handleFilterChange">
              <el-option label="全部" :value="-1" />
              <el-option label="待审核" :value="2" />
              <el-option label="已通过" :value="1" />
              <el-option label="已拒绝" :value="3" />
              <el-option label="隐藏" :value="0" />
            </el-select>
          </div>
        </div>
      </template>

      <el-table :data="venueList" stripe border v-loading="loading">
        <el-table-column type="index" width="60" label="序号" />
        <el-table-column prop="name" label="球馆名称" min-width="150" />
        <el-table-column prop="city" label="城市" width="100" />
        <el-table-column prop="district" label="区域" width="120" />
        <el-table-column prop="address" label="详细地址" min-width="200" show-overflow-tooltip />
        <el-table-column prop="table_count" label="球桌数" width="80" align="center" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="提交时间" width="160" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button 
              v-if="row.status === 2" 
              link 
              type="success" 
              @click="handleReview(row, 1)"
            >
              通过
            </el-button>
            <el-button 
              v-if="row.status === 2" 
              link 
              type="danger" 
              @click="handleReview(row, 3)"
            >
              拒绝
            </el-button>
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

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" title="球馆详情" width="600px">
      <el-descriptions :column="2" border v-if="currentVenue">
        <el-descriptions-item label="球馆名称" :span="2">{{ currentVenue.name }}</el-descriptions-item>
        <el-descriptions-item label="城市">{{ currentVenue.city }}</el-descriptions-item>
        <el-descriptions-item label="区域">{{ currentVenue.district }}</el-descriptions-item>
        <el-descriptions-item label="详细地址" :span="2">{{ currentVenue.address }}</el-descriptions-item>
        <el-descriptions-item label="联系电话">{{ currentVenue.phone || '-' }}</el-descriptions-item>
        <el-descriptions-item label="球桌数量">{{ currentVenue.table_count || 0 }}</el-descriptions-item>
        <el-descriptions-item label="营业时间" :span="2">{{ currentVenue.business_hours || '-' }}</el-descriptions-item>
        <el-descriptions-item label="价格区间">{{ currentVenue.price_range || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(currentVenue.status)">
            {{ getStatusLabel(currentVenue.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="拒绝原因">{{ currentVenue.reject_reason || '-' }}</el-descriptions-item>
        <el-descriptions-item label="提交时间" :span="2">{{ currentVenue.created_at }}</el-descriptions-item>
        <el-descriptions-item label="简介" :span="2">{{ currentVenue.description || '-' }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button 
          v-if="currentVenue && currentVenue.status === 2" 
          type="success" 
          @click="handleReview(currentVenue, 1); detailVisible = false"
        >
          审核通过
        </el-button>
        <el-button 
          v-if="currentVenue && currentVenue.status === 2" 
          type="danger" 
          @click="handleReview(currentVenue, 3); detailVisible = false"
        >
          审核拒绝
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getVenueList,
  getVenueRewardConfig,
  getVenueRewardRecordList,
  reviewVenue,
  updateVenueRewardConfig,
  type Venue,
  type VenueRewardConfig,
  type VenueRewardRecord
} from '@/api/venue'

const loading = ref(false)
const venueList = ref<Venue[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterStatus = ref<number>(-1)
const configLoading = ref(false)
const configSaving = ref(false)
const recordLoading = ref(false)
const rewardRecordList = ref<VenueRewardRecord[]>([])
const recordPage = ref(1)
const recordPageSize = ref(10)
const recordTotal = ref(0)

const detailVisible = ref(false)
const currentVenue = ref<Venue | null>(null)
const rewardConfig = reactive<VenueRewardConfig>({
  enabled: false,
  popup_enabled: false,
  reward_days: 30,
  welcome_reward_enabled: true,
  welcome_reward_days: 7,
  start_at: '',
  end_at: ''
})

const statusMap: Record<number, { label: string; type: 'success' | 'warning' | 'danger' | 'info' }> = {
  0: { label: '隐藏', type: 'info' },
  1: { label: '已通过', type: 'success' },
  2: { label: '待审核', type: 'warning' },
  3: { label: '已拒绝', type: 'danger' }
}

const getStatusLabel = (status: number) => {
  return statusMap[status]?.label || '未知'
}

const getStatusType = (status: number): 'success' | 'warning' | 'danger' | 'info' => {
  return statusMap[status]?.type || 'info'
}

const fetchVenueList = async () => {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value
    }
    if (filterStatus.value !== -1) {
      params.status = filterStatus.value
    }
    const res = await getVenueList(params)
    if (res.success) {
      venueList.value = res.list
      total.value = res.total
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取球馆列表失败')
  } finally {
    loading.value = false
  }
}

const fetchRewardConfig = async () => {
  configLoading.value = true
  try {
    const res = await getVenueRewardConfig()
    if (res.success) {
      rewardConfig.enabled = !!res.enabled
      rewardConfig.popup_enabled = !!res.popup_enabled
      rewardConfig.reward_days = res.reward_days || 30
      rewardConfig.welcome_reward_enabled = res.welcome_reward_enabled ?? true
      rewardConfig.welcome_reward_days = res.welcome_reward_days || 7
      rewardConfig.start_at = res.start_at || ''
      rewardConfig.end_at = res.end_at || ''
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取奖励配置失败')
  } finally {
    configLoading.value = false
  }
}

const fetchRewardRecordList = async () => {
  recordLoading.value = true
  try {
    const res = await getVenueRewardRecordList({
      page: recordPage.value,
      page_size: recordPageSize.value
    })
    if (res.success) {
      rewardRecordList.value = res.list
      recordTotal.value = res.total
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取奖励记录失败')
  } finally {
    recordLoading.value = false
  }
}

const handleSaveRewardConfig = async () => {
  configSaving.value = true
  try {
    const res = await updateVenueRewardConfig({ ...rewardConfig })
    if (res.success) {
      ElMessage.success(res.message || '奖励配置已保存')
      fetchRewardConfig()
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '保存失败')
  } finally {
    configSaving.value = false
  }
}

const handleFilterChange = () => {
  page.value = 1
  fetchVenueList()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  fetchVenueList()
}

const handlePageChange = (val: number) => {
  page.value = val
  fetchVenueList()
}

const handleRecordSizeChange = (val: number) => {
  recordPageSize.value = val
  fetchRewardRecordList()
}

const handleRecordPageChange = (val: number) => {
  recordPage.value = val
  fetchRewardRecordList()
}

const handleReview = async (venue: Venue, status: number) => {
  const action = status === 1 ? '通过' : '拒绝'
  try {
    let rejectReason = ''
    if (status === 3) {
      const prompt = await ElMessageBox.prompt(
        `请填写拒绝"${venue.name}"的原因`,
        '审核拒绝',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputPlaceholder: '例如：地址信息不完整',
          inputValue: venue.reject_reason || ''
        }
      )
      rejectReason = prompt.value.trim()
    }

    await ElMessageBox.confirm(
      `确定要${action}"${venue.name}"的审核吗？`,
      '确认审核',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: status === 1 ? 'success' : 'warning'
      }
    )
    
    const res = await reviewVenue({
      venue_id: venue.id,
      status,
      reject_reason: rejectReason || undefined
    })
    
    if (res.success) {
      ElMessage.success(res.message || `审核${action}成功`)
      fetchVenueList()
      fetchRewardRecordList()
    } else {
      ElMessage.error(res.message || '操作失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '操作失败')
    }
  }
}

const showDetail = (venue: Venue) => {
  currentVenue.value = venue
  detailVisible.value = true
}

onMounted(() => {
  fetchRewardConfig()
  fetchRewardRecordList()
  fetchVenueList()
})
</script>

<style scoped lang="scss">
.venues-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
  margin-top: 4px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 16px;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
}

.header-filters {
  display: flex;
  gap: 12px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
