<template>
  <view class="home-container" :class="{ 'dark-mode': isDarkMode }">
    <view class="status-bar" :style="{ height: `${statusBarHeight}px` }"></view>

    <view class="nav-header">
      <view class="header-content">
        <view class="header-copy">
          <text class="title">追分</text>
          <text class="subtitle">{{ headerSubtitle }}</text>
        </view>
        <!-- #ifndef MP-WEIXIN -->
        <view class="header-right" @tap="goNotification">
          <uni-icons type="chat" size="22" :color="isDarkMode ? '#3A2E16' : '#231C0B'"></uni-icons>
          <view v-if="notificationStore.unreadCount > 0" class="header-badge">
            <text>{{ notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount }}</text>
          </view>
        </view>
        <!-- #endif -->
      </view>
    </view>

    <scroll-view
      scroll-y
      class="home-scroll"
      refresher-enabled
      :refresher-triggered="refreshing"
      @refresherrefresh="onRefresh"
    >
      <view class="page-body">
        <view v-if="homeLoading && !hasContent" class="hero-skeleton">
          <view class="skeleton-line skeleton-title"></view>
          <view class="skeleton-line skeleton-subtitle"></view>
          <view class="skeleton-actions">
            <view class="skeleton-btn"></view>
            <view class="skeleton-btn skeleton-btn-secondary"></view>
          </view>
        </view>

        <view v-else class="home-hero">
          <view class="hero-main">
            <view class="hero-copy">
              <text class="hero-eyebrow">{{ heroEyebrow }}</text>
              <text class="hero-title">{{ heroTitle }}</text>
              <text class="hero-subtitle">{{ heroSubtitle }}</text>
            </view>
            <view class="hero-actions">
              <view class="hero-btn hero-btn-primary" @tap="handlePrimaryAction">
                <text>{{ heroPrimaryText }}</text>
              </view>
              <view class="hero-btn hero-btn-secondary" @tap="handleSecondaryAction">
                <text>{{ heroSecondaryText }}</text>
              </view>
            </view>
          </view>
          <view class="hero-meta">
            <view v-for="tag in heroMetaTags" :key="tag" class="hero-meta-tag">
              <text>{{ tag }}</text>
            </view>
          </view>
        </view>

        <view class="summary-grid">
          <view class="summary-card" @tap="handleSummaryAction('match')">
            <view class="summary-head">
              <text class="summary-label">最近状态</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#9F926E' : '#6E6242'"></uni-icons>
            </view>
            <text class="summary-value">{{ recentMatchSummary.value }}</text>
            <text class="summary-desc">{{ recentMatchSummary.desc }}</text>
            <text class="summary-cta">{{ recentMatchSummary.cta }}</text>
          </view>

          <view class="summary-card" @tap="handleSummaryAction('ranking')">
            <view class="summary-head">
              <text class="summary-label">当前排名</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#9F926E' : '#6E6242'"></uni-icons>
            </view>
            <text class="summary-value">{{ rankingSummary.value }}</text>
            <text class="summary-desc">{{ rankingSummary.desc }}</text>
            <text class="summary-cta">{{ rankingSummary.cta }}</text>
          </view>
        </view>

        <view class="focus-section">
          <view class="section-header">
            <text class="section-title">赛事情报</text>
            <view class="section-more" @tap="goTo('/subPages/tournament/index')">
              <text>全部情报</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
            </view>
          </view>

          <view v-if="topEventNews" class="focus-card" @tap="goTo(`/subPages/tournament/detail?id=${topEventNews.id}`)">
            <view class="focus-card-top">
              <text class="focus-pill">{{ topEventNews.statusText }}</text>
              <text class="focus-aside">{{ topEventNews.showTime ? topEventNews.timeText : topEventNews.dateText }}</text>
            </view>
            <text class="focus-title">{{ topEventNews.title }}</text>
            <text class="focus-desc">{{ topEventNews.summary }}</text>
            <view class="focus-footer">
              <text>{{ topEventNews.gameTypeText }}</text>
              <text>{{ topEventNews.locationText || topEventNews.sourceText || '赛事情报' }}</text>
            </view>
          </view>
          <view v-else class="section-empty">
            <text class="empty-title">还没有可看的赛事情报</text>
            <text class="empty-desc">先去看看最近的赛讯更新，稍后再来刷新。</text>
          </view>
        </view>

        <view class="focus-section">
          <view class="section-header">
            <text class="section-title">排行焦点</text>
            <view class="section-more" @tap="goTo('/pages/ranking/index')">
              <text>完整榜单</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
            </view>
          </view>

          <view v-if="leaderboardTopThree.length" class="ranking-card">
            <view v-for="item in leaderboardTopThree" :key="item.user_id || item.rank" class="ranking-row">
              <view class="ranking-left">
                <text class="ranking-rank">#{{ item.rank }}</text>
                <image class="ranking-avatar" :src="resolveAvatarUrl(item.avatar, item.user_id)" mode="aspectFill"></image>
                <view class="ranking-copy">
                  <text class="ranking-name">{{ item.nickname || '球手' }}</text>
                  <text class="ranking-meta">{{ item.rank_name || '冲榜中' }}</text>
                </view>
              </view>
              <text class="ranking-score">{{ item.rank_score || 0 }}分</text>
            </view>
          </view>
          <view v-else class="section-empty">
            <text class="empty-title">排行榜还在准备中</text>
            <text class="empty-desc">打一局 PK 之后，你的成绩也会出现在这里。</text>
          </view>
        </view>

        <view class="focus-section nearby-venue-section">
          <view class="section-header">
            <text class="section-title">附近球房</text>
            <view class="section-more" @tap="goTo('/subPages/venue/index')">
              <text>查看球房</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
            </view>
          </view>

          <view v-if="nearbyVenueLoading && nearbyVenues.length === 0" class="section-empty">
            <text class="empty-title">正在寻找附近球房</text>
            <text class="empty-desc">定位成功后，会优先展示 5km 内最适合马上开局的球房。</text>
          </view>

          <view v-else-if="nearbyVenues.length > 0" class="venue-preview-list">
            <view
              v-for="item in nearbyVenues"
              :key="item.id"
              class="venue-preview-card"
              @tap="goToVenueDetail(item.id)"
            >
              <view class="venue-preview-main">
                <text class="venue-preview-name">{{ item.name || '球房' }}</text>
                <text class="venue-preview-address">{{ item.address || item.city || '地址待补充' }}</text>
                <view class="venue-preview-tags">
                  <text v-if="item.distance" class="venue-preview-tag accent">{{ formatVenueDistance(item.distance) }}</text>
                  <text v-if="item.table_count" class="venue-preview-tag">{{ item.table_count }}台</text>
                  <text v-if="item.price_range" class="venue-preview-tag">{{ item.price_range }}</text>
                  <text v-if="item.checkin_count" class="venue-preview-tag">{{ item.checkin_count }}人签到</text>
                </view>
              </view>
              <uni-icons type="right" size="18" :color="isDarkMode ? '#8b7a50' : '#9A8C67'"></uni-icons>
            </view>
          </view>

          <view v-else class="section-empty venue-empty-card">
            <text class="empty-title">附近暂无球房</text>
            <text class="empty-desc">{{ homeVenueEmptyAction.desc }}</text>
            <view class="empty-action" @tap="goTo(homeVenueEmptyAction.url)">
              <text>{{ homeVenueEmptyAction.text }}</text>
            </view>
          </view>
        </view>

        <view class="page-spacer"></view>
      </view>
    </scroll-view>

    <gameTypeModal
      v-model:visible="showGameTypeModal"
      :default-type="defaultGameType"
      @confirm="handleGameTypeConfirm"
    />
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useActivityStore } from '@/store/activity.js'
import { useNotificationStore } from '@/store/notification.js'
import { usePublicReadStore } from '@/store/publicRead.js'
import { useUserOverviewStore } from '@/store/userOverview.js'
import { useUserStore } from '@/store/user.js'
import { startMatch } from '@/api/match.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { getGameTypeLabel } from '@/utils/game-types.js'
import { normalizeSaiXunCard } from '@/utils/saixun.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
  buildHomeNearbyVenueParams,
  formatHomeVenueDistance,
  resolveHomeVenueEmptyAction
} from '@/utils/home-index.js'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'
import { readDefaultGameType, saveDefaultGameType } from '@/utils/game-type-preference.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const userStore = useUserStore()
const activityStore = useActivityStore()
const publicReadStore = usePublicReadStore()
const userOverviewStore = useUserOverviewStore()
const { isDarkMode } = usePageTheme()
const notificationStore = useNotificationStore()

