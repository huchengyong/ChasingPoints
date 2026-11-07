<template>
  <div class="gear-pending-page">
    <el-card>
      <template #header>
        <div class="card-header"><span>待收录型号</span><span class="hint">按登记人数倒序；关联后这些用户的装备与历史对局会显示为正式型号</span></div>
      </template>
      <el-table :data="list" stripe border v-loading="loading">
        <el-table-column label="类别" width="120"><template #default="{ row }">{{ categoryLabel(row.category) }}</template></el-table-column>
        <el-table-column prop="brand_name" label="品牌（用户填写）" min-width="150" />
        <el-table-column prop="model_name" label="型号（用户填写）" min-width="180" />
        <el-table-column label="杆型 / 身份" min-width="130">
          <template #default="{ row }">{{ row.category === 'cue' ? formatCueStyle(row.cue_style) : formatRoles(row.roles) }}</template>
        </el-table-column>
        <el-table-column prop="count" label="登记人数" width="100" align="center" />
        <el-table-column prop="last_at" label="最近登记" width="170" />
        <el-table-column label="操作" width="130" align="center">
          <template #default="{ row }"><el-button link type="primary" @click="openLink(row)">关联到型号</el-button></template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next" @size-change="onSize" @current-change="onPage" />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" title="关联到目录型号" width="520px">
      <template v-if="current">
        <p class="summary">{{ current.brand_name }} {{ current.model_name }}（{{ categoryLabel(current.category) }}，{{ current.count }} 人登记）</p>
        <el-select v-model="productId" filterable remote clearable style="width: 100%" placeholder="搜索同类别的启用型号"
          :remote-method="searchTargets" :loading="searching">
          <el-option v-for="p in options" :key="p.id" :label="productTitle(p)" :value="p.id" />
        </el-select>
      </template>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!productId" :loading="linking" @click="link">关联</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getGearPendingList, getGearProductList, linkGearPending, type GearPendingGroup, type GearProduct } from '@/api/gear'
import { describeLinkConfirm, formatGearCueStyle as formatCueStyle, formatGearRoles as formatRoles, gearCategoryLabel as categoryLabel, productTitle } from '@/utils/gear-form'

const list = ref<GearPendingGroup[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const dialogVisible = ref(false)
const current = ref<GearPendingGroup | null>(null)
const productId = ref<number | undefined>()
const options = ref<GearProduct[]>([])
const searching = ref(false)
const linking = ref(false)

// 请求拦截器已经会弹出失败提示，这里只还原加载状态。
const fetchList = async () => {
  loading.value = true
  try {
    const res = await getGearPendingList({ page: page.value, page_size: pageSize.value })
    if (res.success) { list.value = res.list; total.value = res.total }
  } finally { loading.value = false }
}
const onSize = (v: number) => { pageSize.value = v; page.value = 1; fetchList() }
const onPage = (v: number) => { page.value = v; fetchList() }

const searchTargets = async (keyword: string) => {
  if (!current.value) return
  searching.value = true
  try {
    const res = await getGearProductList({ category: current.value.category, status: 1, keyword: keyword.trim() || undefined, page: 1, page_size: 20 })
    if (res.success) options.value = res.list
  } finally { searching.value = false }
}

const openLink = (group: GearPendingGroup) => {
  current.value = group
  productId.value = undefined
  options.value = []
  dialogVisible.value = true
  searchTargets(group.model_name)
}

const link = async () => {
  const group = current.value
  const target = options.value.find((p) => p.id === productId.value)
  if (!group || !target) return
  try {
    await ElMessageBox.confirm(describeLinkConfirm(group, productTitle(target)), '确认关联', { type: 'warning', confirmButtonText: '关联', cancelButtonText: '取消' })
  } catch {
    return
  }
  linking.value = true
  try {
    const res = await linkGearPending({ custom_key: group.custom_key, product_id: target.id })
    if (res.success) {
      ElMessage.success(res.message)
      // 一次最多处理一批；还有剩余时保留弹窗，可以再次关联
      if (res.remaining > 0) group.count = res.remaining
      else dialogVisible.value = false
      fetchList()
    }
  } finally { linking.value = false }
}

onMounted(fetchList)
</script>

<style scoped lang="scss">
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { color: #909399; font-size: 12px; }
.summary { margin: 0 0 12px; font-weight: 600; }
.pagination { margin-top: 20px; display: flex; justify-content: flex-end; }
</style>
