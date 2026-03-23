import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildFeaturedPostTarget,
  pickFeaturedTournament
} from '../utils/home-index.js'
import { shouldFetchAuthState } from '../utils/auth-guards.js'
import { shouldShowMatchPageLoading } from '../utils/match-page.js'
import { shouldFetchUnreadCount } from '../utils/notification.js'
import {
  resolveGuestHeroCopy,
  resolvePrimaryAction,
  resolveSectionTitles,
  resolveStatusCardContent,
  resolveUserHomepageMode
} from '../utils/user-homepage.js'

test('buildFeaturedPostTarget prefers concrete match share route for report posts', () => {
  const target = buildFeaturedPostTarget({
    post_type: 1,
    match_id: 42,
    nickname: '球友'
  })

  assert.deepEqual(target, {
    type: 'navigate',
    url: '/subPages/match/shareResult?match_id=42',
    ctaText: '查看战报'
  })
})

test('buildFeaturedPostTarget falls back to community for report posts without match id', () => {
  const target = buildFeaturedPostTarget({
    post_type: 1,
    opponent_id: 99,
    nickname: '球友'
  })

  assert.deepEqual(target, {
    type: 'community',
    url: '/pages/social/index',
    ctaText: '去社区查看更多'
  })
})

test('pickFeaturedTournament prefers active upcoming tournaments over ended ones', () => {
  const now = '2026-03-11T12:00:00+08:00'
  const picked = pickFeaturedTournament([
    { id: 1, status: 2, start_time: '2026-03-09T20:00:00+08:00' },
    { id: 2, status: 0, start_time: '2026-03-12T20:00:00+08:00' },
    { id: 3, status: 1, start_time: '2026-03-11T20:00:00+08:00' }
  ], now)

  assert.equal(picked.id, 3)
})

test('pickFeaturedTournament falls back to nearest non-ended item when status is mixed', () => {
  const now = '2026-03-11T12:00:00+08:00'
  const picked = pickFeaturedTournament([
    { id: 10, status: 0, start_time: '2026-03-15T20:00:00+08:00' },
    { id: 11, status: 0, start_time: '2026-03-12T09:00:00+08:00' }
  ], now)

  assert.equal(picked.id, 11)
})

test('shouldFetchUnreadCount skips unread count requests for guests', () => {
  assert.equal(shouldFetchUnreadCount(false), false)
})

test('shouldFetchUnreadCount allows unread count requests after login', () => {
  assert.equal(shouldFetchUnreadCount(true), true)
})

test('shouldFetchAuthState skips auth-only state requests for guests', () => {
  assert.equal(shouldFetchAuthState(false), false)
})

test('shouldFetchAuthState allows auth-only state requests after login', () => {
  assert.equal(shouldFetchAuthState(true), true)
})

test('shouldShowMatchPageLoading hides full-page loading during pull refresh', () => {
  assert.equal(shouldShowMatchPageLoading(true, true), false)
})

test('shouldShowMatchPageLoading shows full-page loading for normal initial load', () => {
  assert.equal(shouldShowMatchPageLoading(true, false), true)
})

test('resolveUserHomepageMode returns guest for anonymous users', () => {
  assert.equal(resolveUserHomepageMode({
    isLoggedIn: false,
    hasCurrentMatch: false,
    hasRecentMatch: false
  }), 'guest')
})

test('resolveUserHomepageMode returns ongoing when current match exists', () => {
  assert.equal(resolveUserHomepageMode({
    isLoggedIn: true,
    hasCurrentMatch: true,
    hasRecentMatch: true
  }), 'ongoing')
})

test('resolvePrimaryAction prefers continue match for ongoing state', () => {
  assert.equal(resolvePrimaryAction('ongoing'), 'continue')
})

test('resolveStatusCardContent returns continue action for current match', () => {
  const result = resolveStatusCardContent({
    mode: 'ongoing',
    currentMatch: {
      opponent_name: '张三',
      my_score: 3,
      opponent_score: 2,
      duration_seconds: 768
    }
  })

  assert.equal(result.action, 'continue')
  assert.match(result.title, /继续/)
  assert.match(result.description, /3 : 2/)
})

test('resolveStatusCardContent uses encouraging copy for idle users', () => {
  const result = resolveStatusCardContent({
    mode: 'idle',
    currentMatch: null
  })

  assert.match(result.description, /找回节奏|找回手感/)
})

test('resolveGuestHeroCopy avoids login-wall phrasing', () => {
  const result = resolveGuestHeroCopy()

  assert.equal(result.title, '登录后，解锁你的个人竞技主页')
  assert.doesNotMatch(result.description, /请先完成登录/)
})

test('resolveSectionTitles returns competitive naming', () => {
  const result = resolveSectionTitles()

  assert.equal(result.stats, '竞技概览')
  assert.equal(result.quickActions, '竞技工具')
  assert.equal(result.settings, '设置与支持')
})
