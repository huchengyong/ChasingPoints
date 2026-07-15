<template>
	<view class="login-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="login-content">
			<view class="brand-row">
				<image class="brand-logo" src="/static/logo.png" mode="aspectFit"></image>
				<view class="brand-copy">
					<text class="brand-name">追分竞技</text>
					<text class="brand-en">CHASING POINTS</text>
				</view>
			</view>

			<view v-if="authMode === 'wechat'" class="auth-mode wechat-mode">
				<view class="hero-section">
					<text class="eyebrow">台球竞技记录</text>
					<text class="title">每一杆，</text>
					<text class="title">都值得被记录</text>
					<text class="subtitle">记录战绩、生成战报，找到真正旗鼓相当的对手。</text>
					<view class="benefit-list">
						<text>对局记录</text>
						<text>竞技排名</text>
						<text>战报分享</text>
					</view>
				</view>

				<view class="wechat-actions">
					<button
						class="wechat-login-btn"
						:class="{ authenticating: authGate.isAuthenticating }"
						:disabled="authGate.isAuthenticating"
						@click="handleWechatMiniLogin"
					>
						{{ isWechatLogging ? (isSlowLogging ? '网络稍慢，正在继续…' : '正在安全登录…') : '微信一键进入' }}
					</button>
					<button class="phone-login-btn" :disabled="authGate.isAuthenticating" @click="switchLoginMode">手机号登录</button>
					<text v-if="isSlowLogging" class="auth-slow-hint">网络稍慢，正在继续尝试</text>
				</view>

				<view class="agreement-block">
					<view class="agreement-row" :class="{ disabled: authGate.isAuthenticating }" @click="toggleAgreement">
						<view class="checkbox" :class="{ checked: isAgreed }">
							<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#231c0b"></uni-icons>
						</view>
						<text class="agreement-text">我已阅读并同意</text>
					</view>
					<view class="agreement-links" :class="{ disabled: authGate.isAuthenticating }">
						<text class="link" @click.stop="showAgreement('user')">《用户协议》</text>
						<text class="separator">和</text>
						<text class="link" @click.stop="showAgreement('privacy')">《隐私政策》</text>
					</view>
				</view>
			</view>

			<view v-else class="auth-mode phone-mode">
				<text
					v-if="isWechatMiniProgram"
					class="mode-back"
					:class="{ disabled: authGate.isAuthenticating }"
					@click="switchLoginMode"
				>‹ 微信登录</text>

				<view class="phone-heading">
					<text class="phone-title">手机号登录</text>
					<text class="phone-subtitle">未注册手机号验证后将自动创建账号</text>
				</view>

				<view class="form-section">
					<view class="form-item">
						<text class="form-label">手机号</text>
						<view class="input-container" :class="{ invalid: phoneError }">
							<text class="country-code">+86</text>
							<input
								type="tel"
								v-model="formData.phone"
								placeholder="请输入手机号"
								class="form-input"
								maxlength="11"
								:disabled="authGate.isAuthenticating"
							/>
						</view>
						<text v-if="phoneError" class="field-error">{{ phoneError }}</text>
					</view>

					<view class="form-item">
						<text class="form-label">短信验证码</text>
						<view class="input-container" :class="{ invalid: codeError }">
							<input
								type="number"
								v-model="formData.code"
								placeholder="请输入6位验证码"
								class="form-input code-input"
								maxlength="6"
								:disabled="authGate.isAuthenticating"
							/>
							<button class="send-code-btn" :disabled="!canSendCode" @click="handleSendCode">
								{{ sendCodeText }}
							</button>
						</view>
						<text v-if="codeError" class="field-error">{{ codeError }}</text>
					</view>
				</view>

				<view class="agreement-block phone-agreement">
					<view class="agreement-row" @click="toggleAgreement">
						<view class="checkbox" :class="{ checked: isAgreed }">
							<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#231c0b"></uni-icons>
						</view>
						<text class="agreement-text">我已阅读并同意</text>
					</view>
					<view class="agreement-links">
						<text class="link" @click.stop="showAgreement('user')">《用户协议》</text>
						<text class="separator">和</text>
						<text class="link" @click.stop="showAgreement('privacy')">《隐私政策》</text>
					</view>
				</view>

				<button class="login-btn" :class="{ authenticating: authGate.isAuthenticating }" :disabled="!canSubmit" @click="handleLogin">
					{{ isLogging ? (isSlowLogging ? '网络稍慢，正在继续…' : '正在安全登录…') : '登录 / 注册' }}
				</button>
				<text v-if="isSlowLogging" class="auth-slow-hint">网络稍慢，正在继续尝试</text>
				<text
					v-if="isWechatMiniProgram"
					class="mode-switch"
					:class="{ disabled: authGate.isAuthenticating }"
					@click="switchLoginMode"
				>切换到微信一键进入</text>

				<view v-if="showHuaweiLogin" class="third-party-login">
					<view class="divider">
						<view class="divider-line"></view>
						<text class="divider-text">其他可用方式</text>
						<view class="divider-line"></view>
					</view>
					<button class="third-party-btn" :class="{ authenticating: authGate.isAuthenticating }" :disabled="authGate.isAuthenticating" @click="handleHuaweiLogin">
						<image src="/static/images/huawei.svg" mode="aspectFit" />
						<text>{{ isHuaweiLogging ? (isSlowLogging ? '网络稍慢，正在继续…' : '正在安全登录…') : '华为账号登录' }}</text>
					</button>
					<text v-if="isSlowLogging && isHuaweiLogging" class="auth-slow-hint">网络稍慢，正在继续尝试</text>
				</view>
			</view>
		</view>

		<bindPhone
			:show="showBindPhoneModal"
			:closable="true"
			:is-dark-mode="isDarkMode"
			:use-sms-binding="true"
			:require-agreement="false"
			@close="handleBindPhoneClose"
			@success="handleBindPhoneSuccess"
		/>

		<agreementConsentSheet
			:show="showAgreementSheet"
			:is-dark-mode="isDarkMode"
			@close="closeAgreementSheet"
			@agree="handleAgreementAccepted"
			@open-user="showAgreement('user')"
			@open-privacy="showAgreement('privacy')"
		/>
	</view>
