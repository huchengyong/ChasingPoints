// 消息 Tab 的纯逻辑：通知列表何时需要重新加载、顶部入口条目，以及按身份取共享计数。
export const NOTIFICATION_REFRESH_DEBOUNCE_MS = 500

// load：需要重新加载首屏；wait：该身份的加载已在进行；reuse：沿用已加载的列表。
// 只在身份变化（切号、重新登录）或 notification scope 失效时回源，避免每次切回 Tab 都请求。
export const resolveNotificationReload = ({
  identityKey,
  loadedIdentityKey,
  requestedIdentityKey,
  loadedScopeVersion,
  scopeVersion
}) => {
  if (loadedIdentityKey !== identityKey) {
    return requestedIdentityKey === identityKey ? 'wait' : 'load'
  }
  return loadedScopeVersion === scopeVersion ? 'reuse' : 'load'
}

const toCount = (value) => {
  const count = Number(value)
  return Number.isFinite(count) && count > 0 ? Math.floor(count) : 0
}

export const formatEntryCount = (value) => {
  const count = toCount(value)
  if (count === 0) return ''
  return count > 99 ? '99+' : String(count)
}

export const buildMessageEntries = ({ friendRequestCount = 0, challengeCount = 0 } = {}) => {
  const friends = toCount(friendRequestCount)
  const invites = toCount(challengeCount)
  return [
    {
      key: 'friend-requests',
      title: '好友申请',
      icon: 'personadd',
      url: '/subPages/social/friendRequests',
      countText: formatEntryCount(friends),
      desc: friends > 0 ? `${friends} 条待处理` : '暂无待处理申请'
    },
    {
      key: 'challenge-invites',
      title: '约球邀请',
      icon: 'email',
      url: '/subPages/social/challenges?tab=received',
      countText: formatEntryCount(invites),
      desc: invites > 0 ? `${invites} 条待回应` : '暂无待回应的邀请'
    }
  ]
}

// 共享计数 store 带有所属身份：切号后、新身份首次读取前，旧账号的计数不得显示。
export const ownedCount = (store, identity, count) => (
  store.matchesIdentity(identity) ? toCount(count) : 0
)
