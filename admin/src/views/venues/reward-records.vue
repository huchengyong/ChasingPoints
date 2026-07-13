<template>
  <div class="venue-reward-records-page">
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getVenueRewardRecordList, type VenueRewardRecord } from '@/api/venue'

const recordLoading = ref(false)
const rewardRecordList = ref<VenueRewardRecord[]>([])
const recordPage = ref(1)
const recordPageSize = ref(10)
const recordTotal = ref(0)

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

const handleRecordSizeChange = (val: number) => {
  recordPageSize.value = val
  fetchRewardRecordList()
}

const handleRecordPageChange = (val: number) => {
  recordPage.value = val
  fetchRewardRecordList()
}

onMounted(() => {
  fetchRewardRecordList()
})
</script>

<style scoped lang="scss">
.venue-reward-records-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
