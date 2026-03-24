<template>
  <div class="event-news-page">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <div class="title-block">
            <span class="title">赛事情报管理</span>
            <span class="subtitle">维护斯诺克、中式八球、中式九球的赛程与赛况信息</span>
          </div>
          <div class="header-actions">
            <el-select v-model="filterGameType" placeholder="球种筛选" style="width: 160px" @change="handleFilter">
              <el-option label="全部" :value="-1" />
              <el-option label="斯诺克" :value="1" />
              <el-option label="中式八球" :value="3" />
              <el-option label="中式九球" :value="2" />
            </el-select>
            <el-select v-model="filterStatus" placeholder="状态筛选" style="width: 140px" @change="handleFilter">
              <el-option label="全部" :value="-1" />
              <el-option label="即将开始" :value="0" />
              <el-option label="进行中" :value="1" />
              <el-option label="已结束" :value="2" />
              <el-option label="已取消" :value="3" />
            </el-select>
            <el-select v-model="filterPublished" placeholder="发布状态" style="width: 140px" @change="handleFilter">
              <el-option label="全部" :value="-1" />
              <el-option label="未发布" :value="0" />
              <el-option label="已发布" :value="1" />
            </el-select>
            <el-button type="primary" @click="openCreateDialog">新建草稿</el-button>
          </div>
        </div>
      </template>

      <el-table :data="eventNewsList" stripe border>
        <el-table-column type="index" width="60" label="序号" :index="(index: number) => (page - 1) * pageSize + index + 1" />
        <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
        <el-table-column prop="game_type" label="球种" width="120">
          <template #default="{ row }">
            {{ getGameTypeLabel(row.game_type) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="featured" label="Featured" width="110">
          <template #default="{ row }">
            <el-tag :type="row.featured ? 'warning' : 'info'">
              {{ row.featured ? '焦点' : '普通' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="published" label="Published" width="120">
          <template #default="{ row }">
            <el-tag :type="row.published ? 'success' : 'info'">
              {{ row.published ? '已发布' : '未发布' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="开始时间" width="170" />
        <el-table-column prop="updated_at" label="更新时间" width="170" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button
              link
              :type="row.published ? 'warning' : 'success'"
              @click="handlePublishToggle(row)"
            >
              {{ row.published ? '下线' : '发布' }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
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

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="960px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="formModel" :rules="rules" label-width="110px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="标题" prop="title">
              <el-input v-model="formModel.title" placeholder="请输入赛事标题" maxlength="128" show-word-limit />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="球种" prop="game_type">
              <el-select v-model="formModel.game_type" placeholder="请选择球种" style="width: 100%">
                <el-option v-for="item in gameTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="来源类型">
              <el-select v-model="formModel.source_type" placeholder="请选择来源类型" style="width: 100%">
                <el-option v-for="item in sourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="来源名称">
              <el-input v-model="formModel.source_name" placeholder="如：WST / 独牙传奇" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="来源链接">
              <el-input v-model="formModel.source_url" placeholder="https://..." />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="封面图">
              <el-input v-model="formModel.cover_image" placeholder="封面图片地址" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="摘要">
              <el-input v-model="formModel.summary" type="textarea" :rows="2" placeholder="用于列表和首页焦点展示" maxlength="512" show-word-limit />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="正文">
              <el-input v-model="formModel.content" type="textarea" :rows="6" placeholder="赛事情报正文" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="国家">
              <el-input v-model="formModel.country" placeholder="国家 / 地区" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="城市">
              <el-input v-model="formModel.city" placeholder="举办城市" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="场馆">
              <el-input v-model="formModel.venue" placeholder="比赛场馆" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="开始时间">
              <el-date-picker
                v-model="formModel.start_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="请选择开始时间"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结束时间">
              <el-date-picker
                v-model="formModel.end_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="请选择结束时间"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="状态" prop="status">
              <el-select v-model="formModel.status" placeholder="请选择状态" style="width: 100%">
                <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="阶段说明">
              <el-input v-model="formModel.stage_text" placeholder="如：资格赛 / 决赛日" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="赛果摘要">
              <el-input v-model="formModel.result_text" placeholder="如：赵心童晋级 8 强" maxlength="255" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="焦点推荐">
              <el-switch v-model="formModel.featured" active-text="是" inactive-text="否" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="排序时间">
              <el-date-picker
                v-model="formModel.sort_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="不填则默认使用开始时间"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  createEventNews,
  deleteEventNews,
  getEventNewsList,
  publishEventNews,
  updateEventNews,
  type EventNewsFormPayload,
  type EventNewsItem
} from '@/api/event-news'

interface EventNewsFormModel extends EventNewsFormPayload {
  event_news_id?: number
}

const loading = ref(false)
const saving = ref(false)
const eventNewsList = ref<EventNewsItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterGameType = ref(-1)
const filterStatus = ref(-1)
const filterPublished = ref(-1)

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

function formatDateTime(value: Date) {
  const year = value.getFullYear()
  const month = `${value.getMonth() + 1}`.padStart(2, '0')
  const day = `${value.getDate()}`.padStart(2, '0')
  const hour = `${value.getHours()}`.padStart(2, '0')
  const minute = `${value.getMinutes()}`.padStart(2, '0')
  const second = `${value.getSeconds()}`.padStart(2, '0')
  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}

const getNowString = () => formatDateTime(new Date())

const createEmptyForm = (): EventNewsFormModel => ({
  title: '',
  game_type: 1,
  source_type: 'manual',
  source_name: '',
  source_url: '',
  cover_image: '',
  summary: '',
  content: '',
  country: '',
  city: '',
  venue: '',
  start_time: getNowString(),
  end_time: '',
  status: 0,
  stage_text: '',
  result_text: '',
  featured: false,
  sort_time: getNowString()
})

const formModel = ref<EventNewsFormModel>(createEmptyForm())

const gameTypeOptions = [
  { label: '斯诺克', value: 1 },
  { label: '中式八球', value: 3 },
  { label: '中式九球', value: 2 }
]

const sourceTypeOptions = [
  { label: '手动录入', value: 'manual' },
  { label: '官方来源', value: 'official' },
  { label: '导入同步', value: 'imported' }
]

const statusOptions = [
  { label: '即将开始', value: 0 },
  { label: '进行中', value: 1 },
  { label: '已结束', value: 2 },
  { label: '已取消', value: 3 }
]

const rules: FormRules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  game_type: [{ required: true, message: '请选择球种', trigger: 'change' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新建赛事情报草稿' : '编辑赛事情报'))

const getGameTypeLabel = (value: number) => {
  return gameTypeOptions.find(item => item.value === value)?.label || '-'
}

const getStatusLabel = (value: number) => {
  return statusOptions.find(item => item.value === value)?.label || '-'
}

const getStatusTagType = (value: number): 'success' | 'warning' | 'danger' | 'info' => {
  switch (value) {
    case 1:
      return 'warning'
    case 2:
      return 'success'
    case 3:
      return 'danger'
    default:
      return 'info'
  }
}

const applyEmptyForm = () => {
  formModel.value = createEmptyForm()
}

const loadList = async () => {
  loading.value = true
  try {
    const params: {
      page: number
      page_size: number
      game_type?: number
      status?: number
      published?: number
    } = {
      page: page.value,
      page_size: pageSize.value
    }

    if (filterGameType.value !== -1) {
      params.game_type = filterGameType.value
    }
    if (filterStatus.value !== -1) {
      params.status = filterStatus.value
    }
    if (filterPublished.value !== -1) {
      params.published = filterPublished.value
    }

    const res = await getEventNewsList(params)
    if (res.success) {
      eventNewsList.value = res.list || []
      total.value = res.total || 0
    } else {
      ElMessage.error(res.message || '获取赛事情报列表失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取赛事情报列表失败')
  } finally {
    loading.value = false
  }
}

const handleFilter = () => {
  page.value = 1
  loadList()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  loadList()
}

const handlePageChange = (val: number) => {
  page.value = val
  loadList()
}

const openCreateDialog = () => {
  dialogMode.value = 'create'
  editingId.value = null
  applyEmptyForm()
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

const openEditDialog = (row: EventNewsItem) => {
  dialogMode.value = 'edit'
  editingId.value = row.id
  formModel.value = {
    event_news_id: row.id,
    title: row.title,
    game_type: row.game_type,
    source_type: row.source_type || 'manual',
    source_name: row.source_name || '',
    source_url: row.source_url || '',
    cover_image: row.cover_image || '',
    summary: row.summary || '',
    content: row.content || '',
    country: row.country || '',
    city: row.city || '',
    venue: row.venue || '',
    start_time: row.start_time || '',
    end_time: row.end_time || '',
    status: row.status,
    stage_text: row.stage_text || '',
    result_text: row.result_text || '',
    featured: row.featured,
    sort_time: row.sort_time || row.start_time || getNowString()
  }
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

const buildFormPayload = () => {
  const form = formModel.value
  const startTime = (form.start_time || '').trim()
  const sortTime = (form.sort_time || '').trim() || startTime

  return {
    title: form.title.trim(),
    game_type: form.game_type ?? 0,
    source_type: form.source_type.trim(),
    source_name: form.source_name.trim(),
    source_url: form.source_url.trim(),
    cover_image: form.cover_image.trim(),
    summary: form.summary.trim(),
    content: form.content,
    country: form.country.trim(),
    city: form.city.trim(),
    venue: form.venue.trim(),
    start_time: startTime,
    end_time: (form.end_time || '').trim(),
    status: form.status ?? 0,
    stage_text: form.stage_text.trim(),
    result_text: form.result_text.trim(),
    featured: form.featured,
    sort_time: sortTime
  } as EventNewsFormPayload
}

const handleSubmit = async () => {
  const valid = formRef.value ? await formRef.value.validate().catch(() => false) : false
  if (!valid) {
    return
  }

  const payload = buildFormPayload()
  if (!payload.start_time && !payload.sort_time) {
    ElMessage.warning('开始时间和排序时间至少填写一个')
    return
  }

  saving.value = true
  try {
    const res = dialogMode.value === 'create'
      ? await createEventNews(payload)
      : await updateEventNews({
          event_news_id: editingId.value || 0,
          ...payload
        })

    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      dialogVisible.value = false
      loadList()
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const handlePublishToggle = async (row: EventNewsItem) => {
  const targetPublished = !row.published
  const action = targetPublished ? '发布' : '下线'

  try {
    await ElMessageBox.confirm(`确定要${action}「${row.title}」吗？`, '确认操作', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: targetPublished ? 'success' : 'warning'
    })

    const res = await publishEventNews({
      event_news_id: row.id,
      published: targetPublished
    })

    if (res.success) {
      ElMessage.success(res.message || `${action}成功`)
      loadList()
    } else {
      ElMessage.error(res.message || `${action}失败`)
    }
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || `${action}失败`)
    }
  }
}

const handleDelete = async (row: EventNewsItem) => {
  try {
    await ElMessageBox.confirm(`确定要删除「${row.title}」吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const res = await deleteEventNews({ event_news_id: row.id })
    if (res.success) {
      ElMessage.success(res.message || '删除成功')
      loadList()
    } else {
      ElMessage.error(res.message || '删除失败')
    }
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

onMounted(() => {
  loadList()
})
</script>

<style scoped lang="scss">
.event-news-page {
  padding: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.title-block {
  display: flex;
  flex-direction: column;
  gap: 4px;

  .title {
    font-size: 18px;
    font-weight: 600;
    color: #303133;
  }

  .subtitle {
    font-size: 13px;
    color: #909399;
  }
}

.header-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
