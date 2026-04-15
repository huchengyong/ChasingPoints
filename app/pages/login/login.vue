<template>
	<view class="login-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="ambient-glow ambient-glow-top"></view>
		<view class="ambient-glow ambient-glow-bottom"></view>

		<view class="login-content">
			<view class="header-section">
				<view class="header-badge">
					<text class="header-badge-text">追分竞技入口</text>
				</view>
				<view class="logo-shell">
					<view class="logo">
						<image src="/static/logo.png" mode="aspectFit" />
					</view>
				</view>
				<text class="title">手机号登录 / 注册</text>
				<text class="subtitle">未注册手机号验证后将自动创建账号</text>
			</view>

			<view class="form-card">
				<view class="form-section">
					<view class="form-item">
						<text class="form-label">手机号</text>
						<view class="input-container" :class="{ invalid: phoneError }">
							<uni-icons class="input-icon" type="phone-filled" size="24"></uni-icons>
							<input
								type="tel"
								v-model="formData.phone"
								placeholder="请输入手机号"
								class="form-input"
								maxlength="11"
							/>
						</view>
						<text v-if="phoneError" class="field-error">{{ phoneError }}</text>
					</view>

					<view class="form-item">
						<view class="label-row">
							<text class="form-label">短信验证码</text>
							<text class="field-hint">验证码可用于登录或自动注册</text>
						</view>
						<view class="input-container" :class="{ invalid: codeError }">
							<uni-icons class="input-icon" type="locked-filled" size="24"></uni-icons>
							<input
								type="number"
								v-model="formData.code"
								placeholder="请输入6位验证码"
								class="form-input code-input"
								maxlength="6"
							/>
							<button
								class="send-code-btn"
								:disabled="!canSendCode"
								@click="handleSendCode"
							>
								{{ sendCodeText }}
							</button>
						</view>
						<text v-if="codeError" class="field-error">{{ codeError }}</text>
					</view>
				</view>

				<view class="agreement-block">
					<view class="agreement-row" @click="toggleAgreement">
						<view class="checkbox" :class="{ checked: isAgreed }">
							<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#231c0b"></uni-icons>
						</view>
						<text class="agreement-text">我已阅读并同意</text>
					</view>
					<view class="agreement-links">
						<text class="link" @click.stop="showAgreement('user')">用户协议</text>
						<text class="separator">和</text>
						<text class="link" @click.stop="showAgreement('privacy')">隐私政策</text>
					</view>
				</view>

				<view class="login-btn-container">
					<button class="login-btn" :disabled="!canSubmit" @click="handleLogin">
						{{ isLogging ? '进入中...' : '立即进入' }}
					</button>
					<text class="cta-hint">
						{{ isAgreed ? '验证成功后将立即进入首页' : '勾选协议后即可继续' }}
					</text>
				</view>

				<view v-if="showHuaweiLogin" class="third-party-login">
					<view class="divider">
						<view class="divider-line"></view>
						<text class="divider-text">其他可用方式</text>
						<view class="divider-line"></view>
					</view>
					<button
						class="third-party-btn"
						@click="handleHuaweiLogin"
					>
						<image src="/static/images/huawei.svg" mode="aspectFit" />
						<text>华为账号登录</text>
					</button>
				</view>
			</view>
		</view>

		<bindPhone
			:show="showBindPhoneModal"
			:closable="true"
			:is-dark-mode="isDarkMode"
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
import { onShow, onUnload } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { sendSms, login, loginByOauth } from '@/api/auth.js'
import agreementConsentSheet from '@/components/agreementConsentSheet.vue'
import bindPhone from '@/components/bindPhone.vue'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	canAttemptLogin,
	canRequestSms,
	getCodeError,
	getPhoneError,
	isCodeValid,
	isPhoneValid,
	resolveEntryFunnelAgreementState,
	resolvePostLoginNavigation,
	resolveSmsFeedback,
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

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

const formData = reactive({
	phone: '',
	code: ''
})

