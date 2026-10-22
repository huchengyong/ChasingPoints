import test from 'node:test'
import assert from 'node:assert/strict'

import {
  normalizeMatchDetailPayload,
  shouldShowSpectateBadge,
  shouldUsePublicMatchDetail
} from '../utils/match-detail.js'

test('history match detail uses public detail data without showing spectate badge', () => {
  assert.equal(shouldUsePublicMatchDetail({ source: 'history' }), true)
  assert.equal(shouldShowSpectateBadge({ source: 'history' }), false)
})

test('spectate match detail uses public detail data and shows spectate badge', () => {
  assert.equal(shouldUsePublicMatchDetail({ mode: 'spectate' }), true)
  assert.equal(shouldShowSpectateBadge({ mode: 'spectate' }), true)
})

test('normalizeMatchDetailPayload maps private match detail fields into display fields', () => {
  const detail = normalizeMatchDetailPayload({
    my_score: 7,
    opponent_score: 5,
    my_name: '张三',
    opponent_name: '李四',
    my_avatar: 'a.png',
    opponent_avatar: 'b.png',
    game_type: 2,
    status: 2
  })

  assert.deepEqual(detail.matchData, {
    player1_score: 7,
    player2_score: 5,
    game_type: 2,
    status: 2,
    duration_seconds: 0,
    current_round: 1,
    total_rounds: 0,
    viewer_role: '',
    referee_bound: false,
    referee_user_id: 0,
    referee_name: '',
    referee_avatar: '',
    referee_joined_at: '',
    referee_duration_seconds: 0,
    completed_by_user_id: 0,
    completion_source: 'unknown'
  })
  assert.deepEqual(detail.player1Info, {
    name: '张三',
    avatar: 'a.png',
    vipLevel: 0,
    rankLevel: 0
  })
  assert.deepEqual(detail.player2Info, {
    name: '李四',
    avatar: 'b.png',
    vipLevel: 0,
    rankLevel: 0
  })
})

test('normalizeMatchDetailPayload keeps each side member identity next to its own avatar', () => {
  const detail = normalizeMatchDetailPayload({
    my_name: '张三',
    opponent_name: '李四',
    my_avatar: 'a.png',
    opponent_avatar: 'b.png',
    my_vip_level: 0,
    my_rank_level: 0,
    opponent_vip_level: 5,
    opponent_rank_level: 4
  })

  // 免费用户看到对手会员身份，自己一侧没有。
  assert.deepEqual([detail.player1Info.vipLevel, detail.player1Info.rankLevel], [0, 0])
  assert.deepEqual([detail.player2Info.vipLevel, detail.player2Info.rankLevel], [5, 4])
})

test('normalizeMatchDetailPayload swaps member identity together with the avatar for the other perspective', () => {
  const detail = normalizeMatchDetailPayload({
    player1_id: 1,
    player2_id: 2,
    player1_name: '创建者',
    player2_name: '当前用户',
    player1_avatar: 'p1.png',
    player2_avatar: 'p2.png',
    player1_vip_level: 3,
    player1_rank_level: 2,
    player2_vip_level: 0,
    player2_rank_level: 0
  }, { perspectiveUserId: 2 })

  assert.equal(detail.player1Info.avatar, 'p2.png')
  assert.equal(detail.player1Info.vipLevel, 0)
  assert.equal(detail.player2Info.avatar, 'p1.png')
  assert.deepEqual([detail.player2Info.vipLevel, detail.player2Info.rankLevel], [3, 2])
})

test('normalizeMatchDetailPayload treats a payload without member fields as non-members', () => {
  const detail = normalizeMatchDetailPayload({ player1_id: 1, player2_id: 2, player1_name: 'A', player2_name: 'B' })
  assert.deepEqual([detail.player1Info.vipLevel, detail.player2Info.vipLevel], [0, 0])
})

test('normalizeMatchDetailPayload keeps referee identity and completion attribution', () => {
  const detail = normalizeMatchDetailPayload({
    player1_score: 5,
    player2_score: 3,
    game_type: 3,
    status: 2,
    viewer_role: 'referee',
    referee_bound: true,
    referee_user_id: 3,
    referee_name: '裁判丙',
    referee_avatar: 'referee.png',
    referee_joined_at: '2026-07-22T14:00:00+08:00',
    referee_duration_seconds: 600,
    completed_by_user_id: 3,
    completion_source: 'referee'
  })

  assert.equal(detail.matchData.viewer_role, 'referee')
  assert.equal(detail.matchData.referee_name, '裁判丙')
  assert.equal(detail.matchData.referee_avatar, 'referee.png')
  assert.equal(detail.matchData.completion_source, 'referee')
})

test('normalizeMatchDetailPayload converts public rounds to the requested user perspective', () => {
  const detail = normalizeMatchDetailPayload({
    player1_id: 1,
    player1_name: '创建者',
    player1_score: 66,
    player2_id: 2,
    player2_name: '当前用户',
    player2_score: 52,
    rounds: [
      { round_number: 1, player1_score: 21, player2_score: 0, winner: 1 },
      { round_number: 2, player1_score: 21, player2_score: 10, winner: 2 }
    ]
  }, {
    perspectiveUserId: 2
  })

  assert.equal(detail.player1Info.name, '当前用户')
  assert.equal(detail.player2Info.name, '创建者')
  assert.equal(detail.matchData.player1_score, 52)
  assert.equal(detail.matchData.player2_score, 66)
  assert.deepEqual(detail.roundRecords.map((round) => ({
    score: `${round.player1_score}-${round.player2_score}`,
    result: round.result,
    resultText: round.resultText
  })), [
    { score: '0-21', result: 'loss', resultText: '负' },
    { score: '10-21', result: 'win', resultText: '胜' }
  ])
})
