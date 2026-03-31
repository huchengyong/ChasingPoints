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
 * 获取用户隐私设置
 * @returns {Promise}
 */
export const getUserPrivacy = () => {
  return get('/api/user/privacy')
}

/**
 * 更新用户隐私设置
 * @param {Object} data
 * @param {boolean} data.hide_match_record 是否隐藏战绩
 * @returns {Promise}
 */
export const updateUserPrivacy = (data) => {
  return post('/api/user/privacy', data)
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
  getUserPrivacy,
  getUserStats,
  updateUserPrivacy,
  updateNickname
}
