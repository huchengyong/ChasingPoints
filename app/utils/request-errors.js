/**
 * 请求错误分类与失效会话处理（纯逻辑，可注入依赖测试）
 */

export const SESSION_INVALID_REASON = 'SESSION_INVALID'

export const REQUEST_ERROR_CATEGORY = Object.freeze({
  NETWORK: 'network',
  SESSION: 'session',
  PERMISSION: 'permission',
  NOT_FOUND: 'not-found',
  RATE_LIMIT: 'rate-limit',
  SERVER: 'server',
  BUSINESS: 'business'
})

export const isSessionInvalidResponse = (data = {}) => (
  Boolean(data && typeof data === 'object' && data.reason === SESSION_INVALID_REASON)
)

export const isRefreshSessionCurrent = ({ startedRefreshToken = '', currentRefreshToken = '' } = {}) => (
  Boolean(startedRefreshToken) && startedRefreshToken === currentRefreshToken
)

export const resolveUnauthorizedAction = ({
  requestGeneration,
  currentGeneration,
  requestToken = '',
  currentToken = ''
} = {}) => {
  if (requestGeneration !== currentGeneration) return 'stale'
  if (currentToken && currentToken !== requestToken) return 'replay'
  return 'refresh'
}

export const classifyRequestError = ({ statusCode = 0, data = null, network = false } = {}) => {
  if (network) return REQUEST_ERROR_CATEGORY.NETWORK
  if (statusCode === 401 || isSessionInvalidResponse(data)) return REQUEST_ERROR_CATEGORY.SESSION
  if (statusCode === 403) return REQUEST_ERROR_CATEGORY.PERMISSION
  if (statusCode === 404) return REQUEST_ERROR_CATEGORY.NOT_FOUND
  if (statusCode === 429) return REQUEST_ERROR_CATEGORY.RATE_LIMIT
  if (statusCode >= 500) return REQUEST_ERROR_CATEGORY.SERVER
  return REQUEST_ERROR_CATEGORY.BUSINESS
}

export const isHandledRequestError = (error) => Boolean(error?._isHandled)

export const isSupersededRequestError = (error) => Boolean(error?._isSuperseded)

export const shouldIgnoreRequestError = (error) => (
  isHandledRequestError(error) || isSupersededRequestError(error)
)

export const createRequestError = ({ statusCode = 0, data = null, message = '请求失败', category = 'business', handled = false, silent = false, superseded = false } = {}) => {
  const error = new Error(message)
  error.statusCode = statusCode
  error.responseData = data ?? null
  error.category = category
  if (handled) {
    error._isHandled = true
    error._isSilent = Boolean(silent)
  }
  if (superseded) {
    error._isSuperseded = true
  }
  return error
}

/**
 * 创建按 access token 与 auth generation 隔离的会话失效处理器。
 * 同一会话的并发失败共享清理、提示和导航；旧会话的延迟响应只能拒绝
 * 旧调用方，并以 superseded 标记提示页面不得影响或误报新会话。
 */
export const createSessionInvalidHandler = ({
  logout,
  showToast,
  reLaunch,
  getCurrentToken = () => '',
  getCurrentGeneration = () => undefined,
  isLoggedIn = () => false,
  delay = 800
} = {}) => {
  const inFlightBySession = new Map()

  const buildError = ({ data, message, silent, superseded = false }) => createRequestError({
    statusCode: 401,
    data: data || {
      success: false,
      reason: SESSION_INVALID_REASON,
      message: message || '登录状态已失效，请重新登录'
    },
    message: message || data?.message || '登录状态已失效，请重新登录',
    category: 'session',
    handled: true,
    silent,
    superseded
  })

  const handle = ({ sessionKey = '', sessionGeneration, data = null, message = '', silent = false } = {}) => {
    const normalizedKey = typeof sessionKey === 'string' ? sessionKey : ''
    const hasGeneration = sessionGeneration !== undefined && sessionGeneration !== null
    const actionKey = `${hasGeneration ? String(sessionGeneration) : '__unknown_generation__'}:${normalizedKey || '__missing_access_token__'}`
    let action = inFlightBySession.get(actionKey)

    if (!action) {
      const currentToken = String(getCurrentToken() || '')
      const currentGeneration = getCurrentGeneration()
      const tokenMatches = normalizedKey
        ? currentToken === normalizedKey
        : Boolean(isLoggedIn())
      const generationMatches = !hasGeneration || currentGeneration === sessionGeneration

      if (!tokenMatches || !generationMatches) {
        return Promise.reject(buildError({ data, message, silent, superseded: true }))
      }

      let finishAction
      action = new Promise(resolve => {
        finishAction = resolve
      })
      inFlightBySession.set(actionKey, action)

      ;(async () => {
        let superseded = false
        try {
          logout()
          const invalidatedGeneration = getCurrentGeneration()
          showToast('登录状态已失效，请重新登录')
          await new Promise(resolve => setTimeout(resolve, delay))

          // 用户可能已在提示期间完成新登录；即使新旧 JWT 字符串相同，
          // auth generation 变化也必须阻止旧会话把新用户导航走。
          const latestToken = String(getCurrentToken() || '')
          const latestGeneration = getCurrentGeneration()
          if (latestToken || (invalidatedGeneration !== undefined && latestGeneration !== invalidatedGeneration)) {
            superseded = true
            return
          }
          reLaunch('/pages/login/login')
        } catch (_) {
          // 清理或导航 API 的异常不应覆盖调用方可识别的 session error。
        } finally {
          finishAction({ superseded })
          if (inFlightBySession.get(actionKey) === action) {
            inFlightBySession.delete(actionKey)
          }
        }
      })()
    }

    return action.then(({ superseded = false } = {}) => {
      throw buildError({ data, message, silent, superseded })
    })
  }

  return handle
}
