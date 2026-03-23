<template>
	<view class="welcome-container" :class="{ 'dark-mode': isDarkMode }" v-if="shouldShow">
		<!-- 背景图片层 -->
		<image class="bg-image" src="/static/images/welcome-bg.jpg" mode="aspectFill"></image>
		
		<!-- 跳过按钮 -->
		<view class="skip-btn" @click="handleSkip">
			<text class="skip-text">跳过</text>
		</view>
		
		<!-- 内容层 -->
		<view class="content">
			<!-- 占位区域，用于将内容推到底部 -->
			<view class="spacer"></view>
			
			<!-- 标题区域 -->
			<view class="title-section">
				<text class="main-title">记录你的每一次精彩</text>
				<text class="sub-title">数据分析，智能匹配，寻找你的宿命对手</text>
			</view>
			
			<!-- 按钮区域 -->
			<view class="button-section">
				<button class="btn-register" @click="handleRegister">注册</button>
				<button class="btn-login" @click="handleLogin">登录</button>
			</view>
			
			<!-- 第三方登录区域 -->
			<view class="third-party-section">
				<text class="third-party-text">或通过以下方式快速登录</text>
				<view class="third-party-buttons">
					<view class="third-party-btn huawei-btn" @click="handleHuaweiLogin">
						<image class="third-party-icon" src="/static/images/huawei.svg" mode="aspectFit"></image>
					</view>
				</view>
			</view>
			
			<!-- 协议和隐私 -->
			<view class="agreement">
				<view class="agreement-row" @click="isAgreed = !isAgreed">
					<view class="checkbox" :class="{ checked: isAgreed }">
						<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#000000"></uni-icons>
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
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'

const WELCOME_PAGE_VIEWED_KEY = 'welcome_page_viewed'

// ========== 状态管理 ==========
const themeStore = useThemeStore()
const isDarkMode = computed(() => themeStore.isDarkMode)
const shouldShow = ref(false)
const isAgreed = ref(false)

// ========== 生命周期 ==========

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()

	const hasViewed = uni.getStorageSync(WELCOME_PAGE_VIEWED_KEY)
	if (hasViewed) {
		navigateToHome()
	} else {
		// 只有需要显示 welcome 页面时才渲染内容
		shouldShow.value = true
	}
})

// ========== 方法 ==========

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
 * 处理跳过按钮点击
 */
const handleSkip = () => {
	markAsViewedAndNavigate()
}

/**
 * 验证协议勾选
 */
const validateAgreement = () => {
	if (!isAgreed.value) {
		uni.showToast({
			title: '请先阅读并同意用户协议',
			icon: 'none'
		})
		return false
	}
	return true
}

/**
 * 注册
 */
const handleRegister = () => {
	if (!validateAgreement()) return
	
	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	uni.navigateTo({
		url: '/pages/login/login'
	})
}

/**
 * 登录
 */
const handleLogin = () => {
	if (!validateAgreement()) return

	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	uni.navigateTo({
		url: '/pages/login/login'
	})
}

/**
 * 华为登录
 */
const handleHuaweiLogin = () => {
	if (!validateAgreement()) return

	uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)
	uni.navigateTo({
		url: '/pages/login/login'
	})
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

