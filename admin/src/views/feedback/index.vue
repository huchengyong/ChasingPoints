<template>
  <div class="feedback-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>投诉举报管理</span>
          <div class="header-filters">
            <el-select v-model="filterStatus" placeholder="处理状态" style="width: 140px" @change="handleFilterChange">
              <el-option label="全部" :value="-1" />
              <el-option label="待处理" :value="1" />
              <el-option label="处理中" :value="2" />
              <el-option label="已办结" :value="3" />
              <el-option label="已关闭" :value="4" />
            </el-select>
            <el-select v-model="filterCategory" placeholder="提交类型" clearable style="width: 140px" @change="handleFilterChange">
              <el-option label="意见反馈" value="feedback" />
              <el-option label="投诉" value="complaint" />
              <el-option label="举报" value="report" />
            </el-select>
            <el-select v-model="filterSource" placeholder="来源" clearable style="width: 120px" @change="handleFilterChange">
              <el-option label="App" value="app" />
              <el-option label="官网" value="website" />
            </el-select>
          </div>
        </div>
      </template>

      <el-table :data="ticketList" stripe border v-loading="loading">
        <el-table-column type="index" width="60" label="序号" :index="(index: number) => (page - 1) * pageSize + index + 1" />
        <el-table-column prop="source_text" label="来源" width="90" />
        <el-table-column prop="category_text" label="类型" width="110" />
        <el-table-column label="内容摘要" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.content || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="contact" label="联系方式" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.contact || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status_text" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusMap[row.status]?.type || 'info'">
              {{ row.status_text || statusMap[row.status]?.label || '未知' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="提交时间" width="170" />
        <el-table-column prop="processed_at" label="处理时间" width="170">
          <template #default="{ row }">
            {{ row.processed_at || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">详情</el-button>
            <el-button link type="success" @click="openProcessDialog(row)">处理</el-button>
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

    <el-dialog v-model="detailVisible" title="工单详情" width="720px">
      <el-descriptions v-if="currentTicket" :column="2" border>
        <el-descriptions-item label="来源">{{ currentTicket.source_text }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ currentTicket.category_text }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusMap[currentTicket.status]?.type || 'info'">
            {{ currentTicket.status_text || statusMap[currentTicket.status]?.label || '未知' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="用户ID">{{ currentTicket.user_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="联系方式" :span="2">{{ currentTicket.contact || '-' }}</el-descriptions-item>
        <el-descriptions-item label="提交内容" :span="2">{{ currentTicket.content || '-' }}</el-descriptions-item>
        <el-descriptions-item label="处理人员">{{ currentTicket.handler_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="处理时间">{{ currentTicket.processed_at || '-' }}</el-descriptions-item>
        <el-descriptions-item label="处理结果" :span="2">{{ currentTicket.process_result || '-' }}</el-descriptions-item>
        <el-descriptions-item label="提交时间">{{ currentTicket.created_at }}</el-descriptions-item>
        <el-descriptions-item label="更新时间">{{ currentTicket.updated_at }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button v-if="currentTicket" type="primary" @click="openProcessDialog(currentTicket)">处理</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="processDialogVisible" title="处理工单" width="560px">
      <el-form label-width="96px">
        <el-form-item label="处理状态" required>
          <el-select v-model="processForm.status" style="width: 100%">
            <el-option label="处理中" :value="2" />
            <el-option label="已办结" :value="3" />
            <el-option label="已关闭" :value="4" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理结果" :required="processForm.status !== 2">
          <el-input
            v-model="processForm.processResult"
            type="textarea"
            :rows="5"
            maxlength="500"
            show-word-limit
            placeholder="记录处理动作、结论或需补充的信息"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="processDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="processing" @click="submitProcess">
          保存处理结果
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getAdminFeedbackTicketList,
  processAdminFeedbackTicket,
  type AdminFeedbackTicket
} from '@/api/feedback'

const loading = ref(false)
const processing = ref(false)
const ticketList = ref<AdminFeedbackTicket[]>([])
const currentTicket = ref<AdminFeedbackTicket | null>(null)
const detailVisible = ref(false)
const processDialogVisible = ref(false)

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterStatus = ref(-1)
const filterCategory = ref('')
const filterSource = ref('')

const processForm = reactive({
  ticketId: 0,
  status: 2,
  processResult: ''
})

const statusMap: Record<number, { label: string; type: 'success' | 'warning' | 'danger' | 'info' | 'primary' }> = {
  1: { label: '待处理', type: 'warning' },
  2: { label: '处理中', type: 'primary' },
  3: { label: '已办结', type: 'success' },
  4: { label: '已关闭', type: 'info' }
}

const fetchTicketList = async () => {
  loading.value = true
  try {
    const params: Record<string, number | string> = {
      page: page.value,
      page_size: pageSize.value
    }
    if (filterStatus.value !== -1) {
      params.status = filterStatus.value
    }
    if (filterCategory.value) {
      params.category = filterCategory.value
    }
    if (filterSource.value) {
      params.source = filterSource.value
    }

    const res = await getAdminFeedbackTicketList(params)
    if (res.success) {
      ticketList.value = res.list || []
      total.value = res.total || 0
    } else {
      ElMessage.error(res.message || '获取工单列表失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取工单列表失败')
  } finally {
    loading.value = false
  }
}

const handleFilterChange = () => {
  page.value = 1
  fetchTicketList()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  fetchTicketList()
}

const handlePageChange = (current: number) => {
  page.value = current
  fetchTicketList()
}

const showDetail = (ticket: AdminFeedbackTicket) => {
  currentTicket.value = ticket
  detailVisible.value = true
}

const openProcessDialog = (ticket: AdminFeedbackTicket) => {
  currentTicket.value = ticket
  processForm.ticketId = ticket.id
  processForm.status = ticket.status === 1 ? 2 : ticket.status
  processForm.processResult = ticket.process_result || ''
  processDialogVisible.value = true
}

const submitProcess = async () => {
  if (!processForm.ticketId) return
  const result = processForm.processResult.trim()
  if (processForm.status !== 2 && !result) {
    ElMessage.warning('请填写处理结果')
    return
  }

  processing.value = true
  try {
    const res = await processAdminFeedbackTicket({
      ticket_id: processForm.ticketId,
      status: processForm.status,
      process_result: result || undefined
    })
    if (res.success) {
      ElMessage.success(res.message || '处理成功')
      processDialogVisible.value = false
      detailVisible.value = false
      await fetchTicketList()
    } else {
      ElMessage.error(res.message || '处理失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '处理失败')
  } finally {
    processing.value = false
  }
}

onMounted(() => {
  fetchTicketList()
})
</script>

<style scoped lang="scss">
.feedback-page {
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
  gap: 12px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
