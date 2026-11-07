<template>
  <div class="gear-brands-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>装备品牌</span>
          <el-button type="primary" @click="openCreate">新增品牌</el-button>
        </div>
      </template>

      <el-form :inline="true" class="filters" @submit.prevent="search">
        <el-form-item label="关键词">
          <el-input v-model="keyword" placeholder="品牌名或别名" clearable @clear="search" @keyup.enter="search" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="list" stripe border v-loading="loading">
        <el-table-column prop="name" label="品牌名称" min-width="160" />
        <el-table-column label="搜索别名" min-width="200">
          <template #default="{ row }">{{ row.aliases || '-' }}</template>
        </el-table-column>
        <el-table-column prop="product_count" label="型号数" width="100" align="center" />
        <el-table-column prop="sort" label="排序" width="90" align="center" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑品牌' : '新增品牌'" width="480px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="品牌名称" required>
          <el-input v-model="form.name" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="搜索别名">
          <el-input v-model="form.aliases" placeholder="多个别名用逗号分隔，例如：阿尔法,alp" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="-9999" :max="9999" />
          <span class="hint">数值越小越靠前</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-alert
          v-if="form.status === 0"
          type="warning"
          :closable="false"
          show-icon
          title="停用后用户不能再选择这个品牌及其型号；已经登记的装备和历史对局不受影响。"
        />
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getGearBrandList, saveGearBrand, type GearBrand } from '@/api/gear'
import { buildBrandPayload, defaultBrandForm, validateBrandForm, type BrandForm } from '@/utils/gear-form'

const keyword = ref('')
const list = ref<GearBrand[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<BrandForm>(defaultBrandForm())

// 请求拦截器已经会弹出业务失败与网络错误的提示，这里只负责还原加载状态。
const fetchList = async () => {
  loading.value = true
  try {
    const res = await getGearBrandList({ keyword: keyword.value.trim() || undefined, page: page.value, page_size: pageSize.value })
    if (res.success) {
      list.value = res.list
      total.value = res.total
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

const openCreate = () => {
  Object.assign(form, defaultBrandForm())
  dialogVisible.value = true
}

const openEdit = (row: GearBrand) => {
  Object.assign(form, { id: row.id, name: row.name, aliases: row.aliases, sort: row.sort, status: row.status })
  dialogVisible.value = true
}

const save = async () => {
  const message = validateBrandForm(form)
  if (message) {
    ElMessage.warning(message)
    return
  }
  saving.value = true
  try {
    const res = await saveGearBrand(buildBrandPayload(form))
    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      dialogVisible.value = false
      fetchList()
    }
  } finally {
    saving.value = false
  }
}

onMounted(fetchList)
</script>

<style scoped lang="scss">
.gear-brands-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filters {
  margin-bottom: 8px;
}

.hint {
  margin-left: 12px;
  color: #909399;
  font-size: 12px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
