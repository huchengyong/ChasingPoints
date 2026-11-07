/**
 * 封装 uni.request 请求工具
 */
import pinia from '@/store/index.js'
import { useUserStore } from '@/store/user.js'
import {
  shouldResolveBusinessResponse,
  unwrapBusinessResponse
} from './request-response.js'
import {
  classifyRequestError,
  createRequestError,
  createSessionInvalidHandler,
  isRefreshSessionCurrent,
  isSessionInvalidResponse,
  resolveUnauthorizedAction
} from './request-errors.js'
import { NETWORK_CONFIG } from './runtime-config.js'
import { recordRequest } from './request-metrics.js'

const BASE_URL = NETWORK_CONFIG.httpBaseUrl
const authRefreshPromises = new Map()

// 会话失效处理按请求 token 与 auth generation 隔离：旧请求不得清除新登录会话。
const handleSessionInvalid = createSessionInvalidHandler({
  logout: () => useUserStore(pinia).logout(),
  getCurrentToken: () => String(uni.getStorageSync('token') || ''),
  getCurrentGeneration: () => useUserStore(pinia).authGeneration,
  isLoggedIn: () => useUserStore(pinia).isLoggedIn,
  showToast: (title) => {
    uni.showToast({ title, icon: 'none' })
  },
  reLaunch: (url) => {
    uni.reLaunch({ url })
  },
  delay: 800
})

// WebSocket 等非 HTTP 通道复用同一会话失效入口，仍以当前 token 与认证代次隔离。
export const handleCurrentSessionInvalid = ({ data = null, message = '', silent = false } = {}) => {
  const userStore = useUserStore(pinia)
  return handleSessionInvalid({
    sessionKey: String(uni.getStorageSync('token') || ''),
    sessionGeneration: userStore.authGeneration,
    data,
    message,
    silent
  })
}

// 请求拦截器
const requestInterceptor = (config, token = '') => {
  if (token) {
    config.header = {
      ...config.header,
      'Authorization': `Bearer ${token}`
    }
  }
  return config
}

const refreshAuthToken = () => {
  const refreshToken = String(uni.getStorageSync('refreshToken') || '')
  if (!refreshToken) {
    return Promise.reject(new Error('缺少刷新令牌'))
  }
  const existingRefresh = authRefreshPromises.get(refreshToken)
  if (existingRefresh) {
    return existingRefresh
  }

  const refreshPromise = new Promise((resolve, reject) => {
    recordRequest('POST', '/api/auth/refresh-token')
    uni.request({
      url: BASE_URL + '/api/auth/refresh-token',
      method: 'POST',
      data: {
        refresh_token: refreshToken
      },
      header: {
        'Content-Type': 'application/json'
      },
      timeout: 30000,
      success: (res) => {
        const { statusCode, data } = res
        if (statusCode < 200 || statusCode >= 300 || !data || typeof data !== 'object' || !shouldResolveBusinessResponse(data)) {
          reject(createRequestError({
            statusCode,
            data: typeof data === 'object' ? data : null,
            message: data?.message || data?.msg || '登录状态已失效',
            category: classifyRequestError({ statusCode, data })
          }))
          return
        }

        const authPayload = unwrapBusinessResponse(data)
        if (!authPayload?.access_token) {
          reject(new Error('刷新登录态响应无效'))
          return
        }

        // logout、重新登录或另一轮 refresh 已替换 refresh token 时，旧响应不得写回。
        if (!isRefreshSessionCurrent({
          startedRefreshToken: refreshToken,
          currentRefreshToken: String(uni.getStorageSync('refreshToken') || '')
        })) {
          reject(new Error('登录状态已变更'))
          return
        }

        const userStore = useUserStore(pinia)
        userStore.refreshAuth(authPayload)
        resolve(authPayload)
      },
      fail: (err) => {
        reject(new Error(err.errMsg || '刷新登录态失败'))
      }
    })
  }).finally(() => {
    if (authRefreshPromises.get(refreshToken) === refreshPromise) {
      authRefreshPromises.delete(refreshToken)
    }
  })

  authRefreshPromises.set(refreshToken, refreshPromise)
  return refreshPromise
}

// 响应拦截器。silent 只抑制瞬时网络反馈；已确认失效会话始终走全局重新登录。
const responseInterceptor = (response, { silent = false, sessionKey = '', sessionGeneration } = {}) => {
  const { statusCode, data } = response

  // HTTP 状态码处理
  if (statusCode === 401) {
    return handleSessionInvalid({ sessionKey, sessionGeneration, data, silent })
  }

  if (statusCode >= 200 && statusCode < 300) {
    if (data == null) {
      return Promise.reject(createRequestError({
        statusCode,
        message: '响应数据为空',
        category: 'server'
      }))
    }

    if (typeof data !== 'object') {
      return Promise.resolve(data)
    }

    // 业务状态码处理
    if (shouldResolveBusinessResponse(data)) {
      return Promise.resolve(unwrapBusinessResponse(data))
    } else {
      return Promise.reject(createRequestError({
        statusCode,
        data,
        message: data.message || data.msg || '请求失败',
        category: classifyRequestError({ statusCode, data })
      }))
    }
  }

  return Promise.reject(createRequestError({
    statusCode,
    data: typeof data === 'object' ? data : null,
    message: `HTTP Error: ${statusCode}`,
    category: classifyRequestError({ statusCode, data })
  }))
}

