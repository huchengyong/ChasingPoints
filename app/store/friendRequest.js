import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getFriendRequests } from '@/api/friend.js'
import { shouldFetchAuthState } from '@/utils/auth-guards.js'

const normalizeIdentity = (identity = {}) => ({
  userId: Number(identity?.userId) || 0,
  authGeneration: Number.isFinite(Number(identity?.authGeneration))
    ? Number(identity.authGeneration)
    : -1
})

export const useFriendRequestStore = defineStore('friendRequest', () => {
  const ownerUserId = ref(0)
  const authGeneration = ref(-1)
  const pendingCount = ref(0)
  const inFlight = ref(null)

  const matchesIdentity = (identity) => {
    const expected = normalizeIdentity(identity)
    return ownerUserId.value === expected.userId && authGeneration.value === expected.authGeneration
  }

  const clear = () => {
    ownerUserId.value = 0
    authGeneration.value = -1
    pendingCount.value = 0
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
      pendingCount.value = 0
      inFlight.value = null
    }
    return true
  }

  const fetchPendingCount = async (identity) => {
    if (!ensureIdentity(identity) || !shouldFetchAuthState(Boolean(uni.getStorageSync('token')))) {
      return 0
    }
    if (inFlight.value) return inFlight.value
    const expected = normalizeIdentity(identity)
    const request = getFriendRequests({ page: 1, page_size: 1 })
      .then((res) => {
        if (matchesIdentity(expected)) {
          pendingCount.value = Math.max(0, Number(res?.total) || 0)
        }
        return pendingCount.value
      })
      .catch((error) => {
        console.error('获取好友申请数失败:', error)
        return pendingCount.value
      })
      .finally(() => {
        if (inFlight.value === request) inFlight.value = null
      })
    inFlight.value = request
    return request
  }

  const setPendingCount = (count, identity) => {
    if (identity && !ensureIdentity(identity)) return
    pendingCount.value = Math.max(0, Number(count) || 0)
  }

  const decrementPendingCount = (identity) => {
    if (identity && !ensureIdentity(identity)) return
    if (pendingCount.value > 0) pendingCount.value--
  }

  const clearPendingCount = (identity) => {
    if (identity) {
      if (!ensureIdentity(identity)) return
      pendingCount.value = 0
      return
    }
    clear()
  }

  return {
    ownerUserId,
    authGeneration,
    pendingCount,
    ensureIdentity,
    matchesIdentity,
    fetchPendingCount,
    setPendingCount,
    decrementPendingCount,
    clearPendingCount,
    clear
  }
})
