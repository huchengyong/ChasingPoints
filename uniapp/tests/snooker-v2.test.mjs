import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildSnookerCountPages,
  buildSnookerV2FoulPayload,
  buildSnookerV2NoScorePayload,
  buildSnookerV2PotPayload,
  getSnookerV2BallOnLabel,
  getSnookerV2FreeBallOptions,
  getSnookerV2MinimumPenalty,
  getSnookerV2PhaseLabel,
  getSnookerV2PotOptions,
  isSnookerRulesV2,
  normalizeSnookerV2State
} from '../utils/snooker-v2.js'

test('normalizeSnookerV2State exposes authoritative visit and referee state', () => {
  const state = normalizeSnookerV2State({
    snooker_rules_version: 2,
    best_of_frames: 7,
    starting_actor: 2,
    snooker_phase: 'colors',
    snooker_ball_on: 'blue',
    snooker_striker: 2,
    snooker_visit_no: 6,
    snooker_current_break: 38,
    snooker_reds_remaining: 0,
    snooker_free_ball_available: true,
    snooker_cue_ball_in_hand: true,
    snooker_miss_warning_active: true,
    current_frame_started: true
  })
  assert.equal(isSnookerRulesV2({ snooker_rules_version: 2 }), true)
  assert.equal(state.bestOfFrames, 7)
  assert.equal(state.startingActor, 2)
  assert.equal(state.striker, 2)
  assert.equal(state.visitNo, 6)
  assert.equal(state.currentBreak, 38)
  assert.equal(state.freeBallAvailable, true)
  assert.equal(state.missWarningActive, true)
})

test('pot options follow red, color choice and ordered clearance ball-on', () => {
  assert.deepEqual(getSnookerV2PotOptions({ currentFrameStarted: true, status: 1, phase: 'reds', ballOn: 'red', redsRemaining: 15 }).map(item => item.value), [1])
  assert.deepEqual(getSnookerV2PotOptions({ currentFrameStarted: true, status: 1, phase: 'color_after_red', ballOn: 'color_choice' }).map(item => item.value), [2, 3, 4, 5, 6, 7])
  assert.deepEqual(getSnookerV2PotOptions({ currentFrameStarted: true, status: 1, phase: 'colors', ballOn: 'pink' }).map(item => item.value), [6])
  assert.deepEqual(getSnookerV2PotOptions({ currentFrameStarted: true, status: 1, phase: 'respotted_black_pending', ballOn: 'black' }), [])
})

test('red count pages never exceed the cross-platform action sheet limit', () => {
  assert.deepEqual(buildSnookerCountPages(1, 15), [
    [1, 2, 3, 4, 5, 6],
    [7, 8, 9, 10, 11, 12],
    [13, 14, 15]
  ])
  assert.deepEqual(buildSnookerCountPages(0, 15, 20).map(page => page.length), [6, 6, 4])
})

test('free ball options exclude the true ball on', () => {
  const state = { freeBallAvailable: true, ballOn: 'blue' }
  assert.deepEqual(getSnookerV2FreeBallOptions(state).map(item => item.value), [2, 3, 4, 6, 7])
})

test('stroke builders separate real reds, free ball and true target', () => {
  const redState = { ballOn: 'red' }
  assert.deepEqual(buildSnookerV2PotPayload({
    state: redState,
    actor: 1,
    pottedReds: 2,
    freeBallValue: 7,
    freeBallPotted: true
  }), {
    actor: 1,
    outcome: 'pot',
    ball_on_value: 1,
    potted_reds: 2,
    ball_on_potted: false,
    free_ball_value: 7,
    free_ball_potted: true
  })

  const colorState = { ballOn: 'color_choice' }
  assert.equal(buildSnookerV2PotPayload({ state: colorState, actor: 2, ballOnValue: 6, ballOnPotted: true }).ball_on_value, 6)
  assert.deepEqual(buildSnookerV2NoScorePayload({ state: { ballOn: 'red' }, actor: 2 }), {
    actor: 2,
    outcome: 'no_score',
    ball_on_value: 1
  })
  assert.deepEqual(buildSnookerV2NoScorePayload({ state: { ballOn: 'yellow' }, actor: 2, freeBallValue: 3 }), {
    actor: 2,
    outcome: 'no_score',
    ball_on_value: 2,
    free_ball_value: 3
  })
})

test('foul builder keeps only decisions valid for the selected resolution', () => {
  const replay = buildSnookerV2FoulPayload({
    state: { ballOn: 'black' },
    actor: 1,
    penalty: 7,
    redsRemoved: 1,
    resolution: 'offender_replays_original',
    freeBallAwarded: true,
    foulAndMiss: true,
    missSequenceEligible: true,
    cueBallInHand: true
  })
  assert.equal(replay.free_ball_awarded, false)
  assert.equal(replay.miss_sequence_eligible, true)
  assert.equal(replay.cue_ball_in_hand, false)
  assert.equal(getSnookerV2MinimumPenalty({ ballOn: 'pink' }), 6)
})

test('phase and target copy are derived from authoritative state', () => {
  assert.equal(getSnookerV2PhaseLabel({ phase: 'respotted_black' }), '重置黑球决胜')
  assert.equal(getSnookerV2BallOnLabel({ ballOn: 'color_choice' }), '自选彩球')
})

test('playing page contains version split and all referee-assisted v2 actions', () => {
  const page = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
  assert.match(page, /data-snooker-v2-panel/)
  assert.match(page, /data-snooker-v1-panel/)
  assert.match(page, /合法未进球 \/ 上手结束/)
  assert.match(page, /data-snooker-v2-foul-editor/)
  assert.match(page, /Foul and a Miss/)
  assert.match(page, /start_respotted_black/)
  assert.match(page, /offer_concession/)
  assert.match(page, /accept_concession/)
  assert.match(page, /snookerStroke/)
  assert.match(page, /snookerFrameAction/)
  assert.match(page, /仅真实目标球入袋/)
  assert.match(page, /buildSnookerCountPages/)
  assert.match(page, /currentFrameStarted && !snookerV2State\.pendingConcessionActor/)
  assert.match(page, /normalizedActor !== 1 && normalizedActor !== 2\) return '待确定'/)
  assert.match(page, /\{\{ snookerActorLabel\(1\) \}\}先打/)
  assert.doesNotMatch(page, /Array\.from\(\{ length: (?:maxReds|state\.redsRemaining \+ 1)/)
})

test('match page starts new snooker matches with free format without pre-scan format prompts', () => {
  const page = readFileSync(new URL('../pages/match/index.vue', import.meta.url), 'utf8')
  const helper = readFileSync(new URL('../utils/start-match.js', import.meta.url), 'utf8')
  assert.doesNotMatch(page, /chooseSnookerFormatThenScan|chooseSnookerStartFormat|snookerBestOfFrames|snookerStartingActor/)
  assert.match(helper, /snooker_format = 'free'/)
  assert.match(helper, /snooker_target_wins = 0/)
})
