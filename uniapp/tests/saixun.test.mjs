import test from 'node:test'
import assert from 'node:assert/strict'

import {
  DEFAULT_EVENT_COVER,
  DEFAULT_PLAYER_AVATAR,
  OFFICIAL_PLAYER_AVATAR_PLACEHOLDER,
  buildEventRegionText,
  buildSaiXunDetailRounds,
  buildSaiXunHeroStats,
  buildSaiXunListSections,
  formatBestOfText,
  formatEventCountdownText,
  formatEventDateRange,
  formatEventTimeRange,
  localizeTournamentTitle,
  normalizeSaiXunCard,
  resolvePlayerFlag,
  resolveSaiXunChampion,
  resolveSaiXunFocusRoundKey,
  sanitizeFlagEmoji,
  standardizeRoundText
} from '../utils/saixun.js'

test('DEFAULT_EVENT_COVER uses a bundled application asset', () => {
  assert.equal(DEFAULT_EVENT_COVER, '/static/images/default-event-cover.png')
  assert.equal(DEFAULT_EVENT_COVER.startsWith('http'), false)
})

test('formatEventDateRange prefers date-only output', () => {
  assert.equal(formatEventDateRange('2026-03-30', '2026-04-05'), '3月30日 - 4月5日')
  assert.equal(formatEventDateRange('2026-03-30', '2026-03-30'), '3月30日')
  assert.equal(formatEventDateRange('', ''), '日期待定')
})

test('formatEventTimeRange returns empty string when event has no explicit time', () => {
  assert.equal(formatEventTimeRange('', '', '2026-04-01T12:00:00+08:00'), '')
  assert.equal(
    formatEventTimeRange('2026-04-01T20:00:00+08:00', '2026-04-05T20:00:00+08:00', '2026-04-01T12:00:00+08:00'),
    '今天 20:00 - 4月5日 20:00'
  )
})

test('normalizeSaiXunCard maps赛事摘要并在缺失封面时回退默认值', () => {
  const card = normalizeSaiXunCard({
    id: 7,
    title: '斯诺克公开赛总决赛',
    tournament_name: 'Sportsbet.io Tour Championship 2026',
    latest_result_text: '今晚决出冠军',
    current_round_text: 'Semi Finals',
    match_count: 11,
    game_type: 1,
    status: 1,
    start_date: '2026-03-30',
    end_date: '2026-04-05',
    city: '曼彻斯特',
    venue: 'Manchester Central',
    source_name: 'WST'
  }, '2026-04-01T12:00:00+08:00')

  assert.equal(card.id, 7)
  assert.equal(card.title, '斯诺克巡回锦标赛 2026')
  assert.equal(card.summary, '今晚决出冠军')
  assert.equal(card.statusText, '进行中')
  assert.equal(card.dateText, '3月30日 - 4月5日')
  assert.equal(card.timeText, '')
  assert.equal(card.showTime, false)
  assert.equal(card.locationText, '曼彻斯特 · Manchester Central')
  assert.equal(card.currentRoundText, '半决赛')
  assert.equal(card.matchCountText, '11 场比赛')
  assert.equal(card.sourceText, 'WST')
  assert.equal(card.coverImage, DEFAULT_EVENT_COVER)
})

test('normalizeSaiXunCard keeps a mirrored CDN cover URL', () => {
  const coverImage = 'https://cdn.example.com/wst/tournaments/tour-cover.png'
  const card = normalizeSaiXunCard({
    id: 8,
    title: 'Tour Championship 2026',
    cover_image: coverImage
  })

  assert.equal(card.coverImage, coverImage)
})

test('localizeTournamentTitle strips sponsor prefix and keeps year for standard WST events', () => {
  assert.equal(localizeTournamentTitle('Sportsbet.io Tour Championship 2026'), '斯诺克巡回锦标赛 2026')
  assert.equal(localizeTournamentTitle('BetVictor Scottish Open 2025'), '苏格兰公开赛 2025')
  assert.equal(localizeTournamentTitle('Victorian Plumbing UK Championship 2025'), '英国锦标赛 2025')
  assert.equal(localizeTournamentTitle('Sportsbet.io Champion of Champions'), '冠中冠')
})

