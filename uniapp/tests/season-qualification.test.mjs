import test from 'node:test'
import assert from 'node:assert/strict'

import {
  formatSeasonNetChange,
  mergeLeaderboardPage,
  resolveLeaderboardRank,
  resolveSeasonQualification
} from '../utils/season-qualification.js'

test('legacy seasons without thresholds treat every record as qualified', () => {
  const view = resolveSeasonQualification({ id: 1 }, { matches_played: 1 })
  assert.equal(view.hasThreshold, false)
  assert.equal(view.qualified, true)
  assert.equal(view.progressText, '')
  assert.equal(view.thresholdText, '')
})

test('v2 seasons report capped progress until the server marks the record qualified', () => {
  const season = { qualify_min_matches: 5, qualify_min_opponents: 3 }
  const pending = resolveSeasonQualification(season, { matches_played: 7, distinct_opponents: 1, qualified: false })
  assert.equal(pending.qualified, false)
  assert.equal(pending.progressText, '场次 5/5 · 不同对手 1/3')
  assert.equal(pending.thresholdText, '本季打满 5 场、遇到 3 位不同对手即参与赛季排名')
  assert.equal(resolveSeasonQualification(season, { qualified: true }).qualified, true)
  assert.equal(resolveSeasonQualification(season, null).qualified, false)
})

test('formatSeasonNetChange signs the season delta', () => {
  assert.equal(formatSeasonNetChange({ start_rank_score: 1250, end_rank_score: 1310 }), '+60')
  assert.equal(formatSeasonNetChange({ start_rank_score: 1250, end_rank_score: 1200 }), '-50')
  assert.equal(formatSeasonNetChange({ start_rank_score: 1250, end_rank_score: 1250 }), '0')
  assert.equal(formatSeasonNetChange(null), '')
})

test('mergeLeaderboardPage keeps the first occurrence when ranks shift between pages', () => {
  const merged = mergeLeaderboardPage(
    [{ user_id: 1, rank: 1 }, { user_id: 2, rank: 2 }],
    [{ user_id: 2, rank: 3 }, { user_id: 3, rank: 4 }]
  )
  assert.deepEqual(merged.map((item) => item.user_id), [1, 2, 3])
})

test('resolveLeaderboardRank hides ranks for unqualified rows and keeps old responses working', () => {
  assert.deepEqual(resolveLeaderboardRank({ rank: 2, qualified: true }), { rank: 2, text: '2', unqualified: false })
  assert.deepEqual(resolveLeaderboardRank({ rank: 0, qualified: false }), { rank: 0, text: '—', unqualified: true })
  assert.deepEqual(resolveLeaderboardRank({ rank: 5 }), { rank: 5, text: '5', unqualified: false })
})
