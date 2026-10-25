<template>
  <view class="home-container" :class="{ 'dark-mode': isDarkMode }">
    <view class="status-bar" :style="{ height: `${statusBarHeight}px` }"></view>

    <view class="nav-header">
      <view class="header-content">
        <view class="header-copy">
          <text class="title">追分</text>
          <text class="subtitle">{{ greetingText }}</text>
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
        <view v-if="homePageError && !hasContent" class="home-load-error">
          <uni-icons type="info" size="48" :color="mutedIconColor"></uni-icons>
          <text class="home-load-error__title">{{ homePageError.title }}</text>
          <text class="home-load-error__desc">{{ homePageError.description }}</text>
          <button class="home-load-error__action" @tap="handleHomeErrorAction">{{ homePageError.actionText }}</button>
        </view>

        <template v-else>
          <!-- 我的对局 / 开局：首页唯一的主操作 -->
          <view v-if="playCardLoading" class="play-card play-skeleton">
            <view class="skeleton-line skeleton-line--title"></view>
            <view class="skeleton-line"></view>
            <view class="skeleton-btn"></view>
          </view>

          <view v-else-if="ongoingCard" class="play-card play-card--ongoing">
            <view class="play-match" @tap="handleContinueCurrentMatch">
              <view class="play-head">
                <view class="state-chip" :class="ongoingCard.isPending ? 'state-chip--pending' : 'state-chip--live'">
                  <view class="state-chip-dot"></view>
                  <text>{{ ongoingCard.statusText }}</text>
                </view>
                <text class="play-meta">{{ ongoingCard.metaText }}</text>
                <text class="play-duration">{{ ongoingCard.durationText }}</text>
              </view>
              <view class="versus">
                <view class="versus-player">
                  <image class="versus-avatar" :src="resolveAvatarUrl(ongoingCard.players[0].avatar, ongoingCard.players[0].id)" mode="aspectFill"></image>
                  <view class="versus-name-row">
                    <text class="versus-name">{{ ongoingCard.players[0].name }}</text>
                    <text v-if="ongoingCard.players[0].isMe" class="me-tag">我</text>
                  </view>
                </view>
                <text class="versus-score">{{ ongoingCard.players[0].score }} : {{ ongoingCard.players[1].score }}</text>
                <view class="versus-player">
                  <image class="versus-avatar" :src="resolveAvatarUrl(ongoingCard.players[1].avatar, ongoingCard.players[1].id)" mode="aspectFill"></image>
                  <view class="versus-name-row">
                    <text class="versus-name">{{ ongoingCard.players[1].name }}</text>
                    <text v-if="ongoingCard.players[1].isMe" class="me-tag">我</text>
                  </view>
                </view>
              </view>
            </view>
            <button class="play-btn" @tap="handleContinueCurrentMatch">{{ ongoingCard.actionLabel }}</button>
          </view>

          <view v-else-if="isLoggedIn" class="play-card">
            <text class="play-title">来一局</text>
            <text class="play-desc">扫码或选择球友开局，比分实时同步给双方</text>
            <view class="play-actions">
              <button class="play-btn" @tap="handleStartMatch">发起对局</button>
              <button class="play-btn play-btn--ghost" @tap="goTo('/subPages/match/challengeCompose')">约球</button>
            </view>
          </view>

          <view v-else class="play-card">
            <text class="play-title">登录后开始记分</text>
            <text class="play-desc">发起对局、约球，排位赛成绩自动计入排行榜</text>
            <button class="play-btn" @tap="goLogin">登录 / 注册</button>
          </view>

          <view
            v-if="challengeCard"
            class="challenge-card"
            :class="{ 'challenge-card--action': challengeCard.needsAction }"
            @tap="goTo(challengeCard.url)"
          >
            <view class="challenge-copy">
              <text class="challenge-title">{{ challengeCard.title }}</text>
              <text class="challenge-sub">{{ challengeCard.subText }}</text>
            </view>
            <view class="challenge-action">
              <text>{{ challengeCard.actionLabel }}</text>
            </view>
          </view>

          <!-- 约球状态读不到时不能当作「没有约球」，给出重试入口 -->
          <view v-if="isLoggedIn && !activityStore.challengeAvailable" class="challenge-unavailable" @tap="retryChallengeActivity">
            <text class="challenge-unavailable__text">约球状态读取失败，点击重试</text>
          </view>

          <view v-if="homeRefreshError || homePageError" class="home-refresh-error">
            <text>{{ (homeRefreshError || homePageError).description }}</text>
            <text class="home-refresh-error__action" @tap="retryHomeData">重试</text>
          </view>

          <!-- 正在进行：公开大厅的预览，完整列表在观赛大厅 -->
          <view class="home-section">
            <view class="section-header">
              <text class="section-title">正在进行</text>
              <view class="section-more" @tap="goTo('/pages/match/index')">
                <text>观赛大厅</text>
                <uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
            </view>
            <view v-if="liveMatchError" class="section-empty">
              <text class="empty-title">{{ liveMatchError.title }}</text>
              <text class="empty-desc">{{ liveMatchError.description }}</text>
              <text class="section-error__action" @tap="retryHomeData">重试</text>
            </view>
            <view v-else-if="liveMatches.length" class="live-card">
              <view v-for="match in liveMatches" :key="match.id" class="live-item" @tap="handleLiveMatchTap(match)">
                <view class="live-item-head">
                  <text class="live-item-meta">{{ match.metaText }}</text>
                  <view class="live-state">
                    <view class="live-state-dot"></view>
                    <text>{{ match.statusText }}</text>
                  </view>
                </view>
                <view v-for="(player, index) in match.players" :key="index" class="live-player">
                  <image class="live-player-avatar" :src="resolveAvatarUrl(player.avatar, player.id)" mode="aspectFill"></image>
                  <text class="live-player-name">{{ player.name }}</text>
                  <text v-if="player.isMe" class="me-tag">我</text>
                  <text class="live-player-score">{{ player.score }}</text>
                </view>
              </view>
            </view>
            <view v-else-if="homeLoading" class="skeleton-card skeleton-card--live"></view>
            <view v-else class="section-empty">
              <text class="empty-title">暂无正在进行的对局</text>
              <text class="empty-desc">公开对局开打后会出现在这里。</text>
            </view>
          </view>

          <!-- 赛事 -->
          <view class="home-section">
            <view class="section-header">
              <text class="section-title">赛事</text>
              <view class="section-more" @tap="goTo('/pages/tournament/index', true)">
                <text>全部赛讯</text>
                <uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
            </view>

            <view v-if="eventNewsError" class="section-empty">
              <text class="empty-title">{{ eventNewsError.title }}</text>
              <text class="empty-desc">{{ eventNewsError.description }}</text>
              <text class="section-error__action" @tap="retryHomeData">重试</text>
            </view>
            <view v-else-if="topEventNews" class="event-card" @tap="goTo(`/subPages/tournament/detail?id=${topEventNews.id}`)">
              <view class="event-media">
                <image v-if="topEventNews.hasCover" class="event-cover" :src="topEventNews.coverImage" mode="aspectFill"></image>
                <view class="event-scrim"></view>
                <view class="event-badge" :class="'event-badge--' + topEventNews.statusTone">
                  <view v-if="topEventNews.statusTone === 'live'" class="event-badge-dot"></view>
                  <text>{{ topEventNews.statusLabel }}</text>
                </view>
                <view class="event-copy">
                  <text class="event-title">{{ topEventNews.title }}</text>
                  <text class="event-meta">{{ topEventNews.metaText }}</text>
                </view>
              </view>
              <view v-if="topEventNews.highlight" class="event-strip">
                <text v-if="topEventNews.highlight.label" class="event-strip-label">{{ topEventNews.highlight.label }}</text>
                <text class="event-strip-text">{{ topEventNews.highlight.primary }}</text>
                <text v-if="topEventNews.highlight.secondary" class="event-strip-detail">{{ topEventNews.highlight.secondary }}</text>
              </view>
            </view>
            <view v-else-if="homeLoading" class="skeleton-card skeleton-card--event"></view>
            <view v-else class="section-empty">
              <text class="empty-title">暂时没有赛讯</text>
              <text class="empty-desc">新的职业赛事公布后会出现在这里。</text>
            </view>
          </view>

          <!-- 排行榜 -->
          <view class="home-section">
            <view class="section-header">
              <view class="section-title-row">
                <text class="section-title">排行榜</text>
                <text class="section-caption">中式八球</text>
              </view>
              <view class="section-more" @tap="goTo('/pages/ranking/index')">
                <text>完整榜单</text>
                <uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
            </view>

            <view v-if="leaderboardError" class="section-empty">
              <text class="empty-title">{{ leaderboardError.title }}</text>
              <text class="empty-desc">{{ leaderboardError.description }}</text>
              <text class="section-error__action" @tap="retryHomeData">重试</text>
            </view>
            <view v-else-if="leaderboardTopThree.length" class="ranking-card">
              <view v-for="item in leaderboardTopThree" :key="item.user_id || item.rank" class="ranking-row">
                <text class="ranking-medal" :class="'ranking-medal--' + item.rank">{{ item.rank }}</text>
                <memberAvatar class="ranking-avatar" :avatar="item.avatar" :user-id="item.user_id" :vip-level="item.vip_level" :rank-level="item.rank_level" :size="68" reserve-frame />
                <view class="ranking-copy">
                  <text class="ranking-name">{{ item.nickname || '球手' }}</text>
                  <text class="ranking-meta">{{ item.rank_name || '冲榜中' }}</text>
                </view>
                <text class="ranking-score">{{ item.rank_score || 0 }}<text class="ranking-score-unit">分</text></text>
              </view>
              <view v-if="isLoggedIn" class="ranking-mine" @tap="goTo('/pages/ranking/index')">
                <text class="ranking-mine-label">我</text>
                <text class="ranking-mine-text">{{ myRankingText }}</text>
                <uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
            </view>
            <view v-else-if="homeLoading" class="skeleton-card skeleton-card--ranking"></view>
            <view v-else class="section-empty">
              <text class="empty-title">排行榜还在准备中</text>
              <text class="empty-desc">打一场排位赛，你的成绩也会出现在这里。</text>
            </view>
          </view>

          <!-- 附近球房 -->
          <view class="home-section nearby-venue-section">
            <view class="section-header">
              <text class="section-title">附近球房</text>
              <view class="section-more" @tap="goTo('/subPages/venue/index')">
                <text>查看球房</text>
                <uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
              </view>
            </view>

            <view v-if="nearbyVenueFeedback" class="section-empty">
              <text class="empty-title">{{ nearbyVenueFeedback.title }}</text>
              <text class="empty-desc">{{ nearbyVenueFeedback.description }}</text>
              <text class="section-error__action" @tap="retryNearbyVenues">重试</text>
            </view>

            <view v-else-if="nearbyVenueLoading && nearbyVenues.length === 0" class="skeleton-card skeleton-card--venue"></view>

            <view v-else-if="nearbyVenues.length > 0" class="venue-card">
              <view
                v-for="item in nearbyVenues"
                :key="item.id"
                class="venue-row"
                @tap="goToVenueDetail(item.id)"
              >
                <view class="venue-main">
                  <text class="venue-name">{{ item.name || '球房' }}</text>
                  <text class="venue-address">{{ item.address || item.city || '地址待补充' }}</text>
                  <view class="venue-tags">
                    <text v-if="item.table_count" class="venue-tag">{{ item.table_count }}台</text>
                    <text v-if="item.price_range" class="venue-tag">{{ item.price_range }}</text>
                    <text v-if="item.checkin_count" class="venue-tag">{{ item.checkin_count }}人签到</text>
                  </view>
                </view>
                <text v-if="item.distance" class="venue-distance">{{ formatVenueDistance(item.distance) }}</text>
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
        </template>

        <view class="page-spacer"></view>
      </view>
    </scroll-view>


    <view v-if="showMatchQrModal" class="qr-mask" @tap="closeMatchQrModal">
      <view class="qr-panel" @tap.stop>
        <view class="qr-header">
          <view>
            <text class="qr-title">出示我的匹配二维码</text>
            <text class="qr-subtitle">请让对手使用发起对局扫码入口识别</text>
          </view>
          <button class="qr-close" @tap="closeMatchQrModal">
            <uni-icons type="closeempty" size="22" :color="mutedIconColor"></uni-icons>
          </button>
        </view>
        <view class="qr-body">
          <view v-if="matchQrLoading" class="qr-state">
            <uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
            <text>二维码生成中...</text>
          </view>
          <view v-if="!matchQrLoading && matchQrFailed" class="qr-state">
            <text class="qr-error">二维码加载失败</text>
            <button class="qr-retry" @tap="loadMyMatchQRCode">重试</button>
          </view>
          <canvas
            v-if="matchQrContent && !matchQrFailed"
            canvas-id="home-match-invite-qrcode"
            id="home-match-invite-qrcode"
            class="qr-image"
            :class="{ 'is-loading': matchQrLoading }"
            :width="240"
            :height="240"
          />
        </view>
        <button class="qr-done" @tap="closeMatchQrModal">关闭</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, nextTick, onUnmounted, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useActivityStore } from '@/store/activity.js'
