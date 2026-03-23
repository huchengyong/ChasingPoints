import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getFriendRequests } from '@/api/friend.js'
import { shouldFetchAuthState } from '@/utils/auth-guards.js'

export const useFriendRequestStore = defineStore('friendRequest', () => {
  const pendingCount = ref(0)

  const fetchPendingCount = async () => {
    if (!shouldFetchAuthState(Boolean(uni.getStorageSync('token')))) {
      pendingCount.value = 0
      return
    }

    try {
      const res = await getFriendRequests({ page: 1, page_size: 1 })
      pendingCount.value = res.total || 0
    } catch (error) {
      console.error('获取好友申请数失败:', error)
    }
  }

  const decrementPendingCount = () => {
    if (pendingCount.value > 0) pendingCount.value--
  }

  const clearPendingCount = () => {
    pendingCount.value = 0
  }

  return {
    pendingCount,
    fetchPendingCount,
    decrementPendingCount,
    clearPendingCount
  }
})
