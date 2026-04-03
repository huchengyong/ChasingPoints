import { getGameTypeLabel } from './game-types.js'
import { formatEventNewsTime, getEventNewsStatusText } from './home-index.js'

export const DEFAULT_EVENT_COVER = 'https://images.gc.wstservices.co.uk/fit-in/400x600/4ddad400-99d3-11ee-94e8-c9d138e537ff.png'
export const DEFAULT_PLAYER_AVATAR = '/static/images/default-avatar.png'

const MATCH_ACTIVE_STATUSES = new Set([0, 1])

const parseDateOnly = (value) => {
  if (!value) return null
  const parsed = new Date(`${value}T00:00:00+08:00`)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const parseDateTime = (value) => {
  if (!value) return null

  const normalized = typeof value === 'string' && value.includes('T')
    ? value
    : `${String(value).replace(' ', 'T')}+08:00`
  const parsed = new Date(normalized)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const formatMonthDay = (date) => {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return '日期待定'
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

export const formatEventDateRange = (startDate, endDate) => {
  const start = parseDateOnly(startDate)
  const end = parseDateOnly(endDate || startDate)

  if (!start && !end) return '日期待定'
  if (start && end && start.getTime() === end.getTime()) return formatMonthDay(start)
  if (!start) return formatMonthDay(end)
  if (!end) return formatMonthDay(start)
  return `${formatMonthDay(start)} - ${formatMonthDay(end)}`
}

export const formatEventTimeRange = (startTime, endTime, now = Date.now()) => {
  const start = parseDateTime(startTime)
  const end = parseDateTime(endTime)

  if (!start && !end) return ''
  if (start && end) {
    return `${formatEventNewsTime(start.toISOString(), now)} - ${formatEventNewsTime(end.toISOString(), now)}`
  }
  return formatEventNewsTime((start || end).toISOString(), now)
}

export const buildEventLocationText = (item = {}) => {
  const city = typeof item.city === 'string' ? item.city.trim() : ''
  const venue = typeof item.venue === 'string' ? item.venue.trim() : ''

  if (city && venue) return `${city} · ${venue}`
  return city || venue || ''
}

export const normalizeSaiXunCard = (item = {}, now = Date.now()) => {
  const currentRoundText = typeof item.current_round_text === 'string' ? item.current_round_text.trim() : ''
  const matchCount = Number(item.match_count || 0)
  const timeText = formatEventTimeRange(item.start_time, item.end_time, now)

  return {
    ...item,
    id: item.id,
    title: item.tournament_name || item.title || '赛事情报',
    summary: item.summary || item.latest_result_text || '官方赛讯持续更新中',
    statusText: getEventNewsStatusText(item.status, '赛讯更新中'),
    gameTypeText: getGameTypeLabel(item.game_type, '台球'),
    dateText: formatEventDateRange(item.start_date, item.end_date),
    timeText,
    showTime: Boolean(timeText),
    locationText: buildEventLocationText(item),
    currentRoundText: currentRoundText || '轮次待更新',
    resultText: item.latest_result_text || '',
    matchCount,
    matchCountText: matchCount > 0 ? `${matchCount} 场比赛` : '比赛待更新',
    sourceText: typeof item.source_name === 'string' ? item.source_name.trim() : '',
    coverImage: item.cover_image || DEFAULT_EVENT_COVER
  }
}

export const buildSaiXunHeroStats = (list = []) => ({
  totalCount: list.length,
  liveCount: list.filter((item) => Number(item?.status) === 1).length,
  upcomingCount: list.filter((item) => Number(item?.status) === 0).length
})

const buildMatchMetaText = (match) => {
  const parts = []
  if (Number(match.best_of || 0) > 0) parts.push(`Best of ${Number(match.best_of || 0)}`)
  if (match.is_placeholder) parts.push('占位赛程')
  return parts.join(' · ') || '等待比赛开始'
}

const buildScoreText = (match) => {
  const status = Number(match.status || 0)
  const scoreReady = status === 1 || status === 2 || Number(match.home_score || 0) > 0 || Number(match.away_score || 0) > 0
  return scoreReady ? `${Number(match.home_score || 0)} : ${Number(match.away_score || 0)}` : '-'
}

const normalizeSaiXunMatch = (match = {}, now = Date.now()) => {
  const startAt = parseDateTime(match.start_time)
  return {
    id: match.id,
    roundName: match.round_name || '轮次待更新',
    roundOrder: Number(match.round_order || 0),
    matchOrder: Number(match.match_order || 0),
    status: Number(match.status || 0),
    statusText: getEventNewsStatusText(match.status, '待更新'),
    startAt,
    startTimeText: startAt ? formatEventNewsTime(startAt.toISOString(), now) : '时间待定',
    homePlayerName: match.home_player_name || '待定',
    homePlayerAvatar: match.home_player_avatar || DEFAULT_PLAYER_AVATAR,
    awayPlayerName: match.away_player_name || '待定',
    awayPlayerAvatar: match.away_player_avatar || DEFAULT_PLAYER_AVATAR,
    scoreText: buildScoreText(match),
    winnerSide: Number(match.winner_side || 0),
    metaText: buildMatchMetaText(match),
    isPlaceholder: Boolean(match.is_placeholder)
  }
}

const compareSaiXunMatches = (left, right) => {
  const leftStatus = Number(left.status || 0)
  const rightStatus = Number(right.status || 0)

  if (leftStatus !== rightStatus) {
    const priority = { 1: 0, 0: 1, 2: 2, 3: 3 }
    return (priority[leftStatus] ?? 9) - (priority[rightStatus] ?? 9)
  }

  const leftTime = left.startAt?.getTime() ?? Number.POSITIVE_INFINITY
  const rightTime = right.startAt?.getTime() ?? Number.POSITIVE_INFINITY

  if (MATCH_ACTIVE_STATUSES.has(leftStatus)) {
    if (leftTime !== rightTime) return leftTime - rightTime
  } else if (leftStatus === 2 || leftStatus === 3) {
    if (leftTime !== rightTime) return rightTime - leftTime
  }

  if (left.roundOrder !== right.roundOrder) return right.roundOrder - left.roundOrder
  if (left.matchOrder !== right.matchOrder) return left.matchOrder - right.matchOrder
  return Number(left.id || 0) - Number(right.id || 0)
}

export const buildSaiXunDetailRounds = (matches = [], now = Date.now()) => {
  const normalized = matches.map((item) => normalizeSaiXunMatch(item, now))
  const activeRoundOrders = normalized
    .filter((item) => MATCH_ACTIVE_STATUSES.has(item.status))
    .map((item) => item.roundOrder)
    .filter((value) => value > 0)
  const maxVisibleRoundOrder = activeRoundOrders.length ? Math.min(...activeRoundOrders) : Number.POSITIVE_INFINITY
  const visibleMatches = normalized
    .filter((item) => item.roundOrder <= maxVisibleRoundOrder)
    .sort(compareSaiXunMatches)

  const roundMap = new Map()
  visibleMatches.forEach((item) => {
    const key = `${item.roundOrder}-${item.roundName}`
    if (!roundMap.has(key)) {
      roundMap.set(key, {
        key,
        roundName: item.roundName,
        roundOrder: item.roundOrder,
        matches: []
      })
    }
    roundMap.get(key).matches.push(item)
  })

  return Array.from(roundMap.values())
}
