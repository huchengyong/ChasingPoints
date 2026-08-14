/**
 * 会话持久化读写（纯逻辑，依赖注入 storage 以便单测）
 */

export const SESSION_STORAGE_KEYS = {
  token: 'token',
  refreshToken: 'refreshToken',
  loginMethod: 'loginMethod',
  userStore: 'user-store'
}

export const readStoredSession = (storage = {}) => {
  const token = typeof storage.getStorageSync === 'function'
    ? storage.getStorageSync(SESSION_STORAGE_KEYS.token)
    : ''
  const refreshToken = typeof storage.getStorageSync === 'function'
    ? storage.getStorageSync(SESSION_STORAGE_KEYS.refreshToken)
    : ''
  const normalizedToken = typeof token === 'string' ? token : ''
  return {
    token: normalizedToken,
    refreshToken: typeof refreshToken === 'string' ? refreshToken : '',
    hasSession: Boolean(normalizedToken)
  }
}

export const clearUserScopedRuntimeState = ({
  rankStore,
  notificationStore,
  friendRequestStore,
  activityStore,
  userOverviewStore,
  userDataInvalidationStore,
  publicReadStore,
  userSocket
} = {}) => {
  rankStore?.clear()
  if (typeof notificationStore?.clear === 'function') {
    notificationStore.clear()
  } else {
    notificationStore?.clearUnread()
  }
  if (typeof friendRequestStore?.clear === 'function') {
    friendRequestStore.clear()
  } else {
    friendRequestStore?.clearPendingCount()
  }
  activityStore?.clear()
  userOverviewStore?.clear()
  userDataInvalidationStore?.clear()
  publicReadStore?.clearLeaderboard()
  userSocket?.disconnect()
}

/**
 * 清除全部持久化身份状态。必须同时移除直接存储键与 Pinia 持久化
 * 的 user-store，避免下次启动从任一键恢复旧用户身份。
 */
export const clearStoredSession = (storage = {}) => {
  if (typeof storage.removeStorageSync !== 'function') return
  storage.removeStorageSync(SESSION_STORAGE_KEYS.token)
  storage.removeStorageSync(SESSION_STORAGE_KEYS.refreshToken)
  storage.removeStorageSync(SESSION_STORAGE_KEYS.loginMethod)
  storage.removeStorageSync(SESSION_STORAGE_KEYS.userStore)
}
