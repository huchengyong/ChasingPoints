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
    total_rounds: 0
  })
  assert.deepEqual(detail.player1Info, {
    name: '张三',
    avatar: 'a.png'
  })
  assert.deepEqual(detail.player2Info, {
    name: '李四',
    avatar: 'b.png'
  })
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