test('localizeTournamentTitle translates championship league phases and falls back for unknown titles', () => {
  assert.equal(
    localizeTournamentTitle('BetVictor Championship League Snooker 2025 (Stage One/WK1)'),
    '冠军联赛 2025（第一阶段 / 第1周）'
  )
  assert.equal(localizeTournamentTitle('Championship League Group Three'), '冠军联赛第3组')
  assert.equal(localizeTournamentTitle('Unknown Cup 2025'), 'Unknown Cup 2025')
})

test('buildSaiXunHeroStats counts total live and upcoming items', () => {
  const stats = buildSaiXunHeroStats([
    { status: 1 },
    { status: 0 },
    { status: 1 },
    { status: 2 }
  ])

  assert.deepEqual(stats, {
    totalCount: 4,
    liveCount: 2,
    upcomingCount: 1
  })
})

test('buildSaiXunDetailRounds orders rounds chronologically and hides all-placeholder rounds', () => {
  const rounds = buildSaiXunDetailRounds([
    {
      id: 101,
      round_name: 'Quarter Finals',
      round_order: 2,
      match_order: 2,
      start_time: '2026-04-01T20:00:00+08:00',
      status: 2,
      best_of: 19,
      home_player_name: 'Neil Robertson',
      home_player_first_name: 'Neil',
      home_player_last_name: 'Robertson',
      home_player_flag_emoji: '🇦🇺',
      home_player_country_code: 'au',
      away_player_name: 'Barry Hawkins',
      away_player_first_name: 'Barry',
      away_player_last_name: 'Hawkins',
      away_player_flag_emoji: '🏴',
      away_player_country_code: 'gb-eng',
      home_score: 10,
      away_score: 8,
      winner_side: 1
    },
    {
      id: 102,
      round_name: 'Semi Finals',
      round_order: 3,
      match_order: 1,
      start_time: '2026-04-03T20:00:00+08:00',
      status: 0,
      best_of: 19,
      source_type: 'official',
      home_player_name: 'Judd Trump',
      away_player_name: '待定',
      home_player_avatar: 'https://cdn.example.com/wst/players/trump.png',
      is_placeholder: false
    },
    {
      id: 103,
      round_name: 'Final',
      round_order: 4,
      match_order: 1,
      start_time: '2026-04-05T20:00:00+08:00',
      status: 0,
      is_placeholder: true
    }
  ], '2026-04-03T12:00:00+08:00')

  assert.equal(rounds.length, 2)
  assert.equal(rounds[0].roundName, '四分之一决赛')
  assert.equal(rounds[1].roundName, '半决赛')
  assert.equal(rounds[1].matches[0].homePlayerAvatar, 'https://cdn.example.com/wst/players/trump.png')
  assert.equal(rounds[1].matches[0].awayPlayerAvatar, OFFICIAL_PLAYER_AVATAR_PLACEHOLDER)
  assert.equal(rounds[0].matches[0].scoreText, '10 : 8')
  assert.equal(rounds[0].matches[0].showScore, true)
  assert.equal(rounds[0].matches[0].homeScore, 10)
  assert.equal(rounds[0].matches[0].awayScore, 8)
  assert.equal(rounds[0].matches[0].metaText, '19局10胜')
  assert.equal(rounds[0].metaText, '19局10胜')
  assert.equal(rounds[0].matches[0].homeResultText, '胜')
  assert.equal(rounds[0].matches[0].awayResultText, '败')
  assert.equal(rounds[0].matches[0].homePlayerFirstName, 'Neil')
  assert.equal(rounds[0].matches[0].homePlayerLastName, 'Robertson')
  assert.equal(rounds[0].matches[0].homePlayerFlagEmoji, '🇦🇺')
  assert.equal(rounds[0].matches[0].homePlayerCountryCode, 'au')
  assert.deepEqual(rounds[0].matches[0].homePlayerFlag, { type: 'emoji', value: '🇦🇺' })
  assert.equal(rounds[0].matches[0].awayPlayerFirstName, 'Barry')
  assert.equal(rounds[0].matches[0].awayPlayerLastName, 'Hawkins')
  assert.equal(rounds[0].matches[0].awayPlayerFlagEmoji, '')
  assert.equal(rounds[0].matches[0].awayPlayerCountryCode, 'gb-eng')
  assert.deepEqual(rounds[0].matches[0].awayPlayerFlag, { type: 'image', value: '/static/flags/eng.png' })
  assert.equal(rounds.some((round) => round.roundName === '决赛'), false)
  assert.equal(resolveSaiXunFocusRoundKey(rounds), rounds[1].key)
  assert.equal(resolveSaiXunChampion(rounds), null)
})

