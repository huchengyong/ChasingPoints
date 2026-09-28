<template>
	<view class="stats-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载中 -->
		<view v-if="isInitialLoading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载统计数据...</text>
		</view>

		<view v-else-if="statsPageError" class="loading-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="loading-text">{{ statsPageError.title }}</text>
			<text class="error-desc">{{ statsPageError.description }}</text>
			<button class="error-action" @tap="handleStatsErrorAction">{{ statsPageError.actionText }}</button>
		</view>

		<template v-else>
			<view v-if="statsRefreshError" class="refresh-error-banner">
				<text>{{ statsRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryStats">重试</text>
			</view>
			<!-- 分球种统计 -->
			<view class="section">
				<text class="section-title">分球种统计</text>
				<view class="game-tabs">
					<view
						v-for="tab in gameTabs"
						:key="tab.key"
						class="tab-item"
						:class="{ active: currentGame === tab.key, pending: isStatsRefreshing && currentGame === tab.key }"
						@tap="handleGameChange(tab.key)"
					>
						<text>{{ tab.label }}</text>
						<uni-icons
							class="tab-loading-icon"
							v-if="isStatsRefreshing && currentGame === tab.key"
							type="spinner-cycle"
							size="12"
							color="#ffffff"
						></uni-icons>
					</view>
				</view>
				<view v-if="currentGameStats" class="stats-grid">
					<view class="stat-item">
						<text class="stat-value highlight">{{ formatPercent(currentGameStats.win_rate) }}%</text>
						<text class="stat-label">胜率</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ currentGameStats.total_matches || 0 }}</text>
						<text class="stat-label">总场次</text>
					</view>
					<view class="stat-item">
						<text class="stat-value green">{{ currentGameStats.wins || 0 }}</text>
						<text class="stat-label">胜场</text>
					</view>
					<view class="stat-item">
						<text class="stat-value red">{{ currentGameStats.losses || 0 }}</text>
						<text class="stat-label">负场</text>
					</view>
				</view>
				<view class="win-rate-bar">
					<view class="bar-fill" :style="{ width: formatPercent(currentGameStats?.win_rate) + '%' }"></view>
				</view>
			</view>

			<!-- 近期趋势 -->
			<view class="section">
				<view class="section-header">
					<text class="section-title">近期趋势</text>
					<view class="month-switch">
						<view class="month-btn" @tap="handleStatsMonthChange(-1)">‹</view>
						<text class="month-label">{{ trendCalendar.title }}</text>
						<view class="month-btn" @tap="handleStatsMonthChange(1)">›</view>
					</view>
				</view>
				<view class="trend-summary">
					<text class="trend-rate">本月{{ trendCalendar.summary.matchCount }}场，胜率{{ trendCalendar.summary.winRate }}%</text>
				</view>
				<view class="stats-calendar">
					<view class="calendar-weekdays">
						<text v-for="weekday in trendCalendar.weekdays" :key="weekday">{{ weekday }}</text>
					</view>
					<view class="calendar-grid">
						<view
							v-for="cell in trendCalendar.cells"
							:key="cell.dateKey"
							class="calendar-cell"
							:class="{ muted: !cell.isCurrentMonth, active: cell.hasMatches }"
						>
							<text class="calendar-day">{{ cell.day }}</text>
							<view v-if="cell.isCurrentMonth && cell.hasMatches" class="calendar-results">
								<view v-if="cell.winCount > 0" class="calendar-result-row">
									<view class="calendar-dot win"></view>
									<text>×{{ cell.winCount }}</text>
								</view>
								<view v-if="cell.lossCount > 0" class="calendar-result-row">
									<view class="calendar-dot loss"></view>
									<text>×{{ cell.lossCount }}</text>
								</view>
							</view>
						</view>
					</view>
				</view>
				<view class="trend-legend">
					<view class="legend-item"><view class="dot-sample win"></view><text>胜</text></view>
					<view class="legend-item"><view class="dot-sample loss"></view><text>负</text></view>
				</view>
			</view>

			<!-- 段位分变化 -->
			<view class="section">
				<view class="section-header">
					<text class="section-title">段位分变化</text>
					<view class="month-switch">
						<view class="month-btn" @tap="handleStatsMonthChange(-1)">‹</view>
						<text class="month-label">{{ rankDeltaChart.title }}</text>
						<view class="month-btn" @tap="handleStatsMonthChange(1)">›</view>
					</view>
				</view>
				<view class="rank-summary">
					<view class="rank-current">
						<text class="rank-value">{{ currentRankScore }}</text>
						<text class="rank-label">当前分</text>
					</view>
					<view class="rank-meta">
						<view class="meta-item">
							<text class="meta-value">{{ peakRankScore }}</text>
							<text class="meta-label">最高分</text>
						</view>
						<view class="meta-item">
							<text class="meta-value" :class="{ up: rankDeltaChart.summary.netDelta > 0, down: rankDeltaChart.summary.netDelta < 0 }">
								{{ rankDeltaChart.summary.netDelta > 0 ? '+' : '' }}{{ rankDeltaChart.summary.netDelta }}
							</text>
							<text class="meta-label">本月净变</text>
						</view>
					</view>
				</view>
				<view v-if="rankDeltaChart.summary.upDays + rankDeltaChart.summary.downDays > 0" class="rank-chart">
					<view class="rank-chart-body">
						<view class="rank-zero-line"></view>
						<view
							v-for="day in rankDeltaChart.days"
							:key="day.dateKey"
							class="rank-chart-day"
						>
							<view class="bar-half upper">
								<view
									v-if="day.isUp"
									class="rank-bar up"
									:style="{ height: day.heightPercent + '%' }"
								></view>
							</view>
							<view class="bar-half lower">
								<view
									v-if="day.isDown"
									class="rank-bar down"
									:style="{ height: day.heightPercent + '%' }"
								></view>
							</view>
						</view>
					</view>
					<view class="rank-chart-axis">
						<text
							v-for="day in rankDeltaChart.days"
							:key="day.dateKey"
							:class="{ visible: day.showTick }"
						>{{ day.showTick ? day.label : '' }}</text>
					</view>
				</view>
				<view v-else class="empty-hint">
					<text>暂无段位变化记录</text>
				</view>
			</view>

			<!-- 单杆最高分 -->
			<view class="section">
				<text class="section-title">单杆最高分</text>
				<view class="high-score-card">
					<view class="score-main">
						<text class="score-number">{{ highScore.personal_best || 0 }}</text>
						<text class="score-label">个人最佳</text>
					</view>
					<view class="score-sub">
						<text class="score-month">本月最佳：{{ highScore.month_best || 0 }}</text>
					</view>
				</view>
			</view>

			<!-- 对局时长 -->
			<view class="section">
				<text class="section-title">对局时长</text>
				<view class="duration-grid">
					<view class="duration-item">
						<text class="duration-value">{{ formatDuration(durationData.average) }}</text>
						<text class="duration-label">平均</text>
					</view>
					<view class="duration-item">
						<text class="duration-value green">{{ formatDuration(durationData.fastest) }}</text>
						<text class="duration-label">最快</text>
					</view>
					<view class="duration-item">
						<text class="duration-value amber">{{ formatDuration(durationData.longest) }}</text>
						<text class="duration-label">最长</text>
					</view>
				</view>
			</view>

			<!-- 强弱对手分析 -->
			<view class="section">
				<text class="section-title">强弱对手分析</text>
				<view class="opponent-list">
					<view
						v-for="(tier, index) in opponentData"
						:key="index"
						class="opponent-item"
					>
						<text class="tier-name">{{ tier.tier_name || ('段位' + (index + 1)) }}</text>
						<view class="tier-bar">
							<view
								class="tier-fill"
								:style="{ width: formatPercent(tier.win_rate) + '%' }"
								:class="{ good: tier.win_rate >= 50, bad: tier.win_rate < 50 }"
							></view>
						</view>
						<text class="tier-rate" :class="{ good: tier.win_rate >= 50, bad: tier.win_rate < 50 }">{{ formatPercent(tier.win_rate) }}%</text>
					</view>
				</view>
				<view v-if="opponentData.length === 0" class="empty-hint">
					<text>暂无对手分析数据</text>
				</view>
			</view>
		</template>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getStatsOverview } from '@/api/stats.js'
