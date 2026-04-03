const NETWORK_CONFIG_BY_ENV = {
  development: {
    httpBaseUrl: 'https://api-zhuifen.kekemate.cn'
  },
  production: {
    httpBaseUrl: 'https://api-bm.dianzaozao.com'
  }
}

const normalizeBaseUrl = (value = '') => value.replace(/\/+$/, '')

export const inferWsBaseUrl = (httpBaseUrl = '') => {
  if (!httpBaseUrl) {
    return ''
  }

  return normalizeBaseUrl(httpBaseUrl)
    .replace(/^https:\/\//, 'wss://')
    .replace(/^http:\/\//, 'ws://')
}

const resolveAppEnv = () => {
  if (typeof process !== 'undefined' && process?.env?.NODE_ENV === 'production') {
    return 'production'
  }

  return 'development'
}

export const resolveNetworkConfig = ({ env = resolveAppEnv(), override = {} } = {}) => {
  const envConfig = NETWORK_CONFIG_BY_ENV[env] || NETWORK_CONFIG_BY_ENV.development
  const httpBaseUrl = normalizeBaseUrl(override.httpBaseUrl || envConfig.httpBaseUrl)
  const wsBaseUrl = normalizeBaseUrl(override.wsBaseUrl || inferWsBaseUrl(httpBaseUrl))

  return {
    httpBaseUrl,
    wsBaseUrl
  }
}

export const NETWORK_CONFIG = resolveNetworkConfig()
