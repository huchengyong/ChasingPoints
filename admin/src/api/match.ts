import request from '@/utils/request'

export interface Match {
  id: number
  game_type: number
  game_type_name: string
  player1_name: string
  player2_name: string
  my_score: number
  opponent_score: number
  status: number
  status_text: string
  result?: number
  winner_name: string
  match_time: string
  created_at: string
}

export interface MatchListParams {
  page?: number
  page_size?: number
  status?: number
  game_type?: number
}

export interface MatchListResult {
  total: number
  list: Match[]
}

// 获取对局列表
export const getMatchList = (params: MatchListParams = {}): Promise<MatchListResult> => {
  return request.get<unknown, { code: number; success: boolean; total: number; list: Match[] }>('/api/admin/match/list', { params })
    .then(res => ({
      total: res.total || 0,
      list: res.list || []
    }))
}