</template>

<script setup>
import { computed, onUnmounted, reactive, ref } from 'vue'
import { onLoad, onShow, onUnload } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { sendSms, login, loginByOauth, wechatMiniLogin } from '@/api/auth.js'
import agreementConsentSheet from '@/components/agreementConsentSheet.vue'
import bindPhone from '@/components/bindPhone.vue'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	canAttemptLogin,
	canAttemptWechatMiniLogin,
	canRequestSms,
	AUTH_SLOW_FEEDBACK_DELAY,
	getCodeError,
	getPhoneError,
	isCodeValid,
	isPhoneValid,
	resolveAlternateLoginMode,
	resolveAuthenticationGate,
	resolveEntryFunnelAgreementState,
	resolveLoginMode,
	resolvePostLoginNavigation,
	resolveSmsFeedback,
	resolveWechatPostLoginState,
	shouldClearEntryFunnelAgreementSession,
	shouldClearPendingPostLoginIntent
} from '@/utils/entry-funnel.js'
import {
	clearPostLoginIntent,
	getPostLoginIntent
} from '@/utils/post-login-intent.js'

const WELCOME_PAGE_VIEWED_KEY = 'welcome_page_viewed'
const ENTRY_FUNNEL_AGREEMENT_KEY = 'entry_funnel_agreement_accepted'
const ENTRY_FUNNEL_SESSION_KEY = 'entry_funnel_session_active'

let isHarmonyPlatform = false
// #ifdef APP-HARMONY
isHarmonyPlatform = true
// #endif

let isWechatMiniProgram = false
// #ifdef MP-WEIXIN
isWechatMiniProgram = true
// #endif

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const authMode = ref(resolveLoginMode({ isWechatMini: isWechatMiniProgram }))

const formData = reactive({
	phone: '',
	code: ''
})

