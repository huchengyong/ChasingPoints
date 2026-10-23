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
const agreementSheetSource = readFileSync(new URL('../components/agreementConsentSheet.vue', import.meta.url), 'utf8')

function extractSourceBetween(source, startMarker, endMarker) {
	const start = source.indexOf(startMarker)
	const end = source.indexOf(endMarker, start)
	return start >= 0 && end > start ? source.slice(start, end) : ''
}

function extractReloginBranch(source) {
	return source.match(/if \(payload\?\.action === 'relogin'\) \{[\s\S]*?\n\t\}/)?.[0] || ''
}

const loginBindPhoneSuccessSource = extractSourceBetween(
	loginSource,
	'const handleBindPhoneSuccess',
	'const handleBindPhoneClose'
)
const loginReloginBranchSource = extractReloginBranch(loginBindPhoneSuccessSource)
const loginNormalBindSuccessSource = loginBindPhoneSuccessSource.replace(loginReloginBranchSource, '')

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

test('auth api sends the phone code with the WeChat login code and keeps the legacy shape without it', () => {
  assert.match(authApiSource, /wechatMiniLogin = \(code, phoneCode = ''\)/)
  assert.match(authApiSource, /phoneCode \? \{ code, phone_code: phoneCode \} : \{ code \}/)
  assert.match(authApiSource, /post\('\/api\/auth\/wechat-mini-login', data, \{ silent: true \}\)/)
})

