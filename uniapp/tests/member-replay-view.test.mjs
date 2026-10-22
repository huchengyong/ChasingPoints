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

test('nine-ball chase uses the longest-run wording', () => {
  const view = resolveReplayView({ game_type: 2, frame_count: 1, summary: { my_frame_wins: 1, opponent_frame_wins: 0, my_max_run: 9, opponent_max_run: 3 }, frames: [{ frame_no: 1, winner: 1, my_score: 9, opponent_score: 3, my_max_run: 9, opponent_max_run: 3 }] })
  assert.equal(view.summaryRows[1].label, '最大连续得分')
  assert.equal(view.frames[0].details[0].label, '最大连续得分')
  assert.equal(view.frames[0].scoreText, '9 : 3')
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
