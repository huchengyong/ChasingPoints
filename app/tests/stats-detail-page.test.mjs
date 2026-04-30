import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildRankDeltaChartViewModel,
  buildStatsTrendCalendarViewModel,
  normalizeDurationStats,
  normalizeOpponentStrengthStats,
  resolveStatsDetailLoadingMode,
  shouldApplyStatsDetailResponse
} from '../utils/stats-detail.js'

const statsDetailSource = readFileSync(
  new URL('../subPages/user/statsDetail.vue', import.meta.url),
  'utf8'
)

test('stats detail page redirects guests straight to login instead of rendering a login-required placeholder', () => {
  assert.doesNotMatch(statsDetailSource, /login-required/)
  assert.match(statsDetailSource, /const token = uni\.getStorageSync\('token'\)/)
  assert.match(statsDetailSource, /if \(!token\) \{\s*goLogin\(\)\s*return/s)
  assert.match(statsDetailSource, /const goLogin = \(\) => \{\s*uni\.navigateTo\(\{ url: '\/pages\/login\/login' \}\)/s)
})

test('stats detail page only loads stats after passing the login guard', () => {
  assert.match(statsDetailSource, /onLoad\(\(options\) => \{[\s\S]*currentGame\.value = gameTypeKeyMap\[gameType\][\s\S]*\}\)\s*\n\s*\n\s*onShow/s)
  assert.match(statsDetailSource, /onShow\([\s\S]*if \(!token\) \{\s*goLogin\(\)\s*return[\s\S]*loadAllStats\(\)/s)
  assert.doesNotMatch(statsDetailSource, /watch\(currentGame/)
})

test('stats detail loading mode uses full-page loading only before first render', () => {
  assert.equal(resolveStatsDetailLoadingMode({
    hasLoadedOnce: false,
    isFetching: true
  }), 'initial')

  assert.equal(resolveStatsDetailLoadingMode({
    hasLoadedOnce: true,
    isFetching: true
  }), 'refreshing')

  assert.equal(resolveStatsDetailLoadingMode({
    hasLoadedOnce: true,
    isFetching: false
  }), 'idle')
})

test('stats detail ignores stale game switch responses', () => {
  assert.equal(shouldApplyStatsDetailResponse({
    requestId: 6,
    latestRequestId: 7
  }), false)

  assert.equal(shouldApplyStatsDetailResponse({
    requestId: 7,
    latestRequestId: 7
  }), true)
})

test('stats detail trend calendar groups wins and losses by day in the selected month', () => {
  const calendar = buildStatsTrendCalendarViewModel([
    { match_id: 1, date: '2026-04-03', result: 1 },
    { match_id: 2, date: '2026-04-03', result: 1 },
    { match_id: 3, date: '2026-04-03', result: 2 },
    { match_id: 4, date: '2026-05-01', result: 1 }
  ], { monthKey: '2026-04' })

  const day = calendar.cells.find(item => item.dateKey === '2026-04-03')
  assert.equal(calendar.title, '2026年4月')
  assert.equal(calendar.summary.matchCount, 3)
  assert.equal(calendar.summary.winCount, 2)
  assert.equal(calendar.summary.lossCount, 1)
  assert.equal(day.winCount, 2)
  assert.equal(day.lossCount, 1)
  assert.equal(day.hasMatches, true)
})

test('stats detail rank chart aggregates daily rank score deltas for the selected month', () => {
  const chart = buildRankDeltaChartViewModel([
    { date: '2026-04-03', rank_score: 1030 },
    { date: '2026-04-03', rank_score: 1010 },
    { date: '2026-04-02', rank_score: 1020 },
    { date: '2026-03-31', rank_score: 1000 }
  ], { monthKey: '2026-04' })

  const aprilSecond = chart.days.find(item => item.dateKey === '2026-04-02')
  const aprilThird = chart.days.find(item => item.dateKey === '2026-04-03')

  assert.equal(chart.summary.netDelta, 30)
  assert.equal(chart.summary.upDays, 2)
  assert.equal(chart.summary.downDays, 0)
  assert.equal(aprilSecond.delta, 20)
  assert.equal(aprilThird.delta, 10)
  assert.equal(aprilThird.label, '3日')
  assert.equal(aprilThird.showTick, false)
})

test('stats detail renders recent trend as a monthly calendar and rank changes as a daily bar chart', () => {
  assert.match(statsDetailSource, /class="stats-calendar"/)
  assert.match(statsDetailSource, /calendar-result-row/)
  assert.match(statsDetailSource, /×\{\{ cell\.winCount \}\}/)
  assert.match(statsDetailSource, /class="rank-chart"/)
  assert.match(statsDetailSource, /rankDeltaChart\.days/)
  assert.match(statsDetailSource, /day\.showTick/)
})

test('stats detail normalizes backend duration stats into display fields', () => {
  assert.deepEqual(normalizeDurationStats({
    stats: {
      average_seconds: 125,
      fastest_seconds: 40,
      longest_seconds: 360,
      total_matches: 3
    }
  }), {
    average: 125,
    fastest: 40,
    longest: 360,
    total: 3
  })
})

test('stats detail normalizes backend opponent strength rank ranges for display', () => {
  assert.deepEqual(normalizeOpponentStrengthStats({
    list: [{
      rank_range: '中级(1001-2000)',
      matches: 25,
      wins: 22,
      win_rate: 88
    }]
  }), [{
    tier_name: '中级(1001-2000)',
    matches: 25,
    wins: 22,
    win_rate: 88
  }])
})

test('stats detail game switch keeps content mounted and shows tab-local loading', () => {
  assert.match(statsDetailSource, /v-if="isInitialLoading"/)
  assert.match(statsDetailSource, /pending: isStatsRefreshing && currentGame === tab\.key/)
  assert.match(statsDetailSource, /class="tab-loading-icon"/)
})

test('stats detail formats percent values from backend percent numbers', () => {
  assert.match(statsDetailSource, /const formatPercent = \(value\) => \{/)
  assert.match(statsDetailSource, /{{ formatPercent\(currentGameStats\.win_rate\) }}%/)
  assert.match(statsDetailSource, /:style="\{ width: formatPercent\(currentGameStats\?\.win_rate\) \+ '%' \}"/)
  assert.match(statsDetailSource, /{{ formatPercent\(tier\.win_rate\) }}%/)
})

test('stats detail requests opponent strength for the selected game type', () => {
  assert.match(statsDetailSource, /getOpponentStrengthAnalysis\(\{ game_type: gameType \}\)/)
})

test('stats detail page exposes dark-mode styles for its main surfaces', () => {
  assert.match(statsDetailSource, /class="stats-page" :class="\{ 'dark-mode': isDarkMode \}"/)
  assert.match(statsDetailSource, /\.stats-page\s*\{[\s\S]*&\.dark-mode\s*\{[\s\S]*background:\s*#141109;/)
  assert.match(statsDetailSource, /&\.dark-mode\s*\{[\s\S]*\.section\s*\{[\s\S]*background:\s*#1e180d;/)
  assert.match(statsDetailSource, /&\.dark-mode\s*\{[\s\S]*\.section-title,[\s\S]*color:\s*#fff7e1;/)
})
