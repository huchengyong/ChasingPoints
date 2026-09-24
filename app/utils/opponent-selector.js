import { resolveFriendUserId } from './friend-entry.js'

export const buildOpponentSelectionList = ({ friends = [], matches = [] } = {}) => {
  const seen = new Set()
  const result = []

  for (const item of Array.isArray(friends) ? friends : []) {
    const userId = resolveFriendUserId(item)
    if (!userId || seen.has(userId)) continue
    seen.add(userId)
    result.push({
      user_id: userId,
      nickname: item.nickname || '球友',
      avatar: item.avatar || '',
      source: 'friend'
    })
  }

  for (const item of Array.isArray(matches) ? matches : []) {
    const userId = Number(item.opponent_id || 0)
    if (!Number.isFinite(userId) || userId <= 0 || seen.has(userId)) continue
    seen.add(userId)
    result.push({
      user_id: userId,
      nickname: item.opponent_name || '球友',
      avatar: item.opponent_avatar || '',
      source: 'recent'
    })
  }

  return result
}
