<template>
  <view class="pk-report-page" :class="{ 'dark-mode': isDarkMode }">
    <canvas canvas-id="pkReportPoster" class="poster-canvas"></canvas>

    <view v-if="pageStatus === 'loading'" class="loading-state">
      <uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
      <text class="loading-text">正在生成 PK 报表...</text>
    </view>

    <view v-else-if="pageStatus === 'error'" class="loading-state error-state">
      <uni-icons type="info-filled" size="40" color="#94a3b8"></uni-icons>
      <text class="loading-text">{{ loadErrorMessage || 'PK 报表加载失败' }}</text>
      <button class="retry-btn" @tap="loadData">
        <text>重新加载</text>
      </button>
    </view>

    <scroll-view v-else scroll-y class="report-scroll">
      <view class="hero-card">
        <text class="hero-kicker">真实线下交锋</text>
        <text class="hero-title">{{ heroViewModel.title }}</text>
        <text class="hero-meta">{{ heroViewModel.metaText }}</text>

        <view class="player-row">
          <view class="player-card">
            <image v-if="myAvatar" class="avatar" :src="myAvatar" mode="aspectFill" />
            <view v-else class="avatar placeholder"><text>我</text></view>
            <text class="player-name">你</text>
          </view>
          <view class="vs-block">
            <text class="vs-title">VS</text>
            <text class="vs-sub">PK 报表</text>
          </view>
          <view class="player-card">
            <image v-if="opponent.avatar" class="avatar" :src="opponent.avatar" mode="aspectFill" />
            <view v-else class="avatar placeholder"><text>{{ getAvatarText(opponent.name) }}</text></view>
            <text class="player-name">{{ opponent.name || '对手' }}</text>
          </view>
        </view>

        <view class="score-row">
          <text class="score-text">{{ heroViewModel.scoreText }}</text>
          <text class="score-desc">历史交锋胜场</text>
        </view>
      </view>

      <view class="section-card">
        <view class="section-header">
          <text class="section-title">核心证据</text>
          <text class="section-tip">一眼看懂这份 PK 报表为什么成立</text>
        </view>
        <view v-if="evidenceList.length > 0" class="evidence-list">
          <view v-for="(item, index) in evidenceList" :key="item" class="evidence-item">
            <text class="evidence-index">0{{ index + 1 }}</text>
            <text class="evidence-text">{{ item }}</text>
          </view>
        </view>
        <view v-else class="empty-state">
          <text>还没有足够的交锋样本，先约一场见真章。</text>
        </view>

        <view v-if="latestMatchSummary" class="summary-card compact">
          <text class="summary-title">最近一次交锋</text>
          <text class="summary-text">{{ latestMatchSummary }}</text>
        </view>
      </view>

      <view class="section-card">
        <view class="section-header">
          <text class="section-title">分享海报</text>
          <text class="section-tip">可保存到相册后发给好友</text>
        </view>
        <view v-if="posterPath" class="poster-preview">
          <image class="poster-image" :src="posterPath" mode="widthFix" />
        </view>
        <view v-else class="empty-state">
          <text>{{ posterLoading ? '海报生成中...' : posterEmptyText }}</text>
        </view>
      </view>

      <view class="section-card actions-card">
        <view class="section-header">
          <text class="section-title">下一步</text>
          <text class="section-tip">先表达结论，再决定要不要继续约战</text>
        </view>
        <view class="action-list">
          <button class="primary-btn" @tap="shareToPost">
            <text>发到动态</text>
          </button>
          <button class="secondary-btn" @tap="savePoster">
            <text>{{ posterLoading ? '生成海报中' : '保存海报' }}</text>
          </button>
          <button class="secondary-btn" @tap="goToH2H">
            <text>查看完整交锋</text>
          </button>
          <button class="secondary-btn" @tap="openPkInvite">
            <text>发 PK 邀约</text>
          </button>
        </view>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { useUserStore } from '@/store/user.js'
import { getH2HHistory, getH2HStats } from '@/api/match.js'
import { formatRelativeTime } from '@/utils/format.js'
import { generatePkReportPoster, savePosterToAlbum } from '@/utils/posterGenerator.js'
import {
  buildPkEvidenceList,
  buildPkPosterPayload,
  buildPkReportHero,
  resolvePkReportStatus
} from '@/utils/pk-report-view-model.js'

const themeStore = useThemeStore()
const userStore = useUserStore()

const isDarkMode = computed(() => themeStore.isDarkMode)
const posterLoading = ref(false)
const posterPath = ref('')
const myAvatar = ref('')
const opponentId = ref(0)
const statsLoaded = ref(false)
const historyLoaded = ref(false)
const loadErrorMessage = ref('')
const hasCoreError = ref(false)
const historyList = ref([])

const opponent = reactive({
  id: 0,
  name: '',
  avatar: ''
})

const statsData = reactive({
  total_matches: 0,
  my_wins: 0,
  opponent_wins: 0,
  win_rate: 0,
  avg_score_diff: 0,
  max_win_streak: 0
})

const heroViewModel = computed(() => buildPkReportHero({
  stats: statsData,
  opponent,
  history: historyList.value
}))

const evidenceList = computed(() => buildPkEvidenceList({
  stats: statsData,
  history: historyList.value
}))

const pageStatus = computed(() => resolvePkReportStatus({
  statsLoaded: statsLoaded.value,
  historyLoaded: historyLoaded.value,
  totalMatches: statsData.total_matches,
  hasError: hasCoreError.value
}))

