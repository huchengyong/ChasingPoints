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

test('request layer shares refresh only within the same refresh-token generation', () => {
  assert.match(requestSource, /const refreshAuthToken = \(\) =>/)
  assert.match(requestSource, /const authRefreshPromises = new Map\(\)/)
  assert.match(requestSource, /authRefreshPromises\.get\(refreshToken\)/)
  assert.match(requestSource, /isRefreshSessionCurrent/)
  assert.match(requestSource, /resolveUnauthorizedAction/)
  assert.match(requestSource, /requestGeneration/)
  assert.match(requestSource, /\/api\/auth\/refresh-token/)
  assert.match(requestSource, /skipAuthRefresh/)
})

test('request replay failures are returned directly instead of being treated as refresh failures', () => {
  assert.match(requestSource, /request\(\{\s*\.\.\.options,\s*skipAuthRefresh:\s*true\s*\}\)\.then\(resolve, reject\)/s)
  assert.doesNotMatch(
    requestSource,
    /request\(\{\s*\.\.\.options,\s*skipAuthRefresh:\s*true\s*\}\)[\s\S]*?\.then\(resolve\)[\s\S]*?\.catch\([\s\S]*?handleSessionInvalid/
  )
})

test('request layer skips refresh for SESSION_INVALID and scopes cleanup to the request token', () => {
  assert.match(requestSource, /isSessionInvalidResponse\(res\.data\)/)
  assert.match(requestSource, /createSessionInvalidHandler/)
  assert.match(requestSource, /sessionKey:\s*requestToken/)
  assert.match(requestSource, /sessionGeneration:\s*requestGeneration/)
  assert.match(requestSource, /getCurrentToken:/)
  assert.match(requestSource, /getCurrentGeneration:/)
})

test('user store can update only token fields after refresh without losing user profile', () => {
  assert.match(userStoreSource, /refreshAuth\(data\)/)
  assert.match(userStoreSource, /this\.token = data\.token \|\| data\.access_token/)
  assert.match(userStoreSource, /uni\.setStorageSync\('token', this\.token\)/)
})

test('login replacement and logout both advance auth generation and clear user-scoped runtime caches', () => {
  assert.match(userStoreSource, /authGeneration:\s*0/)
  assert.equal((userStoreSource.match(/this\.authGeneration \+= 1/g) || []).length, 2)
  assert.match(userStoreSource, /useNotificationStore/)
  assert.match(userStoreSource, /useFriendRequestStore/)
  assert.equal((userStoreSource.match(/clearCurrentUserRuntimeState\(\)/g) || []).length, 2)
})