import { usePublicReadStore } from '@/store/publicRead.js'
import { useUserOverviewStore } from '@/store/userOverview.js'
import { useUserStore } from '@/store/user.js'
import { getMatchQRCode } from '@/api/match.js'
import memberAvatar from '@/components/memberAvatar.vue'
import { normalizeSaiXunCard } from '@/utils/saixun.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useMainTabBar, setMainTabBarVisible } from '@/utils/use-main-tab-bar.js'
import { startQuickScan } from '@/utils/quick-scan-entry.js'
import {
  buildHomeNearbyVenueParams,
  formatHomeMyRanking,
  formatHomeVenueDistance,
  resolveHomeGreeting,
  resolveHomeVenueEmptyAction,
  shouldReloadHomeNearbyVenues
} from '@/utils/home-index.js'
import { buildOngoingMatchCard } from '@/utils/match-core-flow.js'
import { resolveChallengeQuickCard } from '@/utils/challenge-view.js'
import { buildPlayingRoute } from '@/utils/ongoing-match-guard.js'
import { buildHomeLiveMatches, shouldOpenPlayingForSpectatorMatch } from '@/utils/spectator-lobby.js'
import { getCurrentLocation } from '@/utils/permission-helper.js'
import { renderLocalQRCode } from '@/utils/local-qrcode.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
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

