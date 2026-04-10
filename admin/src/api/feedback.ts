import request from '@/utils/request'

export interface AdminFeedbackTicket {
  id: number
  user_id?: number
  source: string
  source_text: string
  category: string
  category_text: string
  content: string
  contact: string
  status: number
  status_text: string
  handler_id?: number
  process_result?: string
  processed_at?: string
  created_at: string
  updated_at: string
}

export interface AdminFeedbackTicketListParams {
  page?: number
  page_size?: number
  status?: number
  category?: string
  source?: string
}

export interface AdminFeedbackTicketListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: AdminFeedbackTicket[]
}

export interface AdminFeedbackTicketProcessParams {
  ticket_id: number
  status: number
  process_result?: string
}

export const getAdminFeedbackTicketList = (
  params: AdminFeedbackTicketListParams = {}
): Promise<AdminFeedbackTicketListResult> => {
  return request.get('/api/admin/feedback/list', { params })
}

export const processAdminFeedbackTicket = (
  data: AdminFeedbackTicketProcessParams
): Promise<{ code: number; success: boolean; message: string }> => {
  return request.post('/api/admin/feedback/process', data)
}
