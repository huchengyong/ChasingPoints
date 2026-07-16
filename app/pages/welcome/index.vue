<template>
	<view v-if="shouldShow" class="welcome-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="content">
			<view class="brand-row">
				<image class="brand-logo" src="/static/logo.png" mode="aspectFit"></image>
				<view class="brand-copy">
					<text class="brand-name">追分竞技</text>
					<text class="brand-en">CHASING POINTS</text>
				</view>
			</view>

			<view class="hero-section">
				<text class="eyebrow">台球竞技记录</text>
				<text class="main-title">每一杆，</text>
				<text class="main-title">都值得被记录</text>
				<text class="sub-title">记录战绩、生成战报，找到真正旗鼓相当的对手。</text>
				<view class="benefit-list">
					<text>对局记录</text>
					<text>竞技排名</text>
					<text>战报分享</text>
				</view>
			</view>

			<view class="button-section">
				<button
					class="btn-primary"
					:class="{ 'wechat-primary': isWechatMiniProgram, authenticating: authGate.isAuthenticating }"
					:disabled="welcomeActions.primaryDisabled"
					@click="handlePrimaryEntry"
				>
					{{ welcomeActions.primaryText }}
				</button>
				<view
					v-if="authFeedback.visible"
					class="auth-feedback-card"
					:class="`is-${authFeedback.phase}`"
				>
					<view class="auth-feedback-progress"></view>
					<view class="auth-feedback-dot"></view>
					<text class="auth-feedback-message">{{ authFeedback.message }}</text>
				</view>

				<button
					v-if="welcomeActions.showPhoneLogin"
					class="phone-login-btn"
					:disabled="authGate.isAuthenticating"
					@click="navigateToLogin"
				>
					{{ welcomeActions.secondaryText }}
				</button>

				<button
					v-if="welcomeActions.showHuaweiLogin"
					class="btn-secondary"
					:disabled="welcomeActions.secondaryDisabled"
					@click="handleHuaweiLogin"
				>
					<image class="button-icon" src="/static/images/huawei.svg" mode="aspectFit"></image>
					<text>{{ welcomeActions.secondaryText }}</text>
				</button>

				<button class="btn-tertiary" :disabled="authGate.isAuthenticating" @click="handleBrowse">
					{{ welcomeActions.tertiaryText }}
				</button>
			</view>

			<view class="agreement">
				<view class="agreement-row" :class="{ disabled: authGate.isAuthenticating }" @click="toggleAgreement">
					<view class="checkbox" :class="{ checked: isAgreed }">
						<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#231c0b"></uni-icons>
					</view>
					<text class="agreement-text">我已阅读并同意</text>
				</view>
				<view class="agreement-links" :class="{ disabled: authGate.isAuthenticating }">
					<text class="link" @click.stop="openUserAgreement">《用户协议》</text>
					<text class="separator">和</text>
					<text class="link" @click.stop="openPrivacyPolicy">《隐私政策》</text>
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
			@open-user="openUserAgreement"
			@open-privacy="openPrivacyPolicy"
		/>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow, onUnload } from '@dcloudio/uni-app'
import agreementConsentSheet from '@/components/agreementConsentSheet.vue'
import bindPhone from '@/components/bindPhone.vue'
import { wechatMiniLogin } from '@/api/auth.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	AUTH_SLOW_FEEDBACK_DELAY,
	resolveAuthenticationFeedback,
	resolveAuthenticationGate,
	resolveEntryFunnelAgreementState,
	resolveWechatPostLoginState,
	resolveWelcomeActions,
	shouldClearEntryFunnelAgreementSession
} from '@/utils/entry-funnel.js'

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
const shouldShow = ref(false)
const isPageActive = ref(true)
const isAgreed = ref(false)
const isWechatLogging = ref(false)
const isSlowLogging = ref(false)
const showBindPhoneModal = ref(false)
const showAgreementSheet = ref(false)
const pendingAgreementAction = ref('')
const authGate = computed(() => resolveAuthenticationGate({
	isWechatLogging: isWechatLogging.value,
	isPageActive: isPageActive.value
}))
const authFeedback = computed(() => resolveAuthenticationFeedback({
	isAuthenticating: authGate.value.isAuthenticating,
	isSlow: isSlowLogging.value
}))
let authSlowFeedbackTimer = null

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
const welcomeActions = computed(() => resolveWelcomeActions({
	isHarmony: isHarmonyPlatform,
	isWechatMini: isWechatMiniProgram,
	isAgreed: isAgreed.value,
	isLogging: isWechatLogging.value,
	isSlowLogging: isSlowLogging.value
}))

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

	const hasViewed = uni.getStorageSync(WELCOME_PAGE_VIEWED_KEY)
	if (hasViewed && !showBindPhoneModal.value) {
		navigateToHome()
		return
	}

	shouldShow.value = true
})

