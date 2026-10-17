<template>
	<view class="match-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 我的对局 / 开局：同一场对局只在这里出现一次 -->
		<view class="play-card match-primary-action">
			<template v-if="myMatchCard">
				<view class="play-match" @tap="handlePrimaryAction">
					<view class="play-head">
						<view class="state-chip" :class="myMatchCard.isPending ? 'state-chip--pending' : 'state-chip--live'">
							<view class="state-chip-dot"></view>
							<text>{{ myMatchCard.statusText }}</text>
						</view>
						<text class="play-meta">{{ myMatchCard.metaText }}</text>
						<text class="play-duration">{{ myMatchCard.durationText }}</text>
					</view>
					<view class="versus">
						<view class="versus-player">
							<image class="versus-avatar" :src="resolveAvatarUrl(myMatchCard.players[0].avatar, myMatchCard.players[0].id)" mode="aspectFill"></image>
							<view class="versus-name-row">
								<text class="versus-name">{{ myMatchCard.players[0].name }}</text>
								<text v-if="myMatchCard.players[0].isMe" class="me-tag">我</text>
							</view>
						</view>
						<text class="versus-score">{{ myMatchCard.players[0].score }} : {{ myMatchCard.players[1].score }}</text>
						<view class="versus-player">
							<image class="versus-avatar" :src="resolveAvatarUrl(myMatchCard.players[1].avatar, myMatchCard.players[1].id)" mode="aspectFill"></image>
							<view class="versus-name-row">
								<text class="versus-name">{{ myMatchCard.players[1].name }}</text>
								<text v-if="myMatchCard.players[1].isMe" class="me-tag">我</text>
							</view>
						</view>
					</view>
				</view>
				<button class="play-btn" @tap="handlePrimaryAction">{{ myMatchCard.actionLabel }}</button>
			</template>
			<template v-else>
				<view class="play-intro">
					<text class="play-title">来一局</text>
					<text class="play-desc">扫码或选择球友开局，比分实时同步给双方</text>
				</view>
				<button class="play-btn" @tap="handlePrimaryAction">{{ userStore.isLoggedIn ? '发起对局' : '登录后发起对局' }}</button>
			</template>

			<view v-if="userStore.isLoggedIn" class="play-entries">
				<view class="challenge-entry-btn" @tap="goChallengeCompose">
					<uni-icons type="calendar" size="22" :color="entryIconColor"></uni-icons>
					<text class="entry-label">发起约球</text>
				</view>
				<view class="challenge-entry-btn" @tap="goChallenges('received')">
					<view class="entry-icon">
						<uni-icons type="email" size="22" :color="entryIconColor"></uni-icons>
						<text v-if="receivedChallengeCount > 0" class="entry-badge">{{ receivedChallengeCount > 99 ? '99+' : receivedChallengeCount }}</text>
					</view>
					<text class="entry-label">收到的邀请</text>
				</view>
				<view class="challenge-entry-btn" @tap="goChallenges('history')">
					<uni-icons type="list" size="22" :color="entryIconColor"></uni-icons>
					<text class="entry-label">约球记录</text>
				</view>
				<view class="challenge-entry-btn" @tap="handleScanAsReferee">
					<uni-icons type="scan" size="22" :color="entryIconColor"></uni-icons>
					<text class="entry-label">扫码执裁</text>
				</view>
			</view>
		</view>

		<view
			v-if="challengeCard"
			class="challenge-quick-card"
			:class="{ 'challenge-quick-card--action': challengeCard.needsAction }"
			@tap="openChallengeWaiting"
		>
			<view class="challenge-quick-copy">
				<text class="challenge-quick-title">{{ challengeCard.title }}</text>
				<text class="challenge-quick-sub">{{ challengeCard.subText }}</text>
			</view>
			<view class="challenge-quick-action">
				<text>{{ challengeCard.actionLabel }}</text>
			</view>
		</view>
		<view class="challenge-unavailable" v-if="userStore.isLoggedIn && !activityStore.challengeAvailable" @tap="retryChallengeActivity">
			<text class="challenge-unavailable__text">约球状态读取失败，点击重试</text>
		</view>

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

		<!-- 选择比赛类型弹框（用于开始对战） -->
		<gameTypeModal
			v-model:visible="showGameTypeModal"
			:default-type="defaultGameType"
			@confirm="handleGameTypeConfirm"
		/>

		<view v-if="showMatchQrModal" class="match-qr-mask" @click="closeMatchQrModal">
			<view class="match-qr-panel" @click.stop>
				<view class="match-qr-header">
					<view>
						<text class="match-qr-title">出示我的匹配二维码</text>
						<text class="match-qr-subtitle">请让对手使用发起对局扫码入口识别</text>
					</view>
					<button class="match-qr-close" @click="closeMatchQrModal">
						<uni-icons type="closeempty" size="22" :color="mutedIconColor"></uni-icons>
					</button>
				</view>
				<view class="match-qr-body">
					<view v-if="matchQrLoading" class="match-qr-state">
						<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
						<text>二维码生成中...</text>
					</view>
					<view v-if="!matchQrLoading && matchQrFailed" class="match-qr-state">
						<text class="match-qr-error">二维码加载失败</text>
						<button class="match-qr-retry" @click="loadMyMatchQRCode">重试</button>
					</view>
					<canvas
						v-if="matchQrContent && !matchQrFailed"
						canvas-id="match-page-invite-qrcode"
						id="match-page-invite-qrcode"
						class="match-qr-image"
						:class="{ 'is-loading': matchQrLoading }"
						:width="240"
						:height="240"
					/>
				</view>
				<button class="match-qr-done" @click="closeMatchQrModal">关闭</button>
			</view>
		</view>

		<view v-if="showRefereePreview" class="referee-preview-mask" @click="closeRefereePreview">
			<view class="referee-preview-panel" @click.stop>
				<text class="referee-preview-title">确认担任本场裁判</text>
				<text class="referee-preview-desc">确认后，你将负责本场记分和结算，双方选手会切换为只读。</text>

				<view v-if="refereePreviewLoading" class="referee-preview-state">
					<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
					<text>正在核验裁判码...</text>
				</view>

				<view v-else-if="refereePreviewError" class="referee-preview-state">
					<text class="referee-preview-error">{{ refereePreviewError }}</text>
					<button class="referee-preview-retry" @click="retryRefereeScan">重新扫码</button>
				</view>

				<view v-else-if="refereePreview" class="referee-preview-content">
					<view class="referee-preview-players">
						<view class="referee-preview-player">
							<image :src="resolveAvatarUrl(refereePreview.player1_avatar, refereePreview.player1_id)" mode="aspectFill"></image>
							<text>{{ refereePreview.player1_name || '选手1' }}</text>
						</view>
						<text class="referee-preview-vs">VS</text>
						<view class="referee-preview-player">
							<image :src="resolveAvatarUrl(refereePreview.player2_avatar, refereePreview.player2_id)" mode="aspectFill"></image>
							<text>{{ refereePreview.player2_name || '选手2' }}</text>
						</view>
					</view>
					<text class="referee-preview-score">{{ refereePreview.player1_score }} : {{ refereePreview.player2_score }}</text>
					<text class="referee-preview-meta">{{ refereePreview.game_type_name || '对局' }} · {{ getRefereePreviewMode(refereePreview.match_mode) }}</text>
				</view>

				<view v-if="refereePreview && !refereePreviewLoading && !refereePreviewError" class="referee-preview-actions">
					<button class="referee-preview-cancel" :disabled="refereeJoining" @click="closeRefereePreview">暂不担任</button>
					<button class="referee-preview-confirm" :disabled="refereeJoining" @click="confirmRefereeJoin">{{ refereeJoining ? '确认中...' : '确认担任裁判' }}</button>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'

