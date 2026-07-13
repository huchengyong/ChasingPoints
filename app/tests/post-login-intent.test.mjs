import test from 'node:test'
import assert from 'node:assert/strict'

import {
  POST_LOGIN_ACTIONS,
  clearPostLoginIntent,
  consumePostLoginIntent,
  getPostLoginIntent,
  normalizePostLoginIntent,
  setPostLoginIntent
} from '../utils/post-login-intent.js'

const createUniStorage = () => {
  const storage = new Map()
  return {
    getStorageSync(key) {
      return storage.get(key) || ''
    },
    setStorageSync(key, value) {
      storage.set(key, value)
    },
    removeStorageSync(key) {
      storage.delete(key)
    }
  }
}

test('normalizePostLoginIntent only accepts supported PK actions', () => {
  assert.equal(normalizePostLoginIntent(POST_LOGIN_ACTIONS.START_PK), POST_LOGIN_ACTIONS.START_PK)
  assert.equal(normalizePostLoginIntent(POST_LOGIN_ACTIONS.SHOW_PK_CODE), POST_LOGIN_ACTIONS.SHOW_PK_CODE)
  assert.equal(normalizePostLoginIntent('ranking'), '')
  assert.equal(normalizePostLoginIntent(''), '')
})

test('setPostLoginIntent stores valid actions and get/consume read them once', () => {
  global.uni = createUniStorage()

  setPostLoginIntent(POST_LOGIN_ACTIONS.START_PK)
  assert.equal(getPostLoginIntent(), POST_LOGIN_ACTIONS.START_PK)
  assert.equal(consumePostLoginIntent(), POST_LOGIN_ACTIONS.START_PK)
  assert.equal(getPostLoginIntent(), '')

  delete global.uni
})

test('invalid stored actions are ignored and cleared', () => {
  global.uni = createUniStorage()

  global.uni.setStorageSync('post_login_intent', 'bad-action')
  assert.equal(getPostLoginIntent(), '')
  assert.equal(consumePostLoginIntent(), '')

  setPostLoginIntent('also-bad')
  assert.equal(getPostLoginIntent(), '')

  setPostLoginIntent(POST_LOGIN_ACTIONS.SHOW_PK_CODE)
  clearPostLoginIntent()
  assert.equal(getPostLoginIntent(), '')

  delete global.uni
})
