<template>
	<view class="season-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
				<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="seasonPageError" class="page-error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="page-error-title">{{ seasonPageError.title }}</text>
			<text class="page-error-text">{{ seasonPageError.description }}</text>
			<button class="retry-btn" @tap="handleSeasonErrorAction">{{ seasonPageError.actionText }}</button>
		</view>

		<view v-else class="season-content">
			<view v-if="seasonRefreshError" class="refresh-error-banner">
				<text>{{ seasonRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retrySeason">重试</text>
			</view>
			<!-- 赛季信息卡片 -->
			<view class="season-card" v-if="hasActiveSeason">
				<view class="season-header">
					<text class="season-name">{{ season.name }}</text>
					<view class="season-status" :class="'status-' + season.status">
						<text>{{ statusMap[season.status] || '未知' }}</text>
					</view>
				</view>
				<view class="season-dates">
					<text>{{ season.start_date }} ~ {{ season.end_date }}</text>
				</view>
				<view class="season-progress" v-if="season.status === 1">
					<view class="progress-bar">
						<view class="progress-fill" :style="{ width: progressPercent + '%' }"></view>
					</view>
					<text class="progress-text">剩余 {{ remainDays }} 天</text>
				</view>
			</view>

			<view v-else class="no-season-card">
				<text class="no-season-icon">🏆</text>
				<text class="no-season-text">{{ seasonEmptyState.title }}</text>
				<text class="no-season-description">{{ seasonEmptyState.description }}</text>
			</view>

			<template v-if="hasActiveSeason">
				<view class="game-type-tabs">
					<view
						v-for="item in gameTypeTabs"
						:key="item.value"
						class="game-type-tab"
						:class="{ active: currentGameType === item.value }"
						@tap="handleGameTypeChange(item.value)"
					>
						<text>{{ item.label }}</text>
					</view>
				</view>

				<!-- 我的赛季数据 -->
				<view class="my-record-card" v-if="myRecord">
				<text class="card-title">我的赛季数据</text>
				<view class="record-stats">
					<view class="stat-item">
						<text class="stat-value">{{ myRecord.matches_played }}</text>
						<text class="stat-label">场次</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ myRecord.wins }}</text>
						<text class="stat-label">胜场</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ winRate }}%</text>
						<text class="stat-label">胜率</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ myRecord.peak_rank_score }}</text>
						<text class="stat-label">峰值分</text>
					</view>
				</view>
				<view class="record-rank" v-if="myRecord.final_rank > 0">
					<text>当前排名：第 {{ myRecord.final_rank }} 名</text>
				</view>
			</view>

				<!-- 赛季排行榜 -->
				<view class="leaderboard-card">
				<view class="card-header">
					<text class="card-title">赛季排行榜</text>
					<view class="report-btn" v-if="season && season.status === 2" @tap="goReport">
						<text>赛季报告</text>
					</view>
				</view>
				<view v-if="leaderboard.length > 0" class="leaderboard-list">
					<view
						v-for="(item, idx) in leaderboard"
						:key="item.user_id"
						class="rank-item"
					>
						<view class="rank-num" :class="{ 'top-1': idx === 0, 'top-2': idx === 1, 'top-3': idx === 2 }">
							<text>{{ idx + 1 }}</text>
						</view>
						<image
							class="rank-avatar"
							:src="resolveAvatarUrl(item.avatar, item.user_id)"
							mode="aspectFill"
						></image>
						<view class="rank-info">
							<text class="rank-name">{{ item.nickname || '球友' }}</text>
							<text class="rank-score">{{ item.rank_score }} 分</text>
						</view>
						<view class="rank-stats">
							<text class="rank-wins">{{ item.wins }}胜</text>
						</view>
					</view>
				</view>
				<view v-else class="empty-list">
					<text>暂无排行数据</text>
				</view>

				<!-- 加载更多 -->
				<view v-if="leaderboard.length > 0 && hasMore" class="load-more" @tap="loadMoreLeaderboard">
					<text>加载更多</text>
				</view>
				<view v-if="leaderboard.length > 0 && !hasMore" class="no-more">
					<text>没有更多了</text>
				</view>
				</view>
			</template>
		</view>
	</view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { getSeasonLeaderboard, getSeasonOverview } from '@/api/season.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { useUserStore } from '@/store/user.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { resolveCurrentSeasonState } from '@/utils/honor-wall.js'
