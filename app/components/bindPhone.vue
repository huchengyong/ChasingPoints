<template>
	<view class="bind-phone-overlay" :class="{ 'dark-mode': displayDarkMode }" v-if="show" @click="handleOverlayClick">
		<!-- 底部弹框 -->
		<view class="bind-phone-sheet" @click.stop>
			<!-- 拖拽条和关闭按钮 -->
			<view class="sheet-header">
				<view class="drag-handle"></view>
				<view class="close-btn" :class="{ disabled: !canClose }" @click="handleClose" v-if="closable">
					<text class="close-icon">×</text>
				</view>
			</view>
			
			<!-- 标题 -->
			<text class="sheet-title">绑定手机号</text>
			<text class="sheet-subtitle">用于账号找回、重要赛事通知和已有账号安全合并</text>
			
			<!-- #ifdef MP-WEIXIN -->
			<template v-if="!useSmsBinding">
				<view class="wechat-phone-copy">
					<text>授权微信手机号后即可完成绑定，可随时稍后处理。</text>
				</view>
				<view class="binding-benefits">
					<text>账号找回</text>
					<text>赛事通知</text>
					<text>账号合并</text>
				</view>

				<view class="bind-btn-container">
					<button
						class="bind-btn"
						:class="{ active: canAuthorizeWechatPhone }"
						:disabled="!canAuthorizeWechatPhone || loading"
						open-type="getPhoneNumber"
						@getphonenumber="handleWechatPhoneNumber"
					>
						{{ loading ? '绑定中...' : '授权微信手机号' }}
					</button>
					<button v-if="closable" class="close-text-btn" :disabled="!canClose" @click="handleClose">
						稍后绑定
					</button>
				</view>
			</template>
			<!-- #endif -->

			<view v-if="showSmsBinding" class="sms-binding-form">
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
						<button
							class="send-code-btn"
							:class="{ disabled: !canSendCode }"
							:disabled="isSendingCode || countdown > 0"
							@click="handleSendCode"
						>
							{{ isSendingCode ? '发送中...' : (countdown > 0 ? `${countdown}s后重发` : '获取验证码') }}
						</button>
					</view>
					<text v-if="slowAction === 'send'" class="slow-action-hint">网络稍慢，正在继续尝试</text>
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
					<button v-if="closable" class="close-text-btn" :disabled="!canClose" @click="handleClose">
						稍后绑定
					</button>
				</view>
				<text v-if="slowAction === 'bind'" class="slow-action-hint">网络稍慢，正在继续尝试</text>
			</view>
			
			<!-- 协议文本 -->
			<view v-if="requireAgreement" class="agreement-section">
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
</template>

<script>
import { sendSms, bindPhone, wechatMiniBindPhone } from '@/api/auth.js'
import { useUserStore } from '@/store/user.js'
import {
	canCloseBindPhone,
	canRequestBindPhoneSms,
	getWechatPhoneNumberCode,
	isValidBindPhone,
	resolveBindPhoneSuccess,
	shouldResetBindPhoneVerification
} from '@/utils/bind-phone-flow.js'
import { AUTH_SLOW_FEEDBACK_DELAY, resolveEntryFunnelAgreementState } from '@/utils/entry-funnel.js'

const ENTRY_FUNNEL_AGREEMENT_KEY = 'entry_funnel_agreement_accepted'
const ENTRY_FUNNEL_SESSION_KEY = 'entry_funnel_session_active'

