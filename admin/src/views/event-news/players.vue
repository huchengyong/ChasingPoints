<template>
  <div class="event-news-players-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>赛讯球员中文名</span>
          <span class="header-hint">WST 只提供英文名；这里填写的中文名显示在小程序赛程卡片上（英文名小字保留），留空则只显示英文名。</span>
        </div>
      </template>

      <el-form :inline="true" class="filters" @submit.prevent="search">
        <el-form-item label="关键词">
          <el-input v-model="keyword" placeholder="英文名或中文名" clearable @clear="search" @keyup.enter="search" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="missingOnly" @change="search">只看未填写中文名</el-checkbox>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="list" stripe border v-loading="loading">
        <el-table-column label="球员" min-width="220">
          <template #default="{ row }">
            <div class="player-cell">
              <el-avatar :size="32" :src="row.avatar || undefined">{{ row.display_name.slice(0, 1) }}</el-avatar>
              <span>{{ row.display_name || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="国家/地区" width="110" align="center">
          <template #default="{ row }">{{ row.country_code ? row.country_code.toUpperCase() : '-' }}</template>
        </el-table-column>
        <el-table-column label="中文名" min-width="260">
          <template #default="{ row }">
            <el-input
              v-model="drafts[row.id]"
              :maxlength="PLAYER_NAME_ZH_MAX_LENGTH"
              placeholder="未填写"
              clearable
              @keyup.enter="save(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              :disabled="!isPlayerNameZhChanged(row.name_zh, drafts[row.id])"
              :loading="savingId === row.id"
              @click="save(row)"
            >
              保存
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="最近同步" width="170" align="center" />
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getEventNewsPlayers, updateEventNewsPlayer, type EventNewsPlayerItem } from '@/api/event-news'
import { PLAYER_NAME_ZH_MAX_LENGTH, isPlayerNameZhChanged, normalizePlayerNameZh } from '@/utils/player-name-zh'

const keyword = ref('')
const missingOnly = ref(false)
const list = ref<EventNewsPlayerItem[]>([])
const drafts = reactive<Record<number, string>>({})
const loading = ref(false)
const savingId = ref(0)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 请求拦截器已经会弹出业务失败与网络错误的提示，这里只负责还原加载状态。
const fetchList = async () => {
  loading.value = true
  try {
    const res = await getEventNewsPlayers({
      keyword: keyword.value.trim() || undefined,
      missing_zh: missingOnly.value ? 1 : undefined,
      page: page.value,
      page_size: pageSize.value
    })
    if (res.success) {
      list.value = res.list
      total.value = res.total
      for (const key of Object.keys(drafts)) delete drafts[Number(key)]
      for (const item of res.list) drafts[item.id] = item.name_zh
    }
  } finally {
    loading.value = false
  }
}

const search = () => {
  page.value = 1
  fetchList()
}

const handleSizeChange = (value: number) => {
  pageSize.value = value
  page.value = 1
  fetchList()
}

const handlePageChange = (value: number) => {
  page.value = value
  fetchList()
}

// 保存后只更新当前行，不重新拉列表，避免"只看未填写"时刚填好的行立刻消失、打断连续录入
const save = async (row: EventNewsPlayerItem) => {
  if (savingId.value || !isPlayerNameZhChanged(row.name_zh, drafts[row.id])) return
  const nameZh = normalizePlayerNameZh(drafts[row.id])
  savingId.value = row.id
  try {
    const res = await updateEventNewsPlayer({ player_id: row.id, name_zh: nameZh })
    if (res.success) {
      row.name_zh = nameZh
      drafts[row.id] = nameZh
      ElMessage.success(nameZh ? `已保存：${nameZh}` : '已清除中文名')
    }
  } finally {
    savingId.value = 0
  }
}

onMounted(fetchList)
</script>

<style scoped lang="scss">
.event-news-players-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}

.header-hint {
  color: #909399;
  font-size: 12px;
}

.filters {
  margin-bottom: 8px;
}

.player-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
