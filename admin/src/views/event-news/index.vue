<template>
  <div class="event-news-page">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <div class="title-block">
            <span class="title">赛事情报管理</span>
            <span class="subtitle">按赛事维护基础信息，并在赛事下管理比赛列表</span>
          </div>
          <div class="header-actions">
            <el-select v-model="filterGameType" placeholder="球种筛选" style="width: 160px" @change="handleFilter">
              <el-option label="全部" :value="-1" />
              <el-option label="斯诺克" :value="1" />
              <el-option label="中式九球" :value="2" />
              <el-option label="中式八球" :value="3" />
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
        <el-table-column prop="title" label="赛事标题" min-width="200" show-overflow-tooltip />
        <el-table-column prop="tournament_name" label="赛事名称" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.tournament_name || row.title || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="game_type" label="球种" width="110">
          <template #default="{ row }">
            {{ getGameTypeLabel(row.game_type) }}
          </template>
        </el-table-column>
        <el-table-column prop="current_round_text" label="当前轮次" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.current_round_text || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="latest_result_text" label="最新赛果" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.latest_result_text || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="match_count" label="比赛数" width="90" />
        <el-table-column prop="status" label="赛事状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="赛事日期" min-width="180">
          <template #default="{ row }">
            <div class="date-cell">
              <span class="date-main">{{ formatDateRange(row.start_date, row.end_date) }}</span>
              <span v-if="hasEventTime(row)" class="date-sub">{{ formatTimeRange(row.start_time, row.end_time) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="published" label="发布状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.published ? 'success' : 'info'">
              {{ row.published ? '已发布' : '未发布' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="170" />
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="primary" @click="openMatchPanel(row)">管理比赛</el-button>
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
      v-model="eventDialogVisible"
      :title="eventDialogTitle"
      width="1100px"
      destroy-on-close
    >
      <el-form ref="eventFormRef" :model="eventFormModel" :rules="eventRules" label-width="110px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="赛事标题" prop="title">
              <el-input v-model="eventFormModel.title" placeholder="请输入赛事标题" maxlength="128" show-word-limit />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="赛事名称" prop="tournament_name">
              <el-input v-model="eventFormModel.tournament_name" placeholder="请输入赛事名称" maxlength="128" show-word-limit />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="球种" prop="game_type">
              <el-select v-model="eventFormModel.game_type" placeholder="请选择球种" style="width: 100%">
                <el-option v-for="item in gameTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="赛事状态" prop="status">
              <el-select v-model="eventFormModel.status" placeholder="请选择状态" style="width: 100%">
                <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="来源类型">
              <el-select v-model="eventFormModel.source_type" placeholder="请选择来源类型" style="width: 100%">
                <el-option v-for="item in sourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="来源名称">
              <el-input v-model="eventFormModel.source_name" placeholder="如：WST / 独牙传奇" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="来源链接">
              <el-input v-model="eventFormModel.source_url" placeholder="https://..." />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="封面图">
              <el-input v-model="eventFormModel.cover_image" placeholder="封面图片地址" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="摘要">
              <el-input v-model="eventFormModel.summary" type="textarea" :rows="2" placeholder="用于列表和首页焦点展示" maxlength="512" show-word-limit />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="赛讯正文">
              <el-input v-model="eventFormModel.content" type="textarea" :rows="5" placeholder="展示在赛讯详情页的正文内容" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="赛事描述">
              <el-input v-model="eventFormModel.description" type="textarea" :rows="3" placeholder="写入赛事实体描述，用于补充赛制与背景信息" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="国家">
              <el-input v-model="eventFormModel.country" placeholder="国家 / 地区" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="城市">
              <el-input v-model="eventFormModel.city" placeholder="举办城市" maxlength="64" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="场馆">
              <el-input v-model="eventFormModel.venue" placeholder="比赛场馆" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="开始日期" prop="start_date">
              <el-date-picker
                v-model="eventFormModel.start_date"
                type="date"
                value-format="YYYY-MM-DD"
                format="YYYY-MM-DD"
                placeholder="请选择开始日期"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结束日期">
              <el-date-picker
                v-model="eventFormModel.end_date"
                type="date"
                value-format="YYYY-MM-DD"
                format="YYYY-MM-DD"
                placeholder="不填则默认开始日期"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="开始时间">
              <el-date-picker
                v-model="eventFormModel.start_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="可选，未填则不显示"
                style="width: 100%"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结束时间">
              <el-date-picker
                v-model="eventFormModel.end_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="可选，未填则不显示"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="排序时间">
              <el-date-picker
                v-model="eventFormModel.sort_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="不填则默认开始时间或开始日期"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <template #footer>
        <el-button @click="eventDialogVisible = false">取消</el-button>
        <template v-if="eventDialogMode === 'create'">
          <el-button type="primary" plain :loading="eventSaving" @click="handleEventSubmit()">保存赛事</el-button>
          <el-button type="primary" :loading="eventSaving" @click="handleEventSubmit('matches')">保存并录入比赛</el-button>
        </template>
        <el-button v-else type="primary" :loading="eventSaving" @click="handleEventSubmit()">保存赛事</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="matchPanelVisible"
      :title="matchPanelTitle"
      width="1200px"
      destroy-on-close
    >
      <div class="match-panel">
        <div class="match-panel-header">
          <div class="match-panel-copy">
            <div class="match-panel-title">{{ currentEvent?.title || '-' }}</div>
            <div class="match-panel-subtitle">
              {{ currentEvent?.tournament_name || currentEvent?.title || '-' }}
              <span v-if="currentEvent?.match_count"> · 共 {{ currentEvent?.match_count }} 场</span>
            </div>
          </div>
          <el-button type="primary" :disabled="!currentEvent" @click="openCreateMatchDialog">新增比赛</el-button>
        </div>

        <el-table :data="matchList" border stripe class="match-table">
          <el-table-column prop="round_name" label="轮次" min-width="140" show-overflow-tooltip />
          <el-table-column prop="round_order" label="轮次排序" width="90" />
          <el-table-column prop="match_order" label="场次排序" width="90" />
          <el-table-column prop="start_time" label="开赛时间" width="170">
            <template #default="{ row }">
              {{ row.start_time || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">
                {{ getStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="best_of" label="best_of" width="90">
            <template #default="{ row }">
              {{ row.best_of || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="主侧选手" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              {{ formatPlayer(row.home_player_name, row.home_player_id) }}
            </template>
          </el-table-column>
          <el-table-column label="客侧选手" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              {{ formatPlayer(row.away_player_name, row.away_player_id) }}
            </template>
          </el-table-column>
          <el-table-column label="比分" width="110">
            <template #default="{ row }">
              {{ formatScore(row.home_score, row.away_score) }}
            </template>
          </el-table-column>
          <el-table-column prop="winner_side" label="胜方" width="90">
            <template #default="{ row }">
              {{ getWinnerSideLabel(row.winner_side) }}
            </template>
          </el-table-column>
          <el-table-column prop="is_placeholder" label="占位" width="90">
            <template #default="{ row }">
              {{ row.is_placeholder ? '是' : '否' }}
            </template>
          </el-table-column>
          <el-table-column label="来源" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">
              {{ formatSource(row.source_type, row.source_match_id) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEditMatchDialog(row)">编辑</el-button>
              <el-button link type="danger" @click="handleDeleteMatch(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <el-button @click="matchPanelVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="matchDialogVisible"
      :title="matchDialogTitle"
      width="980px"
      append-to-body
      destroy-on-close
    >
      <el-form ref="matchFormRef" :model="matchFormModel" :rules="matchRules" label-width="110px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="轮次名称" prop="round_name">
              <el-input v-model="matchFormModel.round_name" placeholder="如：Quarter Finals / 决赛" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="轮次排序" prop="round_order">
              <el-input-number v-model="matchFormModel.round_order" :min="1" :step="10" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="场次排序" prop="match_order">
              <el-input-number v-model="matchFormModel.match_order" :min="1" :step="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="开赛时间">
              <el-date-picker
                v-model="matchFormModel.start_time"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="请选择开赛时间"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="状态" prop="status">
              <el-select v-model="matchFormModel.status" placeholder="请选择状态" style="width: 100%">
                <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="best_of">
              <el-input-number v-model="matchFormModel.best_of" :min="0" :step="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="主侧选手ID">
              <el-input-number v-model="matchFormModel.home_player_id" :min="0" :step="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="主侧选手名">
              <el-input v-model="matchFormModel.home_player_name" placeholder="主侧选手名称" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="客侧选手ID">
              <el-input-number v-model="matchFormModel.away_player_id" :min="0" :step="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="客侧选手名">
              <el-input v-model="matchFormModel.away_player_name" placeholder="客侧选手名称" maxlength="128" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="主侧比分">
              <el-input-number v-model="matchFormModel.home_score" :min="0" :step="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="客侧比分">
              <el-input-number v-model="matchFormModel.away_score" :min="0" :step="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="胜方">
              <el-select v-model="matchFormModel.winner_side" placeholder="请选择胜方" style="width: 100%">
                <el-option label="未定" :value="0" />
                <el-option label="主侧" :value="1" />
                <el-option label="客侧" :value="2" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="占位">
              <el-switch v-model="matchFormModel.is_placeholder" active-text="是" inactive-text="否" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="来源类型">
              <el-select v-model="matchFormModel.source_type" placeholder="请选择来源类型" style="width: 100%">
                <el-option v-for="item in sourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="来源比赛ID">
              <el-input v-model="matchFormModel.source_match_id" placeholder="外部赛事源比赛ID" maxlength="128" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <template #footer>
        <el-button @click="matchDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="matchSaving" @click="handleMatchSubmit">保存比赛</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  createEventNews,
  createEventNewsMatch,
  deleteEventNews,
  deleteEventNewsMatch,
  getEventNewsList,
  getEventNewsMatches,
  publishEventNews,
  updateEventNews,
  updateEventNewsMatch,
  type EventNewsFormPayload,
  type EventNewsItem,
  type EventNewsMatchCreatePayload,
  type EventNewsMatchItem
} from '@/api/event-news'

interface EventNewsFormModel extends EventNewsFormPayload {
  event_id?: number
}

interface EventNewsMatchFormModel extends EventNewsMatchCreatePayload {
  match_id?: number
}

const loading = ref(false)
const eventSaving = ref(false)
const matchSaving = ref(false)
const eventNewsList = ref<EventNewsItem[]>([])
const matchList = ref<EventNewsMatchItem[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterGameType = ref(-1)
const filterStatus = ref(-1)
const filterPublished = ref(-1)

const eventDialogVisible = ref(false)
const eventDialogMode = ref<'create' | 'edit'>('create')
const eventFormRef = ref<FormInstance>()
const currentEvent = ref<EventNewsItem | null>(null)
const matchPanelVisible = ref(false)
const matchDialogVisible = ref(false)
const matchDialogMode = ref<'create' | 'edit'>('create')
const matchFormRef = ref<FormInstance>()
const defaultEventCoverImage = 'https://images.gc.wstservices.co.uk/fit-in/400x600/4ddad400-99d3-11ee-94e8-c9d138e537ff.png'

const createEmptyEventForm = (): EventNewsFormModel => ({
  title: '',
  tournament_name: '',
  game_type: 1,
  source_type: 'manual',
  source_name: '',
  source_url: '',
  cover_image: defaultEventCoverImage,
  summary: '',
  content: '',
  description: '',
  country: '',
  city: '',
  venue: '',
  start_date: '',
  end_date: '',
  start_time: '',
  end_time: '',
  sort_time: '',
  status: 0,
  published: false
})

const createEmptyMatchForm = (eventId = 0): EventNewsMatchFormModel => ({
  event_id: eventId,
  round_name: '',
  round_order: 10,
  match_order: 1,
  start_time: '',
  status: 0,
  best_of: 0,
  home_player_id: 0,
  home_player_name: '',
  away_player_id: 0,
  away_player_name: '',
  home_score: 0,
  away_score: 0,
  winner_side: 0,
  is_placeholder: false,
  source_type: 'manual',
  source_match_id: ''
})

const eventFormModel = ref<EventNewsFormModel>(createEmptyEventForm())
const matchFormModel = ref<EventNewsMatchFormModel>(createEmptyMatchForm())

const gameTypeOptions = [
  { label: '斯诺克', value: 1 },
  { label: '中式九球', value: 2 },
  { label: '中式八球', value: 3 }
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

const eventRules: FormRules = {
  title: [{ required: true, message: '请输入赛事标题', trigger: 'blur' }],
  tournament_name: [{ required: true, message: '请输入赛事名称', trigger: 'blur' }],
  game_type: [{ required: true, message: '请选择球种', trigger: 'change' }],
  start_date: [{ required: true, message: '请选择开始日期', trigger: 'change' }],
  status: [{ required: true, message: '请选择赛事状态', trigger: 'change' }]
}

const matchRules: FormRules = {
  round_name: [{ required: true, message: '请输入轮次名称', trigger: 'blur' }],
  round_order: [{ required: true, message: '请输入轮次排序', trigger: 'change' }],
  match_order: [{ required: true, message: '请输入场次排序', trigger: 'change' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

const eventDialogTitle = computed(() => (eventDialogMode.value === 'create' ? '新建赛事' : '编辑赛事'))
const matchPanelTitle = computed(() => `比赛管理${currentEvent.value?.title ? ` - ${currentEvent.value.title}` : ''}`)
const matchDialogTitle = computed(() => (matchDialogMode.value === 'create' ? '新增比赛' : '编辑比赛'))

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

const formatPlayer = (name: string, id: number) => {
  const trimmed = (name || '').trim()
  if (trimmed) {
    return id > 0 ? `${trimmed} (#${id})` : trimmed
  }
  return id > 0 ? `#${id}` : '-'
}

const formatScore = (homeScore: number, awayScore: number) => {
  if (homeScore === 0 && awayScore === 0) {
    return '-'
  }
  return `${homeScore} - ${awayScore}`
}

const formatSource = (sourceType: string, sourceMatchId: string) => {
  const prefix = sourceType || 'manual'
  const suffix = (sourceMatchId || '').trim()
  return suffix ? `${prefix} / ${suffix}` : prefix
}

const formatDateRange = (startDate: string, endDate: string) => {
  const start = (startDate || '').trim()
  const end = (endDate || '').trim()

  if (!start && !end) return '-'
  if (start && !end) return start
  if (!start && end) return end
  if (start === end) return start
  return `${start} - ${end}`
}

const formatTimeRange = (startTime: string, endTime: string) => {
  const start = (startTime || '').trim()
  const end = (endTime || '').trim()

  if (!start && !end) return ''
  if (start && !end) return start
  if (!start && end) return end
  if (start === end) return start
  return `${start} - ${end}`
}

const hasEventTime = (row: EventNewsItem) => Boolean((row.start_time || '').trim() || (row.end_time || '').trim())

const getWinnerSideLabel = (value: number) => {
  if (value === 1) {
    return '主侧'
  }
  if (value === 2) {
    return '客侧'
  }
  return '-'
}

const sortMatchList = (list: EventNewsMatchItem[] = []) => {
  return [...list].sort((left, right) => {
    if (left.round_order !== right.round_order) {
      return left.round_order - right.round_order
    }
    if (left.match_order !== right.match_order) {
      return left.match_order - right.match_order
    }
    return left.id - right.id
  })
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
      if (currentEvent.value) {
        const refreshed = eventNewsList.value.find(item => item.id === currentEvent.value?.id)
        currentEvent.value = refreshed || currentEvent.value
      }
    } else {
      console.error('获取赛事情报列表失败', res.message)
    }
  } catch (error) {
    console.error('获取赛事情报列表失败', error)
  } finally {
    loading.value = false
  }
}

const loadMatches = async (eventId: number) => {
  if (!eventId) {
    matchList.value = []
    return
  }

  try {
    const res = await getEventNewsMatches({ event_id: eventId })
    if (res.success) {
      matchList.value = sortMatchList(res.list || [])
    } else {
      matchList.value = []
      console.error('获取比赛列表失败', res.message)
    }
  } catch (error) {
    matchList.value = []
    console.error('获取比赛列表失败', error)
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
  eventDialogMode.value = 'create'
  eventFormModel.value = createEmptyEventForm()
  eventDialogVisible.value = true
  eventFormRef.value?.clearValidate()
}

const openEditDialog = (row: EventNewsItem) => {
  eventDialogMode.value = 'edit'
  eventFormModel.value = {
    event_id: row.id,
    tournament_id: row.tournament_id || 0,
    title: row.title,
    tournament_name: row.tournament_name || row.title || '',
    game_type: row.game_type,
    source_type: row.source_type || 'manual',
    source_name: row.source_name || '',
    source_url: row.source_url || '',
    cover_image: row.cover_image || defaultEventCoverImage,
    summary: row.summary || '',
    content: row.content || '',
    description: row.description || '',
    country: row.country || '',
    city: row.city || '',
    venue: row.venue || '',
    start_date: row.start_date || '',
    end_date: row.end_date || row.start_date || '',
    start_time: row.start_time || '',
    end_time: row.end_time || '',
    sort_time: row.sort_time || '',
    status: row.status,
    published: row.published
  }
  eventDialogVisible.value = true
  eventFormRef.value?.clearValidate()
}

const buildEventPayload = (): EventNewsFormPayload => {
  const form = eventFormModel.value
  const startDate = (form.start_date || '').trim()
  const endDate = (form.end_date || '').trim() || startDate
  const startTime = (form.start_time || '').trim()
  const endTime = (form.end_time || '').trim()
  const sortTime = (form.sort_time || '').trim()
  const coverImage = (form.cover_image || '').trim() || defaultEventCoverImage

  return {
    title: form.title.trim(),
    tournament_name: form.tournament_name.trim() || form.title.trim(),
    game_type: form.game_type ?? 0,
    source_type: form.source_type.trim(),
    source_name: form.source_name.trim(),
    source_url: form.source_url.trim(),
    cover_image: coverImage,
    summary: form.summary.trim(),
    content: form.content,
    description: (form.description || '').trim(),
    country: form.country.trim(),
    city: form.city.trim(),
    venue: form.venue.trim(),
    start_date: startDate,
    end_date: endDate,
    start_time: startTime || undefined,
    end_time: endTime || undefined,
    sort_time: sortTime || undefined,
    status: form.status ?? 0,
    published: !!form.published,
    tournament_id: form.tournament_id && form.tournament_id > 0 ? form.tournament_id : undefined
  }
}

const handleEventSubmit = async (nextAction?: 'matches') => {
  const valid = eventFormRef.value ? await eventFormRef.value.validate().catch(() => false) : false
  if (!valid) {
    return
  }

  const payload = buildEventPayload()
  eventSaving.value = true
  try {
    const res = eventDialogMode.value === 'create'
      ? await createEventNews(payload)
      : await updateEventNews({
          event_id: eventFormModel.value.event_id || 0,
          ...payload
        })

    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      eventDialogVisible.value = false
      await loadList()
      if (nextAction === 'matches' && eventDialogMode.value === 'create' && res.event_id) {
        const createdEvent = eventNewsList.value.find(item => item.id === res.event_id) || {
          id: res.event_id,
          title: payload.title,
          tournament_name: payload.tournament_name,
          match_count: 0
        } as EventNewsItem
        await openMatchPanel(createdEvent)
        openCreateMatchDialog()
      }
    }
  } catch (error) {
    console.error('保存赛事失败', error)
  } finally {
    eventSaving.value = false
  }
}

const openMatchPanel = async (row: EventNewsItem) => {
  currentEvent.value = row
  matchPanelVisible.value = true
  await loadMatches(row.id)
}

const openCreateMatchDialog = () => {
  if (!currentEvent.value) {
    ElMessage.warning('请先选择赛事')
    return
  }
  matchDialogMode.value = 'create'
  matchFormModel.value = createEmptyMatchForm(currentEvent.value.id)
  matchDialogVisible.value = true
  matchFormRef.value?.clearValidate()
}

const openEditMatchDialog = (row: EventNewsMatchItem) => {
  matchDialogMode.value = 'edit'
  matchFormModel.value = {
    match_id: row.id,
    event_id: row.event_id,
    round_name: row.round_name || '',
    round_order: row.round_order || 10,
    match_order: row.match_order || 1,
    start_time: row.start_time || '',
    status: row.status,
    best_of: row.best_of || 0,
    home_player_id: row.home_player_id || 0,
    home_player_name: row.home_player_name || '',
    away_player_id: row.away_player_id || 0,
    away_player_name: row.away_player_name || '',
    home_score: row.home_score || 0,
    away_score: row.away_score || 0,
    winner_side: row.winner_side || 0,
    is_placeholder: row.is_placeholder || false,
    source_type: row.source_type || 'manual',
    source_match_id: row.source_match_id || ''
  }
  matchDialogVisible.value = true
  matchFormRef.value?.clearValidate()
}

const buildMatchPayload = (): EventNewsMatchCreatePayload => {
  const form = matchFormModel.value
  return {
    event_id: form.event_id || currentEvent.value?.id || 0,
    round_name: form.round_name.trim(),
    round_order: Number(form.round_order || 0),
    match_order: Number(form.match_order || 0),
    start_time: (form.start_time || '').trim(),
    status: form.status ?? 0,
    best_of: Number(form.best_of || 0),
    home_player_id: Number(form.home_player_id || 0),
    home_player_name: form.home_player_name.trim(),
    away_player_id: Number(form.away_player_id || 0),
    away_player_name: form.away_player_name.trim(),
    home_score: Number(form.home_score || 0),
    away_score: Number(form.away_score || 0),
    winner_side: Number(form.winner_side || 0),
    is_placeholder: !!form.is_placeholder,
    source_type: form.source_type.trim(),
    source_match_id: form.source_match_id.trim()
  }
}

const handleMatchSubmit = async () => {
  const valid = matchFormRef.value ? await matchFormRef.value.validate().catch(() => false) : false
  if (!valid) {
    return
  }

  const payload = buildMatchPayload()
  if (!payload.event_id) {
    ElMessage.warning('请先选择赛事')
    return
  }

  matchSaving.value = true
  try {
    const res = matchDialogMode.value === 'create'
      ? await createEventNewsMatch(payload)
      : await updateEventNewsMatch({
          match_id: matchFormModel.value.match_id || 0,
          ...payload
        })

    if (res.success) {
      ElMessage.success(res.message || '保存成功')
      matchDialogVisible.value = false
      if (currentEvent.value) {
        await loadMatches(currentEvent.value.id)
        await loadList()
      }
    }
  } catch (error) {
    console.error('保存比赛失败', error)
  } finally {
    matchSaving.value = false
  }
}

const handleDeleteMatch = async (row: EventNewsMatchItem) => {
  try {
    await ElMessageBox.confirm(`确定要删除轮次「${row.round_name}」中的这场比赛吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const res = await deleteEventNewsMatch({ match_id: row.id })
    if (res.success) {
      ElMessage.success(res.message || '删除成功')
      if (currentEvent.value) {
        await loadMatches(currentEvent.value.id)
        await loadList()
      }
    }
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除比赛失败', error)
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
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      console.error(`${action}赛事失败`, error)
    }
  }
}

const handleDelete = async (row: EventNewsItem) => {
  try {
    await ElMessageBox.confirm(`确定要删除「${row.title}」及其关联比赛吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const res = await deleteEventNews({ event_id: row.id })
    if (res.success) {
      ElMessage.success(res.message || '删除成功')
      if (currentEvent.value?.id === row.id) {
        currentEvent.value = null
        matchList.value = []
        matchPanelVisible.value = false
      }
      await loadList()
    }
  } catch (error) {
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

.match-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.match-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.match-panel-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.match-panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.match-panel-subtitle {
  font-size: 12px;
  color: #909399;
}

.match-table {
  margin-top: 4px;
}

.date-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.date-main {
  font-size: 13px;
  color: #303133;
  line-height: 1.4;
}

.date-sub {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}
</style>
