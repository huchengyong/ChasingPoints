import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const pageSource = readFileSync(
  new URL('../subPages/user/notification.vue', import.meta.url),
  'utf8'
)

const apiSource = readFileSync(
  new URL('../api/notification.js', import.meta.url),
  'utf8'
)

test('notification settings page aligns switches with five real business notification types', () => {
  assert.match(pageSource, /对局结果/)
  assert.match(pageSource, /好友申请/)
  assert.match(pageSource, /约球提醒/)
  assert.match(pageSource, /赛事报名提醒/)
  assert.match(pageSource, /新关注提醒/)
  assert.match(pageSource, /match_result_enabled/)
  assert.match(pageSource, /friend_request_enabled/)
  assert.match(pageSource, /challenge_enabled/)
  assert.match(pageSource, /tournament_enabled/)
  assert.match(pageSource, /follow_enabled/)
})

test('notification settings hides the follow row without dropping its stored value', () => {
  assert.match(pageSource, /canShowSocialPublishing/)
  assert.match(pageSource, /<view v-if="showFollowPreference" class="settings-item">/)
  // 隐藏只影响展示，保存仍带原值。
  assert.match(pageSource, /follow_enabled:\s*preferences\.follow_enabled/)
})

test('notification settings page reads and saves server-backed preferences instead of local-only storage', () => {
  assert.match(pageSource, /getNotificationPreferences/)
  assert.match(pageSource, /saveNotificationPreferences/)
  assert.doesNotMatch(pageSource, /notification_settings/)
  assert.match(apiSource, /export const getNotificationPreferences = \(\) =>/)
  assert.match(apiSource, /\/api\/notification\/preferences/)
  assert.match(apiSource, /export const saveNotificationPreferences = \(data\) =>/)
})