test('official matches use the official avatar placeholder while platform ones keep the app avatar', () => {
  const base = {
    id: 1,
    round_name: 'Round 1',
    round_order: 1,
    match_order: 1,
    status: 2
  }
  const official = buildSaiXunDetailRounds([{ ...base, source_type: 'official' }], Date.now())
  const platform = buildSaiXunDetailRounds([{ ...base, source_type: '' }], Date.now())

  assert.equal(official[0].matches[0].homePlayerAvatar, OFFICIAL_PLAYER_AVATAR_PLACEHOLDER)
  assert.equal(official[0].matches[0].awayPlayerAvatar, OFFICIAL_PLAYER_AVATAR_PLACEHOLDER)
  assert.equal(platform[0].matches[0].homePlayerAvatar, DEFAULT_PLAYER_AVATAR)
  assert.equal(platform[0].matches[0].awayPlayerAvatar, DEFAULT_PLAYER_AVATAR)
})

test('buildSaiXunDetailRounds promotes overdue scheduled matches to live status', () => {
  const rounds = buildSaiXunDetailRounds([
    {
      id: 201,
      round_name: 'Semi Finals',
      round_order: 3,
      match_order: 1,
      start_time: '2026-04-04T20:00:00+08:00',
      status: 0,
      home_player_name: 'John Higgins',
      away_player_name: 'Zhao Xintong',
      winner_side: 0
    }
  ], '2026-04-04T20:53:00+08:00')

  assert.equal(rounds[0].matches[0].status, 1)
  assert.equal(rounds[0].matches[0].statusText, '进行中')
})

test('buildSaiXunDetailRounds promotes overdue scored matches with winner to completed status', () => {
  const rounds = buildSaiXunDetailRounds([
    {
      id: 202,
      round_name: 'Quarter Finals',
      round_order: 2,
      match_order: 1,
      start_time: '2026-04-04T18:00:00+08:00',
      status: 0,
      home_player_name: 'Neil Robertson',
      away_player_name: 'Barry Hawkins',
      home_score: 10,
      away_score: 8,
      winner_side: 1
    }
  ], '2026-04-04T20:53:00+08:00')

  assert.equal(rounds[0].matches[0].status, 2)
  assert.equal(rounds[0].matches[0].statusText, '已结束')
  assert.equal(rounds[0].matches[0].homeResultText, '胜')
  assert.equal(rounds[0].matches[0].awayResultText, '败')
})

test('buildSaiXunDetailRounds marks stale overdue matches as completed instead of live forever', () => {
  const rounds = buildSaiXunDetailRounds([
    {
      id: 203,
      round_name: 'Semi Finals',
      round_order: 3,
      match_order: 1,
      start_time: '2026-04-03T20:00:00+08:00',
      status: 0,
      home_player_name: 'Neil Robertson',
      away_player_name: 'Judd Trump',
      winner_side: 0
    },
    {
      id: 204,
      round_name: 'Semi Finals',
      round_order: 3,
      match_order: 2,
      start_time: '2026-04-04T20:00:00+08:00',
      status: 0,
      home_player_name: 'John Higgins',
      away_player_name: 'Zhao Xintong',
      winner_side: 0
    }
  ], '2026-04-04T21:03:00+08:00')

  const staleMatch = rounds[0].matches.find((match) => match.id === 203)
  const liveMatch = rounds[0].matches.find((match) => match.id === 204)
  assert.deepEqual(rounds[0].matches.map((match) => match.id), [203, 204])
  assert.equal(liveMatch.status, 1)
  assert.equal(liveMatch.statusText, '进行中')
  assert.equal(liveMatch.isLive, true)
  assert.equal(liveMatch.showScore, false)
  assert.equal(staleMatch.status, 2)
  assert.equal(staleMatch.statusText, '已结束')
  assert.equal(staleMatch.showScore, false)
  assert.equal(liveMatch.homeResultText, '')
  assert.equal(liveMatch.awayResultText, '')
  assert.equal(staleMatch.homeResultText, '')
  assert.equal(staleMatch.awayResultText, '')
  assert.equal(rounds[0].hasLive, true)
  assert.deepEqual(rounds[0].days.map((day) => [day.label, day.matches.length]), [
    ['4月3日 周五 · 昨天', 1],
    ['4月4日 周六 · 今天', 1]
  ])
})

