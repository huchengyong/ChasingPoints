<template>
	<view class="bind-phone-overlay" :class="{ 'dark-mode': isDarkMode }" v-if="show" @click="handleOverlayClick">
		<!-- 底部弹框 -->
		<view class="bind-phone-sheet" @click.stop>
			<!-- 拖拽条和关闭按钮 -->
			<view class="sheet-header">
				<view class="drag-handle"></view>
				<view class="close-btn" @click="handleClose" v-if="closable">
					<text class="close-icon">×</text>
				</view>
			</view>
			
			<!-- 标题 -->
			<text class="sheet-title">绑定手机号</text>
			<text class="sheet-subtitle">为了保障您的账户安全，并为您匹配对手。</text>
			
			<!-- 手机号输入 -->
			<view class="input-group">
				<text class="input-label">手机号</text>
				<view class="input-wrapper">
					<text class="country-code">+86</text>
					<input
						class="phone-input"
						type="number"
						v-model="phone"
						placeholder="请输入手机号码"
						maxlength="11"
						@input="handlePhoneInput"
					/>
				</view>
				<text class="error-text" v-if="phoneError">{{ phoneError }}</text>
			</view>
			
			<!-- 验证码输入 -->
			<view class="input-group">
				<text class="input-label">验证码</text>
				<view class="input-wrapper code-wrapper">
					<input
						class="code-input"
						type="number"
						v-model="code"
						placeholder="请输入6位验证码"
						maxlength="6"
					/>
					<text
						class="send-code-btn"
						:class="{ disabled: !canSendCode }"
						@click="handleSendCode"
					>
						{{ isSendingCode ? '发送中...' : (countdown > 0 ? `${countdown}s后重发` : '获取验证码') }}
					</text>
				</view>
			</view>
			
			<!-- 绑定按钮 -->
			<view class="bind-btn-container">
				<button
					class="bind-btn"
					:class="{ active: canBind }"
					:disabled="!canBind || loading"
					@click="handleBind"
				>
					{{ loading ? '绑定中...' : '立即绑定' }}
				</button>
			</view>
			
			<!-- 协议文本 -->
			<view class="agreement-section">
				<view class="agreement-row" @click="isAgreed = !isAgreed">
					<view class="checkbox" :class="{ checked: isAgreed }">
						<text v-if="isAgreed" class="check-icon">✓</text>
					</view>
					<text class="agreement-label">我已阅读并同意</text>
				</view>
				<view class="agreement-links">
					<text class="link" @click.stop="openUserAgreement">《用户协议》</text>
					<text>和</text>
					<text class="link" @click.stop="openPrivacyPolicy">《隐私政策》</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
import { sendSms, bindPhone } from '@/api/auth.js'
import { useUserStore } from '@/store/user.js'
import {
	canRequestBindPhoneSms,
	isValidBindPhone,
	resolveBindPhoneSuccess,
	shouldResetBindPhoneVerification
} from '@/utils/bind-phone-flow.js'

