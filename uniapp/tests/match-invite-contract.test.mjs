import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')
const unapprovedQrDomain = ['api', 'qrserver', 'com'].join('.')
const unapprovedQrDomainPattern = new RegExp(unapprovedQrDomain.replace(/\./g, '\\.'))

const apiSource = read('../api/match.js')
const startPayloadSource = read('../utils/start-match.js')
const matchPageSource = read('../pages/match/index.vue')
const homePageSource = read('../pages/index/index.vue')
const playingPageSource = read('../subPages/match/playing.vue')
const userPageSource = read('../pages/user/index.vue')
const rankingPageSource = read('../pages/ranking/index.vue')
const resultPageSource = read('../subPages/match/matchResult.vue')
const confirmPageSource = read('../subPages/match/scanConfirm.vue')
const quickScanSource = read('../utils/quick-scan.js')
const quickScanEntrySource = read('../utils/quick-scan-entry.js')
const appSource = read('../App.vue')
const qrSources = [
  homePageSource,
  playingPageSource,
  userPageSource
]

test('invite API and start payload use a server-verifiable token instead of mutable profile fields', () => {
  assert.match(apiSource, /export const previewMatchInvite/)
  assert.match(apiSource, /post\('\/api\/match\/invite\/preview'/)
  assert.match(startPayloadSource, /if \(normalizedInviteToken\) \{/)
  assert.match(startPayloadSource, /payload\.invite_token = normalizedInviteToken/)
  assert.match(startPayloadSource, /if \(!inviteToken && \(!Number\.isFinite\(opponentId\) \|\| opponentId <= 0\)\)/)
})

test('every scan entry goes through the unified preview-first flow', () => {
  // 解析与预览只在统一流程里：先预览取得对手，再把 invite token 交给确认页开局
  assert.match(quickScanSource, /scanAndResolveMatchCode/)
  assert.match(quickScanEntrySource, /previewMatchInvite/)
  assert.match(confirmPageSource, /inviteToken\s*\n\s*\}\)\)/)
  assert.match(confirmPageSource, /buildStartMatchPayload\(\{[\s\S]*inviteToken[\s\S]*\}\)/)

  for (const source of [homePageSource, matchPageSource, rankingPageSource, userPageSource, resultPageSource, confirmPageSource]) {
    assert.doesNotMatch(source, /scanAndResolveMatchCode/)
    assert.doesNotMatch(source, /JSON.parse\(scanResult\)/)
  }
  for (const source of [homePageSource, rankingPageSource, resultPageSource, appSource]) {
    assert.match(source, /startQuickScan/)
  }
  assert.match(appSource, /uni\.onTabBarMidButtonTap\(\(\) => startQuickScan\(\)\)/)
})

test('all match and referee QR displays render locally without a third-party QR image service', () => {
  for (const source of qrSources) {
    assert.match(source, /renderLocalQRCode/)
    assert.doesNotMatch(source, unapprovedQrDomainPattern)
  }
})

test('local QR displays expose loading, failure, and refresh states', () => {
  assert.match(homePageSource, /v-if="matchQrLoading"/)
  assert.match(homePageSource, /matchQrFailed/)
  assert.match(homePageSource, /@tap="loadMyMatchQRCode"/)

  assert.match(userPageSource, /v-if="matchQrLoading"/)
  assert.match(userPageSource, /matchQrFailed/)
  assert.match(userPageSource, /@click="loadMyMatchQRCode"/)

  assert.match(playingPageSource, /refereeQrcodeLoading/)
  assert.match(playingPageSource, /refereeQrcodeError/)
  assert.match(playingPageSource, /refereeQrcodeContent\.value = ''/)
  assert.match(playingPageSource, /@click="openRefereeQrModal"/)

  // 观赛大厅只有列表，不再带二维码弹层
  assert.doesNotMatch(matchPageSource, /matchQrLoading|renderLocalQRCode|canvas/)
})

test('scan credentials stay in memory and out of routes, storage and logs', () => {
  assert.doesNotMatch(quickScanSource, /StorageSync|console\./)
  assert.doesNotMatch(confirmPageSource, /StorageSync\('(?!default-game-type)|setStorageSync\(.*(token|invite)/i)
  // 页面路由只带对局信息，凭证不外泄
  assert.doesNotMatch(confirmPageSource, /navigateTo\(\{[^}]*(inviteToken|join_token)/)
  assert.doesNotMatch(confirmPageSource, /redirectTo\(\{ url: `[^`]*(inviteToken|join_token)/)
  // 约球扫码上下文已移除，「再来一局」不再借本地存储交接
  for (const source of [matchPageSource, homePageSource, userPageSource, resultPageSource, confirmPageSource, quickScanSource]) {
    assert.doesNotMatch(source, /pending_match_challenge|pending_match_rematch/)
  }
})
