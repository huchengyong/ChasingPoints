import { MAIN_TABS, resolveMainTabBarPlatform } from './main-tab-bar.js'

export const TAB_BAR_ROUTES = MAIN_TABS.map((tab) => tab.route)

const TAB_BAR_ITEMS = MAIN_TABS.map((tab) => [tab.icon, `${tab.icon}-selected`])

const normalizeRoute = (route = '') => String(route).replace(/^\/+/, '').split('?')[0]

export const isConfiguredTabBarRoute = (route) => TAB_BAR_ROUTES.includes(normalizeRoute(route))

const getCurrentRoute = () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  return currentPage?.route || ''
}

const getThemeStyles = ({ isDarkMode, animationDuration }) => {
  const suffix = isDarkMode ? 'dark' : 'light'

  if (isDarkMode) {
    return {
      navigationBar: {
        frontColor: '#ffffff',
        backgroundColor: '#141109',
        animation: {
          duration: animationDuration,
          timingFunc: 'easeIn'
        }
      },
      tabBar: {
        backgroundColor: '#141109',
        borderStyle: 'white',
        color: '#c6b78c',
        selectedColor: '#E0AE12'
      },
      background: '#141109',
      backgroundTextStyle: 'light',
      tabBarItems: TAB_BAR_ITEMS.map(([icon, selectedIcon]) => ({
        iconPath: `/static/tabbar/${icon}_${suffix}.png`,
        selectedIconPath: `/static/tabbar/${selectedIcon}_${suffix}.png`
      }))
    }
  }

  return {
    navigationBar: {
      frontColor: '#000000',
      backgroundColor: '#F7F4EC',
      animation: {
        duration: animationDuration,
        timingFunc: 'easeIn'
      }
    },
    tabBar: {
      backgroundColor: '#F7F4EC',
      borderStyle: 'black',
      color: '#6E6242',
      selectedColor: '#E0AE12'
    },
    background: '#F7F4EC',
    backgroundTextStyle: 'dark',
    tabBarItems: TAB_BAR_ITEMS.map(([icon, selectedIcon]) => ({
      iconPath: `/static/tabbar/${icon}_${suffix}.png`,
      selectedIconPath: `/static/tabbar/${selectedIcon}_${suffix}.png`
    }))
  }
}

export const applyRuntimeTheme = ({
  uniApi,
  route = getCurrentRoute(),
  isDarkMode,
  animationDuration = 300
} = {}) => {
  const themeStyles = getThemeStyles({ isDarkMode, animationDuration })

  if (typeof uniApi.setNavigationBarColor === 'function') {
    uniApi.setNavigationBarColor(themeStyles.navigationBar)
  }

  if (typeof uniApi.setBackgroundColor === 'function') {
    uniApi.setBackgroundColor({
      backgroundColor: themeStyles.background,
      backgroundColorTop: themeStyles.background,
      backgroundColorBottom: themeStyles.background
    })
  }

  if (typeof uniApi.setBackgroundTextStyle === 'function') {
    uniApi.setBackgroundTextStyle({ textStyle: themeStyles.backgroundTextStyle })
  }

  // 小程序启用自定义 TabBar 后 setTabBarStyle / setTabBarItem 失效，主题由各 Tab 页经 use-main-tab-bar 下发。
  if (isConfiguredTabBarRoute(route) && resolveMainTabBarPlatform(uniApi) !== 'custom') {
    if (typeof uniApi.setTabBarStyle === 'function') {
      uniApi.setTabBarStyle(themeStyles.tabBar)
    }

    if (typeof uniApi.setTabBarItem === 'function') {
      themeStyles.tabBarItems.forEach((item, index) => {
        uniApi.setTabBarItem({ index, ...item })
      })
    }
  }
}
