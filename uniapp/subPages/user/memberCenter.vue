<template>
	<view class="member-center-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="member-center-scroll">
			<view class="member-hero">
				<view class="member-hero-head">
					<text class="member-hero-eyebrow">{{ isComplianceMode ? 'VIP权益' : 'VIP中心' }}</text>
					<text class="member-hero-status">{{ summary.statusText }}</text>
				</view>
				<text class="member-hero-title">{{ summary.title }}</text>
				<text class="member-hero-desc">{{ summary.description }}</text>
			</view>

			<view class="member-section member-identity-section">
				<view class="member-section-head">
					<text class="member-section-title">身份展示</text>
					<text v-if="identityPreview.active" class="member-section-tip">{{ identityPreview.identity.tierLabel }}外观</text>
				</view>
				<view class="member-identity-preview">
					<memberAvatar
						class="member-identity-avatar"
						:avatar="ownAvatar"
						:user-id="userStore.userId"
						:vip-level="identityPreview.identity.vipLevel"
						:rank-level="rankLevel"
						:size="112"
						reserve-frame
					/>
					<view class="member-identity-copy">
						<text class="member-identity-name">{{ ownName }}</text>
						<text class="member-identity-hint">{{ identityPreview.hint }}</text>
					</view>
				</view>
				<view class="member-tier-list">
					<view
						v-for="item in identityPreview.tiers"
						:key="item.tier"
						class="member-tier-row"
						:class="{ current: item.current, reached: item.reached && identityPreview.active }"
					>
						<memberAvatar class="member-tier-avatar" :avatar="ownAvatar" :user-id="userStore.userId" :vip-level="item.sampleLevel" :size="64" />
						<view class="member-tier-copy">
							<text class="member-tier-name">{{ item.label }}</text>
							<text class="member-tier-levels">成长 {{ item.levelText }}</text>
						</view>
						<text v-if="item.current" class="member-tier-current">当前</text>
					</view>
				</view>
				<text class="member-identity-note">成长等级解锁更高阶外观。外观只是展示身份，不影响排位分，也不影响对局功能。</text>
			</view>

			<view class="member-section member-analysis-section">
				<view class="member-section-head">
					<text class="member-section-title">深度分析</text>
					<text class="member-section-tip">会员权益</text>
				</view>
				<view class="member-analysis-row" @click="openPerformance">
					<view class="member-analysis-copy">
						<text class="member-analysis-title">表现分析</text>
						<text class="member-analysis-desc">按对手实力反推近 90 天真实水平，附 6 个月实力曲线</text>
					</view>
					<uni-icons type="right" size="16" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
				</view>
				<view class="member-analysis-row" @click="openMatchHistory">
					<view class="member-analysis-copy">
						<text class="member-analysis-title">单场复盘</text>
						<text class="member-analysis-desc">在比赛记录里打开已完成的对局，回看每局走势与关键局</text>
					</view>
					<uni-icons type="right" size="16" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
				</view>
			</view>

			<view class="member-section member-growth-section">
				<view class="member-section-head member-growth-head">
					<view class="member-growth-head-copy">
						<text class="member-section-title">VIP成长</text>
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
					<text v-if="isComplianceMode" class="tips-item">VIP权益由活动发放，到账后会自动同步到这里，无需订阅或支付。</text>
					<text v-if="isComplianceMode && venueRewardClaimable" class="tips-item">补充常玩球馆并审核通过后，可获得 {{ venueRewardText }}，无需订阅或支付。</text>
					<text v-if="!isComplianceMode" class="tips-item">支付成功后会自动更新VIP状态。</text>
					<text v-if="!isComplianceMode" class="tips-item">如果你当前VIP仍在有效期内，续费会在现有到期时间基础上顺延。</text>
					<text class="tips-item">VIP成长只统计真实完赛对局，每天最多计入 {{ growthCard.dailyCap }} 场。</text>
					<text class="tips-item">VIP到期后成长会冻结但保留，续开VIP后会从原进度继续成长。</text>
					<text class="tips-item">VIP不影响排位分：排位分只看胜负、对手实力与赛制长度，会员与普通用户规则相同；高光仍会记录在战报和成就里。</text>
					<text class="tips-item">对局的基本流程对所有用户完整开放，VIP只提供身份展示与深度分析。</text>
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

