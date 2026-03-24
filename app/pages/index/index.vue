<template>
  <view class="home-container" :class="{ 'dark-mode': isDarkMode }">
    <view class="status-bar" :style="{ height: `${statusBarHeight}px` }"></view>

    <view class="nav-header">
      <view class="header-content">
        <view class="header-copy">
          <text class="title">追分</text>
          <text class="subtitle">{{ headerSubtitle }}</text>
        </view>
        <view class="header-right" @tap="goNotification">
          <uni-icons type="chat" size="22" :color="isDarkMode ? '#e2e8f0' : '#1e293b'"></uni-icons>
          <view v-if="notificationStore.unreadCount > 0" class="header-badge">
            <text>{{ notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount }}</text>
          </view>
        </view>
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
              <uni-icons type="right" size="14" :color="isDarkMode ? '#94a3b8' : '#64748b'"></uni-icons>
            </view>
            <text class="summary-value">{{ recentMatchSummary.value }}</text>
            <text class="summary-desc">{{ recentMatchSummary.desc }}</text>
            <text class="summary-cta">{{ recentMatchSummary.cta }}</text>
          </view>

          <view class="summary-card" @tap="handleSummaryAction('ranking')">
            <view class="summary-head">
              <text class="summary-label">当前排名</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#94a3b8' : '#64748b'"></uni-icons>
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
              <uni-icons type="right" size="14" :color="isDarkMode ? '#94a3b8' : '#94a3b8'"></uni-icons>
            </view>
          </view>

          <view v-if="featuredEventNews" class="focus-card" @tap="goTo(`/subPages/tournament/detail?id=${featuredEventNews.id}`)">
            <view class="focus-card-top">
              <text class="focus-pill">{{ featuredEventNews.statusText }}</text>
              <text class="focus-aside">{{ featuredEventNews.timeText }}</text>
            </view>
            <text class="focus-title">{{ featuredEventNews.title }}</text>
            <text class="focus-desc">{{ featuredEventNews.summary }}</text>
            <view class="focus-footer">
              <text>{{ featuredEventNews.typeText }}</text>
              <text>{{ featuredEventNews.locationText || featuredEventNews.sourceText || '赛事情报' }}</text>
            </view>
          </view>
          <view v-else class="section-empty">
            <text class="empty-title">还没有可看的赛事情报</text>
            <text class="empty-desc">先去看看最近的赛事动态，稍后再来刷新。</text>
          </view>
        </view>

        <view class="focus-section">
          <view class="section-header">
            <text class="section-title">排行焦点</text>
            <view class="section-more" @tap="goTo('/pages/ranking/index')">
              <text>完整榜单</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#94a3b8' : '#94a3b8'"></uni-icons>
            </view>
          </view>

          <view v-if="leaderboardTopThree.length" class="ranking-card">
            <view v-for="item in leaderboardTopThree" :key="item.user_id || item.rank" class="ranking-row">
              <view class="ranking-left">
                <text class="ranking-rank">#{{ item.rank }}</text>
                <image class="ranking-avatar" :src="item.avatar || '/static/images/default-avatar.png'" mode="aspectFill"></image>
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

        <view class="focus-section">
          <view class="section-header">
            <text class="section-title">精选内容</text>
            <view class="section-more" @tap="goCommunity">
              <text>更多动态</text>
              <uni-icons type="right" size="14" :color="isDarkMode ? '#94a3b8' : '#94a3b8'"></uni-icons>
            </view>
          </view>

          <view v-if="featuredPost" class="featured-post-card" @tap="handleFeaturedPostAction">
            <view class="featured-top">
              <image class="featured-avatar" :src="featuredPost.avatar || '/static/images/default-avatar.png'" mode="aspectFill"></image>
              <view class="featured-copy">
                <view class="featured-name-row">
                  <text class="featured-name">{{ featuredPost.nickname || '球友' }}</text>
                  <text class="featured-tag">{{ featuredPost.tagText }}</text>
                </view>
                <text class="featured-time">{{ featuredPost.relativeTime }}</text>
              </view>
            </view>
            <text class="featured-content">{{ featuredPost.content || '来社区看看大家今天的竞技状态。' }}</text>
            <view class="featured-footer">
              <text>❤️ {{ featuredPost.likes_count || 0 }}</text>
              <text>💬 {{ featuredPost.comments_count || 0 }}</text>
              <view class="featured-action" @tap.stop="handleFeaturedPostAction">
                <text>{{ featuredPost.actionText }}</text>
              </view>
            </view>
          </view>
          <view v-else class="section-empty">
            <text class="empty-title">社区今天有点安静</text>
            <text class="empty-desc">去发一条动态，或者晚点回来看看新的战报。</text>
          </view>
        </view>

        <view class="focus-section">
          <view class="section-header">
            <text class="section-title">常用工具</text>
          </view>
          <scroll-view scroll-x class="tool-scroll" show-scrollbar="false">
            <view class="tool-chips">
              <view v-for="item in toolEntries" :key="item.label" class="tool-chip" @tap="goTo(item.url, item.isTab)">
                <view class="tool-icon">
                  <uni-icons :type="item.icon" size="22" :color="item.iconColor"></uni-icons>
                </view>
                <view class="tool-copy">
                  <text class="tool-label">{{ item.label }}</text>
                  <text class="tool-desc">{{ item.desc }}</text>
                </view>
              </view>
            </view>
          </scroll-view>
        </view>

        <view class="page-spacer"></view>
      </view>
    </scroll-view>

    <gameTypeModal
      v-model:visible="showGameTypeModal"
      @confirm="handleGameTypeConfirm"
    />
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { useThemeStore } from '@/store/theme.js'
import { useNotificationStore } from '@/store/notification.js'
import { getFeaturedEventNews } from '@/api/event-news.js'
import { getCurrentMatch, startMatch } from '@/api/match.js'
import { getPublicPosts } from '@/api/social.js'
import { getLeaderboard } from '@/api/rank.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { formatRelativeTime } from '@/utils/format.js'
import { getGameTypeLabel } from '@/utils/game-types.js'
import { buildFeaturedPostTarget, normalizeFeaturedEventNews } from '@/utils/home-index.js'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'

