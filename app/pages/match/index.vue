<template>
	<view class="match-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 主内容区 -->
		<view class="main-content">
			<view class="match-primary-action" v-if="lobbyState.status !== ASYNC_PAGE_STATUS.EMPTY">
				<view class="match-primary-action__copy">
					<text class="match-primary-action__eyebrow">双人对局</text>
					<text class="match-primary-action__title">{{ homePrimaryAction.label }}</text>
					<text class="match-primary-action__desc">{{ homePrimaryAction.type === 'start' ? '扫描、出示二维码或从邀约继续' : '你的待处理对局始终显示在这里' }}</text>
				</view>
				<button class="match-primary-action__button" @click="handlePrimaryAction">{{ homePrimaryAction.label }}</button>
			</view>
			<view class="lobby-toolbar">
				<view class="lobby-toolbar-row">
					<view class="scope-tabs">
						<button
							v-for="item in scopeOptions"
							:key="item.value"
							class="scope-tab"
							:class="{ active: currentScope === item.value }"
							@click="handleScopeChange(item.value)"
						>
							<text>{{ item.label }}</text>
						</button>
					</view>
					<button class="filter-button" :class="{ active: isSpectatorFilterActive }" @click="openFilterPanel">
						<uni-icons type="tune-filled" size="20" :color="isSpectatorFilterActive ? '#ffffff' : (isDarkMode ? '#d7c89b' : '#6E6242')"></uni-icons>
					</button>
				</view>
			</view>

			<!-- 加载状态 -->
			<view v-if="showPageLoading" class="loading-container">
					<uni-icons type="spinner-cycle" size="48" color="#E0AE12"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<view v-else-if="pageError" class="empty-state error-state">
				<uni-icons type="info" size="72" color="#9A8C67" class="empty-icon"></uni-icons>
				<text class="empty-title">{{ pageError.title }}</text>
				<text class="empty-subtitle">{{ pageError.description }}</text>
				<button class="start-button" @click="handlePageErrorAction">{{ pageError.actionText }}</button>
			</view>

			<!-- 空数据状态 -->
			<view v-else-if="lobbyState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
				<uni-icons type="medal" size="128" color="#6E6242" class="empty-icon"></uni-icons>
				<text class="empty-title">{{ emptyState.title }}</text>
				<text class="empty-subtitle">{{ emptyState.subtitle }}</text>
				<button class="start-button" @click="handleStartMatch">
					<text>发起好友对局</text>
				</button>
				<button class="secondary-link" @click="handleScanAsReferee">
					<text>专业比赛需要裁判？扫码进入</text>
					<uni-icons type="arrowright" size="16" :color="isDarkMode ? '#9f926e' : '#6E6242'"></uni-icons>
				</button>
			</view>

			<!-- 对局列表 -->
			<view v-else class="match-list">
				<view v-if="refreshError" class="refresh-error-banner">
					<text>{{ refreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryLobby">重试</text>
				</view>
				<!-- 进行中的对局 -->
				<view v-if="visibleCurrentMatch" class="match-card my-match" @click="handleContinueMatch(visibleCurrentMatch)">
					<!-- MY MATCH 标签 -->
					<view class="my-match-badge">我的对局</view>
					<view v-if="visibleCurrentMatch.viewer_role === 'referee'" class="referee-match-info">
						<text class="referee-match-info__title">你正在担任本场裁判</text>
						<text class="referee-match-info__score">{{ visibleCurrentMatch.my_score }} : {{ visibleCurrentMatch.opponent_score }}</text>
						<text class="referee-match-info__hint">点击进入裁判记分页</text>
					</view>
					<view v-else class="match-info">
						<!-- 我方玩家 -->
						<view class="player">
							<view class="avatar me-avatar" :class="{ winner: visibleCurrentMatch.my_score > visibleCurrentMatch.opponent_score }">
								<image :src="userAvatar" mode="aspectFill" />
								<view class="me-badge">我</view>
							</view>
							<text class="name">{{ userName }}</text>
						</view>

						<!-- 比分 -->
						<view class="score-area">
							<text class="score">{{ visibleCurrentMatch.my_score }} : {{ visibleCurrentMatch.opponent_score }}</text>
							<text class="vs-text">VS</text>
						</view>

						<!-- 对手玩家 -->
						<view class="player">
							<view class="avatar" :class="{ winner: visibleCurrentMatch.opponent_score > visibleCurrentMatch.my_score }">
								<image :src="getMatchAvatar(visibleCurrentMatch, 'opponent')" mode="aspectFill" />
							</view>
							<text class="name">{{ visibleCurrentMatch.opponent_name }}</text>
						</view>
					</view>

					<!-- 底部信息 -->
					<view class="match-footer">
						<view :class="['game-type-tag', getGameTypeClass(visibleCurrentMatch.game_type)]">
							{{ visibleCurrentMatch.game_type_name }}
						</view>
						<view class="match-status">
							<uni-icons type="circle" size="14" color="#22c55e"></uni-icons>
							<text>{{ getStatusText(visibleCurrentMatch) }}</text>
						</view>
					</view>
				</view>

				<!-- 公开观赛对局列表 -->
				<view
					v-for="match in spectatorMatches"
					:key="match.id"
					:class="['match-card', { 'my-match': isMyMatch(match) }]"
					@click="handleMatchClick(match)"
				>
					<!-- 我的对局标签 -->
					<view v-if="isMyMatch(match)" class="my-match-badge">我的对局</view>
					<view class="match-info">
						<!-- 玩家1 -->
						<view class="player">
							<view
								:class="['avatar', { winner: match.player1_score > match.player2_score, 'me-avatar': isPlayer1Me(match) }]"
							>
								<image :src="getMatchAvatar(match, 'player1')" mode="aspectFill" />
								<view v-if="isPlayer1Me(match)" class="me-badge">我</view>
							</view>
							<text class="name">{{ match.player1_name }}</text>
						</view>

						<!-- 比分 -->
						<view class="score-area">
							<text class="score">{{ match.player1_score }} : {{ match.player2_score }}</text>
							<text class="vs-text">VS</text>
						</view>

						<!-- 玩家2 -->
						<view class="player">
							<view
								:class="['avatar', { winner: match.player2_score > match.player1_score, 'me-avatar': isPlayer2Me(match) }]"
							>
								<image :src="getMatchAvatar(match, 'player2')" mode="aspectFill" />
								<view v-if="isPlayer2Me(match)" class="me-badge">我</view>
							</view>
							<text class="name">{{ match.player2_name }}</text>
						</view>
					</view>

					<!-- 底部信息 -->
					<view class="match-footer">
						<view :class="['game-type-tag', getGameTypeClass(match.game_type)]">
							{{ match.game_type_name }}
						</view>
						<view class="match-status" :class="{ finished: match.status === 2 }">
							<uni-icons type="circle" size="14" :color="match.status === 2 ? '#9A8C67' : '#22c55e'"></uni-icons>
							<text>{{ getStatusText(match) }}</text>
						</view>
					</view>
				</view>
			</view>
		</view>

		<!-- 选择比赛类型弹框（用于开始对战） -->
		<gameTypeModal
			v-model:visible="showGameTypeModal"
			:default-type="defaultGameType"
			@confirm="handleGameTypeConfirm"
		/>

		<view v-if="showFilterPanel" class="filter-panel-mask" @click="closeFilterPanel">
			<view class="filter-panel" @click.stop>
				<view class="filter-panel-header">
					<text class="filter-panel-title">筛选对局</text>
					<button class="filter-close" @click="closeFilterPanel">
						<uni-icons type="closeempty" size="20" :color="isDarkMode ? '#d7c89b' : '#6E6242'"></uni-icons>
					</button>
				</view>

				<view v-if="currentScope === 'hall'" class="filter-group">
					<text class="filter-group-title">状态</text>
					<view class="status-segmented">
						<button
							v-for="item in statusOptions"
							:key="item.value"
							class="status-segmented-item"
							:class="{ active: draftStatus === item.value }"
							@click="draftStatus = item.value"
						>
							<text>{{ item.label }}</text>
						</button>
					</view>
				</view>

				<view class="filter-group">
					<text class="filter-group-title">球种</text>
					<view class="game-type-choice-list">
						<view
							v-for="item in gameTypeOptions"
							:key="item.value"
							class="game-type-choice"
							:class="{ active: draftGameType === item.value }"
							@click="draftGameType = item.value"
						>
							<text class="game-type-choice-label">{{ item.label }}</text>
							<uni-icons
								v-if="draftGameType === item.value"
								type="checkmarkempty"
								size="18"
								color="#E0AE12"
							></uni-icons>
						</view>
					</view>
				</view>

				<view class="filter-actions">
					<button class="filter-cancel" @click="closeFilterPanel">
						<text>取消</text>
					</button>
					<button class="filter-confirm" @click="confirmSpectatorFilters">
						<text>确定</text>
					</button>
				</view>
			</view>
		</view>

		<view v-if="showMatchQrModal" class="match-qr-mask" @click="closeMatchQrModal">
			<view class="match-qr-panel" @click.stop>
				<view class="match-qr-header">
					<view>
						<text class="match-qr-title">出示我的匹配二维码</text>
						<text class="match-qr-subtitle">请让对手使用发起对局扫码入口识别</text>
					</view>
					<button class="match-qr-close" @click="closeMatchQrModal">
						<uni-icons type="closeempty" size="22" color="#6E6242"></uni-icons>
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

import { onLoad, onShow, onPullDownRefresh } from '@dcloudio/uni-app'
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
	SPECTATOR_SCOPES,
	SPECTATOR_STATUS_OPTIONS,
	buildSpectatorFinishedMatchDetailUrl,
	buildSpectatorMatchListParams,
	getSpectatorEmptyState,
	getSpectatorMatchStatusText,
	shouldOpenPlayingForSpectatorMatch
} from '@/utils/spectator-lobby.js'
import { buildFinishedMatchDetailRoute, resolveMatchHomePrimaryAction } from '@/utils/match-core-flow.js'
import {
	buildStartMatchPayload,
	normalizePendingMatchContext,
	validateScannedOpponentForContext
} from '@/utils/start-match.js'
import { readDefaultGameType, saveDefaultGameType } from '@/utils/game-type-preference.js'
import { scanAndResolveMatchCode } from '@/utils/match-scan.js'
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
const showFilterPanel = ref(false)
const currentScope = ref(userStore.isLoggedIn ? 'friends' : 'hall')
const currentStatus = ref(1)
const currentGameType = ref(0)
const draftStatus = ref(1)
const draftGameType = ref(0)
const showPageLoading = computed(() => lobbyState.value.status === ASYNC_PAGE_STATUS.LOADING)
const pageError = computed(() => {
	if (lobbyState.value.status !== ASYNC_PAGE_STATUS.ERROR) return null
	return resolveAsyncPageErrorFeedback(lobbyState.value.error, { resource: '对局大厅' })
})
const refreshError = computed(() => (
	lobbyState.value.refreshError
		? resolveAsyncPageErrorFeedback(lobbyState.value.refreshError, { resource: '对局大厅' })
		: null
))
const scopeOptions = SPECTATOR_SCOPES
const statusOptions = SPECTATOR_STATUS_OPTIONS
const gameTypeOptions = GAME_TYPE_FILTER_OPTIONS_WITH_ALL
const isSpectatorFilterActive = computed(() => {
	return currentGameType.value > 0 || (currentScope.value === 'hall' && currentStatus.value !== 1)
})

// 用户信息
const userName = computed(() => userStore.userInfo?.nickname || '我')
const userId = computed(() => userStore.userInfo?.id || 0)
const userAvatar = computed(() => resolveAvatarUrl(userStore.userInfo?.avatar, userId.value))
const visibleCurrentMatch = computed(() => {
	if (!currentMatch.value) return null
	if (Number(currentMatch.value.status) === 1 || currentMatch.value.finish_state === 'pending_confirmation') return currentMatch.value
	if (currentScope.value !== 'hall' || currentStatus.value !== 1) return null
	if (currentGameType.value > 0 && Number(currentMatch.value.game_type) !== Number(currentGameType.value)) return null
	return currentMatch.value
})
const homePrimaryAction = computed(() => resolveMatchHomePrimaryAction({ currentMatch: currentMatch.value }))
const emptyState = computed(() => getSpectatorEmptyState({
	scope: currentScope.value,
	status: currentStatus.value,
	gameType: currentGameType.value
}))

/**
 * 判断是否是当前用户参与的对局
 */
const isMyMatch = (match) => {
	if (!userStore.isLoggedIn || !userId.value) return false
	// CurrentMatchInfo 结构：必然是当前用户的对局
	if (match.player1_id === undefined && match.player2_id === undefined) {
		return match.opponent_id !== undefined
	}
	return match.player1_id === userId.value || match.player2_id === userId.value
}

/**
 * 判断玩家1是否是当前用户
 */
const isPlayer1Me = (match) => {
	if (!userStore.isLoggedIn || !userId.value) return false
	return match.player1_id === userId.value
}

/**
 * 判断玩家2是否是当前用户
 */
const isPlayer2Me = (match) => {
	if (!userStore.isLoggedIn || !userId.value) return false
	return match.player2_id === userId.value
}

const getMatchAvatar = (match = {}, side) => {
	if (side === 'opponent') {
		if (match.opponent_id && match.opponent_avatar) return resolveAvatarUrl(match.opponent_avatar, match.opponent_id)
		if (match.player1_id === userId.value) return resolveAvatarUrl(match.player2_avatar, match.player2_id)
		if (match.player2_id === userId.value) return resolveAvatarUrl(match.player1_avatar, match.player1_id)
		return resolveAvatarUrl(match.opponent_avatar, match.opponent_id)
	}
	const idKey = side === 'player1' ? 'player1_id' : 'player2_id'
	const avatarKey = side === 'player1' ? 'player1_avatar' : 'player2_avatar'
	return resolveAvatarUrl(match[avatarKey], match[idKey])
}

const filteredSpectatorList = (list = []) => {
	if (!visibleCurrentMatch.value) return list
	return list.filter(match => match.id !== visibleCurrentMatch.value.id)
}

// ========== 生命周期 ==========
onMounted(() => {
	loadData()
})

onLoad(() => {
	consumePendingChallengeContext()
})

// 下拉刷新
onPullDownRefresh(async () => {
	try {
		await loadData({ forceActivity: true })
	} finally {
		uni.stopPullDownRefresh()
	}
})

onShow(() => {
	consumePendingChallengeContext()
	if (!hasLoadedOnce.value) loadData()
})

const consumePendingChallengeContext = () => {
	for (const storageKey of ['pending_match_challenge', 'pending_match_rematch']) {
		const raw = uni.getStorageSync(storageKey)
		if (!raw) continue
		const result = normalizePendingMatchContext(storageKey, raw)
		uni.removeStorageSync(storageKey)
		if (!result.valid) {
			uni.showToast({ title: result.message, icon: 'none' })
			continue
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
		return
	}
}

const clearPendingStartContext = () => {
	pendingStartContext.value = null
	pendingStartAuthGeneration.value = -1
	uni.removeStorageSync('pending_match_challenge')
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
			scope: currentScope.value,
			status: currentStatus.value,
			gameType: currentGameType.value,
			page: 1,
			pageSize: 20
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
		if (lobbyState.value.authGeneration !== request.authGeneration || lobbyState.value.requestId !== request.requestId) return
		if (matchResult.error || !matchResult.value?.success) {
			const error = matchResult.error || createRequestError({
				message: matchResult.value?.message || '加载对局大厅失败',
				category: 'business'
			})
			if (currentMatch.value) {
				lobbyState.value = resolveAsyncPageLoad(lobbyState.value, request, {
					data: [currentMatch.value]
				})
			}
			lobbyState.value = rejectAsyncPageLoad(lobbyState.value, request, error)
			return
		}

		const list = matchResult.value.list || []
		spectatorMatches.value = userStore.isLoggedIn
			? filteredSpectatorList(list)
			: list
		lobbyState.value = resolveAsyncPageLoad(lobbyState.value, request, {
			data: [currentMatch.value, ...spectatorMatches.value].filter(Boolean)
		})
		hasLoadedOnce.value = true
	} catch (error) {
		lobbyState.value = rejectAsyncPageLoad(lobbyState.value, request, error)
	}
}

const retryLobby = () => loadData({ forceActivity: true })

const handlePageErrorAction = () => {
	if (lobbyState.value.error?.category === 'permission' || lobbyState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryLobby()
}

const handleScopeChange = (scope) => {
	if (currentScope.value === scope) return
	currentScope.value = scope
	showFilterPanel.value = false
	loadData()
}

const openFilterPanel = () => {
	draftStatus.value = currentStatus.value
	draftGameType.value = currentGameType.value
	showFilterPanel.value = true
	uni.hideTabBar({ animation: true })
}

const closeFilterPanel = () => {
	showFilterPanel.value = false
	uni.showTabBar({ animation: true })
}

const confirmSpectatorFilters = () => {
	const nextStatus = currentScope.value === 'hall' ? draftStatus.value : currentStatus.value
	const nextGameType = draftGameType.value
	const changed = currentStatus.value !== nextStatus || currentGameType.value !== nextGameType

	currentStatus.value = nextStatus
	currentGameType.value = nextGameType
	closeFilterPanel()

	if (changed) {
		loadData()
	}
}

onUnmounted(() => {
	if (showFilterPanel.value || showMatchQrModal.value || showRefereePreview.value) {
		uni.showTabBar({ animation: false })
	}
})

/**
 * 获取游戏类型样式类
 */
const getGameTypeClass = (gameType) => {
	switch (gameType) {
		case 1:
			return 'snooker'
		case 2:
		case 4:
			return 'american-9ball'
		case 3:
		default:
			return 'chinese-8ball'
	}
}

const getStatusText = (match) => {
	return getSpectatorMatchStatusText({
		status: match.status || 1,
		durationSeconds: match.duration_seconds || 0
	})
}

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
	if (homePrimaryAction.value.type === 'start') {
		handleStartMatch()
		return
	}
	if (visibleCurrentMatch.value) handleContinueMatch(visibleCurrentMatch.value)
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
		itemList: ['练习赛（默认私密，不影响竞技权益）', '练习赛（公开展示，不影响竞技权益）', '排位赛（公开展示，需结束确认）'],
		success: ({ tapIndex }) => {
			startMatchMode.value = tapIndex === 2 ? 'ranked' : 'practice'
			startMatchVisibility.value = tapIndex === 1 || tapIndex === 2 ? 'public' : 'private'
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
		const scanMessage = validateScannedOpponentForContext(pendingStartContext.value || {}, opponentData)
		if (scanMessage) {
			clearPendingStartContext()
			uni.showToast({ title: scanMessage, icon: 'none' })
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
			challengeId: pendingStartContext.value?.challenge_id || 0,
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
