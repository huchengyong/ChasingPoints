<template>
  <view class="tournament-page" :class="{ 'dark-mode': isDarkMode }">
    <view class="hero-card">
      <view class="hero-copy">
        <text class="hero-eyebrow">赛讯</text>
        <text class="hero-title">官方赛事信息集中看</text>
        <text class="hero-desc">第一时间掌握各大赛事的赛程与赛况</text>
      </view>
    </view>

    <view class="filter-bar">
      <view class="filter-chip" @tap="openYearSheet">
        <text class="filter-chip-label">年份</text>
        <text class="filter-chip-value">{{ filter.year }}年</text>
        <uni-icons type="bottom" size="16" :color="isDarkMode ? '#9f926e' : '#a16207'"></uni-icons>
      </view>
      <view class="filter-chip" @tap="openDatePresetSheet">
        <text class="filter-chip-label">日期</text>
        <text class="filter-chip-value">{{ filterDateLabel }}</text>
        <uni-icons type="bottom" size="16" :color="isDarkMode ? '#9f926e' : '#a16207'"></uni-icons>
      </view>
      <view class="filter-chip" @tap="openFilterSheet">
        <text class="filter-chip-label">筛选</text>
        <text class="filter-chip-value">{{ filterSummary }}</text>
        <uni-icons type="bottom" size="16" :color="isDarkMode ? '#9f926e' : '#a16207'"></uni-icons>
      </view>
    </view>

    <view v-if="showFilterSheet" class="filter-sheet-backdrop" @tap="closeFilterSheet">
      <view class="filter-sheet" @tap.stop>
        <view class="filter-sheet-header">
          <text class="filter-sheet-title">筛选</text>
          <text class="filter-sheet-close" @tap="closeFilterSheet">完成</text>
        </view>
        <view class="filter-sheet-body">
          <picker :range="GAME_TYPE_OPTIONS" range-key="label" @change="onGameTypeChange">
            <view class="filter-sheet-row">
              <text class="filter-sheet-row-label">球种</text>
              <view class="filter-sheet-row-value">
                <text>{{ selectedGameType.label }}</text>
                <uni-icons type="right" size="16" :color="isDarkMode ? '#9f926e' : '#a16207'"></uni-icons>
              </view>
            </view>
          </picker>
          <picker :range="STATUS_OPTIONS" range-key="label" @change="onStatusChange">
            <view class="filter-sheet-row">
              <text class="filter-sheet-row-label">状态</text>
              <view class="filter-sheet-row-value">
                <text>{{ selectedStatus.label }}</text>
                <uni-icons type="right" size="16" :color="isDarkMode ? '#9f926e' : '#a16207'"></uni-icons>
              </view>
            </view>
          </picker>
        </view>
      </view>
    </view>

    <view v-if="showCustomDateEditor" class="custom-date-card">
      <picker mode="date" :value="customDateDraft.from" @change="onCustomFromChange">
        <view class="custom-date-field">
          <text class="custom-date-label">开始日期</text>
          <text class="custom-date-value">{{ customDateDraft.from }}</text>
        </view>
      </picker>
      <picker mode="date" :value="customDateDraft.to" @change="onCustomToChange">
        <view class="custom-date-field">
          <text class="custom-date-label">结束日期</text>
          <text class="custom-date-value">{{ customDateDraft.to }}</text>
        </view>
      </picker>
    </view>

    <scroll-view
      scroll-y
      class="feed-scroll"
      refresher-enabled
      :refresher-triggered="refreshing"
      @refresherrefresh="onRefresh"
      @scrolltolower="loadMore"
	  >
	    <view v-if="loading" class="state-block">
        <uni-icons type="spinner-cycle" size="34" color="#E0AE12"></uni-icons>
        <text class="state-text">加载中...</text>
      </view>

	    <view v-else-if="list.length > 0" class="post-list">
	      <view v-if="eventNewsRefreshError" class="refresh-error-banner">
	        <text>{{ eventNewsRefreshError.description }}</text>
	        <text class="refresh-error-action" @tap="retryEventNews">重试</text>
	      </view>
	      <view v-for="item in list" :key="item.id" class="post-card" @tap="openDetail(item.id)">
          <image class="post-cover" :src="item.coverImage" mode="aspectFill"></image>
          <view class="post-overlay"></view>
          <view class="post-body">
            <view class="post-topline">
              <text class="post-date">{{ item.dateText }}</text>
              <view class="post-status" :class="'status-' + item.status">
                <text>{{ item.statusText }}</text>
              </view>
            </view>
            <view class="post-content-block">
              <text class="post-eyebrow">{{ item.gameTypeText }}</text>
              <text class="post-title">{{ item.title }}</text>
            </view>
          </view>
        </view>
      </view>

	    <view v-else-if="eventNewsPageError" class="state-block error-state">
	      <uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
	      <text class="state-title">{{ eventNewsPageError.title }}</text>
	      <text class="state-text">{{ eventNewsPageError.description }}</text>
	      <button class="retry-btn" @tap="handleEventNewsErrorAction">{{ eventNewsPageError.actionText }}</button>
	    </view>

	    <view v-else-if="eventNewsState.status === ASYNC_PAGE_STATUS.EMPTY" class="state-block">
        <text class="state-icon">🗓️</text>
        <text class="state-title">暂时还没有可看的赛讯</text>
        <text class="state-text">官方赛事内容更新中，稍后再来刷新看看。</text>
      </view>

      <view v-if="hasMore && list.length > 0" class="load-more">
        <text>{{ loadingMore ? '加载更多...' : '上拉加载更多' }}</text>
      </view>
      <view v-if="!hasMore && list.length > 0" class="load-more">
        <view class="load-more-line"></view>
        <text>已展示全部赛事</text>
        <view class="load-more-line"></view>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'

