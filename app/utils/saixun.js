import { getGameTypeLabel } from './game-types.js'
import { formatEventNewsTime, getEventNewsStatusText } from './home-index.js'

const formatLocationText = (item = {}) => {
  const city = typeof item.city === 'string' ? item.city.trim() : ''
  const venue = typeof item.venue === 'string' ? item.venue.trim() : ''

  if (city && venue) return `${city} · ${venue}`
  return city || venue || ''
}

export const normalizeSaiXunCard = (item = {}, now = Date.now()) => {
  const timeSource = item.start_time || item.sort_time || item.end_time || item.published_at || item.created_at

  return {
    ...item,
    id: item.id,
    title: item.title || '赛事情报',
    summary: item.summary || item.latest_result_text || '官方赛讯持续更新中',
    statusText: getEventNewsStatusText(item.status, '赛讯更新中'),
    gameTypeText: getGameTypeLabel(item.game_type, '台球'),
    timeText: formatEventNewsTime(timeSource, now),
    locationText: formatLocationText(item),
    stageText: item.current_stage_text || '阶段待更新',
    resultText: item.latest_result_text || '',
    sourceText: typeof item.source_name === 'string' ? item.source_name.trim() : ''
  }
}

export const buildSaiXunHeroStats = (list = []) => ({
  totalCount: list.length,
  liveCount: list.filter((item) => Number(item?.status) === 1).length,
  upcomingCount: list.filter((item) => Number(item?.status) === 0).length
})
