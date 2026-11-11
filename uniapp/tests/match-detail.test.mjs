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

test('normalizeMatchDetailPayload swaps participant round records once for player 2', () => {
  const rounds = [
    { round_number: 1, player1_score: 21, player2_score: 10, winner: 1 },
    { round_number: 2, player1_score: 33, player2_score: 21, winner: 2 },
    { round_number: 3, player1_score: 8, player2_score: 60, winner: 2 }
  ]
  const asPlayer2 = normalizeMatchDetailPayload({
    is_player1: false,
    viewer_role: 'player2',
    rounds
  })
  assert.deepEqual(asPlayer2.roundRecords.map((round) => ({
    score: `${round.player1_score}-${round.player2_score}`,
    winner: round.winner,
    result: round.result,
    resultText: round.resultText
  })), [
    { score: '10-21', winner: 2, result: 'loss', resultText: '负' },
    { score: '21-33', winner: 1, result: 'win', resultText: '胜' },
    { score: '60-8', winner: 1, result: 'win', resultText: '胜' }
  ])
  assert.equal(asPlayer2.roundRecordsSwapped, true)

  // 参与者详情同时带 perspective_user_id 时也只换一次边。
  const withPerspective = normalizeMatchDetailPayload({
    is_player1: false,
    viewer_role: 'player2',
    player1_id: 1,
    player2_id: 2,
    rounds
  }, { perspectiveUserId: 2 })
  assert.deepEqual(withPerspective.roundRecords.map((round) => `${round.player1_score}-${round.player2_score}`), ['10-21', '21-33', '60-8'])
})

test('normalizeMatchDetailPayload keeps round records for player 1 and referees', () => {
  const rounds = [{ round_number: 1, player1_score: 21, player2_score: 10, winner: 1 }]
  const asPlayer1 = normalizeMatchDetailPayload({ is_player1: true, viewer_role: 'player1', rounds })
  assert.deepEqual(asPlayer1.roundRecords.map((round) => [round.player1_score, round.player2_score, round.resultText]), [[21, 10, '胜']])
  assert.equal(asPlayer1.roundRecordsSwapped, false)

  const asReferee = normalizeMatchDetailPayload({ is_player1: false, viewer_role: 'referee', rounds })
  assert.deepEqual(asReferee.roundRecords.map((round) => [round.player1_score, round.player2_score, round.resultText]), [[21, 10, '胜']])
  assert.equal(asReferee.roundRecordsSwapped, false)
})

const chasingBreakdown = {
  available: true,
  player1: { score: 16, normal_wins: 2, small_golds: 1, big_golds: 0, foul_count: 3, foul_points_given: 3, other_points: 0 },
  player2: { score: 13, normal_wins: 0, small_golds: 0, big_golds: 1, foul_count: 1, foul_points_given: 1, other_points: 0 },
  events: [
    { seq: 1, kind: 'big_gold', scorer: 2, points: 10, player1_score: 5, player2_score: 10, created_at: '2026-11-10 12:00:03' },
    { seq: 2, kind: 'foul', scorer: 1, points: 1, player1_score: 11, player2_score: 10, created_at: '2026-11-10 12:00:04' }
  ],
  events_truncated: false
}

const chasingMatch = {
  player1_id: 1,
  player2_id: 2,
  player1_name: '创建者',
  player2_name: '当前用户',
  player1_score: 16,
  player2_score: 13,
  game_type: 2,
  status: 2,
  score_breakdown: chasingBreakdown
}

test('normalizeMatchDetailPayload maps chasing score breakdown for player 2 with swapped sides', () => {
  const detail = normalizeMatchDetailPayload(chasingMatch, { perspectiveUserId: 2 })
  const view = detail.scoreBreakdown
  assert.deepEqual([view.leftLabel, view.rightLabel], ['我', '对手'])
  assert.deepEqual([view.leftScore, view.rightScore], [13, 16])
  assert.deepEqual(view.rows.slice(0, 4).map((row) => [row.label, row.left, row.right]), [
    ['总分', '13', '16'],
    ['普胜', '0', '2'],
    ['小金', '0', '1'],
    ['大金', '1', '0']
  ])
  assert.deepEqual([view.events[0].side, view.events[0].desc], ['right', '我犯规'])
  assert.equal(view.events[0].scoreText, '10 : 11')
  assert.deepEqual([view.events[1].side, view.events[1].scoreText], ['left', '10 : 5'])
})