const userStore = useUserStore()
const activityStore = useActivityStore()
const publicReadStore = usePublicReadStore()
const userOverviewStore = useUserOverviewStore()
const { isDarkMode } = usePageTheme()
useMainTabBar('index')

const statusBarHeight = ref(uni.getSystemInfoSync().statusBarHeight || 0)
const isLoggedIn = computed(() => userStore.isLoggedIn)
const userId = computed(() => userStore.userId)
const userName = computed(() => userStore.nickname || '球友')
const greetingHour = ref(new Date().getHours())

const refreshing = ref(false)
const homeState = ref(createAsyncPageState({
  authGeneration: userStore.authGeneration,
  data: []
}))
const homeLoading = computed(() => homeState.value.status === ASYNC_PAGE_STATUS.LOADING)
const homePageError = computed(() => {
  if (homeState.value.status !== ASYNC_PAGE_STATUS.ERROR) return null
  return resolveAsyncPageErrorFeedback(homeState.value.error, { resource: '首页内容' })
})
const homeRefreshError = computed(() => (
  homeState.value.refreshError
    ? resolveAsyncPageErrorFeedback(homeState.value.refreshError, { resource: '首页内容' })
    : null
))
const showMatchQrModal = ref(false)
const matchQrLoading = ref(false)
const matchQrFailed = ref(false)
const matchQrContent = ref('')
const currentMatch = computed(() => (isLoggedIn.value ? activityStore.currentMatch : null))
const leaderboardTopThree = ref([])
const myRanking = ref(null)
const topEventNews = ref(null)
const liveMatchList = ref([])
const nearbyVenues = ref([])
const nearbyVenueLoading = ref(false)
const nearbyVenueError = ref(null)
const nearbyVenueAttemptAt = ref(0)
const favoriteVenueRewardStatus = ref(null)
const homeSectionErrors = ref({ leaderboard: null, eventNews: null, liveMatches: null })
const leaderboardError = computed(() => (
  homeSectionErrors.value.leaderboard
    ? resolveAsyncPageErrorFeedback(homeSectionErrors.value.leaderboard, { resource: '排行榜' })
    : null
))
const eventNewsError = computed(() => (
  homeSectionErrors.value.eventNews
    ? resolveAsyncPageErrorFeedback(homeSectionErrors.value.eventNews, { resource: '赛事情报' })
    : null
))
const liveMatchError = computed(() => (
  homeSectionErrors.value.liveMatches
    ? resolveAsyncPageErrorFeedback(homeSectionErrors.value.liveMatches, { resource: '正在进行的对局' })
    : null
))
const nearbyVenueLocationIssue = ref('')
const nearbyVenueFeedback = computed(() => {
  if (nearbyVenueLocationIssue.value === 'denied') {
    return { title: '未开启定位权限', description: '开启定位权限后可查看附近球房' }
  }
  if (nearbyVenueLocationIssue.value === 'unavailable') {
    return { title: '暂时无法获取位置', description: '请检查定位服务后重试' }
  }
  return nearbyVenueError.value
    ? resolveAsyncPageErrorFeedback(nearbyVenueError.value, { resource: '附近球房' })
    : null
})

