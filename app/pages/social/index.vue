<template>
  <view class="social-page" :class="{ 'dark-mode': isDarkMode }">
    <view class="social-tabs">
      <view
        v-for="item in tabs"
        :key="item.value"
        class="tab-item"
        :class="{ active: activeTab === item.value }"
        @tap="switchTab(item.value)"
      >
        <text>{{ item.label }}</text>
      </view>
    </view>

    <view class="quick-actions">
      <view class="action-card primary" @tap="goCreatePost">
        <text class="action-icon">📝</text>
        <text class="action-title">发动态</text>
        <text class="action-desc">分享台球时刻</text>
      </view>
      <view class="action-card" @tap="goGeneratePkReport">
        <text class="action-icon">📊</text>
        <text class="action-title">生成PK报表</text>
        <text class="action-desc">选择好友生成对比卡</text>
      </view>
      <view class="action-card" @tap="goPkRecords">
        <text class="action-icon">⚔️</text>
        <text class="action-title">PK记录</text>
        <text class="action-desc">查看收到和发出的邀约</text>
      </view>
    </view>

    <view class="tool-row">
      <view class="tool-chip" @tap="goFriendList">
        <uni-icons type="staff-filled" size="16" color="#E0AE12"></uni-icons>
        <text>好友列表</text>
      </view>
      <view class="tool-chip" @tap="goFriendRequests">
        <uni-icons type="person-filled" size="16" color="#E0AE12"></uni-icons>
        <text>好友请求</text>
        <view v-if="friendRequestStore.pendingCount > 0" class="tool-badge">
          <text>{{ friendRequestStore.pendingCount > 99 ? '99+' : friendRequestStore.pendingCount }}</text>
        </view>
      </view>
    </view>

    <view class="feed-section-head">
      <view class="feed-section-copy">
        <text class="feed-section-title">{{ currentTabMeta.title }}</text>
        <text class="feed-section-desc">{{ currentTabMeta.desc }}</text>
      </view>
      <view class="feed-section-link" @tap="openCurrentFeed()">
        <text>{{ currentTabMeta.cta }}</text>
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

      <view v-else-if="visiblePosts.length > 0" class="post-list">
        <view v-for="item in visiblePosts" :key="item.id" class="post-card" @tap="handlePreviewCardTap(item)">
          <view class="post-header">
            <image
              class="post-avatar"
              :src="item.avatar || '/static/images/default-avatar.png'"
              mode="aspectFill"
            />
            <view class="post-user">
              <view class="post-name-row">
                <text class="post-name">{{ item.nickname || '球友' }}</text>
                <view class="post-tag" :class="item.tagClass">
                  <text>{{ item.tagText }}</text>
                </view>
              </view>
              <text class="post-time">{{ item.relativeTime }}</text>
            </view>
          </view>

          <text class="post-content">{{ item.content || '暂无内容' }}</text>

          <view v-if="item.images.length > 0" class="image-grid">
            <image
              v-for="(img, idx) in item.images.slice(0, 3)"
              :key="`${item.id}-${idx}`"
              class="post-image"
              :src="img"
              mode="aspectFill"
              @tap.stop="previewImage(item.images, idx)"
            />
          </view>

          <view class="post-footer">
            <text>❤️ {{ item.likes_count || 0 }}</text>
            <text>💬 {{ item.comments_count || 0 }}</text>
          </view>

          <view v-if="item.post_type === 1" class="report-actions">
            <view class="report-btn" @tap.stop="openPkReport(item)">
              <text>查看PK报表</text>
            </view>
          </view>
        </view>
      </view>

      <view v-else class="state-block">
        <text class="state-icon">{{ emptyState.icon }}</text>
        <text class="state-title">{{ emptyState.title }}</text>
        <text class="state-text">{{ emptyState.desc }}</text>
        <view v-if="emptyState.ctaText" class="state-cta" @tap="handleEmptyStateCta">
          <text>{{ emptyState.ctaText }}</text>
        </view>
      </view>

      <view v-if="showLoadMore" class="load-more">
        <text>{{ loadingMore ? '加载更多...' : '上拉加载更多' }}</text>
      </view>
      <view v-if="showNoMore" class="load-more">
        <text>没有更多内容了</text>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onHide, onShow } from '@dcloudio/uni-app'
