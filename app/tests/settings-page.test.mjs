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

test('settings page adds a bottom logout action', () => {
  assert.match(settingsSource, /handleLogout/)
  assert.match(settingsSource, /退出登录/)
})

test('settings page dark theme uses the current project gold palette', () => {
  assert.match(settingsScssSource, /\$bg-dark:\s*#141109;/)
  assert.match(settingsScssSource, /\$card-bg-dark:\s*#1e180d;/)
  assert.match(settingsScssSource, /\$text-primary-dark:\s*#fff7e1;/)
  assert.match(settingsScssSource, /\$primary-color:\s*#E0AE12;/)
})
