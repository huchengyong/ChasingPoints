import test from 'node:test'
import assert from 'node:assert/strict'

import {
  DEFAULT_EVENT_COVER,
  DEFAULT_PLAYER_AVATAR,
  buildSaiXunDetailRounds,
  buildSaiXunHeroStats,
  formatEventDateRange,
  formatEventTimeRange,
  normalizeSaiXunCard
} from '../utils/saixun.js'

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
  assert.equal(card.title, 'Sportsbet.io Tour Championship 2026')
  assert.equal(card.summary, '今晚决出冠军')
  assert.equal(card.statusText, '进行中')
  assert.equal(card.dateText, '3月30日 - 4月5日')
  assert.equal(card.timeText, '')
  assert.equal(card.showTime, false)
  assert.equal(card.locationText, '曼彻斯特 · Manchester Central')
  assert.equal(card.currentRoundText, 'Semi Finals')
  assert.equal(card.matchCountText, '11 场比赛')
  assert.equal(card.sourceText, 'WST')
  assert.equal(card.coverImage, DEFAULT_EVENT_COVER)
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
      away_player_name: 'Barry Hawkins',
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
      home_player_avatar: 'https://example.com/trump.png',
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
  assert.equal(rounds[0].roundName, 'Semi Finals')
  assert.equal(rounds[1].roundName, 'Quarter Finals')
  assert.equal(rounds[0].matches[0].homePlayerAvatar, 'https://example.com/trump.png')
  assert.equal(rounds[0].matches[0].awayPlayerAvatar, DEFAULT_PLAYER_AVATAR)
  assert.equal(rounds[1].matches[0].scoreText, '10 : 8')
  assert.equal(rounds.some((round) => round.roundName === 'Final'), false)
})
