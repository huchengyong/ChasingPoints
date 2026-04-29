<template>
	<view class="h2h-record-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="h2h-summary-card" v-if="showSummaryCard">
			<view class="avatar-group">
				<view class="avatar-section">
					<image
						v-if="subjectAvatar"
						class="avatar"
						:src="subjectAvatar"
						mode="aspectFill"
					/>
					<view v-else class="avatar-placeholder">
						<text class="avatar-text">{{ getAvatarText(routeViewModel.subjectName) }}</text>
					</view>
					<text class="name">{{ routeViewModel.subjectName }}</text>
				</view>
				<view class="vs-section">
					<text class="vs-text">VS</text>
				</view>
				<view class="avatar-section">
					<image
						v-if="opponentData.avatar"
						class="avatar"
						:src="opponentData.avatar"
						mode="aspectFill"
					/>
					<view v-else class="avatar-placeholder">
						<text class="avatar-text">{{ getAvatarText(opponentData.name) }}</text>
					</view>
					<view class="name-row">
						<text class="name">{{ opponentData.name }}</text>
						<text v-if="routeViewModel.isOpponentMe" class="me-badge">我</text>
					</view>
				</view>
			</view>

			<view class="hero-copy">
				<text class="hero-title">{{ heroViewModel.title }}</text>
				<text class="hero-subtitle">{{ heroViewModel.subtitle }}</text>
			</view>

			<view class="total-score">
				<text class="score-text">{{ heroViewModel.scoreText }}</text>
			</view>

			<view class="badge-list">
				<view class="badge-chip" v-for="badge in heroViewModel.badges" :key="badge">
					<text class="badge-text">{{ badge }}</text>
				</view>
			</view>

			<view class="stats-grid">
				<view class="stat-item">
					<text class="stat-label">总场次</text>
					<text class="stat-value">{{ statsData.totalMatches }}</text>
				</view>
				<view class="stat-item">
					<text class="stat-label">平均分差</text>
					<text class="stat-value">{{ statsData.avgScoreDiff > 0 ? '+' : '' }}{{ statsData.avgScoreDiff.toFixed(1) }}</text>
				</view>
				<view class="stat-item">
					<text class="stat-label">最长连胜</text>
					<text class="stat-value">{{ statsData.maxWinStreak }}</text>
				</view>
			</view>
		</view>

		<view class="filter-bar" v-if="pageStatus !== 'loading' && pageStatus !== 'error'">
			<view class="filter-buttons">
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 0, pending: isHistoryRefreshing && currentFilter === 0 }"
					@click="handleFilterChange(0)"
				>
					<text class="btn-text">全部</text>
					<uni-icons class="filter-loading-icon" v-if="isHistoryRefreshing && currentFilter === 0" type="spinner-cycle" size="13" :color="isDarkMode ? '#ffffff' : '#6b7280'"></uni-icons>
				</button>
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 1, pending: isHistoryRefreshing && currentFilter === 1 }"
					@click="handleFilterChange(1)"
				>
					<text class="btn-text">胜利</text>
					<uni-icons class="filter-loading-icon" v-if="isHistoryRefreshing && currentFilter === 1" type="spinner-cycle" size="13" :color="isDarkMode ? '#ffffff' : '#6b7280'"></uni-icons>
				</button>
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 2, pending: isHistoryRefreshing && currentFilter === 2 }"
					@click="handleFilterChange(2)"
				>
					<text class="btn-text">失败</text>
					<uni-icons class="filter-loading-icon" v-if="isHistoryRefreshing && currentFilter === 2" type="spinner-cycle" size="13" :color="isDarkMode ? '#ffffff' : '#6b7280'"></uni-icons>
				</button>
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 3, pending: isHistoryRefreshing && currentFilter === 3 }"
					@click="handleFilterChange(3)"
				>
					<text class="btn-text">最近5场</text>
					<uni-icons class="filter-loading-icon" v-if="isHistoryRefreshing && currentFilter === 3" type="spinner-cycle" size="13" :color="isDarkMode ? '#ffffff' : '#6b7280'"></uni-icons>
				</button>
			</view>
		</view>

		<view class="trend-section" v-if="pageStatus !== 'loading' && pageStatus !== 'error' && trendItems.length > 0">
			<view class="history-header">
				<text class="header-text">最近走势</text>
			</view>
			<scroll-view class="trend-scroll" scroll-x>
				<view class="trend-row">
					<view
						v-for="item in trendItems"
						:key="item.matchId"
						class="trend-pill"
						:class="getResultClass(item.result)"
						@click="goToMatchDetail(item.matchId)"
					>
						<text class="trend-pill-text">{{ item.label }}</text>
					</view>
				</view>
			</scroll-view>
		</view>

		<view class="history-header" v-if="pageStatus !== 'loading' && pageStatus !== 'error' && matchCards.length > 0">
			<text class="header-text">比赛历史</text>
		</view>

		<scroll-view
			class="history-list"
			scroll-y
			@scrolltolower="onLoadMore"
		>
			<view class="loading-wrapper" v-if="pageStatus === 'loading'">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<view class="error-wrapper" v-else-if="pageStatus === 'error'">
				<uni-icons type="info-filled" size="52" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="error-text">{{ loadErrorMessage || '交锋数据加载失败' }}</text>
				<button class="retry-btn" @click="fetchData">
					<text>重新加载</text>
				</button>
			</view>

			<view v-else>
				<view class="error-banner" v-if="pageStatus === 'partial'">
					<text class="error-banner-text">{{ loadErrorMessage || '部分数据加载失败，先展示已获取内容' }}</text>
					<text class="error-banner-link" @click="fetchData">重新加载</text>
				</view>

				<view class="empty-wrapper" v-if="matchCards.length === 0">
					<uni-icons type="list" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					<text class="empty-text">暂无对局记录</text>
				</view>

				<view v-else>
					<view
						class="history-item"
						:class="getResultClass(item.result)"
						v-for="item in matchCards"
						:key="item.id"
						@click="goToMatchDetail(item.id)"
					>
						<view class="item-left">
							<text class="match-date">{{ formatMatchDate(item.dateText) }}</text>
							<text class="game-type">{{ item.gameTypeText }}</text>
							<view class="match-tags" v-if="item.tags.length > 0">
								<text class="match-tag" v-for="tag in item.tags" :key="tag">{{ tag }}</text>
							</view>
						</view>
						<view class="item-right">
							<text class="score">{{ item.scoreText }}</text>
							<view class="result-wrapper">
								<text class="result-text" :class="getResultClass(item.result)">{{ item.resultText }}</text>
								<text class="diff-text">{{ item.diffText }}</text>
							</view>
							<uni-icons type="right" size="20" :color="isDarkMode ? '#64748b' : '#9ca3af'"></uni-icons>
						</view>
					</view>

					<view class="load-more" v-if="currentFilter !== 3 && matchCards.length > 0">
						<text v-if="isLoadingMore" class="load-more-text">加载中...</text>
						<text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
					</view>
				</view>
			</view>
		</scroll-view>

		<view class="action-bar" v-if="showSummaryCard">
			<button class="primary-action-btn" @click="openPrimaryAction">
				<text>{{ heroViewModel.primaryActionText }}</text>
			</button>
		</view>
	</view>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { getH2HStats, getH2HHistory } from '@/api/match.js'
