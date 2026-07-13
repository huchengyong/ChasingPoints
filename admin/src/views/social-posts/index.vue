<template>
  <div class="social-posts-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>动态审核管理</span>
          <div class="header-filters">
            <el-select v-model="filterStatus" placeholder="审核状态" clearable @change="handleFilterChange">
              <el-option label="全部" :value="-1" />
              <el-option label="待审核" :value="2" />
              <el-option label="已发布" :value="1" />
              <el-option label="已拒绝" :value="3" />
            </el-select>
          </div>
        </div>
      </template>

      <el-table :data="postList" stripe border v-loading="loading">
        <el-table-column type="index" width="60" label="序号" />
        <el-table-column prop="nickname" label="发布人" width="140" />
        <el-table-column label="类型" width="110">
          <template #default="{ row }">
            {{ formatPostType(row.post_type) }}
          </template>
        </el-table-column>
        <el-table-column label="内容摘要" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.content || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="图片" width="90" align="center">
          <template #default="{ row }">
            {{ row.images?.length || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusMap[row.status]?.type || 'info'">
              {{ row.status_text || statusMap[row.status]?.label || '未知' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="发布时间" width="170" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 2" link type="success" @click="handleReview(row, 1)">
              通过
            </el-button>
            <el-button v-if="row.status === 2" link type="danger" @click="openRejectDialog(row)">
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

    <el-dialog v-model="detailVisible" title="动态详情" width="720px">
      <el-descriptions v-if="currentPost" :column="2" border>
        <el-descriptions-item label="发布人">{{ currentPost.nickname || '-' }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ formatPostType(currentPost.post_type) }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusMap[currentPost.status]?.type || 'info'">
            {{ currentPost.status_text || statusMap[currentPost.status]?.label || '未知' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="发布时间">{{ currentPost.created_at }}</el-descriptions-item>
        <el-descriptions-item label="拒绝原因" :span="2">{{ currentPost.reject_reason || '-' }}</el-descriptions-item>
        <el-descriptions-item label="内容" :span="2">{{ currentPost.content || '-' }}</el-descriptions-item>
        <el-descriptions-item label="图片" :span="2">
          <div v-if="currentPost.images?.length" class="image-grid">
            <el-image
              v-for="(img, idx) in currentPost.images"
              :key="`${currentPost.id}-${idx}`"
              :src="img"
              fit="cover"
              class="post-image"
              :preview-src-list="currentPost.images"
              preview-teleported
            />
          </div>
          <span v-else>-</span>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button v-if="currentPost?.status === 2" type="success" @click="handleReview(currentPost, 1)">
          审核通过
        </el-button>
        <el-button v-if="currentPost?.status === 2" type="danger" @click="openRejectDialog(currentPost)">
          审核拒绝
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rejectDialogVisible" title="审核拒绝" width="520px">
      <el-form label-width="88px">
        <el-form-item label="拒绝原因" required>
          <el-input
            v-model="rejectReason"
            type="textarea"
            :rows="4"
            maxlength="200"
            show-word-limit
            placeholder="请填写拒绝原因，作者会在“我的动态”里看到"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="reviewing" @click="submitReject">
          确认拒绝
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getAdminSocialPostList,
  reviewAdminSocialPost,
  type AdminSocialPost
} from '@/api/social'

const loading = ref(false)
const reviewing = ref(false)
const postList = ref<AdminSocialPost[]>([])
const currentPost = ref<AdminSocialPost | null>(null)
const detailVisible = ref(false)
const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejectTarget = ref<AdminSocialPost | null>(null)

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterStatus = ref<number>(-1)

const statusMap: Record<number, { label: string; type: 'success' | 'warning' | 'danger' | 'info' }> = {
  1: { label: '已发布', type: 'success' },
  2: { label: '待审核', type: 'warning' },
  3: { label: '已拒绝', type: 'danger' }
}

const formatPostType = (postType: number) => {
  if (postType === 1) return '战绩分享'
  if (postType === 2) return '打卡动态'
  return '日常动态'
}

const fetchPostList = async () => {
  loading.value = true
  try {
    const params: Record<string, number> = {
      page: page.value,
      page_size: pageSize.value
    }
    if (filterStatus.value !== -1) {
      params.status = filterStatus.value
    }
    const res = await getAdminSocialPostList(params)
    if (res.success) {
      postList.value = res.list || []
      total.value = res.total || 0
    }
  } finally {
    loading.value = false
  }
}

const handleFilterChange = () => {
  page.value = 1
  fetchPostList()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  fetchPostList()
}

const handlePageChange = (current: number) => {
  page.value = current
  fetchPostList()
}

const showDetail = (post: AdminSocialPost) => {
  currentPost.value = post
  detailVisible.value = true
}

const handleReview = async (post: AdminSocialPost, status: number, reason = '') => {
  reviewing.value = true
  try {
    const res = await reviewAdminSocialPost({
      post_id: post.id,
      status,
      reject_reason: reason || undefined
    })
    if (res.success) {
      ElMessage.success(res.message || '审核成功')
      detailVisible.value = false
      rejectDialogVisible.value = false
      rejectReason.value = ''
      rejectTarget.value = null
      await fetchPostList()
    }
  } finally {
    reviewing.value = false
  }
}

const openRejectDialog = (post: AdminSocialPost) => {
  rejectTarget.value = post
  rejectReason.value = post.reject_reason || ''
  rejectDialogVisible.value = true
}

const submitReject = async () => {
  if (!rejectTarget.value) return
  const trimmedReason = rejectReason.value.trim()
  if (!trimmedReason) {
    ElMessage.warning('请填写拒绝原因')
    return
  }
  await handleReview(rejectTarget.value, 3, trimmedReason)
}

onMounted(() => {
  fetchPostList()
})
</script>

<style scoped lang="scss">
.social-posts-page {
  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .header-filters {
    display: flex;
    gap: 12px;
  }

  .pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: 20px;
  }

  .image-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }

  .post-image {
    width: 104px;
    height: 104px;
    border-radius: 10px;
    overflow: hidden;
    background: #f3f4f6;
  }
}
</style>
