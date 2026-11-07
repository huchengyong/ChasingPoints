/**
 * 赛事系统 API 接口
 */
import { get, post } from '@/utils/request.js'

/**
 * 获取赛事列表
 * @param {Object} params { page, page_size, city, game_type, status }
 */
export const getTournamentList = (params) => {
  return get('/api/tournament/list', params)
}

/**
 * 获取赛事详情
 * @param {Object} params { tournament_id }
 */
export const getTournamentDetail = (params) => {
  return get('/api/tournament/detail', params)
}

/**
 * 获取赛事对阵图
 * @param {Object} params { tournament_id }
 */
export const getTournamentBracket = (params) => {
  return get('/api/tournament/bracket', params)
}

/**
 * 创建赛事
 * @param {Object} data { name, description, game_type, format, max_players, city, venue_name, start_time }
 */
export const createTournament = (data) => {
  return post('/api/tournament/create', data)
}

/**
 * 报名参加赛事
 * @param {Object} data { tournament_id }
 */
export const joinTournament = (data) => {
  return post('/api/tournament/join', data)
}

/**
 * 退出赛事
 * @param {Object} data { tournament_id }
 */
export const leaveTournament = (data) => {
  return post('/api/tournament/leave', data)
}
