<template>
	<view class="performance-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="game-tabs">
			<view
				v-for="tab in gameTabs"
				:key="tab.value"
				class="tab-item"
				:class="{ active: currentGameType === tab.value }"
				@tap="handleGameChange(tab.value)"
			>
				<text>{{ tab.label }}</text>
			</view>
		</view>

		<view v-if="isInitialLoading" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-title">加载表现分析...</text>
		</view>

		<view v-else-if="pageError" class="state-block">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="state-title">{{ pageError.title }}</text>
			<text class="state-desc">{{ pageError.description }}</text>
			<button class="state-action" @tap="retry">{{ pageError.actionText }}</button>
		</view>

		<template v-else-if="data">
			<view v-if="refreshError" class="refresh-error-banner">
				<text>{{ refreshError.description }}</text>
				<text class="refresh-error-action" @tap="retry">重试</text>
			</view>

			<!-- 锁定态：只有服务端给出的样本数与本月预览 -->
			<template v-if="lockedView">
				<view class="card hero-card">
					<text class="hero-kicker">会员权益</text>
					<text class="hero-title">看清你真实打出的水平</text>
					<text class="hero-desc">表现分按你每场的对手实力与赛制长度，反推出这段时间的实际水平。</text>
					<text v-if="lockedView.hasSamples" class="hero-meta">近 90 天已有 {{ lockedView.sampleText }} 场可分析的排位</text>
					<text v-else class="hero-meta">完成排位对局后开始积累分析样本</text>
				</view>

				<view class="card preview-card">
					<text class="card-title">{{ lockedView.preview.monthText }}表现分预览</text>
					<view class="preview-rating">
						<text v-if="lockedView.preview.qualifier" class="rating-qualifier">{{ lockedView.preview.qualifier }}</text>
						<text class="rating-value" :class="{ empty: !lockedView.preview.available }">{{ lockedView.preview.available ? lockedView.preview.value : '—' }}</text>
					</view>
					<text class="preview-hint">{{ lockedView.preview.hint }}</text>
				</view>

				<view class="card">
					<text class="card-title">完整分析包含</text>
					<view v-for="feature in lockedView.features" :key="feature" class="feature-row">
						<uni-icons type="checkmarkempty" size="16" color="#E0AE12"></uni-icons>
						<text class="feature-text">{{ feature }}</text>
					</view>
					<button v-if="lockedView.showPurchase" class="state-action unlock-action" @tap="openMemberCenter">开通会员</button>
					<text v-else class="feature-footnote">完整分析为会员权益</text>
				</view>
			</template>

			<!-- 会员态 -->
			<template v-else>
				<view class="card summary-card">
					<text class="card-title">近 90 天表现分</text>
					<view v-if="summaryView.rating.available" class="rating-block">
						<text v-if="summaryView.rating.qualifier" class="rating-qualifier">{{ summaryView.rating.qualifier }}</text>
						<text class="rating-value">{{ summaryView.rating.value }}</text>
					</view>
					<view v-else class="rating-block">
						<text class="rating-value empty">—</text>
					</view>
					<text v-if="summaryView.ratingHint" class="rating-hint">{{ summaryView.ratingHint }}</text>

					<view v-if="summaryView.available" class="wins-row">
						<view class="wins-cell">
							<text class="wins-value">{{ summaryView.wins }}</text>
							<text class="wins-label">实际胜场</text>
						</view>
						<view class="wins-cell">
							<text class="wins-value">{{ summaryView.expectedText }}</text>
							<text class="wins-label">期望胜场</text>
						</view>
						<view class="wins-cell">
							<text class="wins-value" :class="`tone-${summaryView.tone}`">{{ summaryView.diffText }}</text>
							<text class="wins-label">差值</text>
						</view>
					</view>
					<text v-if="summaryView.available" class="summary-headline">{{ summaryView.headline }}（{{ summaryView.sampleText }} 场样本）</text>
					<text v-else class="rating-hint">近 90 天还没有可分析的排位样本</text>
				</view>

				<view v-if="notableViews.length" class="card">
					<text class="card-title">值得回看的两场</text>
					<view
						v-for="item in notableViews"
						:key="item.kind"
						class="notable-row"
						@tap="openMatch(item.matchId)"
					>
						<view class="notable-copy">
							<text class="notable-title" :class="item.kind === 'best_win' ? 'tone-up' : 'tone-down'">{{ item.title }}</text>
							<text class="notable-headline">{{ item.headline }}</text>
							<text class="notable-detail">{{ item.detail }}</text>
							<text v-if="item.playedAt" class="notable-time">{{ item.playedAt }}</text>
						</view>
						<uni-icons type="right" size="16" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
					</view>
				</view>

				<view class="card">
					<text class="card-title">最近 6 个月实力曲线</text>
					<view v-if="chart.hasData" class="month-chart">
						<view class="chart-plot">
							<view v-for="bar in chart.bars" :key="bar.month" class="chart-col">
								<view class="chart-area">
									<view v-if="bar.status === 'ready'" class="chart-bar" :class="{ current: bar.isCurrent }" :style="{ height: `${bar.heightPercent}%` }"></view>
									<view v-if="bar.dotPercent !== null" class="chart-dot" :style="{ bottom: `${bar.dotPercent}%` }"></view>
								</view>
								<text class="chart-label">{{ bar.isCurrent ? '本月' : bar.label }}</text>
								<text class="chart-value" :class="{ muted: bar.status !== 'ready' }">{{ bar.ratingText }}</text>
								<text v-if="bar.note" class="chart-note">{{ bar.note }}</text>
							</view>
						</view>
						<text class="chart-footnote">{{ monthFootnote }}</text>
						<view class="chart-legend">
							<view class="legend-item"><view class="legend-swatch bar"></view><text>当月表现分</text></view>
							<view class="legend-item"><view class="legend-swatch dot"></view><text>月末排位分</text></view>
						</view>
					</view>
					<text v-else class="empty-hint">最近 6 个月还没有可画出曲线的排位记录</text>
				</view>
			</template>

			<text class="performance-note">{{ note }}</text>
		</template>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useMemberPerformanceStore } from '@/store/memberPerformance.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { canShowPaidMembershipEntry } from '@/utils/compliance-mode.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import {
	PERFORMANCE_MONTH_FOOTNOTE,
	PERFORMANCE_NOTE,
	buildMonthChart,
	resolveLockedView,
	resolveNotableMatchView,
	resolvePerformanceVersion,
	resolveSummaryView
} from '@/utils/member-performance-view.js'
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
const performanceStore = useMemberPerformanceStore()
const userDataInvalidationStore = useUserDataInvalidationStore()

