/**
 * 段位相关 API 接口
 */
import { get } from '@/utils/request.js'
import { normalizeUserRankInfos } from '@/utils/rank-cache.js'

export { normalizeUserRankInfos }

/**
 * 获取用户段位信息
 * @returns {Promise}
 */
export const getUserRankInfo = (params = {}) => {
  return get('/api/rank/info', params)
}

export const getUserRankInfos = () => {
  return get('/api/rank/infos')
}

/**
 * 获取段位列表
 * @returns {Promise}
 */
export const getRankList = (params = {}) => {
  return get('/api/rank/list', params)
}

/**
 * 获取段位排行榜
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.page_size 每页条数
 * @returns {Promise}
 */
export const getLeaderboard = (params = {}) => {
  return get('/api/public/rank/leaderboard', params)
}

export const getLeaderboardSummary = (params = {}) => {
  return get('/api/public/rank/leaderboard-summary', params)
}

export const getRankConfigs = () => {
  return get('/api/public/rank/configs')
}
