/**
 * 通知消息 API 接口
 */
import { get, post } from '@/utils/request.js'

/**
 * 获取通知列表
 * @param {Object} params { page, page_size, type }
 * @returns {Promise}
 */
export const getNotificationList = (params = {}) => {
  return get('/api/notification/list', params)
}

/**
 * 标记单条已读
 * @param {Object} data { notification_id }
 * @returns {Promise}
 */
export const markAsRead = (data) => {
  return post('/api/notification/read', data)
}

/**
 * 全部标记已读
 * @param {Object} data
 * @returns {Promise}
 */
export const markAllAsRead = (data = {}) => {
  return post('/api/notification/read-all', data)
}

/**
 * 获取未读数量
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getUnreadCount = (params = {}) => {
  return get('/api/notification/unread-count', params)
}

/**
 * 获取通知偏好
 * @returns {Promise}
 */
export const getNotificationPreferences = () => {
  return get('/api/notification/preferences')
}

/**
 * 保存通知偏好
 * @param {Object} data
 * @returns {Promise}
 */
export const saveNotificationPreferences = (data) => {
  return post('/api/notification/preferences', data)
}

/**
 * 删除通知
 * @param {Object} data { notification_id }
 * @returns {Promise}
 */
export const deleteNotification = (data) => {
  return post('/api/notification/delete', data)
}
