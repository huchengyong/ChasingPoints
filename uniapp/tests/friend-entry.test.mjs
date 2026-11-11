import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildBlacklistFriendPayload,
  buildDeleteFriendPayload,
  buildFriendH2HUrl,
  buildFriendHomepageUrl,
  buildFriendOpponentRecordUrl,
  buildFriendPkReportUrl,
  buildUnblacklistConfirmCopy,
  buildUnblacklistPayload,
  normalizeFriendListItem,
  normalizeSearchResultItem,
  removeBlacklistItem,
  resolveFriendRequestUserId,
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

test('buildFriendOpponentRecordUrl carries target user context for friend battle history', () => {
  assert.equal(
    buildFriendOpponentRecordUrl({
      user_id: 52,
      nickname: '球友 B',
      avatar: 'https://img.example/b.png'
    }),
    '/subPages/user/opponentRecord?target_user_id=52&target_name=%E7%90%83%E5%8F%8B%20B&target_avatar=https%3A%2F%2Fimg.example%2Fb.png'
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

test('buildFriendHomepageUrl carries identity fields needed by the lite homepage', () => {
  assert.equal(
    buildFriendHomepageUrl({
      user_id: 66,
      nickname: '球友 D',
      avatar: 'https://img.example/d.png',
      rank_name: '星耀'
    }),
    '/subPages/social/friendHomepage?opponent_id=66&opponent_name=%E7%90%83%E5%8F%8B%20D&opponent_avatar=https%3A%2F%2Fimg.example%2Fd.png&rank_name=%E6%98%9F%E8%80%80'
  )
})

test('buildFriendHomepageUrl carries vip_level and rank_level only for active members', () => {
  const member = buildFriendHomepageUrl({ user_id: 66, nickname: 'A', avatar: '', rank_name: '星耀', vip_level: 3, rank_level: 4 })
  assert.match(member, /&vip_level=3&rank_level=4$/)

  for (const vip_level of [0, undefined, null, 'x']) {
    assert.doesNotMatch(buildFriendHomepageUrl({ user_id: 66, vip_level, rank_level: 4 }), /vip_level|rank_level/)
  }
})

test('buildDeleteFriendPayload uses friend_user_id expected by backend contract', () => {
  assert.deepEqual(buildDeleteFriendPayload({ user_id: 77 }), { friend_user_id: 77 })
})

test('buildBlacklistFriendPayload uses friend_user_id expected by backend contract', () => {
  assert.deepEqual(buildBlacklistFriendPayload({ friend_id: 88 }), { friend_user_id: 88 })
})

test('normalizeSearchResultItem maps has_pending_request to the existing pending status', () => {
  assert.equal(normalizeSearchResultItem({ user_id: 1, status: 'friend', has_pending_request: true }).status, 'friend')
  assert.equal(normalizeSearchResultItem({ user_id: 2, has_pending_request: true }).status, 'pending')
  assert.equal(normalizeSearchResultItem({ user_id: 3, status: 'pending' }).status, 'pending')
  assert.equal(normalizeSearchResultItem({ user_id: 4 }).status, undefined)
})

test('resolveFriendRequestUserId reads the applicant id from from_user_id first', () => {
  assert.equal(resolveFriendRequestUserId({ id: 9, from_user_id: 52, user_id: 18 }), 52)
  assert.equal(resolveFriendRequestUserId({ id: 9, user_id: 18 }), 18)
})

test('blacklist removal helpers keep the canonical user id and confirmation copy', () => {
  assert.deepEqual(buildUnblacklistPayload({ user_id: 66 }), { friend_user_id: 66 })
  assert.deepEqual(
    removeBlacklistItem([
      { user_id: 66, nickname: 'A' },
      { user_id: 67, nickname: 'B' }
    ], 66),
    [{ user_id: 67, nickname: 'B' }]
  )
  assert.equal(
    buildUnblacklistConfirmCopy('球友 B'),
    '确定将「球友 B」移出黑名单吗？移出后对方可以重新搜索并添加你。'
  )
})
