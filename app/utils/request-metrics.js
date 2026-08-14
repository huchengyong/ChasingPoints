const requestCounts = new Map()

const metricsEnabled = () => {
  if (globalThis.__CHASING_POINTS_REQUEST_METRICS__ === true) return true
  return typeof process !== 'undefined' && process.env?.NODE_ENV !== 'production'
}

export const normalizeRequestRoute = (url = '') => {
  const value = String(url || '').trim()
  if (!value) return '/'

  const path = value.replace(/^[a-z]+:\/\/[^/]+/i, '').split('?')[0] || '/'
  return path.startsWith('/') ? path : `/${path}`
}

export const recordRequest = (method = 'GET', url = '') => {
  if (!metricsEnabled()) return

  const key = `${String(method || 'GET').toUpperCase()} ${normalizeRequestRoute(url)}`
  requestCounts.set(key, (requestCounts.get(key) || 0) + 1)
}

export const getRequestCountSnapshot = () => Object.fromEntries(requestCounts)

export const resetRequestCounts = () => {
  requestCounts.clear()
}
