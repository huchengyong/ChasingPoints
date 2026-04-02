import { getGameTypeLabel } from './game-types.js'
import { formatEventNewsTime, getEventNewsStatusText } from './home-index.js'

const formatLocationText = (item = {}) => {
  const city = typeof item.city === 'string' ? item.city.trim() : ''
  const venue = typeof item.venue === 'string' ? item.venue.trim() : ''

  if (city && venue) return `${city} · ${venue}`
  return city || venue || ''
}

export const normalizeSaiXunCard = (item = {}, now = Date.now()) => {
  const startTime = item.start_time || item.sort_time || item.created_at
  const endTime = item.end_time || ''
  const currentRoundText = typeof item.current_round_text === 'string' ? item.current_round_text.trim() : ''
  const matchCount = Number(item.match_count || 0)

  return {
    ...item,
    id: item.id,
    title: item.tournament_name || item.title || '赛事情报',
    summary: item.summary || item.latest_result_text || '官方赛讯持续更新中',
    statusText: getEventNewsStatusText(item.status, '赛讯更新中'),
    gameTypeText: getGameTypeLabel(item.game_type, '台球'),
    timeText: formatEventDateRange(startTime, endTime, now),
    locationText: formatLocationText(item),
    currentRoundText: currentRoundText || '轮次待更新',
    resultText: item.latest_result_text || '',
    matchCount,
    matchCountText: matchCount > 0 ? `${matchCount} 场比赛` : '比赛待更新',
    sourceText: typeof item.source_name === 'string' ? item.source_name.trim() : ''
  }
}

export const buildSaiXunHeroStats = (list = []) => ({
  totalCount: list.length,
  liveCount: list.filter((item) => Number(item?.status) === 1).length,
  upcomingCount: list.filter((item) => Number(item?.status) === 0).length
})

function formatEventDateRange(startTime, endTime, now) {
  const startText = formatEventNewsTime(startTime, now)
  if (!endTime) return startText

  const endText = formatEventNewsTime(endTime, now)
  if (startText === '时间待定') return endText
  if (endText === '时间待定') return startText
  return `${startText} - ${endText}`
}
