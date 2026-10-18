<template>
	<view class="season-history-page" :class="{ 'dark-mode': isDarkMode }">
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

		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="historyPageError" class="page-error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="page-error-title">{{ historyPageError.title }}</text>
			<text class="page-error-text">{{ historyPageError.description }}</text>
			<button class="retry-btn" @tap="handleHistoryErrorAction">{{ historyPageError.actionText }}</button>
		</view>

		<view v-else-if="historyState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
			<text class="empty-title">暂无已结束的赛季</text>
			<text class="empty-description">赛季结算后，这里会留下你的排名与赛季报告</text>
		</view>

		<view v-else class="history-list">
			<view
				v-for="item in historyItems"
				:key="item.id"
				class="history-item"
				:class="{ disabled: !item.hasRecord }"
				@tap="openReport(item)"
			>
				<view class="history-item-header">
					<text class="history-season-name">{{ item.name }}</text>
					<text class="history-season-dates">{{ item.dateRange }}</text>
				</view>
				<view v-if="item.hasRecord" class="history-record">
					<text class="history-rank">{{ item.rankText }}</text>
					<view class="history-stats">
						<text>{{ item.matches }} 场</text>
						<text>{{ item.wins }} 胜</text>
						<text>胜率 {{ item.winRate }}%</text>
					</view>
					<uni-icons type="right" size="14" :color="isDarkMode ? '#9f926e' : '#9A8C67'"></uni-icons>
				</view>
				<text v-else class="history-absent">未参加该球种排位</text>
			</view>

			<view v-if="historyRefreshError" class="load-error">
				<text>{{ historyRefreshError.description }}</text>
				<text class="load-error-action" @tap="loadMore">重试</text>
			</view>
			<view v-else-if="hasMore" class="load-more" @tap="loadMore">
				<text>{{ loadingMore ? '加载中...' : '加载更多' }}</text>
			</view>
			<view v-else class="no-more">
				<text>没有更多了</text>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getSeasonHistory } from '@/api/season.js'
import { useUserStore } from '@/store/user.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { SEASON_HISTORY_PAGE_SIZE, buildSeasonHistoryItem, hasMoreSeasonHistory } from '@/utils/season-history.js'
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
const gameTypeTabs = GAME_TYPE_TABS

const currentGameType = ref(3)
const historyItems = ref([])
const total = ref(0)
const page = ref(1)
const lastPageCount = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const loadedIdentityKey = ref('')
const historyState = ref(createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] }))

const currentIdentityKey = () => `${userStore.userId}:${userStore.authGeneration}`
const hasMore = computed(() => hasMoreSeasonHistory({
	loaded: historyItems.value.length,
	total: total.value,
	lastPageCount: lastPageCount.value
}))
const historyPageError = computed(() => (
	historyState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(historyState.value.error, { resource: '历届赛季' })
		: null
))
const historyRefreshError = computed(() => (
	historyState.value.refreshError
		? resolveAsyncPageErrorFeedback(historyState.value.refreshError, { resource: '历届赛季' })
		: null
))

const loadSeasonHistory = async ({ append = false } = {}) => {
	if (historyState.value.status === ASYNC_PAGE_STATUS.LOADING || historyState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const requestIdentityKey = currentIdentityKey()
	const requestGameType = currentGameType.value
	const requestPage = append ? page.value + 1 : 1
	const nextState = beginAsyncPageLoad(historyState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	historyState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	loadingMore.value = append
	const isCurrentRequest = () => (
		currentIdentityKey() === requestIdentityKey &&
		currentGameType.value === requestGameType &&
		historyState.value.requestId === pageRequest.requestId &&
		historyState.value.authGeneration === pageRequest.authGeneration
	)
	try {
		const res = await getSeasonHistory({ game_type: requestGameType, page: requestPage, page_size: SEASON_HISTORY_PAGE_SIZE })
		if (!isCurrentRequest()) return
		if (!res?.success) throw createRequestError({ message: res?.message || '获取历届赛季失败', category: 'business' })
		const nextItems = (res.list || []).map(item => buildSeasonHistoryItem(item, requestGameType)).filter(Boolean)
		historyItems.value = append ? [...historyItems.value, ...nextItems] : nextItems
		total.value = Number(res.total) || 0
		page.value = requestPage
		lastPageCount.value = nextItems.length
		loadedIdentityKey.value = requestIdentityKey
		historyState.value = resolveAsyncPageLoad(historyState.value, pageRequest, { data: historyItems.value })
	} catch (e) {
		if (!isCurrentRequest()) return
		historyState.value = rejectAsyncPageLoad(historyState.value, pageRequest, e)
	} finally {
		if (isCurrentRequest()) {
			loading.value = false
			loadingMore.value = false
		}
	}
}

const resetHistory = () => {
	historyItems.value = []
	total.value = 0
	page.value = 1
	lastPageCount.value = 0
	loading.value = false
	loadingMore.value = false
	historyState.value = {
		...createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] }),
		requestId: historyState.value.requestId + 1
	}
}

