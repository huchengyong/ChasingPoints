import { resolveFavoriteVenueRewardTaskCard } from './favorite-venue-reward.js'

export const formatDuration = (durationSeconds = 0) => {
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

const toSafeNumber = (value) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const toSafeText = (value, fallback = '') => {
  return typeof value === 'string' && value.trim() ? value.trim() : fallback
}

export const resolveGuestHeroCopy = () => ({
  eyebrow: '个人竞技主页',
  title: '登录后，解锁你的个人竞技主页',
  description: '记录比分、查看段位变化、沉淀每一场对局，挑战和消息也会集中在这里。',
  primaryActionText: '登录/注册',
  secondaryActionText: '登录后发起 PK'
})

export const resolveSectionTitles = () => ({
  stats: '核心指标',
  quickActions: '常用入口',
  secondaryServices: '竞技服务',
  settings: '设置与支持'
})

export const resolveUserHomepageMode = ({
  isLoggedIn = false,
  isLoading = false,
  currentMatch = null,
  hasCurrentMatch = false,
  totalMatches = 0,
  hasRecentMatch = false
} = {}) => {
  if (!isLoggedIn) return 'guest'
  if (isLoading) return 'loading'

  if (currentMatch || hasCurrentMatch) {
    return currentMatch?.viewer_role === 'referee' ? 'ongoing-referee' : 'ongoing-player'
  }

  return toSafeNumber(totalMatches) > 0 || hasRecentMatch ? 'idle' : 'newcomer'
}

export const resolvePrimaryAction = (mode) => {
  if (mode === 'ongoing-player' || mode === 'ongoing-referee') return 'continue'
  if (mode === 'guest') return 'login'
  if (mode === 'loading') return ''
  return 'start'
}

const resolveMatchParticipants = (match = {}, currentUser = {}) => {
  const safeCurrentUser = currentUser || {}
  const viewerRole = match.viewer_role || 'player1'
  const userId = toSafeNumber(safeCurrentUser.id)
  const userName = toSafeText(safeCurrentUser.nickname, '我')
  const userAvatar = toSafeText(safeCurrentUser.avatar)
  let player1Id = toSafeNumber(match.player1_id)
  let player1Name = toSafeText(match.player1_name)
  let player1Avatar = toSafeText(match.player1_avatar)
  let player2Id = toSafeNumber(match.player2_id)
  let player2Name = toSafeText(match.player2_name)
  let player2Avatar = toSafeText(match.player2_avatar)

  if (!player1Name && viewerRole === 'player1') {
    player1Id = player1Id || userId
    player1Name = userName
    player1Avatar = player1Avatar || userAvatar
  }
  if (!player2Name && viewerRole === 'player2') {
    player2Id = player2Id || userId
    player2Name = userName
    player2Avatar = player2Avatar || userAvatar
  }
  if (!player1Name && viewerRole === 'player2') {
    player1Id = player1Id || toSafeNumber(match.opponent_id)
    player1Name = toSafeText(match.opponent_name, '玩家1')
    player1Avatar = player1Avatar || toSafeText(match.opponent_avatar)
  }
  if (!player2Name && viewerRole !== 'player2') {
    player2Id = player2Id || toSafeNumber(match.opponent_id)
    player2Name = toSafeText(match.opponent_name, '玩家2')
    player2Avatar = player2Avatar || toSafeText(match.opponent_avatar)
  }

  player1Name = player1Name || '玩家1'
  player2Name = player2Name || '玩家2'

  let player1Score = toSafeNumber(match.my_score ?? match.player1_score)
  let player2Score = toSafeNumber(match.opponent_score ?? match.player2_score)
  if (viewerRole === 'player2') {
    player1Score = toSafeNumber(match.opponent_score ?? match.player1_score)
    player2Score = toSafeNumber(match.my_score ?? match.player2_score)
  }

  return [
    { id: player1Id, name: player1Name, avatar: player1Avatar, score: player1Score },
    { id: player2Id, name: player2Name, avatar: player2Avatar, score: player2Score }
  ]
}

export const resolveStatusCardContent = ({ mode, currentMatch = null, currentUser = {} } = {}) => {
  if ((mode === 'ongoing-player' || mode === 'ongoing-referee') && currentMatch) {
    const players = resolveMatchParticipants(currentMatch, currentUser)
    const isReferee = mode === 'ongoing-referee'

    return {
      type: 'ongoing',
      eyebrow: '当前对局',
      title: isReferee ? '你正在担任本场裁判' : '继续当前对局',
      statusText: isReferee ? '裁判中' : '进行中',
      description: `${toSafeText(currentMatch.game_type_name, '台球对局')} · ${formatDuration(currentMatch.duration_seconds)}`,
      players,
      scoreText: `${players[0].score} : ${players[1].score}`,
      action: 'continue',
      actionText: isReferee ? '进入裁判记分' : '继续记分',
      secondaryAction: '',
      secondaryActionText: ''
    }
  }

  if (mode === 'loading') {
    return {
      type: 'loading',
      loading: true,
      eyebrow: '正在同步',
      title: '正在加载你的竞技状态',
      statusText: '',
      description: '段位、当前对局和核心指标准备好后会一起展示。',
      players: [],
      scoreText: '',
      action: '',
      actionText: '',
      secondaryAction: '',
      secondaryActionText: ''
    }
  }

  if (mode === 'guest') {
    return {
      type: 'guest',
      eyebrow: '登录后解锁',
      title: '登录后，解锁你的个人竞技主页',
      statusText: '',
      description: '登录后查看进行中的对局、个人战绩、段位变化和待处理事项。',
      players: [],
      scoreText: '',
      action: 'login',
      actionText: '登录/注册',
      secondaryAction: 'start_pk',
      secondaryActionText: '登录后发起 PK'
    }
  }

  const isNewcomer = mode === 'newcomer'
  return {
    type: 'empty',
    eyebrow: '准备开杆',
    title: isNewcomer ? '从第一场比赛开始' : '今天还没开杆',
    statusText: '待开局',
    description: isNewcomer
      ? '发起一场 PK，建立你的第一份竞技记录。'
      : '来打一场找回节奏，继续积累你的竞技状态。',
    players: [],
    scoreText: '',
    action: 'start',
    actionText: '发起 PK',
    secondaryAction: 'show_pk_code',
    secondaryActionText: '出示二维码'
  }
}

export const resolveStatusActionVisibility = ({ statusCard = {} } = {}) => {
  return {
    showPrimary: Boolean(statusCard.actionText),
    showSecondary: Boolean(statusCard.secondaryActionText)
  }
}

export const resolveCoreMetrics = ({ stats = {}, rankInfo = null, isLoading = false } = {}) => {
  const safeStats = stats || {}
  const loadingValue = isLoading ? '—' : null
  const winRate = toSafeNumber(safeStats.winRate ?? safeStats.win_rate)
  const wins = toSafeNumber(safeStats.wins)
  const maxStreak = toSafeNumber(safeStats.maxStreak ?? safeStats.max_win_streak)
  const rankScore = toSafeNumber(rankInfo?.rankScore ?? rankInfo?.rank_score)

  return [
    { key: 'win-rate', label: '胜率', value: loadingValue ?? `${winRate}%`, loading: isLoading },
    { key: 'wins', label: '胜场', value: loadingValue ?? wins, loading: isLoading },
    { key: 'max-streak', label: '最高连胜', value: loadingValue ?? maxStreak, loading: isLoading },
    { key: 'rank-score', label: '段位分', value: loadingValue ?? rankScore, loading: isLoading }
  ]
}

const parseMemberExpiresAt = (value) => {
  const expiresAtText = toSafeText(value)
  if (!expiresAtText) return null

  const normalized = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(expiresAtText)
    ? `${expiresAtText.replace(' ', 'T')}+08:00`
    : expiresAtText
  const parsed = new Date(normalized)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const RANK_AVATAR_FRAME_PATHS = {
  1: '/static/images/avatar-frames/frame_bronze.png',
  2: '/static/images/avatar-frames/frame_silver.png',
  3: '/static/images/avatar-frames/frame_gold.png',
  4: '/static/images/avatar-frames/frame_platinum.png',
  5: '/static/images/avatar-frames/frame_diamond.png',
  6: '/static/images/avatar-frames/frame_king.png'
}

export const resolveRankAvatarFramePath = (level) => {
  return RANK_AVATAR_FRAME_PATHS[toSafeNumber(level)] || ''
}

export const resolveMemberRankAvatarFrame = ({
  isLoggedIn = false,
  memberStatus = null,
  rankInfo = null,
  rankLoading = false,
  now = new Date()
} = {}) => {
  if (!isLoggedIn || rankLoading || !rankInfo) return ''

  const expiresAt = parseMemberExpiresAt(memberStatus?.member_expires_at)
  if (!memberStatus?.is_active || !expiresAt || expiresAt.getTime() <= now.getTime()) return ''

  return resolveRankAvatarFramePath(rankInfo.level)
}

export const resolveMemberHeroStrip = (memberStatus = {}, now = new Date(), memberLevelText = '') => {
  const expiresAtText = toSafeText(memberStatus?.member_expires_at)
  if (!expiresAtText) {
    return { visible: false, state: 'none', statusText: '', title: '', description: '', levelText: '' }
  }

  const expiresAt = parseMemberExpiresAt(expiresAtText)
  const isActive = Boolean(memberStatus?.is_active) && (!expiresAt || expiresAt.getTime() > now.getTime())
  const levelText = toSafeText(memberLevelText) || `Lv${Math.max(1, toSafeNumber(memberStatus?.growth_level) || 1)}`

  return {
    visible: true,
    state: isActive ? 'active' : 'expired',
    statusText: isActive ? '会员中' : '已到期',
    title: isActive ? '会员权益已生效' : '会员权益已到期',
    description: isActive ? `有效期至 ${expiresAtText}` : `上次有效期至 ${expiresAtText}`,
    levelText
  }
}

export const resolveCompactRewardEntry = (rewardStatus = {}) => {
  const card = resolveFavoriteVenueRewardTaskCard(rewardStatus || {})
  return {
    visible: card.visible,
    state: toSafeText(rewardStatus?.status, 'not_started'),
    title: card.title,
    description: card.description,
    statusText: card.statusText,
    action: card.actionText ? 'favorite-venue' : '',
    actionText: card.actionText
  }
}

export const resolveUserHomepageModel = ({
  isLoggedIn = false,
  isLoading = false,
  currentMatch = null,
  currentUser = {},
  stats = {},
  rankInfo = null,
  memberStatus = null,
  memberLevelText = '',
  rewardStatus = null,
  now = new Date()
} = {}) => {
  const mode = resolveUserHomepageMode({
    isLoggedIn,
    isLoading,
    currentMatch,
    totalMatches: stats?.totalMatches ?? stats?.total_matches
  })

  return {
    mode,
    memberHeroStrip: resolveMemberHeroStrip(memberStatus || {}, now, memberLevelText),
    statusCard: resolveStatusCardContent({ mode, currentMatch, currentUser }),
    metrics: resolveCoreMetrics({ stats, rankInfo, isLoading: mode === 'loading' }),
    rewardEntry: resolveCompactRewardEntry(rewardStatus || {})
  }
}

const normalizeRankDisplay = (rank = null, fallbackGameType = 0) => {
  if (!rank) return null

  return {
    gameType: rank.gameType || rank.game_type || fallbackGameType || 0,
    name: rank.name || '未定级',
    level: Number(rank.level || 0),
    rankScore: Number(rank.rankScore ?? rank.rank_score ?? 0)
  }
}

export const resolveHighestRankDisplay = (rankList = [], fallbackRank = null) => {
  const candidates = rankList
    .map((item) => normalizeRankDisplay(item))
    .filter(Boolean)

  const highest = candidates.reduce((best, current) => {
    if (!best) return current
    if (current.level !== best.level) return current.level > best.level ? current : best
    if (current.rankScore !== best.rankScore) return current.rankScore > best.rankScore ? current : best
    return best
  }, null)

  return highest || normalizeRankDisplay(fallbackRank)
}
