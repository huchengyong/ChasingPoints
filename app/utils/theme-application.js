export const TAB_BAR_ROUTES = [
  'pages/index/index',
  'pages/match/index',
  'pages/social/index',
  'pages/user/index'
]

const normalizeRoute = (route = '') => String(route).replace(/^\/+/, '').split('?')[0]

export const isConfiguredTabBarRoute = (route) => TAB_BAR_ROUTES.includes(normalizeRoute(route))

const getCurrentRoute = () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  return currentPage?.route || ''
}

const getThemeStyles = ({ isDarkMode, animationDuration }) => {
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
      }
    }
  }

  return {
    navigationBar: {
      frontColor: '#000000',
      backgroundColor: '#ffffff',
      animation: {
        duration: animationDuration,
        timingFunc: 'easeIn'
      }
    },
    tabBar: {
      backgroundColor: '#ffffff',
      borderStyle: 'black',
      color: '#64748b',
      selectedColor: '#E0AE12'
    }
  }
}

export const applyRuntimeTheme = ({
  uniApi,
  route = getCurrentRoute(),
  isDarkMode,
  animationDuration = 300
} = {}) => {
  const themeStyles = getThemeStyles({ isDarkMode, animationDuration })

  uniApi.setNavigationBarColor(themeStyles.navigationBar)

  if (isConfiguredTabBarRoute(route)) {
    uniApi.setTabBarStyle(themeStyles.tabBar)
  }
}