test('agreement sheet can turn its confirm button into the WeChat phone authorization button', () => {
  assert.match(agreementSheetSource, /:open-type="phoneAuthorize \? 'getPhoneNumber' : ''"/)
  assert.match(agreementSheetSource, /@getphonenumber="\$emit\('phone-number', \$event\)"/)
  assert.match(agreementSheetSource, /phoneAuthorize:\s*\{\s*type:\s*Boolean,\s*default:\s*false/)
  assert.match(agreementSheetSource, /defineEmits\(\['close', 'agree', 'phone-number'/)
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
	assert.match(welcomeSource, /handleWechatPhoneNumber/)
	assert.match(welcomeSource, /uni\.login/)
	assert.match(welcomeSource, /handleBrowse/)
	assert.match(welcomeSource, /url: '\/pages\/login\/login\?method=phone'/)
})

test('welcome page logs in with the WeChat phone authorization and never opens a bind sheet', () => {
	assert.match(welcomeSource, /const authGate = computed/)
	assert.match(welcomeSource, /class="phone-login-btn"[\s\S]*?:disabled="authGate\.isAuthenticating"/)
	assert.match(welcomeSource, /class="btn-tertiary"[\s\S]*?:disabled="authGate\.isAuthenticating"/)
	assert.match(welcomeSource, /if \(!authGate\.value\.canLeave\) return/)
	assert.match(welcomeSource, /if \(!authGate\.value\.shouldHandleResult\) return/)
	assert.match(welcomeSource, /:open-type="wechatPhoneOpenType"/)
	assert.match(welcomeSource, /@getphonenumber="handleWechatPhoneNumber"/)
	assert.match(welcomeSource, /isWechatMiniProgram && isAgreed\.value \? 'getPhoneNumber' : ''/)
	assert.match(welcomeSource, /wechatMiniLogin\(loginResult\.code, phoneCode\)/)
	assert.match(welcomeSource, /getWechatPhoneNumberCode\(event\)[\s\S]*?resolveWechatPhoneAuthMessage\(event\)/)
	assert.match(welcomeSource, /resolveAuthErrorMessage\(error, '微信登录失败，请重试'\)/)
	assert.match(welcomeSource, /userStore\.login\(response\)[\s\S]*?navigateToHome\(\)/)
	assert.doesNotMatch(welcomeSource, /bindPhone|showBindPhoneModal|resolveWechatPostLoginState|need_bind_phone/)
})

test('welcome page keeps the agreement sheet open until the phone authorization returns', () => {
	assert.match(welcomeSource, /:phone-authorize="isWechatPhoneAuthorizing"/)
	assert.match(welcomeSource, /@phone-number="handleWechatPhoneNumber"/)
	assert.match(welcomeSource, /if \(isWechatPhoneAuthorizing\.value\) return\n\tcloseAgreementSheet\(\)/)
	assert.match(welcomeSource, /handleWechatPhoneNumber[\s\S]*?closeAgreementSheet\(\)/)
})

test('welcome page matches the ivory light and warm dark login visual', () => {
	assert.match(welcomeStyleSource, /linear-gradient\(180deg, #fffdf8 0%, #f8f3e9 100%\)/)
	assert.match(welcomeStyleSource, /&\.dark-mode\s*\{[\s\S]*?linear-gradient\(180deg, #1b170f 0%, #12100b 100%\)/)
	assert.doesNotMatch(welcomeSource, /bg-image|action-panel/)
})

test('welcome page labels the hard-coded preview card as sample data', () => {
	assert.match(welcomeSource, /<view class="preview-card"[^>]*>\s*<text class="preview-badge">示例数据<\/text>/)
})

test('login page shares the welcome brand row and single-line agreement layout', () => {
	const brandRowStyle = /class="brand-row" :style="\{ marginTop: `\$\{headerLayout\.top\}px`, minHeight: `\$\{headerLayout\.minHeight\}px` \}"/
	assert.match(welcomeSource, brandRowStyle)
	assert.match(loginSource, brandRowStyle)
	assert.match(loginSource, /const headerLayout = resolveWelcomeHeaderLayout\(\{[\s\S]*?menuButton: menuButtonRect/)
	for (const source of [welcomeStyleSource, loginSource]) {
		assert.match(source, /\.brand-logo \{\s*width: 84rpx;\s*height: 84rpx;/)
		assert.match(source, /\.brand-name \{[^}]*font-size: 36rpx;/)
		assert.match(source, /\.brand-en \{[^}]*font-size: 22rpx;/)
		assert.match(source, /\.agreement-row \{\s*padding: 24rpx 0;/)
	}
	assert.match(loginSource, /\.agreement-block \{\s*display: flex;\s*flex-wrap: wrap;/)
})

test('login page renders one login mode at a time and honors the phone route parameter', () => {
  assert.match(loginSource, /const authMode = ref/)
  assert.match(loginSource, /onLoad\(\(options\) => \{[\s\S]*?resolveLoginMode\(\{[\s\S]*?requestedMethod: options\?\.method/)
  assert.match(loginSource, /v-if="authMode === 'wechat'"[^>]*class="auth-mode wechat-mode"/)
  assert.match(loginSource, /v-else[^>]*class="auth-mode phone-mode"/)
  assert.match(loginSource, /resolveAlternateLoginMode/)
  assert.doesNotMatch(loginSource, /class="form-card"/)
})

test('login page logs WeChat users in with one phone-authorized request and keeps the bind sheet for other providers', () => {
  assert.match(loginSource, /const authGate = computed/)
  assert.match(loginSource, /switchLoginMode[\s\S]*?if \(!authGate\.value\.canLeave\) return/)
  assert.match(loginSource, /isLogging:\s*authGate\.value\.isAuthenticating/)
  assert.match(loginSource, /if \(!authGate\.value\.shouldHandleResult\) return/)
  assert.match(loginSource, /:open-type="wechatPhoneOpenType"/)
  assert.match(loginSource, /@getphonenumber="handleWechatPhoneNumber"/)
  assert.match(loginSource, /wechatMiniLogin\(loginResult\.code, phoneCode\)/)
  assert.match(loginSource, /:phone-authorize="isWechatPhoneAuthorizing"/)
  assert.match(loginSource, /@phone-number="handleWechatPhoneNumber"/)
  assert.match(loginSource, /if \(isWechatPhoneAuthorizing\.value\) return\n\tcloseAgreementSheet\(\)/)
  const wechatHandlerSource = extractSourceBetween(loginSource, 'const handleWechatPhoneNumber', 'const handleHuaweiLogin')
  assert.doesNotMatch(wechatHandlerSource, /showBindPhoneModal|resolveWechatPostLoginState/)
  assert.match(loginSource, /loginResult\.need_bind_phone[\s\S]*?showBindPhoneModal\.value = true/)
  assert.match(loginSource, /:use-sms-binding="true"/)
  assert.match(loginSource, /:require-agreement="false"/)
  assert.match(loginSource, /bindingSkipped:\s*true/)
})

test('login page keeps merged accounts on a prefilled phone login state', () => {
	assert.match(loginReloginBranchSource, /userStore\.logout\(\)/)
	assert.match(loginReloginBranchSource, /authMode\.value = 'phone'/)
	assert.match(loginReloginBranchSource, /formData\.phone = payload\?\.phone \|\| ''/)
	assert.match(loginReloginBranchSource, /formData\.code = ''/)
	assert.match(loginReloginBranchSource, /\n\t\treturn\n\t\}$/)
	assert.doesNotMatch(loginReloginBranchSource, /navigateAfterLogin\(\)/)
	assert.match(loginNormalBindSuccessSource, /navigateAfterLogin\(\)/)
})

test('login page uses ivory light styling and a matching warm dark theme', () => {
  assert.match(loginSource, /linear-gradient\(180deg, #fffdf8 0%, #f8f3e9 100%\)/)
  assert.match(loginSource, /&\.dark-mode\s*\{[\s\S]*?linear-gradient\(180deg, #1b170f 0%, #12100b 100%\)/)
  assert.doesNotMatch(loginSource, /ambient-glow|header-badge/)
})
