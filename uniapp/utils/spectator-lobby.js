import { formatRelativeTime } from './format.js'
import { GAME_TYPE_LABEL_MAP, getGameTypeLabel } from './game-types.js'

// 观赛列表标签：「好友」看好友与自己的对局（含进行中和已结束），「进行中/已结束」看公开大厅。
export const SPECTATOR_LOBBY_TABS = [
  { value: 'friends', label: '好友', scope: 'friends', status: 0, requiresLogin: true },
  { value: 'live', label: '进行中', scope: 'hall', status: 1, requiresLogin: false },
  { value: 'finished', label: '已结束', scope: 'hall', status: 2, requiresLogin: false }
]

export const resolveSpectatorLobbyTabs = (isLoggedIn = false) => (
  SPECTATOR_LOBBY_TABS.filter((tab) => isLoggedIn || !tab.requiresLogin)
)

export const resolveSpectatorLobbyQuery = (tabValue = 'live') => {
  const tab = SPECTATOR_LOBBY_TABS.find((item) => item.value === tabValue) || SPECTATOR_LOBBY_TABS[1]
  return { scope: tab.scope, status: tab.status }
}

const MATCH_MODE_TEXT = { ranked: '排位', practice: '练习' }

export const buildSpectatorMatchListParams = ({
  scope = 'hall',
  status = 1,
  gameType = 0,
  page = 1,
  pageSize = 20
} = {}) => {
  const resolvedScope = scope === 'friends' ? 'friends' : 'hall'
  const params = {
    scope: resolvedScope,
    page,
    page_size: pageSize
  }

  if (resolvedScope === 'hall') {
    params.status = Number(status) === 2 ? 2 : 1
  }

  if (Number(gameType) > 0) {
    params.game_type = Number(gameType)
  }

  return params
}

export const formatSpectatorDuration = (durationSeconds) => {
  if (!durationSeconds || durationSeconds < 0) return '0分'

  const totalMinutes = Math.floor(durationSeconds / 60)
  if (totalMinutes < 1) return `${durationSeconds}秒`
  if (totalMinutes < 60) return `${totalMinutes}分`

  const hours = Math.floor(totalMinutes / 60)
  const remainMinutes = totalMinutes % 60
  if (hours < 24) return remainMinutes > 0 ? `${hours}小时${remainMinutes}分` : `${hours}小时`

  const days = Math.floor(hours / 24)
  const remainHours = hours % 24
  return remainHours > 0 ? `${days}天${remainHours}小时` : `${days}天`
}

export const getSpectatorMatchStatusText = ({
  status = 1,
  durationSeconds = 0
} = {}) => {
  if (Number(status) === 2) return '已结束'
  return `进行中 ${formatSpectatorDuration(durationSeconds)}`
}

// 列表卡片：双方各占一行；已结束时按比分标记胜负，平局与进行中不标记。
export const normalizeSpectatorMatch = (match = {}, { userId = 0 } = {}) => {
  const viewerId = Number(userId) || 0
  const isLive = Number(match.status) !== 2
  const scores = [Number(match.player1_score) || 0, Number(match.player2_score) || 0]
  const players = [1, 2].map((side, index) => {
    const id = Number(match[`player${side}_id`]) || 0
    const score = scores[index]
    const otherScore = scores[1 - index]
    let outcome = ''
    if (!isLive && score !== otherScore) outcome = score > otherScore ? 'winner' : 'loser'
    return {
      id,
      name: match[`player${side}_name`] || `选手${side}`,
      avatar: match[`player${side}_avatar`] || '',
      score,
      isMe: viewerId > 0 && id === viewerId,
      outcome
    }
  })
  const endedAgo = !isLive && match.end_time ? formatRelativeTime(match.end_time) : ''

  return {
    ...match,
    isLive,
    isMine: players.some((player) => player.isMe),
    metaText: [match.game_type_name || getGameTypeLabel(match.game_type), MATCH_MODE_TEXT[match.match_mode]].filter(Boolean).join(' · '),
    statusText: endedAgo
      ? `${endedAgo}结束`
      : getSpectatorMatchStatusText({ status: match.status || 1, durationSeconds: match.duration_seconds || 0 }),
    players
  }
}

export const HOME_LIVE_MATCH_LIMIT = 3

