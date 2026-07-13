import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../components/bindPhone.vue', import.meta.url),
  'utf8'
)
const settingsSource = readFileSync(
  new URL('../subPages/user/settings.vue', import.meta.url),
  'utf8'
)

test('bind phone requires agreement by default but can reuse prior login consent', () => {
  assert.match(source, /requireAgreement:\s*\{\s*type:\s*Boolean,\s*default:\s*true/)
  assert.match(source, /v-if="requireAgreement"[^>]*class="agreement-section"/)
  assert.match(source, /canAuthorizeWechatPhone\(\)\s*\{[\s\S]*?return !this\.requireAgreement \|\| this\.isAgreed/)
  assert.match(source, /:disabled="!canAuthorizeWechatPhone \|\| loading"/)
})

test('settings keeps the default agreement requirement for proactive binding', () => {
  assert.doesNotMatch(settingsSource, /require-agreement/)
})

test('denied WeChat phone authorization keeps the sheet usable and explains the later option', () => {
  assert.match(source, /未授权手机号，可稍后绑定/)
  assert.match(source, /if \(!code\) \{[\s\S]*?uni\.showToast[\s\S]*?return/)
})

test('bind phone sheet matches the approved ivory and warm dark visual', () => {
  assert.match(source, /\$light-sheet-bg:\s*#fffdf9;/)
  assert.match(source, /\$dark-sheet-bg:\s*#1b170f;/)
  assert.match(source, /用于账号找回、重要赛事通知和已有账号安全合并/)
  assert.match(source, /class="binding-benefits"/)
  assert.match(source, /账号找回[\s\S]*?赛事通知[\s\S]*?账号合并/)
})

test('bind phone button keeps an explicit active style in dark mode', () => {
  assert.match(
    source,
    /&\.dark-mode\s*\{[\s\S]*?\.bind-btn\s*\{[\s\S]*?&\.active\s*\{/,
    'dark mode should provide an active button style override'
  )
})

test('bind phone helper copy stays readable in dark mode', () => {
  assert.match(
    source,
    /&\.dark-mode\s*\{[\s\S]*?\.wechat-phone-copy\s*\{[\s\S]*?color:\s*\$dark-text-secondary;/,
    'dark mode should override the WeChat phone helper copy color'
  )
})

test('bind phone button uses explicit centering styles for its label', () => {
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?display:\s*flex;/)
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?align-items:\s*center;/)
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?justify-content:\s*center;/)
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?padding:\s*0;/)
})

test('bind phone agreement reuses the entry funnel agreement session state', () => {
  assert.match(source, /resolveEntryFunnelAgreementState/)
  assert.match(source, /ENTRY_FUNNEL_AGREEMENT_KEY/)
  assert.match(source, /syncAgreementState\(\)/)
})

test('bind phone sheet exposes a clear later action when it is closable', () => {
  assert.match(source, /close-text-btn/)
  assert.match(source, /稍后绑定/)
  assert.match(source, /:disabled="!canClose"/)
  assert.match(source, /canClose\(\)\s*\{[\s\S]*?canCloseBindPhone/)
  assert.match(source, /handleClose\(\)\s*\{[\s\S]*?if \(!this\.canClose\) return/)
})

test('bind phone agreement styles stay aligned with the login funnel gold theme', () => {
  assert.match(source, /\.checkbox\s*\{[\s\S]*?border:\s*2rpx solid rgba\(224, 174, 18, 0\.18\);/)
  assert.match(source, /\.checkbox\s*\{[\s\S]*?&\.checked\s*\{[\s\S]*?background:\s*#f7d86a;/)
  assert.match(source, /\.agreement-text\s*\{[\s\S]*?color:\s*#6e6242;/)
  assert.match(source, /\.agreement-links\s*\{[\s\S]*?color:\s*#9a8c67;/)
  assert.match(source, /\.agreement-links\s*\{[\s\S]*?\.link\s*\{[\s\S]*?color:\s*#8a6510;/)
  assert.match(source, /class="separator"/)
})

test('bind phone theme follows an explicit incoming prop instead of forcing dark mode', () => {
  assert.match(source, /isDarkMode:\s*\{\s*type:\s*Boolean/)
  assert.doesNotMatch(source, /isDarkMode:\s*true/)
})

test('bind phone uses native WeChat phone authorization only for mini programs', () => {
  assert.match(source, /#ifdef MP-WEIXIN/)
  assert.match(source, /open-type="getPhoneNumber"/)
  assert.match(source, /@getphonenumber="handleWechatPhoneNumber"/)
  assert.match(source, /wechatMiniBindPhone/)
  assert.match(source, /sessionReplaced/)
})

const loginSource = readFileSync(
  new URL('../pages/login/login.vue', import.meta.url),
  'utf8'
)

test('login page allows bind phone modal to close later and passes current theme through', () => {
  assert.match(loginSource, /:closable="true"/)
  assert.match(loginSource, /:is-dark-mode="isDarkMode"/)
  assert.match(loginSource, /@close="handleBindPhoneClose"/)
})

test('login submit button explicitly centers its text vertically', () => {
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?display:\s*flex;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?align-items:\s*center;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?justify-content:\s*center;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?padding:\s*0;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?margin:\s*0;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?height:\s*96rpx;[\s\S]*?line-height:\s*96rpx;/)
})
