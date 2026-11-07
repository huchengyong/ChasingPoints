import { defineStore } from 'pinia'
import { applyRuntimeTheme } from '@/utils/theme-application.js'
import {
  THEME_MODE_STORAGE_KEY,
  normalizeThemeMode,
  readStoredThemeMode,
  resolveDarkMode,
  resolveSystemDarkMode
} from '@/utils/theme-preference.js'

// 主题变化事件名称
export const THEME_CHANGE_EVENT = 'themeChanged'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    themeMode: readStoredThemeMode(uni),
    systemIsDark: false
  }),

  getters: {
    isDarkMode: (state) => resolveDarkMode(state.themeMode, state.systemIsDark)
  },

  actions: {
    /**
     * 初始化当前系统主题，同时保留用户已保存的模式。
     */
    initializeTheme(themeInfo = {}) {
      this.themeMode = readStoredThemeMode(uni)
      this.systemIsDark = resolveSystemDarkMode(themeInfo, this.systemIsDark)
    },

    /**
     * 记录系统主题变化。只有“跟随系统”模式会改变页面主题。
     * @param {boolean} isDark 是否为深色模式
     */
    setThemeFromSystem(isDark) {
      const oldValue = this.isDarkMode
      this.systemIsDark = Boolean(isDark)

      if (oldValue !== this.isDarkMode) {
        this.broadcastThemeChange()
      }
    },

    /**
     * 设置应用主题模式：跟随系统、浅色或深色。
     */
    setThemeMode(mode) {
      const normalizedMode = normalizeThemeMode(mode)
      const oldValue = this.isDarkMode

      this.themeMode = normalizedMode
      uni.setStorageSync(THEME_MODE_STORAGE_KEY, normalizedMode)

      if (oldValue !== this.isDarkMode) {
        this.broadcastThemeChange()
      } else {
        this.applyNavigationBarTheme()
      }
    },

    /**
     * 广播主题变化事件，通知所有监听的页面
     * 同时更新导航栏和 TabBar 样式
     */
    broadcastThemeChange() {
      console.log('[ThemeStore] 广播主题变化事件, isDarkMode:', this.isDarkMode)

      // 更新导航栏样式
      this.applyNavigationBarTheme()

      // 广播给页面
      uni.$emit(THEME_CHANGE_EVENT, { isDarkMode: this.isDarkMode })
    },

    /**
     * 应用导航栏和 TabBar 主题样式
     */
    applyNavigationBarTheme() {
      applyRuntimeTheme({
        uniApi: uni,
        isDarkMode: this.isDarkMode,
        animationDuration: 300
      })
    },

    /**
     * 同步检测并更新主题
     * 用于页面 onShow 时应用当前主题状态（不会改变主题值）
     * 尊重用户偏好，不会强制同步系统主题
     */
    syncTheme() {
      this.applyNavigationBarTheme()
    }
  }
})
