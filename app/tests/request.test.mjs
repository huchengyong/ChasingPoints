import test from 'node:test'
import assert from 'node:assert/strict'

import {
  classifyRequestError,
  createRequestError,
  createSessionInvalidHandler,
  isRefreshSessionCurrent,
  isSessionInvalidResponse,
  resolveUnauthorizedAction,
  SESSION_INVALID_REASON
} from '../utils/request-errors.js'
import {
  shouldResolveBusinessResponse,
  unwrapBusinessResponse
} from '../utils/request-response.js'

test('shouldResolveBusinessResponse returns true for normal success payloads', () => {
  assert.equal(shouldResolveBusinessResponse({ success: true }), true)
  assert.equal(shouldResolveBusinessResponse({ code: 0 }), true)
})

test('shouldResolveBusinessResponse returns true for optimistic-lock conflict payloads with accepted flag', () => {
  assert.equal(shouldResolveBusinessResponse({
    success: false,
    accepted: false,
    message: 'match revision conflict',
    snapshot: {
      server_revision: 12
    }
  }), true)
})

test('shouldResolveBusinessResponse returns false for regular business failures without recovery payload', () => {
  assert.equal(shouldResolveBusinessResponse({
    success: false,
    message: '请求失败'
  }), false)
})

test('unwrapBusinessResponse preserves falsy data payloads', () => {
  assert.equal(unwrapBusinessResponse({ code: 0, data: 0 }), 0)
  assert.equal(unwrapBusinessResponse({ code: 0, data: false }), false)
  assert.equal(unwrapBusinessResponse({ code: 0, data: '' }), '')
})

test('unwrapBusinessResponse falls back to the full envelope only when data key is absent', () => {
  const response = { code: 0, success: true, message: 'ok' }

  assert.equal(unwrapBusinessResponse(response), response)
})

test('session invalid responses are identified by stable reason', () => {
  assert.equal(SESSION_INVALID_REASON, 'SESSION_INVALID')
  assert.equal(isSessionInvalidResponse({ success: false, reason: 'SESSION_INVALID' }), true)
  assert.equal(isSessionInvalidResponse({ success: false, message: 'token expired' }), false)
  assert.equal(isSessionInvalidResponse(null), false)
  assert.equal(isSessionInvalidResponse('SESSION_INVALID'), false)
})

test('refresh responses may only write back to the session that started them', () => {
  assert.equal(isRefreshSessionCurrent({
    startedRefreshToken: 'refresh-a',
    currentRefreshToken: 'refresh-a'
  }), true)
  assert.equal(isRefreshSessionCurrent({
    startedRefreshToken: 'refresh-a',
    currentRefreshToken: 'refresh-b'
  }), false)
  assert.equal(isRefreshSessionCurrent({
    startedRefreshToken: 'refresh-a',
    currentRefreshToken: ''
  }), false)
  assert.equal(isRefreshSessionCurrent({}), false)
})

test('ordinary 401 handling never refreshes or replays across auth generations', () => {
  assert.equal(resolveUnauthorizedAction({
    requestGeneration: 1,
    currentGeneration: 2,
    requestToken: 'access-a',
    currentToken: 'access-b'
  }), 'stale')
  assert.equal(resolveUnauthorizedAction({
    requestGeneration: 2,
    currentGeneration: 2,
    requestToken: 'access-old',
    currentToken: 'access-refreshed'
  }), 'replay')
  assert.equal(resolveUnauthorizedAction({
    requestGeneration: 2,
    currentGeneration: 2,
    requestToken: 'access-current',
    currentToken: 'access-current'
  }), 'refresh')
  assert.equal(resolveUnauthorizedAction({
    requestGeneration: 2,
    currentGeneration: 2,
    requestToken: 'access-current',
    currentToken: ''
  }), 'refresh')
})

test('request errors are classified by status and payload', () => {
  assert.equal(classifyRequestError({ network: true }), 'network')
  assert.equal(classifyRequestError({ statusCode: 401, data: { reason: 'SESSION_INVALID' } }), 'session')
  assert.equal(classifyRequestError({ statusCode: 401, data: { message: 'expired' } }), 'business')
  assert.equal(classifyRequestError({ statusCode: 403 }), 'forbidden')
  assert.equal(classifyRequestError({ statusCode: 500 }), 'server')
  assert.equal(classifyRequestError({ statusCode: 400 }), 'business')
  assert.equal(classifyRequestError({ statusCode: 0 }), 'business')
})

test('created request errors preserve status, payload, category and actual silent metadata', () => {
  const visibleError = createRequestError({
    statusCode: 401,
    data: { success: false, reason: 'SESSION_INVALID', trace_id: 'trace-1' },
    message: '登录状态已失效，请重新登录',
    category: 'session',
    handled: true,
    silent: false
  })
  assert.equal(visibleError.message, '登录状态已失效，请重新登录')
  assert.equal(visibleError.statusCode, 401)
  assert.deepEqual(visibleError.responseData, {
    success: false,
    reason: 'SESSION_INVALID',
    trace_id: 'trace-1'
  })
  assert.equal(visibleError.category, 'session')
  assert.equal(visibleError._isHandled, true)
  assert.equal(visibleError._isSilent, false)

  const silentError = createRequestError({ handled: true, silent: true, superseded: true })
  assert.equal(silentError._isSilent, true)
  assert.equal(silentError._isSuperseded, true)
})

