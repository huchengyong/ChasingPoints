<template>
	<view class="login-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="login-content">
			<!-- Logo和欢迎信息 -->
			<view class="header-section">
				<view class="logo-container">
					<view class="logo">
						<image src="/static/logo.png" mode="aspectFit" />
					</view>
				</view>
				<text class="title">欢迎来到球艺堂</text>
				<text class="subtitle">记录您的每次精彩击球</text>
			</view>

			<!-- 表单部分 -->
			<view class="form-section">
				<!-- 手机号输入 -->
				<view class="form-item">
					<text class="form-label">手机号</text>
					<view class="input-container">
						<uni-icons class="input-icon" type="phone-filled" size="24"></uni-icons>
						<input
							type="number"
							v-model="formData.phone"
							placeholder="请输入您的手机号"
							class="form-input"
							maxlength="11"
						/>
					</view>
				</view>

				<!-- 验证码输入 -->
				<view class="form-item">
					<text class="form-label">短信验证码</text>
					<view class="input-container">
						<uni-icons class="input-icon" type="locked-filled" size="24"></uni-icons>
						<input
							type="number"
							v-model="formData.code"
							placeholder="请输入验证码"
							class="form-input code-input"
							maxlength="6"
						/>
						<button
							class="send-code-btn"
							:disabled="countdown > 0 || isSending"
							@click="handleSendCode"
						>
							{{ countdown > 0 ? `${countdown}s` : (isSending ? '发送中...' : '发送验证码') }}
						</button>
					</view>
				</view>
			</view>

			<!-- 登录按钮 -->
			<view class="login-btn-container">
				<button class="login-btn" :disabled="isLogging" @click="handleLogin">
					{{ isLogging ? '登录中...' : '登录 / 注册' }}
				</button>
			</view>

			<!-- 第三方登录 -->
			<view class="third-party-login">
				<view class="divider">
					<view class="divider-line"></view>
					<text class="divider-text">其他方式登录</text>
					<view class="divider-line"></view>
				</view>
				<view class="third-party-buttons">
					<button class="third-party-btn" @click="handleHuaweiLogin">
						<image src="/static/images/huawei.svg" mode="aspectFit" />
					</button>
				</view>
			</view>

			<!-- 协议和隐私 -->
			<view class="agreement">
				<view class="agreement-row" @click="isAgreed = !isAgreed">
					<view class="checkbox" :class="{ checked: isAgreed }">
						<uni-icons v-if="isAgreed" type="checkmarkempty" size="14" color="#ffffff"></uni-icons>
					</view>
					<text class="agreement-text">我已阅读并同意</text>
				</view>
				<view class="agreement-links">
					<text class="link" @click.stop="showAgreement('user')">用户协议</text>
					<text class="separator">和</text>
					<text class="link" @click.stop="showAgreement('privacy')">隐私政策</text>
				</view>
			</view>
		</view>

		<!-- 绑定手机号弹窗 -->
		<bindPhone
			:show="showBindPhoneModal"
			:closable="false"
			@close="showBindPhoneModal = false"
			@success="handleBindPhoneSuccess"
		/>
	</view>
</template>

