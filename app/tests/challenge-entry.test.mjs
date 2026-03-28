import test from 'node:test'
import assert from 'node:assert/strict'

import { buildChallengePayload, normalizeChallengeListItem } from '../utils/challenge-entry.js'

test('buildChallengePayload sends to_user_id based on canonical friend user id', () => {
  assert.deepEqual(
    buildChallengePayload({
      targetFriend: { user_id: 88, friend_id: 11, id: 3 },
      gameType: 2,
      message: '今晚来一局'
    }),
    {
      to_user_id: 88,
      game_type: 2,
      message: '今晚来一局'
    }
  )
})

test('buildChallengePayload trims invalid target ids to 0 for caller-side guarding', () => {
  assert.deepEqual(
    buildChallengePayload({
      targetFriend: {},
      gameType: 1,
      message: ''
    }),
    {
      to_user_id: 0,
      game_type: 1,
      message: ''
    }
  )
})

test('normalizeChallengeListItem maps sent challenges to the target user fields', () => {
  assert.deepEqual(
    normalizeChallengeListItem(
      {
        id: 1,
        from_user_id: 18,
        from_nickname: '我',
        from_avatar: 'https://img.example/me.png',
        to_user_id: 29,
        to_nickname: '球友A',
        to_avatar: 'https://img.example/a.png'
      },
      18
    ),
    {
      id: 1,
      from_user_id: 18,
      from_nickname: '我',
      from_avatar: 'https://img.example/me.png',
      to_user_id: 29,
      to_nickname: '球友A',
      to_avatar: 'https://img.example/a.png',
      direction: 'sent',
      opponent_id: 29,
      nickname: '球友A',
      avatar: 'https://img.example/a.png'
    }
  )
})

test('normalizeChallengeListItem maps received challenges to the challenger fields', () => {
  assert.deepEqual(
    normalizeChallengeListItem(
      {
        id: 2,
        from_user_id: 41,
        from_nickname: '球友B',
        from_avatar: 'https://img.example/b.png',
        to_user_id: 18,
        to_nickname: '我',
        to_avatar: 'https://img.example/me.png'
      },
      18
    ),
    {
      id: 2,
      from_user_id: 41,
      from_nickname: '球友B',
      from_avatar: 'https://img.example/b.png',
      to_user_id: 18,
      to_nickname: '我',
      to_avatar: 'https://img.example/me.png',
      direction: 'received',
      opponent_id: 41,
      nickname: '球友B',
      avatar: 'https://img.example/b.png'
    }
  )
})