import { GAME_TYPE_KEY_MAP, GAME_TYPE_STATS_TABS, GAME_TYPE_VALUE_MAP } from '@/utils/game-types.js'
import { formatMonthKey } from '@/utils/format.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { useUserStore } from '@/store/user.js'
import {
	buildRankDeltaChartViewModel,
	buildStatsTrendCalendarViewModel,
	normalizeDurationStats,
	normalizeOpponentStrengthStats,
	resolveStatsDetailLoadingMode,
	shiftMonthKey,
	shouldApplyStatsDetailResponse
} from '@/utils/stats-detail.js'
import {
	ASYNC_PAGE_STATUS,
	beginAsyncPageLoad,
	createAsyncPageState,
	getAsyncPageRequest,
	rejectAsyncPageLoad,
	resolveAsyncPageErrorFeedback,
	resolveAsyncPageLoad
} from '@/utils/async-page-state.js'
import { createRequestError } from '@/utils/request-errors.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const userDataInvalidationStore = useUserDataInvalidationStore()

const isStatsFetching = ref(true)
const hasLoadedOnce = ref(false)
const statsState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: null
}))
const latestStatsRequestId = ref(0)
const loadedIdentityKey = ref('')
const loadedStatsScopeVersion = ref(0)
const currentGame = ref('chinese_eight')
const selectedStatsMonthKey = ref(formatMonthKey(new Date()))

