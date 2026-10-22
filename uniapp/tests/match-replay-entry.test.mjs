import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import { buildMatchReplayUrl, canOpenMatchReplay } from '../utils/match-replay-entry.js'

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')

test('the replay entry is only offered to players of a finished match', () => {
  assert.equal(canOpenMatchReplay({ status: 2, viewerRole: 'player1' }), true)
  assert.equal(canOpenMatchReplay({ status: 2, viewerRole: 'player2' }), true)
  assert.equal(canOpenMatchReplay({ status: 2, viewerRole: 'referee' }), false, 'referees have no replay entry')
  assert.equal(canOpenMatchReplay({ status: 1, viewerRole: 'player1' }), false, 'a running match has no replay')
  assert.equal(canOpenMatchReplay({ status: 3, viewerRole: 'player1' }), false, 'a cancelled match has no replay')
  assert.equal(canOpenMatchReplay({ status: 2 }), false, 'spectators have no replay entry')
  assert.equal(canOpenMatchReplay(), false)
})

test('a history detail opened for the logged-in user counts as their own match', () => {
  assert.equal(canOpenMatchReplay({ status: 2, source: 'history', perspectiveUserId: 5, currentUserId: 5 }), true)
  // 查看别人的历史对局（视角用户不是自己）没有入口。
  assert.equal(canOpenMatchReplay({ status: 2, source: 'history', perspectiveUserId: 6, currentUserId: 5 }), false)
  assert.equal(canOpenMatchReplay({ status: 2, source: 'history', perspectiveUserId: 0, currentUserId: 0 }), false)
  assert.equal(canOpenMatchReplay({ status: 2, source: '', perspectiveUserId: 5, currentUserId: 5 }), false)
})

test('replay url carries the match id and rejects bad ids', () => {
  assert.equal(buildMatchReplayUrl(88), '/subPages/match/matchReplay?match_id=88')
  assert.equal(buildMatchReplayUrl('x'), '/subPages/match/matchReplay?match_id=0')
})

test('result and detail pages wire the replay entry through the shared helper', () => {
  const result = read('subPages/match/matchResult.vue')
  assert.match(result, /canOpenMatchReplay\(\{\s*status: matchData\.value\.status,\s*viewerRole: matchData\.value\.viewer_role/)
  assert.match(result, /v-if="showReplayEntry"[\s\S]*单场复盘/)
  assert.match(result, /uni\.navigateTo\(\{ url: buildMatchReplayUrl\(matchId\.value\) \}\)/)

  const detail = read('subPages/match/matchDetail.vue')
  assert.match(detail, /canOpenMatchReplay\(\{[\s\S]*source: detailSource\.value[\s\S]*perspectiveUserId: perspectiveUserId\.value[\s\S]*currentUserId: userStore\.userId/)
  assert.match(detail, /<view class="detail-entry-row" v-if="showH2H \|\| showRematch \|\| showReplay">/)
  assert.match(detail, /v-if="showReplay"[^>]*@tap="handleOpenReplay"[^>]*>单场复盘/)
  assert.match(detail, /buildMatchReplayUrl\(matchId\.value\)/)
})
