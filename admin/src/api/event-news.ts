import request from '@/utils/request'

export interface EventNewsItem {
  id: number
  title: string
  source_type: string
  source_name: string
  source_url: string
  cover_image: string
  summary: string
  content: string
  tournament_id: number
  tournament_name: string
  game_type: number
  country: string
  city: string
  venue: string
  start_date: string
  end_date: string
  start_time: string
  end_time: string
  status: number
  current_round_text: string
  latest_result_text: string
  match_count: number
  sort_time: string
  published: boolean
  published_at: string
  created_at: string
  updated_at: string
}

export interface EventNewsMatchItem {
  id: number
  event_id: number
  tournament_id: number
  source_type: string
  source_match_id: string
  round_name: string
  round_order: number
  match_order: number
  start_time: string
  status: number
  best_of: number
  home_player_id: number
  home_player_name: string
  home_player_avatar?: string
  away_player_id: number
  away_player_name: string
  away_player_avatar?: string
  home_score: number
  away_score: number
  winner_side: number
  is_placeholder: boolean
  created_at: string
  updated_at: string
}

export interface EventNewsListParams {
  page?: number
  page_size?: number
  game_type?: number
  status?: number
  published?: number
}

export interface EventNewsListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: EventNewsItem[]
}

export interface EventNewsFormPayload {
  title: string
  tournament_name: string
  game_type: number
  source_type: string
  source_name: string
  source_url: string
  cover_image: string
  summary: string
  content: string
  description: string
  country: string
  city: string
  venue: string
  start_date: string
  end_date?: string
  start_time?: string
  end_time?: string
  status: number
  published: boolean
  sort_time?: string
  tournament_id?: number
}

export interface EventNewsUpdatePayload extends EventNewsFormPayload {
  event_id: number
}

export interface EventNewsPublishPayload {
  event_id: number
  published: boolean
}

export interface EventNewsDeletePayload {
  event_id: number
}

export interface EventNewsMatchListParams {
  event_id: number
}

export interface EventNewsMatchListResult {
  code: number
  success: boolean
  message: string
  list: EventNewsMatchItem[]
}

export interface EventNewsMatchCreatePayload {
  event_id: number
  round_name: string
  round_order: number
  match_order: number
  start_time: string
  status: number
  best_of: number
  home_player_id: number
  home_player_name: string
  away_player_id: number
  away_player_name: string
  home_score: number
  away_score: number
  winner_side: number
  is_placeholder: boolean
  source_type: string
  source_match_id: string
}

export interface EventNewsMatchUpdatePayload extends EventNewsMatchCreatePayload {
  match_id: number
}

export interface EventNewsMatchDeletePayload {
  match_id: number
}

export interface WriteResult {
  code: number
  success: boolean
  message: string
}

export const getEventNewsList = (params: EventNewsListParams = {}): Promise<EventNewsListResult> => {
  return request.get('/api/admin/event-news/list', { params })
}

export const createEventNews = (data: EventNewsFormPayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/create', data)
}

export const updateEventNews = (data: EventNewsUpdatePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/update', data)
}

export const publishEventNews = (data: EventNewsPublishPayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/publish', data)
}

export const deleteEventNews = (data: EventNewsDeletePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/delete', data)
}

export const getEventNewsMatches = (params: EventNewsMatchListParams): Promise<EventNewsMatchListResult> => {
  return request.get('/api/admin/event-news/matches', { params })
}

export const createEventNewsMatch = (data: EventNewsMatchCreatePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/match/create', data)
}

export const updateEventNewsMatch = (data: EventNewsMatchUpdatePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/match/update', data)
}

export const deleteEventNewsMatch = (data: EventNewsMatchDeletePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/match/delete', data)
}
