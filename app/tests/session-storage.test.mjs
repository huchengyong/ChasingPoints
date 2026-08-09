import test from 'node:test'
import assert from 'node:assert/strict'

import {
  clearStoredSession,
  clearUserScopedRuntimeState,
  readStoredSession,
  SESSION_STORAGE_KEYS
} from '../utils/session-storage.js'

const createMockStorage = (initial = {}) => {
  const data = { ...initial }
  return {
    data,
    getStorageSync: (key) => data[key],
    setStorageSync: (key, value) => { data[key] = value },
    removeStorageSync: (key) => { delete data[key] }
  }
}

test('readStoredSession restores token and refresh token only when present', () => {
  const storage = createMockStorage({
    token: 'access-token',
    refreshToken: 'refresh-token'
  })
  assert.deepEqual(readStoredSession(storage), {
    token: 'access-token',
    refreshToken: 'refresh-token',
    hasSession: true
  })
})

test('readStoredSession treats missing or non-string values as absent', () => {
  assert.deepEqual(readStoredSession(createMockStorage({})), {
    token: '',
    refreshToken: '',
    hasSession: false
  })
  assert.equal(readStoredSession(createMockStorage({ token: 123 })).hasSession, false)
  assert.equal(readStoredSession(createMockStorage({ token: '' })).hasSession, false)
})

test('clearStoredSession removes every persisted identity key', () => {
  const storage = createMockStorage({
    token: 'access-token',
    refreshToken: 'refresh-token',
    loginMethod: 'wechat-mini',
    'user-store': JSON.stringify({ token: 'access-token', isLoggedIn: true })
  })

  clearStoredSession(storage)

  for (const key of Object.values(SESSION_STORAGE_KEYS)) {
    assert.equal(storage.data[key] === undefined, true, `expected ${key} cleared`)
  }
})

test('user-scoped runtime cleanup clears rank, unread badges and user WebSocket together', () => {
  const calls = []
  clearUserScopedRuntimeState({
    rankStore: { clear: () => calls.push('rank') },
    notificationStore: { clearUnread: () => calls.push('notification') },
    friendRequestStore: { clearPendingCount: () => calls.push('friend-request') },
    userSocket: { disconnect: () => calls.push('websocket') }
  })

  assert.deepEqual(calls, ['rank', 'notification', 'friend-request', 'websocket'])
})

test('logout then init cannot restore a stale identity', () => {
  const storage = createMockStorage({
    token: 'old-access',
    refreshToken: 'old-refresh',
    'user-store': JSON.stringify({ token: 'old-access', isLoggedIn: true, userInfo: { id: 2 } })
  })

  // 模拟 logout：运行状态清空并清除全部持久化键
  clearStoredSession(storage)

  // 模拟重新启动后的 init：只允许从仍然存在的存储恢复
  const restored = readStoredSession(storage)
  assert.equal(restored.hasSession, false)

  // user-store 中残留的旧身份也必须不可恢复
  assert.equal(storage.data[SESSION_STORAGE_KEYS.userStore], undefined)
})
