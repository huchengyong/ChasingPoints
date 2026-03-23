import request from '@/utils/request'

export interface User {
  id: number
  phone: string
  nickname: string
  avatar: string
  status: number
  created_at: string
}

export interface UserListParams {
  page?: number
  page_size?: number
  status?: number
}

export interface UserListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: User[]
}

export interface UpdateUserStatusParams {
  user_id: number
  status: number // 0:禁用 1:正常
}

// 获取用户列表（管理员）
export const getUserList = (params: UserListParams = {}): Promise<UserListResult> => {
  return request.get('/api/admin/users/list', { params })
}

// 更新用户状态
export const updateUserStatus = (data: UpdateUserStatusParams): Promise<{ code: number; success: boolean; message: string }> => {
  return request.post('/api/admin/users/update-status', data)
}
