<template>
	<view class="member-center-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="member-center-scroll">
			<view class="member-hero">
				<view class="member-hero-head">
					<text class="member-hero-eyebrow">{{ isComplianceMode ? '会员权益' : '会员中心' }}</text>
					<text class="member-hero-status">{{ summary.statusText }}</text>
				</view>
				<text class="member-hero-title">{{ summary.title }}</text>
				<text class="member-hero-desc">{{ summary.description }}</text>
			</view>

			<view class="member-section member-growth-section">
				<view class="member-section-head member-growth-head">
					<view class="member-growth-head-copy">
						<text class="member-section-title">会员成长</text>
						<text class="member-growth-status">{{ growthCard.statusText }}</text>
					</view>
					<text class="member-growth-level">{{ growthCard.levelLabel }}</text>
				</view>

				<view class="member-growth-metrics">
					<view class="member-growth-metric">
						<text class="member-growth-metric-label">当前进度</text>
						<text class="member-growth-metric-value">{{ growthCard.growthPointsText }}</text>
					</view>
					<view class="member-growth-metric">
						<text class="member-growth-metric-label">今日成长</text>
						<text class="member-growth-metric-value">{{ growthCard.todayProgressText }}</text>
					</view>
				</view>

				<view class="member-growth-progress">
					<view class="member-growth-progress-track">
						<view class="member-growth-progress-fill" :style="{ width: `${growthCard.progressPercent}%` }"></view>
					</view>
					<text class="member-growth-next">{{ growthCard.nextLevelText }}</text>
				</view>

				<text class="member-growth-desc">{{ growthCard.description }}</text>
			</view>

			<view class="member-section member-ranking-section">
				<view class="member-section-head member-ranking-head">
					<view class="member-ranking-head-copy">
						<text class="member-section-title">排位权益 V2</text>
						<text class="member-ranking-status">{{ rankingRightsCard.statusText }}</text>
					</view>
					<view class="member-ranking-badge">
						<text class="member-ranking-badge-level">{{ rankingRightsCard.levelLabel }}</text>
						<text class="member-ranking-badge-rate">{{ rankingRightsCard.currentPercentText }}</text>
					</view>
				</view>

				<text class="member-ranking-headline">{{ rankingRightsCard.headline }}</text>
				<text class="member-ranking-desc">{{ rankingRightsCard.description }}</text>

				<view class="member-ranking-level-grid">
					<view
						v-for="item in rankingRightsCard.levels"
						:key="item.level"
						class="member-ranking-level-card"
						:class="{ active: item.active }"
					>
						<text class="member-ranking-level-label">{{ item.levelLabel }}</text>
						<text class="member-ranking-level-value">{{ item.percentText }}</text>
					</view>
				</view>

				<view class="member-ranking-rule-list">
					<text
						v-for="item in rankingRightsCard.ruleItems"
						:key="item"
						class="member-ranking-rule-item"
					>{{ item }}</text>
				</view>
			</view>

			<view v-if="!isComplianceMode" class="member-section">
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

			<view v-if="!isComplianceMode" class="member-section">
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
					<text class="member-section-title">{{ isComplianceMode ? '权益说明' : '订阅说明' }}</text>
				</view>
				<view class="tips-card">
					<text v-if="isComplianceMode" class="tips-item">新用户注册会直接获赠会员奖励，前台会自动同步会员状态。</text>
					<text v-if="isComplianceMode" class="tips-item">补充常玩球馆并审核通过后，还可继续获赠会员，无需订阅或支付。</text>
					<text v-if="!isComplianceMode" class="tips-item">支付成功后会自动更新会员状态。</text>
					<text v-if="!isComplianceMode" class="tips-item">如果你当前会员仍在有效期内，续费会在现有到期时间基础上顺延。</text>
					<text class="tips-item">会员成长只统计真实完赛对局，每完成 1 场记 1 点成长，每天最多计入 5 场。</text>
					<text class="tips-item">会员到期后成长会冻结但保留，续开会员后会从原进度继续成长。</text>
					<text class="tips-item">普通用户高光会保留记录但不计入排位；会员按当前等级倍率计入特殊战绩排位分。</text>
					<text class="tips-item">会员特殊战绩排位分仅保留单日上限，当日最多计入 200 分。</text>
					<text class="tips-item">所有展示时间统一按 UTC+8 显示。</text>
				</view>
			</view>
		</view>

		<view class="member-action-bar">
			<button
				class="member-submit-btn"
				:disabled="isComplianceMode || submitting || loading || !plans.length"
				@click="handleSubmit"
			>
				<text>{{ submitting ? '拉起支付中...' : summary.primaryActionText }}</text>
			</button>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'

import { getMemberPlans, getMemberStatus, createMemberSubscriptionOrder, getMemberSubscriptionOrderStatus } from '@/api/member.js'
import { APP_COMPLIANCE_MODE } from '@/utils/compliance-mode.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	createMemberPaymentRequest,
	getMemberPayChannelOptions,
	resolveMemberGrowthCard,
	resolveMemberCenterSummary,
	resolveMemberPlanCards,
	shouldTreatMemberOrderAsPaid
} from '@/utils/member-center.js'
import { resolveMemberRankingRightsCard } from '@/utils/member-ranking-rights.js'

const { isDarkMode } = usePageTheme()

const loading = ref(false)
const submitting = ref(false)
const memberStatus = ref(null)
const plans = ref([])
const selectedPlanCode = ref('member_monthly')
const selectedPayChannel = ref('alipay')

const isComplianceMode = APP_COMPLIANCE_MODE
const payChannelOptions = getMemberPayChannelOptions()
const summary = computed(() => resolveMemberCenterSummary(memberStatus.value || {}, new Date(), {
	complianceMode: isComplianceMode
}))
const growthCard = computed(() => resolveMemberGrowthCard(memberStatus.value || {}, new Date()))
const rankingRightsCard = computed(() => resolveMemberRankingRightsCard(memberStatus.value || {}, new Date()))
const planCards = computed(() => resolveMemberPlanCards(plans.value, selectedPlanCode.value))

onShow(() => {
	loadData()
})

const loadData = async () => {
	if (loading.value) return

	loading.value = true
	try {
		const requests = [getMemberStatus()]
		if (!isComplianceMode) {
			requests.unshift(getMemberPlans())
		}

		const responses = await Promise.all(requests)
		const plansRes = isComplianceMode ? null : responses[0]
		const statusRes = isComplianceMode ? responses[0] : responses[1]

		plans.value = plansRes?.success ? plansRes.plans || [] : []
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
	if (isComplianceMode) {
		uni.showToast({
			title: '当前仅展示会员权益',
			icon: 'none'
		})
		return
	}
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