import { getMemberPlans, createMemberSubscriptionOrder, getMemberSubscriptionOrderStatus } from '@/api/member.js'
import memberAvatar from '@/components/memberAvatar.vue'
import { usePublicReadStore } from '@/store/publicRead.js'
import { useRankStore } from '@/store/rank.js'
import { useUserOverviewStore } from '@/store/userOverview.js'
import { useUserStore } from '@/store/user.js'
import { APP_COMPLIANCE_MODE } from '@/utils/compliance-mode.js'
import { formatRewardDuration, isFavoriteVenueRewardClaimable } from '@/utils/favorite-venue-reward.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	createMemberPaymentRequest,
	getMemberPayChannelOptions,
	resolveMemberGrowthCard,
	resolveMemberCenterSummary,
	resolveMemberIdentityPreview,
	resolveMemberPlanCards,
	shouldTreatMemberOrderAsPaid
} from '@/utils/member-center.js'
import { readDefaultGameType } from '@/utils/game-type-preference.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const publicReadStore = usePublicReadStore()
const userOverviewStore = useUserOverviewStore()
const rankStore = useRankStore()
const userStore = useUserStore()

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
// 头像框跟随默认球种的当前段位；段位还没读到时只显示 VIP 标识，不显示框。
const previewGameType = computed(() => readDefaultGameType(uni, userStore.userId) || 3)
const rankLevel = computed(() => Number(rankStore.rankInfoMap?.[previewGameType.value]?.level) || 0)
const identityPreview = computed(() => resolveMemberIdentityPreview(memberStatus.value || {}, new Date(), {
	rankLevel: rankLevel.value,
	complianceMode: isComplianceMode
}))
const ownAvatar = computed(() => resolveAvatarUrl(userStore.userInfo?.avatar, userStore.userId))
const ownName = computed(() => userStore.userInfo?.nickname || '我')
const planCards = computed(() => resolveMemberPlanCards(plans.value, selectedPlanCode.value))
const venueRewardStatus = computed(() => userOverviewStore.favoriteVenueRewardStatus || {})
const venueRewardClaimable = computed(() => isFavoriteVenueRewardClaimable(venueRewardStatus.value))
const venueRewardText = computed(() => formatRewardDuration(venueRewardStatus.value.reward_days || 30))

onShow(() => {
	loadData()
})

const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const loadData = async ({ force = false } = {}) => {
	if (loading.value) return

	loading.value = true
	try {
		const [overview, plansRes] = await Promise.all([
			userOverviewStore.fetch(getReadIdentity(), { force, silent: true }),
			isComplianceMode
				? Promise.resolve(null)
				: publicReadStore.loadStatic('member-plans', getMemberPlans),
			// 段位只用于选择头像框，读取失败不影响会员中心
			rankStore.ensureFresh(getReadIdentity()).catch(() => null)
		])
		if (overview?.memberStatus?.success) {
			memberStatus.value = overview.memberStatus
		}
		if (plansRes?.success) {
			plans.value = plansRes.plans || []
		}

		if (plans.value.length > 0 && !plans.value.some(item => item.plan_code === selectedPlanCode.value)) {
			selectedPlanCode.value = plans.value[0].plan_code
		}
	} catch (error) {
		console.error('加载VIP中心失败:', error)
		if (!memberStatus.value && !plans.value.length) {
			uni.showToast({
				title: '加载VIP中心失败',
				icon: 'none'
			})
		}
	} finally {
		loading.value = false
	}
}

const openPerformance = () => {
	uni.navigateTo({ url: '/subPages/user/performance' })
}

const openMatchHistory = () => {
	uni.navigateTo({ url: '/subPages/user/matchHistory' })
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
			title: '当前仅展示VIP权益',
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
			title: '同步VIP状态...'
		})

		const finalStatus = await pollOrderStatus(orderRes.order_no)
		uni.hideLoading()

		if (shouldTreatMemberOrderAsPaid(finalStatus || {})) {
			userOverviewStore.markDirty()
			await loadData({ force: true })
			uni.showToast({
				title: 'VIP已开通',
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
		console.error('VIP支付失败:', error)
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
