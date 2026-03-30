<template>
  <view class="opponent-record-container" :class="{ 'dark-mode': isDarkMode }">
    <view class="login-required" v-if="needLogin">
      <uni-icons type="locked" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
      <text class="hint-text">{{ viewModel.loginHint }}</text>
      <button class="login-btn" @click="goLogin">去登录</button>
    </view>

    <template v-else>
      <view class="search-section">
        <view class="search-box">
          <uni-icons type="search" size="20" color="#9ca3af"></uni-icons>
          <input
            v-model="searchKeyword"
            type="text"
            class="search-input"
            :placeholder="viewModel.searchPlaceholder"
            @confirm="handleSearch"
          />
        </view>
      </view>

      <view class="stats-card">
        <view class="stats-copy">
          <text class="stats-title">{{ statsSummary.title }}</text>
          <text class="stats-subtitle">{{ statsSummary.subtitle }}</text>
        </view>

        <view class="stats-content">
          <view class="stat-item">
            <text class="stat-label">总对手数</text>
            <text class="stat-value">{{ statsData.totalOpponents }}</text>
          </view>
          <view class="stat-divider"></view>
          <view class="stat-item">
            <text class="stat-label">总胜场</text>
            <text class="stat-value">{{ statsData.totalWins }}</text>
          </view>
        </view>
      </view>

      <scroll-view
        class="opponent-list"
        scroll-y
        :refresher-enabled="true"
        :refresher-triggered="isRefreshing"
        @refresherrefresh="onRefresh"
        @scrolltolower="onLoadMore"
      >
        <view class="loading-wrapper" v-if="isLoading && cardViewModels.length === 0">
          <uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
          <text class="loading-text">加载中...</text>
        </view>

        <view class="error-wrapper" v-else-if="loadFailed && cardViewModels.length === 0">
          <uni-icons type="info-filled" size="52" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
          <text class="error-text">{{ loadErrorMessage || '对手记录加载失败' }}</text>
          <button class="retry-btn" @click="fetchOpponentList">
            <text>重新加载</text>
          </button>
        </view>

        <view class="empty-wrapper" v-else-if="!isLoading && cardViewModels.length === 0">
          <uni-icons type="contact" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
          <text class="empty-text">{{ emptyState.text }}</text>
          <text class="empty-hint">{{ emptyState.hint }}</text>
        </view>

        <view class="opponent-items" v-else>
          <view
            v-for="opponent in cardViewModels"
            :key="opponent.id || opponent.name"
            class="opponent-item"
            :class="opponent.toneClass"
            @click="handleOpponentDetail(opponent)"
          >
            <view class="opponent-left">
              <view class="avatar-wrapper">
                <image
                  v-if="opponent.avatar"
                  class="avatar"
                  :src="opponent.avatar"
                  mode="aspectFill"
                />
                <view v-else class="avatar-placeholder">
                  <text class="avatar-text">{{ getAvatarText(opponent.name) }}</text>
                </view>
              </view>

              <view class="opponent-info">
                <view class="name-row">
                  <text class="opponent-name">{{ opponent.name }}</text>
                  <view class="relationship-badge" :class="opponent.toneClass">
                    <text>{{ opponent.relationshipBadge }}</text>
                  </view>
                </view>
                <text class="opponent-summary">{{ opponent.relationshipText }}</text>
                <view class="meta-row">
                  <text class="meta-chip">{{ opponent.recordText }}</text>
                  <text class="meta-chip">{{ opponent.lastMatchText }}</text>
                </view>
              </view>
            </view>

            <view class="opponent-right">
              <text class="win-rate" :class="opponent.toneClass">{{ opponent.winRateText }}</text>
              <text class="win-loss">{{ opponent.sampleText }}</text>
              <text class="detail-link">查看交锋</text>
            </view>
          </view>
        </view>

        <view class="load-more" v-if="cardViewModels.length > 0">
          <text v-if="isLoadingMore" class="load-more-text">加载中...</text>
          <text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
        </view>
      </scroll-view>
    </template>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { getOpponentList } from '@/api/match.js'
import {
  buildOpponentCardViewModels,
  buildOpponentH2HUrl,
  buildOpponentRecordRequestParams,
  buildOpponentRecordViewModel,
  buildOpponentStatsSummary,
  normalizeOpponentRecordOptions
} from '@/utils/opponent-record.js'

const themeStore = useThemeStore()

