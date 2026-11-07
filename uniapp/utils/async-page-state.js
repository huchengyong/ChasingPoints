import { shouldIgnoreRequestError } from './request-errors.js'

/**
 * 数据页面的最小异步状态机。
 * 页面在发起请求后保留 request 标识，只有当前认证代次的最新请求可以回写状态。
 */

export const ASYNC_PAGE_STATUS = Object.freeze({
  IDLE: 'idle',
  LOADING: 'loading',
  READY: 'ready',
  EMPTY: 'empty',
  ERROR: 'error',
  REFRESHING: 'refreshing'
})

const isEmptyByDefault = (data) => data == null || (Array.isArray(data) && data.length === 0)

const normalizeError = (error) => ({
  category: typeof error?.category === 'string' ? error.category : 'business',
  message: error?.message || '加载失败，请稍后重试',
  statusCode: Number.isInteger(error?.statusCode) ? error.statusCode : 0
})

export const shouldIgnoreAsyncPageError = shouldIgnoreRequestError

export const resolveAsyncPageErrorFeedback = (error, { resource = '内容' } = {}) => {
  const category = error?.category || 'business'
  const feedback = {
    retryable: true,
    actionText: '重试',
    title: `${resource}加载失败`,
    description: error?.message || '请稍后重试'
  }

  if (category === 'network') {
    return { ...feedback, title: '网络连接失败', description: '请检查网络后重试' }
  }
  if (category === 'permission') {
    return { retryable: false, actionText: '返回', title: '暂无访问权限', description: '你暂时无权查看这部分内容' }
  }
  if (category === 'not-found') {
    return { retryable: false, actionText: '返回', title: '资源不存在或已下线', description: '请返回上一页选择其他内容' }
  }
  if (category === 'rate-limit') {
    return { ...feedback, title: '请求过于频繁', description: '请稍后再试' }
  }
  if (category === 'server') {
    return { ...feedback, title: '服务暂不可用', description: '请稍后重试' }
  }
  if (category === 'session') {
    return { retryable: false, actionText: '返回', title: '登录状态已失效', description: '正在返回登录入口' }
  }
  return feedback
}

export const createAsyncPageState = ({
  authGeneration = 0,
  data = null,
  isEmpty = isEmptyByDefault
} = {}) => ({
  status: ASYNC_PAGE_STATUS.IDLE,
  authGeneration,
  requestId: 0,
  data,
  hasData: !isEmpty(data),
  error: null,
  refreshError: null
})

export const beginAsyncPageLoad = (state, {
  authGeneration = state.authGeneration,
  emptyData = null
} = {}) => {
  const identityChanged = state.authGeneration !== authGeneration
  const hasData = !identityChanged && state.hasData

  return {
    ...state,
    authGeneration,
    requestId: state.requestId + 1,
    status: hasData ? ASYNC_PAGE_STATUS.REFRESHING : ASYNC_PAGE_STATUS.LOADING,
    data: identityChanged ? emptyData : state.data,
    hasData,
    error: null,
    refreshError: null
  }
}

export const getAsyncPageRequest = (state) => ({
  authGeneration: state.authGeneration,
  requestId: state.requestId
})

export const isAsyncPageRequestCurrent = (state, request = {}) => (
  state.authGeneration === request.authGeneration && state.requestId === request.requestId
)

export const resolveAsyncPageLoad = (state, request, {
  data = null,
  isEmpty = isEmptyByDefault
} = {}) => {
  if (!isAsyncPageRequestCurrent(state, request)) return state

  const empty = isEmpty(data)
  return {
    ...state,
    status: empty ? ASYNC_PAGE_STATUS.EMPTY : ASYNC_PAGE_STATUS.READY,
    data,
    hasData: !empty,
    error: null,
    refreshError: null
  }
}

export const rejectAsyncPageLoad = (state, request, error) => {
  if (!isAsyncPageRequestCurrent(state, request) || shouldIgnoreAsyncPageError(error)) {
    return state
  }

  const pageError = normalizeError(error)
  if (state.hasData) {
    return {
      ...state,
      status: ASYNC_PAGE_STATUS.READY,
      error: null,
      refreshError: pageError
    }
  }

  return {
    ...state,
    status: ASYNC_PAGE_STATUS.ERROR,
    error: pageError,
    refreshError: null
  }
}