<script setup>
import { ref, reactive, onUnmounted, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { useUserStore } from '@/store/user.js'
import { sendSms, login, loginByOauth } from '@/api/auth.js'
import bindPhone from '@/components/bindPhone.vue'

// 引导页标记常量
const WELCOME_PAGE_VIEWED_KEY = 'welcome_page_viewed'

// ========== 状态管理 ==========
const themeStore = useThemeStore()
const userStore = useUserStore()

// ========== 响应式数据 ==========
const formData = reactive({
	phone: '',
	code: ''
})

const countdown = ref(0)
const isSending = ref(false)
const isLogging = ref(false)
const showBindPhoneModal = ref(false)
const isAgreed = ref(false)
const isDarkMode = computed(() => themeStore.isDarkMode)

let countdownTimer = null

// ========== 工具函数 ==========


/**
 * 验证手机号格式
 * @param {String} phone 手机号
 * @returns {Boolean} 是否有效
 */
const validatePhone = (phone) => {
	if (!phone) {
		uni.showToast({
			title: '请输入手机号',
			icon: 'none'
		})
		return false
	}

	if (!/^1[3-9]\d{9}$/.test(phone)) {
		uni.showToast({
			title: '请输入正确的手机号',
			icon: 'none'
		})
		return false
	}

	return true
}

/**
 * 开始倒计时
 */
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

// ========== 事件处理 ==========

/**
 * 发送验证码
 */
const handleSendCode = async () => {
	if (!validatePhone(formData.phone)) {
		return
	}

	isSending.value = true

	try {
		await sendSms(formData.phone, 'login')
		uni.showToast({
			title: '验证码已发送',
			icon: 'success'
		})
		// 发送成功后开始倒计时
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

/**
 * 登录处理
 */
const handleLogin = async () => {
	if (!isAgreed.value) {
		uni.showToast({
			title: '请先阅读并同意用户协议',
			icon: 'none'
		})
		return
	}

	if (!validatePhone(formData.phone)) {
		return
	}

	if (!formData.code) {
		uni.showToast({
			title: '请输入验证码',
			icon: 'none'
		})
		return
	}

	isLogging.value = true

	try {
		const res = await login({
			phone: formData.phone,
			sms_code: formData.code
		})

		// 使用 Pinia Store 管理登录状态
		userStore.login(res)

		// 标记引导页为已查看
		uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)

		uni.showToast({
			title: '登录成功',
			icon: 'success'
		})

		// 跳转逻辑：检查页面栈决定返回方式
		setTimeout(() => {
			const pages = getCurrentPages()
			if (pages.length > 1) {
				// 有上一页则返回
				uni.navigateBack()
			} else {
				// 没有上一页则跳转首页
				uni.switchTab({
					url: '/pages/index/index'
				})
			}
		}, 1500)
	} catch (error) {
		uni.showToast({
			title: error.message || '登录失败，请重试',
			icon: 'none'
		})
	} finally {
		isLogging.value = false
	}
}

/**
 * 华为登录
 */
const handleHuaweiLogin = async () => {
	// #ifdef APP-HARMONY
	if (!isAgreed.value) {
		uni.showToast({
			title: '请先阅读并同意用户协议',
			icon: 'none'
		})
		return
	}

	uni.showLoading({
		title: '正在登录...'
	})

	try {
		// 第一步：获取用户信息
		const userInfo = await new Promise((resolve, reject) => {
			uni.getUserInfo({
				provider: 'huawei',
				success: resolve,
				fail: reject
			})
		})

		// 第二步：获取登录凭证和 ID Token
		const loginRes = await new Promise((resolve, reject) => {
			uni.login({
				provider: 'huawei',
				success: resolve,
				fail: reject
			})
		})

		const systemInfo = uni.getSystemInfoSync()
		const platform = systemInfo?.uniPlatform || 'app-plus'

		// 调用后端 OAuth 登录接口
		const loginResult = await loginByOauth({
			provider: 'huawei',
			nick_name: userInfo.userInfo?.nickName || '',
			avatar_url: userInfo.userInfo?.avatarUrl || '',
			open_id: loginRes.authResult?.openid || '',  // 华为返回的是 openid（全小写）
			union_id: loginRes.authResult?.unionID || '', // 华为返回的是 unionID（大写ID）
			platform: platform
		})

		// 使用 Pinia Store 管理登录状态
		userStore.login(loginResult)

		// 标记引导页为已查看
		uni.setStorageSync(WELCOME_PAGE_VIEWED_KEY, true)

		uni.hideLoading()

		// 检查是否需要绑定手机号
		if (loginResult.need_bind_phone) {
			showBindPhoneModal.value = true
			return
		}

		uni.showToast({
			title: '登录成功',
			icon: 'success'
		})

		setTimeout(() => {
			uni.navigateBack({
				delta: 1,
				fail: () => {
					uni.switchTab({
						url: '/pages/index/index'
					})
				}
			})
		}, 1500)
	} catch (error) {
		uni.hideLoading()
		uni.showToast({
			title: error.message || '华为登录失败',
			icon: 'none'
		})
	}
	// #endif

	// #ifndef APP-HARMONY
	uni.showToast({
		title: '请在HarmonyOS中使用华为登录',
		icon: 'none'
	})
	// #endif
}

/**
 * 显示协议
 * @param {String} type 协议类型 user/privacy
 */
/**
 * 绑定手机号成功处理
 */
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

	uni.showToast({
		title: payload?.message || '绑定成功',
		icon: 'success'
	})

	// 跳转到首页
	setTimeout(() => {
		uni.switchTab({
			url: '/pages/index/index'
		})
	}, 1500)
}

/**
 * 显示协议
 * @param {String} type 协议类型 user/privacy
 */
const showAgreement = (type) => {
	const url = type === 'user'
		? '/subPages/agreement/userAgreement'
		: '/subPages/agreement/privacyPolicy'
	uni.navigateTo({ url })
}

// ========== 生命周期 ==========
onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
})

// 组件卸载时清理定时器
onUnmounted(() => {
	if (countdownTimer) {
		clearInterval(countdownTimer)
		countdownTimer = null
	}
})
</script>

