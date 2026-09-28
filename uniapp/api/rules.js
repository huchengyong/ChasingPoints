/**
 * 规则百科 API 接口
 */
import { get } from '@/utils/request.js'

/**
 * 获取规则内容
 * @param {Object} params { category, content_type }
 * @returns {Promise}
 */
export const getRuleContent = (params = {}) => {
  return get('/api/rules/content', params)
}

/**
 * 获取术语词典
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getGlossary = (params = {}) => {
  return get('/api/rules/glossary', params)
}

/**
 * 搜索规则
 * @param {Object} params { keyword }
 * @returns {Promise}
 */
export const searchRules = (params = {}) => {
  return get('/api/rules/search', params)
}