export default {
	name: 'BindPhone',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		closable: {
			type: Boolean,
			default: true
		}
	},
	emits: ['close', 'success'],
	data() {
		return {
			phone: '',
			code: '',
			phoneError: '',
			countdown: 0,
			timer: null,
			isSendingCode: false,
			loading: false,
			isAgreed: false,
			isDarkMode: true
		}
	},
	mounted() {
		const savedTheme = uni.getStorageSync('user_theme_dark')
		this.isDarkMode = savedTheme !== false
	},
	computed: {
		canSendCode() {
			return canRequestBindPhoneSms({
				phone: this.phone,
				countdown: this.countdown,
				isSending: this.isSendingCode
			})
		},
		canBind() {
			return isValidBindPhone(this.phone) && this.code.length === 6 && this.isAgreed
		}
	},
	watch: {
		phone(newPhone, oldPhone) {
			if (shouldResetBindPhoneVerification(oldPhone, newPhone)) {
				this.resetVerificationState()
			}
			this.phoneError = ''
		},
		show(value) {
			if (!value) {
				this.resetFormState()
			}
		}
	},
	methods: {
		handlePhoneInput() {},
		
		handleOverlayClick() {
			if (this.closable) {
				this.handleClose()
			}
		},
		
		handleClose() {
			this.resetFormState()
			this.$emit('close')
		},

		resetVerificationState() {
			this.code = ''
			this.countdown = 0
			this.isSendingCode = false
			if (this.timer) {
				clearInterval(this.timer)
				this.timer = null
			}
		},

		resetFormState() {
			this.resetVerificationState()
			this.phone = ''
			this.phoneError = ''
			this.loading = false
			this.isAgreed = false
		},
		
		async handleSendCode() {
			if (!this.canSendCode) return
			
			// 验证手机号
			if (!isValidBindPhone(this.phone)) {
				this.phoneError = '请输入正确的手机号'
				return
			}
			
			this.isSendingCode = true

			try {
				const res = await sendSms(this.phone, 'bind')
				
				if (res.success) {
					uni.showToast({
						title: '验证码已发送',
						icon: 'success'
					})
					
					// 开始倒计时
					this.countdown = 60
					this.timer = setInterval(() => {
						this.countdown--
						if (this.countdown <= 0) {
							clearInterval(this.timer)
							this.timer = null
						}
					}, 1000)
				} else {
					uni.showToast({
						title: res.message || '发送失败',
						icon: 'none'
					})
				}
			} catch (error) {
				console.error('发送验证码失败:', error)
				uni.showToast({
					title: error.message || '发送失败',
					icon: 'none'
				})
			} finally {
				this.isSendingCode = false
			}
		},
		
		async handleBind() {
			if (!this.canBind || this.loading) return
			
			this.loading = true
			
			try {
				const res = await bindPhone(this.phone, this.code)
				
				if (res.success) {
					const userStore = useUserStore()
					const outcome = resolveBindPhoneSuccess({
						response: res,
						phone: this.phone
					})

					if (outcome.action === 'complete') {
						userStore.bindPhoneSuccess(outcome.maskedPhone)
					}

					this.$emit('success', outcome)
				} else {
					uni.showToast({
						title: res.message || '绑定失败',
						icon: 'none'
					})
				}
			} catch (error) {
				console.error('绑定手机号失败:', error)
				uni.showToast({
					title: error.message || '绑定失败',
					icon: 'none'
				})
			} finally {
				this.loading = false
			}
		},
		
		openUserAgreement() {
			uni.navigateTo({
				url: '/subPages/agreement/userAgreement'
			})
		},
		
		openPrivacyPolicy() {
			uni.navigateTo({
				url: '/subPages/agreement/privacyPolicy'
			})
		}
	},
	
	beforeUnmount() {
		this.resetVerificationState()
	}
}
</script>

<style lang="scss" scoped>
// 浅色模式变量
$light-bg: #ffffff;
$light-sheet-bg: #f5f5f5;
$light-text-primary: #1a1a1a;
$light-text-secondary: rgba(0, 0, 0, 0.6);
$light-border: #e5e5e5;
$light-input-bg: #ffffff;

// 深色模式变量
$dark-bg: rgba(0, 0, 0, 0.5);
$dark-sheet-bg: #1e1e1e;
$dark-text-primary: #ffffff;
$dark-text-secondary: rgba(255, 255, 255, 0.7);
$dark-border: #4a4a4a;
$dark-input-bg: transparent;

