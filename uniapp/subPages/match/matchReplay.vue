<template>
	<view class="replay-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="isInitialLoading" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-title">加载单场复盘...</text>
		</view>

		<view v-else-if="pageError" class="state-block">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="state-title">{{ pageError.title }}</text>
			<text class="state-desc">{{ pageError.description }}</text>
			<button class="state-action" @tap="retry">{{ pageError.actionText }}</button>
		</view>

		<template v-else-if="view">
			<view v-if="refreshError" class="refresh-error-banner">
				<text>{{ refreshError.description }}</text>
				<text class="refresh-error-action" @tap="retry">重试</text>
			</view>

			<!-- 锁定态：只有局数，没有任何复盘明细 -->
			<template v-if="view.locked">
				<view class="card hero-card">
					<text class="hero-kicker">会员权益</text>
					<text class="hero-title">单场复盘</text>
					<text class="hero-desc">回看这场{{ view.gameTypeLabel }}每一局的走势、关键局和胜局方式。</text>
					<text class="hero-meta">这场共打了 {{ view.frameCount }} 局</text>
				</view>
				<view class="card">
					<text class="card-title">完整复盘包含</text>
					<view v-for="feature in view.features" :key="feature" class="feature-row">
						<uni-icons type="checkmarkempty" size="16" color="#E0AE12"></uni-icons>
						<text class="feature-text">{{ feature }}</text>
					</view>
					<button v-if="view.showPurchase" class="state-action unlock-action" @tap="openMemberCenter">开通会员</button>
					<text v-else class="feature-footnote">完整复盘为会员权益</text>
				</view>
			</template>

			<template v-else>
				<view class="card hero-card">
					<text class="hero-kicker">{{ view.title }}</text>
					<text class="hero-score">{{ view.scoreText }}</text>
					<text class="hero-meta">共 {{ view.frameCount }} 局 · 左为我，右为对手</text>
				</view>

				<view v-if="view.notice" class="notice-banner"><text>{{ view.notice }}</text></view>

				<view v-if="view.keyFrames.length" class="card">
					<text class="card-title">关键局</text>
					<view v-for="frame in view.keyFrames" :key="frame.frameNo" class="key-row">
						<text class="key-frame">{{ frame.title }}</text>
						<view class="tag-list">
							<text v-for="tag in frame.tags" :key="tag.key" class="tag" :class="`tag-${tag.key}`">{{ tag.label }}</text>
						</view>
						<text class="key-detail">{{ frame.winnerText }}{{ frame.scoreText ? ` · ${frame.scoreText}` : '' }}{{ frame.deficitText ? ` · ${frame.deficitText}` : '' }}</text>
					</view>
				</view>

				<view v-if="view.summaryRows.length" class="card">
					<text class="card-title">整场数据</text>
					<view class="compare-head"><text>我</text><text class="compare-label"></text><text>对手</text></view>
					<view v-for="row in view.summaryRows" :key="row.label" class="compare-row">
						<text class="compare-value">{{ row.mine }}</text>
						<text class="compare-label">{{ row.label }}</text>
						<text class="compare-value">{{ row.opponent }}</text>
					</view>
				</view>

				<view v-if="view.sequence.length" class="card">
					<text class="card-title">逐局胜负</text>
					<view class="sequence">
						<view v-for="item in view.sequence" :key="item.frameNo" class="sequence-item" :class="`winner-${item.winner}`">
							<text>{{ item.frameNo }}</text>
						</view>
					</view>
					<text class="sequence-legend">金色为我赢，灰色为对手赢</text>
					<view v-if="view.winTypeRows.length" class="win-types">
						<view class="compare-head"><text>我</text><text class="compare-label">胜局方式</text><text>对手</text></view>
						<view v-for="row in view.winTypeRows" :key="row.label" class="compare-row">
							<text class="compare-value">{{ row.mine }}</text>
							<text class="compare-label">{{ row.label }}</text>
							<text class="compare-value">{{ row.opponent }}</text>
						</view>
					</view>
				</view>

				<view v-if="view.frames.length" class="card">
					<text class="card-title">每局详情</text>
					<view v-for="frame in view.frames" :key="frame.frameNo" class="frame-block">
						<view class="frame-head">
							<text class="frame-title">{{ frame.title }}</text>
							<text class="frame-winner" :class="`winner-${frame.winner}`">{{ frame.winnerText }}</text>
							<text v-if="frame.scoreText" class="frame-score">{{ frame.scoreText }}</text>
							<text v-if="frame.winTypeText" class="frame-win-type">{{ frame.winTypeText }}</text>
							<text v-for="tag in frame.tags" :key="tag.key" class="tag" :class="`tag-${tag.key}`">{{ tag.label }}</text>
						</view>
						<text v-if="frame.durationText || frame.deficitText" class="frame-meta">{{ [frame.durationText, frame.deficitText].filter(Boolean).join(' · ') }}</text>

						<view v-if="frame.trend.hasData" class="trend-chart">
							<view class="trend-body">
								<view class="trend-zero"></view>
								<view v-for="(column, index) in frame.trend.columns" :key="index" class="trend-col">
									<view class="trend-half upper">
										<view v-if="column.direction === 'up'" class="trend-bar up" :style="{ height: `${column.heightPercent}%` }"></view>
									</view>
									<view class="trend-half lower">
										<view v-if="column.direction === 'down'" class="trend-bar down" :style="{ height: `${column.heightPercent}%` }"></view>
									</view>
								</view>
							</view>
							<text class="trend-legend">向上为我领先，向下为对手领先（最大分差 {{ frame.trend.maxAbs }}）</text>
						</view>

						<view v-for="item in frame.details" :key="item.label" class="compare-row small">
							<text class="compare-value">{{ item.mine }}</text>
							<text class="compare-label">{{ item.label }}</text>
							<text class="compare-value">{{ item.opponent }}</text>
						</view>
					</view>
				</view>

				<text v-if="view.empty" class="empty-hint">这场对局没有可展示的局记录</text>
			</template>
		</template>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getMemberMatchReplay } from '@/api/member.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { canShowPaidMembershipEntry } from '@/utils/compliance-mode.js'