export default {
	name: 'BindPhone',
	props: {
		show: {
			type: Boolean,
			default: false
		},
		isDarkMode: {
			type: Boolean,
			default: false
		},
		closable: {
			type: Boolean,
			default: true
		},
		requireAgreement: {
			type: Boolean,
			default: true
		},
		useSmsBinding: {
			type: Boolean,
			default: false
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
			slowAction: '',
			slowTimer: null,
			isAgreed: false
		}
	},
	mounted() {
		this.syncAgreementState()
	},
	computed: {
		displayDarkMode() {
			return this.isDarkMode
		},
		showSmsBinding() {
			let isWechatMiniProgram = false
			// #ifdef MP-WEIXIN
			isWechatMiniProgram = true
			// #endif
			return !isWechatMiniProgram || this.useSmsBinding
		},
		canSendCode() {
			return canRequestBindPhoneSms({
				phone: this.phone,
				countdown: this.countdown,
				isSending: this.isSendingCode
			})
		},
		canAuthorizeWechatPhone() {
			return !this.requireAgreement || this.isAgreed
		},
		canClose() {
			return canCloseBindPhone({ isBinding: this.loading })
		},
		canBind() {
			return isValidBindPhone(this.phone) && this.code.length === 6 && this.canAuthorizeWechatPhone
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
			if (value) {
				this.syncAgreementState()
				return
			}

			if (!value) {
				this.resetFormState()
			}
		}
	},
	methods: {
		handlePhoneInput() {},

		syncAgreementState() {
			const sessionActive = Boolean(uni.getStorageSync(ENTRY_FUNNEL_SESSION_KEY))
			this.isAgreed = resolveEntryFunnelAgreementState({
				storedAgreement: uni.getStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY),
				sessionActive
			})
		},

		persistAgreementState(value) {
			uni.setStorageSync(ENTRY_FUNNEL_AGREEMENT_KEY, Boolean(value))
		},

		toggleAgreement() {
			this.isAgreed = !this.isAgreed
			this.persistAgreementState(this.isAgreed)
		},
		
		handleOverlayClick() {
			if (this.closable) {
				this.handleClose()
			}
		},
		
		handleClose() {
			if (!this.canClose) return
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
			this.clearSlowAction()
			this.isAgreed = false
		},

		startSlowAction(action) {
			this.clearSlowAction()
			this.slowTimer = setTimeout(() => {
				if (this.isSendingCode || this.loading) this.slowAction = action
			}, AUTH_SLOW_FEEDBACK_DELAY)
		},

		clearSlowAction() {
			if (this.slowTimer) {
				clearTimeout(this.slowTimer)
				this.slowTimer = null
			}
			this.slowAction = ''
		},
		
		async handleSendCode() {
			if (this.isSendingCode || this.countdown > 0) return
			
			if (!isValidBindPhone(this.phone)) {
				this.phoneError = this.phone ? '请输入正确的手机号' : '请输入手机号'
				uni.showToast({
					title: this.phoneError,
					icon: 'none'
				})
				return
			}
			
			this.isSendingCode = true
			this.startSlowAction('send')

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
				this.clearSlowAction()
			}
		},
		
		async handleBind() {
			if (!this.canBind || this.loading) return
			
			this.loading = true
			this.startSlowAction('bind')
			
			try {
				const res = await bindPhone(this.phone, this.code)
				
				this.handleBindResponse(res, this.phone)
			} catch (error) {
				console.error('绑定手机号失败:', error)
				uni.showToast({
					title: error.message || '绑定失败',
					icon: 'none'
				})
			} finally {
				this.loading = false
				this.clearSlowAction()
			}
		},

		async handleWechatPhoneNumber(event) {
			const code = getWechatPhoneNumberCode(event)
			if (!code) {
				uni.showToast({
					title: '未授权手机号，可稍后绑定',
					icon: 'none'
				})
				return
			}
			if (!this.canAuthorizeWechatPhone || this.loading) return

			this.loading = true
			this.startSlowAction('bind')
			try {
				const res = await wechatMiniBindPhone(code)
				this.handleBindResponse(res, '')
			} catch (error) {
				uni.showToast({
					title: error.message || '绑定失败',
					icon: 'none'
				})
			} finally {
				this.loading = false
				this.clearSlowAction()
			}
		},

		handleBindResponse(res, phone) {
			if (!res.success) {
				uni.showToast({
					title: res.message || '绑定失败',
					icon: 'none'
				})
				return
			}

			const userStore = useUserStore()
			const outcome = resolveBindPhoneSuccess({ response: res, phone })
			if (outcome.sessionReplaced) {
				userStore.login(res)
			} else if (outcome.action === 'complete') {
				userStore.bindPhoneSuccess(outcome.maskedPhone)
			}

			this.$emit('success', outcome)
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
		this.clearSlowAction()
	}
}
</script>

<style lang="scss" scoped>
// 浅色模式变量
$light-bg: #ffffff;
$light-sheet-bg: #fffdf9;
$light-text-primary: #1a1a1a;
$light-text-secondary: rgba(0, 0, 0, 0.6);
$light-border: #e5e5e5;
$light-input-bg: #ffffff;

