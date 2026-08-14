/**
 * 用户相关 API
 */

import { get, post } from '@/utils/request.js'

export const getUserInfo = (options = {}) => {
  return get('/api/user/info', {}, options)
}

export const getUserBootstrap = (options = {}) => {
  return get('/api/user/bootstrap', {}, options)
}

export const getUserOverview = (options = {}) => {
  return get('/api/user/overview', {}, options)
}

export const getUserReputation = () => {
  return get('/api/user/reputation')
}

export const getUserReputationLogs = (params = {}) => {
  return get('/api/user/reputation/logs', params)
}

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
export const getFavoriteVenueRewardStatus = (options = {}) => {
  return get('/api/user/favorite-venue-reward-status', {}, options)
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

export const getQiniuUploadToken = (fileExt = '') => {
  return get('/api/user/upload-token', {
    file_ext: fileExt
  })
}

export const updateUserProfile = (data) => {
  return post('/api/user/profile', data)
}

export default {
  getFavoriteVenueRewardStatus,
  getQiniuUploadToken,
  getUserBootstrap,
  getUserInfo,
  getUserOverview,
  getUserReputation,
  getUserReputationLogs,
  getUserPrivacy,
  getUserStats,
  updateUserPrivacy,
  updateNickname,
  updateUserProfile
}