const loadMore = () => {
	if (!historyRefreshError.value && !hasMore.value) return
	loadSeasonHistory({ append: true })
}

const handleGameTypeChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	resetHistory()
	loadSeasonHistory()
}

const handleHistoryErrorAction = () => {
	if (historyState.value.error?.category === 'permission' || historyState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	loadSeasonHistory()
}

const openReport = (item) => {
	if (!item?.hasRecord || !item.reportUrl) return
	uni.navigateTo({ url: item.reportUrl })
}

onLoad((options) => {
	const gameType = parseInt(options?.game_type || 0)
	if (gameTypeTabs.some(item => item.value === gameType)) {
		currentGameType.value = gameType
	}
	loadSeasonHistory()
})

onShow(() => {
	if (!loadedIdentityKey.value || loadedIdentityKey.value === currentIdentityKey()) return
	resetHistory()
	loadSeasonHistory()
})
</script>

<style lang="scss" scoped>
.season-history-page {
	min-height: 100vh;
	padding: 20rpx 24rpx 40rpx;
	background: #FAF8F2;
	box-sizing: border-box;
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
	text { font-size: 24rpx; font-weight: 600; color: #6E6242; }
	&.active {
		background: #fff7dc;
		border-color: rgba(224, 174, 18, 0.22);
		text { color: #7c5b05; }
	}
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
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 12rpx;
	padding: 120rpx 48rpx;
	text-align: center;
	.empty-title { font-size: 30rpx; font-weight: 600; color: #231C0B; }
	.empty-description { font-size: 24rpx; line-height: 1.6; color: #9A8C67; }
}
.history-item {
	margin-bottom: 20rpx;
	padding: 28rpx 32rpx;
	border-radius: 20rpx;
	background: #fff;
	&.disabled { opacity: 0.72; }
}
.history-item-header {
	display: flex;
	align-items: baseline;
	justify-content: space-between;
	gap: 16rpx;
	.history-season-name { font-size: 32rpx; font-weight: 700; color: #231C0B; }
	.history-season-dates { font-size: 22rpx; color: #9A8C67; }
}
.history-record {
	display: flex;
	align-items: center;
	gap: 20rpx;
	margin-top: 20rpx;
	.history-rank { flex-shrink: 0; font-size: 30rpx; font-weight: 700; color: #c69200; }
	.history-stats { flex: 1; display: flex; flex-wrap: wrap; gap: 20rpx; font-size: 24rpx; color: #6E6242; }
}
.history-absent { display: block; margin-top: 20rpx; font-size: 24rpx; color: #9A8C67; }
.load-error { display: flex; align-items: center; justify-content: space-between; gap: 20rpx; padding: 18rpx 22rpx; border-radius: 14rpx; border: 1rpx solid #EF4444; background: rgba(239, 68, 68, 0.06); color: #b91c1c; font-size: 24rpx; }
.load-error-action { flex-shrink: 0; padding: 8rpx 0 8rpx 16rpx; font-weight: 600; }
.load-more { text-align: center; padding: 24rpx; font-size: 26rpx; color: #c69200; }
.no-more { text-align: center; padding: 24rpx; font-size: 24rpx; color: #9A8C67; }

.season-history-page.dark-mode {
	background: #141109;

	.game-type-tab,
	.history-item {
		background: #1e180d;
		border-color: #3a2e16;
	}

	.game-type-tab text,
	.loading-text,
	.empty-description,
	.history-season-dates,
	.history-absent,
	.no-more {
		color: #9f926e;
	}

	.game-type-tab.active {
		background: rgba(224, 174, 18, 0.18);
		border-color: rgba(224, 174, 18, 0.42);

		text { color: #f7e7a8; }
	}

	.page-error-title,
	.empty-title,
	.history-season-name {
		color: #fff7e1;
	}

	.page-error-text,
	.history-stats {
		color: #d7c89b;
	}

	.history-rank { color: #fbbf24; }

	.load-error {
		background: rgba(239, 68, 68, 0.12);
		color: #fca5a5;
	}
}
</style>
