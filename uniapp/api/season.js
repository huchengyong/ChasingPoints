/**
 * 赛季系统 API 接口
 */
import { get } from '@/utils/request.js'

/**
 * 获取当前赛季
 */
export const getCurrentSeason = () => {
  return get('/api/season/current')
}

/**
 * 获取赛季排行榜
 * @param {Object} params { season_id, page, page_size }
 */
export const getSeasonLeaderboard = (params) => {
  return get('/api/season/leaderboard', params)
}

/**
 * 获取我的赛季记录
 * @param {Object} params { season_id }
 */
export const getMySeasonRecord = (params) => {
  return get('/api/season/my-record', params)
}

export const getSeasonOverview = (params = {}) => {
  return get('/api/season/overview', params)
}

/**
 * 获取赛季报告
 * @param {Object} params { season_id }
 */
export const getSeasonReport = (params) => {
  return get('/api/season/report', params)
}
