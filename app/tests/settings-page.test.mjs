import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const settingsSource = readFileSync(
  new URL('../subPages/user/settings.vue', import.meta.url),
  'utf8'
)

const settingsScssSource = readFileSync(
  new URL('../subPages/user/settings.scss', import.meta.url),
  'utf8'
)

test('settings page exposes a bind phone entry and reuses the bind phone sheet', () => {
  assert.match(settingsSource, /手机号/)
  assert.match(settingsSource, /<bindPhone/)
  assert.match(settingsSource, /showBindPhoneModal/)
  assert.match(settingsSource, /handleBindPhoneSuccess/)
})

test('settings page keeps a replacement mini-program session instead of logging out', () => {
  assert.match(settingsSource, /sessionReplaced/)
})

test('settings page adds a bottom logout action', () => {
  assert.match(settingsSource, /handleLogout/)
  assert.match(settingsSource, /退出登录/)
})

test('settings page owns hidden match history with optimistic rollback', () => {
  assert.match(settingsSource, /隐藏战绩/)
  assert.match(settingsSource, /getUserPrivacy/)
  assert.match(settingsSource, /updateUserPrivacy/)
  assert.match(settingsSource, /const previousValue = isHideMatch\.value/)
  assert.match(settingsSource, /isHideMatch\.value = previousValue/)
})

test('settings page owns help, complaint, and report navigation', () => {
  assert.match(settingsSource, /帮助、投诉与举报/)
  assert.match(settingsSource, /handleHelp/)
  assert.match(settingsSource, /\/subPages\/help\/feedback/)
})

test('settings page dark theme uses the current project gold palette', () => {
  assert.match(settingsScssSource, /\$bg-dark:\s*#141109;/)
  assert.match(settingsScssSource, /\$card-bg-dark:\s*#1e180d;/)
  assert.match(settingsScssSource, /\$text-primary-dark:\s*#fff7e1;/)
  assert.match(settingsScssSource, /\$primary-color:\s*#E0AE12;/)
})

test('settings page exposes system, light and dark theme modes', () => {
  assert.match(settingsSource, /主题模式/)
  assert.match(settingsSource, /跟随系统/)
  assert.match(settingsSource, /浅色/)
  assert.match(settingsSource, /深色/)
  assert.match(settingsSource, /themeMode/)
  assert.match(settingsSource, /setThemeMode/)
})
