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
const welcomeStyleSource = readFileSync(new URL('../pages/welcome/index.scss', import.meta.url), 'utf8')
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
	assert.match(welcomeSource, /<image class="brand-logo" src="\/static\/logo\.png"/)
	assert.match(welcomeSource, /isWechatMiniProgram/)
	assert.match(welcomeSource, /welcomeActions\.showPhoneLogin/)
	assert.match(welcomeSource, /handleWechatMiniLogin/)
	assert.match(welcomeSource, /uni\.login/)
	assert.match(welcomeSource, /handleBrowse/)
	assert.match(welcomeSource, /url: '\/pages\/login\/login\?method=phone'/)
})

test('welcome page delays navigation until an unbound user handles the phone prompt', () => {
	assert.match(welcomeSource, /resolveWechatPostLoginState/)
	assert.match(welcomeSource, /const authGate = computed/)
	assert.match(welcomeSource, /class="phone-login-btn"[\s\S]*?:disabled="authGate\.isAuthenticating"/)
	assert.match(welcomeSource, /class="btn-tertiary"[\s\S]*?:disabled="authGate\.isAuthenticating"/)
	assert.match(welcomeSource, /if \(!authGate\.value\.canLeave\) return/)
	assert.match(welcomeSource, /if \(!authGate\.value\.shouldHandleResult\) return/)
	assert.match(welcomeSource, /postLoginState\.action === 'bind-phone'[\s\S]*?showBindPhoneModal\.value = true[\s\S]*?return[\s\S]*?navigateToHome\(\)/)
	assert.match(welcomeSource, /if \(hasViewed && !showBindPhoneModal\.value\)/)
	assert.match(welcomeSource, /:require-agreement="false"/)
	assert.match(welcomeSource, /bindingSkipped:\s*true/)
	assert.match(welcomeSource, /handleBindPhoneSuccess/)
})

test('welcome page matches the ivory light and warm dark login visual', () => {
	assert.match(welcomeStyleSource, /linear-gradient\(180deg, #fffdf8 0%, #f8f3e9 100%\)/)
	assert.match(welcomeStyleSource, /&\.dark-mode\s*\{[\s\S]*?linear-gradient\(180deg, #1b170f 0%, #12100b 100%\)/)
	assert.doesNotMatch(welcomeSource, /bg-image|action-panel/)
})

test('login page renders one login mode at a time and honors the phone route parameter', () => {
  assert.match(loginSource, /const authMode = ref/)
  assert.match(loginSource, /onLoad\(\(options\) => \{[\s\S]*?resolveLoginMode\(\{[\s\S]*?requestedMethod: options\?\.method/)
  assert.match(loginSource, /v-if="authMode === 'wechat'"[^>]*class="auth-mode wechat-mode"/)
  assert.match(loginSource, /v-else[^>]*class="auth-mode phone-mode"/)
  assert.match(loginSource, /resolveAlternateLoginMode/)
  assert.doesNotMatch(loginSource, /class="form-card"/)
})

test('login page defers navigation for unbound WeChat users and reuses prior consent', () => {
  assert.match(loginSource, /resolveWechatPostLoginState/)
  assert.match(loginSource, /const authGate = computed/)
  assert.match(loginSource, /switchLoginMode[\s\S]*?if \(!authGate\.value\.canLeave\) return/)
  assert.match(loginSource, /isLogging:\s*authGate\.value\.isAuthenticating/)
  assert.match(loginSource, /if \(!authGate\.value\.shouldHandleResult\) return/)
  assert.match(loginSource, /postLoginState\.action === 'bind-phone'[\s\S]*?showBindPhoneModal\.value = true[\s\S]*?return[\s\S]*?navigateAfterLogin\(\)/)
  assert.match(loginSource, /:require-agreement="!isWechatMiniProgram"/)
  assert.match(loginSource, /bindingSkipped:\s*true/)
})

test('login page uses ivory light styling and a matching warm dark theme', () => {
  assert.match(loginSource, /linear-gradient\(180deg, #fffdf8 0%, #f8f3e9 100%\)/)
  assert.match(loginSource, /&\.dark-mode\s*\{[\s\S]*?linear-gradient\(180deg, #1b170f 0%, #12100b 100%\)/)
  assert.doesNotMatch(loginSource, /ambient-glow|header-badge/)
})