import { resolveReplayView } from '@/utils/member-replay-view.js'
import { createRequestError } from '@/utils/request-errors.js'
import {
	ASYNC_PAGE_STATUS,
	beginAsyncPageLoad,
	createAsyncPageState,
	getAsyncPageRequest,
	rejectAsyncPageLoad,
	resolveAsyncPageErrorFeedback,
	resolveAsyncPageLoad
} from '@/utils/async-page-state.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const userDataInvalidationStore = useUserDataInvalidationStore()

const matchId = ref(0)
const loadedKey = ref('')
// 同一身份、同一对局、同一会员版本的读取正在进行时，重复进入页面共用它，不再发第二个请求。
let inFlightKey = ''
const pageState = ref(createAsyncPageState({ authGeneration: userStore.authGeneration, data: null }))

const isInitialLoading = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.LOADING || pageState.value.status === ASYNC_PAGE_STATUS.IDLE
))
const pageError = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(pageState.value.error, { resource: '单场复盘' })
		: null
))
const refreshError = computed(() => (
	pageState.value.refreshError
		? resolveAsyncPageErrorFeedback(pageState.value.refreshError, { resource: '单场复盘' })
		: null
))
const view = computed(() => (
	pageState.value.data
		? resolveReplayView(pageState.value.data, { canShowPurchase: canShowPaidMembershipEntry() })
		: null
))

// 会员状态变化（开通或到期）后，锁定与否会变化，下次进入页面回源。
const currentKey = () => `${userStore.userId}:${userStore.authGeneration}:${matchId.value}:${userDataInvalidationStore.versionOf('member')}`

const goLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}

const loadReplay = async () => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}
	const authGeneration = userStore.authGeneration
	const key = currentKey()
	if (inFlightKey === key) return
	inFlightKey = key
	const nextState = beginAsyncPageLoad(pageState.value, { authGeneration, emptyData: null })
	const request = getAsyncPageRequest(nextState)
	pageState.value = nextState
	try {
		const response = await getMemberMatchReplay({ match_id: matchId.value })
		// 服务端对不存在、未完成、非参与者统一返回 success=false，不区分原因。
		if (!response?.success) {
			throw createRequestError({ message: '暂时无法查看这场对局的复盘', category: 'not-found' })
		}
		const applied = resolveAsyncPageLoad(pageState.value, request, { data: response, isEmpty: () => false })
		if (applied === pageState.value) return
		loadedKey.value = key
		pageState.value = applied
	} catch (error) {
		pageState.value = rejectAsyncPageLoad(pageState.value, request, error)
	} finally {
		if (inFlightKey === key) inFlightKey = ''
	}
}

const retry = () => {
	loadReplay()
}

const openMemberCenter = () => {
	uni.navigateTo({ url: '/subPages/user/memberCenter' })
}

onLoad((options) => {
	matchId.value = Math.max(0, parseInt(options?.match_id, 10) || 0)
})

onShow(() => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}
	if (!matchId.value) {
		const nextState = beginAsyncPageLoad(pageState.value, { authGeneration: userStore.authGeneration, emptyData: null })
		pageState.value = rejectAsyncPageLoad(
			nextState,
			getAsyncPageRequest(nextState),
			createRequestError({ message: '缺少对局信息', category: 'not-found' })
		)
		return
	}
	if (loadedKey.value === currentKey() && pageState.value.status !== ASYNC_PAGE_STATUS.ERROR) return
	loadReplay()
})
</script>

<style lang="scss" scoped>
@import './matchReplay.scss';
</style>
