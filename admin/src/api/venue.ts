import request from '@/utils/request'

export interface Venue {
  id: number
  name: string
  address: string
  city: string
  district: string
  latitude: number
  longitude: number
  phone: string
  images: string
  business_hours: string
  table_count: number
  price_range: string
  description: string
  status: number
  geo_status: number
  owner_user_id: number
  created_at: string
}

export interface VenueListParams {
  page?: number
  page_size?: number
  status?: number
  city?: string
}

export interface VenueListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: Venue[]
}

export interface ReviewParams {
  venue_id: number
  status: number // 1:通过 3:拒绝
  reject_reason?: string
}

// 获取球馆列表（管理员）
export const getVenueList = (params: VenueListParams = {}): Promise<VenueListResult> => {
  return request.get('/api/admin/venue/list', { params })
}

// 审核球馆
export const reviewVenue = (data: ReviewParams): Promise<{ code: number; success: boolean; message: string }> => {
  return request.post('/api/admin/venue/review', data)
}
