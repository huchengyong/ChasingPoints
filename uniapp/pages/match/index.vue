<template>
	<view class="match-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 观赛列表 -->
		<view class="lobby-bar">
			<view class="lobby-tabs">
				<view
					v-for="tab in lobbyTabs"
					:key="tab.value"
					class="lobby-tab"
					:class="{ 'lobby-tab--active': lobbyTab === tab.value }"
					@tap="selectLobbyTab(tab.value)"
				>
					<text class="lobby-tab-text">{{ tab.label }}</text>
				</view>
			</view>
			<view class="lobby-filter" @tap="openGameTypeSheet">
				<text class="lobby-filter-text" :class="{ 'lobby-filter-text--active': currentGameType !== 0 }">{{ selectedGameTypeLabel }}</text>
				<uni-icons type="bottom" size="12" :color="mutedIconColor"></uni-icons>
			</view>
		</view>

		<view v-if="showPageLoading" class="lobby-skeleton">
			<view v-for="index in 3" :key="index" class="skeleton-card">
				<view class="skeleton-line skeleton-line--short"></view>
				<view class="skeleton-row">
					<view class="skeleton-avatar"></view>
					<view class="skeleton-line"></view>
				</view>
				<view class="skeleton-row">
					<view class="skeleton-avatar"></view>
					<view class="skeleton-line"></view>
				</view>
			</view>
		</view>

		<view v-else-if="pageError" class="state-block">
			<view class="state-icon">
				<uni-icons type="info" size="30" :color="mutedIconColor"></uni-icons>
			</view>
			<text class="state-title">{{ pageError.title }}</text>
			<text class="state-text">{{ pageError.description }}</text>
			<button class="state-btn" @tap="handlePageErrorAction">{{ pageError.actionText }}</button>
		</view>

		<view v-else-if="lobbyItems.length === 0" class="state-block">
			<view class="state-icon">
				<uni-icons type="eye" size="30" :color="brandStrongColor"></uni-icons>
			</view>
			<text class="state-title">{{ emptyState.title }}</text>
			<text class="state-text">{{ emptyState.subtitle }}</text>
			<button v-if="currentGameType !== 0" class="state-btn state-btn--ghost" @tap="resetGameType">查看全部球种</button>
			<button v-else-if="lobbyTab === 'friends'" class="state-btn state-btn--ghost" @tap="selectLobbyTab('live')">看看大厅对局</button>
		</view>

		<view v-else class="match-list">
			<view v-if="refreshError" class="refresh-error-banner">
				<text>{{ refreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryLobby">重试</text>
			</view>
			<view v-for="section in lobbySections" :key="section.key" class="lobby-section">
				<text v-if="section.title" class="lobby-section-title">{{ section.title }}</text>
				<view
					v-for="match in section.items"
					:key="match.id"
					class="match-card"
					:class="{ 'match-card--mine': match.isMine }"
					@tap="handleMatchClick(match)"
				>
					<view class="match-card-head">
						<text class="match-card-meta">{{ match.metaText }}</text>
						<view class="match-state" :class="match.isLive ? 'match-state--live' : 'match-state--ended'">
							<view v-if="match.isLive" class="match-state-dot"></view>
							<text>{{ match.statusText }}</text>
						</view>
					</view>
					<view
						v-for="(player, index) in match.players"
						:key="index"
						class="match-player"
						:class="player.outcome ? 'is-' + player.outcome : ''"
					>
						<image class="match-player-avatar" :src="resolveAvatarUrl(player.avatar, player.id)" mode="aspectFill"></image>
						<text class="match-player-name">{{ player.name }}</text>
						<text v-if="player.isMe" class="me-tag">我</text>
						<text class="match-player-score">{{ player.score }}</text>
					</view>
				</view>
			</view>
			<view v-if="hasMore" class="load-more">
				<text>{{ loadingMore ? '正在加载...' : '上拉加载更多' }}</text>
			</view>
			<view v-else-if="total > PAGE_SIZE" class="load-more">
				<view class="load-more-line"></view>
				<text>没有更多对局了</text>
				<view class="load-more-line"></view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'

import { onShow, onPullDownRefresh, onReachBottom } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { getPublicMatches } from '@/api/match.js'
import { buildPlayingRoute } from '@/utils/ongoing-match-guard.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import { GAME_TYPE_FILTER_OPTIONS_WITH_ALL } from '@/utils/game-types.js'
import {
	buildSpectatorMatchListParams,
	buildSpectatorSections,
	getSpectatorEmptyState,
	normalizeSpectatorMatch,
	resolveSpectatorLobbyQuery,
	resolveSpectatorLobbyTabs,
	shouldOpenPlayingForSpectatorMatch
} from '@/utils/spectator-lobby.js'
import { buildFinishedMatchDetailRoute } from '@/utils/match-core-flow.js'
import {
	ASYNC_PAGE_STATUS,
	beginAsyncPageLoad,
	createAsyncPageState,
	getAsyncPageRequest,
	isAsyncPageRequestCurrent,
	rejectAsyncPageLoad,
	resolveAsyncPageErrorFeedback,
	resolveAsyncPageLoad,
	shouldIgnoreAsyncPageError
} from '@/utils/async-page-state.js'
import { createRequestError } from '@/utils/request-errors.js'

const PAGE_SIZE = 20

// ========== 状态管理 ==========
const userStore = useUserStore()
const { isDarkMode } = usePageTheme()

// ========== 响应式数据 ==========
const lobbyState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const spectatorMatches = ref([])
const page = ref(1)
const total = ref(0)
const loadingMore = ref(false)
const lobbyTab = ref(userStore.isLoggedIn ? 'friends' : 'live')
const currentGameType = ref(0)
const showPageLoading = computed(() => (
	lobbyState.value.status === ASYNC_PAGE_STATUS.LOADING || lobbyState.value.status === ASYNC_PAGE_STATUS.IDLE
))
const pageError = computed(() => {
	if (lobbyState.value.status !== ASYNC_PAGE_STATUS.ERROR) return null
	return resolveAsyncPageErrorFeedback(lobbyState.value.error, { resource: '对局大厅' })
})
const refreshError = computed(() => (
	lobbyState.value.refreshError
		? resolveAsyncPageErrorFeedback(lobbyState.value.refreshError, { resource: '对局大厅' })
		: null
))
const gameTypeOptions = GAME_TYPE_FILTER_OPTIONS_WITH_ALL
const lobbyTabs = computed(() => resolveSpectatorLobbyTabs(userStore.isLoggedIn))
const lobbyQuery = computed(() => resolveSpectatorLobbyQuery(lobbyTab.value))
const selectedGameTypeLabel = computed(() => (
	(gameTypeOptions.find((item) => item.value === currentGameType.value) || gameTypeOptions[0]).label
))
const hasMore = computed(() => page.value * PAGE_SIZE < total.value)
const mutedIconColor = computed(() => (isDarkMode.value ? '#9F926E' : '#9A8C67'))
const brandStrongColor = computed(() => (isDarkMode.value ? '#F0C542' : '#C69200'))

// 用户信息
const userId = computed(() => userStore.userInfo?.id || 0)
// 本人参与的对局留在列表里，用「我」标注
const lobbyItems = computed(() => (
	spectatorMatches.value.map((match) => normalizeSpectatorMatch(match, { userId: userId.value }))
))
const lobbySections = computed(() => buildSpectatorSections(lobbyItems.value, lobbyTab.value))
const emptyState = computed(() => getSpectatorEmptyState({
	...lobbyQuery.value,
	gameType: currentGameType.value
}))

/**
 * 判断玩家1是否是当前用户
 */
const isPlayer1Me = (match) => {
	if (!userStore.isLoggedIn || !userId.value) return false
	return match.player1_id === userId.value
}

// ========== 生命周期 ==========
// 下拉刷新
onPullDownRefresh(async () => {
	try {
		await loadData()
	} finally {
		uni.stopPullDownRefresh()
	}
})

onReachBottom(() => {
	loadMore()
})

onShow(() => {
	// 退出登录后「好友」标签不再可用，回到公开大厅
	if (!userStore.isLoggedIn && lobbyTab.value === 'friends') {
		lobbyTab.value = 'live'
		reloadLobby()
		return
	}
	if (!hasLoadedOnce.value) loadData()
})

// ========== 方法 ==========

/**
 * 加载数据
 */
const hasLoadedOnce = ref(false)

const loadData = async () => {
	if (lobbyState.value.status === ASYNC_PAGE_STATUS.LOADING) return
	const currentGeneration = userStore.authGeneration
	const nextState = beginAsyncPageLoad(lobbyState.value, {
		authGeneration: currentGeneration,
		emptyData: []
	})
	const request = getAsyncPageRequest(nextState)
	const identityChanged = lobbyState.value.authGeneration !== currentGeneration
	if (identityChanged) spectatorMatches.value = []
	lobbyState.value = nextState

	try {
		const matchResult = await getPublicMatches(buildSpectatorMatchListParams({
			...lobbyQuery.value,
			gameType: currentGameType.value,
			page: 1,
			pageSize: PAGE_SIZE
		})).then(
			value => ({ value }),
			error => ({ error })
		)
		if (!isAsyncPageRequestCurrent(lobbyState.value, request)) return
		if (matchResult.error || !matchResult.value?.success) {
			const error = matchResult.error || createRequestError({
				message: matchResult.value?.message || '加载对局大厅失败',
				category: 'business'
			})
			lobbyState.value = rejectAsyncPageLoad(lobbyState.value, request, error)
			return
		}

		spectatorMatches.value = matchResult.value.list || []
		total.value = Number(matchResult.value.total) || 0
		page.value = 1
		lobbyState.value = resolveAsyncPageLoad(lobbyState.value, request, {
			data: spectatorMatches.value
		})
		hasLoadedOnce.value = true
	} catch (error) {
		lobbyState.value = rejectAsyncPageLoad(lobbyState.value, request, error)
	}
}

const loadMore = async () => {
	if (!hasMore.value || loadingMore.value) return
	if (lobbyState.value.status !== ASYNC_PAGE_STATUS.READY) return
	const request = getAsyncPageRequest(lobbyState.value)
	const nextPage = page.value + 1
	loadingMore.value = true
	try {
		const res = await getPublicMatches(buildSpectatorMatchListParams({
			...lobbyQuery.value,
			gameType: currentGameType.value,
			page: nextPage,
			pageSize: PAGE_SIZE
		}))
		if (!isAsyncPageRequestCurrent(lobbyState.value, request)) return
		if (!res?.success) {
			throw createRequestError({ message: res?.message || '加载更多对局失败', category: 'business' })
		}
		// 翻页期间新开局会让后续页出现重复对局，按 id 去重
		const knownIds = new Set(spectatorMatches.value.map((match) => match.id))
		const nextList = (res.list || []).filter((match) => !knownIds.has(match.id))
		spectatorMatches.value = [...spectatorMatches.value, ...nextList]
		total.value = Number(res.total) || 0
		page.value = nextPage
	} catch (error) {
		if (isAsyncPageRequestCurrent(lobbyState.value, request) && !shouldIgnoreAsyncPageError(error)) {
			uni.showToast({ title: '加载失败，请上拉重试', icon: 'none' })
		}
	} finally {
		loadingMore.value = false
	}
}

// 切换标签或球种：清空旧列表并作废在途请求，避免新标签下短暂显示旧数据。
const reloadLobby = () => {
	lobbyState.value = {
		...lobbyState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: lobbyState.value.requestId + 1,
		data: [],
		hasData: false,
		error: null,
		refreshError: null
	}
	spectatorMatches.value = []
	total.value = 0
	page.value = 1
	loadData()
}

const retryLobby = () => loadData()

const handlePageErrorAction = () => {
	if (lobbyState.value.error?.category === 'permission' || lobbyState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryLobby()
}

const selectLobbyTab = (value) => {
	if (lobbyTab.value === value) return
	lobbyTab.value = value
	reloadLobby()
}

const openGameTypeSheet = () => {
	uni.showActionSheet({
		itemList: gameTypeOptions.map((item) => item.label),
		success: ({ tapIndex }) => {
			const next = gameTypeOptions[tapIndex]
			if (!next || next.value === currentGameType.value) return
			currentGameType.value = next.value
			reloadLobby()
		}
	})
}

const resetGameType = () => {
	currentGameType.value = 0
	reloadLobby()
}

const navigateToPlayingMatch = (match) => {
	uni.navigateTo({
		url: buildPlayingRoute(match)
	})
}

/**
 * 继续进行中的对局（跳转到 playing.vue 继续对局）
 */
const handleContinueMatch = (match) => {
	navigateToPlayingMatch({
		match_id: match.id,
		game_type: match.game_type || 3,
		opponent_name: getOpponentName(match),
		opponent_avatar: getOpponentAvatar(match)
	})
}

/**
 * 获取对手名称（根据当前用户判断）
 */
const getOpponentName = (match) => {
	// CurrentMatchInfo 结构：直接取 opponent_name
	if (match.opponent_name !== undefined) {
		return match.opponent_name || '对手'
	}
	// 平台对局结构：根据当前用户位置取对方名称
	if (isPlayer1Me(match)) {
		return match.player2_name || '对手'
	}
	return match.player1_name || '对手'
}

/**
 * 获取对手头像（根据当前用户判断）
 */
const getOpponentAvatar = (match) => {
	// CurrentMatchInfo 结构：直接取 opponent_avatar
	if (match.opponent_avatar !== undefined) {
		return match.opponent_avatar || ''
	}
	// 平台对局结构：根据当前用户位置取对方头像
	if (isPlayer1Me(match)) {
		return match.player2_avatar || ''
	}
	return match.player1_avatar || ''
}

/**
 * 点击对局卡片
 */
const handleMatchClick = (match) => {
	const finishedDetailUrl = buildFinishedMatchDetailRoute(match, userId.value)
	if (finishedDetailUrl) {
		uni.navigateTo({ url: finishedDetailUrl })
		return
	}

	if (shouldOpenPlayingForSpectatorMatch({
		match,
		userId: userId.value,
		isLoggedIn: userStore.isLoggedIn
	})) {
		// 自己参与的对局，继续对局（进入 playing.vue）
		handleContinueMatch(match)
	} else {
		// 他人对局，进入观战页面
		handleWatchMatch(match)
	}
}

/**
 * 进入观战页面
 */
const handleWatchMatch = (match) => {
	uni.navigateTo({
		url: `/subPages/match/matchDetail?match_id=${match.id}&mode=spectate`
	})
}
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
