/**
 * 封装 uni.request 请求工具
 */
import pinia from '@/store/index.js'
import { useUserStore } from '@/store/user.js'
import { shouldResolveBusinessResponse } from './request-response.js'

// 基础配置 api-bm
// const BASE_URL = 'https://api-bm.dianzaozao.com' // 根据实际情况配置
const BASE_URL = 'https://api-tunnel2.kekemate.com'

// 请求拦截器
const requestInterceptor = (config) => {
  // 获取 token
  const token = uni.getStorageSync('token')
  if (token) {
    config.header = {
      ...config.header,
      'Authorization': `Bearer ${token}`
    }
  }
  return config
}

// 响应拦截器
// silent 参数：为 true 时遇到401不弹出提示、不跳转登录页
const responseInterceptor = (response, silent = false) => {
  const { statusCode, data } = response

  // HTTP 状态码处理
  if (statusCode === 401) {
    const userStore = useUserStore(pinia)

    // token 过期，清除登录状态
    uni.removeStorageSync('token')
    uni.removeStorageSync('refreshToken')
    userStore.logout()

    // 非静默模式下才弹出提示和跳转
    if (!silent) {
      uni.showToast({
        title: '请先完成登录',
        icon: 'none'
      })
      // 跳转登录页
      setTimeout(() => {
        uni.navigateTo({
          url: '/pages/login/login'
        })
      }, 1500)
    }
    // 标记为已由全局处理，避免页面级重复弹出提示
    const error = new Error('请先完成登录')
    error._isHandled = true
    error._isSilent = silent
    return Promise.reject(error)
  }

  if (statusCode >= 200 && statusCode < 300) {
    if (data == null) {
      return Promise.reject(new Error('响应数据为空'))
    }

    if (typeof data !== 'object') {
      return Promise.resolve(data)
    }

    // 业务状态码处理
    if (shouldResolveBusinessResponse(data)) {
      return Promise.resolve(data.data || data)
    } else {
      const error = new Error(data.message || data.msg || '请求失败')
      error.responseData = data
      return Promise.reject(error)
    }
  }

  return Promise.reject(new Error(`HTTP Error: ${statusCode}`))
}

/**
 * 统一请求方法
 * @param {Object} options 请求配置
 * @param {boolean} options.silent 静默模式，401时不弹toast不跳转
 * @returns {Promise}
 */
const request = (options) => {
  const silent = options.silent || false

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
  })

  return new Promise((resolve, reject) => {
    uni.request({
      ...config,
      success: (res) => {
        responseInterceptor(res, silent)
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
        reject(new Error(err.errMsg || '网络请求失败'))
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