// Flag emoji tests

test('sanitizeFlagEmoji filters black flag base and all subdivision flags', () => {
  assert.equal(sanitizeFlagEmoji(''), '')
  assert.equal(sanitizeFlagEmoji('🇦🇺'), '🇦🇺')
  assert.equal(sanitizeFlagEmoji('🇨🇳'), '🇨🇳')
  // Black flag base (U+1F3F4) — bare or with tag sequence
  assert.equal(sanitizeFlagEmoji('🏴'), '')
  // Correct England flag (🏴 + gbeng tag sequence)
  const englandFlag = '\u{1F3F4}\u{E0067}\u{E0062}\u{E0065}\u{E006E}\u{E0067}\u{E007F}'
  assert.equal(englandFlag.codePointAt(0), 0x1F3F4)
  assert.equal(sanitizeFlagEmoji(englandFlag), '')
  // Wrong historical sequence (eng tag without gb prefix)
  const wrongEngFlag = '\u{1F3F4}\u{E0065}\u{E006E}\u{E0067}\u{E007F}'
  assert.equal(sanitizeFlagEmoji(wrongEngFlag), '')
})

test('resolvePlayerFlag returns local image for England, Scotland, Wales', () => {
  assert.deepEqual(resolvePlayerFlag('', 'gb-eng'), { type: 'image', value: '/static/flags/eng.png' })
  assert.deepEqual(resolvePlayerFlag('', 'gb-sct'), { type: 'image', value: '/static/flags/sco.png' })
  assert.deepEqual(resolvePlayerFlag('', 'gb-wls'), { type: 'image', value: '/static/flags/wls.png' })
  // Short codes
  assert.deepEqual(resolvePlayerFlag('', 'eng'), { type: 'image', value: '/static/flags/eng.png' })
  assert.deepEqual(resolvePlayerFlag('', 'sct'), { type: 'image', value: '/static/flags/sco.png' })
  assert.deepEqual(resolvePlayerFlag('', 'wls'), { type: 'image', value: '/static/flags/wls.png' })
})

test('resolvePlayerFlag returns emoji for normal country codes', () => {
  assert.deepEqual(resolvePlayerFlag('🇦🇺', 'au'), { type: 'emoji', value: '🇦🇺' })
  assert.deepEqual(resolvePlayerFlag('🇨🇳', 'cn'), { type: 'emoji', value: '🇨🇳' })
  assert.deepEqual(resolvePlayerFlag('🇧🇪', 'be'), { type: 'emoji', value: '🇧🇪' })
})

test('resolvePlayerFlag returns image for Northern Ireland', () => {
  assert.deepEqual(resolvePlayerFlag('', 'gb-nir'), { type: 'image', value: '/static/flags/nir.png' })
  assert.deepEqual(resolvePlayerFlag('', 'nir'), { type: 'image', value: '/static/flags/nir.png' })
})

test('resolvePlayerFlag returns none for unknown or empty codes', () => {
  assert.deepEqual(resolvePlayerFlag('', 'gb-xxx'), { type: 'none', value: '' })
  assert.deepEqual(resolvePlayerFlag('', ''), { type: 'none', value: '' })
  assert.deepEqual(resolvePlayerFlag('', 'xx'), { type: 'none', value: '' })
})

