import test from 'node:test'
import assert from 'node:assert/strict'

import { resolveChasingScoreView } from '../utils/chasing-score.js'

const breakdown = {
  available: true,
  player1: { score: 16, normal_wins: 2, small_golds: 1, big_golds: 0, foul_count: 3, foul_points_given: 3, other_points: 0 },
  player2: { score: 13, normal_wins: 0, small_golds: 0, big_golds: 1, foul_count: 1, foul_points_given: 1, other_points: 0 },
  events: [
    { seq: 1, kind: 'foul', scorer: 1, points: 1, player1_score: 1, player2_score: 0, created_at: '2026-11-10 12:00:01' },
    { seq: 2, kind: 'normal', scorer: 1, points: 4, player1_score: 5, player2_score: 0, created_at: '2026-11-10 12:00:02' },
    { seq: 3, kind: 'big_gold', scorer: 2, points: 10, player1_score: 5, player2_score: 10, created_at: '2026-11-10 12:00:03' }
  ],
  events_truncated: false
}

test('returns null for anything that is not a breakdown object', () => {
  for (const value of [null, undefined, [], 'score', 3]) {
    assert.equal(resolveChasingScoreView(value), null)
  }
})

test('player1 perspective shows me and opponent with fixed row order', () => {
  const view = resolveChasingScoreView(breakdown)
  assert.equal(view.available, true)
  assert.deepEqual([view.leftLabel, view.rightLabel], ['我', '对手'])
  assert.deepEqual([view.leftScore, view.rightScore], [16, 13])
  assert.deepEqual(view.rows.map((row) => [row.key, row.label, row.left, row.right]), [
    ['score', '总分', '16', '13'],
    ['normal_wins', '普胜', '2', '0'],
    ['small_golds', '小金', '1', '0'],
    ['big_golds', '大金', '0', '1'],
    ['foul_count', '犯规', '3', '1'],
    ['foul_points_given', '犯规送分', '3', '1']
  ])
  assert.equal(view.notice, '')
  assert.equal(view.eventsTruncated, false)
})

test('player2 perspective swaps rows and event scores', () => {
  const view = resolveChasingScoreView(breakdown, { swap: true })
  assert.deepEqual([view.leftScore, view.rightScore], [13, 16])
  assert.deepEqual(view.rows.slice(0, 3).map((row) => [row.label, row.left, row.right]), [
    ['总分', '13', '16'],
    ['普胜', '0', '2'],
    ['小金', '0', '1']
  ])
  const [latest, , fouled] = view.events
  assert.equal(latest.kindLabel, '大金')
  assert.deepEqual([latest.side, latest.sideLabel], ['left', '我'])
  assert.equal(latest.scoreText, '10 : 5')
  // 犯规方（选手2）在右列，犯规条目的得分方在左列。
  assert.deepEqual([fouled.side, fouled.desc], ['right', '我犯规'])
  assert.equal(fouled.scoreText, '0 : 1')
})

test('neutral perspective ignores swap and labels sides player1 and player2', () => {
  const view = resolveChasingScoreView(breakdown, { swap: true, neutral: true })
  assert.deepEqual([view.leftLabel, view.rightLabel], ['选手1', '选手2'])
  assert.deepEqual([view.leftScore, view.rightScore], [16, 13])
  const [latest, , fouled] = view.events
  assert.deepEqual([latest.side, latest.sideLabel], ['right', '选手2'])
  assert.equal(latest.scoreText, '5 : 10')
  assert.deepEqual([fouled.side, fouled.desc], ['left', '选手2犯规'])
})

test('events are listed newest first with time and description text', () => {
  const view = resolveChasingScoreView(breakdown, { neutral: true })
  assert.deepEqual(view.events.map((event) => event.seq), [3, 2, 1])
  assert.deepEqual(view.events.map((event) => event.kindLabel), ['大金', '普胜', '犯规'])
  assert.deepEqual(view.events.map((event) => event.pointsText), ['+10', '+4', '选手1 +1'])
  assert.deepEqual(view.events.map((event) => event.desc), ['选手2 大金', '选手1 普胜', '选手2犯规'])
  assert.deepEqual(view.events.map((event) => event.timeText), ['12:00:03', '12:00:02', '12:00:01'])
})

test('other points row only appears when either side scored some', () => {
  const withoutOther = resolveChasingScoreView(breakdown)
  assert.equal(withoutOther.rows.some((row) => row.key === 'other_points'), false)

  const view = resolveChasingScoreView({
    ...breakdown,
    player2: { ...breakdown.player2, other_points: 3 }
  })
  assert.deepEqual(view.rows.at(-1), { key: 'other_points', label: '其他得分', left: '0', right: '3' })

  const otherEvent = resolveChasingScoreView({
    ...breakdown,
    events: [{ seq: 4, kind: 'other', scorer: 1, points: 2, player1_score: 18, player2_score: 13, created_at: '2026-11-10 12:00:04' }]
  })
  assert.equal(otherEvent.events[0].desc, '我 得分')
})

test('unavailable breakdown degrades to the total score only', () => {
  const view = resolveChasingScoreView({ ...breakdown, available: false })
  assert.equal(view.available, false)
  assert.deepEqual(view.rows, [{ key: 'score', label: '总分', left: '16', right: '13' }])
  assert.deepEqual(view.events, [])
  assert.equal(view.notice, '记分明细暂不可用，仅显示总分')
})

test('truncated events say only the latest 200 are shown', () => {
  const view = resolveChasingScoreView({ ...breakdown, events_truncated: true })
  assert.equal(view.eventsTruncated, true)
  assert.equal(view.notice, '仅显示最近 200 条记分')
})

test('dirty payloads do not throw and fall back to zero values', () => {
  const view = resolveChasingScoreView({
    available: true,
    player1: { score: 'oops', normal_wins: null, foul_count: Infinity },
    events: [{ seq: 'x', kind: 'mystery', scorer: 9, points: '4', created_at: 42 }, null]
  })
  assert.equal(view.leftScore, 0)
  assert.equal(view.rows[0].right, '0')
  assert.equal(view.rows.find((row) => row.key === 'foul_count').left, '0')
  assert.equal(view.events[0].kindLabel, '其他')
  assert.equal(view.events[0].side, 'left')
  assert.equal(view.events[0].pointsText, '+4')
  assert.equal(view.events[0].timeText, '')
  assert.equal(view.events[1].seq, 0, 'non-object events are tolerated')
})
