import { defineStore } from 'pinia'

// 主题变化事件名称
export const THEME_CHANGE_EVENT = 'themeChanged'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    isDarkMode: false
  }),

  actions: {
    /**
     * 设置主题（由 App.vue 的 onThemeChange 调用）
     * @param {boolean} isDark 是否为深色模式
     * @param {boolean} broadcast 是否广播主题变化事件
     */
    setTheme(isDark, broadcast = true) {
      const oldValue = this.isDarkMode
      this.isDarkMode = isDark

      // 如果主题发生变化，广播通知所有页面
      if (oldValue !== isDark && broadcast) {
        this.broadcastThemeChange()
      }
    },

    /**
     * 由系统主题变化触发的设置主题
     * 只有在用户没有保存偏好时才跟随系统主题
     * @param {boolean} isDark 是否为深色模式
     */
    setThemeFromSystem(isDark) {
      // 如果用户已保存偏好，则不覆盖
      // if (this.hasUserPreference()) {
      //   console.log('[ThemeStore] 用户已有偏好，不跟随系统主题')
      //   return
      // }
      // console.log('[ThemeStore] 跟随系统主题:', isDark)
      // 设置主题并广播
      this.setTheme(isDark, true)
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
      if (this.isDarkMode) {
        // 暗色主题
        uni.setNavigationBarColor({
          frontColor: '#ffffff',
          backgroundColor: '#141109',
          animation: {
            duration: 300,
            timingFunc: 'easeIn'
          }
        })
        uni.setTabBarStyle({
          backgroundColor: '#141109',
          borderStyle: 'white',
          color: '#c6b78c',
          selectedColor: '#E0AE12'
        })
      } else {
        // 亮色主题
        uni.setNavigationBarColor({
          frontColor: '#000000',
          backgroundColor: '#ffffff',
          animation: {
            duration: 300,
            timingFunc: 'easeIn'
          }
        })
        uni.setTabBarStyle({
          backgroundColor: '#ffffff',
          borderStyle: 'black',
          color: '#64748b',
          selectedColor: '#E0AE12'
        })
      }
    },

    /**
     * 切换主题（用户手动切换时调用）
     * 会保存用户偏好并广播事件
     */
    toggleTheme() {
      this.isDarkMode = !this.isDarkMode
      // 保存用户偏好
      uni.setStorageSync('user_theme_dark', this.isDarkMode)
      // 广播主题变化
      this.broadcastThemeChange()
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
