import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildFeaturedPostTarget,
  formatEventNewsTime,
  getEventNewsStatusText
} from '../utils/home-index.js'
import { shouldFetchAuthState } from '../utils/auth-guards.js'
import { shouldShowMatchPageLoading } from '../utils/match-page.js'
import { shouldFetchUnreadCount } from '../utils/notification.js'
import {
  resolveHighestRankDisplay,
  resolveGuestHeroCopy,
  resolvePrimaryAction,
  resolveStatusActionVisibility,
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

test('getEventNewsStatusText maps status codes to readable labels', () => {
  assert.equal(getEventNewsStatusText(0), '即将开始')
  assert.equal(getEventNewsStatusText(1), '进行中')
  assert.equal(getEventNewsStatusText(2), '已结束')
  assert.equal(getEventNewsStatusText(3), '已取消')
})

test('formatEventNewsTime falls back gracefully for missing or invalid time values', () => {
  assert.equal(formatEventNewsTime('', '2026-03-11T12:00:00+08:00'), '时间待定')
  assert.equal(formatEventNewsTime('not-a-date', '2026-03-11T12:00:00+08:00'), '时间待定')
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

test('resolvePrimaryAction routes guest users to the login entry flow', () => {
  assert.equal(resolvePrimaryAction('guest'), 'login')
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
  assert.equal(result.primaryActionText, '登录/注册')
  assert.equal(result.secondaryActionText, '发起PK')
})

test('resolveStatusCardContent aligns guest actions with PK entry intents', () => {
  const result = resolveStatusCardContent({
    mode: 'guest'
  })

  assert.equal(result.action, 'login')
  assert.equal(result.actionText, '登录/注册')
  assert.equal(result.secondaryAction, 'start_pk')
  assert.equal(result.secondaryActionText, '发起PK')
})

test('resolveStatusCardContent removes recent-history CTA from active homepage card', () => {
  const result = resolveStatusCardContent({
    mode: 'active',
    recentMatch: {
      title: '最近手感不错，继续保持',
      description: '累计 12 场对局 · 胜率 67% · 最高连胜 4'
    }
  })

  assert.equal(result.action, 'recent')
  assert.equal(result.actionText, '')
  assert.equal(result.secondaryAction, 'start')
  assert.equal(result.secondaryActionText, '再来一场')
})

test('resolveStatusActionVisibility hides status action row for logged-in active homepage card', () => {
  const result = resolveStatusActionVisibility({
    isLoggedIn: true,
    mode: 'active',
    statusCard: {
      action: 'recent',
      actionText: '',
      secondaryAction: 'start',
      secondaryActionText: '再来一场'
    }
  })

  assert.deepEqual(result, {
    showPrimary: false,
    showSecondary: false
  })
})

test('resolveHighestRankDisplay picks the highest rank by level then score', () => {
  const result = resolveHighestRankDisplay([
    { gameType: 3, name: '白银球手', level: 2, rank_score: 1280 },
    { gameType: 2, name: '钻石王者', level: 5, rank_score: 3680 },
    { gameType: 1, name: '黄金高手', level: 3, rank_score: 2100 }
  ])

  assert.deepEqual(result, {
    gameType: 2,
    name: '钻石王者',
    level: 5,
    rankScore: 3680
  })
})

test('resolveHighestRankDisplay falls back to current rank when no cross-mode rank exists', () => {
  const result = resolveHighestRankDisplay([], {
    gameType: 3,
    name: '黄金高手',
    level: 3,
    rank_score: 2200
  })

  assert.deepEqual(result, {
    gameType: 3,
    name: '黄金高手',
    level: 3,
    rankScore: 2200
  })
})

test('resolveSectionTitles returns competitive naming', () => {
  const result = resolveSectionTitles()

  assert.equal(result.stats, '竞技概览')
  assert.equal(result.quickActions, '竞技社交')
  assert.equal(result.settings, '设置与支持')
})