onUnload(() => {
	isPageActive.value = false
	clearAuthFeedback()
	const visibleRoutes = getCurrentPages().map((page) => page.route)
	if (shouldClearEntryFunnelAgreementSession({
		currentRoute: 'pages/welcome/index',
		visibleRoutes
	})) {
		uni.removeStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY)
		uni.removeStorageSync(ENTRY_FUNNEL_SESSION_KEY)
	}
})

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

const navigateToHome = () => {
	uni.reLaunch({
		url: '/pages/index/index'
	})
}

const markAsViewedAndNavigate = () => {
	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	navigateToHome()
}

const handleBrowse = () => {
	if (!authGate.value.canLeave) return
	markAsViewedAndNavigate()
}

const navigateToLogin = () => {
	if (!authGate.value.canLeave) return
	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	uni.navigateTo({
		url: '/pages/login/login?method=phone'
	})
}

const handlePrimaryEntry = () => {
	if (isWechatMiniProgram) {
		handleWechatMiniLogin()
		return
	}

	if (!isAgreed.value) {
		requestAgreementFor('primary')
		return
	}
	navigateToLogin()
}

const handleWechatMiniLogin = async () => {
	if (!authGate.value.canStart) return
	if (!isAgreed.value) {
		requestAgreementFor('wechat-mini')
		return
	}
	if (isWechatLogging.value) return

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
		const postLoginState = resolveWechatPostLoginState({
			needBindPhone: response.need_bind_phone
		})
		if (postLoginState.action === 'bind-phone') {
			showBindPhoneModal.value = true
			return
		}
		navigateToHome()
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
}

const handleHuaweiLogin = () => {
	if (!isAgreed.value) {
		requestAgreementFor('huawei')
		return
	}
	navigateToLogin()
}

const handleBindPhoneSuccess = (payload) => {
	showBindPhoneModal.value = false
	if (payload?.action === 'relogin') {
		userStore.logout()
		const encodedPhone = encodeURIComponent(payload?.phone || '')
		uni.navigateTo({
			url: `/pages/login/login?method=phone&phone=${encodedPhone}`
		})
		uni.showToast({
			title: payload.message || '账号已合并，请使用手机号登录',
			icon: 'none'
		})
		return
	}

	navigateToHome()
	uni.showToast({
		title: payload?.message || '绑定成功',
		icon: 'success'
	})
}

const handleBindPhoneClose = () => {
	const postLoginState = resolveWechatPostLoginState({
		needBindPhone: userStore.needBindPhone,
		bindingSkipped: true
	})
	userStore.setNeedBindPhone(postLoginState.needBindPhone)
	showBindPhoneModal.value = false
	navigateToHome()
	uni.showToast({
		title: '可稍后绑定手机号',
		icon: 'none'
	})
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

	handlePrimaryEntry()
}

const openUserAgreement = () => {
	if (authGate.value.isAuthenticating) return
	uni.navigateTo({
		url: '/subPages/agreement/userAgreement'
	})
}

const openPrivacyPolicy = () => {
	if (authGate.value.isAuthenticating) return
	uni.navigateTo({
		url: '/subPages/agreement/privacyPolicy'
	})
}
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
