import test from 'node:test'
import assert from 'node:assert/strict'

import {
  PERFORMANCE_MONTH_FOOTNOTE,
  buildMonthChart,
  formatSampleCount,
  resolveLockedView,
  resolveNotableMatchView,
  resolvePerformanceVersion,
  resolveRatingView,
  resolveSummaryView
} from '../utils/member-performance-view.js'

test('rating view shows exact values, at-least/at-most bounds and an empty state', () => {
  assert.deepEqual(resolveRatingView({ rating: 1678 }), { available: true, value: '1678', qualifier: '', text: '1678' })
  assert.deepEqual(resolveRatingView({ rating: 2415, bound: 'at_least' }), { available: true, value: '2415', qualifier: '至少', text: '至少2415' })
  assert.deepEqual(resolveRatingView({ rating: 0, bound: 'at_most' }), { available: true, value: '0', qualifier: '至多', text: '至多0' })
  for (const empty of [null, undefined, {}, { rating: 'x' }]) {
    assert.deepEqual(resolveRatingView(empty), { available: false, value: '', qualifier: '', text: '—' })
  }
})

test('sample counts are capped at the server limit and never negative', () => {
  assert.equal(formatSampleCount(0), '0')
  assert.equal(formatSampleCount(37), '37')
  assert.equal(formatSampleCount(200), '200+')
  assert.equal(formatSampleCount(999), '200+')
  assert.equal(formatSampleCount(-3), '0')
  assert.equal(formatSampleCount('abc'), '0')
})

test('summary view explains actual versus expected wins with a signed difference', () => {
  const better = resolveSummaryView({ wins: 14, expected_wins: 11.2, wins_diff: 2.8, rating: { rating: 1720 } }, 20)
  assert.equal(better.tone, 'up')
  assert.equal(better.diffText, '+2.8')
  assert.equal(better.expectedText, '11.2')
  assert.equal(better.wins, 14)
  assert.equal(better.rating.text, '1720')
  assert.match(better.headline, /更好/)

  const worse = resolveSummaryView({ wins: 8, expected_wins: 9.5, wins_diff: -1.5, rating: null }, 12)
  assert.equal(worse.tone, 'down')
  assert.equal(worse.diffText, '-1.5')
  assert.equal(worse.rating.available, false)
  assert.match(worse.ratingHint, /样本不足 5 场/)

  const flat = resolveSummaryView({ wins: 9, expected_wins: 9.2, wins_diff: -0.2, rating: { rating: 1500 } }, 20)
  assert.equal(flat.tone, 'flat')
  assert.equal(flat.diffText, '-0.2')
  // 四舍五入后为 0 时不带正负号。
  assert.equal(resolveSummaryView({ wins: 9, expected_wins: 9.004, wins_diff: -0.004, rating: null }, 9).diffText, '0.0')
  assert.equal(resolveSummaryView(null, 3).available, false)
})

test('month chart scales bars and end-of-month dots on one shared axis', () => {
  const chart = buildMonthChart([
    { month: '2026-05', sample_count: 0, rating: null, end_rank_score: 0 },
    { month: '2026-06', sample_count: 8, rating: { rating: 1500 }, end_rank_score: 1480 },
    { month: '2026-07', sample_count: 6, rating: { rating: 1700 }, end_rank_score: 1620 },
    { month: '2026-08', sample_count: 2, rating: null, end_rank_score: 1630 },
    { month: '2026-09', sample_count: 5, rating: { rating: 1900, bound: 'at_least' }, end_rank_score: 1700, incomplete: false },
    { month: '2026-10', sample_count: 4, rating: { rating: 1600 }, end_rank_score: 1710 }
  ])
  assert.equal(chart.hasData, true)
  assert.equal(chart.bars.length, 6)
  assert.deepEqual(chart.bars.map((bar) => bar.label), ['5月', '6月', '7月', '8月', '9月', '10月'])
  assert.equal(chart.bars.at(-1).isCurrent, true)
  assert.equal(chart.bars.filter((bar) => bar.isCurrent).length, 1)

  const [may, june, july, august, september, october] = chart.bars
  assert.equal(may.status, 'insufficient')
  assert.equal(may.heightPercent, 0)
  assert.equal(may.dotPercent, null, 'no rank score means no dot')
  assert.equal(august.note, '样本不足')
  assert.equal(august.dotPercent > 0, true, 'a month without a rating can still show its end rank score')
  assert.equal(september.qualifier, '至少')
  // 同一刻度：1700 的柱比 1500 的高，1900 最高。
  assert.ok(june.heightPercent < july.heightPercent && july.heightPercent < september.heightPercent)
  assert.ok(october.heightPercent < july.heightPercent && october.heightPercent > june.heightPercent)
  for (const bar of chart.bars) {
    assert.ok(bar.heightPercent >= 0 && bar.heightPercent <= 100)
    if (bar.dotPercent !== null) assert.ok(bar.dotPercent >= 0 && bar.dotPercent <= 100)
  }
  assert.equal(chart.scaleMin % 100, 0)
  assert.equal(chart.scaleMax % 100, 0)
})