import { onShow, onPullDownRefresh, onReachBottom } from '@dcloudio/uni-app'
import { useActivityStore } from '@/store/activity.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { getPublicMatches, getMatchQRCode, joinMatchReferee, previewMatchInvite, previewMatchReferee, startMatch } from '@/api/match.js'
import { renderLocalQRCode } from '@/utils/local-qrcode.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'
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
import { buildFinishedMatchDetailRoute, buildOngoingMatchCard } from '@/utils/match-core-flow.js'
import { resolveChallengeQuickCard } from '@/utils/challenge-view.js'
import {
	START_MATCH_MODE_OPTIONS,
	buildStartMatchPayload,
	normalizePendingMatchContext
} from '@/utils/start-match.js'
import { readDefaultGameType, saveDefaultGameType } from '@/utils/game-type-preference.js'
import { scanAndResolveMatchCode } from '@/utils/match-scan.js'
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
const activityStore = useActivityStore()
const { isDarkMode } = usePageTheme()

// ========== 响应式数据 ==========
const lobbyState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const currentMatch = computed(() => (userStore.isLoggedIn ? activityStore.currentMatch : null))
const spectatorMatches = ref([])
const page = ref(1)
const total = ref(0)
const loadingMore = ref(false)
const showGameTypeModal = ref(false)
const selectedGameType = ref(null)
const defaultGameType = ref(0)
const startMatchMode = ref('practice')
const startMatchVisibility = ref('private')
const pendingStartContext = ref(null)
const pendingStartAuthGeneration = ref(-1)
const scanIntent = ref('start')
const showMatchQrModal = ref(false)
const matchQrLoading = ref(false)
const matchQrFailed = ref(false)
const matchQrContent = ref('')
const showRefereePreview = ref(false)
const refereePreviewLoading = ref(false)
const refereeJoining = ref(false)
const refereePreview = ref(null)
const refereePreviewError = ref('')
const pendingRefereeJoin = ref(null)
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
const entryIconColor = computed(() => (isDarkMode.value ? '#D7C89B' : '#6E6242'))

