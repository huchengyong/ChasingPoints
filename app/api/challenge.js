/**
 * PK 邀约兼容 API 接口
 * 当前后端仍沿用 challenge 命名，前端语义上按社交 PK 邀约处理。
 */
import { get, post } from '@/utils/request.js'

/**
 * 发起 PK 邀约
 * @param {Object} data { to_user_id, game_type, message }
 */
export const sendChallenge = (data) => {
  return post('/api/challenge/send', data)
}

/**
 * 回应 PK 邀约
 * @param {Object} data { challenge_id }
 */
export const acceptChallenge = (data) => {
  return post('/api/challenge/accept', data)
}

/**
 * 拒绝 PK 邀约
 * @param {Object} data { challenge_id }
 */
export const rejectChallenge = (data) => {
  return post('/api/challenge/reject', data)
}

/**
 * 获取 PK 邀约列表
 * @param {Object} params { page, page_size }
 */
export const getPendingChallenges = (params) => {
  return get('/api/challenge/pending', params)
}