import { formatRelativeTime } from '@/utils/format.js'
import {
	buildH2HHeroViewModel,
	buildH2HMatchCards,
	buildH2HTrendItems,
	resolveH2HPageStatus,
	shouldShowH2HSummaryCard
} from '@/utils/h2h-view-model.js'
import {
	buildH2HHistoryParams,
	buildH2HLoadFailureAction,
	buildH2HViewModel,
	normalizeH2HRecordOptions,
	resolveH2HHistoryLoadingMode,
	shouldApplyH2HHistoryResponse
} from '@/utils/h2h-record.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

const isHistoryFetching = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const hasHandledTargetLoadFailure = ref(false)
const statsLoaded = ref(false)
const historyLoaded = ref(false)
const loadErrorMessage = ref('')
const latestHistoryRequestId = ref(0)

const opponentId = ref(0)
const opponentName = ref('')
const myAvatar = ref('')
const targetUserId = ref(0)
const targetName = ref('')
const targetAvatar = ref('')
const currentFilter = ref(0)

const opponentData = reactive({
	id: 0,
	name: '',
	avatar: ''
})

const statsData = reactive({
	totalMatches: 0,
	myWins: 0,
	opponentWins: 0,
	winRate: 0,
	avgScoreDiff: 0,
	maxWinStreak: 0
})

const historyList = ref([])
const summaryHistory = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

