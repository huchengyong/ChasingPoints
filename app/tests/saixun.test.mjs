import test from 'node:test'
import assert from 'node:assert/strict'

import { buildSaiXunHeroStats, normalizeSaiXunCard } from '../utils/saixun.js'

test('normalizeSaiXunCard maps event news into readonly saixun cards', () => {
  const card = normalizeSaiXunCard({
    id: 7,
    title: '斯诺克公开赛总决赛',
    tournament_name: 'Sportsbet.io Tour Championship 2026',
    latest_result_text: '今晚决出冠军',
    current_round_text: '决赛',
    match_count: 11,
    game_type: 1,
    status: 1,
    start_time: '2026-04-01T20:00:00+08:00',
    end_time: '2026-04-05T20:00:00+08:00',
    city: '上海',
    venue: '浦东馆',
    source_name: '追分官方'
  }, '2026-04-01T12:00:00+08:00')

  assert.equal(card.id, 7)
  assert.equal(card.title, 'Sportsbet.io Tour Championship 2026')
  assert.equal(card.summary, '今晚决出冠军')
  assert.equal(card.statusText, '进行中')
  assert.equal(card.timeText, '今天 20:00 - 4月5日 20:00')
  assert.equal(card.locationText, '上海 · 浦东馆')
  assert.equal(card.currentRoundText, '决赛')
  assert.equal(card.matchCountText, '11 场比赛')
  assert.equal(card.sourceText, '追分官方')
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
