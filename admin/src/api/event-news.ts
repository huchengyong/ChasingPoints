import request from '@/utils/request'

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
  stage_text: string
  result_text: string
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
  stage_text: string
  result_text: string
  featured: boolean
  sort_time: string
}

export interface EventNewsUpdatePayload extends EventNewsFormPayload {
  event_news_id: number
}

export interface EventNewsPublishPayload {
  event_news_id: number
  published: boolean
}

export interface EventNewsDeletePayload {
  event_news_id: number
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
