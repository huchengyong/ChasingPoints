export const SEASON_HISTORY_PAGE_SIZE = 10
export const SEASON_REPORT_PATH = '/subPages/season/report'

export const buildSeasonReportUrl = (seasonId, gameType) => `${SEASON_REPORT_PATH}?id=${seasonId}&game_type=${gameType}`

/**
 * 把历届接口的一项整理为列表展示数据；无本人记录时不可进入报告。
 */
export const buildSeasonHistoryItem = (item, gameType) => {
  const season = item?.season
  const seasonId = Number(season?.id) || 0
  if (seasonId <= 0) return null
  const record = item.record || null
  const matches = Number(record?.matches_played) || 0
  const wins = Number(record?.wins) || 0
  const finalRank = Number(record?.final_rank) || 0
  const hasThreshold = (Number(season.qualify_min_matches) || 0) > 0
  let rankText = '未排名'
  if (finalRank > 0) {
    rankText = `第 ${finalRank} 名`
  } else if (hasThreshold && record?.qualified === false) {
    rankText = '未达标'
  }
  return {
    id: seasonId,
    name: season.name || `赛季 ${seasonId}`,
    dateRange: season.start_date && season.end_date ? `${season.start_date} ~ ${season.end_date}` : '',
    hasRecord: Boolean(record),
    rankText,
    matches,
    wins,
    winRate: matches > 0 ? Math.round((wins / matches) * 100) : 0,
    reportUrl: record ? buildSeasonReportUrl(seasonId, gameType) : ''
  }
}

export const hasMoreSeasonHistory = ({ loaded = 0, total = 0, lastPageCount = 0 } = {}) => (
  lastPageCount > 0 && loaded < (Number(total) || 0)
)
