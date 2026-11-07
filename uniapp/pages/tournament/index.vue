<template>
  <view class="tournament-page" :class="{ 'dark-mode': isDarkMode }">
    <view class="list-header">
      <view class="status-tabs">
        <view
          v-for="tab in STATUS_TABS"
          :key="tab.value"
          class="status-tab"
          :class="{ 'status-tab--active': filter.status === tab.value }"
          @tap="selectStatus(tab.value)"
        >
          <text class="status-tab-text">{{ tab.label }}</text>
        </view>
      </view>
      <view class="filter-row">
        <view class="filter-trigger" @tap="openYearSheet">
          <text class="filter-trigger-text">{{ filter.year }}年</text>
          <uni-icons type="bottom" size="12" :color="mutedIconColor"></uni-icons>
        </view>
        <view class="filter-trigger" @tap="openGameTypeSheet">
          <text class="filter-trigger-text" :class="{ 'filter-trigger-text--active': filter.gameType !== 0 }">{{ selectedGameType.label }}</text>
          <uni-icons type="bottom" size="12" :color="mutedIconColor"></uni-icons>
        </view>
        <text v-if="total > 0" class="filter-count">共 {{ total }} 项赛事</text>
      </view>
    </view>

    <scroll-view
      scroll-y
      class="feed-scroll"
      refresher-enabled
      :refresher-triggered="refreshing"
      @refresherrefresh="onRefresh"
      @scrolltolower="loadMore"
    >
      <view v-if="loading" class="skeleton-list">
        <view v-for="index in 3" :key="index" class="skeleton-card">
          <view class="skeleton-line skeleton-line--title"></view>
          <view class="skeleton-line"></view>
          <view class="skeleton-line skeleton-line--short"></view>
        </view>
      </view>

      <view v-else-if="list.length > 0" class="feed">
        <view v-if="eventNewsRefreshError" class="refresh-error-banner">
          <text>{{ eventNewsRefreshError.description }}</text>
          <text class="refresh-error-action" @tap="retryEventNews">重试</text>
        </view>

        <view v-for="section in sections" :key="section.key" class="feed-section">
          <view class="feed-section-head">
            <text class="feed-section-title">{{ section.title }}</text>
            <text v-if="section.caption" class="feed-section-caption">{{ section.caption }}</text>
          </view>

          <view v-for="item in section.items" :key="item.id" class="feed-item">
            <view v-if="item.isFeatured" class="feature-card" @tap="openDetail(item.id)">
              <view class="feature-media">
                <image v-if="item.hasCover" class="feature-cover" :src="item.coverImage" mode="aspectFill"></image>
                <view class="feature-scrim"></view>
                <view class="feature-badge" :class="'feature-badge--' + item.statusTone">
                  <view v-if="item.statusTone === 'live'" class="status-dot"></view>
                  <text>{{ item.statusLabel }}</text>
                </view>
                <view class="feature-copy">
                  <text class="feature-title">{{ item.title }}</text>
                  <text class="feature-meta">{{ item.metaText }}</text>
                </view>
              </view>
              <view class="feature-strip">
                <text v-if="item.highlight && item.highlight.label" class="feature-strip-label">{{ item.highlight.label }}</text>
                <text class="feature-strip-text">{{ item.highlight ? item.highlight.primary : item.matchCountText }}</text>
                <uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
            </view>

            <view v-else class="event-card" @tap="openDetail(item.id)">
              <view class="event-main">
                <view class="event-copy">
                  <text class="event-title">{{ item.title }}</text>
                  <text class="event-meta">{{ item.metaText }}</text>
                </view>
                <view v-if="item.statusTone === 'canceled'" class="status-badge status-badge--canceled">
                  <text>{{ item.statusLabel }}</text>
                </view>
                <uni-icons v-else type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
              <view v-if="item.highlight" class="champion-row">
                <view class="champion-tag">
                  <uni-icons type="medal-filled" size="13" :color="brandStrongColor"></uni-icons>
                  <text class="champion-tag-text">{{ item.highlight.label }}</text>
                </view>
                <text class="champion-name">{{ item.highlight.primary }}</text>
                <text class="champion-detail">{{ item.highlight.secondary }}</text>
              </view>
            </view>
          </view>
        </view>

        <view v-if="hasMore" class="load-more">
          <text>{{ loadingMore ? '正在加载...' : '上拉加载更多' }}</text>
        </view>
        <view v-else class="load-more">
          <view class="load-more-line"></view>
          <text>已展示全部赛事</text>
          <view class="load-more-line"></view>
        </view>
      </view>

      <view v-else-if="eventNewsPageError" class="state-block">
        <view class="state-icon">
          <uni-icons type="info" size="30" :color="mutedIconColor"></uni-icons>
        </view>
        <text class="state-title">{{ eventNewsPageError.title }}</text>
        <text class="state-text">{{ eventNewsPageError.description }}</text>
        <button class="state-btn" @tap="handleEventNewsErrorAction">{{ eventNewsPageError.actionText }}</button>
      </view>

      <view v-else-if="eventNewsState.status === ASYNC_PAGE_STATUS.EMPTY" class="state-block">
        <view class="state-icon">
          <uni-icons type="calendar" size="30" :color="brandStrongColor"></uni-icons>
        </view>
        <text class="state-title">{{ emptyCopy.title }}</text>
        <text class="state-text">{{ emptyCopy.desc }}</text>
        <button v-if="hasNarrowFilter" class="state-btn" @tap="resetNarrowFilter">查看全部赛事</button>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'

