<template>
  <div class="users-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>用户管理</span>
          <div class="header-filters">
            <el-select v-model="filterStatus" placeholder="用户状态" clearable @change="handleFilterChange">
              <el-option label="全部" :value="-1" />
              <el-option label="正常" :value="1" />
              <el-option label="禁用" :value="0" />
            </el-select>
          </div>
        </div>
      </template>

      <el-table :data="userList" stripe border v-loading="loading">
        <el-table-column type="index" width="60" label="序号" :index="(index: number) => (page - 1) * pageSize + index + 1" />
        <el-table-column prop="id" label="用户ID" width="100" />
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="phone" label="手机号" min-width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="注册时间" width="160" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button 
              v-if="row.status === 1" 
              link 
              type="danger" 
              @click="handleUpdateStatus(row, 0)"
            >
              禁用
            </el-button>
            <el-button 
              v-else 
              link 
              type="success" 
              @click="handleUpdateStatus(row, 1)"
            >
              启用
            </el-button>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserList, updateUserStatus, type User } from '@/api/user'

const loading = ref(false)
const userList = ref<User[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterStatus = ref<number>(-1)

const fetchUserList = async () => {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value
    }
    if (filterStatus.value !== -1) {
      params.status = filterStatus.value
    }
    const res = await getUserList(params)
    if (res.success) {
      userList.value = res.list
      total.value = res.total
    } else {
      ElMessage.error(res.message || '获取用户列表失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取用户列表失败')
  } finally {
    loading.value = false
  }
}

const handleFilterChange = () => {
  page.value = 1
  fetchUserList()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  fetchUserList()
}

const handlePageChange = (val: number) => {
  page.value = val
  fetchUserList()
}

const handleUpdateStatus = async (user: User, status: number) => {
  const action = status === 1 ? '启用' : '禁用'
  try {
    await ElMessageBox.confirm(
      `确定要${action}用户 "${user.nickname}" 吗？`,
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: status === 1 ? 'success' : 'warning'
      }
    )
    
    const res = await updateUserStatus({
      user_id: user.id,
      status
    })
    
    if (res.success) {
      ElMessage.success(res.message || `用户${action}成功`)
      fetchUserList()
    } else {
      ElMessage.error(res.message || '操作失败')
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '操作失败')
    }
  }
}

onMounted(() => {
  fetchUserList()
})
</script>

<style scoped lang="scss">
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
