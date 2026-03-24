<template>
  <view class="pk-report-page" :class="{ 'dark-mode': isDarkMode }">
    <canvas canvas-id="pkReportPoster" class="poster-canvas"></canvas>

    <view v-if="loading" class="loading-state">
      <uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
      <text class="loading-text">正在生成PK报表...</text>
    </view>

    <scroll-view v-else scroll-y class="report-scroll">
      <view class="hero-card">
        <text class="hero-kicker">真实数据对比</text>
        <view class="player-row">
          <view class="player-card">
            <image v-if="myAvatar" class="avatar" :src="myAvatar" mode="aspectFill" />
            <view v-else class="avatar placeholder"><text>我</text></view>
            <text class="player-name">你</text>
          </view>
          <view class="vs-block">
            <text class="vs-title">VS</text>
            <text class="vs-sub">PK报表</text>
          </view>
          <view class="player-card">
            <image v-if="opponent.avatar" class="avatar" :src="opponent.avatar" mode="aspectFill" />
            <view v-else class="avatar placeholder"><text>{{ getAvatarText(opponent.name) }}</text></view>
            <text class="player-name">{{ opponent.name || '对手' }}</text>
          </view>
        </view>

        <view class="score-row">
          <text class="score-text">{{ stats.myWins }} : {{ stats.opponentWins }}</text>
          <text class="score-desc">历史交锋胜场</text>
        </view>

        <view class="summary-card">
          <text class="summary-title">系统结论</text>
          <text class="summary-text">{{ summaryText }}</text>
        </view>
      </view>

      <view class="section-card">
        <view class="section-header">
          <text class="section-title">核心数据</text>
          <text class="section-tip">只统计真实线下对局</text>
        </view>
        <view class="stats-grid">
          <view class="stat-item">
            <text class="label">总场次</text>
            <text class="value">{{ stats.totalMatches }}</text>
          </view>
          <view class="stat-item">
            <text class="label">我的胜率</text>
            <text class="value">{{ formatPercent(stats.winRate) }}</text>
          </view>
          <view class="stat-item">
            <text class="label">平均分差</text>
            <text class="value">{{ formatDiff(stats.avgScoreDiff) }}</text>
          </view>
          <view class="stat-item">
            <text class="label">最长连胜</text>
            <text class="value">{{ stats.maxWinStreak }}</text>
          </view>
        </view>
      </view>

      <view class="section-card">
        <view class="section-header">
          <text class="section-title">最近交锋</text>
          <text class="section-tip">最近 {{ recentMatches.length }} 场</text>
        </view>
        <view v-if="recentMatches.length > 0" class="trend-list">
          <view v-for="item in recentMatches" :key="item.id" class="trend-item">
            <view class="trend-left">
              <text class="trend-date">{{ formatMatchDate(item.match_time) }}</text>
              <text class="trend-type">{{ item.game_type_name || '台球' }}</text>
            </view>
            <view class="trend-right">
              <text class="trend-score">{{ item.my_score }} - {{ item.opponent_score }}</text>
              <text class="trend-result" :class="getResultClass(item.result)">{{ getResultText(item.result) }}</text>
            </view>
          </view>
        </view>
        <view v-else class="empty-state">
          <text>还没有足够的交锋数据生成趋势。</text>
        </view>
      </view>

      <view class="section-card">
        <view class="section-header">
          <text class="section-title">分享海报</text>
          <text class="section-tip">可保存到相册后发送给好友</text>
        </view>
        <view v-if="posterPath" class="poster-preview">
          <image class="poster-image" :src="posterPath" mode="widthFix" />
        </view>
        <view v-else class="empty-state">
          <text>{{ posterLoading ? '海报生成中...' : '海报暂未生成' }}</text>
        </view>
      </view>

      <view class="section-card actions-card">
        <view class="section-header">
          <text class="section-title">下一步</text>
          <text class="section-tip">社交传播，不影响真实战绩</text>
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
            <text>发PK邀约</text>
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

const themeStore = useThemeStore()
const userStore = useUserStore()

const isDarkMode = computed(() => themeStore.isDarkMode)
const loading = ref(true)
const posterLoading = ref(false)
const posterPath = ref('')
const myAvatar = ref('')
const opponentId = ref(0)
const recentMatches = ref([])

const opponent = reactive({
  id: 0,
  name: '',
  avatar: ''
})