.bind-phone-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background-color: rgba(0, 0, 0, 0.5);
	z-index: 999;
	display: flex;
	flex-direction: column;
	justify-content: flex-end;

	// 浅色模式样式
	.bind-phone-sheet {
		background-color: $light-sheet-bg;
		border-radius: 24rpx 24rpx 0 0;
		padding: 0 48rpx 64rpx;
	}

	.sheet-header {
		position: relative;
		display: flex;
		justify-content: center;
		padding: 24rpx 0;
	}

	.drag-handle {
		width: 80rpx;
		height: 8rpx;
		background-color: $light-border;
		border-radius: 4rpx;
	}

	.close-btn {
		position: absolute;
		top: 16rpx;
		right: 0;
		width: 56rpx;
		height: 56rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
	}

	.close-icon {
		font-size: 48rpx;
		color: $light-text-secondary;
	}

	.sheet-title {
		display: block;
		font-size: 48rpx;
		font-weight: bold;
		color: $light-text-primary;
		margin-bottom: 8rpx;
	}

	.sheet-subtitle {
		display: block;
		font-size: 28rpx;
		color: $light-text-secondary;
		margin-bottom: 48rpx;
	}

	.input-group {
		margin-bottom: 32rpx;
	}

	.input-label {
		display: block;
		font-size: 28rpx;
		font-weight: 500;
		color: $light-text-primary;
		margin-bottom: 16rpx;
	}

	.input-wrapper {
		display: flex;
		align-items: center;
		height: 112rpx;
		border: 2rpx solid $light-border;
		border-radius: 16rpx;
		background-color: $light-input-bg;
		overflow: hidden;
	}

	.country-code {
		padding-left: 32rpx;
		padding-right: 24rpx;
		font-size: 32rpx;
		color: $light-text-primary;
		border-right: 2rpx solid $light-border;
		margin-right: 24rpx;
	}

	.phone-input,
	.code-input {
		flex: 1;
		height: 100%;
		font-size: 32rpx;
		color: $light-text-primary;
		background-color: transparent;
	}

	.code-input {
		padding: 0 32rpx;
	}

	.code-wrapper {
		position: relative;
	}

	.send-code-btn {
		position: absolute;
		right: 16rpx;
		padding: 16rpx 24rpx;
		font-size: 28rpx;
		font-weight: 500;
		color: #006400;
		border-radius: 12rpx;

		&.disabled {
			color: #9ca3af;
		}
	}

	.error-text {
		display: block;
		font-size: 24rpx;
		color: #ef4444;
		margin-top: 8rpx;
	}

	.bind-btn-container {
		margin-top: 32rpx;
		margin-bottom: 32rpx;
	}

	.bind-btn {
		width: 100%;
		height: 112rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		background-color: #e5e5e5;
		color: #9ca3af;
		font-size: 32rpx;
		font-weight: bold;
		line-height: 1;
		text-align: center;
		border-radius: 24rpx;
		border: none;

		&.active {
			background-color: #006400;
			color: #ffffff;
		}

		&::after {
			display: none;
		}
	}

	.agreement-section {
		margin-top: 24rpx;
	}

	.agreement-row {
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 12rpx;
	}

	.checkbox {
		width: 36rpx;
		height: 36rpx;
		border: 2rpx solid $light-border;
		border-radius: 8rpx;
		margin-right: 12rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s;

		&.checked {
			background-color: #006400;
			border-color: #006400;
		}
	}

	.check-icon {
		font-size: 24rpx;
		color: #ffffff;
	}

	.agreement-label {
		font-size: 24rpx;
		color: $light-text-secondary;
	}

	.agreement-links {
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 24rpx;
		color: #9ca3af;

		.link {
			color: #006400;
		}
	}

	// 深色模式样式
	&.dark-mode {
		.bind-phone-sheet {
			background-color: $dark-sheet-bg;
		}

		.drag-handle {
			background-color: $dark-border;
		}

		.close-icon {
			color: $dark-text-secondary;
		}

		.sheet-title {
			color: $dark-text-primary;
		}

		.sheet-subtitle {
			color: $dark-text-secondary;
		}

		.input-label {
			color: $dark-text-primary;
		}

		.input-wrapper {
			border-color: $dark-border;
			background-color: $dark-input-bg;
		}

		.country-code {
			color: $dark-text-primary;
			border-right-color: $dark-border;
		}

		.phone-input,
		.code-input {
			color: $dark-text-primary;
		}

		.send-code-btn.disabled {
			color: #6b7280;
		}

		.bind-btn {
			background-color: $dark-border;
			color: #6b7280;

			&.active {
				background-color: #006400;
				color: #ffffff;
			}
		}

		.checkbox {
			border-color: $dark-border;
		}

		.agreement-label {
			color: $dark-text-secondary;
		}

		.agreement-links {
			color: #6b7280;

			.link {
				color: rgba(0, 100, 0, 0.8);
			}
		}
	}
}
</style>
