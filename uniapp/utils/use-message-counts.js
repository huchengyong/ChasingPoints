// 消息相关的共享计数（未读通知、待处理好友申请、收到的约球邀请），主导航角标和消息 Tab 共用。
// 数值来自 Bootstrap / 活动 Store / WebSocket 维护的共享状态，这里只做身份绑定，不发请求。
import { computed } from 'vue'

import { useActivityStore } from '@/store/activity.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { useNotificationStore } from '@/store/notification.js'
import { useUserStore } from '@/store/user.js'
import { ownedCount } from '@/utils/message-center.js'

export const useMessageCounts = () => {
  const userStore = useUserStore()
  const notificationStore = useNotificationStore()
  const friendRequestStore = useFriendRequestStore()
  const activityStore = useActivityStore()

  const identity = () => ({ userId: userStore.userId, authGeneration: userStore.authGeneration })
  const forLoggedIn = (store, count) => (userStore.isLoggedIn ? ownedCount(store, identity(), count) : 0)

  return {
    unreadCount: computed(() => forLoggedIn(notificationStore, notificationStore.unreadCount)),
    friendRequestCount: computed(() => forLoggedIn(friendRequestStore, friendRequestStore.pendingCount)),
    challengeCount: computed(() => forLoggedIn(activityStore, activityStore.receivedChallengeCount))
  }
}
