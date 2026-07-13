/**
 * 对局相关 API 接口
 */
import { get, post } from '@/utils/request.js'
import { validateStartMatchPayload } from '@/utils/start-match.js'

/**
 * 获取对局记录列表
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.page_size 每页条数
 * @param {number} params.game_type 游戏类型（可选）
 * @param {number} params.result 结果过滤（可选）
 * @returns {Promise}
 */
export const getMatchList = (params = {}) => {
  return get('/api/match/list', params)
}

/**
 * 获取进行中的对局
 * @param {Object} options 请求配置
 * @returns {Promise}
 */
export const getCurrentMatch = (options = {}) => {
  return get('/api/match/current', {}, options)
}

/**
 * 获取平台正在进行的对局列表（公开接口）
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.page_size 每页条数
 * @returns {Promise}
 */
export const getOngoingMatches = (params = {}) => {
  return get('/api/public/matches/ongoing', params)
}

/**
 * 获取公开观赛对局列表
 * @param {Object} params 查询参数
 * @param {string} params.scope hall/friends
 * @param {number} params.status 大厅状态筛选：1=进行中 2=已结束
 * @param {number} params.game_type 球种（可选）
 * @returns {Promise}
 */
export const getPublicMatches = (params = {}) => {
  return get('/api/public/matches', params)
}

/**
 * 获取对局详情
 * @param {Object} params { match_id }
 * @returns {Promise}
 */
export const getMatchDetail = (params) => {
  return get('/api/match/detail', params)
}

/**
 * 获取公开对局详情（观战模式，无需登录）
 * @param {Object} params { match_id }
 * @returns {Promise}
 */
export const getPublicMatchDetail = (params) => {
  return get('/api/public/match/detail', params)
}

/**
 * 开始对局
 * @param {Object} data 对局信息
 * @returns {Promise}
 */
export const startMatch = (data) => {
  const message = validateStartMatchPayload(data)
  if (message) {
    return Promise.resolve({
      success: false,
      message
    })
  }
  return post('/api/match/start', data)
}

/**
 * 结束对局
 * @param {Object} data 对局信息
 * @returns {Promise}
 */
export const finishMatch = (data) => {
  return post('/api/match/finish', data)
}

/**
 * 获取交锋统计
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getH2HStats = (params) => {
  return get('/api/match/h2h/stats', params)
}

/**
 * 获取交锋历史
 * @param {Object} params 查询参数
 * @returns {Promise}
 */
export const getH2HHistory = (params) => {
  return get('/api/match/h2h/history', params)
}

/**
 * 获取匹配二维码
 * @returns {Promise}
 */
export const getMatchQRCode = () => {
  return get('/api/match/qrcode')
}

/**
 * 获取本场裁判二维码
 * @param {Object} params { match_id }
 * @returns {Promise}
 */
export const getMatchRefereeQRCode = (params) => {
  return get('/api/match/referee/qrcode', params)
}

/**
 * 扫码加入并担任本场裁判
 * @param {Object} data { match_id, join_token }
 * @returns {Promise}
 */
export const joinMatchReferee = (data) => {
  return post('/api/match/referee/join', data)
}

/**
 * 加分
 * @param {Object} data { match_id, actor, score }
 * @returns {Promise}
 */
export const matchScore = (data) => {
  return post('/api/match/score', data)
}

/**
 * 结束一局
 * @param {Object} data { match_id, winner, win_type }
 * @returns {Promise}
 */
export const endRound = (data) => {
  return post('/api/match/round/end', data)
}

/**
 * 开始下一局
 * @param {Object} data { match_id }
 * @returns {Promise}
 */
export const startNextRound = (data) => {
  return post('/api/match/round/start', data)
}

/**
 * 犯规
 * @param {Object} data { match_id, actor }
 * @returns {Promise}
 */
export const matchFoul = (data) => {
  return post('/api/match/foul', data)
}

/**
 * 撤销操作
 * @param {Object} data { match_id }
 * @returns {Promise}
 */
export const matchUndo = (data) => {
  return post('/api/match/undo', data)
}

/**
 * 获取对手列表
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.page_size 每页条数
 * @param {string} params.keyword 搜索关键词（可选）
 * @returns {Promise}
 */
export const getOpponentList = (params = {}) => {
  return get('/api/opponent/list', params)
}