const statusBarHeight = ref(uni.getSystemInfoSync().statusBarHeight || 0)
const isLoggedIn = computed(() => userStore.isLoggedIn)
const userId = computed(() => userStore.userId)
const userName = computed(() => userStore.nickname || '球友')

const refreshing = ref(false)
const homeLoading = ref(false)
const showGameTypeModal = ref(false)
const selectedGameType = ref(null)
const defaultGameType = ref(0)
const currentMatch = computed(() => (isLoggedIn.value ? activityStore.currentMatch : null))
const leaderboardTopThree = ref([])
const myRanking = ref(null)
const topEventNews = ref(null)
const nearbyVenues = ref([])
const nearbyVenueLoading = ref(false)
const favoriteVenueRewardStatus = ref(null)

const hasContent = computed(() => {
  return Boolean(currentMatch.value || topEventNews.value || leaderboardTopThree.value.length || nearbyVenues.value.length)
})

const homeVenueEmptyAction = computed(() => resolveHomeVenueEmptyAction(favoriteVenueRewardStatus.value))

const headerSubtitle = computed(() => {
  if (notificationStore.unreadCount > 0) {
    return `你有 ${notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount} 条未读互动`
  }
  if (currentMatch.value) {
    return '你的对局还在继续'
  }
  if (isLoggedIn.value) {
    return '今天适合来一场 PK'
  }
  return '记录比分、生成战报、冲击排行榜'
})