import { resolveSeasonTimeline } from '@/utils/season-lifecycle.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
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
const userDataInvalidationStore = useUserDataInvalidationStore()
const userStore = useUserStore()

const statusMap = { 0: '未开始', 1: '进行中', 2: '已结束' }
const gameTypeTabs = GAME_TYPE_TABS

const season = ref(null)
const seasonState = ref('not_started')
const myRecord = ref(null)
const leaderboard = ref([])
const loading = ref(false)
const page = ref(1)
const hasMore = ref(true)
const currentGameType = ref(3)
const hasLoadedOnce = ref(false)
const loadedSeasonScopeVersion = ref(0)
const loadedIdentityKey = ref('')
const seasonPageState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: null
}))

const seasonEmptyState = computed(() => resolveCurrentSeasonState(season.value, seasonState.value))
const hasActiveSeason = computed(() => seasonState.value === 'active' && Boolean(season.value))
const seasonTimeline = computed(() => resolveSeasonTimeline(season.value))

const winRate = computed(() => {
	if (!myRecord.value || myRecord.value.matches_played === 0) return 0
	return Math.round((myRecord.value.wins / myRecord.value.matches_played) * 100)
})

const remainDays = computed(() => seasonTimeline.value.remainDays)
const progressPercent = computed(() => seasonTimeline.value.progressPercent)
const seasonPageError = computed(() => (
	seasonPageState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(seasonPageState.value.error, { resource: '赛季信息' })
		: null
))
const seasonRefreshError = computed(() => (
	seasonPageState.value.refreshError
		? resolveAsyncPageErrorFeedback(seasonPageState.value.refreshError, { resource: '赛季信息' })
		: null
))

const currentSeasonScopeVersion = () => userDataInvalidationStore.versionOf('season')
const currentIdentityKey = () => `${userStore.userId}:${userStore.authGeneration}`

const loadSeasonOverview = async () => {
	if (seasonPageState.value.status === ASYNC_PAGE_STATUS.LOADING || seasonPageState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const requestIdentityKey = currentIdentityKey()
	if (seasonPageState.value.authGeneration !== userStore.authGeneration) {
		season.value = null
		seasonState.value = 'not_started'
		myRecord.value = null
		leaderboard.value = []
		page.value = 1
		hasMore.value = false
		hasLoadedOnce.value = false
		seasonPageState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: null })
	}
	const nextState = beginAsyncPageLoad(seasonPageState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: null
	})
	const pageRequest = getAsyncPageRequest(nextState)
	seasonPageState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const res = await getSeasonOverview({ game_type: currentGameType.value })
		if (currentIdentityKey() !== requestIdentityKey || seasonPageState.value.requestId !== pageRequest.requestId || seasonPageState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '获取赛季概览失败', category: 'business' })
		const nextSeason = res.season || null
		const nextSeasonState = res.season_state || (nextSeason ? 'active' : 'not_started')
		const hasNextActiveSeason = nextSeasonState === 'active' && Boolean(nextSeason)
		if (hasNextActiveSeason && (res.availability?.record === false || res.availability?.leaderboard === false)) {
			throw createRequestError({ message: '赛季数据暂时不完整，请稍后重试', category: 'business' })
		}
		season.value = nextSeason
		seasonState.value = nextSeasonState
		if (!hasActiveSeason.value) {
			myRecord.value = null
			leaderboard.value = []
			hasMore.value = false
			hasLoadedOnce.value = true
			loadedIdentityKey.value = requestIdentityKey
			loadedSeasonScopeVersion.value = currentSeasonScopeVersion()
			seasonPageState.value = resolveAsyncPageLoad(seasonPageState.value, pageRequest, {
				data: res,
				isEmpty: () => false
			})
			return
		}
		myRecord.value = res.record || null
		leaderboard.value = res.leaderboard || []
		page.value = 1
		hasMore.value = leaderboard.value.length >= 20
		hasLoadedOnce.value = true
		loadedIdentityKey.value = requestIdentityKey
		loadedSeasonScopeVersion.value = currentSeasonScopeVersion()
		seasonPageState.value = resolveAsyncPageLoad(seasonPageState.value, pageRequest, {
			data: res,
			isEmpty: () => false
		})
	} catch (e) {
		if (currentIdentityKey() !== requestIdentityKey || seasonPageState.value.requestId !== pageRequest.requestId || seasonPageState.value.authGeneration !== pageRequest.authGeneration) return
		seasonPageState.value = rejectAsyncPageLoad(seasonPageState.value, pageRequest, e)
	} finally {
		if (currentIdentityKey() !== requestIdentityKey || seasonPageState.value.requestId !== pageRequest.requestId || seasonPageState.value.authGeneration !== pageRequest.authGeneration) return
		loading.value = false
	}
}

