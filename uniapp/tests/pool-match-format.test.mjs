import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  POOL_MATCH_FORMAT_FREE,
  POOL_MATCH_FORMAT_LEGACY,
  POOL_MATCH_FORMAT_RACE_TO,
  POOL_TARGET_OPTIONS,
  canContinuePoolMatch,
  canFinishFreePoolMatch,
  getPoolMatchFormatHint,
  getPoolMatchFormatLabel,
  isPoolMatchFormatGameType,
  normalizePoolMatchFormat
} from '../utils/pool-match-format.js'

test('pool match format normalizes legacy, free and race-to targets', () => {
  assert.deepEqual(normalizePoolMatchFormat({}), { format: POOL_MATCH_FORMAT_FREE, targetWins: 0 })
  assert.deepEqual(normalizePoolMatchFormat({ match_format: 'legacy' }), { format: POOL_MATCH_FORMAT_LEGACY, targetWins: 0 })
  assert.deepEqual(normalizePoolMatchFormat({ match_format: 'race_to', target_wins: 65 }), { format: POOL_MATCH_FORMAT_RACE_TO, targetWins: 65 })
  assert.deepEqual(normalizePoolMatchFormat({ match_format: 'race_to', target_wins: 99 }), { format: POOL_MATCH_FORMAT_RACE_TO, targetWins: 65 })
  assert.equal(POOL_TARGET_OPTIONS.length, 65)
})

test('pool match format copy describes free, draw and race-to limits', () => {
  assert.equal(getPoolMatchFormatLabel('free'), '自由局数')
  assert.equal(getPoolMatchFormatLabel('race_to', 65), '抢65局制')
  assert.equal(getPoolMatchFormatHint('free'), '不预设胜局，最多进行129局，允许平局')
  assert.equal(getPoolMatchFormatHint('race_to', 65), '先达到65胜，最多可能进行129局')
})

test('pool finish and continuation rules mirror server limits', () => {
  assert.equal(canFinishFreePoolMatch({ format: 'free', myScore: 0, opponentScore: 0 }), false)
  assert.equal(canFinishFreePoolMatch({ format: 'free', myScore: 1, opponentScore: 0 }), true)
  assert.equal(canFinishFreePoolMatch({ format: 'free', myScore: 64, opponentScore: 64 }), true)
  assert.equal(canContinuePoolMatch({ format: 'free', myScore: 64, opponentScore: 64 }), true)
  assert.equal(canContinuePoolMatch({ format: 'free', myScore: 65, opponentScore: 64 }), false)
  assert.equal(canContinuePoolMatch({ format: 'race_to', targetWins: 65, myScore: 64, opponentScore: 64 }), true)
  assert.equal(canContinuePoolMatch({ format: 'race_to', targetWins: 65, myScore: 65, opponentScore: 64 }), false)
})

test('pool match format applies only to chinese eight and american nine ball', () => {
  assert.equal(isPoolMatchFormatGameType(3), true)
  assert.equal(isPoolMatchFormatGameType(4), true)
  assert.equal(isPoolMatchFormatGameType(2), false)
})

test('playing page exposes pool format editor for type 3 and 4 snapshots only', () => {
  const page = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
  assert.match(page, /isPoolMatchFormatVisible/)
  assert.match(page, /updateMatchFormat\(\{[\s\S]*match_format: draftMatchFormat\.value[\s\S]*base_revision: serverRevision\.value/)
  assert.match(page, /can_change_match_format/)
  assert.match(page, /POOL_TARGET_OPTIONS/)
  assert.match(page, /matchFormat\.value !== POOL_MATCH_FORMAT_LEGACY/)
  assert.match(page, /isPoolMatchFormatGameType\(gameType\.value\)/)
  assert.match(page, /POOL_MATCH_FORMAT_FREE && !canFinishCurrentFreePoolMatch\.value/)
})
