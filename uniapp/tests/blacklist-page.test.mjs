import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const pageSource = readFileSync(
  new URL('../subPages/social/blacklist.vue', import.meta.url),
  'utf8'
)

const apiSource = readFileSync(
  new URL('../api/friend.js', import.meta.url),
  'utf8'
)

const pagesSource = readFileSync(
  new URL('../pages.json', import.meta.url),
  'utf8'
)

test('blacklist page loads a paginated list and unblacklists a row', () => {
  assert.match(pageSource, /getFriendBlacklist/)
  assert.match(pageSource, /unblacklistFriend/)
  assert.match(pageSource, /const pageSize = 20/)
  assert.match(pageSource, /onReachBottom/)
  assert.match(pageSource, /buildUnblacklistConfirmCopy/)
  assert.match(pageSource, /removeBlacklistItem/)
  assert.match(pageSource, /已移出黑名单/)
})

test('blacklist page reuses the async page state contract and isolates identity', () => {
  assert.match(pageSource, /createAsyncPageState/)
  assert.match(pageSource, /beginAsyncPageLoad/)
  assert.match(pageSource, /rejectAsyncPageLoad/)
  assert.match(pageSource, /currentIdentityKey/)
  assert.match(pageSource, /loading-state/)
  assert.match(pageSource, /黑名单是空的/)
  assert.match(pageSource, /refresh-error-banner/)
  assert.match(pageSource, /dark-mode/)
})

test('friend api exposes the blacklist list and unblacklist facades', () => {
  assert.match(apiSource, /export const getFriendBlacklist = \(params = \{\}\) =>/)
  assert.match(apiSource, /\/api\/friend\/blacklist\/list/)
  assert.match(apiSource, /export const unblacklistFriend = \(data\) =>/)
  assert.match(apiSource, /\/api\/friend\/unblacklist/)
})

test('blacklist page is registered in the social subpackage', () => {
  assert.match(pagesSource, /"root": "subPages\/social"/)
  assert.match(pagesSource, /"path": "blacklist"[\s\S]*"navigationBarTitleText": "黑名单"/)
})