test('a month truncated by the read cap is marked incomplete and never drawn as a bar', () => {
  const chart = buildMonthChart([
    { month: '2026-09', sample_count: 12, rating: null, end_rank_score: 1500, incomplete: true },
    // 即使误带了表现分，也不能把残缺数据画成水平。
    { month: '2026-10', sample_count: 12, rating: { rating: 1800 }, end_rank_score: 1520, incomplete: true }
  ])
  assert.deepEqual(chart.bars.map((bar) => bar.status), ['incomplete', 'incomplete'])
  assert.deepEqual(chart.bars.map((bar) => bar.heightPercent), [0, 0])
  assert.deepEqual(chart.bars.map((bar) => bar.note), ['数据不全', '数据不全'])
})

test('month chart reports no data when there is nothing to draw', () => {
  assert.equal(buildMonthChart([]).hasData, false)
  assert.equal(buildMonthChart(undefined).hasData, false)
  assert.equal(buildMonthChart([{ month: '2026-10', sample_count: 1, rating: null, end_rank_score: 0 }]).hasData, false)
})

test('a single flat series still gets a usable scale', () => {
  const chart = buildMonthChart([{ month: '2026-10', sample_count: 6, rating: { rating: 1500 }, end_rank_score: 1500 }])
  assert.ok(chart.scaleMax - chart.scaleMin >= 200)
  assert.ok(chart.bars[0].heightPercent > 0 && chart.bars[0].heightPercent < 100)
})

test('notable matches describe the opponent and the pre-match expectation', () => {
  const best = resolveNotableMatchView({ match_id: 9, opponent_name: '老王', opponent_avatar: 'a.png', opponent_before_score: 1800, expected_win_rate: 23.5, played_at: '2026-10-01 20:00:00' }, 'best_win')
  assert.equal(best.title, '最佳胜利')
  assert.equal(best.headline, '击败 老王')
  assert.match(best.detail, /1800/)
  assert.match(best.detail, /只有 23\.5%/)
  assert.equal(best.matchId, 9)

  const loss = resolveNotableMatchView({ match_id: 10, opponent_name: '', opponent_before_score: 1300, expected_win_rate: 80 }, 'worst_loss')
  assert.equal(loss.title, '爆冷失利')
  assert.equal(loss.headline, '负于 对手')
  assert.match(loss.detail, /有 80%/)
  assert.equal(resolveNotableMatchView(null, 'best_win'), null)
})

test('locked view shows the server sample count and month preview, and the purchase entry follows the compliance switch', () => {
  const response = { locked: true, sample_count: 200, month_preview: { month: '2026-10', sample_count: 6, rating: { rating: 1678 } } }
  const hidden = resolveLockedView(response, { canShowPurchase: false })
  assert.equal(hidden.showPurchase, false)
  assert.equal(hidden.sampleText, '200+')
  assert.equal(hidden.preview.available, true)
  assert.equal(hidden.preview.ratingText, '1678')
  assert.equal(hidden.preview.monthText, '10月')
  assert.match(hidden.preview.hint, /6 场/)
  assert.equal(hidden.features.length, 4)
  assert.equal(resolveLockedView(response, { canShowPurchase: true }).showPurchase, true)
  assert.equal(resolveLockedView(response).showPurchase, false, 'the default is no purchase entry')

  const sparse = resolveLockedView({ locked: true, sample_count: 2, month_preview: { month: '2026-10', sample_count: 2, rating: null } })
  assert.equal(sparse.preview.available, false)
  assert.equal(sparse.preview.ratingText, '—')
  assert.match(sparse.preview.hint, /满 3 场/)

  const bounded = resolveLockedView({ locked: true, sample_count: 5, month_preview: { month: '2026-10', sample_count: 3, rating: { rating: 2300, bound: 'at_least' } } })
  assert.equal(bounded.preview.qualifier, '至少')

  const none = resolveLockedView({ locked: true, sample_count: 0 })
  assert.equal(none.hasSamples, false)
  assert.equal(none.preview.available, false)
  assert.equal(none.preview.monthText, '本月')
})

test('the locked view never carries analysis numbers beyond what the server returned', () => {
  const view = resolveLockedView({ locked: true, sample_count: 12, summary: { wins: 9, expected_wins: 7, wins_diff: 2, rating: { rating: 1800 } }, best_win: { match_id: 1 }, months: [{ month: '2026-10' }] })
  assert.deepEqual(Object.keys(view).sort(), ['features', 'hasSamples', 'preview', 'sampleText', 'showPurchase'])
  assert.doesNotMatch(JSON.stringify(view), /1800|expected|best_win/)
})

test('invalidation version combines the rank and member scopes', () => {
  assert.equal(resolvePerformanceVersion({ rankVersion: 2, memberVersion: 1 }), '2:1')
  assert.equal(resolvePerformanceVersion(), '0:0')
  assert.notEqual(resolvePerformanceVersion({ rankVersion: 1, memberVersion: 0 }), resolvePerformanceVersion({ rankVersion: 0, memberVersion: 1 }))
})

test('the chart footnote explains the minimum samples per month', () => {
  assert.match(PERFORMANCE_MONTH_FOOTNOTE, /满 3 场/)
})
