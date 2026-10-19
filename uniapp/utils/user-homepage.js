import { resolveFavoriteVenueRewardTaskCard } from './favorite-venue-reward.js'
import { GAME_TYPE_TABS } from './game-types.js'

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
    statusText: '',
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

// 生涯数据统计全部球种；段位分随球种切换，只放在段位卡里，不在这里重复。
export const resolveCoreMetrics = ({ stats = {}, isLoading = false } = {}) => {
  const safeStats = stats || {}
  const loadingValue = isLoading ? '—' : null
  const totalMatches = toSafeNumber(safeStats.totalMatches ?? safeStats.total_matches)
  const wins = toSafeNumber(safeStats.wins)
  const winRate = toSafeNumber(safeStats.winRate ?? safeStats.win_rate)
  const maxStreak = toSafeNumber(safeStats.maxStreak ?? safeStats.max_win_streak)

  return [
    { key: 'total', label: '场次', value: loadingValue ?? totalMatches },
    { key: 'wins', label: '胜场', value: loadingValue ?? wins },
    { key: 'win-rate', label: '胜率', value: loadingValue ?? `${winRate}%` },
    { key: 'max-streak', label: '最高连胜', value: loadingValue ?? maxStreak }
  ]
}

const DEFAULT_RANK_ICON = '/static/images/ranks/rank_bronze.png'

// 段位卡：next_score 是下一段位的起始分；6 级（王者）为最高段位。
export const resolveRankCardView = (rankInfo = null) => {
  if (!rankInfo) {
    return { ranked: false, name: '未定级', icon: DEFAULT_RANK_ICON, score: 0, progress: 0, hint: '完成首场排位赛后开始定级' }
  }

  const isMaxLevel = toSafeNumber(rankInfo.level) >= 6
  const score = toSafeNumber(rankInfo.rank_score)
  const nextScore = toSafeNumber(rankInfo.next_score)
  const progress = isMaxLevel ? 100 : Math.min(100, Math.max(0, toSafeNumber(rankInfo.progress)))
  let hint = `晋级进度 ${progress}%`
  if (isMaxLevel) {
    hint = '已达到最高段位'
  } else if (nextScore > 0) {
    hint = `距${toSafeText(rankInfo.next_name, '下一段位')}还差 ${Math.max(0, nextScore - score)} 分`
  }

  return {
    ranked: true,
    name: toSafeText(rankInfo.name, '未定级'),
    icon: toSafeText(rankInfo.icon, DEFAULT_RANK_ICON),
    score,
    progress,
    hint
  }
}

const countRankedMatches = (rankInfo) => (
  toSafeNumber(rankInfo?.total_wins) + toSafeNumber(rankInfo?.total_losses)
)

// 段位卡球种：优先用户设的常用球种；没设时取排位场次最多的球种，同场次比排位分，再按球种固定顺序。
// 只在常用球种与打过排位的球种之间切换，不足两个时不显示切换。
export const resolveRankGameTypeView = ({ preferredGameType = 0, rankInfoMap = {} } = {}) => {
  const safeMap = rankInfoMap || {}
  const played = GAME_TYPE_TABS.filter((item) => countRankedMatches(safeMap[item.value]) > 0)
  const preferred = GAME_TYPE_TABS.find((item) => item.value === Number(preferredGameType))
  const mostPlayed = played.reduce((best, item) => {
    if (!best) return item
    const matchDiff = countRankedMatches(safeMap[item.value]) - countRankedMatches(safeMap[best.value])
    if (matchDiff !== 0) return matchDiff > 0 ? item : best
    return toSafeNumber(safeMap[item.value]?.rank_score) > toSafeNumber(safeMap[best.value]?.rank_score) ? item : best
  }, null)
  const primary = preferred || mostPlayed || GAME_TYPE_TABS[0]
  const tabs = GAME_TYPE_TABS.filter((item) => item === primary || played.includes(item))

  return {
    primaryGameType: primary.value,
    primaryLabel: primary.label,
    tabs: tabs.length > 1 ? tabs : [],
    needsPick: !preferred && played.length === 0
  }
}

export const resolveReputationChip = (reputation = null, isLoading = false) => {
  const statusText = toSafeText(reputation?.status_text)
  if (!statusText) {
    return { text: isLoading ? '信誉同步中' : '信誉暂不可用', restricted: false }
  }
  const restricted = reputation.status === 'restricted'
  return { text: restricted ? statusText : `信誉${statusText}`, restricted }
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

// 会员中心入口的右侧状态：从未开通时不提示，避免把入口做成推销位。
export const resolveMemberEntry = (memberStatus = {}, now = new Date()) => {
  const expiresAtText = toSafeText(memberStatus?.member_expires_at)
  if (!expiresAtText) return { state: 'none', text: '' }

  const expiresAt = parseMemberExpiresAt(expiresAtText)
  const isActive = Boolean(memberStatus?.is_active) && (!expiresAt || expiresAt.getTime() > now.getTime())
  return isActive
    ? { state: 'active', text: `${expiresAtText.slice(0, 10)} 到期` }
    : { state: 'expired', text: '已到期' }
}

export const resolveCompactRewardEntry = (rewardStatus = {}) => {
  const card = resolveFavoriteVenueRewardTaskCard(rewardStatus || {})
  const status = toSafeText(rewardStatus?.status, 'not_started')
  const floatable = card.visible && (status === 'not_started' || status === 'rejected')
  const inline = card.visible && status === 'pending_review'

  let { title, description } = card
  if (status === 'not_started') {
    title = '免费VIP限时赠送'
    description = '绑定你常去的球馆，可免费领取1个月VIP'
  }

  return {
    visible: card.visible,
    floatable,
    inline,
    state: status,
    title,
    description,
    statusText: card.statusText,
    action: floatable ? 'favorite-venue' : '',
    actionText: floatable ? card.actionText : ''
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
