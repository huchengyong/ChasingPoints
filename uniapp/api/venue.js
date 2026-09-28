/**
 * 球馆模块 API 接口
 */
import { get, post } from '@/utils/request.js'

/**
 * 获取球馆列表
 * @param {Object} params { page, page_size, city, latitude, longitude, sort_by }
 */
export const getVenueList = (params) => {
  return get('/api/venue/list', params)
}

/**
 * 获取球馆详情
 * @param {Object} params { venue_id }
 */
export const getVenueDetail = (params) => {
  return get('/api/venue/detail', params)
}

/**
 * 获取附近球馆
 * @param {Object} params { latitude, longitude, radius, limit }
 */
export const getNearbyVenues = (params) => {
  return get('/api/venue/nearby', params)
}

/**
 * 获取地区选项
 * @param {Object} params { parent_id }
 */
export const getVenueAreaOptions = (params) => {
  return get('/api/venue/areas', params)
}

/**
 * 创建/上传球馆
 * @param {Object} data { name, address, city, district }
 */
export const createVenue = (data) => {
  return post('/api/venue/create', data)
}

/**
 * 球馆签到
 * @param {Object} data { venue_id }
 */
export const checkinVenue = (data) => {
  return post('/api/venue/checkin', data)
}