const countdown = ref(0)
const isSending = ref(false)
const isLogging = ref(false)
const showBindPhoneModal = ref(false)
const isAgreed = ref(false)
const showAgreementSheet = ref(false)
const pendingAgreementAction = ref('')
const showHuaweiLogin = computed(() => isHarmonyPlatform)
const phoneError = computed(() => getPhoneError(formData.phone))
const codeError = computed(() => getCodeError(formData.code))
const canSendCode = computed(() => canRequestSms({
	phone: formData.phone,
	countdown: countdown.value,
	isSending: isSending.value
}))
const canSubmit = computed(() => canAttemptLogin({
	phone: formData.phone,
	code: formData.code,
	isLogging: isLogging.value
}))
const sendCodeText = computed(() => {
	if (countdown.value > 0) {
		return `${countdown.value}s 后重试`
	}

	return isSending.value ? '发送中...' : '获取验证码'
})

let countdownTimer = null
let completedLoginFlow = false

const persistAgreementState = (value) => {
	uni.setStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY, Boolean(value))
}

const toggleAgreement = () => {
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

	try {
		const res = await login({
			phone: formData.phone.trim(),
			sms_code: formData.code.trim()
		})

		userStore.login(res)
		uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
		completedLoginFlow = true
		navigateAfterLogin()
		uni.showToast({
			title: '登录成功',
			icon: 'success'
		})
	} catch (error) {
		uni.showToast({
			title: error.message || '登录失败，请重试',
			icon: 'none'
		})
	} finally {
		isLogging.value = false
	}
}

const handleHuaweiLogin = async () => {
	if (!isAgreed.value) {
		requestAgreementFor('huawei')
		return
	}

	// #ifdef APP-HARMONY
	uni.showLoading({
		title: '正在登录...'
	})

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
		uni.hideLoading()

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
		uni.hideLoading()
		uni.showToast({
			title: error.message || '华为登录失败',
			icon: 'none'
		})
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

	handleLogin()
}