test('resolvePlayerFlag prefers image over emoji for subdivision codes', () => {
  // Even if a broken flag emoji is provided, country code wins
  assert.deepEqual(resolvePlayerFlag('🏴', 'gb-eng'), { type: 'image', value: '/static/flags/eng.png' })
  assert.deepEqual(resolvePlayerFlag('🏴', 'gb-sct'), { type: 'image', value: '/static/flags/sco.png' })
})

test('buildSaiXunDetailRounds normalizes subdivision flags to image type and filters bad emoji', () => {
  const rounds = buildSaiXunDetailRounds([
    {
      id: 301,
      round_name: 'Final',
      round_order: 5,
      match_order: 1,
      status: 2,
      home_player_name: 'Judd Trump',
      home_player_flag_emoji: '🇬🇧',
      home_player_country_code: 'gb-eng',
      away_player_name: 'Mark Williams',
      away_player_flag_emoji: '🏴',
      away_player_country_code: 'gb-wls',
      winner_side: 1
    }
  ], '2026-04-06T20:00:00+08:00')

  assert.equal(rounds[0].matches[0].homePlayerFlagEmoji, '🇬🇧')
  assert.deepEqual(rounds[0].matches[0].homePlayerFlag, { type: 'image', value: '/static/flags/eng.png' })
  assert.equal(rounds[0].matches[0].awayPlayerFlagEmoji, '')
  assert.deepEqual(rounds[0].matches[0].awayPlayerFlag, { type: 'image', value: '/static/flags/wls.png' })
})

test('localizeTournamentTitle keeps qualifier phases apart from the main event', () => {
  assert.equal(localizeTournamentTitle('BetVictor Northern Ireland Open 2026'), '北爱尔兰公开赛 2026')
  assert.equal(localizeTournamentTitle('BetVictor Northern Ireland Open 2026 Qualifiers'), '北爱尔兰公开赛 2026 资格赛')
  assert.equal(localizeTournamentTitle('Halo World Championship 2026 Qualifiers'), '世界锦标赛 2026 资格赛')
  assert.equal(localizeTournamentTitle('Saudi Arabia Snooker Masters 2024 Final Stages'), '沙特阿拉伯斯诺克大师赛 2024 决赛阶段')
  assert.equal(localizeTournamentTitle('Shenzhen Open 2026'), '深圳公开赛 2026')
  assert.equal(localizeTournamentTitle('Riyadh Season World Masters of Snooker 2024'), '世界斯诺克大师赛 2024')
  assert.equal(localizeTournamentTitle('World Mixed Doubles 2024'), '世界混双赛 2024')
  assert.equal(localizeTournamentTitle('Asia-Oceania Q School 2026'), '亚洲及大洋洲 Q School 2026')
})

test('localizeTournamentTitle covers sponsored championship league groups', () => {
  assert.equal(
    localizeTournamentTitle('BetVictor Championship League Snooker Invitational 2026 (Group 7)'),
    '冠军联赛 2026（第7组）'
  )
  assert.equal(
    localizeTournamentTitle('BetVictor Championship League Snooker Invitational 2026 (Winners Group)'),
    '冠军联赛 2026（胜者组）'
  )
  assert.equal(localizeTournamentTitle('BetVictor Championship League Group 7'), '冠军联赛第7组')
  assert.equal(localizeTournamentTitle('BetVictor Championship League Winners Group'), '冠军联赛胜者组')
})

test('standardizeRoundText localizes WST round names', () => {
  assert.equal(standardizeRoundText('Round 1'), '第1轮')
  assert.equal(standardizeRoundText('Round 2 (Held Over)'), '第2轮（保留赛）')
  assert.equal(standardizeRoundText('Quarter Finals'), '四分之一决赛')
  assert.equal(standardizeRoundText('Last 16'), '16强')
  assert.equal(standardizeRoundText('League Phase (STAGE TWO / WEEK 2)'), '联赛阶段（第二阶段 / 第2周）')
  assert.equal(standardizeRoundText('Stage One/WK2'), '第一阶段 / 第2周')
  assert.equal(standardizeRoundText('小组赛'), '小组赛')
})

