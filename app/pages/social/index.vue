<template>
  <view class="social-page" :class="{ 'dark-mode': isDarkMode }">
    <view class="hero-card">
      <view class="hero-copy">
        <text class="hero-eyebrow">赛讯</text>
        <text class="hero-title">官方赛事信息集中看</text>
        <text class="hero-desc">第一时间掌握各大赛事的赛程与赛况</text>
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
      <view v-if="loading" class="state-block">
        <uni-icons type="spinner-cycle" size="34" color="#E0AE12"></uni-icons>
        <text class="state-text">加载中...</text>
      </view>

      <view v-else-if="list.length > 0" class="post-list">
        <view v-for="item in list" :key="item.id" class="post-card" @tap="openDetail(item.id)">
          <image class="post-cover" :src="item.coverImage" mode="aspectFill"></image>
          <view class="post-topline">
            <view class="post-chip">
              <text>{{ item.gameTypeText }}</text>
            </view>
            <view class="post-status" :class="'status-' + item.status">
              <text>{{ item.statusText }}</text>
            </view>
          </view>
          <text class="post-title">{{ item.title }}</text>
          <text class="post-content">{{ item.summary }}</text>
          <view class="meta-grid">
            <view class="meta-item">
              <text class="meta-label">日期</text>
              <text class="meta-value">{{ item.dateText }}</text>
            </view>
            <view v-if="item.showTime" class="meta-item">
              <text class="meta-label">时间</text>
              <text class="meta-value">{{ item.timeText }}</text>
            </view>
            <view v-if="item.locationText" class="meta-item">
              <text class="meta-label">地点</text>
              <text class="meta-value">{{ item.locationText }}</text>
            </view>
            <view class="meta-item">
              <text class="meta-label">当前轮次</text>
              <text class="meta-value">{{ item.currentRoundText }}</text>
            </view>
            <view class="meta-item">
              <text class="meta-label">比赛数</text>
              <text class="meta-value">{{ item.matchCountText }}</text>
            </view>
          </view>
          <view class="post-footer">
            <text>{{ item.sourceText || '追分官方' }}</text>
            <text>查看详情</text>
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
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'

import { getEventNewsList } from '@/api/event-news.js'
import { useThemeStore } from '@/store/theme.js'
import { normalizeSaiXunCard } from '@/utils/saixun.js'

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

const fetchData = async ({ replace = false } = {}) => {
  try {
    const listRes = await getEventNewsList({
      page: page.value,
      page_size: pageSize
    }).catch(() => ({ success: false, list: [], total: 0 }))
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

onShow(() => {
  if (!shouldRefreshOnShow.value) {
    shouldRefreshOnShow.value = true
    return
  }

  // 合规收口：原动态 tab 现只保留官方赛讯展示，用户发布和互动入口暂不开放。
  refreshData()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