const heroMode = computed(() => {
  if (currentMatch.value) return 'ongoing'
  if (isLoggedIn.value) return 'ready'
  return 'guest'
})

const heroEyebrow = computed(() => {
  if (heroMode.value === 'ongoing') return '继续你的比赛节奏'
  if (heroMode.value === 'ready') return `${userName.value}，准备开始了吗`
  return '对局驱动的台球首页'
})

const heroTitle = computed(() => {
  if (heroMode.value === 'ongoing') return '你的对局还在继续'
  if (heroMode.value === 'ready') return '开始今天的第一场 PK'
  return '登录后立即开始对局'
})

const heroSubtitle = computed(() => {
  if (heroMode.value === 'ongoing' && currentMatch.value) {
    return `当前比分 ${getCurrentMatchScore(currentMatch.value)}，${getCurrentMatchOpponentName(currentMatch.value)} 还在等你继续。`
  }
  if (heroMode.value === 'ready') {
    if (myRanking.value?.rank > 0) {
      return `你目前排在第 ${myRanking.value.rank} 名，再赢一场还能继续冲榜。`
    }
    return '记录每一场真实比分，把战绩、排行榜和战报串成你的竞技成长线。'
  }
  return '登录后可以发起 PK、记录比分、生成战报并参与排行榜。'
})

const heroPrimaryText = computed(() => {
  if (heroMode.value === 'ongoing') return '继续对局'
  if (heroMode.value === 'ready') return '发起 PK'
  return '发起 PK'
})