// 用户信息
const userId = computed(() => userStore.userInfo?.id || 0)
const myMatchCard = computed(() => buildOngoingMatchCard(currentMatch.value, userStore.userInfo || {}))
const currentChallenge = computed(() => (userStore.isLoggedIn ? activityStore.currentChallenge : null))
const receivedChallengeCount = computed(() => activityStore.receivedChallengeCount || 0)
const challengeCard = computed(() => {
	const card = resolveChallengeQuickCard(currentChallenge.value, {
		userId: userId.value,
		serverTime: activityStore.challengeServerTime
	})
	if (!card || !myMatchCard.value) return card
	// 约球已开局时由上方「我的对局」卡承接，不重复展示
	const sameMatch = card.matchId > 0 && card.matchId === myMatchCard.value.id
	const sameChallenge = card.challengeId > 0 && card.challengeId === Number(currentMatch.value?.challenge_id || 0)
	return sameMatch || sameChallenge ? null : card
})
// 「我的对局」已在顶部展示，列表中不再重复出现
const lobbyItems = computed(() => {
	const currentId = myMatchCard.value?.id || 0
	return spectatorMatches.value
		.filter((match) => !currentId || Number(match.id) !== currentId)
		.map((match) => normalizeSpectatorMatch(match, { userId: userId.value }))
})
const lobbySections = computed(() => buildSpectatorSections(lobbyItems.value, lobbyTab.value))

const openChallengeWaiting = () => {
	if (!challengeCard.value) return
	uni.navigateTo({ url: challengeCard.value.url })
}

const goChallengeCompose = () => {
	uni.navigateTo({ url: '/subPages/match/challengeCompose' })
}

const goChallenges = (tab) => {
	// 明确指定目标 Tab：本人发出的邀请在「发出」，历史在「记录」，避免被默认 Tab 过滤。
	const query = tab ? `?tab=${tab}` : ''
	uni.navigateTo({ url: `/subPages/social/challenges${query}` })
}
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
onMounted(() => {
	loadData()
})