const routeViewModel = computed(() => buildH2HViewModel({
	targetUserId: targetUserId.value,
	targetName: targetName.value,
	opponentId: opponentId.value || opponentData.id,
	currentUserId: userStore.userId,
	opponentName: opponentData.name || opponentName.value
}))

const subjectAvatar = computed(() => (
	targetUserId.value > 0 ? targetAvatar.value : myAvatar.value
))

const heroViewModel = computed(() => buildH2HHeroViewModel({
	stats: {
		total_matches: statsData.totalMatches,
		my_wins: statsData.myWins,
		opponent_wins: statsData.opponentWins,
		avg_score_diff: statsData.avgScoreDiff,
		max_win_streak: statsData.maxWinStreak
	},
	opponent: {
		name: opponentData.name || opponentName.value,
		avatar: opponentData.avatar || ''
	},
	history: summaryHistory.value,
	subjectName: routeViewModel.value.subjectName,
	subjectAvatar: subjectAvatar.value
}))

const trendItems = computed(() => buildH2HTrendItems(summaryHistory.value.slice(0, 10)))
const displayHistoryList = computed(() => currentFilter.value === 3 ? historyList.value.slice(0, 5) : historyList.value)
const matchCards = computed(() => buildH2HMatchCards(displayHistoryList.value))
const pageStatus = computed(() => resolveH2HPageStatus({
	statsLoaded: statsLoaded.value,
	historyLoaded: historyLoaded.value,
	totalMatches: statsData.totalMatches,
	historyLength: summaryHistory.value.length,
	hasError: Boolean(loadErrorMessage.value)
}))
const showSummaryCard = computed(() => shouldShowH2HSummaryCard({
	pageStatus: pageStatus.value,
	statsLoaded: statsLoaded.value
}))
const historyLoadingMode = computed(() => resolveH2HHistoryLoadingMode({
	hasLoadedOnce: historyLoaded.value,
	isFetching: isHistoryFetching.value
}))
const isHistoryRefreshing = computed(() => historyLoadingMode.value === 'refreshing')

onLoad((options) => {
	const normalized = normalizeH2HRecordOptions(options)
	targetUserId.value = normalized.targetUserId
	targetName.value = normalized.targetName
	targetAvatar.value = normalized.targetAvatar
	opponentId.value = normalized.opponentId
	opponentName.value = normalized.opponentName

	uni.setNavigationBarTitle({
		title: buildH2HViewModel({
			targetUserId: normalized.targetUserId,
			targetName: normalized.targetName,
			opponentName: normalized.opponentName
		}).navigationTitle
	})
})

onMounted(() => {
	loadUserInfo()
	fetchData()
})

onShow(() => {
})

const loadUserInfo = () => {
	if (userStore.userInfo && userStore.userInfo.avatar) {
		myAvatar.value = userStore.userInfo.avatar
	}
}

const handleTargetLoadFailure = (error) => {
	if (hasHandledTargetLoadFailure.value) {
		return true
	}

	const action = buildH2HLoadFailureAction({
		targetUserId: targetUserId.value,
		error
	})

	if (!action) {
		return false
	}

	hasHandledTargetLoadFailure.value = true
	uni.showToast({
		title: action.toastMessage,
		icon: 'none'
	})

	if (action.shouldNavigateBack && getCurrentPages().length > 1) {
		setTimeout(() => {
			uni.navigateBack()
		}, 1200)
	}

	return true
}

const fetchData = async () => {
	statsLoaded.value = false
	historyLoaded.value = false
	loadErrorMessage.value = ''

	try {
		await Promise.all([
			fetchStats(),
			fetchHistory(false, false)
		])
	} catch (error) {
		console.error('获取数据失败:', error)
		handleTargetLoadFailure(error)
	}
}