const heroSecondaryText = computed(() => {
  if (heroMode.value === 'ongoing') return '查看 PK 记录'
  if (heroMode.value === 'ready') return '查看排行榜'
  return '出示 PK 码'
})

const heroMetaTags = computed(() => {
  if (heroMode.value === 'ongoing' && currentMatch.value) {
    return [
      getGameTypeLabel(currentMatch.value.game_type, '自由对局'),
      `进行中 ${formatDuration(currentMatch.value.duration_seconds)}`
    ]
  }

  if (heroMode.value === 'ready') {
    const tags = []
    if (myRanking.value?.rank > 0) {
      tags.push(`当前第 ${myRanking.value.rank} 名`)
    }
    if (myRanking.value?.rank_score) {
      tags.push(`排位分 ${myRanking.value.rank_score}`)
    }
    return tags.length ? tags : ['准备开始', '生成战报']
  }

  return ['支持排行', '支持战报']
})

const recentMatchSummary = computed(() => {
  if (currentMatch.value) {
    return {
      value: getCurrentMatchScore(currentMatch.value),
      desc: `${getCurrentMatchOpponentName(currentMatch.value)} · ${formatDuration(currentMatch.value.duration_seconds)}`,
      cta: '继续这一局'
    }
  }

  return {
    value: isLoggedIn.value ? '还没有最近战绩' : '登录后查看最近战绩',
    desc: isLoggedIn.value ? '打一场 PK 之后，这里会显示你的最新比分和结果。' : '登录后可查看自己的最近比分、对手和战报。',
    cta: isLoggedIn.value ? '去发起 PK' : '去登录'
  }
})

const rankingSummary = computed(() => {
  if (myRanking.value?.rank > 0) {
    return {
      value: `第 ${myRanking.value.rank} 名`,
      desc: `${myRanking.value.rank_name || '冲榜中'} · ${myRanking.value.rank_score || 0} 分`,
      cta: '查看完整榜单'
    }
  }

  return {
    value: isLoggedIn.value ? '暂未上榜' : '榜单随时可看',
    desc: isLoggedIn.value ? '打一局真实 PK 后，你的成绩会开始累积。' : '先看看平台高手成绩，再决定你的第一场 PK。',
    cta: '去排行榜'
  }
})

const getActivityIdentity = () => ({
  userId: userStore.userId,
  authGeneration: userStore.authGeneration
})

const loadData = async ({ force = false } = {}) => {
  if (homeLoading.value) return
  homeLoading.value = true

  try {
    const nearbyVenueRequest = loadNearbyVenues({ force })
    const requests = [
      publicReadStore.loadLeaderboardSummary({ game_type: 3 }, {
        force,
        identity: isLoggedIn.value ? getActivityIdentity() : undefined
      }).catch(() => ({ success: false })),
      publicReadStore.loadEventNews({ page: 1, page_size: 1 }, { force }).catch(() => ({ success: false, list: [] }))
    ]

    if (isLoggedIn.value) {
      requests.unshift(activityStore.fetch(getActivityIdentity(), { force, silent: true }).catch(() => activityStore.snapshot()))
      requests.push(userOverviewStore.fetch(getActivityIdentity(), { force, silent: true }).catch(() => userOverviewStore.snapshot()))
    } else {
      favoriteVenueRewardStatus.value = null
    }

    const results = await Promise.all(requests)
    let resultIndex = 0

    if (isLoggedIn.value) {
      resultIndex++
    }

    const leaderboardRes = results[resultIndex++]
    if (leaderboardRes.success) {
      leaderboardTopThree.value = (leaderboardRes.top_three || []).slice(0, 3)
      myRanking.value = leaderboardRes.my_ranking || null
    } else {
      leaderboardTopThree.value = []
      myRanking.value = null
    }

    const eventNewsRes = results[resultIndex++]
    if (eventNewsRes.success && Array.isArray(eventNewsRes.list)) {
      const topItem = eventNewsRes.list[0] || null
      topEventNews.value = topItem ? normalizeSaiXunCard(topItem) : null
    }
    if (isLoggedIn.value) {
      const overview = results[resultIndex]
      favoriteVenueRewardStatus.value = overview?.favoriteVenueRewardStatus?.success
        ? overview.favoriteVenueRewardStatus
        : null
    }
    await nearbyVenueRequest
  } catch (error) {
    console.error('加载首页数据失败', error)
  } finally {
    homeLoading.value = false
  }
}

