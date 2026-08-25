<template>
  <view class="opponent-record-container" :class="{ 'dark-mode': isDarkMode }">
    <view class="login-required" v-if="needLogin">
      <uni-icons type="locked" size="64" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
      <text class="hint-text">{{ viewModel.loginHint }}</text>
      <button class="login-btn" @click="goLogin">去登录</button>
    </view>

    <template v-else>
      <view class="search-section">
        <view class="search-box">
          <uni-icons type="search" size="20" color="#9A8C67"></uni-icons>
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
          <uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
          <text class="loading-text">加载中...</text>
        </view>

        <view class="error-wrapper" v-else-if="opponentPageError">
          <uni-icons type="info-filled" size="52" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
          <text class="error-text">{{ opponentPageError.title }}</text>
          <text class="error-desc">{{ opponentPageError.description }}</text>
          <button class="retry-btn" @click="handleOpponentErrorAction">
            <text>{{ opponentPageError.actionText }}</text>
          </button>
        </view>

        <view class="empty-wrapper" v-else-if="opponentState.status === ASYNC_PAGE_STATUS.EMPTY">
          <uni-icons type="contact" size="64" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
          <text class="empty-text">{{ emptyState.text }}</text>
          <text class="empty-hint">{{ emptyState.hint }}</text>
        </view>

        <view class="opponent-items" v-else>
          <view v-if="opponentRefreshError" class="refresh-error-banner">
            <text>{{ opponentRefreshError.description }}</text>
            <text class="refresh-error-action" @tap="retryOpponentList">重试</text>
          </view>
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
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { getOpponentList } from '@/api/match.js'
import {
  buildOpponentCardViewModels,
  buildOpponentH2HUrl,
  buildOpponentRecordRequestParams,
  buildOpponentRecordViewModel,
  buildOpponentStatsSummary,
  normalizeOpponentRecordOptions
} from '@/utils/opponent-record.js'
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
const userStore = useUserStore()
const userDataInvalidationStore = useUserDataInvalidationStore()

const currentReadIdentity = () => ({
  userId: userStore.userId,
  authGeneration: userStore.authGeneration
})

const currentIdentityKey = () => {
  const identity = currentReadIdentity()
  return `${identity.userId}:${identity.authGeneration}`
}

const isLoading = ref(false)
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const loadFailed = ref(false)
const loadErrorMessage = ref('')
const opponentState = ref(createAsyncPageState({
  authGeneration: userStore.authGeneration,
  data: []
}))
const searchKeyword = ref('')
const needLogin = ref(false)
const targetUserId = ref(0)
const targetName = ref('')
const targetAvatar = ref('')

const opponentList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const latestRequestId = ref(0)
const loadedIdentityKey = ref('')
const loadedOpponentScopeVersion = ref(0)
const opponentPageError = computed(() => (
  opponentState.value.status === ASYNC_PAGE_STATUS.ERROR
    ? resolveAsyncPageErrorFeedback(opponentState.value.error, { resource: '过往对手' })
    : null
))
const opponentRefreshError = computed(() => (
  opponentState.value.refreshError
    ? resolveAsyncPageErrorFeedback(opponentState.value.refreshError, { resource: '过往对手' })
    : null
))

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

  if (!userStore.isLoggedIn || !userStore.userId) {
    needLogin.value = true
    return
  }

  needLogin.value = false
  const identityKey = currentIdentityKey()
  const scopeVersion = userDataInvalidationStore.versionOf('opponents')
  if (loadedIdentityKey.value !== identityKey) {
    opponentList.value = []
    currentPage.value = 1
    total.value = 0
    statsData.totalOpponents = 0
    statsData.totalWins = 0
    isLoading.value = false
    isLoadingMore.value = false
    opponentState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
  }
  if (opponentList.value.length === 0 || opponentState.value.status === ASYNC_PAGE_STATUS.ERROR || loadedOpponentScopeVersion.value !== scopeVersion) {
    fetchOpponentList()
  }
})

const fetchOpponentList = async (isRefresh = false, isLoadMore = false) => {
  if (isLoading.value || isLoadingMore.value || (isLoadMore && !hasMore.value)) return
  const requestId = latestRequestId.value + 1
  const requestIdentityKey = currentIdentityKey()
  const requestScopeVersion = userDataInvalidationStore.versionOf('opponents')
  const nextState = beginAsyncPageLoad(opponentState.value, {
    authGeneration: userStore.authGeneration,
    emptyData: []
  })
  const pageRequest = getAsyncPageRequest(nextState)
  latestRequestId.value = requestId
  opponentState.value = nextState

  if (isRefresh) {
    isRefreshing.value = true
    currentPage.value = 1
    hasMore.value = true
  } else if (isLoadMore) {
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

    if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey || opponentState.value.requestId !== pageRequest.requestId || opponentState.value.authGeneration !== pageRequest.authGeneration) return
    if (!res?.success) throw createRequestError({ message: res?.message || '获取过往对手失败', category: 'business' })
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
    loadedIdentityKey.value = requestIdentityKey
    loadedOpponentScopeVersion.value = requestScopeVersion
    opponentState.value = resolveAsyncPageLoad(opponentState.value, pageRequest, { data: opponentList.value })
  } catch (error) {
    if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
    if (isLoadMore) {
      currentPage.value = Math.max(currentPage.value - 1, 1)
    }
    opponentState.value = rejectAsyncPageLoad(opponentState.value, pageRequest, error)
    loadFailed.value = opponentState.value.status === ASYNC_PAGE_STATUS.ERROR
    loadErrorMessage.value = error.message || '获取过往对手失败'
  } finally {
    if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
    isLoading.value = false
    isRefreshing.value = false
    isLoadingMore.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  hasMore.value = true
  opponentList.value = []
  opponentState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
  fetchOpponentList()
}

const onRefresh = () => {
  fetchOpponentList(true, false)
}

const onLoadMore = () => {
  fetchOpponentList(false, true)
}

const retryOpponentList = () => fetchOpponentList(true, false)

const handleOpponentErrorAction = () => {
  if (opponentState.value.error?.category === 'permission' || opponentState.value.error?.category === 'not-found') {
    uni.navigateBack({ delta: 1 })
    return
  }
  retryOpponentList()
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
