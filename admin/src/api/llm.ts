import request from '@/utils/request'

export type LlmApiFormat = 'anthropic' | 'openai'

export interface LlmConfig {
  // 尚未配置时为空字符串
  api_format: LlmApiFormat | ''
  endpoint_url: string
  has_api_key: boolean
  api_key_masked: string
  model: string
  max_tokens: number
}

export interface LlmConfigUpdate {
  api_format: LlmApiFormat
  endpoint_url: string
  // 留空表示沿用已保存的 API Key
  api_key: string
  model: string
  max_tokens: number
}

export const getLlmConfig = async (): Promise<LlmConfig> => {
  return request.get<unknown, LlmConfig>('/api/admin/llm/config')
}

export const updateLlmConfig = async (
  data: LlmConfigUpdate
): Promise<{ code?: number; success?: boolean; message?: string }> => {
  return request.post('/api/admin/llm/config', data)
}
