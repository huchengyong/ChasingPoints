<template>
  <div class="gear-products-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>装备型号</span>
          <el-button type="primary" @click="openCreate">新增型号</el-button>
        </div>
      </template>

      <el-form :inline="true" class="filters" @submit.prevent="search">
        <el-form-item label="类别">
          <el-select v-model="filters.category" placeholder="全部" clearable style="width: 140px" @change="search">
            <el-option v-for="option in categoryOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="品牌">
          <el-select v-model="filters.brand_id" placeholder="全部" clearable filterable style="width: 180px" @change="search">
            <el-option v-for="brand in brands" :key="brand.id" :label="brand.name" :value="brand.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" style="width: 110px" @change="search">
            <el-option label="全部" :value="-1" />
            <el-option label="启用" :value="1" />
            <el-option label="停用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="filters.keyword" placeholder="型号、品牌名或别名" clearable @clear="search" @keyup.enter="search" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="list" stripe border v-loading="loading">
        <el-table-column label="图片" width="90" align="center">
          <template #default="{ row }">
            <el-image v-if="imageUrl(row.image_path)" :src="imageUrl(row.image_path)" fit="contain" style="width: 56px; height: 56px" />
            <span v-else class="muted">默认图</span>
          </template>
        </el-table-column>
        <el-table-column label="类别" width="120">
          <template #default="{ row }">{{ categoryLabel(row.category) }}</template>
        </el-table-column>
        <el-table-column label="品牌" min-width="130">
          <template #default="{ row }">{{ row.house ? '球房公用' : row.brand_name }}</template>
        </el-table-column>
        <el-table-column prop="name" label="型号" min-width="180" />
        <el-table-column label="杆型 / 身份" min-width="130">
          <template #default="{ row }">{{ row.category === 'cue' ? formatCueStyle(row.cue_style) : formatRoles(row.roles) || '-' }}</template>
        </el-table-column>
        <el-table-column label="官方规格" min-width="200">
          <template #default="{ row }">{{ specText(row) }}</template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" align="center" />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="center">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑型号' : '新增型号'" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="类别" required>
          <el-select v-model="form.category" :disabled="form.id > 0" style="width: 100%" @change="handleCategoryChange">
            <el-option v-for="option in categoryOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
          <div v-if="form.id > 0" class="hint">已有用户装备引用，类别不能修改</div>
        </el-form-item>
        <el-form-item label="品牌" required>
          <el-select v-model="form.brand_id" filterable style="width: 100%" placeholder="选择品牌">
            <el-option v-if="allowsHouse(form.category)" label="无品牌（球房公用条目）" :value="0" />
            <el-option v-for="brand in brands" :key="brand.id" :label="brand.name" :value="brand.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="型号名称" required>
          <el-input v-model="form.name" maxlength="128" show-word-limit />
        </el-form-item>
        <el-form-item v-if="form.category === 'cue'" label="杆型" :required="form.brand_id !== 0">
          <el-radio-group v-model="form.cue_style">
            <el-radio v-for="option in cueStyleOptions" :key="option.value" :value="option.value">{{ option.label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.category === 'break_jump'" label="身份" required>
          <el-checkbox-group v-model="form.roles">
            <el-checkbox v-for="option in roleOptions" :key="option.value" :value="option.value">{{ option.label }}</el-checkbox>
          </el-checkbox-group>
          <div class="hint">开跳两用的杆同时勾选冲杆和跳杆</div>
        </el-form-item>
        <el-form-item v-if="form.category === 'cue' || form.category === 'break_jump'" label="官方长度">
          <el-input-number v-model="form.length_mm" :min="0" :max="3000" />
          <span class="unit">毫米，0 表示品牌未提供</span>
        </el-form-item>
        <el-form-item v-if="form.category === 'cue' || form.category === 'break_jump'" label="官方重量">
          <el-input-number v-model="form.weight_g" :min="0" :max="2000" />
          <span class="unit">克，0 表示品牌未提供</span>
        </el-form-item>
        <el-form-item v-if="form.category === 'cue' || form.category === 'break_jump'" label="皮头直径">
          <el-input-number v-model="form.tip_diameter_mm" :min="0" :max="30" :precision="2" :step="0.1" />
          <span class="unit">毫米，0 表示品牌未提供</span>
        </el-form-item>
        <el-form-item label="目录图路径">
          <el-input v-model="form.image_path" placeholder="/static/images/gear/xxx.png" />
          <div class="hint">图片先上传到 CDN，再在这里填写路径（仅限 /static/images/gear/ 下的 png 或 webp）；留空时用户端显示该类别的默认图。</div>
          <el-image v-if="previewUrl" :src="previewUrl" fit="contain" class="preview" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="-9999" :max="9999" />
          <span class="unit">数值越小越靠前</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getGearBrandList,
  getGearProductList,
  saveGearProduct,
  type GearBrand,
  type GearProduct
} from '@/api/gear'
import {
  GEAR_CATEGORY_OPTIONS,
  GEAR_CUE_STYLE_OPTIONS,
  GEAR_ROLE_OPTIONS,
  allowsHouseEntry,
  buildProductPayload,
  defaultProductForm,
  formatGearCueStyle,
  formatGearRoles,
  gearCategoryLabel,
  gearImageUrl,
  productToForm,
  validateProductForm,
  type ProductForm
} from '@/utils/gear-form'

const categoryOptions = GEAR_CATEGORY_OPTIONS
const cueStyleOptions = GEAR_CUE_STYLE_OPTIONS
const roleOptions = GEAR_ROLE_OPTIONS
const categoryLabel = gearCategoryLabel
const formatCueStyle = formatGearCueStyle
const formatRoles = formatGearRoles
const imageUrl = gearImageUrl
const allowsHouse = allowsHouseEntry

const brands = ref<GearBrand[]>([])
const filters = reactive({ category: '', brand_id: undefined as number | undefined, status: -1, keyword: '' })
const list = ref<GearProduct[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<ProductForm>(defaultProductForm())
const previewUrl = computed(() => gearImageUrl(form.image_path.trim()))

const specText = (row: GearProduct): string => {
  const parts: string[] = []
  if (row.length_mm > 0) parts.push(`${row.length_mm} mm`)
  if (row.weight_g > 0) parts.push(`${row.weight_g} g`)
  if (row.tip_diameter_mm > 0) parts.push(`皮头 ${row.tip_diameter_mm} mm`)
  return parts.join(' · ') || '-'
}

// 品牌下拉最多取 100 个；品牌更多时先用关键词缩小范围再编辑。请求拦截器已经会弹出失败提示。
const fetchBrands = async () => {
  const res = await getGearBrandList({ page: 1, page_size: 100 })
  if (res.success) brands.value = res.list
}

const fetchList = async () => {
  loading.value = true
  try {
    const res = await getGearProductList({
      category: filters.category || undefined,
      brand_id: filters.brand_id || undefined,
      keyword: filters.keyword.trim() || undefined,
      status: filters.status,
      page: page.value,
      page_size: pageSize.value
    })
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
  Object.assign(form, defaultProductForm(filters.category || 'cue'))
  dialogVisible.value = true
}

const openEdit = (row: GearProduct) => {
  Object.assign(form, productToForm(row))
  dialogVisible.value = true
}

// 换类别时清掉不属于新类别的字段，并避免保留不允许的「无品牌」。
const handleCategoryChange = () => {
  form.cue_style = ''
  form.roles = []
  if (!allowsHouseEntry(form.category) && form.brand_id === 0) form.brand_id = null
}

const save = async () => {
  const message = validateProductForm(form)
  if (message) {
    ElMessage.warning(message)
    return
  }
  saving.value = true
  try {
    const res = await saveGearProduct(buildProductPayload(form))
    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      dialogVisible.value = false
      fetchList()
    }
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchBrands()
  fetchList()
})
</script>

<style scoped lang="scss">
.gear-products-page {
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

.muted {
  color: #c0c4cc;
  font-size: 12px;
}

.hint,
.unit {
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}

.unit {
  margin-left: 12px;
}

.preview {
  margin-top: 8px;
  width: 120px;
  height: 120px;
  border: 1px solid #ebeef5;
  border-radius: 4px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