// 首页「正在进行」：去掉本人的当前对局（已在开局卡展示）再取前 3 场，卡片形态与观赛列表一致。
export const buildHomeLiveMatches = (list = [], { userId = 0, currentMatchId = 0 } = {}) => {
  const currentId = Number(currentMatchId) || 0
  return (Array.isArray(list) ? list : [])
    .filter((match) => !currentId || Number(match.id) !== currentId)
    .slice(0, HOME_LIVE_MATCH_LIMIT)
    .map((match) => normalizeSpectatorMatch(match, { userId }))
}

// 好友标签混合两种状态，按「进行中 / 已结束」分组；大厅标签只有一种状态，不加分组标题。
export const buildSpectatorSections = (list = [], tabValue = 'live') => {
  if (tabValue !== 'friends') {
    return list.length ? [{ key: tabValue, title: '', items: list }] : []
  }
  return [
    { key: 'live', title: '进行中', items: list.filter((item) => item.isLive) },
    { key: 'finished', title: '已结束', items: list.filter((item) => !item.isLive) }
  ].filter((section) => section.items.length > 0)
}

export const getSpectatorEmptyState = ({
  scope = 'hall',
  status = 1,
  gameType = 0
} = {}) => {
  const gameTypeLabel = GAME_TYPE_LABEL_MAP[Number(gameType)] || ''

  if (gameTypeLabel) {
    return {
      title: `暂无${gameTypeLabel}${scope === 'friends' ? '好友' : ''}对局`,
      subtitle: scope === 'friends'
        ? '好友参与该球种的对局会集中展示在这里'
        : '该球种有新的公开对局后会出现在这里'
    }
  }

  if (scope === 'friends') {
    return {
      title: '暂无好友对局',
      subtitle: '好友参与的进行中和已结束对局会展示在这里'
    }
  }

  if (Number(status) === 2) {
    return {
      title: '暂无已结束对局',
      subtitle: '有新的完赛记录后会出现在这里'
    }
  }

  return {
    title: '暂无正在进行的对局',
    subtitle: '公开对局开打后会出现在这里'
  }
}

export const shouldOpenPlayingForSpectatorMatch = ({
  match = {},
  userId = 0,
  isLoggedIn = false
} = {}) => {
  if (!isLoggedIn || !userId || Number(match.status) !== 1) return false
  return Number(match.player1_id) === Number(userId) || Number(match.player2_id) === Number(userId)
}

const appendPlayerQuery = (query, keyPrefix, player = {}) => {
  if (Number(player.id) > 0) {
    query.push(`${keyPrefix}_id=${Number(player.id)}`)
  }
  query.push(`${keyPrefix}_name=${encodeURIComponent(player.name || '对手')}`)
}

export const buildSpectatorFinishedMatchDetailUrl = ({
  match = {},
  userId = 0
} = {}) => {
  if (Number(match.status) !== 2) return ''

  const currentUserId = Number(userId) || 0
  const player1 = {
    id: Number(match.player1_id || 0),
    name: match.player1_name || '玩家1',
    avatar: match.player1_avatar || ''
  }
  const player2 = {
    id: Number(match.player2_id || 0),
    name: match.player2_name || '玩家2',
    avatar: match.player2_avatar || ''
  }

  const query = []
  if (currentUserId > 0 && player1.id === currentUserId) {
    appendPlayerQuery(query, 'opponent', player2)
    return `/subPages/user/h2hRecord?${query.join('&')}`
  }

  if (currentUserId > 0 && player2.id === currentUserId) {
    appendPlayerQuery(query, 'opponent', player1)
    return `/subPages/user/h2hRecord?${query.join('&')}`
  }

  if (player1.id <= 0) return ''

  query.push(`target_user_id=${player1.id}`)
  query.push(`target_name=${encodeURIComponent(player1.name || '球友')}`)
  query.push(`target_avatar=${encodeURIComponent(player1.avatar || '')}`)
  appendPlayerQuery(query, 'opponent', player2)
  return `/subPages/user/h2hRecord?${query.join('&')}`
}

export const buildSpectatorFinishedMatchDetailRoute = ({ match = {} } = {}) => {
  const matchId = Number(match.match_id || match.id || 0)
  if (Number(match.status) !== 2 || matchId <= 0) return ''
  return `/subPages/match/matchDetail?match_id=${matchId}&mode=spectate`
}
