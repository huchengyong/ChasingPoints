<template>
	<view class="tournament-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="page-hero">
			<view class="hero-copy">
				<text class="hero-eyebrow">赛事情报</text>
				<text class="hero-title">最近的赛程、赛况和赛果集中看清楚</text>
				<text class="hero-desc">按球种和状态筛选，快速锁定正在进行、即将开始和刚更新结果的重点赛事。</text>
			</view>
			<view class="hero-stats">
				<view class="hero-stat">
					<text class="hero-stat-label">已收录</text>
					<text class="hero-stat-value">{{ totalCount }}</text>
				</view>
				<view class="hero-stat">
					<text class="hero-stat-label">进行中</text>
					<text class="hero-stat-value">{{ liveCount }}</text>
				</view>
				<view class="hero-stat">
					<text class="hero-stat-label">即将开始</text>
					<text class="hero-stat-value">{{ upcomingCount }}</text>
				</view>
			</view>
		</view>

		<view class="toolbar">
			<view class="filter-bar">
				<picker :range="gameTypes" range-key="label" @change="onGameTypeChange">
					<view class="filter-item">
						<view class="filter-copy">
							<text class="filter-label">球种</text>
							<text class="filter-value">{{ selectedGameType.label }}</text>
						</view>
						<uni-icons type="bottom" size="18" color="#C69200"></uni-icons>
					</view>
				</picker>
				<picker :range="statusList" range-key="label" @change="onStatusChange">
					<view class="filter-item">
						<view class="filter-copy">
							<text class="filter-label">状态</text>
							<text class="filter-value">{{ selectedStatus.label }}</text>
						</view>
						<uni-icons type="bottom" size="18" color="#C69200"></uni-icons>
					</view>
				</picker>
			</view>
			<view class="toolbar-tip">
				<text>统计基于当前已加载内容，继续下滑可补充更多赛事情报。</text>
			</view>
		</view>

		<view v-if="loading && page === 1 && list.length === 0" class="loading-state">
				<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<scroll-view
			v-else
			scroll-y
			class="tournament-scroll"
			@scrolltolower="loadMore"
			refresher-enabled
			:refresher-triggered="refreshing"
			@refresherrefresh="onRefresh"
		>
			<view v-if="list.length > 0" class="tournament-list">
				<view v-if="tournamentRefreshError" class="refresh-error-banner">
					<text>{{ tournamentRefreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryTournaments">重试</text>
				</view>
				<view
					v-for="item in list"
					:key="item.id"
					class="tournament-card"
					@tap="goDetail(item.id)"
				>
					<image class="card-cover" :src="item.coverImage" mode="aspectFill"></image>
					<view class="card-topline">
						<view class="card-chip game-chip">
							<text>{{ item.gameTypeText }}</text>
						</view>
						<text class="card-date">{{ item.dateText }}</text>
						<view class="status-tag" :class="'status-' + item.status">
							<text>{{ item.statusText }}</text>
						</view>
					</view>
					<text class="card-title">{{ item.title }}</text>
					<text class="card-summary">{{ item.summary }}</text>

					<view class="card-meta">
						<view class="meta-item">
							<text class="meta-label">日期</text>
							<text class="meta-value">{{ item.dateText }}</text>
						</view>
						<view class="meta-item" v-if="item.showTime">
							<text class="meta-label">时间</text>
							<text class="meta-value">{{ item.timeText }}</text>
						</view>
						<view class="meta-item" v-if="item.locationText">
							<text class="meta-label">地点</text>
							<text class="meta-value">{{ item.locationText }}</text>
						</view>
					<view class="meta-item">
						<text class="meta-label">当前轮次</text>
						<text class="meta-value">{{ item.currentRoundText }}</text>
					</view>
					<view class="meta-item">
						<text class="meta-label">比赛数</text>
						<text class="meta-value">{{ item.matchCountText }}</text>
					</view>
					</view>

					<view class="card-footer">
						<text class="footer-source">{{ item.sourceText || '手动录入' }}</text>
						<text class="footer-link">查看详情</text>
					</view>
				</view>
			</view>

			<view v-else-if="tournamentPageError" class="empty-state error-state">
				<view class="empty-badge">
					<uni-icons type="info" size="28" color="#E0AE12"></uni-icons>
				</view>
				<text class="empty-title">{{ tournamentPageError.title }}</text>
				<text class="empty-text">{{ tournamentPageError.description }}</text>
				<button class="retry-btn" @tap="handleTournamentErrorAction">{{ tournamentPageError.actionText }}</button>
			</view>

			<view v-else-if="tournamentState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
				<view class="empty-badge">
						<uni-icons type="calendar" size="28" color="#E0AE12"></uni-icons>
				</view>
				<text class="empty-title">当前筛选下还没有赛事情报</text>
				<text class="empty-text">赛事正在更新中，稍后再来刷新，或者换个球种看看。</text>
			</view>

			<view v-if="list.length > 0 && !hasMore" class="no-more">
				<text>没有更多了</text>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { usePublicReadStore } from '@/store/publicRead.js'
import { normalizeSaiXunCard } from '@/utils/saixun.js'
import { usePageTheme } from '@/utils/page-theme.js'
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
const publicReadStore = usePublicReadStore()

const gameTypes = ref([
	{ label: '全部球种', value: 0 },
	{ label: '斯诺克', value: 1 },
	{ label: '中式八球', value: 3 },
	{ label: '中式九球', value: 2 }
])

const statusList = ref([
	{ label: '全部状态', value: -1 },
	{ label: '即将开始', value: 0 },
	{ label: '进行中', value: 1 },
	{ label: '已结束', value: 2 },
	{ label: '已取消', value: 3 }
])

const selectedGameType = ref(gameTypes.value[0])
const selectedStatus = ref(statusList.value[0])

const list = ref([])
const loading = ref(false)
const refreshing = ref(false)
const page = ref(1)
const hasMore = ref(true)
const totalCount = ref(0)
const tournamentState = ref(createAsyncPageState({ data: [] }))

const liveCount = computed(() => list.value.filter(item => item.status === 1).length)
const upcomingCount = computed(() => list.value.filter(item => item.status === 0).length)
const tournamentPageError = computed(() => (
	tournamentState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(tournamentState.value.error, { resource: '赛事情报' })
		: null
))
const tournamentRefreshError = computed(() => (
	tournamentState.value.refreshError
		? resolveAsyncPageErrorFeedback(tournamentState.value.refreshError, { resource: '赛事情报' })
		: null
))

const buildQueryParams = (pageValue, gameTypeValue, statusValue) => {
	const params = {
		page: pageValue,
		page_size: 10
	}
	if (gameTypeValue > 0) params.game_type = gameTypeValue
	if (statusValue >= 0) params.status = statusValue
	return params
}

const fetchList = async ({
	pageValue = page.value,
	gameTypeValue = selectedGameType.value.value,
	statusValue = selectedStatus.value.value,
	replace = false,
	commitSelection = null,
	force = false
} = {}) => {
	if (tournamentState.value.status === ASYNC_PAGE_STATUS.LOADING || tournamentState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return false
	const nextState = beginAsyncPageLoad(tournamentState.value, { emptyData: [] })
	const pageRequest = getAsyncPageRequest(nextState)
	tournamentState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const res = await publicReadStore.loadEventNews(
			buildQueryParams(pageValue, gameTypeValue, statusValue),
			{ force }
		)
		if (tournamentState.value.requestId !== pageRequest.requestId) return false
		if (!res?.success) throw createRequestError({ message: res?.message || '获取赛事情报列表失败', category: 'business' })
		if (!Array.isArray(res.list)) {
			throw createRequestError({ message: '赛事情报数据格式异常，请稍后重试', category: 'business' })
		}
		const newList = (res.list || []).map((item) => normalizeSaiXunCard(item))
		if (commitSelection) {
			if (commitSelection.gameType) selectedGameType.value = commitSelection.gameType
			if (commitSelection.status) selectedStatus.value = commitSelection.status
		}
		if (replace) {
			list.value = newList
		} else {
			list.value = [...list.value, ...newList]
		}
		page.value = pageValue
		totalCount.value = res.total || 0
		hasMore.value = list.value.length < (res.total || 0)
		tournamentState.value = resolveAsyncPageLoad(tournamentState.value, pageRequest, { data: list.value })
		return true
	} catch (e) {
		if (tournamentState.value.requestId !== pageRequest.requestId) return false
		tournamentState.value = rejectAsyncPageLoad(tournamentState.value, pageRequest, e)
		return false
	} finally {
		if (tournamentState.value.requestId !== pageRequest.requestId) return
		loading.value = false
		refreshing.value = false
	}
}

const onRefresh = async () => {
	if (tournamentState.value.status === ASYNC_PAGE_STATUS.LOADING || tournamentState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	refreshing.value = true
	await fetchList({
		pageValue: 1,
		replace: true,
		force: true
	})
}

const loadMore = async () => {
	if (!hasMore.value || tournamentState.value.status === ASYNC_PAGE_STATUS.LOADING || tournamentState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	await fetchList({
		pageValue: page.value + 1,
		replace: false
	})
}

const onGameTypeChange = async (e) => {
	if (tournamentState.value.status === ASYNC_PAGE_STATUS.LOADING || tournamentState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const nextGameType = gameTypes.value[e.detail.value]
	selectedGameType.value = nextGameType
	resetTournamentList()
	await fetchList({
		pageValue: 1,
		gameTypeValue: nextGameType.value,
		statusValue: selectedStatus.value.value,
		replace: true
	})
}

const onStatusChange = async (e) => {
	if (tournamentState.value.status === ASYNC_PAGE_STATUS.LOADING || tournamentState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const nextStatus = statusList.value[e.detail.value]
	selectedStatus.value = nextStatus
	resetTournamentList()
	await fetchList({
		pageValue: 1,
		gameTypeValue: selectedGameType.value.value,
		statusValue: nextStatus.value,
		replace: true
	})
}

const resetTournamentList = () => {
	list.value = []
	page.value = 1
	totalCount.value = 0
	hasMore.value = false
	tournamentState.value = {
		...tournamentState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: tournamentState.value.requestId + 1,
		data: [],
		hasData: false,
		error: null,
		refreshError: null
	}
}

const retryTournaments = () => fetchList({ pageValue: 1, replace: true, force: true })

const handleTournamentErrorAction = () => {
	if (tournamentState.value.error?.category === 'permission' || tournamentState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryTournaments()
}

const goDetail = (id) => {
	uni.navigateTo({ url: '/subPages/tournament/detail?id=' + id })
}

onMounted(() => fetchList({
	pageValue: 1,
	replace: true
}))
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