const createSessionHandlerHarness = ({ initialToken = 'old-token', initialGeneration = 1, delay = 1 } = {}) => {
  let currentToken = initialToken
  let currentGeneration = initialGeneration
  const calls = []
  const handle = createSessionInvalidHandler({
    getCurrentToken: () => currentToken,
    getCurrentGeneration: () => currentGeneration,
    isLoggedIn: () => Boolean(currentToken),
    logout: () => {
      calls.push(['logout', currentToken])
      currentToken = ''
      currentGeneration += 1
    },
    showToast: (title) => calls.push(['toast', title]),
    reLaunch: (url) => calls.push(['reLaunch', url]),
    delay
  })
  return {
    calls,
    handle,
    getCurrentToken: () => currentToken,
    getCurrentGeneration: () => currentGeneration,
    setCurrentToken: (token) => {
      currentToken = token
      currentGeneration += 1
    }
  }
}

test('session invalid handler clears session once, toasts once and navigates once under concurrency', async () => {
  const harness = createSessionHandlerHarness()
  const payload = { success: false, reason: 'SESSION_INVALID', trace_id: 'trace-concurrent' }

  const first = harness.handle({ sessionKey: 'old-token', data: payload }).catch(error => error)
  const second = harness.handle({ sessionKey: 'old-token', data: payload }).catch(error => error)
  const [firstError, secondError] = await Promise.all([first, second])

  assert.equal(harness.calls.filter(([type]) => type === 'logout').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'toast').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'reLaunch').length, 1)
  assert.deepEqual(firstError.responseData, payload)
  assert.deepEqual(secondError.responseData, payload)
})

test('a silent caller cannot suppress global invalid-session navigation for a concurrent visible caller', async () => {
  const harness = createSessionHandlerHarness()

  const silent = harness.handle({ sessionKey: 'old-token', silent: true }).catch(error => error)
  const visible = harness.handle({ sessionKey: 'old-token', silent: false }).catch(error => error)
  const [silentError, visibleError] = await Promise.all([silent, visible])

  assert.equal(harness.calls.filter(([type]) => type === 'logout').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'toast').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'reLaunch').length, 1)
  assert.equal(silentError._isSilent, true)
  assert.equal(visibleError._isSilent, false)
})

test('new login during the invalid-session delay cancels the old navigation', async () => {
  const harness = createSessionHandlerHarness({ delay: 10 })

  const invalidation = harness.handle({ sessionKey: 'old-token' }).catch(error => error)
  harness.setCurrentToken('new-token')
  const error = await invalidation

  assert.equal(harness.getCurrentToken(), 'new-token')
  assert.equal(harness.calls.filter(([type]) => type === 'logout').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'toast').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'reLaunch').length, 0)
  assert.equal(error._isHandled, true)
  assert.equal(error._isSuperseded, true)
})

test('a delayed response from the same token string cannot clear a newer auth generation', async () => {
  const harness = createSessionHandlerHarness()
  harness.setCurrentToken('old-token')

  const lateError = await harness.handle({
    sessionKey: 'old-token',
    sessionGeneration: 1,
    data: { reason: 'SESSION_INVALID', request: 'same-token-new-generation' }
  }).catch(error => error)

  assert.equal(harness.getCurrentToken(), 'old-token')
  assert.equal(harness.getCurrentGeneration(), 2)
  assert.equal(harness.calls.filter(([type]) => type === 'logout').length, 0)
  assert.equal(harness.calls.filter(([type]) => type === 'reLaunch').length, 0)
  assert.equal(lateError._isHandled, true)
  assert.equal(lateError._isSuperseded, true)
})

test('a delayed SESSION_INVALID from an old token cannot clear a newly logged-in session', async () => {
  const harness = createSessionHandlerHarness()

  await harness.handle({ sessionKey: 'old-token' }).catch(() => {})
  harness.setCurrentToken('new-token')
  const lateError = await harness.handle({
    sessionKey: 'old-token',
    data: { reason: 'SESSION_INVALID', request: 'late' }
  }).catch(error => error)

  assert.equal(harness.getCurrentToken(), 'new-token')
  assert.equal(harness.calls.filter(([type]) => type === 'logout').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'toast').length, 1)
  assert.equal(harness.calls.filter(([type]) => type === 'reLaunch').length, 1)
  assert.deepEqual(lateError.responseData, { reason: 'SESSION_INVALID', request: 'late' })
  assert.equal(lateError._isHandled, true)
  assert.equal(lateError._isSuperseded, true)
})