const hasContent = computed(() => {
  return Boolean(
    currentMatch.value || topEventNews.value || leaderboardTopThree.value.length || liveMatchList.value.length || nearbyVenues.value.length
  )
})

const homeVenueEmptyAction = computed(() => resolveHomeVenueEmptyAction(favoriteVenueRewardStatus.value))
const mutedIconColor = computed(() => (isDarkMode.value ? '#9F926E' : '#9A8C67'))

const greetingText = computed(() => (
  isLoggedIn.value ? `${resolveHomeGreeting(greetingHour.value)}，${userName.value}` : '记录每一局真实比分'
))

// 活动状态首次返回前不确定是否有进行中的对局，先占位，避免「来一局」闪成「继续对局」
const playCardLoading = computed(() => isLoggedIn.value && !activityStore.loaded && homeLoading.value)
const ongoingCard = computed(() => buildOngoingMatchCard(currentMatch.value, userStore.userInfo || {}))
const challengeCard = computed(() => {
  const card = resolveChallengeQuickCard(isLoggedIn.value ? activityStore.currentChallenge : null, {
    userId: userId.value,
    serverTime: activityStore.challengeServerTime
  })
  if (!card || !ongoingCard.value) return card
  // 约球已开局时由「我的对局」卡承接，不重复展示
  const sameMatch = card.matchId > 0 && card.matchId === ongoingCard.value.id
  const sameChallenge = card.challengeId > 0 && card.challengeId === Number(currentMatch.value?.challenge_id || 0)
  return sameMatch || sameChallenge ? null : card
})
const myRankingText = computed(() => formatHomeMyRanking(myRanking.value))
// 本人的当前对局已在开局卡展示，预览里不再重复
const liveMatches = computed(() => buildHomeLiveMatches(liveMatchList.value, {
  userId: userId.value,
  currentMatchId: ongoingCard.value?.id
}))

