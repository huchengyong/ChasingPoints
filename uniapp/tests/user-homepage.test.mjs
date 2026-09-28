import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveMemberRankAvatarFrame,
  resolveRankAvatarFramePath
} from '../utils/user-homepage.js'

const now = new Date('2026-07-20T12:00:00+08:00')
const activeMember = {
  is_active: true,
  member_expires_at: '2026-08-01 12:00:00'
}

test('resolveRankAvatarFramePath maps all six rank levels to unique local assets', () => {
  assert.deepEqual(
    [1, 2, 3, 4, 5, 6].map(resolveRankAvatarFramePath),
    [
      '/static/images/avatar-frames/frame_bronze.png',
      '/static/images/avatar-frames/frame_silver.png',
      '/static/images/avatar-frames/frame_gold.png',
      '/static/images/avatar-frames/frame_platinum.png',
      '/static/images/avatar-frames/frame_diamond.png',
      '/static/images/avatar-frames/frame_king.png'
    ]
  )
  assert.equal(resolveRankAvatarFramePath(0), '')
  assert.equal(resolveRankAvatarFramePath(7), '')
})

test('resolveMemberRankAvatarFrame shows the current rank frame for an active member', () => {
  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: { level: 5 },
    now
  }), '/static/images/avatar-frames/frame_diamond.png')
})

test('resolveMemberRankAvatarFrame hides frames for ordinary, expired, and failed member states', () => {
  const states = [
    null,
    { is_active: false, member_expires_at: '2026-08-01 12:00:00' },
    { is_active: true, member_expires_at: '2026-07-20 12:00:00' },
    { is_active: true, member_expires_at: 'invalid' }
  ]

  for (const memberStatus of states) {
    assert.equal(resolveMemberRankAvatarFrame({
      isLoggedIn: true,
      memberStatus,
      rankInfo: { level: 6 },
      now
    }), '')
  }
})

test('resolveMemberRankAvatarFrame waits for the real rank and hides stale switch data', () => {
  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: null,
    rankLoading: true,
    now
  }), '')

  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: { level: 5 },
    rankLoading: true,
    now
  }), '')

  assert.equal(resolveMemberRankAvatarFrame({
    isLoggedIn: true,
    memberStatus: activeMember,
    rankInfo: null,
    rankLoading: false,
    now
  }), '')
})
