/**
 * 通知消息状态管理
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getUnreadCount } from '@/api/notification.js'
import { shouldFetchUnreadCount } from '@/utils/notification.js'

const normalizeIdentity = (identity = {}) => ({
  userId: Number(identity?.userId) || 0,
  authGeneration: Number.isFinite(Number(identity?.authGeneration))
    ? Number(identity.authGeneration)
    : -1
})

export const useNotificationStore = defineStore('notification', () => {
  const ownerUserId = ref(0)
  const authGeneration = ref(-1)
  const unreadCount = ref(0)
  const inFlight = ref(null)

  const matchesIdentity = (identity) => {
    const expected = normalizeIdentity(identity)
    return ownerUserId.value === expected.userId && authGeneration.value === expected.authGeneration
  }

  const clear = () => {
    ownerUserId.value = 0
    authGeneration.value = -1
    unreadCount.value = 0
    inFlight.value = null
  }

  const ensureIdentity = (identity) => {
    const expected = normalizeIdentity(identity)
    if (!expected.userId) {
      clear()
      return false
    }
    if (!matchesIdentity(expected)) {
      ownerUserId.value = expected.userId
      authGeneration.value = expected.authGeneration
      unreadCount.value = 0
      inFlight.value = null
    }
    return true
  }

  const fetchUnreadCount = async (identity) => {
    if (!ensureIdentity(identity) || !shouldFetchUnreadCount(Boolean(uni.getStorageSync('token')))) {
      return 0
    }
    if (inFlight.value) return inFlight.value
    const expected = normalizeIdentity(identity)
    const request = getUnreadCount()
      .then((res) => {
        if (matchesIdentity(expected)) {
          unreadCount.value = Math.max(0, Number(res?.count) || 0)
        }
        return unreadCount.value
      })
      .catch((error) => {
        console.error('获取未读通知数失败:', error)
        return unreadCount.value
      })
      .finally(() => {
        if (inFlight.value === request) inFlight.value = null
      })
    inFlight.value = request
    return request
  }

  const setUnreadCount = (count, identity) => {
    if (identity && !ensureIdentity(identity)) return
    unreadCount.value = Math.max(0, Number(count) || 0)
  }

  const decrementUnread = (identity) => {
    if (identity && !ensureIdentity(identity)) return
    if (unreadCount.value > 0) unreadCount.value--
  }

  const clearUnread = (identity) => {
    if (identity) {
      if (!ensureIdentity(identity)) return
      unreadCount.value = 0
      return
    }
    clear()
  }

  return {
    ownerUserId,
    authGeneration,
    unreadCount,
    ensureIdentity,
    matchesIdentity,
    fetchUnreadCount,
    setUnreadCount,
    decrementUnread,
    clearUnread,
    clear
  }
})