// 深色模式变量
$dark-bg: rgba(0, 0, 0, 0.5);
$dark-sheet-bg: #1b170f;
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
		border-radius: 32rpx 32rpx 0 0;
		padding: 0 40rpx calc(40rpx + env(safe-area-inset-bottom));
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

		&.disabled {
			opacity: 0.45;
			pointer-events: none;
		}
	}

	.close-icon {
		font-size: 48rpx;
		color: $light-text-secondary;
	}

	.sheet-title {
		display: block;
		font-size: 40rpx;
		font-weight: bold;
		color: $light-text-primary;
		margin-bottom: 12rpx;
	}

	.sheet-subtitle {
		display: block;
		font-size: 26rpx;
		line-height: 1.6;
		color: $light-text-secondary;
		margin-bottom: 20rpx;
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
		min-width: 0;
		padding: 0 16rpx 0 32rpx;
	}

	.code-wrapper {
		position: relative;
	}

	.send-code-btn {
		position: static;
		flex-shrink: 0;
		height: 64rpx;
		line-height: 64rpx;
		margin: 0 16rpx 0 0;
		padding: 0 24rpx;
		font-size: 28rpx;
		font-weight: 500;
		color: #c69200;
		background: transparent;
		border: none;
		border-radius: 12rpx;
		box-sizing: border-box;
		white-space: nowrap;

		&::after {
			display: none;
		}

		&.disabled {
			color: #9ca3af;
		}

		&[disabled] {
			color: #9ca3af !important;
			background: transparent;
		}
	}

	.error-text {
		display: block;
		font-size: 24rpx;
		color: #ef4444;
		margin-top: 8rpx;
	}

	.slow-action-hint {
		display: block;
		margin-top: 10rpx;
		color: #9a7b2a;
		font-size: 22rpx;
		line-height: 1.4;
		text-align: center;
	}

	.bind-btn-container {
		margin-top: 32rpx;
		margin-bottom: 32rpx;
	}

	.bind-btn {
		width: 100%;
		height: 96rpx;
		line-height: 96rpx;
		margin: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		background-color: #e5e5e5;
		color: #9ca3af;
		font-size: 32rpx;
		font-weight: bold;
		text-align: center;
		border-radius: 24rpx;
		border: none;

		&.active {
			background: linear-gradient(135deg, #e0ae12 0%, #c69200 100%);
			color: #ffffff;
		}

		&[disabled] {
			background-color: #e5e5e5;
			color: #9ca3af;
		}

		&::after {
			display: none;
		}
	}

	.wechat-phone-copy {
		display: block;
		font-size: 24rpx;
		line-height: 1.6;
		color: $light-text-secondary;
	}

	.binding-benefits {
		display: flex;
		gap: 12rpx;
		margin-top: 20rpx;

		text {
			flex: 1;
			padding: 14rpx 8rpx;
			border-radius: 14rpx;
			background: #f8f3e8;
			color: #766a52;
			font-size: 22rpx;
			text-align: center;
		}
	}

	.close-text-btn {
		width: 100%;
		height: 72rpx;
		line-height: 72rpx;
		margin-top: 20rpx;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		background: transparent;
		color: #8a6510;
		font-size: 26rpx;
		font-weight: 600;
		border: none;

		&::after {
			display: none;
		}

		&[disabled] {
			opacity: 0.45;
		}
	}

	.agreement-section {
		margin-top: 24rpx;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10rpx;
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
		transition: all 0.2s;

		&.checked {
			background: #f7d86a;
			border-color: #f7d86a;
		}
	}

	.check-icon {
		font-size: 24rpx;
		color: #ffffff;
	}

	.agreement-text {
		font-size: 24rpx;
		color: #6e6242;
	}

	.agreement-links {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-wrap: wrap;
		font-size: 24rpx;
		color: #9a8c67;

		.link {
			color: #8a6510;
			margin: 0 8rpx;
			text-decoration: underline;
		}
	}

	.separator {
		color: #9a8c67;
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

		.wechat-phone-copy {
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
				background: linear-gradient(135deg, #e5b928 0%, #c99700 100%);
				color: #ffffff;
			}
		}

		.binding-benefits text {
			background: rgba(255, 255, 255, 0.05);
			color: #d7c89b;
		}

		.close-text-btn {
			color: #f7e7a8;
		}

		.checkbox {
			background: rgba(255, 247, 225, 0.06);
			border-color: rgba(247, 231, 168, 0.18);
		}

		.agreement-text {
			color: #c6b78c;
		}

		.agreement-links {
			color: #9a8c67;

			.link {
				color: #f7e7a8;
			}
		}

		.slow-action-hint {
			color: #d7b95c;
		}
	}
}
</style>