const getActivityIdentity = () => ({
  userId: userStore.userId,
  authGeneration: userStore.authGeneration
})

const resolveHomeResponseError = (result, fallbackMessage) => {
  if (result?.status === 'rejected') return result.reason
  if (result?.value?.success) return null
  return createRequestError({
    message: result?.value?.message || fallbackMessage,
    category: 'business'
  })
}

const loadData = async ({ force = false } = {}) => {
  if (homeLoading.value || (homeState.value.status === ASYNC_PAGE_STATUS.REFRESHING && !force)) return
  const currentGeneration = userStore.authGeneration
  const nextState = beginAsyncPageLoad(homeState.value, {
    authGeneration: currentGeneration,
    emptyData: []
  })
  const request = getAsyncPageRequest(nextState)
  const identityChanged = homeState.value.authGeneration !== currentGeneration
  if (identityChanged) {
    leaderboardTopThree.value = []
    myRanking.value = null
    topEventNews.value = null
    favoriteVenueRewardStatus.value = null
  }
  homeState.value = nextState
  homeSectionErrors.value = { leaderboard: null, eventNews: null, liveMatches: null }

  try {
    const nearbyVenueRequest = loadNearbyVenues({ force, passive: !force })
    const requests = [
      publicReadStore.loadLeaderboardSummary({ game_type: 3 }, {
        force,
        identity: isLoggedIn.value ? getActivityIdentity() : undefined
      }),
      publicReadStore.loadEventNews({ page: 1, page_size: 1 }, { force }),
      publicReadStore.loadLiveMatchPreview({ force })
    ]

    if (isLoggedIn.value) {
      requests.unshift(activityStore.fetch(getActivityIdentity(), { force, silent: true }))
      requests.push(userOverviewStore.fetch(getActivityIdentity(), { force, silent: true }))
    } else {
      favoriteVenueRewardStatus.value = null
    }

    const results = await Promise.allSettled(requests)
    if (homeState.value.authGeneration !== request.authGeneration || homeState.value.requestId !== request.requestId) return
    let resultIndex = isLoggedIn.value ? 1 : 0
    const leaderboardResult = results[resultIndex++]
    const eventNewsResult = results[resultIndex++]
    const liveMatchResult = results[resultIndex++]
    const leaderboardLoadError = resolveHomeResponseError(leaderboardResult, '加载排行榜失败')
    const eventNewsLoadError = resolveHomeResponseError(eventNewsResult, '加载赛事情报失败')
    const liveMatchLoadError = resolveHomeResponseError(liveMatchResult, '加载正在进行的对局失败')
    homeSectionErrors.value = {
      leaderboard: leaderboardLoadError,
      eventNews: eventNewsLoadError,
      liveMatches: liveMatchLoadError
    }

    if (!leaderboardLoadError) {
      const leaderboardRes = leaderboardResult.value
      leaderboardTopThree.value = (leaderboardRes.top_three || []).slice(0, 3)
      myRanking.value = leaderboardRes.my_ranking || null
    }

    if (!eventNewsLoadError && Array.isArray(eventNewsResult.value.list)) {
      const topItem = eventNewsResult.value.list[0] || null
      topEventNews.value = topItem ? normalizeSaiXunCard(topItem) : null
    }

    // 预览读取失败时保留上一次的列表，只在区块里提示失败
    if (!liveMatchLoadError) {
      liveMatchList.value = Array.isArray(liveMatchResult.value.list) ? liveMatchResult.value.list : []
    }

    if (isLoggedIn.value) {
      const overviewResult = results[resultIndex]
      const overview = overviewResult?.status === 'fulfilled' ? overviewResult.value : null
      favoriteVenueRewardStatus.value = overview?.favoriteVenueRewardStatus?.success
        ? overview.favoriteVenueRewardStatus
        : null
    }

    if (leaderboardLoadError && eventNewsLoadError) {
      homeState.value = rejectAsyncPageLoad(homeState.value, request, leaderboardLoadError)
      await nearbyVenueRequest
      return
    }

    homeState.value = resolveAsyncPageLoad(homeState.value, request, {
      data: {
        leaderboard: leaderboardLoadError ? null : leaderboardResult.value,
        eventNews: eventNewsLoadError ? null : eventNewsResult.value
      },
      isEmpty: () => false
    })
    await nearbyVenueRequest
  } catch (error) {
    homeState.value = rejectAsyncPageLoad(homeState.value, request, error)
  }
}