const gameTypeStats = ref({})
const trendData = ref([])
const rankData = ref([])
const highScore = ref({})
const durationData = ref({})
const opponentData = ref([])

const gameTabs = GAME_TYPE_STATS_TABS
const gameTypeKeyMap = GAME_TYPE_KEY_MAP
const gameTypeValueMap = GAME_TYPE_VALUE_MAP
const loadingMode = computed(() => resolveStatsDetailLoadingMode({
	hasLoadedOnce: hasLoadedOnce.value,
	isFetching: isStatsFetching.value
}))
const isInitialLoading = computed(() => loadingMode.value === 'initial')
const isStatsRefreshing = computed(() => loadingMode.value === 'refreshing')
const statsPageError = computed(() => (
	statsState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(statsState.value.error, { resource: '统计数据' })
		: null
))
const statsRefreshError = computed(() => (
	statsState.value.refreshError
		? resolveAsyncPageErrorFeedback(statsState.value.refreshError, { resource: '统计数据' })
		: null
))

const currentReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const currentIdentityKey = () => {
	const identity = currentReadIdentity()
	return `${identity.userId}:${identity.authGeneration}`
}

const currentGameStats = computed(() => {
	return gameTypeStats.value[currentGame.value] || {}
})

const trendCalendar = computed(() => buildStatsTrendCalendarViewModel(trendData.value, {
	monthKey: selectedStatsMonthKey.value
}))

const rankDeltaChart = computed(() => buildRankDeltaChartViewModel(rankData.value, {
	monthKey: selectedStatsMonthKey.value
}))

const rankTrendList = computed(() => {
	const list = Array.isArray(rankData.value?.list) ? rankData.value.list : Array.isArray(rankData.value) ? rankData.value : []
	return list.map((item, index) => {
		const prev = list[index + 1]
		const prevScore = prev?.rank_score ?? item.rank_score ?? 0
		return {
			date: item.date || '',
			score: item.rank_score || 0,
			delta: (item.rank_score || 0) - prevScore
		}
	})
})

const currentRankScore = computed(() => rankTrendList.value[0]?.score || 0)
const peakRankScore = computed(() => rankTrendList.value.length ? Math.max(...rankTrendList.value.map(item => item.score || 0)) : 0)

