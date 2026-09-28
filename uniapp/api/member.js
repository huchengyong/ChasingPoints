import { get, post } from '@/utils/request.js'

export const getMemberPlans = () => {
  return get('/api/member/plans')
}

export const getMemberStatus = () => {
  return get('/api/member/status')
}

export const createMemberSubscriptionOrder = (data) => {
  return post('/api/member/order', data)
}

export const getMemberSubscriptionOrderStatus = (params) => {
  return get('/api/member/order/status', params)
}

export default {
  getMemberPlans,
  getMemberStatus,
  createMemberSubscriptionOrder,
  getMemberSubscriptionOrderStatus
}