// 下拉刷新
onPullDownRefresh(async () => {
	try {
		await loadData({ forceActivity: true })
	} finally {
		uni.stopPullDownRefresh()
	}
})

onReachBottom(() => {
	loadMore()
})

onShow(() => {
	consumePendingMatchContext()
	// 退出登录后「好友」标签不再可用，回到公开大厅
	if (!userStore.isLoggedIn && lobbyTab.value === 'friends') {
		lobbyTab.value = 'live'
		reloadLobby()
		return
	}
	if (!hasLoadedOnce.value) {
		loadData()
		return
	}
	// 已挂载过：由 Store 的 dirty/TTL 决定是否回源，避免每次切页都拉取。
	if (userStore.isLoggedIn && activityStore.shouldFetch()) {
		activityStore.fetch({
			userId: userStore.userId,
			authGeneration: userStore.authGeneration
		}, { force: true, silent: true }).then(() => loadData()).catch(() => loadData())
	}
})

// 赛果页「再来一局」上下文：切回对局页后恢复设置并直接进入扫码。
const consumePendingMatchContext = () => {
	const raw = uni.getStorageSync('pending_match_rematch')
	if (!raw) return
	const result = normalizePendingMatchContext('pending_match_rematch', raw)
	uni.removeStorageSync('pending_match_rematch')
	if (!result.valid) {
		uni.showToast({ title: result.message, icon: 'none' })
		return
	}
	pendingStartContext.value = result.context
	pendingStartAuthGeneration.value = userStore.authGeneration
	selectedGameType.value = result.context.game_type
	startMatchMode.value = result.context.match_mode
	startMatchVisibility.value = result.context.visibility
	scanIntent.value = 'start'
	setTimeout(() => {
		handleScanCode()
	}, 80)
}

const retryChallengeActivity = () => {
	if (!userStore.isLoggedIn) return
	activityStore.fetch({
		userId: userStore.userId,
		authGeneration: userStore.authGeneration
	}, { force: true, silent: true })
}


const clearPendingStartContext = () => {
	pendingStartContext.value = null
	pendingStartAuthGeneration.value = -1
	uni.removeStorageSync('pending_match_rematch')
}

// ========== 方法 ==========

/**
 * 加载数据
 */
const hasLoadedOnce = ref(false)