import { usePublicReadStore } from '@/store/publicRead.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { normalizeSaiXunCard } from '@/utils/saixun.js'
import {
  SAIXUN_DATE_PRESETS,
  buildCurrentYearFilter,
  buildCustomDateRange,
  buildEventNewsListParams,
  buildPresetDateRange,
  buildYearOptions,
  formatSaiXunFilterLabel
} from '@/utils/saixun-filter.js'
import {
  ASYNC_PAGE_STATUS,
  beginAsyncPageLoad,
  createAsyncPageState,
  getAsyncPageRequest,
  rejectAsyncPageLoad,
  resolveAsyncPageErrorFeedback,
  resolveAsyncPageLoad
} from '@/utils/async-page-state.js'
import { createRequestError } from '@/utils/request-errors.js'

const { isDarkMode } = usePageTheme()
const publicReadStore = usePublicReadStore()

const GAME_TYPE_OPTIONS = [
  { label: '全部球种', value: 0 },
  { label: '斯诺克', value: 1 },
  { label: '中式八球', value: 3 },
  { label: '中式九球', value: 2 }
]
const STATUS_OPTIONS = [
  { label: '全部状态', value: -1 },
  { label: '即将开始', value: 0 },
  { label: '进行中', value: 1 },
  { label: '已结束', value: 2 },
  { label: '已取消', value: 3 }
]

