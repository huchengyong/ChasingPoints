/**
 * 用户相关 API
 */

import { get, post } from '@/utils/request.js'

/**
 * 获取用户统计数据
 * @returns {Promise} 返回 { total_matches, wins, losses, win_rate, max_win_streak }
 */
export const getUserStats = () => {
  return get('/api/user/stats')
}

/**
 * 更新昵称
 * @param {String} nickname 新昵称
 * @returns {Promise}
 */
export const updateNickname = (nickname) => {
  return post('/api/user/nickname', { nickname })
}

export default {
  getUserStats,
  updateNickname
}
