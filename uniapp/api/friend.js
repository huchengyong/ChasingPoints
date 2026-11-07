/**
 * 好友系统 API 接口
 */
import { get, post } from '@/utils/request.js'

/**
 * 发送好友请求
 * @param {Object} data { to_user_id, message }
 * @returns {Promise}
 */
export const sendFriendRequest = (data) => {
  return post('/api/friend/request', data)
}

/**
 * 接受好友请求
 * @param {Object} data { request_id }
 * @returns {Promise}
 */
export const acceptFriendRequest = (data) => {
  return post('/api/friend/accept', data)
}

/**
 * 拒绝好友请求
 * @param {Object} data { request_id }
 * @returns {Promise}
 */
export const rejectFriendRequest = (data) => {
  return post('/api/friend/reject', data)
}

/**
 * 获取好友列表
 * @param {Object} params { page, page_size }
 * @returns {Promise}
 */
export const getFriendList = (params = {}) => {
  return get('/api/friend/list', params)
}

/**
 * 删除好友
 * @param {Object} data { friend_user_id }
 * @returns {Promise}
 */
export const deleteFriend = (data) => {
  return post('/api/friend/delete', data)
}

/**
 * 加入黑名单
 * @param {Object} data { friend_user_id }
 * @returns {Promise}
 */
export const blacklistFriend = (data) => {
  return post('/api/friend/blacklist', data)
}

/**
 * 获取好友请求列表
 * @param {Object} params { page, page_size }
 * @returns {Promise}
 */
export const getFriendRequests = (params = {}) => {
  return get('/api/friend/requests', params)
}

/**
 * 搜索用户（按昵称/手机号）
 * @param {Object} params { keyword, page, page_size }
 * @returns {Promise}
 */
export const searchUser = (params = {}) => {
  return get('/api/friend/search', params)
}
