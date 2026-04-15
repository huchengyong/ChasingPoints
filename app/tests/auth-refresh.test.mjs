import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const apiSource = readFileSync(
  new URL('../api/auth.js', import.meta.url),
  'utf8'
)

const requestSource = readFileSync(
  new URL('../utils/request.js', import.meta.url),
  'utf8'
)

const userStoreSource = readFileSync(
  new URL('../store/user.js', import.meta.url),
  'utf8'
)

test('auth api exposes refresh-token endpoint for silent session renewal', () => {
  assert.match(apiSource, /export const refreshToken = \(refreshToken\) =>/)
  assert.match(apiSource, /\/api\/auth\/refresh-token/)
  assert.match(apiSource, /refresh_token: refreshToken/)
})

test('request layer refreshes token once before clearing login on 401', () => {
  assert.match(requestSource, /const refreshAuthToken = \(\) =>/)
  assert.match(requestSource, /authRefreshPromise/)
  assert.match(requestSource, /\/api\/auth\/refresh-token/)
  assert.match(requestSource, /skipAuthRefresh/)
  assert.match(requestSource, /request\(\{\s*\.\.\.options,\s*skipAuthRefresh:\s*true\s*\}\)/s)
  assert.match(requestSource, /handleUnauthorized\(silent\)/)
})

test('user store can update only token fields after refresh without losing user profile', () => {
  assert.match(userStoreSource, /refreshAuth\(data\)/)
  assert.match(userStoreSource, /this\.token = data\.token \|\| data\.access_token/)
  assert.match(userStoreSource, /uni\.setStorageSync\('token', this\.token\)/)
})
