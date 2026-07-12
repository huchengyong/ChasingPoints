import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  canAttemptWechatMiniLogin,
  resolveWelcomeActions
} from '../utils/entry-funnel.js'
import {
  getWechatPhoneNumberCode,
  resolveBindPhoneSuccess
} from '../utils/bind-phone-flow.js'

const authApiSource = readFileSync(new URL('../api/auth.js', import.meta.url), 'utf8')
const welcomeSource = readFileSync(new URL('../pages/welcome/index.vue', import.meta.url), 'utf8')
const loginSource = readFileSync(new URL('../pages/login/login.vue', import.meta.url), 'utf8')

test('WeChat mini login only starts after agreement while idle', () => {
  assert.equal(canAttemptWechatMiniLogin({ isAgreed: false, isLogging: false }), false)
  assert.equal(canAttemptWechatMiniLogin({ isAgreed: true, isLogging: true }), false)
  assert.equal(canAttemptWechatMiniLogin({ isAgreed: true, isLogging: false }), true)
})

test('welcome actions make WeChat the primary entry for mini programs', () => {
  const actions = resolveWelcomeActions({
    isHarmony: false,
    isWechatMini: true,
    isAgreed: true,
    isLogging: false
  })

  assert.equal(actions.primaryText, '微信一键进入')
  assert.equal(actions.showPhoneLogin, true)
  assert.equal(actions.showHuaweiLogin, false)
  assert.equal(actions.tertiaryText, '先逛逛')
})

test('phone-number authorization extracts a non-empty WeChat code only', () => {
  assert.equal(getWechatPhoneNumberCode({ detail: { code: 'phone-code' } }), 'phone-code')
  assert.equal(getWechatPhoneNumberCode({ detail: { code: '' } }), '')
  assert.equal(getWechatPhoneNumberCode({ detail: { errMsg: 'getPhoneNumber:fail user deny' } }), '')
})

test('merged mini-program binding replaces the current session instead of requiring relogin', () => {
  const result = resolveBindPhoneSuccess({
    response: {
      success: true,
      merged_account: true,
      access_token: 'replacement-access-token',
      refresh_token: 'replacement-refresh-token',
      user_info: { phone: '138****8000' },
      message: '账号已合并'
    },
    phone: ''
  })

  assert.equal(result.action, 'complete')
  assert.equal(result.sessionReplaced, true)
  assert.equal(result.maskedPhone, '138****8000')
})

test('auth api exposes dedicated WeChat mini-program endpoints', () => {
  assert.match(authApiSource, /wechat-mini-login/)
  assert.match(authApiSource, /wechat-mini-bind-phone/)
  assert.doesNotMatch(authApiSource, /provider\s*:\s*['"]weixin_mini_program/)
})

test('welcome page gives mini programs a WeChat primary action while keeping browse and phone fallback', () => {
  assert.match(welcomeSource, /<!-- #ifdef MP-WEIXIN -->\s*<image class="hero-logo" src="\/static\/logo\.png"/)
  assert.match(welcomeSource, /isWechatMiniProgram/)
  assert.match(welcomeSource, /welcomeActions\.showPhoneLogin/)
  assert.match(welcomeSource, /handleWechatMiniLogin/)
  assert.match(welcomeSource, /uni\.login/)
  assert.match(welcomeSource, /handleBrowse/)
})

test('login page supports WeChat mini-program login without auto-opening phone binding', () => {
  assert.match(loginSource, /isWechatMiniProgram/)
  assert.match(loginSource, /handleWechatMiniLogin/)
  assert.match(loginSource, /wechatMiniLogin/)
  assert.match(loginSource, /action === 'wechat-mini'/)
})
