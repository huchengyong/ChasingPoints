export const resolveFriendUserId = (item = {}) => {
  const rawId = item.user_id ?? item.friend_user_id ?? item.friend_id ?? item.id ?? 0
  const parsed = Number(rawId)

  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0
  }

  return parsed
}

export const normalizeFriendListItem = (item = {}) => ({
  ...item,
  friend_user_id: resolveFriendUserId(item)
})

export const buildFriendH2HUrl = (item = {}) => {
  const friendUserId = resolveFriendUserId(item)
  const query = []

  if (friendUserId > 0) {
    query.push(`opponent_id=${friendUserId}`)
  }

  query.push(`opponent_name=${encodeURIComponent(item.nickname || '球友')}`)

  return `/subPages/user/h2hRecord?${query.join('&')}`
}

export const buildFriendPkReportUrl = (item = {}) => {
  const friendUserId = resolveFriendUserId(item)
  const query = []

  if (friendUserId > 0) {
    query.push(`opponent_id=${friendUserId}`)
  }

  query.push(`opponent_name=${encodeURIComponent(item.nickname || '球友')}`)
  query.push(`opponent_avatar=${encodeURIComponent(item.avatar || '')}`)

  return `/subPages/social/pkReport?${query.join('&')}`
}

export const buildDeleteFriendPayload = (item = {}) => ({
  friend_user_id: resolveFriendUserId(item)
})

export const buildBlacklistFriendPayload = (item = {}) => ({
  friend_user_id: resolveFriendUserId(item)
})
