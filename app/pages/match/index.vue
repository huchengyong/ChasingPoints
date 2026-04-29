<template>
	<view class="match-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 主内容区 -->
		<view class="main-content">
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
						<uni-icons type="tune-filled" size="20" :color="isSpectatorFilterActive ? '#ffffff' : (isDarkMode ? '#d7c89b' : '#64748b')"></uni-icons>
					</button>
				</view>
			</view>

			<!-- 加载状态 -->
			<view v-if="showPageLoading" class="loading-container">
					<uni-icons type="spinner-cycle" size="48" color="#E0AE12"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 空数据状态 -->
			<view v-else-if="!visibleCurrentMatch && spectatorMatches.length === 0" class="empty-state">
				<uni-icons type="medal" size="128" color="#6b7280" class="empty-icon"></uni-icons>
				<text class="empty-title">{{ emptyState.title }}</text>
				<text class="empty-subtitle">{{ emptyState.subtitle }}</text>
				<button class="start-button" @click="handleStartMatch">
					<text>发起PK</text>
				</button>
				<button class="secondary-link" @click="handleScanAsReferee">
					<text>专业比赛需要裁判？扫码进入</text>
				</button>
			</view>

			<!-- 对局列表 -->
			<view v-else class="match-list">
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
								<image :src="userAvatar || '/static/images/default-avatar.png'" mode="aspectFill" />
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
								<image :src="visibleCurrentMatch.opponent_avatar || '/static/images/default-avatar.png'" mode="aspectFill" />
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
								<image :src="match.player1_avatar || '/static/images/default-avatar.png'" mode="aspectFill" />
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
								<image :src="match.player2_avatar || '/static/images/default-avatar.png'" mode="aspectFill" />
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
							<uni-icons type="circle" size="14" :color="match.status === 2 ? '#94a3b8' : '#22c55e'"></uni-icons>
							<text>{{ getStatusText(match) }}</text>
						</view>
					</view>
				</view>
			</view>
		</view>

		<!-- 选择比赛类型弹框（用于开始对战） -->
		<gameTypeModal
			v-model:visible="showGameTypeModal"
			@confirm="handleGameTypeConfirm"
		/>

		<view v-if="showFilterPanel" class="filter-panel-mask" @click="closeFilterPanel">
			<view class="filter-panel" @click.stop>
				<view class="filter-panel-header">
					<text class="filter-panel-title">筛选对局</text>
					<button class="filter-close" @click="closeFilterPanel">
						<uni-icons type="closeempty" size="20" :color="isDarkMode ? '#d7c89b' : '#64748b'"></uni-icons>
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
	</view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

import { onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { getCurrentMatch, getPublicMatches, joinMatchReferee, startMatch } from '@/api/match.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { shouldShowMatchPageLoading } from '@/utils/match-page.js'
import { buildPlayingRoute, resolveMatchScanAction, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'
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

// ========== 状态管理 ==========
const userStore = useUserStore()
const { isDarkMode } = usePageTheme()

// ========== 响应式数据 ==========
const loading = ref(false)
const refreshing = ref(false)
const currentMatch = ref(null)
const spectatorMatches = ref([])
const showGameTypeModal = ref(false)
const selectedGameType = ref(null)
const scanIntent = ref('start')
const showFilterPanel = ref(false)
const currentScope = ref('hall')
const currentStatus = ref(1)
const currentGameType = ref(0)
const draftStatus = ref(1)
const draftGameType = ref(0)
const showPageLoading = computed(() => shouldShowMatchPageLoading(loading.value, refreshing.value))
const scopeOptions = SPECTATOR_SCOPES
const statusOptions = SPECTATOR_STATUS_OPTIONS
const gameTypeOptions = GAME_TYPE_FILTER_OPTIONS_WITH_ALL
const isSpectatorFilterActive = computed(() => {
	return currentGameType.value > 0 || (currentScope.value === 'hall' && currentStatus.value !== 1)
})

// 用户信息
const userName = computed(() => userStore.userInfo?.nickname || '我')
const userAvatar = computed(() => userStore.userInfo?.avatar || '')
const userId = computed(() => userStore.userInfo?.id || 0)
const visibleCurrentMatch = computed(() => {
	if (currentScope.value !== 'hall' || currentStatus.value !== 1 || !currentMatch.value) return null
	if (currentGameType.value > 0 && Number(currentMatch.value.game_type) !== Number(currentGameType.value)) return null
	return currentMatch.value
})
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

const filteredSpectatorList = (list = []) => {
	if (!visibleCurrentMatch.value) return list
	return list.filter(match => match.id !== visibleCurrentMatch.value.id)
}

// ========== 生命周期 ==========
onMounted(() => {
	loadData()
})

// 下拉刷新
onPullDownRefresh(async () => {
	refreshing.value = true
	try {
		await loadData()
	} finally {
		refreshing.value = false
		uni.stopPullDownRefresh()
	}
})

onShow(() => {
	loadData()
})

// ========== 方法 ==========

/**
 * 加载数据
 */
const loadData = async () => {
	if (loading.value) return

	loading.value = true
	try {
		// 并行请求：如果已登录则获取自己进行中的对局，同时获取平台所有正在进行的对局
		const requests = [
			getPublicMatches(buildSpectatorMatchListParams({
				scope: currentScope.value,
				status: currentStatus.value,
				gameType: currentGameType.value,
				page: 1,
				pageSize: 20
			})).catch(() => ({ success: false, list: [] }))
		]

		// 如果已登录，也获取自己进行中的对局
		if (userStore.isLoggedIn) {
			requests.unshift(getCurrentMatch().catch(() => ({ success: false })))
		}

		const results = await Promise.all(requests)

		if (userStore.isLoggedIn) {
			// 设置我进行中的对局
			const currentRes = results[0]
			if (currentRes.success && currentRes.match) {
				currentMatch.value = currentRes.match
			} else {
				currentMatch.value = null
			}

			// 设置公开观赛对局列表
			const matchListRes = results[1]
			if (matchListRes.success && matchListRes.list) {
				spectatorMatches.value = filteredSpectatorList(matchListRes.list)
			} else {
				spectatorMatches.value = []
			}
		} else {
			currentMatch.value = null
			// 设置公开观赛对局列表
			const matchListRes = results[0]
			if (matchListRes.success && matchListRes.list) {
				spectatorMatches.value = matchListRes.list
			} else {
				spectatorMatches.value = []
			}
		}
	} catch (error) {
		console.error('加载对局数据失败:', error)
	} finally {
		loading.value = false
	}
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
	if (showFilterPanel.value) {
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
	scanIntent.value = 'start'
	showGameTypeModal.value = true
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
const handleGameTypeConfirm = (gameType) => {
	scanIntent.value = 'start'
	selectedGameType.value = gameType
	// 开始扫码
	handleScanCode()
}

/**
 * 开始扫码
 */
const handleScanCode = () => {
	// #ifdef APP-PLUS || APP-HARMONY
	uni.scanCode({
		scanType: ['qrCode'],
		success: (res) => {
			console.log('扫码结果:', res.result)
			handleMatchResult(res.result)
		},
		fail: (err) => {
			console.error('扫码失败:', err)
		}
	})
	// #endif
}

/**
 * 处理匹配结果
 */
const handleMatchResult = async (scanResult) => {
	try {
		const scanAction = resolveMatchScanAction(scanResult)
		if (scanAction.type === 'error') {
			uni.showToast({ title: scanAction.message || '无效的二维码', icon: 'none' })
			return
		}

		if (scanAction.type === 'join_referee') {
			uni.showLoading({ title: '加入裁判中...', mask: true })
			const res = await joinMatchReferee(scanAction.refereeJoin)
			uni.hideLoading()
			if (!res?.success) {
				uni.showToast({ title: res?.message || '加入裁判失败', icon: 'none' })
				return
			}
			navigateToPlayingMatch({
				match_id: res.match_id || scanAction.refereeJoin.match_id,
				game_type: res.match?.game_type || 3,
				opponent_name: res.match?.opponent_name || '对手',
				opponent_avatar: res.match?.opponent_avatar || ''
			})
			return
		}

		if (scanIntent.value === 'referee') {
			uni.showToast({ title: '请扫描对局裁判码', icon: 'none' })
			return
		}

		const opponentData = scanAction.opponent

		// 显示加载中
		uni.showLoading({ title: '匹配中...', mask: true })

		// 创建对局
		const res = await startMatch({
			game_type: selectedGameType.value,
			opponent_id: opponentData.user_id,
			opponent_name: opponentData.nickname || '对手'
		})

		// 隐藏加载
		uni.hideLoading()

		handleStartMatchOutcome(resolveStartMatchGuardAction({
			response: res,
			selectedGameType: selectedGameType.value,
			scannedOpponent: opponentData
		}))
	} catch (error) {
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
	const finishedDetailUrl = buildSpectatorFinishedMatchDetailUrl({
		match,
		userId: userId.value
	})
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