const fetchStats = async () => {
	try {
		const params = {}
		if (targetUserId.value > 0) {
			params.target_user_id = targetUserId.value
		}
		if (opponentId.value > 0) {
			params.opponent_id = opponentId.value
		} else if (opponentName.value) {
			params.opponent_name = opponentName.value
		}

		const res = await getH2HStats(params)

		if (res.opponent) {
			opponentData.id = res.opponent.id || 0
			opponentData.name = res.opponent.name || opponentName.value
			opponentData.avatar = res.opponent.avatar || ''
		} else {
			opponentData.name = opponentName.value
		}

		if (res.stats) {
			statsData.totalMatches = res.stats.total_matches || 0
			statsData.myWins = res.stats.my_wins || 0
			statsData.opponentWins = res.stats.opponent_wins || 0
			statsData.winRate = res.stats.win_rate || 0
			statsData.avgScoreDiff = res.stats.avg_score_diff || 0
			statsData.maxWinStreak = res.stats.max_win_streak || 0
		}

		statsLoaded.value = true
	} catch (error) {
		console.error('获取交锋统计失败:', error)
		opponentData.name = opponentName.value
		loadErrorMessage.value = error?.message || '交锋统计加载失败'
		throw error
	}
}

const fetchHistory = async (isRefresh = false, isLoadMore = false) => {
	if (isLoadingMore.value && !isRefresh) return
	const requestId = latestHistoryRequestId.value + 1
	latestHistoryRequestId.value = requestId

	if (isRefresh) {
		currentPage.value = 1
		hasMore.value = true
		isLoadingMore.value = false
	} else if (isLoadMore) {
		if (!hasMore.value || currentFilter.value === 3) return
		isLoadingMore.value = true
		currentPage.value++
	}

	if (!isLoadMore) {
		isHistoryFetching.value = true
		if (!historyLoaded.value) {
			historyLoaded.value = false
		}
		if (statsLoaded.value) {
			loadErrorMessage.value = ''
		}
	}

	try {
		const params = buildH2HHistoryParams({
			targetUserId: targetUserId.value,
			opponentId: opponentId.value,
			fallbackOpponentId: opponentData.id,
			opponentName: opponentData.name || opponentName.value,
			page: currentPage.value,
			pageSize,
			result: currentFilter.value === 3 ? 0 : currentFilter.value
		})

		const res = await getH2HHistory(params)
		if (!shouldApplyH2HHistoryResponse({ requestId, latestRequestId: latestHistoryRequestId.value })) {
			return
		}

		const list = res.list || []
		total.value = res.total || 0

		if (isRefresh || !isLoadMore) {
			historyList.value = list
		} else {
			historyList.value = [...historyList.value, ...list]
		}

		if (currentFilter.value === 0 || currentFilter.value === 3) {
			summaryHistory.value = historyList.value
		}

		hasMore.value = historyList.value.length < total.value
		historyLoaded.value = true
	} catch (error) {
		if (!shouldApplyH2HHistoryResponse({ requestId, latestRequestId: latestHistoryRequestId.value })) {
			return
		}

		console.error('获取交锋历史失败:', error)
		loadErrorMessage.value = error?.message || '交锋历史加载失败'
		handleTargetLoadFailure(error)
		if (!isLoadMore) {
			if (!historyLoaded.value) {
				historyList.value = []
			}
			if (currentFilter.value === 0 || currentFilter.value === 3) {
				if (!historyLoaded.value) {
					summaryHistory.value = []
				}
			}
		}
	} finally {
		if (shouldApplyH2HHistoryResponse({ requestId, latestRequestId: latestHistoryRequestId.value })) {
			isLoadingMore.value = false
			if (!isLoadMore) {
				isHistoryFetching.value = false
			}
		}
	}
}

const handleFilterChange = (filter) => {
	if (currentFilter.value === filter) return
	currentFilter.value = filter
	currentPage.value = 1
	hasMore.value = true
	fetchHistory(true, false)
}

const onLoadMore = () => {
	if (currentFilter.value === 3) return
	fetchHistory(false, true)
}

const goToMatchDetail = (matchId) => {
	if (!matchId) return
	uni.navigateTo({
		url: `/subPages/match/matchDetail?match_id=${matchId}`
	})
}

const openPrimaryAction = () => {
	uni.navigateTo({
		url: '/subPages/social/challenges'
	})
}

const getAvatarText = (name) => {
	if (!name) return '?'
	return name.substring(0, 1).toUpperCase()
}

const formatMatchDate = (dateStr) => {
	if (!dateStr) return ''
	const date = new Date(dateStr)
	return `${date.getMonth() + 1}月${date.getDate()}日 · ${formatRelativeTime(dateStr)}`
}

const getResultClass = (result) => {
	switch (result) {
		case 1:
			return 'win'
		case 2:
			return 'lose'
		case 3:
			return 'draw'
		default:
			return ''
	}
}
</script>

<style lang="scss" scoped>
@import './h2hRecord.scss';
</style>