import { getPostList, getPublicPosts } from '@/api/social.js'
import { useThemeStore } from '@/store/theme.js'
import { useUserStore } from '@/store/user.js'
import { useNotificationStore } from '@/store/notification.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { formatRelativeTime } from '@/utils/format.js'
import { buildFeedUrl, filterReportPosts, resolveSocialEmptyState, SOCIAL_TABS } from '@/utils/social-entry.js'
import { userWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'

const tabs = SOCIAL_TABS

const themeStore = useThemeStore()
const userStore = useUserStore()
const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()

const isDarkMode = computed(() => themeStore.isDarkMode)
const activeTab = ref('recommend')
const loading = ref(true)
const loadingMore = ref(false)
const refreshing = ref(false)
const publicPosts = ref([])
const followingPosts = ref([])
const publicPage = ref(1)
const followingPage = ref(1)
const pageSize = 10
const publicHasMore = ref(false)
const followingHasMore = ref(false)
const shouldRefreshOnShow = ref(true)

const visiblePosts = computed(() => {
  if (activeTab.value === 'friends') {
    return followingPosts.value
  }
  if (activeTab.value === 'reports') {
    return filterReportPosts([...publicPosts.value, ...followingPosts.value])
      .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0))
  }
  return publicPosts.value
})

const emptyState = computed(() => resolveSocialEmptyState({
  tab: activeTab.value,
  isLoggedIn: userStore.isLoggedIn
}))

const currentTabMeta = computed(() => {
  if (activeTab.value === 'friends') {
    return {
      title: '好友动态预览',
      desc: userStore.isLoggedIn ? '先看几条近况，想继续刷就进入完整好友流。' : '登录后可查看完整好友流与真实战绩分享。',
      cta: userStore.isLoggedIn ? '查看好友动态' : '登录后查看'
    }
  }
  if (activeTab.value === 'reports') {
    return {
      title: '战报预览',
      desc: '这里先看精选战报，完整浏览与互动统一进入战报流。',
      cta: '查看全部战报'
    }
  }
  return {
    title: '推荐动态预览',
    desc: '先快速感知社区氛围，继续浏览时进入完整推荐流。',
    cta: '查看全部推荐'
  }
})

const showLoadMore = computed(() => {
  if (activeTab.value === 'friends') {
    return followingHasMore.value
  }
  return activeTab.value === 'recommend' && publicHasMore.value
})

const showNoMore = computed(() => {
  const list = visiblePosts.value
  if (!list.length) return false
  if (activeTab.value === 'friends') {
    return !followingHasMore.value
  }
  if (activeTab.value === 'recommend') {
    return !publicHasMore.value
  }
  return false
})

const normalizePost = (item) => {
  const images = parseImages(item.images)
  const isReport = item.post_type === 1
  const isCheckIn = item.post_type === 2

  return {
    ...item,
    images,
    relativeTime: formatRelativeTime(item.created_at),
    tagText: isReport ? '战绩分享' : isCheckIn ? '球馆打卡' : '日常动态',
    tagClass: isReport ? 'report' : isCheckIn ? 'checkin' : 'daily'
  }
}

const parseImages = (images) => {
  if (!images) return []
  if (Array.isArray(images)) return images
  try {
    return JSON.parse(images)
  } catch (error) {
    return []
  }
}

const loadPublicPosts = async (isRefresh = false) => {
  const res = await getPublicPosts({
    page: publicPage.value,
    page_size: pageSize
  }).catch(() => ({ success: false, list: [] }))

  const list = (res.list || []).map(normalizePost)
  publicPosts.value = isRefresh ? list : [...publicPosts.value, ...list]
  publicHasMore.value = publicPosts.value.length < (res.total || publicPosts.value.length)
}

