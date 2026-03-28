import { resolveFriendUserId } from './friend-entry.js'

export const buildChallengePayload = ({
  targetFriend = {},
  gameType = 1,
  message = ''
} = {}) => ({
  to_user_id: resolveFriendUserId(targetFriend),
  game_type: gameType,
  message
})

export const normalizeChallengeListItem = (item = {}, currentUserId = 0) => {
  const fromUserId = Number(item.from_user_id ?? item.fromUserId ?? 0)
  const toUserId = Number(item.to_user_id ?? item.toUserId ?? 0)
  const normalizedUserId = Number(currentUserId) || 0
  const isSent = normalizedUserId > 0 ? fromUserId === normalizedUserId : item.direction === 'sent'

  return {
    ...item,
    direction: isSent ? 'sent' : 'received',
    opponent_id: isSent ? toUserId : fromUserId,
    nickname: isSent
      ? (item.to_nickname ?? item.toNickname ?? '')
      : (item.from_nickname ?? item.fromNickname ?? ''),
    avatar: isSent
      ? (item.to_avatar ?? item.toAvatar ?? '')
      : (item.from_avatar ?? item.fromAvatar ?? '')
  }
}