const formatDuration = (durationSeconds) => {
	if (!durationSeconds || durationSeconds < 0) return '0分'

	const totalMinutes = Math.floor(durationSeconds / 60)

	if (totalMinutes < 1) {
		return `${durationSeconds}秒`
	}

	if (totalMinutes < 60) {
		return `${totalMinutes}分`
	}

	const hours = Math.floor(totalMinutes / 60)
	const remainMinutes = totalMinutes % 60

	if (hours < 24) {
		if (remainMinutes > 0) {
			return `${hours}小时${remainMinutes}分`
		}
		return `${hours}小时`
	}

	const days = Math.floor(hours / 24)
	const remainHours = hours % 24

	if (remainHours > 0) {
		return `${days}天${remainHours}小时`
	}
	return `${days}天`
}

const formatPercent = (value) => {
	const percent = Number(value || 0)
	if (!Number.isFinite(percent)) return 0
	return Number.isInteger(percent) ? percent : Number(percent.toFixed(2))
}

const normalizeGameTypeStats = (payload) => {
	const source = Array.isArray(payload?.list) ? payload.list : Array.isArray(payload) ? payload : []
	const normalized = {}

	source.forEach(item => {
		const key = gameTypeKeyMap[item?.game_type]
		if (!key) return
		normalized[key] = item
	})

	return normalized
}

const normalizeHighScoreStats = (payload) => {
	const list = Array.isArray(payload?.list) ? payload.list : []
	const personalBest = list.length ? (list[0]?.score || 0) : 0
	const currentMonth = formatMonthKey(new Date())
	const monthBest = list.reduce((best, item) => {
		if (!item?.date || !String(item.date).startsWith(currentMonth)) return best
		return Math.max(best, item.score || 0)
	}, 0)
	return {
		personal_best: personalBest,
		month_best: monthBest
	}
}

const loadAllStats = async () => {
	const requestId = latestStatsRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	const requestScopeVersion = userDataInvalidationStore.versionOf('stats')
	const nextState = beginAsyncPageLoad(statsState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: null
	})
	const pageRequest = getAsyncPageRequest(nextState)
	latestStatsRequestId.value = requestId
	if (loadedIdentityKey.value && loadedIdentityKey.value !== requestIdentityKey) {
		gameTypeStats.value = {}
		trendData.value = []
		rankData.value = []
		highScore.value = {}
		durationData.value = {}
		opponentData.value = []
		hasLoadedOnce.value = false
	}
	statsState.value = nextState
	isStatsFetching.value = true
	const gameType = gameTypeValueMap[currentGame.value] || 3
	try {
		const data = await getStatsOverview({ game_type: gameType, trend_limit: 100 })
		if (!shouldApplyStatsDetailResponse({ requestId, latestRequestId: latestStatsRequestId.value }) || currentIdentityKey() !== requestIdentityKey || statsState.value.requestId !== pageRequest.requestId || statsState.value.authGeneration !== pageRequest.authGeneration) {
			return
		}
		if (!data?.success) throw createRequestError({ message: data?.message || '加载统计数据失败', category: 'business' })

		const availability = data.availability || {}
		if (availability.by_game_type !== false) {
			gameTypeStats.value = normalizeGameTypeStats(data.by_game_type)
		}
		if (availability.recent_trend !== false) {
			trendData.value = data.recent_trend || []
		}
		if (availability.rank_score_trend !== false) {
			rankData.value = data.rank_score_trend || []
		}
		if (availability.single_high_scores !== false) {
			highScore.value = normalizeHighScoreStats({ list: data.single_high_scores })
		}
		if (availability.duration !== false) {
			durationData.value = normalizeDurationStats(data.duration)
		}
		if (availability.opponent_strength !== false) {
			opponentData.value = normalizeOpponentStrengthStats(data.opponent_strength)
		}
		hasLoadedOnce.value = true
		loadedIdentityKey.value = requestIdentityKey
		loadedStatsScopeVersion.value = requestScopeVersion
		statsState.value = resolveAsyncPageLoad(statsState.value, pageRequest, {
			data,
			isEmpty: () => false
		})
	} catch (e) {
		if (shouldApplyStatsDetailResponse({ requestId, latestRequestId: latestStatsRequestId.value }) && currentIdentityKey() === requestIdentityKey) {
			statsState.value = rejectAsyncPageLoad(statsState.value, pageRequest, e)
		}
	} finally {
		if (shouldApplyStatsDetailResponse({ requestId, latestRequestId: latestStatsRequestId.value }) && currentIdentityKey() === requestIdentityKey) {
			isStatsFetching.value = false
		}
	}
}

