import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildStartMatchPayload,
  normalizePendingMatchContext,
  normalizeStartMatchOptions,
  validateScannedOpponentForContext,
  validateStartMatchPayload
} from '../utils/start-match.js'

test('validateStartMatchPayload rejects empty opponent id', () => {
  assert.equal(
    validateStartMatchPayload({
      game_type: 3,
      opponent_id: 0,
      opponent_name: '球友A'
    }),
    '请选择有效的平台对手'
  )
})

test('validateStartMatchPayload accepts positive opponent id', () => {
  assert.equal(
    validateStartMatchPayload({
      game_type: 3,
      opponent_id: 2001,
      opponent_name: '球友A'
    }),
    ''
  )
})

test('validateStartMatchPayload supports flexible snooker formats and legacy requests', () => {
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, snooker_format: 'free', snooker_target_wins: 0 }), '')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, snooker_format: 'free', snooker_target_wins: 1 }), '自由局数不能设置目标胜局')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, snooker_format: 'race_to', snooker_target_wins: 10 }), '')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, snooker_format: 'race_to', snooker_target_wins: 26 }), '抢N局目标必须在1至25之间')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, best_of_frames: 4, starting_actor: 1 }), '斯诺克总局数必须为正奇数')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, best_of_frames: 3, starting_actor: 0 }), '请选择首局开球方')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, best_of_frames: 7, starting_actor: 2 }), '')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, snooker_rules_version: 1 }), '')
  assert.equal(validateStartMatchPayload({ game_type: 1, opponent_id: 2001, snooker_rules_version: 3 }), '不支持的斯诺克规则版本')
})

test('normalizeStartMatchOptions defaults practice to private and ranked to public', () => {
  assert.deepEqual(normalizeStartMatchOptions({ match_mode: 'practice' }), {
    match_mode: 'practice',
    visibility: 'private'
  })
  assert.deepEqual(normalizeStartMatchOptions({ match_mode: 'ranked' }), {
    match_mode: 'ranked',
    visibility: 'public'
  })
})

test('buildStartMatchPayload carries challenge and mode context without bypassing opponent selection', () => {
  assert.deepEqual(
    buildStartMatchPayload({
      gameType: 3,
      opponent: { id: 2001, nickname: '球友A', avatar: 'a.png' },
      matchMode: 'practice',
      visibility: 'public',
      challengeId: 88
    }),
    {
      game_type: 3,
      opponent_id: 2001,
      opponent_name: '球友A',
      opponent_avatar: 'a.png',
      match_mode: 'practice',
      visibility: 'public',
      challenge_id: 88
    }
  )
})

test('buildStartMatchPayload defaults new snooker matches to free format', () => {
  assert.deepEqual(buildStartMatchPayload({
    gameType: 1,
    opponent: { id: 2001, nickname: '球友A' }
  }), {
    game_type: 1,
    opponent_id: 2001,
    opponent_name: '球友A',
    opponent_avatar: '',
    match_mode: 'ranked',
    visibility: 'public',
    snooker_rules_version: 2,
    snooker_format: 'free',
    snooker_target_wins: 0
  })
})

test('pending challenge and rematch contexts use independent required fields', () => {
  assert.equal(normalizePendingMatchContext('pending_match_challenge', JSON.stringify({
    challenge_id: 88,
    opponent_id: 2001,
    game_type: 3
  })).valid, true)
  assert.equal(normalizePendingMatchContext('pending_match_challenge', JSON.stringify({
    opponent_id: 2001,
    game_type: 3
  })).valid, false)
  assert.deepEqual(normalizePendingMatchContext('pending_match_rematch', JSON.stringify({
    opponent_id: 2001,
    game_type: 2,
    match_mode: 'ranked',
    visibility: 'public'
  })).context, {
    context_type: 'rematch',
    challenge_id: 0,
    opponent_id: 2001,
    game_type: 2,
    match_mode: 'ranked',
    visibility: 'public'
  })
  assert.deepEqual(normalizePendingMatchContext('pending_match_rematch', JSON.stringify({
    opponent_id: 2001,
    game_type: 1,
    snooker_format: 'race_to',
    snooker_target_wins: 5
  })).context, {
    context_type: 'rematch',
    challenge_id: 0,
    opponent_id: 2001,
    game_type: 1,
    match_mode: 'practice',
    visibility: 'private',
    snooker_rules_version: 2,
    snooker_format: 'race_to',
    snooker_target_wins: 5
  })
})

test('accepted challenge scan must match the invited opponent while payload uses scanned profile', () => {
  const context = { context_type: 'challenge', challenge_id: 88, opponent_id: 2001, game_type: 3 }
  assert.equal(validateScannedOpponentForContext(context, { user_id: 9999 }), '请扫描邀约中的指定对手')
  assert.equal(validateScannedOpponentForContext(context, { user_id: 2001 }), '')
  assert.deepEqual(buildStartMatchPayload({
    gameType: context.game_type,
    opponent: { user_id: 2001, nickname: '扫码昵称', avatar: 'scan.png' },
    matchMode: 'practice',
    visibility: 'private',
    challengeId: context.challenge_id
  }), {
    game_type: 3,
    opponent_id: 2001,
    opponent_name: '扫码昵称',
    opponent_avatar: 'scan.png',
    match_mode: 'practice',
    visibility: 'private',
    challenge_id: 88
  })
})