const fetchLeaderboard = async (isRefresh = false, isLoadMore = false) => {
	if (!hasActiveSeason.value) return
	if (seasonPageState.value.status === ASYNC_PAGE_STATUS.LOADING || seasonPageState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const requestIdentityKey = currentIdentityKey()
	const requestedPage = page.value
	const nextState = beginAsyncPageLoad(seasonPageState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: null
	})
	const pageRequest = getAsyncPageRequest(nextState)
	seasonPageState.value = nextState
	try {
		const res = await getSeasonLeaderboard({
			season_id: season.value.id,
			game_type: currentGameType.value,
			page: page.value,
			page_size: 20
		})
		if (currentIdentityKey() !== requestIdentityKey || seasonPageState.value.requestId !== pageRequest.requestId || seasonPageState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '获取赛季排行榜失败', category: 'business' })
		const newList = res.list || []
		if (isRefresh) {
			leaderboard.value = newList
		} else {
			leaderboard.value = [...leaderboard.value, ...newList]
		}
		hasMore.value = leaderboard.value.length < (res.total || 0)
		seasonPageState.value = resolveAsyncPageLoad(seasonPageState.value, pageRequest, {
			data: { season: season.value, leaderboard: leaderboard.value },
			isEmpty: () => false
		})
	} catch (e) {
		if (currentIdentityKey() !== requestIdentityKey || seasonPageState.value.requestId !== pageRequest.requestId || seasonPageState.value.authGeneration !== pageRequest.authGeneration) return
		if (isLoadMore && requestedPage > 1) page.value = requestedPage - 1
		seasonPageState.value = rejectAsyncPageLoad(seasonPageState.value, pageRequest, e)
	}
}

