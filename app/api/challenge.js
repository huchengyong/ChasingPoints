/**
 * 约球 API 门面：页面统一通过这里访问后端 challenge 接口。
 */
import { get, post } from '@/utils/request.js'

// 约球写接口以 success=false 返回确认提示和冲突结果；统一请求层会将它们
// 当作业务错误抛出。仅恢复 HTTP 2xx 的业务响应，登录/网络/服务器错误照常抛出。
const writeChallenge = (url, data) => post(url, data).catch((error) => {
  if (error.statusCode >= 200 && error.statusCode < 300 && error.category === 'business' && error.responseData?.success === false) {
    return error.responseData
  }
  throw error
})

/**
 * 发起约球
 * @param {Object} data { to_user_id, game_type, match_mode, visibility, match_format?, target_wins?,
 *   snooker_*?, message?, day_offset, scheduled_date, start_hour, end_hour }
 */
export const sendChallenge = (data) => {
  return writeChallenge('/api/challenge/send', data)
}

/**
 * 接受约球
 * @param {Object} data { challenge_id, confirm_close_own_pending? }
 */
export const acceptChallenge = (data) => {
  return writeChallenge('/api/challenge/accept', data)
}

/**
 * 拒绝约球
 * @param {Object} data { challenge_id }
 */
export const rejectChallenge = (data) => {
  return writeChallenge('/api/challenge/reject', data)
}

/**
 * 取消约球（待回应发起方 / 已接受未开局双方）
 * @param {Object} data { challenge_id }
 */
export const cancelChallenge = (data) => {
  return writeChallenge('/api/challenge/cancel', data)
}

/**
 * 放弃本次约球（另一方等待时）
 * @param {Object} data { challenge_id }
 */
export const abandonChallenge = (data) => {
  return writeChallenge('/api/challenge/abandon', data)
}

/**
 * 进入约球对局（首人等待 / 第二人开局）
 * @param {Object} data { challenge_id, confirm_early? }
 */
export const enterChallenge = (data) => {
  return writeChallenge('/api/challenge/enter', data)
}

/**
 * 退出等待，保留约球
 * @param {Object} data { challenge_id }
 */
export const leaveChallenge = (data) => {
  return writeChallenge('/api/challenge/leave', data)
}

/**
 * 当前约球摘要（活动卡与服务端时间）
 */
export const getChallengeSummary = () => {
  return get('/api/challenge/summary')
}

/**
 * 活动约球列表（收到待回应 + 已接受/已开局 + 本人发出待回应）
 */
export const getPendingChallenges = () => {
  return get('/api/challenge/pending')
}

/**
 * 约球详情
 * @param {Object} params { id }
 */
export const getChallengeDetail = (params) => {
  return get('/api/challenge/detail', params)
}

/**
 * 约球历史分页
 * @param {Object} params { before_id?, page_size? }
 */
export const getChallengeHistory = (params) => {
  return get('/api/challenge/history', params)
}
