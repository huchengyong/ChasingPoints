import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildStartMatchPayload,
  normalizePendingMatchContext,
  normalizeStartMatchOptions,
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

test('buildStartMatchPayload never sends challenge_id; matches come from both players entering', () => {
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
      match_format: 'free',
      target_wins: 0
    }
  )
})

test('signed invite start payload omits mutable opponent identity fields', () => {
  const payload = buildStartMatchPayload({
    gameType: 3,
    opponent: { id: 2001, nickname: '仅用于预览' },
    matchMode: 'practice',
    challengeId: 88,
    inviteToken: 'signed.invite'
  })
  assert.deepEqual(payload, {
    game_type: 3,
    match_mode: 'practice',
    visibility: 'private',
    invite_token: 'signed.invite',
    match_format: 'free',
    target_wins: 0
  })
  assert.equal(validateStartMatchPayload(payload), '')
})

test('validateStartMatchPayload supports pool free and race-to formats', () => {
  assert.equal(validateStartMatchPayload({ game_type: 3, opponent_id: 2001, match_format: 'free', target_wins: 0 }), '')
  assert.equal(validateStartMatchPayload({ game_type: 4, opponent_id: 2001, match_format: 'race_to', target_wins: 65 }), '')
  assert.equal(validateStartMatchPayload({ game_type: 3, opponent_id: 2001, match_format: 'free', target_wins: 1 }), '自由局数不能设置目标胜局')
  assert.equal(validateStartMatchPayload({ game_type: 4, opponent_id: 2001, match_format: 'race_to', target_wins: 66 }), '抢N局目标必须在1至65之间')
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

test('challenge scan context is retired; rematch contexts use independent required fields', () => {
  assert.equal(normalizePendingMatchContext('pending_match_challenge', JSON.stringify({
    challenge_id: 88,
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
    game_type: 4,
    match_format: 'race_to',
    target_wins: 65
  })).context, {
    context_type: 'rematch',
    challenge_id: 0,
    opponent_id: 2001,
    game_type: 4,
    match_mode: 'practice',
    visibility: 'private',
    match_format: 'free',
    target_wins: 0
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

test('scan start payload keeps the signed invite and never sends challenge_id', () => {
  assert.equal(typeof validateScannedOpponentForContext, 'undefined')
  assert.deepEqual(buildStartMatchPayload({
    gameType: 3,
    opponent: { user_id: 2001, nickname: '扫码昵称', avatar: 'scan.png' },
    matchMode: 'practice',
    visibility: 'private',
    inviteToken: 'signed.invite'
  }), {
    game_type: 3,
    match_mode: 'practice',
    visibility: 'private',
    invite_token: 'signed.invite',
    match_format: 'free',
    target_wins: 0
  })
})
