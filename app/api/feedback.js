/**
 * 投诉举报与意见反馈 API
 */
import { post } from '@/utils/request.js'

export const createUserFeedbackTicket = (data) => {
  return post('/api/user/feedback/create', data)
}

export default {
  createUserFeedbackTicket
}
