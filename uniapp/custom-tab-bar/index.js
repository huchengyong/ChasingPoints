// 微信小程序自定义 TabBar：原生组件，不经过 uni-app 编译，只负责渲染与切换，也不引用项目其他模块。
// 选中态、主题、角标、显隐和扫码回调由各 Tab 页在 onShow / 计数变化时下发，见 utils/main-tab-bar.js。
const TABS = [
  { key: 'index', pagePath: '/pages/index/index', text: '首页' },
  { key: 'tournament', pagePath: '/pages/tournament/index', text: '赛讯' },
  { key: 'message', pagePath: '/pages/message/index', text: '消息' },
  { key: 'user', pagePath: '/pages/user/index', text: '我的' }
]

// 与 utils/theme-preference.js 保持一致：新实例首帧先按用户偏好和系统主题取值，避免首次切到某个 Tab 时闪一下亮色。
const THEME_MODE_STORAGE_KEY = 'theme_mode'

const resolveInitialDark = () => {
  try {
    const mode = wx.getStorageSync(THEME_MODE_STORAGE_KEY)
    if (mode === 'dark') return true
    if (mode === 'light') return false
    const info = wx.getAppBaseInfo ? wx.getAppBaseInfo() : wx.getSystemInfoSync()
    const theme = [info.hostTheme, info.theme, info.osTheme].find((value) => value === 'light' || value === 'dark')
    return theme === 'dark'
  } catch (error) {
    return false
  }
}

const resolveInitialSelected = () => {
  try {
    const pages = getCurrentPages()
    const page = pages[pages.length - 1]
    const route = page ? `/${page.route}` : ''
    return TABS.findIndex((item) => item.pagePath === route)
  } catch (error) {
    return -1
  }
}

Component({
  data: {
    list: TABS,
    selected: -1,
    dark: false,
    hidden: false,
    badgeType: 'none',
    badgeText: ''
  },

  lifetimes: {
    attached() {
      this.setData({ selected: resolveInitialSelected(), dark: resolveInitialDark() })
    }
  },

  methods: {
    // 回调不进 data（setData 不能传函数），由页面同步时注入。
    setScanHandler(handler) {
      this.scanHandler = typeof handler === 'function' ? handler : null
    },

    onTapTab(event) {
      const index = Number(event.currentTarget.dataset.index)
      const target = TABS[index]
      if (!target || index === this.data.selected) return
      wx.switchTab({ url: target.pagePath })
    },

    onTapScan() {
      if (this.scanHandler) this.scanHandler()
    }
  }
})