const handleBindPhoneSuccess = (payload) => {
	showBindPhoneModal.value = false

	if (payload?.action === 'relogin') {
		userStore.logout()
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
	navigateAfterLogin()
	uni.showToast({
		title: '可稍后绑定手机号',
		icon: 'none'
	})
}

const showAgreement = (type) => {
	const url = type === 'user'
		? '/subPages/agreement/userAgreement'
		: '/subPages/agreement/privacyPolicy'
	uni.navigateTo({ url })
}

onShow(() => {
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
	if (countdownTimer) {
		clearInterval(countdownTimer)
		countdownTimer = null
	}
})
</script>

<style lang="scss" scoped>
.login-container {
	position: relative;
	min-height: 100vh;
	padding: 64rpx 32rpx 48rpx;
	background:
		radial-gradient(circle at top, rgba(224, 174, 18, 0.18), transparent 32%),
		linear-gradient(180deg, #faf8f2 0%, #f3ecdd 100%);
	overflow: hidden;

	&.dark-mode {
		background:
			radial-gradient(circle at top, rgba(247, 216, 106, 0.16), transparent 30%),
			linear-gradient(180deg, #181209 0%, #141109 58%, #0c0905 100%);
	}
}

.ambient-glow {
	position: absolute;
	border-radius: 50%;
	filter: blur(40rpx);
	opacity: 0.58;
}

.ambient-glow-top {
	top: 40rpx;
	right: -80rpx;
	width: 260rpx;
	height: 260rpx;
	background: rgba(247, 216, 106, 0.28);
}

.ambient-glow-bottom {
	left: -90rpx;
	bottom: 200rpx;
	width: 220rpx;
	height: 220rpx;
	background: rgba(198, 146, 0, 0.14);
}

.login-content {
	position: relative;
	z-index: 1;
	display: flex;
	flex-direction: column;
	gap: 32rpx;
}

.header-section {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	gap: 18rpx;
	padding-top: 20rpx;
}

.header-badge {
	padding: 12rpx 24rpx;
	border-radius: 999rpx;
	background: rgba(255, 247, 225, 0.72);
	border: 1rpx solid rgba(224, 174, 18, 0.18);
}

.dark-mode .header-badge {
	background: rgba(255, 247, 225, 0.08);
	border-color: rgba(247, 231, 168, 0.18);
}

.header-badge-text {
	color: #8a6510;
	font-size: 24rpx;
	font-weight: 700;
	letter-spacing: 2rpx;
}

.dark-mode .header-badge-text {
	color: #f7e7a8;
}

.logo-shell {
	padding: 12rpx;
	border-radius: 28rpx;
	background: rgba(255, 255, 255, 0.72);
	box-shadow: 0 16rpx 32rpx rgba(198, 146, 0, 0.12);
}

.dark-mode .logo-shell {
	background: rgba(30, 24, 13, 0.92);
	box-shadow: 0 16rpx 32rpx rgba(0, 0, 0, 0.2);
}

.logo {
	width: 112rpx;
	height: 112rpx;
	border-radius: 24rpx;
	background: linear-gradient(135deg, #f7d86a 0%, #c69200 100%);
	display: flex;
	align-items: center;
	justify-content: center;
	overflow: hidden;

	image {
		width: 100%;
		height: 100%;
	}
}

.title {
	color: #231c0b;
	font-size: 54rpx;
	font-weight: 700;
	line-height: 1.2;
}

.dark-mode .title {
	color: #fff7e1;
}

.subtitle {
	max-width: 620rpx;
	color: #6e6242;
	font-size: 28rpx;
	line-height: 1.6;
}

.dark-mode .subtitle {
	color: #d7c89b;
}

.form-card {
	padding: 36rpx 28rpx;
	border-radius: 32rpx;
	background: rgba(255, 255, 255, 0.82);
	border: 1rpx solid rgba(224, 174, 18, 0.08);
	box-shadow: 0 28rpx 60rpx rgba(35, 28, 11, 0.08);
	backdrop-filter: blur(18px);
}

.dark-mode .form-card {
	background: rgba(30, 24, 13, 0.96);
	border-color: rgba(247, 231, 168, 0.12);
	box-shadow: 0 28rpx 60rpx rgba(0, 0, 0, 0.22);
}

.form-section {
	display: flex;
	flex-direction: column;
	gap: 28rpx;
}

.form-item {
	display: flex;
	flex-direction: column;
	gap: 14rpx;
}

.label-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 16rpx;
}

.form-label {
	font-size: 28rpx;
	font-weight: 700;
	color: #231c0b;
}

.dark-mode .form-label {
	color: #fff7e1;
}

.field-hint {
	font-size: 22rpx;
	color: #8c805f;
}

.input-container {
	position: relative;
	display: flex;
	align-items: center;
	border: 2rpx solid rgba(224, 174, 18, 0.12);
	border-radius: 20rpx;
	background: rgba(250, 248, 242, 0.92);
	transition: border-color 0.2s ease, box-shadow 0.2s ease;

	&.invalid {
		border-color: rgba(239, 68, 68, 0.52);
		box-shadow: 0 0 0 6rpx rgba(239, 68, 68, 0.08);
	}
}

.dark-mode .input-container {
	background: rgba(20, 17, 9, 0.92);
	border-color: rgba(247, 231, 168, 0.12);
}

.input-icon {
	position: absolute;
	left: 24rpx;
	z-index: 1;
	color: #8a6510;
}

.dark-mode .input-icon {
	color: #f7e7a8;
}

.form-input {
	flex: 1;
	height: 104rpx;
	padding-left: 78rpx;
	padding-right: 28rpx;
	font-size: 30rpx;
	color: #231c0b;
	background: transparent;

	&.code-input {
		padding-right: 238rpx;
	}
}

.dark-mode .form-input {
	color: #fff7e1;
}

.send-code-btn {
	position: absolute;
	right: 14rpx;
	height: 72rpx;
	line-height: 72rpx;
	padding: 0 24rpx;
	border-radius: 14rpx;
	background: linear-gradient(135deg, #f7d86a 0%, #e0ae12 48%, #c69200 100%);
	color: #ffffff;
	font-size: 28rpx;
	font-weight: 700;
	display: flex;
	align-items: center;
	justify-content: center;
	box-shadow: 0 4rpx 8rpx rgba(224, 174, 18, 0.3);

	&::after {
		display: none;
	}

	&[disabled] {
		opacity: 0.45;
		box-shadow: none;
		color: #ffffff !important;
	}
}

.dark-mode .send-code-btn {
	background: linear-gradient(135deg, #f7d86a 0%, #e0ae12 48%, #c69200 100%);
	color: #ffffff;
}

.field-error {
	font-size: 24rpx;
	color: #dc2626;
	padding-left: 6rpx;
}

.agreement-block {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 10rpx;
	margin-top: 32rpx;
}

.agreement-row {
	display: flex;
	align-items: center;
	justify-content: center;
}

.checkbox {
	width: 34rpx;
	height: 34rpx;
	margin-right: 12rpx;
	border-radius: 8rpx;
	border: 2rpx solid rgba(224, 174, 18, 0.18);
	background: rgba(255, 255, 255, 0.72);
	display: flex;
	align-items: center;
	justify-content: center;

	&.checked {
		background: #f7d86a;
		border-color: #f7d86a;
	}
}

.dark-mode .checkbox {
	background: rgba(255, 247, 225, 0.06);
	border-color: rgba(247, 231, 168, 0.18);
}

.agreement-text {
	font-size: 24rpx;
	color: #6e6242;
}

.dark-mode .agreement-text {
	color: #c6b78c;
}

.agreement-links {
	display: flex;
	align-items: center;
	justify-content: center;
	flex-wrap: wrap;
	font-size: 24rpx;
}

.link {
	color: #8a6510;
	margin: 0 8rpx;
	text-decoration: underline;
}

.dark-mode .link {
	color: #f7e7a8;
}

.separator {
	color: #9a8c67;
}

.login-btn-container {
	display: flex;
	flex-direction: column;
	gap: 16rpx;
	margin-top: 32rpx;
}

.login-btn {
	width: 100%;
	height: 100rpx;
	padding: 0;
	border-radius: 999rpx;
	background: linear-gradient(135deg, #f7d86a 0%, #e0ae12 48%, #c69200 100%);
	color: #ffffff;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 32rpx;
	font-weight: 700;
	line-height: 1;
	text-align: center;
	box-shadow: 0 18rpx 34rpx rgba(224, 174, 18, 0.24);

	&::after {
		display: none;
	}

	&[disabled] {
		opacity: 0.45;
		box-shadow: none;
		color: #ffffff !important;
	}
}

.cta-hint {
	font-size: 24rpx;
	text-align: center;
	color: #8c805f;
}

.dark-mode .cta-hint {
	color: #9f926e;
}

.third-party-login {
	margin-top: 36rpx;
	display: flex;
	flex-direction: column;
	gap: 24rpx;
}

.divider {
	display: flex;
	align-items: center;
}

.divider-line {
	flex: 1;
	height: 2rpx;
	background: rgba(154, 140, 103, 0.24);
}

.divider-text {
	padding: 0 16rpx;
	font-size: 24rpx;
	color: #9a8c67;
}

.third-party-btn {
	width: 100%;
	min-height: 92rpx;
	padding: 0 24rpx;
	border-radius: 18rpx;
	background: rgba(255, 247, 225, 0.08);
	border: 1rpx solid rgba(224, 174, 18, 0.12);
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 12rpx;
	color: #6e6242;
	font-size: 28rpx;
	font-weight: 700;

	&::after {
		display: none;
	}

	&[disabled] {
		opacity: 0.45;
	}

	image {
		width: 34rpx;
		height: 34rpx;
	}
}

.dark-mode .third-party-btn {
	color: #fff7e1;
	background: rgba(255, 247, 225, 0.04);
	border-color: rgba(247, 231, 168, 0.12);
}
</style>
