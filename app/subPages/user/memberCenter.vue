<template>
	<view class="member-center-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="member-center-scroll">
			<view class="member-hero">
				<view class="member-hero-head">
					<text class="member-hero-eyebrow">会员中心</text>
					<text class="member-hero-status">{{ summary.statusText }}</text>
				</view>
				<text class="member-hero-title">{{ summary.title }}</text>
				<text class="member-hero-desc">{{ summary.description }}</text>
			</view>

			<view class="member-section">
				<view class="member-section-head">
					<text class="member-section-title">订阅套餐</text>
					<text class="member-section-tip">目前先开放月卡</text>
				</view>
				<view class="plan-list">
					<view
						v-for="item in planCards"
						:key="item.plan_code"
						class="plan-card"
						:class="{ active: item.selected }"
						@click="handleSelectPlan(item.plan_code)"
					>
						<view class="plan-card-head">
							<view class="plan-card-copy">
								<text class="plan-card-title">{{ item.plan_name }}</text>
								<text class="plan-card-desc">{{ item.description }}</text>
							</view>
							<text v-if="item.highlight" class="plan-card-highlight">{{ item.highlight }}</text>
						</view>
						<view class="plan-card-footer">
							<text class="plan-card-price">{{ item.price_yuan }}</text>
							<text class="plan-card-duration">{{ item.duration_label }}</text>
						</view>
					</view>
				</view>
			</view>

			<view class="member-section">
				<view class="member-section-head">
					<text class="member-section-title">支付方式</text>
				</view>
				<view class="channel-list">
					<view
						v-for="item in payChannelOptions"
						:key="item.value"
						class="channel-card"
						:class="{ active: selectedPayChannel === item.value }"
						@click="handleSelectPayChannel(item.value)"
					>
						<view class="channel-copy">
							<text class="channel-title">{{ item.label }}</text>
							<text class="channel-desc">{{ item.description }}</text>
						</view>
						<text class="channel-check">{{ selectedPayChannel === item.value ? '已选中' : '点击选择' }}</text>
					</view>
				</view>
			</view>

			<view class="member-section">
				<view class="member-section-head">
					<text class="member-section-title">订阅说明</text>
				</view>
				<view class="tips-card">
					<text class="tips-item">支付成功后会自动更新会员状态。</text>
					<text class="tips-item">如果你当前会员仍在有效期内，续费会在现有到期时间基础上顺延。</text>
					<text class="tips-item">所有展示时间统一按 UTC+8 显示。</text>
				</view>
			</view>
		</view>

		<view class="member-action-bar">
			<button class="member-submit-btn" :disabled="submitting || loading || !plans.length" @click="handleSubmit">
				<text>{{ submitting ? '拉起支付中...' : summary.primaryActionText }}</text>
			</button>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'

import { getMemberPlans, getMemberStatus, createMemberSubscriptionOrder, getMemberSubscriptionOrderStatus } from '@/api/member.js'
import { useThemeStore } from '@/store/theme.js'
import {
	createMemberPaymentRequest,
	getMemberPayChannelOptions,
	resolveMemberCenterSummary,
	resolveMemberPlanCards,
	shouldTreatMemberOrderAsPaid
} from '@/utils/member-center.js'

const themeStore = useThemeStore()

const loading = ref(false)
const submitting = ref(false)
const memberStatus = ref(null)
const plans = ref([])
const selectedPlanCode = ref('member_monthly')
const selectedPayChannel = ref('alipay')

const isDarkMode = computed(() => themeStore.isDarkMode)
const payChannelOptions = getMemberPayChannelOptions()
const summary = computed(() => resolveMemberCenterSummary(memberStatus.value || {}))
const planCards = computed(() => resolveMemberPlanCards(plans.value, selectedPlanCode.value))

onShow(() => {
	loadData()
})

