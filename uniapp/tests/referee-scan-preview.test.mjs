import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const confirmPage = readFileSync(new URL('../subPages/match/scanConfirm.vue', import.meta.url), 'utf8')
const quickScan = readFileSync(new URL('../utils/quick-scan.js', import.meta.url), 'utf8')
const matchPage = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
const playingPage = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
const detailPage = readFileSync(new URL('../subPages/match/matchDetail.vue', import.meta.url), 'utf8')

test('referee QR scan previews before joining and offers cancel or retry', () => {
  // 裁判码由统一扫码流程识别，进入确认页的执裁模式，先预览再确认
  assert.match(quickScan, /result\.type === 'join_referee'/)
  assert.match(quickScan, /mode: 'referee'/)
  assert.match(confirmPage, /previewMatchReferee\(refereeJoin\)/)
  assert.match(confirmPage, /确认担任本场裁判/)
  assert.match(confirmPage, /暂不担任/)
  assert.match(confirmPage, /重新扫码/)
  assert.match(confirmPage, /joinMatchReferee\(join\)/)
  // 预览成功后才出现「确认担任裁判」，加入只在点击确认时调用
  assert.match(confirmPage, /@tap="handleJoin"/)
  assert.doesNotMatch(confirmPage, /onLoad\([\s\S]{0,900}joinMatchReferee/)
})

test('the lobby no longer carries any referee scan UI', () => {
  assert.doesNotMatch(matchPage, /previewMatchReferee|joinMatchReferee|referee-preview|handleScanAsReferee/)
})

test('playing and detail pages render neutral referee identity while hiding referee H2H', () => {
  assert.match(playingPage, /referee_avatar/)
  assert.match(playingPage, /本场裁判：/)
  assert.match(detailPage, /class="referee-card"/)
  assert.match(detailPage, /const showH2H = computed/)
  assert.match(detailPage, /viewer_role !== 'referee'/)
})