const getHomeLocation = () => new Promise((resolve) => {
  uni.getLocation({
    type: 'gcj02',
    success: (res) => resolve({
      latitude: res.latitude,
      longitude: res.longitude
    }),
    fail: () => resolve(null)
  })
})

const loadNearbyVenues = async ({ force = false } = {}) => {
  if (nearbyVenueLoading.value) return

  nearbyVenueLoading.value = true
  try {
    const location = await getHomeLocation()
    const params = buildHomeNearbyVenueParams(location || {})
    if (!params) {
      nearbyVenues.value = []
      return
    }

    const res = await publicReadStore.loadNearbyVenues(params, { force })
    if (res.success && Array.isArray(res.list)) {
      nearbyVenues.value = res.list.slice(0, params.limit)
    }
  } catch (error) {
    console.error('加载首页附近球房失败', error)
  } finally {
    nearbyVenueLoading.value = false
  }
}

const onRefresh = async () => {
  refreshing.value = true
  await loadData({ force: true })
  refreshing.value = false
}

const formatDuration = (durationSeconds) => {
	if (!durationSeconds || durationSeconds < 0) return '0分'

	const totalMinutes = Math.floor(durationSeconds / 60)

	if (totalMinutes < 1) {
		return `${durationSeconds}秒`
	}

	if (totalMinutes < 60) {
		return `${totalMinutes}分`
	}

	const hours = Math.floor(totalMinutes / 60)
	const remainMinutes = totalMinutes % 60

	if (hours < 24) {
		if (remainMinutes > 0) {
			return `${hours}小时${remainMinutes}分`
		}
		return `${hours}小时`
	}

	const days = Math.floor(hours / 24)
	const remainHours = hours % 24

	if (remainHours > 0) {
		return `${days}天${remainHours}小时`
	}
	return `${days}天`
}

const getCurrentMatchScore = (match) => {
  const leftScore = match.my_score ?? match.player1_score ?? 0
  const rightScore = match.opponent_score ?? match.player2_score ?? 0
  return `${leftScore} : ${rightScore}`
}

const getCurrentMatchOpponentName = (match) => {
  if (match.opponent_name) return match.opponent_name
  if (match.player1_id === userId.value) return match.player2_name || '对手'
  if (match.player2_id === userId.value) return match.player1_name || '对手'
  return match.player2_name || match.player1_name || '对手'
}

const getCurrentMatchOpponentAvatar = (match) => {
  if (match.opponent_avatar) return match.opponent_avatar
  if (match.player1_id === userId.value) return match.player2_avatar || ''
  if (match.player2_id === userId.value) return match.player1_avatar || ''
  return match.player2_avatar || match.player1_avatar || ''
}

const goTo = (url, isTabPage = false) => {
  if (isTabPage) {
    uni.switchTab({ url })
    return
  }
  uni.navigateTo({ url })
}

const goLogin = () => {
  uni.navigateTo({ url: '/pages/login/login' })
}

const goNotification = () => {
  uni.navigateTo({ url: '/subPages/notification/index' })
}

const goToVenueDetail = (id) => {
  if (!id) return
  goTo(`/subPages/venue/detail?id=${id}`)
}

const formatVenueDistance = (meters) => formatHomeVenueDistance(meters)

