<template>
	<view class="welcome-container" :class="{ 'dark-mode': isDarkMode }" v-if="shouldShow">
		<!-- 背景图片层 -->
		<image class="bg-image" src="/static/images/welcome-bg.jpg" mode="aspectFill"></image>
		<view class="bg-overlay"></view>

		<!-- 内容层 -->
		<view class="content">
			<!-- 占位区域，用于将内容推到底部 -->
			<view class="spacer"></view>

			<!-- 标题区域 -->
			<view class="hero-section">
				<view class="hero-badge">
					<text class="hero-badge-text">追分竞技记录</text>
				</view>
				<view class="title-section">
					<text class="main-title">每一杆，都值得被记录</text>
					<text class="sub-title">记录战绩、生成战报、找到真正旗鼓相当的对手</text>
				</view>
			</view>

			<view class="action-panel">
				<view class="panel-copy">
					<text class="panel-title">快速开始你的竞技主页</text>
					<text class="panel-subtitle">手机号验证即可进入，未注册手机号将自动创建账号</text>
				</view>

				<!-- 按钮区域 -->
				<view class="button-section">
					<button
						class="btn-primary"
						:disabled="welcomeActions.primaryDisabled"
						@click="handlePrimaryEntry"
					>
						{{ welcomeActions.primaryText }}
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

					<button class="btn-tertiary" @click="handleBrowse">
						{{ welcomeActions.tertiaryText }}
					</button>
				</view>

				<!-- 协议和隐私 -->
				<view class="agreement">
					<view class="agreement-row" @click="toggleAgreement">
						<view class="checkbox" :class="{ checked: isAgreed }">
							<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#231c0b"></uni-icons>
						</view>
						<text class="agreement-text">我已阅读并同意</text>
					</view>
					<view class="agreement-links">
						<text class="link" @click.stop="openUserAgreement">用户协议</text>
						<text class="separator">和</text>
						<text class="link" @click.stop="openPrivacyPolicy">隐私政策</text>
					</view>
				</view>
			</view>
		</view>

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
import { ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import agreementConsentSheet from '@/components/agreementConsentSheet.vue'
import { resolveWelcomeActions } from '@/utils/entry-funnel.js'

const WELCOME_PAGE_VIEWED_KEY = 'welcome_page_viewed'
const ENTRY_FUNNEL_AGREEMENT_KEY = 'entry_funnel_agreement_accepted'

let isHarmonyPlatform = false
// #ifdef APP-HARMONY
isHarmonyPlatform = true
// #endif

// ========== 状态管理 ==========
const themeStore = useThemeStore()
const isDarkMode = computed(() => themeStore.isDarkMode)
const shouldShow = ref(false)
const isAgreed = ref(false)
const showAgreementSheet = ref(false)
const pendingAgreementAction = ref('')
const welcomeActions = computed(() => resolveWelcomeActions({
	isHarmony: isHarmonyPlatform,
	isAgreed: isAgreed.value
}))

// ========== 生命周期 ==========

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	isAgreed.value = Boolean(uni.getStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY))

	const hasViewed = uni.getStorageSync(WELCOME_PAGE_VIEWED_KEY)
	if (hasViewed) {
		navigateToHome()
	} else {
		// 只有需要显示 welcome 页面时才渲染内容
		shouldShow.value = true
	}
})

// ========== 方法 ==========

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

/**
 * 标记已查看并跳转到首页
 */
const markAsViewedAndNavigate = () => {
	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	navigateToHome()
}

/**
 * 跳转到首页
 */
const navigateToHome = () => {
	uni.reLaunch({
		url: '/pages/index/index'
	})
}

/**
 * 处理浏览按钮点击
 */
const handleBrowse = () => {
	markAsViewedAndNavigate()
}

/**
 * 进入登录页
 */
const navigateToLogin = () => {
	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	uni.navigateTo({
		url: '/pages/login/login'
	})
}

/**
 * 主入口
 */
const handlePrimaryEntry = () => {
	if (!isAgreed.value) {
		requestAgreementFor('primary')
		return
	}
	navigateToLogin()
}

/**
 * 华为登录
 */
const handleHuaweiLogin = () => {
	if (!isAgreed.value) {
		requestAgreementFor('huawei')
		return
	}
	navigateToLogin()
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

	handlePrimaryEntry()
}

/**
 * 用户协议
 */
const openUserAgreement = () => {
	uni.navigateTo({
		url: '/subPages/agreement/userAgreement'
	})
}

/**
 * 隐私政策
 */
const openPrivacyPolicy = () => {
	uni.navigateTo({
		url: '/subPages/agreement/privacyPolicy'
	})
}
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
