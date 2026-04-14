import request from '@/utils/request'

export interface AdminReputationLogItem {
  id: number
  user_id: number
  nickname: string
  match_id?: number
  change_type: string
  change_type_text: string
  reason_code: string
  reason_text: string
  reason_detail?: string
  change_score: number
  before_score: number
  after_score: number
  operator_admin_id?: number
  operator_text: string
  created_at: string
}

export interface AdminReputationLogListParams {
  page?: number
  page_size?: number
  user_id?: number
  change_type?: string
  reason_code?: string
}

export interface AdminReputationLogListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: AdminReputationLogItem[]
}

export const getAdminReputationLogList = (
  params: AdminReputationLogListParams = {}
): Promise<AdminReputationLogListResult> => {
  return request.get('/api/admin/reputation/logs', { params })
}