<style lang="scss" scoped>
.login-container {
	min-height: 100vh;
	background-color: var(--bg-color, #f6f7f8);
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 0 32rpx;

	// 深色模式
	&.dark-mode {
		--bg-color: #0f1712;
		--text-primary: #f3fff6;
		--text-secondary: #a7c0af;
		--text-tertiary: #6f8878;
		--border-color: #24342a;
		--input-bg: #16211a;
		--card-bg: #16211a;
		--primary-color-light: rgba(24, 176, 91, 0.2);
	}
}

.login-content {
	width: 100%;
	max-width: 750rpx;
}

.header-section {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 64rpx 0;
	text-align: center;

	.logo-container {
		margin-bottom: 32rpx;

		.logo {
			width: 128rpx;
			height: 128rpx;
			background: linear-gradient(135deg, #18b05b 0%, #166534 100%);
			border-radius: 50%;
			display: flex;
			align-items: center;
			justify-content: center;
			box-shadow: 0 8rpx 24rpx rgba(24, 176, 91, 0.3);
			overflow: hidden;

			image {
				width: 100%;
				height: 100%;
				border-radius: 50%;
			}
		}
	}

	.title {
		font-size: 48rpx;
		font-weight: 700;
		color: var(--text-primary, #0f172a);
		line-height: 1.4;
	}

	.subtitle {
		font-size: 32rpx;
		color: var(--text-secondary, #64748b);
		line-height: 1.5;
		margin-top: 16rpx;
	}
}

.form-section {
	padding: 32rpx 0;

	.form-item {
		margin-bottom: 32rpx;

		.form-label {
			display: block;
			font-size: 28rpx;
			font-weight: 500;
			color: var(--text-primary, #0f172a);
			margin-bottom: 16rpx;
		}

		.input-container {
			position: relative;
			display: flex;
			align-items: center;

			.input-icon {
				position: absolute;
				left: 24rpx;
				font-size: 36rpx;
				z-index: 1;
			}

			.form-input {
				flex: 1;
				height: 112rpx;
				padding-left: 80rpx;
				padding-right: 32rpx;
				border: 2rpx solid var(--border-color, #e2e8f0);
				border-radius: 16rpx;
				background-color: var(--input-bg, #ffffff);
				font-size: 32rpx;
				color: var(--text-primary, #0f172a);

				&.code-input {
					padding-right: 220rpx;
				}
			}

			.send-code-btn {
				position: absolute;
				right: 16rpx;
				height: 72rpx;
				line-height: 72rpx;
				padding: 0 24rpx;
				background-color: var(--primary-color-light, rgba(24, 176, 91, 0.1));
				color: var(--primary-color, #18b05b);
				font-size: 26rpx;
				font-weight: 500;
				border-radius: 12rpx;

				&::after {
					display: none;
				}

				&:disabled {
					opacity: 0.5;
				}
			}
		}
	}
}

.login-btn-container {
	padding: 32rpx 0;

	.login-btn {
		width: 100%;
		height: 100rpx;
		line-height: 100rpx;
		background: linear-gradient(135deg, #18b05b 0%, #166534 100%);
		color: #ffffff;
		font-size: 32rpx;
		font-weight: 600;
		border-radius: 50rpx;
		box-shadow: 0 8rpx 24rpx rgba(24, 176, 91, 0.35);
		transition: all 0.3s;

    &::after {
      display: none;
    }

		&:active {
			transform: scale(0.98);
			opacity: 0.9;
		}

		&:disabled {
			opacity: 0.6;
		}
	}
}

.third-party-login {
	padding: 48rpx 0;

	.divider {
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 48rpx;

		.divider-line {
			flex: 1;
			height: 2rpx;
			background-color: var(--border-color, #e2e8f0);
		}

		.divider-text {
			padding: 0 24rpx;
			font-size: 26rpx;
			color: var(--text-tertiary, #94a3b8);
		}
	}

	.third-party-buttons {
		display: flex;
		justify-content: center;

		.third-party-btn {
			width: 112rpx;
			height: 112rpx;
			border-radius: 50%;
			border: 2rpx solid var(--border-color, #e2e8f0);
			background-color: var(--card-bg, #ffffff);
			display: flex;
			align-items: center;
			justify-content: center;
			transition: all 0.3s;
			padding: 0;

      &::after {
        display: none;
      }

			image {
				width: 60rpx;
				height: 60rpx;
			}

			&:active {
				transform: scale(0.95);
				background-color: var(--input-bg, #f8f9fa);
			}
		}
	}
}

.agreement {
	padding-top: 48rpx;
	text-align: center;

	.agreement-row {
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 12rpx;
	}

	.checkbox {
		width: 36rpx;
		height: 36rpx;
		border: 2rpx solid var(--border-color, #e2e8f0);
		border-radius: 8rpx;
		margin-right: 12rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s;

		&.checked {
			background: linear-gradient(135deg, #18b05b 0%, #166534 100%);
			border-color: #18b05b;
		}
	}

	.agreement-text {
		font-size: 24rpx;
		color: var(--text-tertiary, #94a3b8);
	}

	.agreement-links {
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 24rpx;

		.link {
			color: var(--primary-color, #18b05b);
			margin: 0 8rpx;
		}

		.separator {
			color: var(--text-tertiary, #94a3b8);
		}
	}
}
</style>
