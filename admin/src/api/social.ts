import request from '@/utils/request'

export interface AdminSocialPost {
  id: number
  user_id: number
  nickname: string
  avatar: string
  content: string
  images: string[]
  post_type: number
  match_id?: number
  likes_count: number
  comments_count: number
  status: number
  status_text: string
  reject_reason: string
  reviewed_at?: string
  reviewed_by?: number
  created_at: string
}

export interface AdminSocialPostListParams {
  page?: number
  page_size?: number
  status?: number
}

export interface AdminSocialPostListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: AdminSocialPost[]
}

export interface AdminSocialPostReviewParams {
  post_id: number
  status: number
  reject_reason?: string
}

export const getAdminSocialPostList = (
  params: AdminSocialPostListParams = {}
): Promise<AdminSocialPostListResult> => {
  return request.get('/api/admin/social/posts', { params })
}

export const reviewAdminSocialPost = (
  data: AdminSocialPostReviewParams
): Promise<{ code: number; success: boolean; message: string }> => {
  return request.post('/api/admin/social/review', data)
}