// passive：页面自动加载时不弹定位提示，结果只体现在区块内；用户点「重试」或下拉刷新才提示
const loadNearbyVenues = async ({ force = false, passive = false } = {}) => {
  if (nearbyVenueLoading.value) return
  if (!shouldReloadHomeNearbyVenues({ force, lastAttemptAt: nearbyVenueAttemptAt.value })) return

  nearbyVenueLoading.value = true
  nearbyVenueError.value = null
  nearbyVenueLocationIssue.value = ''
  nearbyVenueAttemptAt.value = Date.now()
  try {
    // 定位接口在部分平台不可用时会直接抛错，统一按「无法获取位置」展示，不暴露原始报错
    const locationResult = await getCurrentLocation({
      uniApi: uni,
      action: '查看附近球房',
      onPermissionDenied: (permission) => {
        if (passive) return
        uni.showToast({
          title: permission.reason === 'denied' ? '定位权限未开启，请开启后重试' : '获取定位失败，请稍后重试',
          icon: 'none'
        })
      }
    }).catch(() => null)
    if (!locationResult?.success) {
      nearbyVenueLocationIssue.value = locationResult?.permission?.reason === 'denied' ? 'denied' : 'unavailable'
      return
    }
    const params = buildHomeNearbyVenueParams(locationResult.location || {})
    if (!params) {
      nearbyVenueLocationIssue.value = 'unavailable'
      return
    }

    const res = await publicReadStore.loadNearbyVenues(params, { force })
    if (!res?.success) throw createRequestError({ message: res?.message || '加载附近球房失败', category: 'business' })
    nearbyVenues.value = Array.isArray(res.list) ? res.list.slice(0, params.limit) : []
  } catch (error) {
    nearbyVenueError.value = error
  } finally {
    nearbyVenueLoading.value = false
  }
}

