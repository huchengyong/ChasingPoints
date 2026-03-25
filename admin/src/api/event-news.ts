import request from '@/utils/request'

export interface EventNewsStageItem {
  id: number
  event_id: number
  stage_name: string
  stage_order: number
  start_time: string
  end_time: string
  status: number
  result_text: string
  sort_time: string
  created_at: string
  updated_at: string
}

export interface EventNewsItem {
  id: number
  title: string
  game_type: number
  source_type: string
  source_name: string
  source_url: string
  cover_image: string
  summary: string
  content: string
  country: string
  city: string
  venue: string
  start_time: string
  end_time: string
  status: number
  current_stage_text: string
  latest_result_text: string
  stage_count: number
  stages: EventNewsStageItem[]
  featured: boolean
  sort_time: string
  published: boolean
  published_at: string
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
  game_type: number
  source_type: string
  source_name: string
  source_url: string
  cover_image: string
  summary: string
  content: string
  country: string
  city: string
  venue: string
  start_time: string
  end_time: string
  status: number
  featured: boolean
  sort_time: string
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

export interface EventNewsStageFormPayload {
  event_id: number
  stage_name: string
  stage_order: number
  start_time: string
  end_time: string
  status: number
  result_text: string
  sort_time: string
}

export interface EventNewsStageUpdatePayload extends EventNewsStageFormPayload {
  stage_id: number
}

export interface EventNewsStageDeletePayload {
  stage_id: number
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

export const createEventNewsStage = (data: EventNewsStageFormPayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/stage/create', data)
}

export const updateEventNewsStage = (data: EventNewsStageUpdatePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/stage/update', data)
}

export const deleteEventNewsStage = (data: EventNewsStageDeletePayload): Promise<WriteResult> => {
  return request.post('/api/admin/event-news/stage/delete', data)
}