const loading = ref(true)
const loadingMore = ref(false)
const refreshing = ref(false)
const list = ref([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const hasMore = ref(false)
const hasLoadedOnce = ref(false)
const eventNewsState = ref(createAsyncPageState({ data: [] }))
const filter = ref({
  ...buildCurrentYearFilter(Date.now()),
  gameType: 0,
  status: -1
})
const customDateDraft = ref({
  from: filter.value.from,
  to: filter.value.to
})

const selectedGameType = computed(() => (
  GAME_TYPE_OPTIONS.find((item) => item.value === filter.value.gameType) || GAME_TYPE_OPTIONS[0]
))
const selectedStatus = computed(() => (
  STATUS_OPTIONS.find((item) => item.value === filter.value.status) || STATUS_OPTIONS[0]
))

const filterDateLabel = computed(() => formatSaiXunFilterLabel(filter.value))
const yearOptions = computed(() => buildYearOptions(filter.value.year))
const showCustomDateEditor = computed(() => filter.value.preset === 'custom')
const eventNewsPageError = computed(() => (
  eventNewsState.value.status === ASYNC_PAGE_STATUS.ERROR
    ? resolveAsyncPageErrorFeedback(eventNewsState.value.error, { resource: '赛讯' })
    : null
))
const eventNewsRefreshError = computed(() => (
  eventNewsState.value.refreshError
    ? resolveAsyncPageErrorFeedback(eventNewsState.value.refreshError, { resource: '赛讯' })
    : null
))

const fetchData = async ({ replace = false, force = false } = {}) => {
	if (eventNewsState.value.status === ASYNC_PAGE_STATUS.LOADING || eventNewsState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const requestedPage = page.value
	const nextState = beginAsyncPageLoad(eventNewsState.value, { emptyData: [] })
	const pageRequest = getAsyncPageRequest(nextState)
	eventNewsState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
  try {
    const listRes = await publicReadStore.loadEventNews(
      buildEventNewsListParams(filter.value, page.value, pageSize),
      { force }
    )
		if (eventNewsState.value.requestId !== pageRequest.requestId) return
    if (!listRes?.success) throw createRequestError({ message: listRes?.message || '加载赛讯失败', category: 'business' })
    const nextList = Array.isArray(listRes.list) ? listRes.list.map((item) => normalizeSaiXunCard(item)) : []
    list.value = replace ? nextList : [...list.value, ...nextList]
    total.value = Number(listRes.total || 0)
    hasMore.value = list.value.length < total.value
    hasLoadedOnce.value = true
		eventNewsState.value = resolveAsyncPageLoad(eventNewsState.value, pageRequest, { data: list.value })
  } catch (error) {
		if (eventNewsState.value.requestId !== pageRequest.requestId) return
		if (!replace && requestedPage > 1) page.value = requestedPage - 1
		eventNewsState.value = rejectAsyncPageLoad(eventNewsState.value, pageRequest, error)
  } finally {
		if (eventNewsState.value.requestId !== pageRequest.requestId) return
    loading.value = false
    loadingMore.value = false
    refreshing.value = false
  }
}

const applyFilter = async (nextFilter, { syncDraft = true } = {}) => {
  filter.value = {
    ...filter.value,
    ...nextFilter
  }
  if (syncDraft) {
    customDateDraft.value = {
      from: filter.value.from,
      to: filter.value.to
    }
  }
  page.value = 1
	list.value = []
	total.value = 0
	hasMore.value = false
	resetEventNewsList()
  await fetchData({ replace: true })
}

const refreshData = async ({ force = false } = {}) => {
  page.value = 1
  await fetchData({ replace: true, force })
}

const onRefresh = async () => {
	if (eventNewsState.value.status === ASYNC_PAGE_STATUS.LOADING || eventNewsState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
  refreshing.value = true
  await refreshData({ force: true })
}

const loadMore = async () => {
	if (eventNewsState.value.status === ASYNC_PAGE_STATUS.LOADING || eventNewsState.value.status === ASYNC_PAGE_STATUS.REFRESHING || loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  page.value += 1
  await fetchData({ replace: false })
}

const resetEventNewsList = () => {
	eventNewsState.value = {
		...eventNewsState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: eventNewsState.value.requestId + 1,
		data: [],
		hasData: false,
		error: null,
		refreshError: null
	}
}

const retryEventNews = () => refreshData({ force: true })

const handleEventNewsErrorAction = () => {
	if (eventNewsState.value.error?.category === 'permission' || eventNewsState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryEventNews()
}

const openDetail = (id) => {
  if (!id) return
  uni.navigateTo({ url: `/subPages/tournament/detail?id=${id}` })
}

const openYearSheet = () => {
  const options = yearOptions.value
  if (!options.length) return

  uni.showActionSheet({
    itemList: options.map((item) => `${item}年`),
    success: async (res) => {
      const selectedYear = options[res.tapIndex]
      if (!selectedYear || selectedYear === filter.value.year) return
      await applyFilter({
        year: selectedYear,
        ...buildPresetDateRange(selectedYear, 'full_year')
      })
    }
  })
}

const openDatePresetSheet = () => {
  uni.showActionSheet({
    itemList: SAIXUN_DATE_PRESETS.map((item) => item.label),
    success: async (res) => {
      const selected = SAIXUN_DATE_PRESETS[res.tapIndex]
      if (!selected) return

      if (selected.key === 'custom') {
        filter.value = {
          ...filter.value,
          preset: 'custom'
        }
        customDateDraft.value = {
          from: filter.value.from,
          to: filter.value.to
        }
        return
      }

      await applyFilter({
        year: filter.value.year,
        ...buildPresetDateRange(filter.value.year, selected.key)
      })
    }
  })
}

const openFilterSheet = () => {
  showFilterSheet.value = true
  uni.hideTabBar()
}

const closeFilterSheet = () => {
  showFilterSheet.value = false
  uni.showTabBar()
}

const onGameTypeChange = (e) => {
  const next = GAME_TYPE_OPTIONS[e.detail.value]
  if (!next || next.value === filter.value.gameType) {
    closeFilterSheet()
    return
  }
  applyFilter({ gameType: next.value }).then(() => closeFilterSheet())
}

const onStatusChange = (e) => {
  const next = STATUS_OPTIONS[e.detail.value]
  if (!next || next.value === filter.value.status) {
    closeFilterSheet()
    return
  }
  applyFilter({ status: next.value }).then(() => closeFilterSheet())
}

const onCustomFromChange = async (event) => {
  const nextFrom = event?.detail?.value || customDateDraft.value.from
  const nextFilter = buildCustomDateRange(filter.value.year, nextFrom, customDateDraft.value.to)
  customDateDraft.value = {
    from: nextFilter.from,
    to: nextFilter.to
  }
  await applyFilter(nextFilter, { syncDraft: true })
}

const onCustomToChange = async (event) => {
  const nextTo = event?.detail?.value || customDateDraft.value.to
  const nextFilter = buildCustomDateRange(filter.value.year, customDateDraft.value.from, nextTo)
  customDateDraft.value = {
    from: nextFilter.from,
    to: nextFilter.to
  }
  await applyFilter(nextFilter, { syncDraft: true })
}

onShow(() => {
  if (!hasLoadedOnce.value) refreshData()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