const loadData = async () => {
	if (loading.value) return

	loading.value = true
	try {
		const [plansRes, statusRes] = await Promise.all([
			getMemberPlans(),
			getMemberStatus()
		])

		plans.value = plansRes.success ? plansRes.plans || [] : []
		memberStatus.value = statusRes.success ? statusRes : null

		if (plans.value.length > 0 && !plans.value.some(item => item.plan_code === selectedPlanCode.value)) {
			selectedPlanCode.value = plans.value[0].plan_code
		}
	} catch (error) {
		console.error('加载会员中心失败:', error)
		uni.showToast({
			title: '加载会员中心失败',
			icon: 'none'
		})
	} finally {
		loading.value = false
	}
}

const handleSelectPlan = (planCode) => {
	selectedPlanCode.value = planCode
}

const handleSelectPayChannel = (payChannel) => {
	selectedPayChannel.value = payChannel
}

const wait = (ms) => new Promise(resolve => setTimeout(resolve, ms))

const pollOrderStatus = async (orderNo) => {
	for (let attempt = 0; attempt < 6; attempt += 1) {
		const statusRes = await getMemberSubscriptionOrderStatus({ order_no: orderNo })
		if (shouldTreatMemberOrderAsPaid(statusRes)) {
			return statusRes
		}
		if (statusRes.status && statusRes.status !== 'pending') {
			return statusRes
		}
		if (attempt < 5) {
			await wait(1500)
		}
	}

	return null
}

const requestAppPayment = (orderRes) => {
	return new Promise((resolve, reject) => {
		if (orderRes.pay_channel === 'alipay') {
			if (!orderRes.alipay_order_string) {
				reject(new Error('支付宝订单参数缺失'))
				return
			}

			uni.requestPayment({
				provider: 'alipay',
				orderInfo: orderRes.alipay_order_string,
				success: resolve,
				fail: reject
			})
			return
		}

		if (orderRes.pay_channel === 'wechat') {
			const params = orderRes.wechat_app_pay_params
			if (!params) {
				reject(new Error('微信支付参数缺失'))
				return
			}

			uni.requestPayment({
				provider: 'wxpay',
				orderInfo: {
					appid: params.appid,
					partnerid: params.partnerid,
					prepayid: params.prepayid,
					package: params.package,
					noncestr: params.noncestr,
					timestamp: params.timestamp,
					sign: params.sign
				},
				success: resolve,
				fail: reject
			})
			return
		}

		reject(new Error('暂不支持的支付方式'))
	})
}

const handleSubmit = async () => {
	if (submitting.value || loading.value) return
	if (!plans.value.length) {
		uni.showToast({
			title: '暂无可订阅套餐',
			icon: 'none'
		})
		return
	}

	submitting.value = true
	try {
		const orderRes = await createMemberSubscriptionOrder(createMemberPaymentRequest({
			planCode: selectedPlanCode.value,
			payChannel: selectedPayChannel.value
		}))

		if (!orderRes.success) {
			throw new Error(orderRes.message || '创建订单失败')
		}

		await requestAppPayment(orderRes)

		uni.showLoading({
			title: '同步会员状态...'
		})

		const finalStatus = await pollOrderStatus(orderRes.order_no)
		uni.hideLoading()

		if (shouldTreatMemberOrderAsPaid(finalStatus || {})) {
			await loadData()
			uni.showToast({
				title: '会员已开通',
				icon: 'success'
			})
			return
		}

		uni.showToast({
			title: '支付成功，状态同步中',
			icon: 'none'
		})
	} catch (error) {
		uni.hideLoading()
		console.error('会员支付失败:', error)
		const errMsg = String(error?.errMsg || error?.message || '')
		uni.showToast({
			title: /cancel/i.test(errMsg) ? '已取消支付' : (error.message || '支付失败'),
			icon: 'none'
		})
	} finally {
		submitting.value = false
	}
}
</script>

<style lang="scss" scoped>
@import './memberCenter.scss';
</style>
