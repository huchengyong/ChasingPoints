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
 * 获取当前用户的本场新解锁荣誉
 * @param {Object} params { match_id }
 * @returns {Promise}
 */
export const getMatchRewardSummary = (params) => {
  return get('/api/match/reward-summary', params)
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
 * @param {Object} data 对局信息；新斯诺克和中八/美九对局默认使用自由局数
 * @returns {Promise}
 */
export const startMatch = (data = {}) => {
  const gameType = Number(data.game_type || data.gameType || 0)
  const isSnooker = gameType === 1
  const isPoolRoundWin = [3, 4].includes(gameType)
  const hasLegacyFormat = Number(data.best_of_frames || data.bestOfFrames || 0) > 0
  const payload = isSnooker
    ? {
        ...data,
        snooker_rules_version: Number(data.snooker_rules_version || data.snookerRulesVersion || 2),
        ...(!data.snooker_format && !data.snookerFormat && !hasLegacyFormat
          ? { snooker_format: 'free', snooker_target_wins: 0 }
          : {})
      }
    : isPoolRoundWin
      ? {
          ...data,
          ...(!data.match_format && !data.matchFormat
            ? { match_format: 'free', target_wins: 0 }
            : {
                match_format: data.match_format || data.matchFormat,
                target_wins: Number(data.target_wins ?? data.targetWins ?? 0)
              })
        }
    : data
  const message = validateStartMatchPayload(payload)
  if (message) {
    return Promise.resolve({
      success: false,
      message
    })
  }
  return post('/api/match/start', payload)
}

/**
 * 结束对局
 * @param {Object} data 对局信息
 * @returns {Promise}
 */
export const finishMatch = (data) => {
  return post('/api/match/finish', data)
}

export const requestFinishMatch = (data) => {
  return post('/api/match/finish/request', data)
}

export const confirmFinishMatch = (data) => {
  return post('/api/match/finish/confirm', data)
}

export const disputeFinishMatch = (data) => {
  return post('/api/match/finish/dispute', data)
}

export const withdrawFinishMatch = (data) => {
  return post('/api/match/finish/withdraw', data)
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

export const getH2HOverview = (params = {}) => {
  return get('/api/match/h2h/overview', params)
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
 * 裁判码预览
 * @param {Object} data { match_id, join_token }
 * @returns {Promise}
 */
export const previewMatchReferee = (data) => {
  return post('/api/match/referee/preview', data)
}

/**
 * 获取裁判历史
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.page_size 每页条数
 * @returns {Promise}
 */
export const getRefereeHistory = (params = {}) => {
  return get('/api/match/referee/history', params)
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

export const updateSnookerFormat = (data) => {
  return post('/api/match/snooker/format', data)
}

export const updateMatchFormat = (data) => {
  return post('/api/match/format', data)
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
 * 记录版本2斯诺克一次击球结果
 * @param {Object} data 击球方、结果、入袋球及犯规决定
 * @returns {Promise}
 */
export const snookerStroke = (data) => {
  return post('/api/match/snooker/stroke', data)
}

/**
 * 记录版本2斯诺克局级动作
 * @param {Object} data 重置黑球、认输或裁判判局动作
 * @returns {Promise}
 */
export const snookerFrameAction = (data) => {
  return post('/api/match/snooker/frame-action', data)
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