const handlePrimaryAction = () => {
  if (heroMode.value === 'ongoing' && currentMatch.value) {
    handleContinueMatch(currentMatch.value)
    return
  }

  if (heroMode.value === 'ready') {
    handleStartPK()
    return
  }

  goLogin()
}

const handleSecondaryAction = () => {
  if (heroMode.value === 'ongoing') {
    goTo('/subPages/social/challenges')
    return
  }

  if (heroMode.value === 'ready') {
    goTo('/pages/ranking/index')
    return
  }

  goLogin()
}

const handleSummaryAction = (type) => {
  if (type === 'match') {
    if (currentMatch.value) {
      handleContinueMatch(currentMatch.value)
      return
    }

    if (isLoggedIn.value) {
      handleStartPK()
      return
    }

    goLogin()
    return
  }

  goTo('/pages/ranking/index')
}

const handleStartPK = () => {
  if (!isLoggedIn.value) {
    uni.showToast({ title: '请先登录后再发起PK', icon: 'none' })
    setTimeout(() => goLogin(), 1200)
    return
  }

  defaultGameType.value = readDefaultGameType(uni, userId.value)
  showGameTypeModal.value = true
}

const handleGameTypeConfirm = ({ gameType, setAsDefault } = {}) => {
  if (setAsDefault) {
    defaultGameType.value = saveDefaultGameType(uni, userId.value, gameType)
  }
  selectedGameType.value = gameType

  let handledByScan = false
  // #ifdef APP-PLUS || APP-HARMONY
  handledByScan = true
  uni.scanCode({
    scanType: ['qrCode'],
    success: (res) => handleMatchResult(res.result),
    fail: () => {}
  })
  // #endif

  if (handledByScan) {
    return
  }

  uni.showModal({
    title: '当前端暂不支持扫码发起 PK',
    content: '你可以先去查看 PK 记录，或者切换到 App 端发起扫码对局。',
    confirmText: '去 PK 记录',
    cancelText: '我知道了',
    success: ({ confirm }) => {
      if (confirm) {
        goTo('/subPages/social/challenges')
      }
    }
  })
}

const handleMatchResult = async (scanResult) => {
  try {
    const opponentData = JSON.parse(scanResult)
    if (!opponentData.user_id) {
      uni.showToast({ title: '无效的二维码', icon: 'none' })
      return
    }

    uni.showLoading({ title: '匹配中...', mask: true })
    const res = await startMatch({
      game_type: selectedGameType.value,
      opponent_id: opponentData.user_id,
      opponent_name: opponentData.nickname || '对手'
    })
    uni.hideLoading()
    handleStartMatchOutcome(resolveStartMatchGuardAction({
      response: res,
      selectedGameType: selectedGameType.value,
      scannedOpponent: opponentData
    }))
  } catch (error) {
    uni.hideLoading()
    uni.showToast({ title: '匹配失败', icon: 'none' })
  }
}

const navigateToPlayingMatch = (match) => {
  uni.navigateTo({
    url: buildPlayingRoute(match)
  })
}

const handleStartMatchOutcome = (outcome) => {
  if (outcome.type === 'navigate' || outcome.type === 'resume') {
    navigateToPlayingMatch(outcome.match)
    return
  }

  if (outcome.type === 'prompt_self_ongoing') {
    uni.showModal({
      title: '你有未结束的对局',
      content: outcome.message,
      confirmText: '进入该对局',
      cancelText: '稍后处理',
      success: ({ confirm }) => {
        if (confirm) {
          navigateToPlayingMatch(outcome.match)
        }
      }
    })
    return
  }

  uni.showToast({
    title: outcome.message || '创建对局失败',
    icon: 'none'
  })
}

const handleContinueMatch = (match) => {
  navigateToPlayingMatch({
    match_id: match.id,
    game_type: match.game_type || 3,
    opponent_name: getCurrentMatchOpponentName(match),
    opponent_avatar: getCurrentMatchOpponentAvatar(match)
  })
}

onShow(() => {
  loadData()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
