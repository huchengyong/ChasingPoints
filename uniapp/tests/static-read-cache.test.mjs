import test from 'node:test'
import assert from 'node:assert/strict'

import {
  readStaticReadCache,
  removeStaticReadCache,
  writeStaticReadCache
} from '../utils/static-read-cache.js'

test('static read cache persists values and isolates schema versions', () => {
  const values = new Map()
  globalThis.uni = {
    getStorageSync: (key) => values.get(key),
    setStorageSync: (key, value) => values.set(key, value),
    removeStorageSync: (key) => values.delete(key)
  }

  writeStaticReadCache('rank-configs', { success: true }, 1, 123)
  assert.deepEqual(readStaticReadCache('rank-configs', 1), {
    value: { success: true },
    loadedAt: 123
  })
  assert.equal(readStaticReadCache('rank-configs', 2), null)

  removeStaticReadCache('rank-configs')
  assert.equal(readStaticReadCache('rank-configs', 1), null)
  delete globalThis.uni
})