const countdown = ref(0)
const isSending = ref(false)
const isLogging = ref(false)
const isWechatLogging = ref(false)
const isHuaweiLogging = ref(false)
const isSlowLogging = ref(false)
const isPageActive = ref(true)
const showBindPhoneModal = ref(false)
const isAgreed = ref(false)
const showAgreementSheet = ref(false)
const pendingAgreementAction = ref('')
const authGate = computed(() => resolveAuthenticationGate({
	isWechatLogging: isWechatLogging.value,
	isPhoneLogging: isLogging.value,
	isHuaweiLogging: isHuaweiLogging.value,
	isPageActive: isPageActive.value
}))
const showHuaweiLogin = computed(() => isHarmonyPlatform)
const phoneError = computed(() => getPhoneError(formData.phone))
const codeError = computed(() => getCodeError(formData.code))
const canSendCode = computed(() => canRequestSms({
	phone: formData.phone,
	countdown: countdown.value,
	isSending: isSending.value || authGate.value.isAuthenticating
}))
const canSubmit = computed(() => canAttemptLogin({
	phone: formData.phone,
	code: formData.code,
	isLogging: authGate.value.isAuthenticating
}))
const canAttemptWechatLogin = computed(() => canAttemptWechatMiniLogin({
	isAgreed: isAgreed.value,
	isLogging: authGate.value.isAuthenticating
}))
const sendCodeText = computed(() => {
	if (countdown.value > 0) {
		return `${countdown.value}s 后重试`
	}

	return isSending.value ? '发送中...' : '获取验证码'
})

let countdownTimer = null
let authSlowFeedbackTimer = null
let completedLoginFlow = false

const startAuthFeedback = () => {
	isSlowLogging.value = false
	if (authSlowFeedbackTimer) clearTimeout(authSlowFeedbackTimer)
	authSlowFeedbackTimer = setTimeout(() => {
		if (authGate.value.isAuthenticating) isSlowLogging.value = true
	}, AUTH_SLOW_FEEDBACK_DELAY)
}

const clearAuthFeedback = () => {
	if (authSlowFeedbackTimer) {
		clearTimeout(authSlowFeedbackTimer)
		authSlowFeedbackTimer = null
	}
	isSlowLogging.value = false
}

const switchLoginMode = () => {
	if (!isWechatMiniProgram) return
	if (!authGate.value.canLeave) return
	authMode.value = resolveAlternateLoginMode(authMode.value)
}

const persistAgreementState = (value) => {
	uni.setStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY, Boolean(value))
}

const toggleAgreement = () => {
	if (authGate.value.isAuthenticating) return
	isAgreed.value = !isAgreed.value
	persistAgreementState(isAgreed.value)
}

const requestAgreementFor = (action) => {
	pendingAgreementAction.value = action
	showAgreementSheet.value = true
}

const closeAgreementSheet = () => {
	showAgreementSheet.value = false
	pendingAgreementAction.value = ''
}

const startCountdown = () => {
	countdown.value = 60
	countdownTimer = setInterval(() => {
		countdown.value--
		if (countdown.value <= 0) {
			clearInterval(countdownTimer)
			countdownTimer = null
		}
	}, 1000)
}

const navigateAfterLogin = () => {
	const pages = getCurrentPages()
	const pendingAction = getPostLoginIntent()
	const action = resolvePostLoginNavigation({
		pageCount: pages.length,
		pendingAction
	})

	if (action === 'intent') {
		uni.switchTab({ url: '/pages/user/index' })
		return
	}

	if (action === 'back') {
		uni.navigateBack({
			fail: () => {
				uni.switchTab({ url: '/pages/index/index' })
			}
		})
		return
	}

	uni.switchTab({ url: '/pages/index/index' })
}

const handleSendCode = async () => {
	if (!isPhoneValid(formData.phone)) {
		uni.showToast({
			title: formData.phone ? '请输入正确的手机号' : '请输入手机号',
			icon: 'none'
		})
		return
	}

	isSending.value = true

	try {
		await sendSms(formData.phone.trim(), 'login')
		uni.showToast({
			title: resolveSmsFeedback(formData.phone),
			icon: 'none'
		})
		startCountdown()
	} catch (error) {
		uni.showToast({
			title: error.message || '发送失败，请重试',
			icon: 'none'
		})
	} finally {
		isSending.value = false
	}
}

const handleLogin = async () => {
	if (!authGate.value.canStart) return
	if (!isAgreed.value) {
		requestAgreementFor('submit')
		return
	}

	if (!isPhoneValid(formData.phone)) {
		uni.showToast({
			title: formData.phone ? '请输入正确的手机号' : '请输入手机号',
			icon: 'none'
		})
		return
	}

	if (!isCodeValid(formData.code)) {
		uni.showToast({
			title: formData.code ? '请输入6位验证码' : '请输入验证码',
			icon: 'none'
		})
		return
	}

	isLogging.value = true
	startAuthFeedback()

	try {
		const res = await login({
			phone: formData.phone.trim(),
			sms_code: formData.code.trim()
		})
		if (!authGate.value.shouldHandleResult) return

		userStore.login(res)
		uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
		completedLoginFlow = true
		navigateAfterLogin()
		uni.showToast({
			title: '登录成功',
			icon: 'success'
		})
	} catch (error) {
		if (!authGate.value.shouldHandleResult) return
		uni.showToast({
			title: error.message || '登录失败，请重试',
			icon: 'none'
		})
	} finally {
		isLogging.value = false
		clearAuthFeedback()
	}
}