const loadMoreLeaderboard = () => {
	if (!hasActiveSeason.value || !hasMore.value || seasonPageState.value.status === ASYNC_PAGE_STATUS.LOADING || seasonPageState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	page.value++
	fetchLeaderboard(false, true)
}

const goReport = () => {
	if (hasActiveSeason.value) {
		uni.navigateTo({ url: `/subPages/season/report?id=${season.value.id}&game_type=${currentGameType.value}` })
	}
}

const handleGameTypeChange = async (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	resetSeasonData()
	page.value = 1
	hasMore.value = true
	await loadSeasonOverview()
}

const resetSeasonData = () => {
	season.value = null
	seasonState.value = 'not_started'
	myRecord.value = null
	leaderboard.value = []
	hasLoadedOnce.value = false
	seasonPageState.value = {
		...seasonPageState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: seasonPageState.value.requestId + 1,
		data: null,
		hasData: false,
		error: null,
		refreshError: null
	}
}

const retrySeason = () => loadSeasonOverview()

const handleSeasonErrorAction = () => {
	if (seasonPageState.value.error?.category === 'permission' || seasonPageState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retrySeason()
}

onMounted(() => {
	loadSeasonOverview()
})

onShow(() => {
	if (!hasLoadedOnce.value || seasonPageState.value.authGeneration !== userStore.authGeneration || loadedIdentityKey.value !== currentIdentityKey() || loadedSeasonScopeVersion.value !== currentSeasonScopeVersion()) {
		loadSeasonOverview()
	}
})
</script>

<style lang="scss" scoped>
.season-page {
	min-height: 100vh;
	background: #FAF8F2;
}
.loading-state {
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
	min-height: 60vh;
	.loading-text { font-size: 28rpx; color: #9A8C67; margin-top: 16rpx; }
}
.page-error-state { min-height: 60vh; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16rpx; padding: 0 48rpx; text-align: center; }
.page-error-title { font-size: 32rpx; font-weight: 600; color: #231C0B; }
.page-error-text { font-size: 24rpx; line-height: 1.7; color: #6E6242; }
.retry-btn { min-width: 200rpx; height: 76rpx; line-height: 76rpx; margin: 12rpx 0 0; padding: 0 32rpx; border-radius: 38rpx; background: #E0AE12; color: #ffffff; font-size: 28rpx; font-weight: 600; }
.retry-btn::after { display: none; }
.refresh-error-banner { display: flex; align-items: center; justify-content: space-between; gap: 20rpx; margin-bottom: 20rpx; padding: 18rpx 22rpx; border-radius: 14rpx; background: rgba(224, 174, 18, 0.12); color: #8a5b00; font-size: 24rpx; }
.refresh-error-action { flex-shrink: 0; color: #a86f00; font-weight: 600; }
.season-content {
	padding: 20rpx 24rpx;
}
.game-type-tabs {
	display: flex;
	gap: 12rpx;
	margin-bottom: 20rpx;
	overflow-x: auto;
}
.game-type-tab {
	flex-shrink: 0;
	min-width: 150rpx;
	padding: 14rpx 24rpx;
	border-radius: 999rpx;
	background: rgba(255, 255, 255, 0.86);
	border: 1rpx solid rgba(148, 163, 184, 0.24);
	box-sizing: border-box;
	text-align: center;
	text {
		font-size: 24rpx;
		font-weight: 600;
		color: #6E6242;
	}
	&.active {
		background: #fff7dc;
		border: 1rpx solid rgba(224, 174, 18, 0.22);
		text {
			color: #7c5b05;
		}
	}
	}
	.season-card {
		background: linear-gradient(135deg, #5f4306 0%, #b68108 100%);
	border-radius: 20rpx;
	padding: 32rpx;
	margin-bottom: 20rpx;
	.season-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12rpx;
		.season-name { font-size: 36rpx; font-weight: 700; color: #fff; }
		.season-status {
			padding: 6rpx 16rpx;
			border-radius: 8rpx;
			font-size: 22rpx;
			background: rgba(255,255,255,0.2);
			color: #fff;
			&.status-1 { background: rgba(34,197,94,0.3); }
			&.status-2 { background: rgba(148,163,184,0.3); }
		}
	}
	.season-dates { font-size: 24rpx; color: rgba(255,255,255,0.7); margin-bottom: 16rpx; }
	.season-progress {
		.progress-bar {
			height: 12rpx;
			background: rgba(255,255,255,0.2);
			border-radius: 6rpx;
			overflow: hidden;
			margin-bottom: 8rpx;
			.progress-fill { height: 100%; background: #22c55e; border-radius: 6rpx; min-width: 4rpx; }
		}
		.progress-text { font-size: 22rpx; color: rgba(255,255,255,0.8); }
	}
}
.no-season-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 60rpx 32rpx;
	margin-bottom: 20rpx;
	display: flex;
	flex-direction: column;
	align-items: center;
	.no-season-icon { font-size: 64rpx; margin-bottom: 16rpx; }
	.no-season-text { font-size: 28rpx; color: #9A8C67; }
	.no-season-description { margin-top: 12rpx; font-size: 24rpx; color: #9A8C67; text-align: center; line-height: 1.6; }
}
.my-record-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 32rpx;
	margin-bottom: 20rpx;
	.record-stats {
		display: flex;
		justify-content: space-around;
		margin-top: 20rpx;
		.stat-item {
			text-align: center;
			.stat-value { font-size: 40rpx; font-weight: 700; color: #231C0B; display: block; }
			.stat-label { font-size: 22rpx; color: #9A8C67; }
		}
	}
		.record-rank {
			margin-top: 20rpx;
			text-align: center;
			font-size: 26rpx;
			color: #c69200;
			font-weight: 500;
		}
}
.leaderboard-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 32rpx;
	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 20rpx;
			.report-btn {
				padding: 8rpx 20rpx;
				background: #e0ae12;
				border-radius: 8rpx;
				font-size: 24rpx;
				color: #231c0b;
			}
		}
}
.card-title {
	font-size: 30rpx;
	font-weight: 600;
	color: #231C0B;
}
.leaderboard-list {
	display: flex;
	flex-direction: column;
}
.rank-item {
	display: flex;
	align-items: center;
	padding: 16rpx 0;
	border-bottom: 1rpx solid #FAF8F2;
	&:last-child { border-bottom: none; }
	.rank-num {
		width: 48rpx;
		height: 48rpx;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 24rpx;
		font-weight: 600;
		color: #9A8C67;
		background: #FAF8F2;
		margin-right: 16rpx;
		&.top-1 { background: #fef3c7; color: #d97706; }
		&.top-2 { background: #FAF8F2; color: #6E6242; }
		&.top-3 { background: #fed7aa; color: #c2410c; }
	}
	.rank-avatar {
		width: 72rpx;
		height: 72rpx;
		border-radius: 50%;
		margin-right: 16rpx;
		background: #E9E2CF;
	}
	.rank-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		.rank-name { font-size: 28rpx; color: #231C0B; font-weight: 500; }
		.rank-score { font-size: 22rpx; color: #9A8C67; margin-top: 4rpx; }
	}
		.rank-stats {
			.rank-wins { font-size: 24rpx; color: #c69200; font-weight: 500; }
		}
	}
.empty-list {
	text-align: center;
	padding: 40rpx;
	font-size: 28rpx;
	color: #9A8C67;
}
	.load-more {
		text-align: center;
		padding: 24rpx;
		font-size: 26rpx;
		color: #c69200;
	}
.no-more {
	text-align: center;
	padding: 24rpx;
	font-size: 24rpx;
	color: #9A8C67;
}

.season-page.dark-mode {
	background: #141109;

	.game-type-tab,
	.no-season-card,
	.my-record-card,
	.leaderboard-card {
		background: #1e180d;
		border-color: #3a2e16;
	}

	.game-type-tab text,
	.loading-text,
	.no-season-text,
	.no-season-description,
	.stat-label,
	.rank-score,
	.empty-list,
	.no-more {
		color: #9f926e;
	}

	.game-type-tab.active {
		background: rgba(224, 174, 18, 0.18);
		border-color: rgba(224, 174, 18, 0.42);

		text {
			color: #f7e7a8;
		}
	}

	.card-title,
	.my-record-card .record-stats .stat-value,
	.rank-item .rank-info .rank-name {
		color: #fff7e1;
	}

	.rank-item {
		border-bottom-color: #3a2e16;

		.rank-num,
		.rank-num.top-2,
		.rank-avatar {
			background: #3a2e16;
			color: #d7c89b;
		}

		.rank-num.top-1 {
			background: rgba(224, 174, 18, 0.2);
			color: #fbbf24;
		}

		.rank-num.top-3 {
			background: rgba(249, 115, 22, 0.18);
			color: #fdba74;
		}
	}
}
</style>