/**
 * 统一请求方法
 * @param {Object} options 请求配置
 * @param {boolean} options.silent 静默网络失败提示；会话失效仍由全局流程接管
 * @returns {Promise}
 */
const request = (options) => {
  const silent = options.silent || false
  const requestToken = String(uni.getStorageSync('token') || '')
  const requestGeneration = useUserStore(pinia).authGeneration

  // 应用请求拦截器
  const config = requestInterceptor({
    url: BASE_URL + options.url,
    method: options.method || 'GET',
    data: options.data,
    header: {
      'Content-Type': 'application/json',
      ...options.header
    },
    timeout: options.timeout || 30000
  }, requestToken)

  return new Promise((resolve, reject) => {
    recordRequest(config.method, options.url)
    uni.request({
      ...config,
      success: (res) => {
        // 失效主体（SESSION_INVALID）不得用同属失效用户的 refresh token 重试
        if (res.statusCode === 401 && isSessionInvalidResponse(res.data) && !options.skipAuthRefresh) {
          handleSessionInvalid({
            sessionKey: requestToken,
            sessionGeneration: requestGeneration,
            data: res.data,
            silent
          }).catch(reject)
          return
        }
        // 普通 access token 过期：只允许在发起请求的账号世代内刷新或重放。
        if (res.statusCode === 401 && !options.skipAuthRefresh) {
          const currentUserStore = useUserStore(pinia)
          const unauthorizedAction = resolveUnauthorizedAction({
            requestGeneration,
            currentGeneration: currentUserStore.authGeneration,
            requestToken,
            currentToken: String(uni.getStorageSync('token') || '')
          })
          if (unauthorizedAction === 'stale') {
            reject(createRequestError({
              statusCode: 401,
              data: typeof res.data === 'object' ? res.data : null,
              message: '请求所属登录状态已变更',
              category: 'session',
              handled: true,
              silent,
              superseded: true
            }))
            return
          }
          if (unauthorizedAction === 'replay') {
            request({ ...options, skipAuthRefresh: true }).then(resolve, reject)
            return
          }

          refreshAuthToken().then(
            () => {
              // 重放请求的网络、服务或业务失败应原样返回，不得再次解释为 refresh 失败。
              request({ ...options, skipAuthRefresh: true }).then(resolve, reject)
            },
            (refreshError) => {
              handleSessionInvalid({
                sessionKey: requestToken,
                sessionGeneration: requestGeneration,
                data: refreshError?.responseData,
                message: refreshError?.message,
                silent
              }).catch(reject)
            }
          )
          return
        }

        responseInterceptor(res, {
          silent,
          sessionKey: requestToken,
          sessionGeneration: requestGeneration
        })
          .then(resolve)
          .catch(reject)
      },
      fail: (err) => {
        if (!silent) {
          uni.showToast({
            title: '网络请求失败',
            icon: 'none'
          })
        }
        reject(createRequestError({
          message: err.errMsg || '网络请求失败',
          category: 'network'
        }))
      }
    })
  })
}

/**
 * GET 请求
 * @param {String} url 请求地址
 * @param {Object} data 请求参数
 * @param {Object} options 其他配置
 * @returns {Promise}
 */
export const get = (url, data = {}, options = {}) => {
  return request({
    url,
    method: 'GET',
    data,
    ...options
  })
}

/**
 * POST 请求
 * @param {String} url 请求地址
 * @param {Object} data 请求参数
 * @param {Object} options 其他配置
 * @returns {Promise}
 */
export const post = (url, data = {}, options = {}) => {
  return request({
    url,
    method: 'POST',
    data,
    ...options
  })
}

/**
 * PUT 请求
 * @param {String} url 请求地址
 * @param {Object} data 请求参数
 * @param {Object} options 其他配置
 * @returns {Promise}
 */
export const put = (url, data = {}, options = {}) => {
  return request({
    url,
    method: 'PUT',
    data,
    ...options
  })
}

/**
 * DELETE 请求
 * @param {String} url 请求地址
 * @param {Object} data 请求参数
 * @param {Object} options 其他配置
 * @returns {Promise}
 */
export const del = (url, data = {}, options = {}) => {
  return request({
    url,
    method: 'DELETE',
    data,
    ...options
  })
}

// 导出 request 函数作为默认导出
export default request
