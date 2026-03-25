<template>
  <div class="event-news-page">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <div class="title-block">
            <span class="title">赛事情报管理</span>
            <span class="subtitle">按赛事维护基础信息，并在赛事下管理阶段赛程</span>
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
            <el-button type="primary" @click="openCreateDialog">新建赛事</el-button>
          </div>
        </div>
      </template>

      <el-table :data="eventNewsList" stripe border>
        <el-table-column type="index" width="60" label="序号" :index="(index: number) => (page - 1) * pageSize + index + 1" />
        <el-table-column prop="title" label="赛事标题" min-width="220" show-overflow-tooltip />
        <el-table-column prop="game_type" label="球种" width="120">
          <template #default="{ row }">
            {{ getGameTypeLabel(row.game_type) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="赛事状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="current_stage_text" label="当前阶段" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.current_stage_text || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="latest_result_text" label="最新赛果" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.latest_result_text || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="stage_count" label="阶段数" width="90" />
        <el-table-column prop="featured" label="焦点" width="90">
          <template #default="{ row }">
            <el-tag :type="row.featured ? 'warning' : 'info'">
              {{ row.featured ? '焦点' : '普通' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="published" label="发布状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.published ? 'success' : 'info'">
              {{ row.published ? '已发布' : '未发布' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="开始时间" width="170" />
        <el-table-column prop="updated_at" label="更新时间" width="170" />
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="primary" @click="openEditDialog(row, true)">管理阶段</el-button>
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
      width="1100px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="formModel" :rules="rules" label-width="110px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="赛事标题" prop="title">
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
              <el-input v-model="formModel.content" type="textarea" :rows="5" placeholder="赛事介绍正文" />
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
            <el-form-item label="赛事状态" prop="status">
              <el-select v-model="formModel.status" placeholder="请选择状态" style="width: 100%">
                <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="焦点推荐">
              <el-switch v-model="formModel.featured" active-text="是" inactive-text="否" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
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

      <el-divider content-position="left">阶段赛程</el-divider>

      <div class="stage-header">
        <div class="stage-copy">
          <span class="stage-title">阶段列表</span>
          <span class="stage-tip">赛事保存后即可继续维护资格赛、32 强、16 强等阶段信息。</span>
        </div>
        <el-button type="primary" :disabled="!editingId" @click="openCreateStageDialog">新增阶段</el-button>
      </div>

      <el-alert
        v-if="!editingId"
        title="请先保存赛事基础信息，再进入编辑态维护阶段赛程。"
        type="info"
        :closable="false"
        class="stage-alert"
      />

      <el-table v-else :data="stageList" border stripe class="stage-table">
        <el-table-column type="index" width="60" label="序号" />
        <el-table-column prop="stage_name" label="阶段名称" min-width="160" />
        <el-table-column prop="stage_order" label="阶段排序" width="100" />
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="开始时间" width="170">
          <template #default="{ row }">
            {{ row.start_time || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="result_text" label="赛果摘要" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.result_text || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditStageDialog(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDeleteStage(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSubmit">保存赛事</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="stageDialogVisible"
      :title="stageDialogTitle"
      width="760px"
      destroy-on-close
    >
      <el-form ref="stageFormRef" :model="stageFormModel" :rules="stageRules" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="阶段名称" prop="stage_name">
              <el-input v-model="stageFormModel.stage_name" placeholder="如：资格赛 / 32强 / 决赛" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="阶段排序" prop="stage_order">
              <el-input-number v-model="stageFormModel.stage_order" :min="1" :step="10" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="开始时间">
              <el-date-picker
                v-model="stageFormModel.start_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="请选择开始时间"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结束时间">
              <el-date-picker
                v-model="stageFormModel.end_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="请选择结束时间"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="阶段状态" prop="status">
              <el-select v-model="stageFormModel.status" placeholder="请选择阶段状态" style="width: 100%">
                <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="排序时间">
              <el-date-picker
                v-model="stageFormModel.sort_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="不填则默认使用开始时间"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="赛果摘要">
              <el-input v-model="stageFormModel.result_text" type="textarea" :rows="3" placeholder="如：赵心童晋级16强" maxlength="255" show-word-limit />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <template #footer>
        <el-button @click="stageDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="stageSaving" @click="handleStageSubmit">保存阶段</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  createEventNews,
  createEventNewsStage,
  deleteEventNews,
  deleteEventNewsStage,
  getEventNewsList,
  publishEventNews,
  updateEventNews,
  updateEventNewsStage,
  type EventNewsFormPayload,
  type EventNewsItem,
  type EventNewsStageFormPayload,
  type EventNewsStageItem
} from '@/api/event-news'

interface EventNewsFormModel extends EventNewsFormPayload {
  event_id?: number
}

interface EventNewsStageFormModel {
  stage_id?: number
  stage_name: string
  stage_order: number
  start_time: string
  end_time: string
  status: number
  result_text: string
  sort_time: string
}

const loading = ref(false)
const saving = ref(false)
const stageSaving = ref(false)
const eventNewsList = ref<EventNewsItem[]>([])
const stageList = ref<EventNewsStageItem[]>([])
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

const stageDialogVisible = ref(false)
const stageDialogMode = ref<'create' | 'edit'>('create')
const stageFormRef = ref<FormInstance>()

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
  start_time: '',
  end_time: '',
  status: 0,
  featured: false,
  sort_time: ''
})

const createEmptyStageForm = (): EventNewsStageFormModel => ({
  stage_name: '',
  stage_order: 10,
  start_time: '',
  end_time: '',
  status: 0,
  result_text: '',
  sort_time: ''
})

const formModel = ref<EventNewsFormModel>(createEmptyForm())
const stageFormModel = ref<EventNewsStageFormModel>(createEmptyStageForm())

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
  title: [{ required: true, message: '请输入赛事标题', trigger: 'blur' }],
  game_type: [{ required: true, message: '请选择球种', trigger: 'change' }],
  status: [{ required: true, message: '请选择赛事状态', trigger: 'change' }]
}

const stageRules: FormRules = {
  stage_name: [{ required: true, message: '请输入阶段名称', trigger: 'blur' }],
  stage_order: [{ required: true, message: '请输入阶段排序', trigger: 'change' }],
  status: [{ required: true, message: '请选择阶段状态', trigger: 'change' }]
}

const dialogTitle = computed(() => (dialogMode.value === 'create' ? '新建赛事' : '编辑赛事'))
const stageDialogTitle = computed(() => (stageDialogMode.value === 'create' ? '新增阶段' : '编辑阶段'))

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

const normalizeStages = (stages: EventNewsStageItem[] = []) => {
  return [...stages].sort((left, right) => {
    if (left.stage_order !== right.stage_order) {
      return left.stage_order - right.stage_order
    }
    return left.id - right.id
  })
}

const applyEmptyForm = () => {
  formModel.value = createEmptyForm()
}

const applyEmptyStageForm = () => {
  stageFormModel.value = createEmptyStageForm()
}

const refreshEditingEventFromList = () => {
  if (!editingId.value) {
    stageList.value = []
    return
  }
  const latest = eventNewsList.value.find(item => item.id === editingId.value)
  stageList.value = normalizeStages(latest?.stages || [])
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
      refreshEditingEventFromList()
    } else {
      console.error('获取赛事情报列表失败', res.message)
    }
  } catch (error: any) {
    console.error('获取赛事情报列表失败', error)
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
  stageList.value = []
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

const openEditDialog = (row: EventNewsItem, focusStages = false) => {
  dialogMode.value = 'edit'
  editingId.value = row.id
  formModel.value = {
    event_id: row.id,
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
    featured: row.featured,
    sort_time: row.sort_time || row.start_time || ''
  }
  stageList.value = normalizeStages(row.stages || [])
  dialogVisible.value = true
  formRef.value?.clearValidate()
  if (focusStages) {
    setTimeout(() => {
      const table = document.querySelector('.stage-table')
      table?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    }, 50)
  }
}

const buildFormPayload = (): EventNewsFormPayload => {
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
    featured: form.featured,
    sort_time: sortTime
  }
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
          event_id: editingId.value || 0,
          ...payload
        })

    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      dialogVisible.value = false
      await loadList()
    }
  } catch (error: any) {
    console.error('保存赛事失败', error)
  } finally {
    saving.value = false
  }
}

const openCreateStageDialog = () => {
  if (!editingId.value) {
    ElMessage.warning('请先保存赛事基础信息')
    return
  }
  stageDialogMode.value = 'create'
  applyEmptyStageForm()
  stageDialogVisible.value = true
  stageFormRef.value?.clearValidate()
}

const openEditStageDialog = (row: EventNewsStageItem) => {
  stageDialogMode.value = 'edit'
  stageFormModel.value = {
    stage_id: row.id,
    stage_name: row.stage_name || '',
    stage_order: row.stage_order || 10,
    start_time: row.start_time || '',
    end_time: row.end_time || '',
    status: row.status,
    result_text: row.result_text || '',
    sort_time: row.sort_time || row.start_time || ''
  }
  stageDialogVisible.value = true
  stageFormRef.value?.clearValidate()
}

const buildStagePayload = (): EventNewsStageFormPayload => {
  const form = stageFormModel.value
  const startTime = (form.start_time || '').trim()
  const sortTime = (form.sort_time || '').trim() || startTime

  return {
    event_id: editingId.value || 0,
    stage_name: form.stage_name.trim(),
    stage_order: Number(form.stage_order || 0),
    start_time: startTime,
    end_time: (form.end_time || '').trim(),
    status: form.status ?? 0,
    result_text: form.result_text.trim(),
    sort_time: sortTime
  }
}

const handleStageSubmit = async () => {
  if (!editingId.value) {
    ElMessage.warning('请先保存赛事基础信息')
    return
  }
  const valid = stageFormRef.value ? await stageFormRef.value.validate().catch(() => false) : false
  if (!valid) {
    return
  }

  const payload = buildStagePayload()
  stageSaving.value = true
  try {
    const res = stageDialogMode.value === 'create'
      ? await createEventNewsStage(payload)
      : await updateEventNewsStage({
          stage_id: stageFormModel.value.stage_id || 0,
          ...payload
        })

    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      stageDialogVisible.value = false
      await loadList()
      refreshEditingEventFromList()
    }
  } catch (error: any) {
    console.error('保存阶段失败', error)
  } finally {
    stageSaving.value = false
  }
}

const handleDeleteStage = async (row: EventNewsStageItem) => {
  try {
    await ElMessageBox.confirm(`确定要删除阶段「${row.stage_name}」吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const res = await deleteEventNewsStage({ stage_id: row.id })
    if (res.success) {
      ElMessage.success(res.message || '删除成功')
      await loadList()
      refreshEditingEventFromList()
    }
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除阶段失败', error)
    }
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
      event_id: row.id,
      published: targetPublished
    })

    if (res.success) {
      ElMessage.success(res.message || `${action}成功`)
      await loadList()
    }
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      console.error(`${action}赛事失败`, error)
    }
  }
}

const handleDelete = async (row: EventNewsItem) => {
  try {
    await ElMessageBox.confirm(`确定要删除「${row.title}」及其全部阶段吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const res = await deleteEventNews({ event_id: row.id })
    if (res.success) {
      ElMessage.success(res.message || '删除成功')
      if (editingId.value === row.id) {
        dialogVisible.value = false
      }
      await loadList()
    }
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除赛事失败', error)
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

.stage-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.stage-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stage-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.stage-tip {
  font-size: 12px;
  color: #909399;
}

.stage-alert {
  margin-bottom: 12px;
}

.stage-table {
  margin-bottom: 12px;
}
</style>
