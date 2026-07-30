/**
 * 成就系统 API 接口
 */
import { get, post } from '@/utils/request.js'

/**
 * 获取成就列表（含用户进度）
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getAchievementList = (params = {}) => {
  return get('/api/achievement/list', params)
}

/**
 * 获取荣誉墙聚合数据
 * @param {Object} params { user_id, game_type, history_season_id, history_page, history_page_size }
 * @returns {Promise}
 */
export const getHonorWall = (params = {}) => {
  return get('/api/achievement/honor-wall', params)
}

/**
 * 装备/卸下称号
 * @param {Object} data { title_id, equip }
 * @returns {Promise}
 */
export const equipTitle = (data) => {
  return post('/api/achievement/title/equip', data)
}

/**
 * 获取用户称号列表
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getUserTitles = (params = {}) => {
  return get('/api/achievement/titles', params)
}