const gameTabs = GAME_TYPE_TABS
const note = PERFORMANCE_NOTE
const monthFootnote = PERFORMANCE_MONTH_FOOTNOTE
const currentGameType = ref(3)
const loadedKey = ref('')
const pageState = ref(createAsyncPageState({ authGeneration: userStore.authGeneration, data: null }))

const data = computed(() => pageState.value.data)
const isInitialLoading = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.LOADING || pageState.value.status === ASYNC_PAGE_STATUS.IDLE
))
const pageError = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(pageState.value.error, { resource: '表现分析' })
		: null
))
const refreshError = computed(() => (
	pageState.value.refreshError
		? resolveAsyncPageErrorFeedback(pageState.value.refreshError, { resource: '表现分析' })
		: null
))

const lockedView = computed(() => (
	data.value?.locked ? resolveLockedView(data.value, { canShowPurchase: canShowPaidMembershipEntry() }) : null
))
const summaryView = computed(() => resolveSummaryView(data.value?.summary, data.value?.sample_count))
const notableViews = computed(() => [
	resolveNotableMatchView(data.value?.best_win, 'best_win'),
	resolveNotableMatchView(data.value?.worst_loss, 'worst_loss')
].filter(Boolean))
const chart = computed(() => buildMonthChart(data.value?.months))

const currentIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const currentVersion = () => resolvePerformanceVersion({
	rankVersion: userDataInvalidationStore.versionOf('rank'),
	memberVersion: userDataInvalidationStore.versionOf('member')
})

const currentKey = () => {
	const identity = currentIdentity()
	return `${identity.userId}:${identity.authGeneration}:${currentGameType.value}:${currentVersion()}`
}

const goLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}

const loadPerformance = async () => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}
	const identity = currentIdentity()
	const gameType = currentGameType.value
	const key = currentKey()
	// 页面上的旧数据可能属于别的球种：只沿用同身份同球种的缓存撑住页面，没有缓存才显示加载态。
	const cached = performanceStore.peek(identity, gameType)
	const nextState = beginAsyncPageLoad({ ...pageState.value, data: cached, hasData: Boolean(cached) }, {
		authGeneration: identity.authGeneration,
		emptyData: null
	})
	const request = getAsyncPageRequest(nextState)
	pageState.value = nextState
	try {
		const response = await performanceStore.load(identity, gameType, currentVersion())
		// null 表示读取期间身份已变化，旧身份的结果不能回写。
		if (!response) return
		const applied = resolveAsyncPageLoad(pageState.value, request, { data: response, isEmpty: () => false })
		// 不是当前请求（用户已切球种或又发起了新读取）时状态保持不变。
		if (applied === pageState.value) return
		loadedKey.value = key
		pageState.value = applied
	} catch (error) {
		pageState.value = rejectAsyncPageLoad(pageState.value, request, error)
	}
}

const handleGameChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	loadPerformance()
}

const retry = () => {
	loadPerformance()
}

const openMatch = (matchId) => {
	if (!matchId) return
	uni.navigateTo({ url: `/subPages/match/matchDetail?match_id=${matchId}` })
}

const openMemberCenter = () => {
	uni.navigateTo({ url: '/subPages/user/memberCenter' })
}

onLoad((options) => {
	const gameType = Number(options?.game_type)
	if (GAME_TYPE_TABS.some((tab) => tab.value === gameType)) {
		currentGameType.value = gameType
	}
})

onShow(() => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}
	// 同身份、同球种、同失效版本时复用已加载的结果，不重复读取。
	if (loadedKey.value === currentKey() && pageState.value.status !== ASYNC_PAGE_STATUS.ERROR) return
	loadPerformance()
})
</script>

<style lang="scss" scoped>
@import './performance.scss';
</style>
