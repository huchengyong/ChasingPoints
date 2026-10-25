// 主导航（首页 | 赛讯 | 扫码 | 消息 | 我的）的纯逻辑：Tab 清单、消息角标规则，以及两种实现的状态同步。
// - 微信小程序：原生自定义 TabBar（custom-tab-bar/），每个 Tab 页各有一个实例，状态由当前页经 getTabBar().setData 下发。
// - 鸿蒙（H5 仅用于本地截图预览）：uni 原生 TabBar，角标与显隐走 uni.* 接口；选中态和主题由运行时负责。
// 本文件不引用 store，避免与 theme-application.js 形成循环依赖；与 store 的衔接见 use-main-tab-bar.js。

// 顺序必须与 pages.json 的 tabBar.list、custom-tab-bar/index.js 保持一致。
export const MAIN_TABS = [
  { key: 'index', route: 'pages/index/index', icon: 'index' },
  { key: 'tournament', route: 'pages/tournament/index', icon: 'tournament' },
  { key: 'message', route: 'pages/message/index', icon: 'message' },
  { key: 'user', route: 'pages/user/index', icon: 'user' }
]

const MESSAGE_TAB_INDEX = MAIN_TABS.findIndex((tab) => tab.key === 'message')

const normalizeRoute = (route = '') => String(route).replace(/^\/+/, '').split('?')[0]

export const resolveMainTabIndex = (route) => {
  const normalized = normalizeRoute(route)
  return normalized ? MAIN_TABS.findIndex((tab) => tab.route === normalized) : -1
}

export const isMainTabRoute = (route) => resolveMainTabIndex(route) >= 0

const toCount = (value) => {
  const count = Number(value)
  return Number.isFinite(count) && count > 0 ? Math.floor(count) : 0
}

// 角标只表达一种计数：有未读通知显示数字；没有未读但有待处理事项（好友申请、收到的约球邀请）显示红点。
// 不把三者相加：默认偏好下一次好友申请本身就会产生一条未读通知，相加会让一件事算成两件。
export const resolveMessageBadge = ({
  isLoggedIn = false,
  unreadCount = 0,
  friendRequestCount = 0,
  challengeCount = 0
} = {}) => {
  if (!isLoggedIn) return { type: 'none', text: '' }
  const unread = toCount(unreadCount)
  if (unread > 0) return { type: 'text', text: unread > 99 ? '99+' : String(unread) }
  if (toCount(friendRequestCount) + toCount(challengeCount) > 0) return { type: 'dot', text: '' }
  return { type: 'none', text: '' }
}

export const resolveMainTabBarState = ({
  tabKey,
  isDarkMode = false,
  hidden = false,
  badge = { type: 'none', text: '' }
} = {}) => ({
  selected: MAIN_TABS.findIndex((tab) => tab.key === tabKey),
  dark: Boolean(isDarkMode),
  hidden: Boolean(hidden),
  badgeType: badge.type,
  badgeText: badge.text
})

const platformCache = new WeakMap()

// 沿用登录页等处的做法，用运行时 uniPlatform 判断。uniApi 必须由调用方显式传入：
// 小程序里 uni 是编译期注入的，globalThis.uni 不一定存在；显式注入也便于在 Node 里测试两端分支。
export const resolveMainTabBarPlatform = (uniApi) => {
  if (!uniApi || typeof uniApi !== 'object') return 'native'
  if (platformCache.has(uniApi)) return platformCache.get(uniApi)
  let platform = 'native'
  try {
    platform = uniApi.getSystemInfoSync?.()?.uniPlatform === 'mp-weixin' ? 'custom' : 'native'
  } catch (error) {
    return 'native'
  }
  platformCache.set(uniApi, platform)
  return platform
}

const defaultGetPages = () => (typeof getCurrentPages === 'function' ? getCurrentPages() : [])

// 鸿蒙的原生 TabBar 全局只有一个实例，缓存已下发的状态，避免每次显示 Tab 页都重复调用原生接口
// （重复的 showTabBar 会再播一次动画）。
const nativeApplied = { badgeKey: null, hidden: false }
let desiredHidden = false

export const resetMainTabBarState = () => {
  nativeApplied.badgeKey = null
  nativeApplied.hidden = false
  desiredHidden = false
}

export const getMainTabBarHidden = () => desiredHidden

const CUSTOM_STATE_KEYS = ['selected', 'dark', 'hidden', 'badgeType', 'badgeText']

const syncCustomTabBar = ({ state, onScan, getPages }) => {
  const pages = getPages()
  const page = pages[pages.length - 1]
  const tabBar = page && typeof page.getTabBar === 'function' ? page.getTabBar() : null
  if (!tabBar) return false

  const data = {}
  for (const key of CUSTOM_STATE_KEYS) {
    if (state[key] !== undefined) data[key] = state[key]
  }
  if (Object.keys(data).length) tabBar.setData(data)
  if (typeof onScan === 'function') tabBar.setScanHandler?.(onScan)
  return true
}

const applyNativeBadge = (uniApi, type, text) => {
  const key = `${type}:${text}`
  if (nativeApplied.badgeKey === key) return
  nativeApplied.badgeKey = key
  const index = MESSAGE_TAB_INDEX
  // 鸿蒙的 removeTabBarBadge / hideTabBarRedDot 都会同时清掉数字和红点，所以先清再设。
  if (type === 'text') {
    uniApi.hideTabBarRedDot?.({ index })
    uniApi.setTabBarBadge?.({ index, text })
  } else if (type === 'dot') {
    uniApi.removeTabBarBadge?.({ index })
    uniApi.showTabBarRedDot?.({ index })
  } else {
    uniApi.removeTabBarBadge?.({ index })
    uniApi.hideTabBarRedDot?.({ index })
  }
}

const syncNativeTabBar = ({ state, uniApi }) => {
  if (state.badgeType !== undefined) applyNativeBadge(uniApi, state.badgeType, state.badgeText || '')
  if (state.hidden !== undefined && Boolean(state.hidden) !== nativeApplied.hidden) {
    nativeApplied.hidden = Boolean(state.hidden)
    if (nativeApplied.hidden) uniApi.hideTabBar?.({ animation: true })
    else uniApi.showTabBar?.({ animation: true })
  }
  return true
}

// state 只需包含要变更的键（selected / dark / hidden / badgeType / badgeText）。
// 返回 false 表示小程序里当前页的 TabBar 实例还不可用，调用方可稍后重试。
export const syncMainTabBar = ({
  state = {},
  onScan,
  uniApi,
  getPages = defaultGetPages
} = {}) => {
  if (resolveMainTabBarPlatform(uniApi) === 'custom') {
    return syncCustomTabBar({ state, onScan, getPages })
  }
  return syncNativeTabBar({ state, uniApi })
}

// 供会覆盖底部区域的弹层使用：打开时隐藏、关闭或页面卸载时恢复。
// 记住期望的显隐状态，Tab 页重新显示时的整体同步不会把弹层下的 TabBar 又显示出来。
export const setMainTabBarVisible = (visible, options = {}) => {
  desiredHidden = !visible
  return syncMainTabBar({ ...options, state: { hidden: desiredHidden } })
}