const loadData = async ({ forceActivity = false } = {}) => {
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
		const publicMatchesRequest = getPublicMatches(buildSpectatorMatchListParams({
			...lobbyQuery.value,
			gameType: currentGameType.value,
			page: 1,
			pageSize: PAGE_SIZE
		}))

		const activityRequest = userStore.isLoggedIn
			? activityStore.fetch({
				userId: userStore.userId,
				authGeneration: userStore.authGeneration
			}, { force: forceActivity, silent: true })
			: Promise.resolve(null)
		const [matchResult] = await Promise.all([
			publicMatchesRequest.then(
				value => ({ value }),
				error => ({ error })
			),
			activityRequest.catch(() => null)
		])
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

const retryLobby = () => loadData({ forceActivity: true })

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

onUnmounted(() => {
	if (showMatchQrModal.value || showRefereePreview.value) {
		uni.showTabBar({ animation: false })
	}
})

/**
 * 开始对战 - 打开选择比赛类型弹框
 */
const handleStartMatch = () => {
	// 检查登录状态
	if (!userStore.isLoggedIn) {
		uni.showToast({
			title: '请先登录后再发起PK',
			icon: 'none',
			duration: 1500
		})
		setTimeout(() => {
			uni.navigateTo({
				url: '/pages/login/login'
			})
		}, 1500)
		return
	}
	uni.showActionSheet({
		itemList: ['扫描对手二维码', '出示我的二维码', '选择好友或最近对手'],
		success: ({ tapIndex }) => {
			if (tapIndex === 0) {
				scanIntent.value = 'start'
				defaultGameType.value = readDefaultGameType(uni, userStore.userId)
				showGameTypeModal.value = true
				return
			}
			if (tapIndex === 1) {
				showMyMatchQRCode()
				return
			}
			uni.navigateTo({ url: '/subPages/match/opponentSelector' })
		}
	})
}

const handlePrimaryAction = () => {
	if (myMatchCard.value && currentMatch.value) {
		handleContinueMatch(currentMatch.value)
		return
	}
	handleStartMatch()
}

const showMyMatchQRCode = async () => {
	showMatchQrModal.value = true
	uni.hideTabBar({ animation: true })
	await loadMyMatchQRCode()
}

const loadMyMatchQRCode = async () => {
	matchQrLoading.value = true
	matchQrFailed.value = false
	matchQrContent.value = ''
	try {
		const res = await getMatchQRCode()
		if (!res?.success || !res.qrcode_data) {
			matchQrLoading.value = false
			matchQrFailed.value = true
			return
		}
		matchQrContent.value = res.qrcode_data
		await nextTick()
		renderLocalQRCode({
			canvasId: 'match-page-invite-qrcode',
			content: matchQrContent.value,
			size: 240,
			uniApi: uni
		})
		matchQrLoading.value = false
	} catch (error) {
		matchQrLoading.value = false
		matchQrFailed.value = true
	}
}

const closeMatchQrModal = () => {
	showMatchQrModal.value = false
	matchQrLoading.value = false
	matchQrFailed.value = false
	matchQrContent.value = ''
	uni.showTabBar({ animation: true })
}

const getRefereePreviewMode = (mode) => mode === 'ranked' ? '排位赛' : '练习赛'

const openRefereePreview = async (refereeJoin) => {
	pendingRefereeJoin.value = refereeJoin
	refereePreview.value = null
	refereePreviewError.value = ''
	refereePreviewLoading.value = true
	showRefereePreview.value = true
	uni.hideTabBar({ animation: true })
	try {
		const res = await previewMatchReferee(refereeJoin)
		if (!res?.success || !res.preview) {
			refereePreviewError.value = res?.message || '裁判码已失效，请重新扫码'
			return
		}
		refereePreview.value = res.preview
	} catch (error) {
		console.error('预览裁判对局失败:', error)
		refereePreviewError.value = '裁判码已失效，请重新扫码'
	} finally {
		refereePreviewLoading.value = false
	}
}

const closeRefereePreview = (force = false) => {
	if (refereeJoining.value && !force) return
	showRefereePreview.value = false
	refereePreviewLoading.value = false
	refereePreview.value = null
	refereePreviewError.value = ''
	pendingRefereeJoin.value = null
	uni.showTabBar({ animation: true })
}

const retryRefereeScan = () => {
	closeRefereePreview()
	scanIntent.value = 'referee'
	setTimeout(handleScanCode, 50)
}

const confirmRefereeJoin = async () => {
	if (!pendingRefereeJoin.value || refereeJoining.value) return
	refereeJoining.value = true
	try {
		const res = await joinMatchReferee(pendingRefereeJoin.value)
		if (!res?.success) {
			refereePreviewError.value = res?.message || '裁判码已失效，请重新扫码'
			refereePreview.value = null
			return
		}
		const joinedMatchId = res.match_id || pendingRefereeJoin.value.match_id
		closeRefereePreview(true)
		navigateToPlayingMatch({
			match_id: joinedMatchId,
			game_type: res.match?.game_type || 3,
			opponent_name: res.match?.opponent_name || '选手2',
			opponent_avatar: res.match?.opponent_avatar || ''
		})
	} catch (error) {
		console.error('确认担任裁判失败:', error)
		refereePreviewError.value = '加入裁判失败，请重新扫码'
		refereePreview.value = null
	} finally {
		refereeJoining.value = false
	}
}

const handleScanAsReferee = () => {
	if (!userStore.isLoggedIn) {
		uni.showToast({
			title: '请先登录后再扫码担任裁判',
			icon: 'none',
			duration: 1500
		})
		return
	}
	scanIntent.value = 'referee'
	handleScanCode()
}

/**
 * 处理比赛类型确认
 */
const handleGameTypeConfirm = ({ gameType, setAsDefault } = {}) => {
	scanIntent.value = 'start'
	if (setAsDefault) {
		defaultGameType.value = saveDefaultGameType(uni, userStore.userId, gameType)
	}
	selectedGameType.value = gameType
	uni.showActionSheet({
		itemList: START_MATCH_MODE_OPTIONS.map((item) => item.label),
		success: ({ tapIndex }) => {
			const option = START_MATCH_MODE_OPTIONS[tapIndex]
			if (!option) return
			startMatchMode.value = option.matchMode
			startMatchVisibility.value = option.visibility
			handleScanCode()
		}
	})
}

/**
 * 开始扫码
 */
const handleScanCode = async () => {
	const scanAction = await scanAndResolveMatchCode({
		uniApi: uni,
		previewMatchInvite
	})
	if (scanAction.type === 'cancelled') {
		clearPendingStartContext()
		return
	}
	if (scanAction.type === 'error') {
		clearPendingStartContext()
		uni.showToast({ title: scanAction.message || '扫码失败', icon: 'none' })
		return
	}
	await handleMatchResult(scanAction)
}

/**
 * 处理匹配结果
 */
const handleMatchResult = async (scanAction) => {
	try {
		if (pendingStartContext.value && pendingStartAuthGeneration.value !== userStore.authGeneration) {
			clearPendingStartContext()
			uni.showToast({ title: '登录状态已变更，请重新发起扫码', icon: 'none' })
			return
		}
		if (scanAction.type === 'join_referee') {
			clearPendingStartContext()
			await openRefereePreview(scanAction.refereeJoin)
			return
		}

		if (scanIntent.value === 'referee') {
			clearPendingStartContext()
			uni.showToast({ title: '请扫描对局裁判码', icon: 'none' })
			return
		}

		const opponentData = scanAction.opponent
		// 重赛上下文锁定对手；约球比赛不再走扫码开局通道。
		const expectedOpponentId = Number(pendingStartContext.value?.opponent_id || 0)
		const scannedOpponentId = Number(opponentData?.id || opponentData?.user_id || opponentData?.opponent_id || 0)
		if (expectedOpponentId > 0 && scannedOpponentId > 0 && scannedOpponentId !== expectedOpponentId) {
			clearPendingStartContext()
			uni.showToast({ title: '请扫描指定对手的匹配码', icon: 'none' })
			return
		}

		// 显示加载中
		uni.showLoading({ title: '匹配中...', mask: true })

		// 创建对局
		const res = await startMatch(buildStartMatchPayload({
			gameType: pendingStartContext.value?.game_type || selectedGameType.value,
			opponent: opponentData,
			matchMode: startMatchMode.value,
			visibility: startMatchVisibility.value,
			inviteToken: scanAction.inviteToken
		}))

		// 隐藏加载
		uni.hideLoading()

		handleStartMatchOutcome(resolveStartMatchGuardAction({
			response: res,
			selectedGameType: selectedGameType.value,
			scannedOpponent: opponentData
		}))
		clearPendingStartContext()
	} catch (error) {
		clearPendingStartContext()
		uni.hideLoading()
		console.error('处理匹配结果失败:', error)
		uni.showToast({ title: '匹配失败', icon: 'none' })
	}
}

const navigateToPlayingMatch = (match) => {
	uni.navigateTo({
		url: buildPlayingRoute(match)
	})
}

const handleStartMatchOutcome = (outcome) => {
	if (outcome.type === 'navigate' || outcome.type === 'resume') {
		navigateToPlayingMatch(outcome.match)
		return
	}

	if (outcome.type === 'prompt_self_ongoing') {
		uni.showModal({
			title: '你有未结束的对局',
			content: outcome.message,
			confirmText: '进入该对局',
			cancelText: '稍后处理',
			success: ({ confirm }) => {
				if (confirm) {
					navigateToPlayingMatch(outcome.match)
				}
			}
		})
		return
	}

	uni.showToast({
		title: outcome.message || '创建对局失败',
		icon: 'none'
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
