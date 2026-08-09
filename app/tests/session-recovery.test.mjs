import test from 'node:test'
import assert from 'node:assert/strict'

import {
  canApplySessionRecoveryResult,
  createSessionRecovery,
  isRetryableSessionCheckError
} from '../utils/session-recovery.js'

test('session recovery marks network and server errors as retryable', () => {
  assert.equal(isRetryableSessionCheckError({ category: 'network' }), true)
  assert.equal(isRetryableSessionCheckError({ category: 'server' }), true)
  assert.equal(isRetryableSessionCheckError({ category: 'session' }), false)
  assert.equal(isRetryableSessionCheckError({ category: 'business' }), false)
  assert.equal(isRetryableSessionCheckError({}), false)
})

test('valid session returns server profile without applying it inside the shared helper', async () => {
  const recovery = createSessionRecovery({
    getUserInfo: async () => ({ success: true, user_info: { id: 1, nickname: '球手' } })
  })

  const result = await recovery.validate('generation-1')
  assert.deepEqual(result, {
    valid: true,
    retryable: false,
    userInfo: { id: 1, nickname: '球手' }
  })
})

test('SESSION_INVALID triggers the global handling callback and reports invalid', async () => {
  const invalidHandled = []
  const recovery = createSessionRecovery({
    getUserInfo: async () => {
      const error = new Error('登录状态已失效，请重新登录')
      error.category = 'session'
      error._isHandled = true
      throw error
    },
    onSessionInvalid: (error) => invalidHandled.push(error.category)
  })

  const result = await recovery.validate('generation-1')
  assert.deepEqual(result, { valid: false, retryable: false })
  assert.deepEqual(invalidHandled, ['session'])
})

test('network failure keeps credentials and stays retryable', async () => {
  const invalidHandled = []
  const recovery = createSessionRecovery({
    getUserInfo: async () => {
      const error = new Error('网络请求失败')
      error.category = 'network'
      throw error
    },
    onSessionInvalid: () => invalidHandled.push(true)
  })

  const result = await recovery.validate('generation-1')
  assert.deepEqual(result, { valid: false, retryable: true })
  assert.equal(invalidHandled.length, 0)
})

test('server failure keeps credentials and stays retryable', async () => {
  const recovery = createSessionRecovery({
    getUserInfo: async () => {
      const error = new Error('内部服务错误')
      error.category = 'server'
      throw error
    }
  })

  const result = await recovery.validate('generation-1')
  assert.deepEqual(result, { valid: false, retryable: true })
})

test('concurrent validations share one request only within the same auth generation', async () => {
  let requests = 0
  let resolveFirst
  const recovery = createSessionRecovery({
    getUserInfo: () => {
      requests += 1
      return new Promise(resolve => {
        resolveFirst = () => resolve({ success: true, user_info: { id: 1 } })
      })
    }
  })

  const first = recovery.validate('generation-1')
  const second = recovery.validate('generation-1')
  assert.equal(requests, 1)

  resolveFirst()
  const results = await Promise.all([first, second])
  assert.deepEqual(results, [
    { valid: true, retryable: false, userInfo: { id: 1 } },
    { valid: true, retryable: false, userInfo: { id: 1 } }
  ])
})

test('different auth generations never share a stale recovery promise', async () => {
  const requests = []
  const resolvers = new Map()
  const recovery = createSessionRecovery({
    getUserInfo: (generation) => {
      requests.push(generation)
      return new Promise(resolve => resolvers.set(generation, resolve))
    }
  })

  const userA = recovery.validate('generation-a')
  const userB = recovery.validate('generation-b')
  assert.deepEqual(requests, ['generation-a', 'generation-b'])

  resolvers.get('generation-a')({ success: true, user_info: { id: 'A' } })
  resolvers.get('generation-b')({ success: true, user_info: { id: 'B' } })

  assert.deepEqual(await userA, {
    valid: true,
    retryable: false,
    userInfo: { id: 'A' }
  })
  assert.deepEqual(await userB, {
    valid: true,
    retryable: false,
    userInfo: { id: 'B' }
  })
})

test('business failures are not mislabeled as SESSION_INVALID', async () => {
  const invalidHandled = []
  const recovery = createSessionRecovery({
    getUserInfo: async () => ({ success: false, message: '会话校验失败' }),
    onSessionInvalid: () => invalidHandled.push(true)
  })

  const result = await recovery.validate('generation-1')
  assert.deepEqual(result, { valid: false, retryable: false })
  assert.equal(invalidHandled.length, 0)
})

test('recovery result is applied only to the same foreground auth generation', () => {
  assert.equal(canApplySessionRecoveryResult({
    expectedGeneration: 3,
    currentGeneration: 3,
    isLoggedIn: true,
    isForeground: true
  }), true)
  assert.equal(canApplySessionRecoveryResult({
    expectedGeneration: 3,
    currentGeneration: 4,
    isLoggedIn: true,
    isForeground: true
  }), false)
  assert.equal(canApplySessionRecoveryResult({
    expectedGeneration: 3,
    currentGeneration: 3,
    isLoggedIn: false,
    isForeground: true
  }), false)
  assert.equal(canApplySessionRecoveryResult({
    expectedGeneration: 3,
    currentGeneration: 3,
    isLoggedIn: true,
    isForeground: false
  }), false)
  assert.equal(canApplySessionRecoveryResult({
    expectedGeneration: 3,
    currentGeneration: 3,
    expectedLifecycleGeneration: 7,
    currentLifecycleGeneration: 8,
    isLoggedIn: true,
    isForeground: true
  }), false)
})
