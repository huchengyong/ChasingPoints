import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  SNOOKER_FORMAT_FREE,
  SNOOKER_FORMAT_RACE_TO,
  SNOOKER_TARGET_OPTIONS,
  canFinishFreeSnookerMatch,
  canStartNextSnookerFrame,
  getSnookerFormatHint,
  getSnookerFormatLabel,
  normalizeSnookerMatchFormat
} from '../utils/snooker-match-format.js'

test('snooker format normalizes free and clamps race-to targets', () => {
  assert.deepEqual(normalizeSnookerMatchFormat({}), { format: SNOOKER_FORMAT_FREE, targetWins: 0 })
  assert.deepEqual(normalizeSnookerMatchFormat({ snooker_format: 'race_to', snooker_target_wins: 10 }), { format: SNOOKER_FORMAT_RACE_TO, targetWins: 10 })
  assert.deepEqual(normalizeSnookerMatchFormat({ snooker_format: 'race_to', snooker_target_wins: 99 }), { format: SNOOKER_FORMAT_RACE_TO, targetWins: 25 })
  assert.equal(SNOOKER_TARGET_OPTIONS.length, 25)
})

test('snooker format copy describes free and race-to limits', () => {
  assert.equal(getSnookerFormatLabel('free'), '自由局数')
  assert.equal(getSnookerFormatLabel('race_to', 10), '抢10局制')
  assert.equal(getSnookerFormatHint('free'), '不预设胜局，最多进行49局')
  assert.equal(getSnookerFormatHint('race_to', 10), '先达到10胜，最多可能进行19局')
})

test('free format can end only after a completed frame and stops at 49 frames', () => {
  assert.equal(canFinishFreeSnookerMatch({ format: 'free', currentFrameStarted: false, myScore: 1, opponentScore: 1 }), true)
  assert.equal(canFinishFreeSnookerMatch({ format: 'free', currentFrameStarted: true, myScore: 1, opponentScore: 0 }), false)
  assert.equal(canFinishFreeSnookerMatch({ format: 'free', currentFrameStarted: false, myScore: 0, opponentScore: 0 }), false)
  assert.equal(canStartNextSnookerFrame({ format: 'free', myScore: 24, opponentScore: 24 }), true)
  assert.equal(canStartNextSnookerFrame({ format: 'free', myScore: 25, opponentScore: 24 }), false)
})

test('race-to format stops when either player reaches the target', () => {
	assert.equal(canStartNextSnookerFrame({ format: 'race_to', targetWins: 10, myScore: 9, opponentScore: 9 }), true)
	assert.equal(canStartNextSnookerFrame({ format: 'race_to', targetWins: 10, myScore: 10, opponentScore: 9 }), false)
})

test('playing page scopes the inline format editor to unlocked snooker v2 matches', () => {
	const page = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
	const overviewIndex = page.indexOf('class="snooker-v2-status snooker-format-overview"')
	const actionPanelIndex = page.indexOf('v-if="viewerUi.showActionPanel" class="action-panel"')
	assert.ok(overviewIndex >= 0 && overviewIndex < actionPanelIndex)
	assert.match(page.slice(overviewIndex, actionPanelIndex), /snookerFormatLabel/)
	assert.match(page, /v-if="gameType === 1 && isSnookerV2" class="snooker-v2-layout"/)
	assert.match(page, /v-if="canChangeSnookerFormat" type="right"/)
	assert.match(page, /v-if="showSnookerFormatEditor && canChangeSnookerFormat"/)
	assert.match(page, /SNOOKER_TARGET_OPTIONS/)
	assert.match(page, /updateSnookerFormat\(\{[\s\S]*base_revision: serverRevision\.value/)
	assert.match(page, /if \(res\?\.snapshot\) applyMatchSnapshot\(res\.snapshot\)/)
	assert.doesNotMatch(page, /赛制已锁定/)
})

test('playing page keeps free-format normal finish separate from match concession', () => {
	const page = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
	assert.match(page, /canFinishFreeSnookerMatch/)
	assert.match(page, /结束整场对局/)
	assert.match(page, /viewerUi\.showFinishButton[\s\S]{0,300}snookerFormat === SNOOKER_FORMAT_FREE[\s\S]{0,120}'结束整场对局'/)
	assert.doesNotMatch(page, /snookerFormat === 'free'/)
	assert.match(page, /handleOfferSnookerConcession\('match'\)/)
	assert.match(page, /gameType === 1[\s\S]*canStartNextSnookerFrameValue/)
})
