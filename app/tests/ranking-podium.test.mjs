import test from 'node:test'
import assert from 'node:assert/strict'

import { buildLeaderboardPodiumSlots } from '../utils/ranking-podium.js'

test('buildLeaderboardPodiumSlots keeps three fixed podium slots when fewer than three players exist', () => {
  const leaderboard = [
    { user_id: 101, nickname: '冠军', rank: 1, rank_score: 88 },
    { user_id: 102, nickname: '亚军', rank: 2, rank_score: 55 }
  ]

  assert.deepEqual(buildLeaderboardPodiumSlots(leaderboard), [
    {
      key: 'second',
      placement: 'second',
      rank: 2,
      user: { user_id: 102, nickname: '亚军', rank: 2, rank_score: 55 }
    },
    {
      key: 'first',
      placement: 'first',
      rank: 1,
      user: { user_id: 101, nickname: '冠军', rank: 1, rank_score: 88 }
    },
    {
      key: 'third',
      placement: 'third',
      rank: 3,
      user: null
    }
  ])
})

test('buildLeaderboardPodiumSlots pads missing entries even when only the champion exists', () => {
  const leaderboard = [
    { user_id: 201, nickname: '第一名', rank: 1, rank_score: 120 }
  ]

  const slots = buildLeaderboardPodiumSlots(leaderboard)

  assert.equal(slots.length, 3)
  assert.equal(slots[0].user, null)
  assert.equal(slots[1].user?.nickname, '第一名')
  assert.equal(slots[2].user, null)
})
