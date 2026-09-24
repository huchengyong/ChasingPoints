import { computed, onUnmounted } from 'vue'
import { onHide, onShow } from '@dcloudio/uni-app'

import { THEME_CHANGE_EVENT, useThemeStore } from '@/store/theme.js'

export const usePageTheme = () => {
  const themeStore = useThemeStore()
  const isDarkMode = computed(() => themeStore.isDarkMode)
  const themeMode = computed(() => themeStore.themeMode)

  const applyTheme = () => {
    themeStore.applyNavigationBarTheme()
  }

  const cleanupThemeListener = () => {
    uni.$off(THEME_CHANGE_EVENT, applyTheme)
  }

  onShow(() => {
    themeStore.syncTheme()
    cleanupThemeListener()
    uni.$on(THEME_CHANGE_EVENT, applyTheme)
  })

  onHide(cleanupThemeListener)
  onUnmounted(cleanupThemeListener)

  return {
    isDarkMode,
    themeMode,
    setThemeMode: (mode) => themeStore.setThemeMode(mode)
  }
}
