import request from '@/utils/request'

export interface DashboardStats {
  total_users: number
  today_new_users: number
  total_matches: number
  ongoing_matches: number
}

export interface RecentUser {
  id: number
  nickname: string
  phone: string
  created_at: string
}

export interface RecentMatch {
  id: number
  game_type: number
  game_type_name: string
  player1_name: string
  player2_name: string
  my_score: number
  opponent_score: number
  status: number
  created_at: string
}

// 获取首页统计数据
export const getDashboardStats = (): Promise<DashboardStats> => {
  return request.get<unknown, { code: number; success: boolean; message: string } & DashboardStats>('/api/admin/dashboard/stats')
    .then(res => ({
      total_users: res.total_users || 0,
      today_new_users: res.today_new_users || 0,
      total_matches: res.total_matches || 0,
      ongoing_matches: res.ongoing_matches || 0
    }))
}

// 获取最近注册用户
export const getRecentUsers = (): Promise<RecentUser[]> => {
  return request.get<unknown, { code: number; success: boolean; list: RecentUser[] }>('/api/admin/dashboard/recent-users')
    .then(res => res.list || [])
}

// 获取最近对局
export const getRecentMatches = (): Promise<RecentMatch[]> => {
  return request.get<unknown, { code: number; success: boolean; list: RecentMatch[] }>('/api/admin/dashboard/recent-matches')
    .then(res => res.list || [])
}