const handleWechatMiniLogin = async () => {
	if (!authGate.value.canStart) return
	if (!isAgreed.value) {
		requestAgreementFor('wechat-mini')
		return
	}
	if (!canAttemptWechatLogin.value) return

	// #ifdef MP-WEIXIN
	isWechatLogging.value = true
	startAuthFeedback()
	try {
		const loginResult = await new Promise((resolve, reject) => {
			uni.login({
				success: resolve,
				fail: reject
			})
		})
		if (!authGate.value.shouldHandleResult) return
		if (!loginResult?.code) {
			throw new Error('微信登录失败，请重试')
		}

		const response = await wechatMiniLogin(loginResult.code)
		if (!authGate.value.shouldHandleResult) return
		userStore.login(response)
		uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
		completedLoginFlow = true
		const postLoginState = resolveWechatPostLoginState({
			needBindPhone: response.need_bind_phone
		})
		if (postLoginState.action === 'bind-phone') {
			showBindPhoneModal.value = true
			return
		}
		navigateAfterLogin()
	} catch (error) {
		if (!authGate.value.shouldHandleResult) return
		uni.showToast({
			title: error.message || '微信登录失败，请重试',
			icon: 'none'
		})
	} finally {
		isWechatLogging.value = false
		clearAuthFeedback()
	}
	// #endif
}

const handleHuaweiLogin = async () => {
	if (!authGate.value.canStart) return
	if (!isAgreed.value) {
		requestAgreementFor('huawei')
		return
	}

	// #ifdef APP-HARMONY
	isHuaweiLogging.value = true
	startAuthFeedback()

	try {
		const userInfo = await new Promise((resolve, reject) => {
			uni.getUserInfo({
				provider: 'huawei',
				success: resolve,
				fail: reject
			})
		})

		const loginRes = await new Promise((resolve, reject) => {
			uni.login({
				provider: 'huawei',
				success: resolve,
				fail: reject
			})
		})

		const systemInfo = uni.getSystemInfoSync()
		const platform = systemInfo?.uniPlatform || 'app-plus'

		const loginResult = await loginByOauth({
			provider: 'huawei',
			nick_name: userInfo.userInfo?.nickName || '',
			avatar_url: userInfo.userInfo?.avatarUrl || '',
			open_id: loginRes.authResult?.openid || '',
			union_id: loginRes.authResult?.unionID || '',
			platform
		})

		userStore.login(loginResult)
		uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)

		if (loginResult.need_bind_phone) {
			showBindPhoneModal.value = true
			return
		}

		completedLoginFlow = true
		navigateAfterLogin()
		uni.showToast({
			title: '登录成功',
			icon: 'success'
		})
	} catch (error) {
		uni.showToast({
			title: error.message || '华为登录失败',
			icon: 'none'
		})
	} finally {
		isHuaweiLogging.value = false
		clearAuthFeedback()
	}
	// #endif
}

const handleAgreementAccepted = () => {
	isAgreed.value = true
	persistAgreementState(true)
	const action = pendingAgreementAction.value
	closeAgreementSheet()

	if (action === 'huawei') {
		handleHuaweiLogin()
		return
	}
	if (action === 'wechat-mini') {
		handleWechatMiniLogin()
		return
	}

	handleLogin()
}

const handleBindPhoneSuccess = (payload) => {
	showBindPhoneModal.value = false

	if (payload?.action === 'relogin') {
		userStore.logout()
		authMode.value = 'phone'
		formData.phone = payload?.phone || ''
		formData.code = ''
		uni.showToast({
			title: payload.message || '账号已合并，请使用手机号登录',
			icon: 'none'
		})
		return
	}

	completedLoginFlow = true
	navigateAfterLogin()
	uni.showToast({
		title: payload?.message || '绑定成功',
		icon: 'success'
	})
}

