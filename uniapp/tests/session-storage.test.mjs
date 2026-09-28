import test from 'node:test'
import assert from 'node:assert/strict'

import {
  clearStoredSession,
  MATCH_START_CONTEXT_STORAGE_KEYS,
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
    'user-store': JSON.stringify({ token: 'access-token', isLoggedIn: true }),
    pending_match_challenge: JSON.stringify({ challenge_id: 88 }),
    pending_match_rematch: JSON.stringify({ opponent_id: 2 })
  })

  clearStoredSession(storage)

  for (const key of Object.values(SESSION_STORAGE_KEYS)) {
    assert.equal(storage.data[key] === undefined, true, `expected ${key} cleared`)
  }
  for (const key of MATCH_START_CONTEXT_STORAGE_KEYS) {
    assert.equal(storage.data[key] === undefined, true, `expected ${key} cleared`)
  }
})

test('user-scoped runtime cleanup clears rank, unread badges, invalidation state and user WebSocket together', () => {
  const calls = []
  clearUserScopedRuntimeState({
    rankStore: { clear: () => calls.push('rank') },
    notificationStore: { clearUnread: () => calls.push('notification') },
    friendRequestStore: { clearPendingCount: () => calls.push('friend-request') },
    userDataInvalidationStore: { clear: () => calls.push('invalidation') },
    publicReadStore: { clearLeaderboard: () => calls.push('leaderboard') },
    userSocket: { disconnect: () => calls.push('user-websocket') },
    matchSocket: { disconnect: () => calls.push('match-websocket') }
  })

  assert.deepEqual(calls, ['rank', 'notification', 'friend-request', 'invalidation', 'leaderboard', 'user-websocket', 'match-websocket'])
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
