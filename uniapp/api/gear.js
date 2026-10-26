/**
 * 装备 API 接口：目录、我的装备、好友装备、本场用杆
 */
import { get, post } from '@/utils/request.js'

/**
 * 获取目录品牌
 * @param {Object} params { category: cue/break_jump/tip/chalk }
 */
export const getGearBrands = (params = {}, options = {}) => {
  return get('/api/gear/catalog/brands', params, options)
}

/**
 * 获取目录型号
 * @param {Object} params { category, brand_id, keyword, house, page, page_size }
 */
export const getGearProducts = (params = {}, options = {}) => {
  return get('/api/gear/catalog/products', params, options)
}

export const getMyGear = (options = {}) => {
  return get('/api/gear/mine', {}, options)
}

/**
 * 登记装备：product_id > 0 选目录型号，否则登记未收录型号
 */
export const createGearItem = (data) => {
  return post('/api/gear/item/create', data)
}

/**
 * 修改装备状态或皮头巧克：update_tip / update_chalk 为 true 才会改对应子属性
 */
export const updateGearItem = (data) => {
  return post('/api/gear/item/update', data)
}

export const deleteGearItem = (data) => {
  return post('/api/gear/item/delete', data)
}

/**
 * 设置或清除主力装备：item_id 为 0 表示清除
 * @param {Object} data { game_type, slot: cue/break, item_id }
 */
export const setGearDefault = (data) => {
  return post('/api/gear/default/set', data)
}

export const getUserGear = (params = {}, options = {}) => {
  return get('/api/gear/user', params, options)
}

export const getGearMatch = (params = {}, options = {}) => {
  return get('/api/gear/match', params, options)
}

export const updateGearMatch = (data) => {
  return post('/api/gear/match/update', data)
}

export default {
  getGearBrands,
  getGearProducts,
  getMyGear,
  createGearItem,
  updateGearItem,
  deleteGearItem,
  setGearDefault,
  getUserGear,
  getGearMatch,
  updateGearMatch
}