test('formatBestOfText follows the Chinese frame format convention', () => {
  assert.equal(formatBestOfText(9), '9局5胜')
  assert.equal(formatBestOfText(35), '35局18胜')
  assert.equal(formatBestOfText(4), '共4局')
  assert.equal(formatBestOfText(0), '')
})

test('formatEventCountdownText counts calendar days to the event start', () => {
  const now = '2026-10-17T19:30:00+08:00'
  assert.equal(formatEventCountdownText('2026-10-17', now), '今天开赛')
  assert.equal(formatEventCountdownText('2026-10-18', now), '明天开赛')
  assert.equal(formatEventCountdownText('2026-10-24', now), '7天后开赛')
  assert.equal(formatEventCountdownText('2026-12-01', now), '12月1日开赛')
  assert.equal(formatEventCountdownText('2026-10-10', now), '即将开始')
  assert.equal(formatEventCountdownText('', now), '即将开始')
})

test('normalizeSaiXunCard derives champion highlight only from a finished final', () => {
  const finished = normalizeSaiXunCard({
    id: 164,
    tournament_name: 'Shenzhen Open 2026',
    status: 2,
    country: 'China',
    city: 'Shenzhen',
    start_date: '2026-09-28',
    end_date: '2026-10-04',
    current_round_text: 'Final',
    latest_result_text: 'Yuan Sijun 5 - 10 Wu Yize'
  }, '2026-10-17T12:00:00+08:00')

  assert.equal(finished.title, '深圳公开赛 2026')
  assert.equal(finished.originalTitle, 'Shenzhen Open 2026')
  assert.equal(finished.statusTone, 'ended')
  assert.equal(finished.isFeatured, false)
  assert.equal(finished.metaText, '9月28日 - 10月4日 · 中国 · Shenzhen')
  assert.deepEqual(finished.highlight, {
    type: 'champion',
    label: '冠军',
    primary: 'Wu Yize',
    secondary: '决赛 10-5 胜 Yuan Sijun'
  })

  const qualifier = normalizeSaiXunCard({
    tournament_name: 'BetVictor Scottish Open 2026 Qualifiers',
    status: 2,
    current_round_text: 'Round 2',
    latest_result_text: 'Liu Hongyu 4 - 3 Jimmy White'
  })
  assert.equal(qualifier.highlight, null)

  const upcoming = normalizeSaiXunCard({
    tournament_name: 'BetVictor Northern Ireland Open 2026',
    status: 0,
    start_date: '2026-10-18',
    current_round_text: 'Round 1',
    latest_result_text: 'Jack Lisowski - Huang Jiahao'
  }, '2026-10-17T12:00:00+08:00')
  assert.equal(upcoming.isFeatured, true)
  assert.equal(upcoming.statusTone, 'upcoming')
  assert.equal(upcoming.statusLabel, '明天开赛')
  assert.deepEqual(upcoming.highlight, {
    type: 'upcoming',
    label: '第1轮',
    primary: 'Jack Lisowski vs Huang Jiahao',
    secondary: ''
  })

  const live = normalizeSaiXunCard({ status: 1, current_round_text: 'Semi Finals', latest_result_text: 'Judd Trump 3 - 2 Mark Selby' })
  assert.equal(live.statusLabel, '进行中')
  assert.equal(live.highlight.primary, 'Judd Trump 3 - 2 Mark Selby')
  assert.equal(live.highlight.label, '半决赛')
})

test('buildEventRegionText localizes country and drops duplicated parts', () => {
  assert.equal(buildEventRegionText({ country: 'Northern Ireland', city: 'Belfast' }), '北爱尔兰 · Belfast')
  assert.equal(buildEventRegionText({ country: '中国', city: '中国', venue: '' }), '中国')
  assert.equal(buildEventRegionText({}), '')
})

