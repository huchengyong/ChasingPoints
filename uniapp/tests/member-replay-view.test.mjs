import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildFrameSequence,
  buildTrendColumns,
  formatFrameDuration,
  REPLAY_MAX_TREND_COLUMNS,
  resolveReplayView
} from '../utils/member-replay-view.js'

const snookerResponse = {
  success: true,
  match_id: 7,
  game_type: 1,
  locked: false,
  frame_count: 2,
  summary: { my_frame_wins: 1, opponent_frame_wins: 1, my_max_run: 68, opponent_max_run: 41 },
  frames: [
    {
      frame_no: 1, winner: 1, my_score: 72, opponent_score: 40, duration_seconds: 1260, max_deficit: 12, key_tags: ['comeback'],
      my_max_run: 68, opponent_max_run: 20, my_foul_count: 1, opponent_foul_count: 0, my_foul_points: 4, opponent_foul_points: 0,
      trend: [{ step: 0, diff: 0 }, { step: 1, diff: -12 }, { step: 2, diff: 6 }, { step: 3, diff: 32 }]
    },
    { frame_no: 2, winner: 2, my_score: 30, opponent_score: 61, duration_seconds: 45, max_deficit: 0, key_tags: ['decider'], my_max_run: 18, opponent_max_run: 41, trend: [] }
  ]
}

test('locked replay carries only the frame count and the purchase entry follows the compliance switch', () => {
  const response = { locked: true, game_type: 3, frame_count: 7, frames: [{ frame_no: 1, winner: 1 }], summary: { my_frame_wins: 4, opponent_frame_wins: 3 } }
  const view = resolveReplayView(response, { canShowPurchase: false })
  assert.equal(view.locked, true)
  assert.equal(view.frameCount, 7)
  assert.equal(view.showPurchase, false)
  assert.equal(resolveReplayView(response, { canShowPurchase: true }).showPurchase, true)
  assert.equal(resolveReplayView(response).showPurchase, false)
  // 锁定时即使响应里误带了明细，也不会进入展示模型。
  assert.equal(view.frames, undefined)
  assert.equal(view.summaryRows, undefined)
  assert.doesNotMatch(JSON.stringify(view), /4 : 3|frame_no/)
})

test('snooker replay shows frame scores, single breaks, fouls, tags and the comeback description', () => {
  const view = resolveReplayView(snookerResponse)
  assert.equal(view.wholeMatch, false)
  assert.equal(view.title, '斯诺克复盘')
  assert.equal(view.scoreText, '1 : 1')
  assert.deepEqual(view.summaryRows.map((row) => [row.label, row.mine, row.opponent]), [['赢下局数', '1', '1'], ['单杆最高', '68', '41']])

  const [first, second] = view.frames
  assert.equal(first.scoreText, '72 : 40')
  assert.equal(first.winnerText, '我赢')
  assert.equal(first.durationText, '21分')
  assert.deepEqual(first.tags.map((tag) => tag.label), ['逆转局'])
  assert.equal(first.deficitText, '我曾落后 12 分后赢下')
  assert.deepEqual(first.details.map((item) => item.label), ['单杆最高', '犯规（次 / 罚分）'])
  assert.deepEqual(first.details[1], { label: '犯规（次 / 罚分）', mine: '1 / 4', opponent: '0 / 0' })
  assert.equal(first.trend.hasData, true)

  assert.equal(second.winnerText, '对手赢')
  assert.equal(second.durationText, '45秒')
  assert.equal(second.deficitText, '', 'no deficit text when the winner never trailed')
  assert.equal(second.trend.hasData, false)
  assert.deepEqual(view.keyFrames.map((frame) => frame.frameNo), [1, 2])
  assert.equal(view.notice, '')
  assert.equal(view.hasDetail, true)
})

const nineBallBreakdown = {
  available: true,
  player1: { score: 21, normal_wins: 3, small_golds: 1, big_golds: 1, foul_count: 2, foul_points_given: 2, other_points: 0 },
  player2: { score: 15, normal_wins: 1, small_golds: 2, big_golds: 0, foul_count: 1, foul_points_given: 1, other_points: 0 },
  events: [],
  events_truncated: false
}

