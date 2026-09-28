import { get } from '@/utils/request.js'

export const getOpponentCandidates = (params = {}) => {
  return get('/api/opponent/candidates', params)
}
