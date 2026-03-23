import request from '@/utils/request'

export interface LoginParams {
  email: string
  password: string
}

export interface LoginResult {
  token: string
  userInfo: {
    id: number
    email: string
    nickname: string
    avatar?: string
    role: string
  }
}

export interface AdminActionResult {
  code: number
  success: boolean
  message: string
}

export interface InitAdminParams extends LoginParams {
  setupToken: string
}

// 管理员登录
export const login = (data: LoginParams): Promise<LoginResult> => {
  return request.post<unknown, LoginResult>('/api/admin/login', data)
}

// 初始化管理员账号
export const initAdmin = (data: InitAdminParams): Promise<AdminActionResult> => {
  return request.post<unknown, AdminActionResult>('/api/admin/init', {
    email: data.email,
    password: data.password,
    setup_token: data.setupToken
  })
}

// 获取当前用户信息
export const getUserInfo = (): Promise<LoginResult['userInfo']> => {
  return request
    .get<unknown, { userInfo: LoginResult['userInfo'] }>('/api/admin/user/info')
    .then((res) => res.userInfo)
}

// 修改密码
export const changePassword = (data: { oldPassword: string; newPassword: string }): Promise<AdminActionResult> => {
  return request.post<unknown, AdminActionResult>('/api/admin/user/change-password', {
    old_password: data.oldPassword,
    new_password: data.newPassword
  })
}

// 检查管理员账号是否存在
export const checkAdminExists = (): Promise<{ code: number; success: boolean; message: string; exists: boolean }> => {
  return request.get('/api/admin/exists')
}
