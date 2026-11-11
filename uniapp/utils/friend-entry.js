import { normalizeVipLevel } from './member-identity.js'

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

export const buildFriendOpponentRecordUrl = (item = {}) => {
  const friendUserId = resolveFriendUserId(item)
  const query = []

  if (friendUserId > 0) {
    query.push(`target_user_id=${friendUserId}`)
  }

  query.push(`target_name=${encodeURIComponent(item.nickname || '球友')}`)
  query.push(`target_avatar=${encodeURIComponent(item.avatar || '')}`)

  return `/subPages/user/opponentRecord?${query.join('&')}`
}

export const buildFriendHomepageUrl = (item = {}) => {
  const friendUserId = resolveFriendUserId(item)
  const query = []

  if (friendUserId > 0) {
    query.push(`opponent_id=${friendUserId}`)
  }

  query.push(`opponent_name=${encodeURIComponent(item.nickname || '球友')}`)
  query.push(`opponent_avatar=${encodeURIComponent(item.avatar || '')}`)
  query.push(`rank_name=${encodeURIComponent(item.rank_name || '')}`)
  // 会员外观只在有效会员时透传，主页据此展示头像框与 VIP 标识。
  const vipLevel = normalizeVipLevel(item.vip_level)
  if (vipLevel > 0) {
    query.push(`vip_level=${vipLevel}`)
    query.push(`rank_level=${Math.max(0, Math.trunc(Number(item.rank_level)) || 0)}`)
  }

  return `/subPages/social/friendHomepage?${query.join('&')}`
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

export const buildUnblacklistPayload = (item = {}) => ({
  friend_user_id: resolveFriendUserId(item)
})

// 好友申请卡片里申请人 ID 可能在 from_user_id，拉黑时按同一规范取值。
export const resolveFriendRequestUserId = (item = {}) => (
  resolveFriendUserId({ ...item, user_id: item.from_user_id || item.user_id })
)

export const normalizeSearchResultItem = (item = {}) => {
  if (item.status === 'friend') return item
  if (item.status === 'pending' || item.has_pending_request) {
    return { ...item, status: 'pending' }
  }
  return item
}

export const removeBlacklistItem = (list = [], userId = 0) => (
  list.filter((item) => resolveFriendUserId(item) !== Number(userId))
)

export const buildUnblacklistConfirmCopy = (nickname = '') => (
  `确定将「${nickname || '球友'}」移出黑名单吗？移出后对方可以重新搜索并添加你。`
)
