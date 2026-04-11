import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  AVATAR_IMAGE_MAX_EDGE,
  AVATAR_IMAGE_QUALITY,
  buildQiniuAvatarObjectKey,
  formatSettingsPhone,
  getAvatarCompressionDimensions,
  normalizeQiniuUploadResult
} from '../utils/settings-profile.js'

const settingsSource = readFileSync(
  new URL('../subPages/user/settings.vue', import.meta.url),
  'utf8'
)
const settingsProfileSource = readFileSync(
  new URL('../utils/settings-profile.js', import.meta.url),
  'utf8'
)

test('formatSettingsPhone keeps +86 prefix and masked mainland number', () => {
  assert.equal(formatSettingsPhone('188****5657'), '+86 188****5657')
  assert.equal(formatSettingsPhone('13800138000'), '+86 138****8000')
  assert.equal(formatSettingsPhone(''), '')
})

test('buildQiniuAvatarObjectKey scopes avatar uploads by user and date', () => {
  assert.equal(
    buildQiniuAvatarObjectKey({
      userId: 18,
      filePath: '/tmp/avatar.png',
      timestamp: new Date('2026-04-09T16:20:00+08:00').getTime()
    }).startsWith('avatars/18/20260409/'),
    true
  )
})

test('normalizeQiniuUploadResult joins domain and key into a public avatar url', () => {
  assert.equal(
    normalizeQiniuUploadResult({
      domain: 'https://cdn.example.com/',
      key: 'avatars/18/20260409/a.png'
    }),
    'https://cdn.example.com/avatars/18/20260409/a.png'
  )
})

test('avatar compression keeps the longest edge within 512 while preserving aspect ratio', () => {
  assert.equal(AVATAR_IMAGE_MAX_EDGE, 512)
  assert.equal(AVATAR_IMAGE_QUALITY, 80)

  assert.deepEqual(
    getAvatarCompressionDimensions({ width: 2400, height: 1200 }),
    { width: 512, height: 256 }
  )

  assert.deepEqual(
    getAvatarCompressionDimensions({ width: 1200, height: 2400 }),
    { width: 256, height: 512 }
  )

  assert.deepEqual(
    getAvatarCompressionDimensions({ width: 400, height: 400 }),
    { width: 400, height: 400 }
  )
})

test('settings page exposes avatar, nickname, phone, and image picker entry points', () => {
  assert.match(settingsSource, /头像/)
  assert.match(settingsSource, /用户昵称/)
  assert.match(settingsSource, /手机号/)
  assert.match(settingsSource, /chooseAvatarSource/)
  assert.match(settingsSource, /pickAvatarFromCamera/)
  assert.match(settingsSource, /pickAvatarFromAlbum/)
  assert.match(settingsSource, /pickAvatarImage\(\['camera'\]\)/)
  assert.match(settingsSource, /pickAvatarImage\(\['album'\]\)/)
  assert.match(settingsSource, /prepareAvatarForUpload/)
  assert.match(settingsProfileSource, /uniApi\.compressImage/)
  assert.match(settingsSource, /getQiniuUploadToken/)
  assert.match(settingsSource, /updateUserProfile/)
})

test('settings page limits nickname input to 12 characters for profile layout safety', () => {
  assert.match(settingsSource, /maxlength="12"/)
  assert.match(settingsSource, /nickname\.length < 2 \|\| nickname\.length > 12/)
  assert.match(settingsSource, /昵称长度需要在2-12个字符之间/)
})
