/**
 * 竞技分析 API 接口
 */
import { get } from '@/utils/request.js'

/**
 * 分球种统计
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getStatsByGameType = (params = {}) => {
  return get('/api/stats/by-game-type', params)
}

export const getStatsOverview = (params = {}) => {
  return get('/api/stats/overview', params)
}

/**
 * 近期趋势（近N场胜率）
 * @param {Object} params { limit }
 * @returns {Promise}
 */
export const getRecentTrend = (params = {}) => {
  return get('/api/stats/recent-trend', params)
}

/**
 * 段位分变化趋势
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getRankScoreTrend = (params = {}) => {
  return get('/api/stats/rank-score-trend', params)
}

/**
 * 单杆最高分记录
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getSingleHighScore = (params = {}) => {
  return get('/api/stats/single-high-score', params)
}

/**
 * 对局时长分析
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getMatchDurationStats = (params = {}) => {
  return get('/api/stats/match-duration', params)
}

/**
 * 强弱对手分析
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getOpponentStrengthAnalysis = (params = {}) => {
  return get('/api/stats/opponent-strength', params)
}