const retryStats = () => loadAllStats()

const handleStatsErrorAction = () => {
	if (statsState.value.error?.category === 'permission' || statsState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryStats()
}

onLoad((options) => {
	const gameType = Number(options?.game_type || 0)
	if (gameTypeKeyMap[gameType]) {
		currentGame.value = gameTypeKeyMap[gameType]
	}
})

onShow(() => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}

	if (!hasLoadedOnce.value || loadedIdentityKey.value !== currentIdentityKey() || loadedStatsScopeVersion.value !== userDataInvalidationStore.versionOf('stats')) {
		loadAllStats()
	}
})

const handleGameChange = (gameKey) => {
	if (currentGame.value === gameKey) return
	currentGame.value = gameKey
	loadAllStats()
}

const handleStatsMonthChange = (offset) => {
	selectedStatsMonthKey.value = shiftMonthKey(selectedStatsMonthKey.value, offset)
}

const goLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}
</script>

<style lang="scss" scoped>
.stats-page {
	min-height: 100vh;
	background: #FAF8F2;
	padding-bottom: 60rpx;
	padding-top: 24rpx;
	box-sizing: border-box;

	&.dark-mode {
		background: #141109;

		.section {
			background: #1e180d;
			border: 1rpx solid #3a2e16;
		}

		.section-title,
		.stat-value,
		.meta-value,
		.duration-value {
			color: #fff7e1;
		}

		.loading-text,
		.stat-label,
		.trend-rate,
		.legend-item,
		.rank-label,
		.meta-label,
		.score-label,
		.score-month,
		.duration-label,
		.tier-name,
		.empty-hint {
			color: #d7c89b;
		}

		.game-tabs .tab-item {
			background: #241d10;
			color: #d7c89b;

			&.active {
				background: #E0AE12;
				color: #ffffff;
			}
		}

		.win-rate-bar {
			background: rgba(239, 68, 68, 0.18);
		}

		.stats-calendar .calendar-cell,
		.opponent-list .opponent-item .tier-bar {
			background: #3a2e16;
		}

		.stats-calendar .calendar-weekdays,
		.month-label,
		.rank-chart-axis text {
			color: #d7c89b;
		}

		.stats-calendar .calendar-cell .calendar-day {
			color: #fff7e1;
		}

		.rank-chart-body {
			background: #241d10;
		}

		.rank-zero-line {
			background: #3a2e16;
		}
	}
}

.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 300rpx;
	.loading-text {
		margin-top: 16rpx;
		font-size: 28rpx;
		color: #9A8C67;
	}
}

.error-desc {
	color: #6E6242;
	font-size: 26rpx;
	line-height: 1.6;
	text-align: center;
}

.error-action {
	min-width: 200rpx;
	height: 76rpx;
	line-height: 76rpx;
	margin: 12rpx 0 0;
	padding: 0 32rpx;
	border-radius: 38rpx;
	background: #E0AE12;
	color: #ffffff;
	font-size: 28rpx;
	font-weight: 600;
}

.error-action::after {
	display: none;
}

.refresh-error-banner {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 20rpx;
	margin: 0 24rpx 20rpx;
	padding: 18rpx 22rpx;
	border-radius: 14rpx;
	background: rgba(224, 174, 18, 0.12);
	color: #8a5b00;
	font-size: 24rpx;
}

.refresh-error-action {
	flex-shrink: 0;
	color: #a86f00;
	font-weight: 600;
}

