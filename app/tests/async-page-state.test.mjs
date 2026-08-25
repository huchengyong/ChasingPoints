import test from 'node:test'
import assert from 'node:assert/strict'
import {
  ASYNC_PAGE_STATUS,
  beginAsyncPageLoad,
  createAsyncPageState,
  getAsyncPageRequest,
  rejectAsyncPageLoad,
  resolveAsyncPageErrorFeedback,
  resolveAsyncPageLoad,
  shouldIgnoreAsyncPageError
} from '../utils/async-page-state.js'

test('async page state distinguishes loading, ready, and successful empty data', () => {
  let state = createAsyncPageState({ authGeneration: 3, data: [] })
  assert.equal(state.status, ASYNC_PAGE_STATUS.IDLE)

  state = beginAsyncPageLoad(state)
  const request = getAsyncPageRequest(state)
  assert.equal(state.status, ASYNC_PAGE_STATUS.LOADING)

  state = resolveAsyncPageLoad(state, request, { data: [{ id: 1 }] })
  assert.equal(state.status, ASYNC_PAGE_STATUS.READY)
  assert.equal(state.hasData, true)

  state = beginAsyncPageLoad(state)
  state = resolveAsyncPageLoad(state, getAsyncPageRequest(state), { data: [] })
  assert.equal(state.status, ASYNC_PAGE_STATUS.EMPTY)
  assert.equal(state.hasData, false)
})

test('async page state surfaces first-load error categories and retains data on refresh failure', () => {
  let state = createAsyncPageState({ authGeneration: 4, data: [] })
  state = beginAsyncPageLoad(state)
  state = rejectAsyncPageLoad(state, getAsyncPageRequest(state), {
    category: 'network',
    message: '网络不可用'
  })

  assert.equal(state.status, ASYNC_PAGE_STATUS.ERROR)
  assert.deepEqual(state.error, {
    category: 'network',
    message: '网络不可用',
    statusCode: 0
  })

  state = beginAsyncPageLoad(state)
  const recoveryRequest = getAsyncPageRequest(state)
  state = resolveAsyncPageLoad(state, recoveryRequest, { data: [{ id: 1 }] })

  state = beginAsyncPageLoad(state)
  const refreshRequest = getAsyncPageRequest(state)
  assert.equal(state.status, ASYNC_PAGE_STATUS.REFRESHING)

  state = rejectAsyncPageLoad(state, refreshRequest, {
    category: 'server',
    message: '服务暂不可用',
    statusCode: 503
  })
  assert.equal(state.status, ASYNC_PAGE_STATUS.READY)
  assert.deepEqual(state.data, [{ id: 1 }])
  assert.deepEqual(state.refreshError, {
    category: 'server',
    message: '服务暂不可用',
    statusCode: 503
  })
})

test('async page state drops stale auth-generation responses and globally handled errors', () => {
  let state = createAsyncPageState({ authGeneration: 8, data: [] })
  state = beginAsyncPageLoad(state)
  const oldRequest = getAsyncPageRequest(state)

  state = beginAsyncPageLoad(state, { authGeneration: 9, emptyData: [] })
  const currentState = { ...state }

  assert.equal(
    resolveAsyncPageLoad(state, oldRequest, { data: [{ id: 'old-user' }] }),
    state
  )
  assert.deepEqual(state, currentState)

  assert.equal(shouldIgnoreAsyncPageError({ _isHandled: true }), true)
  assert.equal(shouldIgnoreAsyncPageError({ _isSuperseded: true }), true)
  assert.equal(
    rejectAsyncPageLoad(state, getAsyncPageRequest(state), {
      category: 'session',
      _isHandled: true
    }),
    state
  )
})

test('async page error feedback keeps retry and navigation actions aligned with categories', () => {
  assert.deepEqual(resolveAsyncPageErrorFeedback({ category: 'network' }), {
    retryable: true,
    actionText: '重试',
    title: '网络连接失败',
    description: '请检查网络后重试'
  })
  assert.deepEqual(resolveAsyncPageErrorFeedback({ category: 'not-found' }), {
    retryable: false,
    actionText: '返回',
    title: '资源不存在或已下线',
    description: '请返回上一页选择其他内容'
  })
})
