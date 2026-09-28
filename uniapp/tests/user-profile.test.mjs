import test from 'node:test'
import assert from 'node:assert/strict'

import {
  DEFAULT_USER_AVATAR,
  getDefaultAvatarPath,
  resolveAvatarUrl
} from '../utils/user-profile.js'

test('getDefaultAvatarPath maps user ids to stable billiard numbers', () => {
  assert.equal(getDefaultAvatarPath(1), '/static/images/default-avatars/ball-01.png')
  assert.equal(getDefaultAvatarPath(15), '/static/images/default-avatars/ball-15.png')
  assert.equal(getDefaultAvatarPath(16), '/static/images/default-avatars/ball-01.png')
})

test('getDefaultAvatarPath falls back to the first ball for missing ids', () => {
  assert.equal(DEFAULT_USER_AVATAR, '/static/images/default-avatars/ball-01.png')
  assert.equal(getDefaultAvatarPath(), DEFAULT_USER_AVATAR)
  assert.equal(getDefaultAvatarPath('invalid'), DEFAULT_USER_AVATAR)
})

test('resolveAvatarUrl uses the billiard fallback for empty values', () => {
  assert.equal(resolveAvatarUrl('', 8), '/static/images/default-avatars/ball-08.png')
  assert.equal(resolveAvatarUrl(null, 15), '/static/images/default-avatars/ball-15.png')
})

test('resolveAvatarUrl preserves a real avatar URL regardless of user id', () => {
  const avatar = 'https://img.example.com/avatar.png'

  assert.equal(resolveAvatarUrl(avatar, 8), avatar)
})