.section {
	margin: 0 24rpx 24rpx;
	background: #fff;
	border-radius: 16rpx;
	padding: 28rpx;

	.section-title {
		font-size: 30rpx;
		font-weight: 700;
		color: #231C0B;
		margin-bottom: 20rpx;
		display: block;
	}
}

.section-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 16rpx;
	margin-bottom: 20rpx;

	.section-title {
		margin-bottom: 0;
	}
}

.month-switch {
	display: flex;
	align-items: center;
	gap: 10rpx;
	flex-shrink: 0;

	.month-btn {
		width: 44rpx;
		height: 44rpx;
		line-height: 40rpx;
		text-align: center;
		border-radius: 22rpx;
		background: #FAF8F2;
		color: #6E6242;
		font-size: 30rpx;
		font-weight: 700;
	}

	.month-label {
		min-width: 132rpx;
		text-align: center;
		font-size: 24rpx;
		color: #6E6242;
	}
}

.game-tabs {
	display: flex;
	gap: 12rpx;
	margin-bottom: 20rpx;

	.tab-item {
		flex: 1;
		min-width: 0;
		text-align: center;
		position: relative;
		padding: 12rpx 0;
		border-radius: 10rpx;
		background: #FAF8F2;
		font-size: 26rpx;
		color: #6E6242;
		display: flex;
		align-items: center;
		justify-content: center;

		text {
			white-space: nowrap;
		}

		&.active {
			background: #E0AE12;
			color: #ffffff;
		}

		&.pending {
			opacity: 0.78;
		}

		.tab-loading-icon {
			position: absolute;
			right: 6rpx;
			top: 50%;
			transform: translateY(-50%);
		}
	}
}

