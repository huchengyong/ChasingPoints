import { buildHonorWallUrl } from './honor-wall.js'

export const MATCH_REWARD_MAX_ATTEMPTS = 3
export const MATCH_REWARD_RETRY_DELAY_MS = 800

const normalizeRewardItem = (item = {}) => ({
  id: Number(item.achievement_id || 0),
  key: item.achievement_key || '',
  achievementName: item.achievement_name || '新成就',
  description: item.description || '这项成就已永久记录到荣誉墙',
  icon: item.icon || '',
  rewardTitleId: Number(item.reward_title_id || 0),
  rewardTitleName: item.reward_title_name || '',
  unlockedAt: item.unlocked_at || ''
})

export const resolveMatchRewardSummary = (response = {}, { attempt = 1, maxAttempts = MATCH_REWARD_MAX_ATTEMPTS } = {}) => {
  const status = String(response?.status || '').toLowerCase()
  const rewards = Array.isArray(response?.list) ? response.list.map(normalizeRewardItem) : []

  if (status === 'pending') {
    return {
      state: 'pending',
      visible: true,
      rewards: [],
      message: response?.message || '本场荣誉奖励仍在处理中',
      shouldRetry: attempt < maxAttempts
    }
  }

  if (response?.success && status === 'ready' && rewards.length > 0) {
    return {
      state: 'ready',
      visible: true,
      rewards,
      message: `本场首次解锁 ${rewards.length} 项荣誉`,
      shouldRetry: false
    }
  }

  if (response?.success && status === 'ready') {
    return {
      state: 'empty',
      visible: false,
      rewards: [],
      message: '',
      shouldRetry: false
    }
  }

  return {
    state: 'error',
    visible: false,
    rewards: [],
    message: response?.message || '奖励摘要暂时不可用',
    shouldRetry: false
  }
}

export const buildMatchRewardHonorWallUrl = (gameType) => buildHonorWallUrl({ gameType })
