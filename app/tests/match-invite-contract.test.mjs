import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')
const unapprovedQrDomain = ['api', 'qrserver', 'com'].join('.')
const unapprovedQrDomainPattern = new RegExp(unapprovedQrDomain.replace(/\./g, '\\.'))

const apiSource = read('../api/match.js')
const startPayloadSource = read('../utils/start-match.js')
const userPageSource = read('../pages/user/index.vue')
const matchPageSource = read('../pages/match/index.vue')
const playingPageSource = read('../subPages/match/playing.vue')
const scanSources = [
  read('../pages/index/index.vue'),
  matchPageSource,
  read('../pages/ranking/index.vue'),
  userPageSource
]
const qrSources = [
  userPageSource,
  matchPageSource,
  playingPageSource
]

test('invite API and start payload use a server-verifiable token instead of mutable profile fields', () => {
  assert.match(apiSource, /export const previewMatchInvite/)
  assert.match(apiSource, /post\('\/api\/match\/invite\/preview'/)
  assert.match(startPayloadSource, /if \(normalizedInviteToken\) \{/)
  assert.match(startPayloadSource, /payload\.invite_token = normalizedInviteToken/)
  assert.match(startPayloadSource, /if \(!inviteToken && \(!Number\.isFinite\(opponentId\) \|\| opponentId <= 0\)\)/)
})

test('all PK scan entries use the same preview-first helper and pass its invite token to start payload', () => {
  for (const source of scanSources) {
    assert.match(source, /scanAndResolveMatchCode/)
    assert.match(source, /previewMatchInvite/)
    assert.doesNotMatch(source, /JSON.parse(scanResult)/)
  }
  for (const source of scanSources) {
    assert.match(source, /inviteToken: scanAction.inviteToken/)
  }
})

test('all match and referee QR displays render locally without a third-party QR image service', () => {
  for (const source of qrSources) {
    assert.match(source, /renderLocalQRCode/)
    assert.doesNotMatch(source, unapprovedQrDomainPattern)
  }
})

test('local QR displays expose loading, failure, and refresh states', () => {
  assert.match(userPageSource, /qrcodeLoading/)
  assert.match(userPageSource, /qrcodeError/)
  assert.match(userPageSource, /@click="generateQRCode"/)

  assert.match(matchPageSource, /matchQrLoading/)
  assert.match(matchPageSource, /matchQrFailed/)
  assert.match(matchPageSource, /@click="loadMyMatchQRCode"/)

  assert.match(playingPageSource, /refereeQrcodeLoading/)
  assert.match(playingPageSource, /refereeQrcodeError/)
  assert.match(playingPageSource, /refereeQrcodeContent\.value = ''/)
  assert.match(playingPageSource, /@click="openRefereeQrModal"/)
})

test('accepted challenge context is cleared after cancellation, failure, or identity change', () => {
  assert.match(matchPageSource, /const clearPendingStartContext/)
  assert.match(matchPageSource, /scanAction\.type === 'cancelled'/)
  assert.match(matchPageSource, /pendingStartAuthGeneration\.value !== userStore\.authGeneration/)
  assert.match(matchPageSource, /uni\.removeStorageSync\('pending_match_challenge'\)/)
  assert.match(matchPageSource, /uni\.removeStorageSync\('pending_match_rematch'\)/)
})