const handleBindPhoneClose = () => {
	showBindPhoneModal.value = false
	completedLoginFlow = true
	const postLoginState = resolveWechatPostLoginState({
		needBindPhone: userStore.needBindPhone,
		bindingSkipped: true
	})
	userStore.setNeedBindPhone(postLoginState.needBindPhone)
	navigateAfterLogin()
	uni.showToast({
		title: '可稍后绑定手机号',
		icon: 'none'
	})
}

const showAgreement = (type) => {
	if (authGate.value.isAuthenticating) return
	const url = type === 'user'
		? '/subPages/agreement/userAgreement'
		: '/subPages/agreement/privacyPolicy'
	uni.navigateTo({ url })
}

onLoad((options) => {
	authMode.value = resolveLoginMode({
		isWechatMini: isWechatMiniProgram,
		requestedMethod: options?.method
	})
	formData.phone = options?.phone
		? decodeURIComponent(options.phone)
		: ''
})

onShow(() => {
	isPageActive.value = true
	const sessionActive = Boolean(uni.getStorageSync(ENTRY_FUNNEL_SESSION_KEY))
	isAgreed.value = resolveEntryFunnelAgreementState({
		storedAgreement: uni.getStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY),
		sessionActive
	})
	uni.setStorageSync(ENTRY_FUNNEL_SESSION_KEY, true)

	if (!sessionActive) {
		uni.removeStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY)
	}
})

onUnload(() => {
	isPageActive.value = false
	clearAuthFeedback()
	const visibleRoutes = getCurrentPages().map((page) => page.route)
	if (shouldClearEntryFunnelAgreementSession({
		currentRoute: 'pages/login/login',
		visibleRoutes
	})) {
		uni.removeStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY)
		uni.removeStorageSync(ENTRY_FUNNEL_SESSION_KEY)
	}

	if (shouldClearPendingPostLoginIntent({
		hasPendingIntent: getPostLoginIntent(),
		completedLoginFlow
	})) {
		clearPostLoginIntent()
	}
})

onUnmounted(() => {
	clearAuthFeedback()
	if (countdownTimer) {
		clearInterval(countdownTimer)
		countdownTimer = null
	}
})
</script>

