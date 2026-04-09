<template>
  <div class="venue-review-page">
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
          @click="handleDialogReview(1)"
        >
          审核通过
        </el-button>
        <el-button
          v-if="currentVenue && currentVenue.status === 2"
          type="danger"
          @click="handleDialogReview(3)"
        >
          审核拒绝
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getVenueList, reviewVenue, type Venue } from '@/api/venue'

const loading = ref(false)
const venueList = ref<Venue[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterStatus = ref(-1)
const detailVisible = ref(false)
const currentVenue = ref<Venue | null>(null)

const statusMap: Record<number, { label: string; type: 'success' | 'warning' | 'danger' | 'info' }> = {
  0: { label: '隐藏', type: 'info' },
  1: { label: '已通过', type: 'success' },
  2: { label: '待审核', type: 'warning' },
  3: { label: '已拒绝', type: 'danger' }
}

const getStatusLabel = (status: number) => statusMap[status]?.label || '未知'
const getStatusType = (status: number): 'success' | 'warning' | 'danger' | 'info' => statusMap[status]?.type || 'info'

const fetchVenueList = async () => {
  loading.value = true
  try {
    const params: Record<string, number> = {
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
      detailVisible.value = false
      fetchVenueList()
    } else {
      ElMessage.error(res.message || '操作失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '操作失败')
    }
  }
}

const handleDialogReview = (status: number) => {
  if (!currentVenue.value) return
  handleReview(currentVenue.value, status)
}

const showDetail = (venue: Venue) => {
  currentVenue.value = venue
  detailVisible.value = true
}

onMounted(() => {
  fetchVenueList()
})
</script>

<style scoped lang="scss">
.venue-review-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
