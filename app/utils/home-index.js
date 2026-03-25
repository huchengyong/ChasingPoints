import { getGameTypeLabel } from './game-types.js'

const REPORT_POST_TYPE = 1
const EVENT_NEWS_STATUS_MAP = Object.freeze({
  0: '即将开始',
  1: '进行中',
  2: '已结束',
  3: '已取消'
})

const formatTimestamp = (dateTime) => {
  if (!dateTime) return Number.POSITIVE_INFINITY
  const parsed = new Date(dateTime).getTime()
  return Number.isNaN(parsed) ? Number.POSITIVE_INFINITY : parsed
}

export const getEventNewsStatusText = (status, fallback = '赛事情报') => {
  return EVENT_NEWS_STATUS_MAP[Number(status)] || fallback
}

export const formatEventNewsTime = (dateTime, now = Date.now()) => {
  const currentTime = typeof now === 'string' ? new Date(now).getTime() : now
  const eventTime = formatTimestamp(dateTime)

  if (!Number.isFinite(eventTime) || !Number.isFinite(currentTime)) {
    return '时间待定'
  }

  const date = new Date(eventTime)
  const currentDate = new Date(currentTime)
  const today = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate()).getTime()
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  const diffDays = Math.round((target - today) / (24 * 60 * 60 * 1000))
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')

  if (diffDays === 0) return `今天 ${hh}:${mm}`
  if (diffDays === 1) return `明天 ${hh}:${mm}`
  if (diffDays === 2) return `后天 ${hh}:${mm}`

  return `${date.getMonth() + 1}月${date.getDate()}日 ${hh}:${mm}`
}

const getEventNewsLocationText = (item) => {
  const city = typeof item.city === 'string' ? item.city.trim() : ''
  const venue = typeof item.venue === 'string' ? item.venue.trim() : ''

  if (city && venue) return `${city} · ${venue}`
  return city || venue || ''
}

export const normalizeFeaturedEventNews = (item, now = Date.now()) => {
  if (!item) return null

  const timeSource = item.start_time || item.sort_time || item.end_time || item.published_at || item.created_at
  const locationText = getEventNewsLocationText(item)
  const sourceText = typeof item.source_name === 'string' ? item.source_name.trim() : ''

  return {
    id: item.id,
    title: item.title || '赛事情报',
    summary: item.summary || item.latest_result_text || '查看最新赛程赛况',
    statusText: getEventNewsStatusText(item.status),
    timeText: formatEventNewsTime(timeSource, now),
    typeText: item.game_type != null ? getGameTypeLabel(item.game_type, '台球') : '台球',
    locationText,
    sourceText,
    sourceUrl: item.source_url || '',
    currentStageText: item.current_stage_text || '',
    latestResultText: item.latest_result_text || '',
    gameType: item.game_type,
    status: item.status
  }
}

export const buildFeaturedPostTarget = (post) => {
  if (!post || post.post_type !== REPORT_POST_TYPE) {
    return {
      type: 'community',
      url: '/pages/social/index',
      ctaText: '去社区查看更多'
    }
  }

  if (post.match_id) {
    return {
      type: 'navigate',
      url: `/subPages/match/shareResult?match_id=${post.match_id}`,
      ctaText: '查看战报'
    }
  }

  return {
    type: 'community',
    url: '/pages/social/index',
    ctaText: '去社区查看更多'
  }
}
