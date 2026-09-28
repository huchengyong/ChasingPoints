import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const matchPage = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
const playingPage = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
const detailPage = readFileSync(new URL('../subPages/match/matchDetail.vue', import.meta.url), 'utf8')

test('referee QR scan previews before joining and offers cancel or retry', () => {
  assert.match(matchPage, /previewMatchReferee/)
  assert.match(matchPage, /确认担任本场裁判/)
  assert.match(matchPage, /暂不担任/)
  assert.match(matchPage, /重新扫码/)
  assert.match(matchPage, /await openRefereePreview\(scanAction\.refereeJoin\)/)
})

test('playing and detail pages render neutral referee identity while hiding referee H2H', () => {
  assert.match(playingPage, /referee_avatar/)
  assert.match(playingPage, /本场裁判：/)
  assert.match(detailPage, /class="referee-card"/)
  assert.match(detailPage, /const showH2H = computed/)
  assert.match(detailPage, /viewer_role !== 'referee'/)
})
