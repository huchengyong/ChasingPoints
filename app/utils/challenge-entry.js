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

export const resolveChallengeAction = (item = {}) => {
  const matchId = Number(item.match_id || item.matchId || 0)
  if (Number(item.status) === 1 && matchId > 0) {
    return { type: 'view_match', label: '查看对局', match_id: matchId }
  }
  if (Number(item.status) === 1 && matchId === 0) {
    return { type: 'offline_start', label: '线下扫码开局', match_id: 0 }
  }
  return { type: 'report', label: '查看PK报表', match_id: matchId }
}

export const buildChallengeStartContext = (item = {}) => {
  if (resolveChallengeAction({ ...item, status: item.status ?? 1 }).type !== 'offline_start') return null
  return {
    challenge_id: Number(item.id || item.challenge_id || 0),
    opponent_id: Number(item.opponent_id || 0),
    opponent_name: item.opponent_name || item.nickname || '球友',
    opponent_avatar: item.avatar || '',
    game_type: Number(item.game_type) || 3
  }
}
