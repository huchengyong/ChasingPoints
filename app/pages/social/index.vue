<template>
  <view class="social-page" :class="{ 'dark-mode': isDarkMode }">
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
      </view>
      <view class="filter-chip" @tap="openDatePresetSheet">
        <text class="filter-chip-label">日期</text>
        <text class="filter-chip-value">{{ filterDateLabel }}</text>
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
        <view v-for="item in list" :key="item.id" class="post-card" @tap="openDetail(item.id)">
          <image class="post-cover" :src="item.coverImage" mode="aspectFill"></image>
          <view class="post-overlay"></view>
          <view class="post-body">
            <view class="post-topline">
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

      <view v-else class="state-block">
        <text class="state-icon">🗓️</text>
        <text class="state-title">暂时还没有可看的赛讯</text>
        <text class="state-text">官方赛事内容更新中，稍后再来刷新看看。</text>
      </view>

      <view v-if="hasMore && list.length > 0" class="load-more">
        <text>{{ loadingMore ? '加载更多...' : '上拉加载更多' }}</text>
      </view>
      <view v-if="!hasMore && list.length > 0" class="load-more">
        <text>没有更多内容了</text>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed, onUnmounted, ref } from 'vue'
import { onHide, onShow } from '@dcloudio/uni-app'

import { getEventNewsList } from '@/api/event-news.js'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
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

const themeStore = useThemeStore()

const isDarkMode = computed(() => themeStore.isDarkMode)
const loading = ref(true)
const loadingMore = ref(false)
const refreshing = ref(false)
const list = ref([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const hasMore = ref(false)
const shouldRefreshOnShow = ref(true)
const filter = ref(buildCurrentYearFilter(Date.now()))
const customDateDraft = ref({
  from: filter.value.from,
  to: filter.value.to
})

const filterDateLabel = computed(() => formatSaiXunFilterLabel(filter.value))
const yearOptions = computed(() => buildYearOptions(filter.value.year))
const showCustomDateEditor = computed(() => filter.value.preset === 'custom')

const fetchData = async ({ replace = false } = {}) => {
  try {
    const listRes = await getEventNewsList(buildEventNewsListParams(filter.value, page.value, pageSize))
      .catch(() => ({ success: false, list: [], total: 0 }))
    const nextList = Array.isArray(listRes.list) ? listRes.list.map((item) => normalizeSaiXunCard(item)) : []
    list.value = replace ? nextList : [...list.value, ...nextList]
    total.value = Number(listRes.total || 0)
    hasMore.value = list.value.length < total.value
  } finally {
    loading.value = false
    loadingMore.value = false
    refreshing.value = false
  }
}

const applyFilter = async (nextFilter, { syncDraft = true } = {}) => {
  filter.value = {
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
  loading.value = true
  await fetchData({ replace: true })
}

const refreshData = async () => {
  page.value = 1
  loading.value = true
  await fetchData({ replace: true })
}

const onRefresh = async () => {
  if (loading.value) return
  refreshing.value = true
  await refreshData()
}

const loadMore = async () => {
  if (loading.value || loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  page.value += 1
  await fetchData({ replace: false })
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

const handleThemeChange = () => {
  themeStore.applyNavigationBarTheme()
}

onShow(() => {
  themeStore.syncTheme()
  themeStore.applyNavigationBarTheme()
  uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
  uni.$on(THEME_CHANGE_EVENT, handleThemeChange)

  if (!shouldRefreshOnShow.value) {
    shouldRefreshOnShow.value = true
    return
  }

  // 合规收口：原动态 tab 现只保留官方赛讯展示，用户发布和互动入口暂不开放。
  refreshData()
})

onHide(() => {
  uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

onUnmounted(() => {
  uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