const retryHomeData = () => loadData({ force: true })

const retryChallengeActivity = () => {
  activityStore.fetch(getActivityIdentity(), { force: true, silent: true })
}

const retryNearbyVenues = () => loadNearbyVenues({ force: true })

const handleHomeErrorAction = () => {
  if (homeState.value.error?.category === 'permission' || homeState.value.error?.category === 'not-found') {
    uni.navigateBack({ delta: 1 })
    return
  }
  retryHomeData()
}

const onRefresh = async () => {
  refreshing.value = true
  await loadData({ force: true })
  refreshing.value = false
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

const goToVenueDetail = (id) => {
  if (!id) return
  goTo(`/subPages/venue/detail?id=${id}`)
}

const formatVenueDistance = (meters) => formatHomeVenueDistance(meters)

const handleContinueCurrentMatch = () => {
  if (currentMatch.value) handleContinueMatch(currentMatch.value)
}

// 三种开局方式：扫描对手二维码走统一扫码流程（先扫后选球种与类型）
const handleStartMatch = () => {
  if (!isLoggedIn.value) {
    goLogin()
    return
  }
  uni.showActionSheet({
    itemList: ['扫描对手二维码', '出示我的二维码', '选择好友或最近对手'],
    success: ({ tapIndex }) => {
      if (tapIndex === 0) {
        startQuickScan()
        return
      }
      if (tapIndex === 1) {
        showMyMatchQRCode()
        return
      }
      goTo('/subPages/match/opponentSelector')
    }
  })
}

const showMyMatchQRCode = async () => {
  showMatchQrModal.value = true
  setMainTabBarVisible(false)
  await loadMyMatchQRCode()
}

const loadMyMatchQRCode = async () => {
  matchQrLoading.value = true
  matchQrFailed.value = false
  matchQrContent.value = ''
  try {
    const res = await getMatchQRCode()
    if (!res?.success || !res.qrcode_data) {
      matchQrLoading.value = false
      matchQrFailed.value = true
      return
    }
    matchQrContent.value = res.qrcode_data
    await nextTick()
    renderLocalQRCode({
      canvasId: 'home-match-invite-qrcode',
      content: matchQrContent.value,
      size: 240,
      uniApi: uni
    })
    matchQrLoading.value = false
  } catch (error) {
    matchQrLoading.value = false
    matchQrFailed.value = true
  }
}

const closeMatchQrModal = () => {
  showMatchQrModal.value = false
  matchQrLoading.value = false
  matchQrFailed.value = false
  matchQrContent.value = ''
  setMainTabBarVisible(true)
}

const navigateToPlayingMatch = (match) => {
  uni.navigateTo({
    url: buildPlayingRoute(match)
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

// 与观赛大厅一致：本人参与的进行中对局进入计分页，其他对局进入观战页
const handleLiveMatchTap = (match) => {
  if (shouldOpenPlayingForSpectatorMatch({ match, userId: userId.value, isLoggedIn: isLoggedIn.value })) {
    handleContinueMatch(match)
    return
  }
  goTo(`/subPages/match/matchDetail?match_id=${match.id}&mode=spectate`)
}

onShow(() => {
  greetingHour.value = new Date().getHours()
  loadData()
})

onUnmounted(() => {
  if (showMatchQrModal.value) setMainTabBarVisible(true)
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
