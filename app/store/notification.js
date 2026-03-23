/**
 * 通知消息状态管理
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getUnreadCount } from '@/api/notification.js'
import { shouldFetchUnreadCount } from '@/utils/notification.js'

export const useNotificationStore = defineStore('notification', () => {
  const unreadCount = ref(0)

  const fetchUnreadCount = async () => {
    if (!shouldFetchUnreadCount(Boolean(uni.getStorageSync('token')))) {
      unreadCount.value = 0
      return
    }

    try {
      const res = await getUnreadCount()
      unreadCount.value = res.count || 0
    } catch (e) {
      console.error('获取未读通知数失败:', e)
    }
  }

  const decrementUnread = () => {
    if (unreadCount.value > 0) unreadCount.value--
  }

  const clearUnread = () => {
    unreadCount.value = 0
  }

  return { unreadCount, fetchUnreadCount, decrementUnread, clearUnread }
})
