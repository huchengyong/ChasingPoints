import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appRoot = new URL('..', import.meta.url)
const readPage = (path) => readFileSync(new URL(path, appRoot), 'utf8')

test('retained private read pages bind refreshes to identity and exact invalidation scopes', () => {
  const stats = readPage('subPages/user/statsDetail.vue')
  const h2h = readPage('subPages/user/h2hRecord.vue')
  const honor = readPage('subPages/achievement/index.vue')
  const history = readPage('subPages/user/matchHistory.vue')
  const opponents = readPage('subPages/user/opponentRecord.vue')
  const notifications = readPage('subPages/notification/index.vue')
  const requests = readPage('subPages/social/friendRequests.vue')

  assert.match(stats, /loadedIdentityKey\.value !== currentIdentityKey\(\)/)
  assert.match(stats, /userDataInvalidationStore\.versionOf\('stats'\)/)
  assert.match(h2h, /userDataInvalidationStore\.versionOf\('h2h'\)/)
  assert.match(h2h, /currentIdentityKey\(\) !== requestIdentityKey/)
  assert.match(honor, /userDataInvalidationStore\.versionOf\('honor'\)/)
  assert.match(honor, /currentIdentityKey\(\) !== requestIdentityKey/)
  assert.match(history, /userDataInvalidationStore\.versionOf\('history'\)/)
  assert.match(history, /resetHistoryForIdentity/)
  assert.match(opponents, /userDataInvalidationStore\.versionOf\('opponents'\)/)
  assert.match(opponents, /currentIdentityKey\(\) !== requestIdentityKey/)
  assert.match(notifications, /currentIdentityKey\(\) !== requestIdentityKey/)
  assert.match(requests, /currentIdentityKey\(\) !== requestIdentityKey/)
})
