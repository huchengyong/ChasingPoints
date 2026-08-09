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
const sendCodeMethodSource = source.slice(
  source.indexOf('async handleSendCode()'),
  source.indexOf('async handleBind()')
)
const sendCodeStyleSource = source.slice(
  source.indexOf('\t.send-code-btn {'),
  source.indexOf('\t.error-text {')
)

test('bind phone requires agreement by default but can reuse prior login consent', () => {
  assert.match(source, /requireAgreement:\s*\{\s*type:\s*Boolean,\s*default:\s*true/)
  assert.match(source, /v-if="requireAgreement"[^>]*class="agreement-section"/)
  assert.match(source, /canAuthorizeWechatPhone\(\)\s*\{[\s\S]*?return !this\.requireAgreement \|\| this\.isAgreed/)
  assert.match(source, /:disabled="!canAuthorizeWechatPhone \|\| loading"/)
})

test('bind phone uses native authorization by default and enables an explicit SMS mode', () => {
  assert.match(source, /useSmsBinding:\s*\{\s*type:\s*Boolean,\s*default:\s*false/)
  assert.match(source, /v-if="!useSmsBinding"[\s\S]*?open-type="getPhoneNumber"/)
  assert.match(source, /v-if="showSmsBinding"[^>]*class="sms-binding-form"/)
  assert.match(source, /showSmsBinding\(\)\s*\{[\s\S]*?this\.useSmsBinding/)
  assert.match(source, /class="phone-input"[\s\S]*?class="code-input"[\s\S]*?获取验证码[\s\S]*?立即绑定/)
})

test('bind phone SMS action reports an invalid phone instead of ignoring the tap', () => {
  assert.match(
    source,
    /<button\s+class="send-code-btn"[\s\S]*?:disabled="isSendingCode \|\| countdown > 0"[\s\S]*?@click="handleSendCode"/
  )
  assert.match(sendCodeMethodSource, /if \(!isValidBindPhone\(this\.phone\)\)[\s\S]*?this\.phoneError = this\.phone \? '请输入正确的手机号' : '请输入手机号'/)
  assert.match(sendCodeMethodSource, /uni\.showToast\(\{[\s\S]*?title: this\.phoneError[\s\S]*?icon: 'none'/)
  assert.doesNotMatch(sendCodeMethodSource, /if \(!this\.canSendCode\) return/)
  assert.ok(
    sendCodeMethodSource.indexOf('if (!isValidBindPhone(this.phone))') <
      sendCodeMethodSource.indexOf('this.isSendingCode = true')
  )
})

test('bind phone SMS action does not overlap the native code input hit area', () => {
  assert.match(source, /\.code-input\s*\{[\s\S]*?min-width:\s*0;/)
  assert.match(sendCodeStyleSource, /position:\s*static;/)
  assert.match(sendCodeStyleSource, /flex-shrink:\s*0;/)
  assert.doesNotMatch(sendCodeStyleSource, /position:\s*absolute;/)
  assert.doesNotMatch(sendCodeStyleSource, /transform:\s*translateY/)
})

test('bind phone exposes a slow-request hint without a global loading overlay', () => {
  assert.match(source, /网络稍慢，正在继续尝试/)
  assert.match(source, /AUTH_SLOW_FEEDBACK_DELAY/)
  assert.match(source, /startSlowAction\(/)
  assert.match(source, /clearSlowAction\(/)
})

test('settings keeps the default agreement requirement for proactive binding', () => {
  assert.doesNotMatch(settingsSource, /require-agreement/)
  assert.doesNotMatch(settingsSource, /use-sms-binding/)
  assert.match(source, /#ifdef MP-WEIXIN[\s\S]*?v-if="!useSmsBinding"[\s\S]*?open-type="getPhoneNumber"/)
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

test('bind phone replaces the session atomically when merge returns target user tokens', () => {
  // 短信与微信合并响应都必须走 userStore.login 原子替换 token/refresh/user_info
  assert.match(source, /if \(outcome\.sessionReplaced\) \{/)
  assert.match(source, /userStore\.login\(res\)/)
  assert.match(source, /else if \(outcome\.action === 'complete'\) \{[\s\S]*?userStore\.bindPhoneSuccess\(outcome\.maskedPhone\)/)
})

const loginSource = readFileSync(
  new URL('../pages/login/login.vue', import.meta.url),
  'utf8'
)
const welcomeSource = readFileSync(
  new URL('../pages/welcome/index.vue', import.meta.url),
  'utf8'
)
const editProfileSource = readFileSync(
  new URL('../subPages/user/editProfile.vue', import.meta.url),
  'utf8'
)

test('parent pages keep the compat relogin fallback for merged accounts without tokens', () => {
  for (const pageSource of [loginSource, welcomeSource, settingsSource, editProfileSource]) {
    assert.match(pageSource, /payload\?\.action === 'relogin'/)
    assert.match(pageSource, /userStore\.logout\(\)/)
  }
})

test('settings and edit profile skip phone-only updates when the session was replaced', () => {
  for (const pageSource of [settingsSource, editProfileSource]) {
    assert.match(pageSource, /if \(!payload\?\.sessionReplaced\) \{/)
    assert.match(pageSource, /userStore\.bindPhoneSuccess\(payload\?\.maskedPhone/)
  }
})

test('login and welcome finish the flow after an atomic session replacement', () => {
  assert.match(loginSource, /navigateAfterLogin\(\)/)
  assert.match(welcomeSource, /navigateToHome\(\)/)
})

test('login page allows bind phone modal to close later and passes current theme through', () => {
  assert.match(loginSource, /:closable="true"/)
  assert.match(loginSource, /:is-dark-mode="isDarkMode"/)
  assert.match(loginSource, /@close="handleBindPhoneClose"/)
})

test('login page opens the post-login bind sheet in SMS mode without repeated agreement', () => {
  assert.match(loginSource, /:use-sms-binding="true"/)
  assert.match(loginSource, /:require-agreement="false"/)
  assert.doesNotMatch(loginSource, /:require-agreement="!isWechatMiniProgram"/)
})

test('login submit button explicitly centers its text vertically', () => {
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?display:\s*flex;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?align-items:\s*center;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?justify-content:\s*center;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?padding:\s*0;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?margin:\s*0;/)
	assert.match(loginSource, /\.login-btn\s*\{[\s\S]*?height:\s*96rpx;[\s\S]*?line-height:\s*96rpx;/)
})

test('login funnel gives immediate visible feedback and avoids a global loading overlay', () => {
	assert.match(loginSource, /正在安全登录/)
	assert.match(loginSource, /auth-feedback-card/)
	assert.match(loginSource, /AUTH_SLOW_FEEDBACK_DELAY/)
	assert.match(loginSource, /auth-pulse/)
	assert.doesNotMatch(loginSource, /uni\.showLoading/)
})

test('login pages keep a visible inline status card for the whole authentication request', () => {
	for (const pageSource of [loginSource, welcomeSource]) {
		assert.match(pageSource, /v-if="authFeedback\.visible"[^>]*class="auth-feedback-card"/)
		assert.match(pageSource, /auth-feedback-progress/)
		assert.match(pageSource, /authFeedback\.message/)
		assert.doesNotMatch(pageSource, /authFeedback\.title/)
		assert.doesNotMatch(pageSource, /authFeedback\.detail/)
	}
})
