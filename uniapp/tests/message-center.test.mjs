import test from 'node:test'
import assert from 'node:assert/strict'

import {
  NOTIFICATION_REFRESH_DEBOUNCE_MS,
  buildMessageEntries,
  formatEntryCount,
  ownedCount,
  resolveNotificationReload
} from '../utils/message-center.js'

const base = {
  identityKey: '7:1',
  loadedIdentityKey: '7:1',
  requestedIdentityKey: '7:1',
  loadedScopeVersion: 3,
  scopeVersion: 3
}

test('an unchanged identity and scope reuses the loaded list', () => {
  assert.equal(resolveNotificationReload(base), 'reuse')
})

test('a bumped notification scope reloads the first page even though the identity is unchanged', () => {
  assert.equal(resolveNotificationReload({ ...base, scopeVersion: 4 }), 'load')
})

test('a first visit loads, and a request already in flight for this identity is awaited', () => {
  assert.equal(
    resolveNotificationReload({ ...base, loadedIdentityKey: '', requestedIdentityKey: '' }),
    'load'
  )
  assert.equal(
    resolveNotificationReload({ ...base, loadedIdentityKey: '', requestedIdentityKey: '7:1' }),
    'wait'
  )
})

test('a different account or auth generation always loads again', () => {
  assert.equal(resolveNotificationReload({ ...base, identityKey: '8:1' }), 'load')
  assert.equal(resolveNotificationReload({ ...base, identityKey: '7:2', requestedIdentityKey: '7:1' }), 'load')
  assert.equal(
    resolveNotificationReload({ ...base, identityKey: '8:1', loadedIdentityKey: '7:1', requestedIdentityKey: '8:1' }),
    'wait',
    'a request for the new identity is already running'
  )
})

test('count formatting caps at 99+', () => {
  assert.equal(formatEntryCount(0), '')
  assert.equal(formatEntryCount(5), '5')
  assert.equal(formatEntryCount(99), '99')
  assert.equal(formatEntryCount(100), '99+')
  assert.equal(formatEntryCount(-3), '')
  assert.equal(formatEntryCount('x'), '')
})

test('message entries point at the existing social pages and describe pending counts', () => {
  const entries = buildMessageEntries({ friendRequestCount: 2, challengeCount: 0 })

  assert.deepEqual(entries.map((entry) => entry.key), ['friend-requests', 'challenge-invites'])
  assert.deepEqual(entries.map((entry) => entry.url), [
    '/subPages/social/friendRequests',
    '/subPages/social/challenges?tab=received'
  ])
  assert.deepEqual(entries.map((entry) => entry.countText), ['2', ''])
  assert.equal(entries[0].desc, '2 条待处理')
  assert.equal(entries[1].desc, '暂无待回应的邀请')
})

test('message entries tolerate missing counts', () => {
  const entries = buildMessageEntries()
  assert.deepEqual(entries.map((entry) => entry.countText), ['', ''])
  assert.equal(entries[0].desc, '暂无待处理申请')
})

test('a count only counts for the identity its store belongs to', () => {
  const identity = { userId: 7, authGeneration: 1 }
  const owner = { matchesIdentity: (candidate) => candidate.userId === 7 && candidate.authGeneration === 1 }

  assert.equal(ownedCount(owner, identity, 4), 4)
  assert.equal(ownedCount(owner, { userId: 8, authGeneration: 1 }, 4), 0)
  assert.equal(ownedCount(owner, { userId: 7, authGeneration: 2 }, 4), 0)
  assert.equal(ownedCount(owner, identity, -1), 0)
  assert.equal(ownedCount(owner, identity, undefined), 0)
})

test('live refreshes are debounced so a burst of notifications triggers one reload', () => {
  assert.equal(NOTIFICATION_REFRESH_DEBOUNCE_MS, 500)
})