.stats-grid {
	display: flex;
	flex-wrap: wrap;

	.stat-item {
		width: 50%;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 16rpx 0;

		.stat-value {
			font-size: 40rpx;
			font-weight: 700;
			color: #231C0B;

			&.highlight { color: #C69200; }
			&.green { color: #22c55e; }
			&.red { color: #ef4444; }
		}
		.stat-label {
			font-size: 24rpx;
			color: #9A8C67;
			margin-top: 4rpx;
		}
	}
}

.win-rate-bar {
	height: 16rpx;
	background: #fee2e2;
	border-radius: 8rpx;
	overflow: hidden;
	margin-top: 12rpx;

	.bar-fill {
		height: 100%;
		background: #22c55e;
		border-radius: 8rpx;
	}
}

.trend-summary {
	margin-bottom: 16rpx;
	.trend-rate {
		font-size: 26rpx;
		color: #6E6242;
	}
}

.stats-calendar {
	.calendar-weekdays {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		margin-bottom: 8rpx;
		font-size: 22rpx;
		color: #9A8C67;
		text-align: center;
	}

	.calendar-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 8rpx;
	}

	.calendar-cell {
		min-height: 78rpx;
		border-radius: 10rpx;
		background: #FAF8F2;
		padding: 6rpx;
		box-sizing: border-box;
		opacity: 1;

		&.muted {
			opacity: 0.35;
		}

		&.active {
			background: #fff7e6;
		}

		.calendar-day {
			display: block;
			font-size: 20rpx;
			line-height: 22rpx;
			color: #6E6242;
		}

		.calendar-results {
			margin-top: 4rpx;
			display: flex;
			flex-direction: column;
			gap: 2rpx;
		}

		.calendar-result-row {
			display: flex;
			align-items: center;
			gap: 3rpx;
			font-size: 18rpx;
			line-height: 20rpx;
			color: #6E6242;
		}

		.calendar-dot {
			width: 10rpx;
			height: 10rpx;
			border-radius: 50%;
			flex-shrink: 0;

			&.win { background: #22c55e; }
			&.loss { background: #ef4444; }
		}
	}
}

.trend-legend {
	display: flex;
	gap: 24rpx;
	margin-top: 16rpx;

	.legend-item {
		display: flex;
		align-items: center;
		gap: 8rpx;
		font-size: 22rpx;
		color: #9A8C67;

		.dot-sample {
			width: 16rpx;
			height: 16rpx;
			border-radius: 4rpx;
			&.win { background: #22c55e; }
			&.loss { background: #ef4444; }
		}
	}
}

.rank-summary {
	display: flex;
	align-items: center;
	gap: 32rpx;
	margin-bottom: 20rpx;

	.rank-current {
		display: flex;
		flex-direction: column;
		align-items: center;

		.rank-value {
			font-size: 56rpx;
			font-weight: 800;
			color: #C69200;
		}
		.rank-label {
			font-size: 24rpx;
			color: #9A8C67;
		}
	}

	.rank-meta {
		flex: 1;
		display: flex;
		gap: 32rpx;

		.meta-item {
			display: flex;
			flex-direction: column;
			align-items: center;

			.meta-value {
				font-size: 32rpx;
				font-weight: 600;
				color: #231C0B;

				&.up { color: #22c55e; }
				&.down { color: #ef4444; }
			}
			.meta-label {
				font-size: 22rpx;
				color: #9A8C67;
			}
		}
	}
}

.rank-chart {
	.rank-chart-body {
		position: relative;
		height: 220rpx;
		display: flex;
		align-items: stretch;
		background: #FAF8F2;
		border-radius: 12rpx;
		padding: 12rpx 8rpx;
		box-sizing: border-box;
		overflow: hidden;
	}

	.rank-zero-line {
		position: absolute;
		left: 8rpx;
		right: 8rpx;
		top: 50%;
		height: 1rpx;
		background: #E9E2CF;
	}

	.rank-chart-day {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		z-index: 1;
	}

	.bar-half {
		width: 100%;
		height: 50%;
		display: flex;
		justify-content: center;
	}

	.bar-half.upper {
		align-items: flex-end;
	}

	.bar-half.lower {
		align-items: flex-start;
	}

	.rank-bar {
		width: 8rpx;
		min-height: 8rpx;
		border-radius: 4rpx;

		&.up { background: #22c55e; }
		&.down { background: #ef4444; }
	}

	.rank-chart-axis {
		display: flex;
		margin-top: 8rpx;

		text {
			flex: 1;
			min-width: 0;
			text-align: center;
			font-size: 18rpx;
			color: transparent;

			&.visible {
				color: #9A8C67;
			}
		}
	}
}

.high-score-card {
	display: flex;
	align-items: center;
	gap: 32rpx;

	.score-main {
		display: flex;
		flex-direction: column;
		align-items: center;

		.score-number {
			font-size: 64rpx;
			font-weight: 800;
			color: #f59e0b;
		}
		.score-label {
			font-size: 24rpx;
			color: #9A8C67;
		}
	}
	.score-sub {
		.score-month {
			font-size: 26rpx;
			color: #6E6242;
		}
	}
}

.duration-grid {
	display: flex;

	.duration-item {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;

		.duration-value {
			font-size: 36rpx;
			font-weight: 700;
			color: #231C0B;

			&.green { color: #22c55e; }
			&.amber { color: #f59e0b; }
		}
		.duration-label {
			font-size: 24rpx;
			color: #9A8C67;
			margin-top: 4rpx;
		}
	}
}

.opponent-list {
	.opponent-item {
		display: flex;
		align-items: center;
		gap: 16rpx;
		margin-bottom: 16rpx;

		.tier-name {
			width: 120rpx;
			font-size: 24rpx;
			color: #6E6242;
			flex-shrink: 0;
		}

		.tier-bar {
			flex: 1;
			height: 20rpx;
			background: #FAF8F2;
			border-radius: 10rpx;
			overflow: hidden;

			.tier-fill {
				height: 100%;
				border-radius: 10rpx;
				&.good { background: #22c55e; }
				&.bad { background: #ef4444; }
			}
		}

		.tier-rate {
			width: 80rpx;
			text-align: right;
			font-size: 24rpx;
			font-weight: 600;
			flex-shrink: 0;
			&.good { color: #22c55e; }
			&.bad { color: #ef4444; }
		}
	}
}

.empty-hint {
	text-align: center;
	padding: 32rpx 0;
	font-size: 26rpx;
	color: #9A8C67;
}
</style>