const stats = reactive({
  totalMatches: 0,
  myWins: 0,
  opponentWins: 0,
  winRate: 0,
  avgScoreDiff: 0,
  maxWinStreak: 0
})

const summaryText = computed(() => {
  if (!stats.totalMatches) {
    return `你和${opponent.name || '对手'}还没有形成真实交锋数据，先在线下打完一场再来生成更有说服力的报表。`
  }

  if (stats.myWins > stats.opponentWins) {
    return `你在与${opponent.name || '对手'}的 ${stats.totalMatches} 场真实交锋里占优，当前胜率 ${formatPercent(stats.winRate)}。`
  }

  if (stats.myWins < stats.opponentWins) {
    return `${opponent.name || '对手'}在历史交锋里暂时领先，你当前胜率 ${formatPercent(stats.winRate)}，适合发起一次线下复仇局。`
  }

  return `你和${opponent.name || '对手'}目前平分秋色，历史交锋 ${stats.myWins} 比 ${stats.opponentWins}。`
})

const loadData = async () => {
  loading.value = true
  try {
    const params = {}
    if (opponentId.value > 0) {
      params.opponent_id = opponentId.value
    } else if (opponent.name) {
      params.opponent_name = opponent.name
    }

    const [statsRes, historyRes] = await Promise.all([
      getH2HStats(params).catch(() => ({})),
      getH2HHistory({ ...params, page: 1, page_size: 5 }).catch(() => ({ list: [] }))
    ])

    if (statsRes.opponent) {
      opponent.id = statsRes.opponent.id || opponentId.value
      opponent.name = statsRes.opponent.name || opponent.name
      opponent.avatar = statsRes.opponent.avatar || opponent.avatar
    }

    if (statsRes.stats) {
      stats.totalMatches = statsRes.stats.total_matches || 0
      stats.myWins = statsRes.stats.my_wins || 0
      stats.opponentWins = statsRes.stats.opponent_wins || 0
      stats.winRate = statsRes.stats.win_rate || 0
      stats.avgScoreDiff = statsRes.stats.avg_score_diff || 0
      stats.maxWinStreak = statsRes.stats.max_win_streak || 0
    }

    recentMatches.value = (historyRes.list || []).map((item) => ({
      ...item,
      relative_time: formatRelativeTime(item.match_time)
    }))

    await generatePoster()
  } finally {
    loading.value = false
  }
}

const generatePoster = async () => {
  posterLoading.value = true
  try {
    posterPath.value = await generatePkReportPoster('pkReportPoster', {
      myName: userStore.userInfo?.nickname || '我',
      opponentName: opponent.name || '对手',
      myWins: stats.myWins,
      opponentWins: stats.opponentWins,
      totalMatches: stats.totalMatches,
      winRateLabel: formatPercent(stats.winRate),
      avgScoreDiffLabel: formatDiff(stats.avgScoreDiff),
      maxWinStreak: stats.maxWinStreak,
      summaryText: summaryText.value,
      gameTypeLabel: recentMatches.value[0]?.game_type_name || '真实交锋数据'
    })
  } catch (error) {
    posterPath.value = ''
  } finally {
    posterLoading.value = false
  }
}

const formatPercent = (value) => `${Number(value || 0).toFixed(2)}%`
const formatDiff = (value) => `${value > 0 ? '+' : ''}${Number(value || 0).toFixed(1)}`

const formatMatchDate = (dateStr) => {
  if (!dateStr) return '未知时间'
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}月${date.getDate()}日 · ${formatRelativeTime(dateStr)}`
}

const getResultClass = (result) => {
  if (result === 1) return 'win'
  if (result === 2) return 'lose'
  return 'draw'
}

const getResultText = (result) => {
  if (result === 1) return '胜'
  if (result === 2) return '负'
  return '平'
}

const getAvatarText = (name) => {
  if (!name) return '?'
  return name.slice(0, 1)
}

const shareToPost = () => {
  const content = `我和${opponent.name || '对手'}的 PK 报表：历史交锋 ${stats.myWins} : ${stats.opponentWins}，我的胜率 ${formatPercent(stats.winRate)}。这份数据只统计真实线下对局。`
  uni.navigateTo({
    url: `/subPages/social/postCreate?content=${encodeURIComponent(content)}&post_type=1`
  })
}

const savePoster = async () => {
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
  uni.showToast({ title: '可在PK记录页继续发起邀约', icon: 'none' })
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