const nineBallResponse = {
  success: true,
  match_id: 21,
  game_type: 2,
  viewer_actor: 1,
  locked: false,
  frame_count: 3,
  summary: { my_frame_wins: 0, opponent_frame_wins: 0, my_max_run: 12, opponent_max_run: 5, score_breakdown: nineBallBreakdown },
  frames: [{
    frame_no: 1,
    winner: 1,
    my_score: 21,
    opponent_score: 15,
    duration_seconds: 1500,
    max_deficit: 4,
    my_max_run: 12,
    opponent_max_run: 5,
    my_foul_count: 2,
    opponent_foul_count: 1,
    my_foul_points: 2,
    opponent_foul_points: 1,
    trend: [{ step: 0, diff: 0 }, { step: 1, diff: -4 }, { step: 2, diff: 3 }, { step: 3, diff: 6 }]
  }]
}

test('nine-ball chase replays the whole match instead of per-frame results', () => {
  const view = resolveReplayView(nineBallResponse)
  assert.equal(view.wholeMatch, true)
  assert.equal(view.scoreText, '21 : 15')
  assert.deepEqual(view.summaryRows.map((row) => [row.label, row.mine, row.opponent]), [
    ['总分', '21', '15'],
    ['普胜', '3', '1'],
    ['小金', '1', '2'],
    ['大金', '1', '0'],
    ['犯规', '2', '1'],
    ['犯规送分', '2', '1'],
    ['最大连续得分', '12', '5']
  ])
  assert.equal(view.summaryRows.some((row) => row.label.includes('赢下局数')), false)
  assert.deepEqual(view.winTypeRows, [])
  assert.deepEqual(view.sequence, [])
  assert.deepEqual(view.keyFrames, [])

  assert.equal(view.frames.length, 1)
  const [frame] = view.frames
  assert.equal(frame.title, '整场走势')
  assert.equal(frame.winTypeText, '')
  assert.deepEqual(frame.tags, [])
  assert.equal(frame.isKey, false)
  assert.equal(frame.winnerText, '我赢')
  assert.equal(frame.scoreText, '21 : 15')
  assert.equal(frame.durationText, '25分')
  assert.equal(frame.deficitText, '我曾落后 4 分后赢下')
  assert.equal(frame.trend.hasData, true)
  assert.equal(view.notice, '')
  assert.equal(view.hasDetail, true)
  assert.equal(view.empty, false)
})

test('nine-ball chase swaps the comparison for the player 2 perspective', () => {
  const view = resolveReplayView({
    ...nineBallResponse,
    viewer_actor: 2,
    frames: [{ ...nineBallResponse.frames[0], winner: 2, my_score: 15, opponent_score: 21 }]
  })
  assert.equal(view.scoreText, '15 : 21')
  assert.deepEqual(view.summaryRows.slice(0, 4).map((row) => [row.label, row.mine, row.opponent]), [
    ['总分', '15', '21'],
    ['普胜', '1', '3'],
    ['小金', '2', '1'],
    ['大金', '0', '1']
  ])
  assert.equal(view.frames[0].winnerText, '对手赢')
  // 连续得分只在整场数据里出现一次，片段不重复。
  assert.deepEqual(view.frames[0].details, [])
  assert.equal(view.summaryRows.at(-1).label, '最大连续得分')
})

test('nine-ball chase says why only the total score is shown', () => {
  const frameOnly = resolveReplayView({ ...nineBallResponse, frame_only: true })
  assert.equal(frameOnly.notice, '这场对局的记分记录不完整，只显示总分。')
  assert.equal(frameOnly.hasDetail, false)

  const truncated = resolveReplayView({ ...nineBallResponse, detail_truncated: true, frame_only: true })
  assert.equal(truncated.notice, '这场对局的记录太长，只显示总分。')
  assert.equal(truncated.hasDetail, false)
})

test('locked nine-ball chase lists whole-match features without frame counts', () => {
  const view = resolveReplayView({ locked: true, game_type: 2, frame_count: 7 }, { canShowPurchase: true })
  assert.equal(view.wholeMatch, true)
  assert.equal(view.frameCount, 7)
  assert.equal(view.showPurchase, true)
  assert.deepEqual(view.features, [
    '整场走势：谁领先、什么时候被追上',
    '双方普胜、小金、大金次数对照',
    '犯规次数与送分',
    '最大连续得分'
  ])
  assert.doesNotMatch(JSON.stringify(view), /第 \d+ 局|局数/)
})