const isDarkMode = computed(() => themeStore.isDarkMode)
const isLoading = ref(false)
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const loadFailed = ref(false)
const loadErrorMessage = ref('')
const searchKeyword = ref('')
const needLogin = ref(false)
const targetUserId = ref(0)
const targetName = ref('')
const targetAvatar = ref('')

const opponentList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

const statsData = reactive({
  totalOpponents: 0,
  totalWins: 0
})

const viewModel = computed(() => buildOpponentRecordViewModel({
  targetName: targetName.value,
  targetUserId: targetUserId.value
}))

const statsSummary = computed(() => buildOpponentStatsSummary({
  totalOpponents: statsData.totalOpponents,
  totalWins: statsData.totalWins,
  targetName: targetName.value,
  targetUserId: targetUserId.value
}))

const cardViewModels = computed(() => buildOpponentCardViewModels({
  opponents: opponentList.value,
  subjectName: viewModel.value.subjectName
}))

const emptyState = computed(() => {
  if (searchKeyword.value.trim()) {
    return {
      text: '没有找到匹配的对手',
      hint: '换个昵称试试，或者下拉刷新后再看'
    }
  }

  return {
    text: viewModel.value.emptyText,
    hint: viewModel.value.emptyHint
  }
})

onLoad((options) => {
  const normalized = normalizeOpponentRecordOptions(options)
  targetUserId.value = normalized.targetUserId
  targetName.value = normalized.targetName
  targetAvatar.value = normalized.targetAvatar

  uni.setNavigationBarTitle({
    title: buildOpponentRecordViewModel({
      targetName: normalized.targetName,
      targetUserId: normalized.targetUserId
    }).navigationTitle
  })
})

onShow(() => {
  themeStore.syncTheme()
  themeStore.applyNavigationBarTheme()

  const token = uni.getStorageSync('token')
  if (!token) {
    needLogin.value = true
    return
  }

  needLogin.value = false
  if (opponentList.value.length === 0 || loadFailed.value) {
    fetchOpponentList()
  }
})

const fetchOpponentList = async (isRefresh = false, isLoadMore = false) => {
  if (isLoading.value || isLoadingMore.value) return

  if (isRefresh) {
    isRefreshing.value = true
    currentPage.value = 1
    hasMore.value = true
  } else if (isLoadMore) {
    if (!hasMore.value) return
    isLoadingMore.value = true
    currentPage.value += 1
  } else {
    isLoading.value = true
  }

  loadFailed.value = false
  loadErrorMessage.value = ''

  try {
    const res = await getOpponentList(buildOpponentRecordRequestParams({
      page: currentPage.value,
      pageSize,
      keyword: searchKeyword.value.trim(),
      targetUserId: targetUserId.value
    }))

    const list = Array.isArray(res.list) ? res.list : []
    total.value = Number(res.total || 0)
    statsData.totalOpponents = Number(res.total_opponents || 0)
    statsData.totalWins = Number(res.total_wins || 0)

    if (isRefresh) {
      opponentList.value = list
    } else if (isLoadMore) {
      opponentList.value = [...opponentList.value, ...list]
    } else {
      opponentList.value = list
    }

    hasMore.value = opponentList.value.length < total.value
  } catch (error) {
    if (isLoadMore) {
      currentPage.value = Math.max(currentPage.value - 1, 1)
    }
    loadFailed.value = opponentList.value.length === 0
    loadErrorMessage.value = error.message || '获取对手记录失败'
    uni.showToast({
      title: loadErrorMessage.value,
      icon: 'none'
    })
  } finally {
    isLoading.value = false
    isRefreshing.value = false
    isLoadingMore.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  hasMore.value = true
  opponentList.value = []
  fetchOpponentList()
}

const onRefresh = () => {
  fetchOpponentList(true, false)
}

const onLoadMore = () => {
  fetchOpponentList(false, true)
}

const getAvatarText = (name) => {
  if (!name) return '?'
  return name.substring(0, 2).toUpperCase()
}

const handleOpponentDetail = (opponent) => {
  uni.navigateTo({
    url: buildOpponentH2HUrl({
      opponent,
      targetUserId: targetUserId.value,
      targetName: targetName.value,
      targetAvatar: targetAvatar.value
    })
  })
}

const goLogin = () => {
  uni.navigateTo({
    url: '/pages/login/login'
  })
}
</script>

<style lang="scss" scoped>
@import './opponentRecord.scss';
</style>