test('normalizeMatchDetailPayload keeps player1 and player2 labels for spectators', () => {
  const view = normalizeMatchDetailPayload(chasingMatch, { perspectiveUserId: 0 }).scoreBreakdown
  assert.deepEqual([view.leftLabel, view.rightLabel], ['选手1', '选手2'])
  assert.deepEqual([view.leftScore, view.rightScore], [16, 13])
  assert.deepEqual([view.events[0].side, view.events[0].sideLabel], ['left', '选手1'])
  assert.equal(view.events[0].desc, '选手2犯规')
  assert.deepEqual([view.events[1].side, view.events[1].sideLabel], ['right', '选手2'])
})

test('normalizeMatchDetailPayload keeps the referee perspective neutral even for a player', () => {
  const view = normalizeMatchDetailPayload({ ...chasingMatch, viewer_role: 'referee' }, { perspectiveUserId: 2 }).scoreBreakdown
  assert.deepEqual([view.leftLabel, view.rightLabel], ['选手1', '选手2'])
  assert.deepEqual([view.leftScore, view.rightScore], [16, 13])
})

test('normalizeMatchDetailPayload has no score breakdown for other game types or missing data', () => {
  assert.equal(normalizeMatchDetailPayload({ ...chasingMatch, game_type: 3 }).scoreBreakdown, null)
  assert.equal(normalizeMatchDetailPayload({ ...chasingMatch, score_breakdown: undefined }).scoreBreakdown, null)
  assert.equal(normalizeMatchDetailPayload({ game_type: 3, player1_score: 5, player2_score: 3 }).scoreBreakdown, null)
})

// 参与者详情接口：比分已按查看者返回「我 / 对手」，得分构成仍是固定的选手1 / 选手2。
const ownChasingMatch = (overrides = {}) => ({
  player1_id: 1,
  player2_id: 2,
  my_name: '我',
  opponent_name: '对手',
  game_type: 2,
  status: 2,
  score_breakdown: chasingBreakdown,
  ...overrides
})

test('normalizeMatchDetailPayload follows the viewer on the participant detail without a perspective user', () => {
  const asPlayer2 = normalizeMatchDetailPayload(ownChasingMatch({
    is_player1: false, viewer_role: 'player2', my_score: 13, opponent_score: 16
  }))
  assert.deepEqual([asPlayer2.matchData.player1_score, asPlayer2.matchData.player2_score], [13, 16])
  assert.deepEqual([asPlayer2.scoreBreakdown.leftLabel, asPlayer2.scoreBreakdown.rightLabel], ['我', '对手'])
  assert.deepEqual([asPlayer2.scoreBreakdown.leftScore, asPlayer2.scoreBreakdown.rightScore], [13, 16])

  const asPlayer1 = normalizeMatchDetailPayload(ownChasingMatch({
    is_player1: true, viewer_role: 'player1', my_score: 16, opponent_score: 13
  }))
  assert.deepEqual([asPlayer1.scoreBreakdown.leftLabel, asPlayer1.scoreBreakdown.leftScore, asPlayer1.scoreBreakdown.rightScore], ['我', 16, 13])
})

test('normalizeMatchDetailPayload keeps the referee neutral on the participant detail', () => {
  const view = normalizeMatchDetailPayload(ownChasingMatch({
    is_player1: false, viewer_role: 'referee', my_score: 16, opponent_score: 13
  })).scoreBreakdown
  assert.deepEqual([view.leftLabel, view.rightLabel], ['选手1', '选手2'])
  assert.deepEqual([view.leftScore, view.rightScore], [16, 13])
})
