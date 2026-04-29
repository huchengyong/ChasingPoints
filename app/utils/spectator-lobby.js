import { GAME_TYPE_LABEL_MAP } from './game-types.js'

export const SPECTATOR_SCOPES = [
  { value: 'hall', label: '大厅' },
  { value: 'friends', label: '好友' }
]

export const SPECTATOR_STATUS_OPTIONS = [
  { value: 1, label: '进行中' },
  { value: 2, label: '已结束' }
]

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
    subtitle: '可以先发起 PK，或稍后回来观赛'
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