<style lang="scss" scoped>
.login-container {
	min-height: 100vh;
	box-sizing: border-box;
	padding: calc(76rpx + env(safe-area-inset-top)) 40rpx calc(32rpx + env(safe-area-inset-bottom));
	background:
		radial-gradient(circle at 88% 5%, rgba(224, 174, 18, 0.18), transparent 28%),
		linear-gradient(180deg, #fffdf8 0%, #f8f3e9 100%);
	overflow-y: auto;

	&.dark-mode {
		background:
			radial-gradient(circle at 88% 6%, rgba(224, 174, 18, 0.15), transparent 28%),
			linear-gradient(180deg, #1b170f 0%, #12100b 100%);
	}
}

.login-content {
	min-height: calc(100vh - 108rpx - env(safe-area-inset-top) - env(safe-area-inset-bottom));
	display: flex;
	flex-direction: column;
}

.brand-row {
	display: flex;
	align-items: center;
	gap: 18rpx;
	margin-top: 112rpx;
}

.brand-logo {
	width: 76rpx;
	height: 76rpx;
	border-radius: 22rpx;
	box-shadow: 0 14rpx 30rpx rgba(198, 146, 0, 0.18);
}

.brand-copy {
	display: flex;
	flex-direction: column;
	gap: 4rpx;
}

.brand-name {
	color: #2a2419;
	font-size: 26rpx;
	font-weight: 800;
	letter-spacing: 1rpx;
}

.brand-en {
	color: #a08c5d;
	font-size: 16rpx;
	letter-spacing: 3rpx;
}

.dark-mode .brand-name {
	color: #fff8e8;
}

.dark-mode .brand-en {
	color: #a99a77;
}

.auth-mode {
	flex: 1;
	display: flex;
	flex-direction: column;
}

.hero-section {
	margin-top: 156rpx;
	display: flex;
	flex-direction: column;
}

.eyebrow {
	margin-bottom: 18rpx;
	color: #a47a14;
	font-size: 20rpx;
	font-weight: 800;
	letter-spacing: 4rpx;
}

.title {
	color: #211e18;
	font-size: 54rpx;
	font-weight: 800;
	line-height: 1.18;
	letter-spacing: -1rpx;
}

.subtitle {
	max-width: 590rpx;
	margin-top: 22rpx;
	color: #746b5a;
	font-size: 27rpx;
	line-height: 1.6;
}

.benefit-list {
	display: flex;
	flex-wrap: wrap;
	gap: 12rpx;
	margin-top: 30rpx;

	text {
		padding: 9rpx 16rpx;
		border: 1rpx solid #eadfbf;
		border-radius: 999rpx;
		background: rgba(255, 255, 255, 0.56);
		color: #856a2b;
		font-size: 20rpx;
	}
}

.dark-mode .eyebrow {
	color: #e1b63a;
}

.dark-mode .title {
	color: #fff8e8;
}

.dark-mode .subtitle {
	color: #c9bea2;
}

.dark-mode .benefit-list text {
	border-color: rgba(224, 174, 18, 0.28);
	background: rgba(255, 255, 255, 0.04);
	color: #d7c89b;
}

.wechat-actions {
	display: flex;
	flex-direction: column;
	gap: 18rpx;
	margin-top: auto;
}

.wechat-login-btn,
.phone-login-btn,
.login-btn,
.third-party-btn {
	width: 100%;
	margin: 0;
	padding: 0;
	box-sizing: border-box;
	text-align: center;

	&::after {
		display: none;
	}
}

.wechat-login-btn,
.phone-login-btn,
.login-btn {
	height: 96rpx;
	line-height: 96rpx;
	border-radius: 28rpx;
	font-size: 30rpx;
	font-weight: 800;
}

.wechat-login-btn {
	background: #07c160;
	color: #ffffff;
	box-shadow: 0 16rpx 30rpx rgba(7, 193, 96, 0.17);
	transition: opacity 0.2s ease, transform 0.2s ease;

	&.authenticating {
		animation: auth-pulse 1.4s ease-in-out infinite;
	}

	&[disabled] {
		opacity: 0.5;
		color: #ffffff !important;
		box-shadow: none;
	}
}

.phone-login-btn {
	border: 1rpx solid #e5cf93;
	background: rgba(255, 255, 255, 0.66);
	color: #7d5d11;
}

.phone-login-btn[disabled] {
	opacity: 0.45;
}

.authenticating {
	animation: auth-pulse 1.4s ease-in-out infinite;
}

.auth-slow-hint {
	align-self: center;
	margin-top: 12rpx;
	color: #9a7b2a;
	font-size: 22rpx;
	line-height: 1.4;
}

.dark-mode .auth-slow-hint {
	color: #d7b95c;
}

.dark-mode .phone-login-btn {
	border-color: rgba(224, 174, 18, 0.42);
	background: rgba(255, 255, 255, 0.04);
	color: #efd476;
}

.agreement-block {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8rpx;
	margin-top: 22rpx;
}

.agreement-row,
.agreement-links {
	display: flex;
	align-items: center;
	justify-content: center;
}

.agreement-row.disabled,
.agreement-links.disabled {
	opacity: 0.45;
	pointer-events: none;
}

.checkbox {
	width: 32rpx;
	height: 32rpx;
	margin-right: 10rpx;
	border: 2rpx solid rgba(224, 174, 18, 0.42);
	border-radius: 8rpx;
	background: rgba(255, 255, 255, 0.72);
	display: flex;
	align-items: center;
	justify-content: center;
	box-sizing: border-box;

	&.checked {
		background: #f7d86a;
		border-color: #f7d86a;
	}
}

.agreement-text,
.agreement-links {
	color: #918978;
	font-size: 22rpx;
}

.link {
	margin: 0 4rpx;
	color: #846414;
}

.separator {
	color: #918978;
}

.dark-mode .checkbox {
	border-color: rgba(224, 174, 18, 0.42);
	background: rgba(255, 255, 255, 0.04);
}

.dark-mode .agreement-text,
.dark-mode .agreement-links,
.dark-mode .separator {
	color: #8d836c;
}

.dark-mode .link {
	color: #efd476;
}

@keyframes auth-pulse {
	0%,
	100% {
		opacity: 0.62;
	}
	50% {
		opacity: 0.9;
	}
}

.phone-mode {
	padding-top: 34rpx;
}

.mode-back {
	align-self: flex-start;
	padding: 12rpx 0;
	color: #7d6d4e;
	font-size: 24rpx;
	font-weight: 700;
}

.phone-heading {
	display: flex;
	flex-direction: column;
	gap: 14rpx;
	margin-top: 54rpx;
}

.phone-title {
	color: #211e18;
	font-size: 50rpx;
	font-weight: 800;
}

.phone-subtitle {
	color: #817867;
	font-size: 25rpx;
	line-height: 1.5;
}

.dark-mode .mode-back {
	color: #d7c89b;
}

.dark-mode .phone-title {
	color: #fff8e8;
}

.dark-mode .phone-subtitle {
	color: #a99f89;
}

.form-section {
	display: flex;
	flex-direction: column;
	gap: 26rpx;
	margin-top: 54rpx;
}

.form-item {
	display: flex;
	flex-direction: column;
	gap: 12rpx;
}

.form-label {
	color: #4b4336;
	font-size: 25rpx;
	font-weight: 700;
}

.input-container {
	position: relative;
	display: flex;
	align-items: center;
	height: 96rpx;
	border: 1rpx solid #e9dfcc;
	border-radius: 24rpx;
	background: rgba(255, 255, 255, 0.78);
	box-shadow: 0 8rpx 18rpx rgba(85, 65, 26, 0.04);
	box-sizing: border-box;

	&.invalid {
		border-color: rgba(220, 38, 38, 0.52);
	}
}

.country-code {
	padding: 0 22rpx;
	border-right: 1rpx solid #ece3d2;
	color: #443c31;
	font-size: 27rpx;
	font-weight: 700;
}

.form-input {
	flex: 1;
	height: 96rpx;
	padding: 0 24rpx;
	color: #2b261e;
	font-size: 28rpx;
	background: transparent;
	box-sizing: border-box;

	&.code-input {
		padding-right: 220rpx;
	}
}

.send-code-btn {
	position: absolute;
	right: 12rpx;
	height: 64rpx;
	line-height: 64rpx;
	margin: 0;
	padding: 0 20rpx;
	border-radius: 18rpx;
	background: #f3e4b6;
	color: #86620d;
	font-size: 23rpx;
	font-weight: 700;
	box-sizing: border-box;

	&::after {
		display: none;
	}

	&[disabled] {
		opacity: 0.48;
		color: #86620d !important;
	}
}

.field-error {
	padding-left: 6rpx;
	color: #dc2626;
	font-size: 22rpx;
}

.dark-mode .form-label {
	color: #ede3ca;
}

.dark-mode .input-container {
	border-color: rgba(224, 174, 18, 0.2);
	background: rgba(255, 255, 255, 0.04);
}

.dark-mode .country-code {
	border-right-color: rgba(224, 174, 18, 0.18);
	color: #f1e6cb;
}

.dark-mode .form-input {
	color: #fff8e8;
}

.phone-agreement {
	margin-top: 28rpx;
}

.login-btn {
	height: 96rpx;
	line-height: 96rpx;
	margin: 28rpx 0 0;
	padding: 0;
	background: linear-gradient(135deg, #e5b928 0%, #c99700 100%);
	color: #ffffff;
	display: flex;
	align-items: center;
	justify-content: center;
	box-shadow: 0 14rpx 28rpx rgba(201, 151, 0, 0.18);

	&[disabled] {
		opacity: 0.45;
		color: #ffffff !important;
		box-shadow: none;
	}
}

.mode-switch {
	align-self: center;
	padding: 24rpx;
	color: #8b6915;
	font-size: 23rpx;
	font-weight: 700;
}

.mode-back.disabled,
.mode-switch.disabled {
	opacity: 0.45;
	pointer-events: none;
}

.dark-mode .mode-switch {
	color: #efd476;
}

.third-party-login {
	margin-top: 22rpx;
	display: flex;
	flex-direction: column;
	gap: 20rpx;
}

.divider {
	display: flex;
	align-items: center;
}

.divider-line {
	flex: 1;
	height: 1rpx;
	background: rgba(154, 140, 103, 0.24);
}

.divider-text {
	padding: 0 16rpx;
	color: #9a8c67;
	font-size: 22rpx;
}

.third-party-btn {
	height: 88rpx;
	line-height: 88rpx;
	border: 1rpx solid #e5cf93;
	border-radius: 24rpx;
	background: rgba(255, 255, 255, 0.5);
	color: #6e6242;
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 12rpx;
	font-size: 27rpx;
	font-weight: 700;

	image {
		width: 32rpx;
		height: 32rpx;
	}
}

.dark-mode .third-party-btn {
	border-color: rgba(224, 174, 18, 0.32);
	background: rgba(255, 255, 255, 0.04);
	color: #fff8e8;
}
</style>
