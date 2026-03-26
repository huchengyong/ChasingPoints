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
 * 获取常玩球馆奖励状态
 * @returns {Promise} 返回 { enabled, popup_enabled, reward_days, new_user_window_days, status }
 */
export const getFavoriteVenueRewardStatus = () => {
  return get('/api/user/favorite-venue-reward-status')
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
  getFavoriteVenueRewardStatus,
  getUserStats,
  updateNickname
}
