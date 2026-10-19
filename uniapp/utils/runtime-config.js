const NETWORK_CONFIG_BY_ENV = {
  development: {
    httpBaseUrl: 'https://api-zhuifen.kekemate.cn',
    wsBaseUrl: 'wss://api-zhuifen.kekemate.cn'
  },
  production: {
    httpBaseUrl: 'https://api.zhuifen.cn',
    wsBaseUrl: 'wss://ws.zhuifen.cn'
  }
}

const normalizeBaseUrl = (value = '') => value.replace(/\/+$/, '')

// process.env.NODE_ENV 由 UniApp 编译器在构建期替换；小程序与 App 运行时通常没有 process 全局，不能用 typeof 守卫。
const resolveAppEnv = () => (process.env.NODE_ENV === 'production' ? 'production' : 'development')

export const resolveNetworkConfig = ({ env = resolveAppEnv(), override = {} } = {}) => {
  const envConfig = NETWORK_CONFIG_BY_ENV[env] || NETWORK_CONFIG_BY_ENV.development
  const httpBaseUrl = normalizeBaseUrl(override.httpBaseUrl || envConfig.httpBaseUrl)
  const wsBaseUrl = normalizeBaseUrl(override.wsBaseUrl || envConfig.wsBaseUrl)

  return {
    httpBaseUrl,
    wsBaseUrl
  }
}

export const NETWORK_CONFIG = resolveNetworkConfig()