test('Chinese eight-ball shows the win sequence, longest streak, win types and no score per frame', () => {
  const view = resolveReplayView({
    game_type: 3,
    frame_count: 8,
    summary: {
      my_frame_wins: 5, opponent_frame_wins: 3, my_longest_streak: 4, opponent_longest_streak: 2,
      win_type_stats: [{ win_type: 'normal', label: '普胜', my_count: 3, opponent_count: 3 }, { win_type: 'break_clear', label: '炸清', my_count: 2, opponent_count: 0 }]
    },
    frames: [
      { frame_no: 1, winner: 1, my_score: 1, opponent_score: 0, win_type: 'normal' },
      { frame_no: 2, winner: 2, my_score: 0, opponent_score: 1, win_type: 'break_clear' },
      { frame_no: 3, winner: 1, my_score: 1, opponent_score: 0, win_type: 'normal', key_tags: ['decider'] }
    ]
  })
  assert.deepEqual(view.sequence.map((item) => item.winner), ['me', 'opponent', 'me'])
  assert.equal(view.wholeMatch, false)
  assert.deepEqual(view.summaryRows.map((row) => row.label), ['赢下局数', '最长连胜（局）'])
  assert.deepEqual(view.winTypeRows, [{ label: '普胜', mine: '3', opponent: '3' }, { label: '炸清', mine: '2', opponent: '0' }])
  assert.equal(view.frames[0].scoreText, '')
  assert.equal(view.frames[1].winTypeText, '炸清')
  assert.deepEqual(view.keyFrames.map((frame) => frame.frameNo), [3])
})

test('frame-only and truncated replays say why there is no per-frame detail', () => {
  const frameOnly = resolveReplayView({ game_type: 1, frame_count: 1, frame_only: true, summary: { my_frame_wins: 1, opponent_frame_wins: 0 }, frames: [{ frame_no: 1, winner: 1, my_score: 60, opponent_score: 20 }] })
  assert.match(frameOnly.notice, /没有逐杆记录/)
  assert.equal(frameOnly.hasDetail, false)
  const truncated = resolveReplayView({ game_type: 2, frame_count: 1, detail_truncated: true, frame_only: true, frames: [{ frame_no: 1, winner: 2 }] })
  assert.match(truncated.notice, /记录太长/)
  assert.equal(truncated.hasDetail, false)
  const none = resolveReplayView({ game_type: 3, frame_count: 0, frames: [] })
  assert.equal(none.empty, true)
})

test('trend columns are centered on zero with heights normalized to the largest gap', () => {
  const trend = buildTrendColumns([{ step: 0, diff: 0 }, { step: 1, diff: 20 }, { step: 2, diff: -10 }, { step: 3, diff: 5 }])
  assert.equal(trend.hasData, true)
  assert.equal(trend.maxAbs, 20)
  assert.deepEqual(trend.columns.map((column) => column.direction), ['flat', 'up', 'down', 'up'])
  assert.deepEqual(trend.columns.map((column) => column.heightPercent), [0, 100, 50, 25])

  // 单点或全零没有可画的走势。
  assert.equal(buildTrendColumns([{ step: 0, diff: 0 }]).hasData, false)
  assert.equal(buildTrendColumns([{ step: 0, diff: 0 }, { step: 1, diff: 0 }]).hasData, false)
  assert.equal(buildTrendColumns(undefined).hasData, false)

  const many = buildTrendColumns(Array.from({ length: 120 }, (_, index) => ({ step: index, diff: index - 60 })))
  assert.equal(many.columns.length, REPLAY_MAX_TREND_COLUMNS)
})

test('frame sequence and duration helpers are defensive about missing values', () => {
  assert.deepEqual(buildFrameSequence([{ frame_no: 1, winner: 1 }, { frame_no: 2, winner: 0 }, {}]), [
    { frameNo: 1, winner: 'me' }, { frameNo: 2, winner: 'none' }, { frameNo: 0, winner: 'none' }
  ])
  assert.deepEqual([0, undefined, 45, 60, 61, 1260].map(formatFrameDuration), ['', '', '45秒', '1分', '1分1秒', '21分'])
})