const loadFollowingPosts = async (isRefresh = false) => {
  if (!userStore.isLoggedIn) {
    followingPosts.value = []
    followingHasMore.value = false
    return
  }

  const res = await getPostList({
    page: followingPage.value,
    page_size: pageSize
  }).catch(() => ({ success: false, list: [] }))

  const list = (res.list || []).map(normalizePost)
  followingPosts.value = isRefresh ? list : [...followingPosts.value, ...list]
  followingHasMore.value = followingPosts.value.length < (res.total || followingPosts.value.length)
}

const loadData = async (isRefresh = false) => {
  if (isRefresh) {
    publicPage.value = 1
    followingPage.value = 1
  }

  loading.value = !isRefresh
  try {
    await Promise.all([
      loadPublicPosts(true),
      loadFollowingPosts(true)
    ])
  } finally {
    loading.value = false
    refreshing.value = false
    loadingMore.value = false
  }
}

const switchTab = (tab) => {
  if (activeTab.value === tab) return
  activeTab.value = tab
}

const onRefresh = async () => {
  refreshing.value = true
  await loadData(true)
}

const loadMore = async () => {
  if (loadingMore.value) return

  if (activeTab.value === 'friends' && followingHasMore.value) {
    loadingMore.value = true
    followingPage.value += 1
    await loadFollowingPosts(false)
    loadingMore.value = false
    return
  }

  if (activeTab.value === 'recommend' && publicHasMore.value) {
    loadingMore.value = true
    publicPage.value += 1
    await loadPublicPosts(false)
    loadingMore.value = false
  }
}

const goCreatePost = () => {
  shouldRefreshOnShow.value = true
  uni.navigateTo({ url: '/subPages/social/postCreate' })
}

const goPkRecords = () => {
  uni.navigateTo({ url: '/subPages/social/challenges' })
}

const goGeneratePkReport = () => {
  if (!userStore.isLoggedIn) {
    uni.showToast({ title: '请先登录后再生成PK报表', icon: 'none' })
    return
  }
  uni.navigateTo({ url: '/subPages/social/friendList?mode=pk-report' })
}

const goFriendList = () => {
  uni.navigateTo({ url: '/subPages/social/friendList' })
}

const goFriendRequests = () => {
  uni.navigateTo({ url: '/subPages/social/friendRequests' })
}

const openCurrentFeed = (tab = activeTab.value) => {
  if (tab === 'friends' && !userStore.isLoggedIn) {
    uni.navigateTo({ url: '/pages/login/login' })
    return
  }
  uni.navigateTo({ url: buildFeedUrl(tab) })
}

const connectUserWS = async () => {
  if (!userStore.isLoggedIn) return

  try {
    await userWS.connect()
    userWS.off(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
    userWS.on(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
  } catch (error) {
    console.error('[SocialPage] 用户WS连接失败:', error)
  }
}

const disconnectUserWS = () => {
  userWS.off(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
  userWS.disconnect()
}

const handleNotificationUpdate = () => {
  notificationStore.fetchUnreadCount()
  friendRequestStore.fetchPendingCount()
}

const openPkReport = (item) => {
  const opponentId = item.opponent_id || item.target_id || item.user_id || 0
  const opponentName = item.opponent_name || item.nickname || ''
  const query = [`opponent_name=${encodeURIComponent(opponentName)}`]

  if (opponentId) {
    query.unshift(`opponent_id=${opponentId}`)
  }

  uni.navigateTo({ url: `/subPages/social/pkReport?${query.join('&')}` })
}

const handlePreviewCardTap = () => {
  openCurrentFeed()
}

const previewImage = (images, index) => {
  uni.previewImage({
    urls: images,
    current: index
  })
}

const handleEmptyStateCta = () => {
  if (activeTab.value === 'friends' && !userStore.isLoggedIn) {
    uni.navigateTo({ url: '/pages/login/login' })
  }
}

onShow(() => {
  themeStore.syncTheme()
  themeStore.applyNavigationBarTheme()
  if (userStore.isLoggedIn) {
    notificationStore.fetchUnreadCount()
    friendRequestStore.fetchPendingCount()
  } else {
    friendRequestStore.clearPendingCount()
  }
  connectUserWS()
  if (shouldRefreshOnShow.value) {
    shouldRefreshOnShow.value = false
    loadData(true)
  }
})

onHide(() => {
  disconnectUserWS()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