import { usePublicReadStore } from '@/store/publicRead.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useMainTabBar } from '@/utils/use-main-tab-bar.js'
import { buildSaiXunListSections, normalizeSaiXunCard } from '@/utils/saixun.js'
import {
  buildCurrentYearFilter,
  buildEventNewsListParams,
  buildYearDateRange,
  buildYearOptions
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
useMainTabBar('tournament')
const publicReadStore = usePublicReadStore()

const STATUS_TABS = [
  { label: '全部', value: -1 },
  { label: '进行中', value: 1 },
  { label: '即将开始', value: 0 },
  { label: '已结束', value: 2 }
]
const GAME_TYPE_OPTIONS = [
  { label: '全部球种', value: 0 },
  { label: '斯诺克', value: 1 },
  { label: '中式八球', value: 3 },
  { label: '中式九球', value: 2 }
]
const EMPTY_COPY = {
  [-1]: { title: '这一年还没有赛讯', desc: '换个年份看看，或稍后下拉刷新。' },
  1: { title: '当前没有进行中的赛事', desc: '可以先看看即将开始的赛事。' },
  0: { title: '暂无即将开始的赛事', desc: '新赛程公布后会第一时间更新。' },
  2: { title: '暂无已结束的赛事', desc: '换个年份或球种看看。' }
}

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

const mutedIconColor = computed(() => (isDarkMode.value ? '#9F926E' : '#9A8C67'))
const brandStrongColor = computed(() => (isDarkMode.value ? '#F0C542' : '#C69200'))
const selectedGameType = computed(() => (
  GAME_TYPE_OPTIONS.find((item) => item.value === filter.value.gameType) || GAME_TYPE_OPTIONS[0]
))
const sections = computed(() => buildSaiXunListSections(list.value))
const hasNarrowFilter = computed(() => filter.value.status !== -1 || filter.value.gameType !== 0)
const emptyCopy = computed(() => EMPTY_COPY[filter.value.status] || EMPTY_COPY[-1])
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

const applyFilter = async (nextFilter) => {
  filter.value = {
    ...filter.value,
    ...nextFilter
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

const selectStatus = (status) => {
  if (status === filter.value.status) return
  applyFilter({ status })
}

const resetNarrowFilter = () => {
  applyFilter({ status: -1, gameType: 0 })
}

const openYearSheet = () => {
  const options = buildYearOptions(filter.value.year)
  if (!options.length) return

  uni.showActionSheet({
    itemList: options.map((item) => `${item}年`),
    success: (res) => {
      const selectedYear = options[res.tapIndex]
      if (!selectedYear || selectedYear === filter.value.year) return
      applyFilter({
        year: selectedYear,
        ...buildYearDateRange(selectedYear)
      })
    }
  })
}

const openGameTypeSheet = () => {
  uni.showActionSheet({
    itemList: GAME_TYPE_OPTIONS.map((item) => item.label),
    success: (res) => {
      const next = GAME_TYPE_OPTIONS[res.tapIndex]
      if (!next || next.value === filter.value.gameType) return
      applyFilter({ gameType: next.value })
    }
  })
}

onShow(() => {
  if (!hasLoadedOnce.value) refreshData()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
