/**
 * 分享系统 API 接口
 */
import { get } from '@/utils/request.js'

/**
 * 获取对局分享数据
 * @param {Object} params { match_id }
 */
export const getMatchShareData = (params) => {
  return get('/api/share/match', params)
}
