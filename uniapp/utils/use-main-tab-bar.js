// Tab 页接入主导航：显示时同步选中态、主题、消息角标和扫码回调，页面显示期间计数或主题变化时重新同步。
// 纯逻辑与平台分支见 main-tab-bar.js；本文件只负责与 store、页面生命周期衔接。
import { computed, onUnmounted, watch } from 'vue'
import { onHide, onShow } from '@dcloudio/uni-app'

import { useThemeStore } from '@/store/theme.js'
import { useUserStore } from '@/store/user.js'
import {
  getMainTabBarHidden,
  resolveMainTabBarState,
  resolveMessageBadge,
  setMainTabBarVisible as applyMainTabBarVisible,
  syncMainTabBar
} from '@/utils/main-tab-bar.js'
import { startQuickScan } from '@/utils/quick-scan-entry.js'
import { useMessageCounts } from '@/utils/use-message-counts.js'

// 弹层覆盖底部区域时隐藏 TabBar，关闭或页面卸载时恢复。
export const setMainTabBarVisible = (visible) => applyMainTabBarVisible(visible, { uniApi: uni })

const handleScan = () => startQuickScan()

// 小程序的 TabBar 实例在页面首次显示时偶尔还没挂好，最多补推两次。
const MAX_RETRIES = 2
const RETRY_DELAY = 80

export const useMainTabBar = (tabKey) => {
  const userStore = useUserStore()
  const themeStore = useThemeStore()
  // 计数已绑定当前 userId + authGeneration：切号后、新身份首次读取前，旧账号的计数不会出现在角标上。
  const { unreadCount, friendRequestCount, challengeCount } = useMessageCounts()

  const state = computed(() => resolveMainTabBarState({
    tabKey,
    isDarkMode: themeStore.isDarkMode,
    badge: resolveMessageBadge({
      isLoggedIn: userStore.isLoggedIn,
      unreadCount: unreadCount.value,
      friendRequestCount: friendRequestCount.value,
      challengeCount: challengeCount.value
    })
  }))

  let active = false
  let retriesLeft = 0

  const push = () => {
    if (!active) return
    const applied = syncMainTabBar({
      state: { ...state.value, hidden: getMainTabBarHidden() },
      onScan: handleScan,
      uniApi: uni
    })
    if (!applied && retriesLeft > 0) {
      retriesLeft -= 1
      setTimeout(push, RETRY_DELAY)
    }
  }

  onShow(() => {
    active = true
    retriesLeft = MAX_RETRIES
    push()
  })
  onHide(() => {
    active = false
  })
  onUnmounted(() => {
    active = false
  })

  watch(state, push)
}
