import test from 'node:test'
import assert from 'node:assert/strict'

import { buildOpponentSelectionList } from '../utils/opponent-selector.js'
import { readFileSync } from 'node:fs'

test('opponent selector loads server-deduplicated candidates instead of downloading friends and matches', () => {
  const source = readFileSync(new URL('../subPages/match/opponentSelector.vue', import.meta.url), 'utf8')
  assert.match(source, /getOpponentCandidates\(\{ limit: 30 \}\)/)
  assert.doesNotMatch(source, /getFriendList\(|getMatchList\(|buildOpponentSelectionList/)
})

test('opponent selector prefers friends and keeps recent match order after de-duplication', () => {
  assert.deepEqual(buildOpponentSelectionList({
    friends: [
      { friend_user_id: 20, nickname: '好友A', avatar: 'friend-a.png' },
      { friend_user_id: 20, nickname: '重复好友' }
    ],
    matches: [
      { opponent_id: 30, opponent_name: '最近B', opponent_avatar: 'b.png' },
      { opponent_id: 20, opponent_name: '历史A', opponent_avatar: 'old-a.png' },
      { opponent_id: 0, opponent_name: '无效记录' },
      { opponent_id: 30, opponent_name: '更早B' },
      { opponent_id: 40, opponent_name: '最近C', opponent_avatar: 'c.png' }
    ]
  }), [
    { user_id: 20, nickname: '好友A', avatar: 'friend-a.png', source: 'friend' },
    { user_id: 30, nickname: '最近B', avatar: 'b.png', source: 'recent' },
    { user_id: 40, nickname: '最近C', avatar: 'c.png', source: 'recent' }
  ])
})
