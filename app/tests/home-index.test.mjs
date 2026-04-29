import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildHomeNearbyVenueParams,
  buildFeaturedPostTarget,
  formatEventNewsTime,
  formatHomeVenueDistance,
  getEventNewsStatusText,
  HOME_NEARBY_VENUE_LIMIT,
  HOME_NEARBY_VENUE_RADIUS_METERS,
  resolveHomeVenueEmptyAction,
  resolveHomeToolNavigation,
  shouldShowHomeToolEdgeMask
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

const homeIndexScssSource = readFileSync(
  new URL('../pages/index/index.scss', import.meta.url),
  'utf8'
)

const homeIndexVueSource = readFileSync(
  new URL('../pages/index/index.vue', import.meta.url),
  'utf8'
)

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

test('resolveHomeToolNavigation sends guests to login for stats detail tool', () => {
  assert.deepEqual(resolveHomeToolNavigation({
    url: '/subPages/user/statsDetail',
    isLoggedIn: false
  }), {
    type: 'login',
    url: '/pages/login/login'
  })
})

test('resolveHomeToolNavigation keeps public tools directly navigable for guests', () => {
  assert.deepEqual(resolveHomeToolNavigation({
    url: '/subPages/rules/index',
    isLoggedIn: false
  }), {
    type: 'navigate',
    url: '/subPages/rules/index'
  })
})

test('resolveHomeToolNavigation keeps stats detail navigable after login', () => {
  assert.deepEqual(resolveHomeToolNavigation({
    url: '/subPages/user/statsDetail',
    isLoggedIn: true
  }), {
    type: 'navigate',
    url: '/subPages/user/statsDetail'
  })
})

test('home tool carousel leaves a trailing safe area so the last card is not visually clipped at the screen edge', () => {
  assert.match(homeIndexVueSource, /<view class="tool-scroll-shell" :class="\{ 'mask-hidden': !showToolScrollMask \}">/)
  assert.match(homeIndexVueSource, /<scroll-view scroll-x class="tool-scroll" show-scrollbar="false" @scroll="handleToolScroll">/)
  assert.match(homeIndexScssSource, /\.tool-scroll-shell\s*\{[\s\S]*position:\s*relative;/)
  assert.match(homeIndexScssSource, /\.tool-scroll\s*\{[\s\S]*&::\-webkit-scrollbar\s*\{[\s\S]*display:\s*none;/)
  assert.match(homeIndexScssSource, /\.tool-scroll\s*\{[\s\S]*scrollbar-width:\s*none;/)
  assert.match(homeIndexScssSource, /&::after\s*\{[\s\S]*width:\s*48rpx;/)
  assert.match(homeIndexScssSource, /&\.mask-hidden::after\s*\{[\s\S]*opacity:\s*0;/)
  assert.match(homeIndexScssSource, /&::after\s*\{[\s\S]*linear-gradient\(90deg,\s*rgba\(255,\s*255,\s*255,\s*0\),\s*var\(--home-bg\)/)
})

test('shouldShowHomeToolEdgeMask stays visible until the carousel reaches the right edge', () => {
  assert.equal(shouldShowHomeToolEdgeMask({ maxScrollLeft: 180, scrollLeft: 0 }), true)
  assert.equal(shouldShowHomeToolEdgeMask({ maxScrollLeft: 180, scrollLeft: 96 }), true)
  assert.equal(shouldShowHomeToolEdgeMask({ maxScrollLeft: 180, scrollLeft: 176 }), false)
  assert.equal(shouldShowHomeToolEdgeMask({ maxScrollLeft: 0, scrollLeft: 0 }), false)
})

test('home nearby venue params use a 5km radius and three-card preview limit', () => {
  assert.equal(HOME_NEARBY_VENUE_RADIUS_METERS, 5000)
  assert.equal(HOME_NEARBY_VENUE_LIMIT, 3)
  assert.deepEqual(buildHomeNearbyVenueParams({
    latitude: 22.5431,
    longitude: 114.0579
  }), {
    latitude: 22.5431,
    longitude: 114.0579,
    radius: 5000,
    limit: 3
  })
})

test('home nearby venue params reject missing coordinates', () => {
  assert.equal(buildHomeNearbyVenueParams({ latitude: 0, longitude: 114.0579 }), null)
  assert.equal(buildHomeNearbyVenueParams({ latitude: 22.5431, longitude: 0 }), null)
})

test('formatHomeVenueDistance keeps home cards compact', () => {
  assert.equal(formatHomeVenueDistance(860), '860m')
  assert.equal(formatHomeVenueDistance(1260), '1.3km')
  assert.equal(formatHomeVenueDistance(0), '')
})

test('home nearby venue empty action mentions member reward only when activity is enabled', () => {
  assert.deepEqual(resolveHomeVenueEmptyAction({ enabled: true }), {
    text: '添加球馆领取会员',
    desc: '你可以提交常玩球馆，审核通过后会员会自动到账。',
    url: '/subPages/venue/submit'
  })
  assert.deepEqual(resolveHomeVenueEmptyAction({ enabled: false }), {
    text: '添加附近球馆',
    desc: '你可以提交常去球馆，审核通过后会展示给附近球友。',
    url: '/subPages/venue/submit'
  })
  assert.deepEqual(resolveHomeVenueEmptyAction(null), {
    text: '添加附近球馆',
    desc: '你可以提交常去球馆，审核通过后会展示给附近球友。',
    url: '/subPages/venue/submit'
  })
})

test('home page exposes nearby venues preview and configurable add venue CTA', () => {
  assert.match(homeIndexVueSource, /getNearbyVenues/)
  assert.match(homeIndexVueSource, /getFavoriteVenueRewardStatus/)
  assert.match(homeIndexVueSource, /nearby-venue-section/)
  assert.match(homeIndexVueSource, /查看球房/)
  assert.match(homeIndexVueSource, /\/subPages\/venue\/index/)
  assert.match(homeIndexVueSource, /homeVenueEmptyAction\.text/)
  assert.match(homeIndexVueSource, /homeVenueEmptyAction\.url/)
  assert.match(homeIndexScssSource, /\.venue-empty-card\s*\{[\s\S]*text-align:\s*center;/)
  assert.match(homeIndexScssSource, /\.venue-empty-card\s*\{[\s\S]*\.empty-action\s*\{[\s\S]*align-self:\s*center;/)
})
