import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, parse } from '@vue/compiler-sfc'

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

test('settings page exposes data export and irreversible account deletion flows', () => {
  assert.match(settingsSource, /导出个人数据/)
  assert.match(settingsSource, /handleExportPersonalData/)
  assert.match(settingsSource, /exportPersonalDataToWriter/)
  assert.match(settingsSource, /getPersonalDataExportCapability/)
  assert.match(settingsSource, /注销账号/)
  assert.match(settingsSource, /我已知晓，继续/)
  assert.match(settingsSource, /sendSms\(deletePhone\.value, 'delete_account'\)/)
  assert.match(settingsSource, /verify_type: 'sms'/)
  assert.match(settingsSource, /verify_type: 'oauth'/)
  assert.match(settingsSource, /weixin_mini_program/)
  assert.match(settingsSource, /userStore\.logout\(\)/)
})

test('settings page guards repeated lifecycle actions and remains a valid SFC', () => {
  assert.match(settingsSource, /if \(exportInProgress\.value\) return/)
  assert.match(settingsSource, /if \(deleteAccountSubmitting\.value\) return/)
  assert.match(settingsSource, /closeDeleteAccountDialog\(\{ force: true \}\)/)

  const { descriptor, errors } = parse(settingsSource, { filename: 'settings.vue' })
  assert.deepEqual(errors, [])
  assert.doesNotThrow(() => compileScript(descriptor, { id: 'settings-page-test' }))
})