test('buildSaiXunListSections groups adjacent items by status and finished month', () => {
  const sections = buildSaiXunListSections([
    { id: 1, status: 1, start_date: '2026-10-12' },
    { id: 2, status: 0, start_date: '2026-10-18' },
    { id: 3, status: 0, start_date: '2026-11-02' },
    { id: 4, status: 2, start_date: '2026-09-28' },
    { id: 5, status: 2, start_date: '2026-09-07' },
    { id: 6, status: 2, start_date: '2026-08-31' },
    { id: 7, status: 3, start_date: '2026-08-01' }
  ])

  assert.deepEqual(sections.map((section) => [section.title, section.caption, section.items.map((item) => item.id)]), [
    ['进行中', '', [1]],
    ['即将开始', '', [2, 3]],
    ['9月', '已结束', [4, 5]],
    ['8月', '已结束', [6]],
    ['已取消', '', [7]]
  ])
  assert.equal(new Set(sections.map((section) => section.key)).size, sections.length)
})

test('detail rounds focus the live round first, then the next scheduled round, then the last round', () => {
  const finishedMatch = { status: 2, home_score: 4, away_score: 1, winner_side: 1, start_time: '2026-09-20T17:00:00+08:00' }
  const twoRounds = (secondRoundStatus) => buildSaiXunDetailRounds([
    { id: 1, round_name: 'Round 1', round_order: 10, match_order: 1, ...finishedMatch },
    { id: 2, round_name: 'Round 2', round_order: 20, match_order: 1, status: secondRoundStatus, start_time: '2026-09-22T17:00:00+08:00' }
  ], '2026-09-21T12:00:00+08:00')

  const upcomingRounds = twoRounds(0)
  assert.equal(resolveSaiXunFocusRoundKey(upcomingRounds), upcomingRounds[1].key)

  const liveRounds = buildSaiXunDetailRounds([
    { id: 1, round_name: 'Round 1', round_order: 10, match_order: 1, status: 1, start_time: '2026-09-21T11:00:00+08:00' },
    { id: 2, round_name: 'Round 2', round_order: 20, match_order: 1, status: 0, start_time: '2026-09-22T17:00:00+08:00' }
  ], '2026-09-21T12:00:00+08:00')
  assert.equal(resolveSaiXunFocusRoundKey(liveRounds), liveRounds[0].key)

  const finishedRounds = twoRounds(2)
  assert.equal(resolveSaiXunFocusRoundKey(finishedRounds), finishedRounds[1].key)
  assert.equal(resolveSaiXunFocusRoundKey([]), '')
})

test('held-over rounds sharing round_order stay in schedule order', () => {
  const rounds = buildSaiXunDetailRounds([
    { id: 3, round_name: 'Round 2 (Held Over)', round_order: 0, match_order: 1, status: 2, start_time: '2026-09-28T19:00:00+08:00' },
    { id: 1, round_name: 'Round 1 (Held Over)', round_order: 0, match_order: 1, status: 2, start_time: '2026-09-28T09:30:00+08:00' },
    { id: 4, round_name: 'Round 3', round_order: 30, match_order: 1, status: 2, start_time: '2026-09-29T14:00:00+08:00' }
  ], '2026-10-17T12:00:00+08:00')

  assert.deepEqual(rounds.map((round) => round.roundName), ['第1轮（保留赛）', '第2轮（保留赛）', '第3轮'])
  assert.equal(rounds[0].matches[0].clockText, '09:30')
})

test('resolveSaiXunChampion reads the finished final from detail rounds', () => {
  const rounds = buildSaiXunDetailRounds([
    {
      id: 1,
      round_name: 'Final',
      round_order: 70,
      match_order: 1,
      status: 2,
      source_type: 'official',
      home_player_name: 'Yuan Sijun',
      away_player_name: 'Wu Yize',
      away_player_country_code: 'CN',
      away_player_flag_emoji: '🇨🇳',
      away_player_avatar: 'https://cdn.example.com/wst/players/wu.png',
      home_score: 5,
      away_score: 10,
      winner_side: 2
    }
  ], '2026-10-17T12:00:00+08:00')

  assert.deepEqual(resolveSaiXunChampion(rounds), {
    name: 'Wu Yize',
    avatar: 'https://cdn.example.com/wst/players/wu.png',
    flag: { type: 'emoji', value: '🇨🇳' },
    opponentName: 'Yuan Sijun',
    scoreText: '10-5'
  })
})