const latestMatchSummary = computed(() => {
  const latestMatch = historyList.value[0]
  if (!latestMatch) {
    return ''
  }

  const scoreText = `${latestMatch.my_score || 0} : ${latestMatch.opponent_score || 0}`
  const resultText = latestMatch.result === 1 ? '你赢了' : latestMatch.result === 2 ? '你输了' : '打平了'
  const timeText = formatMatchDate(latestMatch.match_time)
  return `${timeText} · ${latestMatch.game_type_name || '台球'} · ${resultText} ${scoreText}`
})

const posterEmptyText = computed(() => {
  if (!statsData.total_matches) {
    return '至少完成一场真实交锋后再生成海报'
  }
  return '海报暂未生成'
})

const buildRequestParams = () => {
  const params = {}
  if (opponentId.value > 0) {
    params.opponent_id = opponentId.value
  } else if (opponent.name) {
    params.opponent_name = opponent.name
  }
  return params
}

const applyStats = (stats = {}) => {
  statsData.total_matches = Number(stats.total_matches || 0)
  statsData.my_wins = Number(stats.my_wins || 0)
  statsData.opponent_wins = Number(stats.opponent_wins || 0)
  statsData.win_rate = Number(stats.win_rate || 0)
  statsData.avg_score_diff = Number(stats.avg_score_diff || 0)
  statsData.max_win_streak = Number(stats.max_win_streak || 0)
}

const generatePoster = async () => {
  if (!statsData.total_matches) {
    posterPath.value = ''
    return
  }

  posterLoading.value = true
  try {
    const payload = buildPkPosterPayload({
      hero: heroViewModel.value,
      evidenceList: evidenceList.value,
      stats: statsData,
      userName: userStore.userInfo?.nickname || '我',
      opponentName: opponent.name || '对手',
      gameTypeLabel: historyList.value[0]?.game_type_name || '真实交锋数据'
    })
    posterPath.value = await generatePkReportPoster('pkReportPoster', payload)
  } catch (error) {
    posterPath.value = ''
  } finally {
    posterLoading.value = false
  }
}

const loadData = async () => {
  statsLoaded.value = false
  historyLoaded.value = false
  hasCoreError.value = false
  loadErrorMessage.value = ''
  posterPath.value = ''

  const params = buildRequestParams()
  const [statsResult, historyResult] = await Promise.allSettled([
    getH2HStats(params),
    getH2HHistory({ ...params, page: 1, page_size: 5 })
  ])

  if (statsResult.status === 'fulfilled' && statsResult.value?.stats) {
    statsLoaded.value = true
    const response = statsResult.value
    if (response.opponent) {
      opponent.id = Number(response.opponent.id || opponentId.value)
      opponent.name = response.opponent.name || opponent.name
      opponent.avatar = response.opponent.avatar || opponent.avatar
    }
    applyStats(response.stats)
  } else {
    hasCoreError.value = true
    loadErrorMessage.value = '交锋统计加载失败，请稍后重试'
  }

  if (historyResult.status === 'fulfilled') {
    historyLoaded.value = true
    historyList.value = Array.isArray(historyResult.value?.list) ? historyResult.value.list : []
  } else {
    hasCoreError.value = true
    loadErrorMessage.value = loadErrorMessage.value || '最近交锋加载失败，请稍后重试'
    historyList.value = []
  }

  if (pageStatus.value === 'ready') {
    await generatePoster()
  }
}

const formatMatchDate = (dateStr) => {
  if (!dateStr) return '未知时间'
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}月${date.getDate()}日 · ${formatRelativeTime(dateStr)}`
}

const getAvatarText = (name) => {
  if (!name) return '?'
  return name.slice(0, 1)
}

const shareToPost = () => {
  if (!statsData.total_matches) {
    uni.showToast({ title: '还没有足够的交锋数据', icon: 'none' })
    return
  }

  const evidence = evidenceList.value[0] || heroViewModel.value.metaText
  const content = `${heroViewModel.value.title}，历史交锋 ${heroViewModel.value.scoreText}。${evidence}。`
  uni.navigateTo({
    url: `/subPages/social/postCreate?content=${encodeURIComponent(content)}&post_type=1`
  })
}

const savePoster = async () => {
  if (!statsData.total_matches) {
    uni.showToast({ title: '至少完成一场交锋后再生成海报', icon: 'none' })
    return
  }
  if (posterLoading.value) {
    uni.showToast({ title: '海报生成中，请稍后', icon: 'none' })
    return
  }
  if (!posterPath.value) {
    await generatePoster()
  }
  if (!posterPath.value) {
    uni.showToast({ title: '海报生成失败', icon: 'none' })
    return
  }
  await savePosterToAlbum(posterPath.value)
}

const goToH2H = () => {
  const query = []
  if (opponent.id) {
    query.push(`opponent_id=${opponent.id}`)
  }
  if (opponent.name) {
    query.push(`opponent_name=${encodeURIComponent(opponent.name)}`)
  }
  uni.navigateTo({ url: `/subPages/user/h2hRecord?${query.join('&')}` })
}

const openPkInvite = () => {
  uni.navigateTo({ url: '/subPages/social/challenges' })
  uni.showToast({ title: '可在 PK 记录页继续发起邀约', icon: 'none' })
}

onLoad((options) => {
  opponentId.value = Number(options.opponent_id || 0)
  opponent.id = opponentId.value
  opponent.name = options.opponent_name ? decodeURIComponent(options.opponent_name) : ''
  opponent.avatar = options.opponent_avatar ? decodeURIComponent(options.opponent_avatar) : ''
  myAvatar.value = userStore.userInfo?.avatar || ''
})

onShow(() => {
  themeStore.syncTheme()
  themeStore.applyNavigationBarTheme()
  loadData()
})
</script>

<style lang="scss" scoped>
@import './pkReport.scss';
</style>
