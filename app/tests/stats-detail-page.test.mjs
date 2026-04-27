import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
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

test('stats detail page exposes dark-mode styles for its main surfaces', () => {
  assert.match(statsDetailSource, /class="stats-page" :class="\{ 'dark-mode': isDarkMode \}"/)
  assert.match(statsDetailSource, /\.stats-page\s*\{[\s\S]*&\.dark-mode\s*\{[\s\S]*background:\s*#141109;/)
  assert.match(statsDetailSource, /&\.dark-mode\s*\{[\s\S]*\.section\s*\{[\s\S]*background:\s*#1e180d;/)
  assert.match(statsDetailSource, /&\.dark-mode\s*\{[\s\S]*\.section-title,[\s\S]*color:\s*#fff7e1;/)
})
