import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildDeleteFriendPayload,
  buildFriendH2HUrl,
  buildFriendPkReportUrl,
  normalizeFriendListItem,
  resolveFriendUserId
} from '../utils/friend-entry.js'

test('resolveFriendUserId prefers backend friend list user_id', () => {
  assert.equal(resolveFriendUserId({ user_id: 18, friend_id: 9, id: 1 }), 18)
})

test('normalizeFriendListItem exposes one canonical friend_user_id field', () => {
  assert.deepEqual(normalizeFriendListItem({
    id: 3,
    user_id: 28,
    nickname: '球友 A',
    avatar: 'https://img.example/a.png'
  }), {
    id: 3,
    user_id: 28,
    nickname: '球友 A',
    avatar: 'https://img.example/a.png',
    friend_user_id: 28
  })
})

test('buildFriendH2HUrl carries the canonical opponent id and display name', () => {
  assert.equal(
    buildFriendH2HUrl({
      user_id: 52,
      nickname: '球友 B'
    }),
    '/subPages/user/h2hRecord?opponent_id=52&opponent_name=%E7%90%83%E5%8F%8B%20B'
  )
})

test('buildFriendPkReportUrl keeps avatar, name, and canonical user id aligned', () => {
  assert.equal(
    buildFriendPkReportUrl({
      user_id: 99,
      nickname: '球友 C',
      avatar: 'https://img.example/c c.png'
    }),
    '/subPages/social/pkReport?opponent_id=99&opponent_name=%E7%90%83%E5%8F%8B%20C&opponent_avatar=https%3A%2F%2Fimg.example%2Fc%20c.png'
  )
})

test('buildDeleteFriendPayload uses friend_user_id expected by backend contract', () => {
  assert.deepEqual(buildDeleteFriendPayload({ user_id: 77 }), { friend_user_id: 77 })
})
