import test from 'node:test'
import assert from 'node:assert/strict'

import {
  DEFAULT_EVENT_COVER,
  DEFAULT_PLAYER_AVATAR,
  buildSaiXunDetailRounds,
  buildSaiXunHeroStats,
  formatEventDateRange,
  formatEventTimeRange,
  localizeTournamentTitle,
  normalizeSaiXunCard,
  resolvePlayerFlag,
  sanitizeFlagEmoji
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

test('buildSaiXunDetailRounds prioritizes live and upcoming rounds while hiding later placeholders', () => {
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
  assert.equal(rounds[0].roundName, '半决赛')
  assert.equal(rounds[1].roundName, '四分之一决赛')
  assert.equal(rounds[0].matches[0].homePlayerAvatar, 'https://cdn.example.com/wst/players/trump.png')
  assert.equal(rounds[0].matches[0].awayPlayerAvatar, DEFAULT_PLAYER_AVATAR)
  assert.equal(rounds[1].matches[0].scoreText, '10 : 8')
  assert.equal(rounds[1].matches[0].homeResultText, '胜')
  assert.equal(rounds[1].matches[0].awayResultText, '败')
  assert.equal(rounds[1].matches[0].homePlayerFirstName, 'Neil')
  assert.equal(rounds[1].matches[0].homePlayerLastName, 'Robertson')
  assert.equal(rounds[1].matches[0].homePlayerFlagEmoji, '🇦🇺')
  assert.equal(rounds[1].matches[0].homePlayerCountryCode, 'au')
  assert.deepEqual(rounds[1].matches[0].homePlayerFlag, { type: 'emoji', value: '🇦🇺' })
  assert.equal(rounds[1].matches[0].awayPlayerFirstName, 'Barry')
  assert.equal(rounds[1].matches[0].awayPlayerLastName, 'Hawkins')
  assert.equal(rounds[1].matches[0].awayPlayerFlagEmoji, '')
  assert.equal(rounds[1].matches[0].awayPlayerCountryCode, 'gb-eng')
  assert.deepEqual(rounds[1].matches[0].awayPlayerFlag, { type: 'image', value: '/static/flags/eng.png' })
  assert.equal(rounds.some((round) => round.roundName === '决赛'), false)
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

  assert.equal(rounds[0].matches[0].status, 1)
  assert.equal(rounds[0].matches[0].statusText, '进行中')
  assert.equal(rounds[0].matches[1].status, 2)
  assert.equal(rounds[0].matches[1].statusText, '已结束')
  assert.equal(rounds[0].matches[0].homeResultText, '')
  assert.equal(rounds[0].matches[0].awayResultText, '')
  assert.equal(rounds[0].matches[1].homeResultText, '')
  assert.equal(rounds[0].matches[1].awayResultText, '')
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

test('resolvePlayerFlag returns none for unknown or empty codes', () => {
  assert.deepEqual(resolvePlayerFlag('', 'gb-nir'), { type: 'none', value: '' })
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
