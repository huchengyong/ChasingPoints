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
  resolveCompactRewardEntry,
  resolveCoreMetrics,
  resolveHighestRankDisplay,
  resolveGuestHeroCopy,
  resolveMemberHeroStrip,
  resolvePrimaryAction,
  resolveStatusActionVisibility,
  resolveSectionTitles,
  resolveStatusCardContent,
  resolveUserHomepageModel,
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

test('resolveUserHomepageMode covers all stable homepage modes', () => {
  const cases = [
    [{ isLoggedIn: false }, 'guest'],
    [{ isLoggedIn: true, isLoading: true }, 'loading'],
    [{ isLoggedIn: true, totalMatches: 0 }, 'newcomer'],
    [{ isLoggedIn: true, totalMatches: 8 }, 'idle'],
    [{ isLoggedIn: true, currentMatch: { viewer_role: 'player1' } }, 'ongoing-player'],
    [{ isLoggedIn: true, currentMatch: { viewer_role: 'referee' } }, 'ongoing-referee']
  ]

  for (const [input, expected] of cases) {
    assert.equal(resolveUserHomepageMode(input), expected)
  }
})

test('resolvePrimaryAction maps stable modes to their main task', () => {
  assert.equal(resolvePrimaryAction('ongoing-player'), 'continue')
  assert.equal(resolvePrimaryAction('ongoing-referee'), 'continue')
  assert.equal(resolvePrimaryAction('guest'), 'login')
  assert.equal(resolvePrimaryAction('loading'), '')
  assert.equal(resolvePrimaryAction('idle'), 'start')
})

test('resolveStatusCardContent keeps fixed participants and scores for player2 view', () => {
  const result = resolveStatusCardContent({
    mode: 'ongoing-player',
    currentUser: { id: 22, nickname: '选手乙' },
    currentMatch: {
      viewer_role: 'player2',
      player1_id: 11,
      player1_name: '选手甲',
      player1_avatar: 'player1.png',
      player2_id: 22,
      player2_name: '选手乙',
      player2_avatar: 'player2.png',
      my_score: 3,
      opponent_score: 5,
      game_type_name: '中式八球',
      duration_seconds: 768
    }
  })

  assert.equal(result.action, 'continue')
  assert.equal(result.scoreText, '5 : 3')
  assert.deepEqual(result.players.map(item => [item.id, item.name, item.score]), [
    [11, '选手甲', 5],
    [22, '选手乙', 3]
  ])
})

test('resolveStatusCardContent gives referee the shared ongoing card structure', () => {
  const result = resolveStatusCardContent({
    mode: 'ongoing-referee',
    currentMatch: {
      viewer_role: 'referee',
      player1_name: '选手甲',
      player2_name: '选手乙',
      my_score: 4,
      opponent_score: 2
    }
  })

  assert.equal(result.type, 'ongoing')
  assert.equal(result.players.length, 2)
  assert.equal(result.scoreText, '4 : 2')
  assert.equal(result.actionText, '进入裁判记分')
})

test('loading status and metrics stay neutral before core requests finish', () => {
  const statusCard = resolveStatusCardContent({ mode: 'loading' })
  const metrics = resolveCoreMetrics({
    stats: { wins: 99, win_rate: 80 },
    rankInfo: { rank_score: 3000 },
    isLoading: true
  })

  assert.equal(statusCard.loading, true)
  assert.doesNotMatch(statusCard.description, /未定级|零战绩|还没开杆/)
  assert.deepEqual(metrics.map(item => item.value), ['—', '—', '—', '—'])
})

test('newcomer and idle cards share actions but use different copy', () => {
  const newcomer = resolveStatusCardContent({ mode: 'newcomer' })
  const idle = resolveStatusCardContent({ mode: 'idle' })

  assert.match(newcomer.title, /第一场/)
  assert.match(idle.title, /还没开杆/)
  assert.equal(newcomer.action, 'start')
  assert.equal(idle.secondaryAction, 'show_pk_code')
})

test('resolveGuestHeroCopy avoids login-wall phrasing', () => {
  const result = resolveGuestHeroCopy()

  assert.equal(result.title, '登录后，解锁你的个人竞技主页')
  assert.doesNotMatch(result.description, /请先完成登录/)
  assert.equal(result.primaryActionText, '登录/注册')
  assert.equal(result.secondaryActionText, '登录后发起 PK')
})

test('resolveStatusCardContent aligns guest actions with PK entry intents', () => {
  const result = resolveStatusCardContent({
    mode: 'guest'
  })

  assert.equal(result.action, 'login')
  assert.equal(result.actionText, '登录/注册')
  assert.equal(result.secondaryAction, 'start_pk')
  assert.equal(result.secondaryActionText, '登录后发起 PK')
})

test('resolveStatusActionVisibility follows the normalized status card actions', () => {
  const result = resolveStatusActionVisibility({
    statusCard: {
      actionText: '发起 PK',
      secondaryActionText: '出示二维码'
    }
  })

  assert.deepEqual(result, {
    showPrimary: true,
    showSecondary: true
  })
})

test('resolveMemberHeroStrip only exposes active or historical membership', () => {
  const now = new Date('2026-07-20T12:00:00+08:00')
  const active = resolveMemberHeroStrip({
    is_active: true,
    member_expires_at: '2026-08-20 12:00:00',
    growth_level: 3
  }, now)
  const expired = resolveMemberHeroStrip({
    is_active: true,
    member_expires_at: '2026-06-20 12:00:00'
  }, now)
  const none = resolveMemberHeroStrip({}, now)

  assert.equal(active.state, 'active')
  assert.equal(active.levelText, 'Lv3')
  assert.equal(expired.state, 'expired')
  assert.equal(none.visible, false)
})

test('resolveCompactRewardEntry keeps only actionable or pending reward states', () => {
  assert.equal(resolveCompactRewardEntry({}).visible, false)

  const notStarted = resolveCompactRewardEntry({ enabled: true, status: 'not_started', reward_days: 30 })
  const pending = resolveCompactRewardEntry({ enabled: true, status: 'pending_review' })
  const rejected = resolveCompactRewardEntry({ enabled: true, status: 'rejected', reject_reason: '资料不完整' })

  assert.equal(notStarted.action, 'favorite-venue')
  assert.equal(pending.visible, true)
  assert.equal(pending.action, '')
  assert.equal(rejected.actionText, '重新提交')
})

test('resolveUserHomepageModel safely normalizes missing API payloads', () => {
  const model = resolveUserHomepageModel({
    isLoggedIn: true,
    currentUser: null,
    stats: null,
    rankInfo: null,
    memberStatus: null,
    rewardStatus: null
  })

  assert.equal(model.mode, 'newcomer')
  assert.deepEqual(model.metrics.map(item => item.value), ['0%', 0, 0, 0])
  assert.equal(model.memberHeroStrip.visible, false)
  assert.equal(model.rewardEntry.visible, false)
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

  assert.equal(result.stats, '核心指标')
  assert.equal(result.quickActions, '常用入口')
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

test('home page hides the common tools section after moving tools to user services', () => {
  assert.doesNotMatch(homeIndexVueSource, /<text class="section-title">常用工具<\/text>/)
  assert.doesNotMatch(homeIndexVueSource, /label: '规则说明'/)
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