const userStore = useUserStore()
const themeStore = useThemeStore()
const notificationStore = useNotificationStore()

const statusBarHeight = ref(uni.getSystemInfoSync().statusBarHeight)
const isDarkMode = computed(() => themeStore.isDarkMode)
const isLoggedIn = computed(() => userStore.isLoggedIn)
const userId = computed(() => userStore.userId)
const userName = computed(() => userStore.nickname || '球友')

const refreshing = ref(false)
const homeLoading = ref(true)
const showGameTypeModal = ref(false)
const selectedGameType = ref(null)
const currentMatch = ref(null)
const leaderboardTopThree = ref([])
const myRanking = ref(null)
const featuredEventNews = ref(null)
const featuredPost = ref(null)

const hasContent = computed(() => {
  return Boolean(currentMatch.value || featuredEventNews.value || featuredPost.value || leaderboardTopThree.value.length)
})

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
  return '登录 / 注册'
})

const heroSecondaryText = computed(() => {
  if (heroMode.value === 'ongoing') return '查看 PK 记录'
  if (heroMode.value === 'ready') return '查看排行榜'
  return '先看排行榜'
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
      tags.push(`积分 ${myRanking.value.rank_score}`)
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

  if (featuredPost.value?.post_type === 1) {
    return {
      value: '焦点战报',
      desc: `${featuredPost.value.nickname || '球友'} 刚分享了一条真实战绩`,
      cta: featuredPost.value.actionText
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

const toolEntries = computed(() => [
  { label: 'PK记录', desc: '查看邀约与结果', icon: 'flag-filled', iconColor: '#E0AE12', url: '/subPages/social/challenges' },
  { label: '深度统计', desc: '看你的竞技画像', icon: 'bars', iconColor: '#E0AE12', url: '/subPages/user/statsDetail' },
  { label: '规则说明', desc: '快速查台球规则', icon: 'help', iconColor: '#7c3aed', url: '/subPages/rules/index' },
  { label: '赛事情报', desc: '查看最近赛程赛况', icon: 'calendar', iconColor: '#ea580c', url: '/subPages/tournament/index' },
  { label: '球房场馆', desc: '寻找附近球房', icon: 'location', iconColor: '#0f766e', url: '/subPages/venue/index' }
])

const loadData = async () => {
  homeLoading.value = true

  try {
    const requests = [
      getLeaderboard({ page: 1, page_size: 3 }).catch(() => ({ success: false })),
      getFeaturedEventNews().catch(() => ({ success: false })),
      getPublicPosts({ page: 1, page_size: 6 }).catch(() => ({ success: false }))
    ]

    if (isLoggedIn.value) {
      requests.unshift(getCurrentMatch({ silent: true }).catch(() => ({ success: false })))
    }

    const results = await Promise.all(requests)
    let resultIndex = 0

    if (isLoggedIn.value) {
      const currentMatchRes = results[resultIndex++]
      currentMatch.value = currentMatchRes.success && currentMatchRes.match ? currentMatchRes.match : null
    } else {
      currentMatch.value = null
    }

    const leaderboardRes = results[resultIndex++]
    if (leaderboardRes.success) {
      leaderboardTopThree.value = (leaderboardRes.top_three || []).slice(0, 3)
      myRanking.value = leaderboardRes.my_ranking || null
    } else {
      leaderboardTopThree.value = []
      myRanking.value = null
    }

    const featuredRes = results[resultIndex++]
    featuredEventNews.value = featuredRes.success
      ? normalizeFeaturedEventNews(featuredRes.event_news || featuredRes.eventNews)
      : null

    const postRes = results[resultIndex]
    featuredPost.value = postRes.success ? pickFeaturedPost(postRes.list || []) : null
  } catch (error) {
    console.error('加载首页数据失败', error)
  } finally {
    homeLoading.value = false
  }
}

const onRefresh = async () => {
  refreshing.value = true
  await Promise.all([loadData(), notificationStore.fetchUnreadCount()])
  refreshing.value = false
}

const pickFeaturedPost = (list) => {
  const target = list.find((item) => item.post_type === 1) || list[0]
  if (!target) return null

  const action = buildFeaturedPostTarget(target)

  return {
    ...target,
    tagText: getPostTagText(target.post_type),
    relativeTime: formatRelativeTime(target.created_at),
    action,
    actionText: action.ctaText
  }
}

const getPostTagText = (postType) => {
  if (postType === 1) return '战报'
  if (postType === 2) return '打卡'
  return '动态'
}

const formatDuration = (durationSeconds) => {
  if (!durationSeconds || durationSeconds < 0) return '00:00'
  const minutes = Math.floor(durationSeconds / 60)
  const seconds = durationSeconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
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

const goCommunity = () => {
  uni.switchTab({ url: '/pages/social/index' })
}

const goLogin = () => {
  uni.navigateTo({ url: '/pages/login/login' })
}

const goNotification = () => {
  uni.navigateTo({ url: '/subPages/notification/index' })
}

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

  goTo('/pages/ranking/index')
}

const handleSummaryAction = (type) => {
  if (type === 'match') {
    if (currentMatch.value) {
      handleContinueMatch(currentMatch.value)
      return
    }

    if (featuredPost.value?.post_type === 1) {
      handleFeaturedPostAction()
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

const handleFeaturedPostAction = () => {
  if (!featuredPost.value) {
    goCommunity()
    return
  }

  if (featuredPost.value.action?.type === 'navigate' && featuredPost.value.action.url) {
    uni.navigateTo({ url: featuredPost.value.action.url })
    return
  }

  goCommunity()
}

const handleStartPK = () => {
  if (!isLoggedIn.value) {
    uni.showToast({ title: '请先登录后再发起PK', icon: 'none' })
    setTimeout(() => goLogin(), 1200)
    return
  }

  showGameTypeModal.value = true
}

const handleGameTypeConfirm = (gameType) => {
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
  themeStore.syncTheme()
  themeStore.applyNavigationBarTheme()
  notificationStore.fetchUnreadCount()
  loadData()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
