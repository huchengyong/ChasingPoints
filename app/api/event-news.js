/**
 * 赛事情报 API 接口
 */
import { get } from '@/utils/request.js'

/**
 * 获取赛事情报列表
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.page_size 每页条数
 * @param {number} params.game_type 球种（可选）
 * @param {number} params.status 状态（可选）
 * @param {string} params.city 城市（可选）
 * @returns {Promise}
 */
export const getEventNewsList = (params = {}) => {
  return get('/api/event-news/list', params)
}

/**
 * 获取赛讯赛事视图
 * @param {Object} params 查询参数
 * @param {number} params.event_id 赛事 ID
 * @returns {Promise}
 */
export const getEventNewsView = (params = {}) => {
  return get('/api/event-news/view', params)
}
