<template>
	<view class="ranking-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载状态 -->
		<view class="loading-wrapper" v-if="isLoading">
			<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="pageError" class="empty-state error-state">
			<uni-icons type="info" size="72" :color="isDarkMode ? '#d7c89b' : '#9A8C67'" class="empty-icon"></uni-icons>
			<text class="empty-title">{{ pageError.title }}</text>
			<text class="empty-subtitle">{{ pageError.description }}</text>
			<button class="start-button" @click="handlePageErrorAction">
				<text>{{ pageError.actionText }}</text>
			</button>
		</view>

			<template v-else>
				<view v-if="refreshError" class="refresh-error-banner">
					<text>{{ refreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryLeaderboard">重试</text>
				</view>
				<scroll-view class="game-type-tabs" scroll-x :show-scrollbar="false" enable-flex>
					<view
						v-for="item in gameTypeTabs"
						:key="item.value"
					class="game-type-tab"
					:class="{ active: currentGameType === item.value }"
					@tap="handleGameTypeChange(item.value)"
				>
						<text>{{ item.label }}</text>
					</view>
				</scroll-view>

			<!-- 空数据状态 -->
			<view class="empty-state" v-if="isEmpty">
			    <uni-icons type="medal" size="128" :color="isDarkMode ? '#9F926E' : '#9A8C67'" class="empty-icon"></uni-icons>
			    <text class="empty-title">暂无排行数据</text>
			    <text class="empty-subtitle">快去对战提升排名吧！</text>
			    <button class="start-button" @click="handleStartMatch">
			        <text>发起PK</text>
			    </button>
			</view>

			<scroll-view
				class="ranking-scroll"
				scroll-y
				v-else
				:refresher-enabled="true"
				:refresher-triggered="isRefreshing"
				@refresherrefresh="onRefresh"
				@scrolltolower="onLoadMore"
			>

				<!-- 领奖台区域 -->
				<view class="podium-section">
				<!-- 领奖台背景 -->
				<view class="podium-base">
					<view class="podium-block second">
						<text class="podium-rank-name" v-if="podiumSlots[0].user">{{ podiumSlots[0].user.rank_name }}</text>
						<text class="podium-winrate" v-if="podiumSlots[0].user">胜率 {{ podiumSlots[0].user.win_rate }}%</text>
					</view>
					<view class="podium-block first">
						<text class="podium-rank-name" v-if="podiumSlots[1].user">{{ podiumSlots[1].user.rank_name }}</text>
						<text class="podium-winrate" v-if="podiumSlots[1].user">胜率 {{ podiumSlots[1].user.win_rate }}%</text>
					</view>
					<view class="podium-block third">
						<text class="podium-rank-name" v-if="podiumSlots[2].user">{{ podiumSlots[2].user.rank_name }}</text>
						<text class="podium-winrate" v-if="podiumSlots[2].user">胜率 {{ podiumSlots[2].user.win_rate }}%</text>
					</view>
				</view>
				
				<!-- 前三名用户 -->
				<view class="podium-users">
					<!-- 第二名 -->
					<view class="podium-user second" :class="{ 'is-empty': !podiumSlots[0].user }" @click="podiumSlots[0].user && handleUserClick(podiumSlots[0].user)">
						<template v-if="podiumSlots[0].user">
							<view class="avatar-wrapper">
								<image class="avatar" :src="resolveAvatarUrl(podiumSlots[0].user.avatar, podiumSlots[0].user.user_id)" mode="aspectFill"></image>
								<view class="rank-badge silver">2</view>
							</view>
							<text class="nickname">{{ podiumSlots[0].user.nickname || '球手' }}</text>
							<text class="score">{{ podiumSlots[0].user.rank_score }}分</text>
						</template>
					</view>
					
					<!-- 第一名 -->
					<view class="podium-user first" :class="{ 'is-empty': !podiumSlots[1].user }" @click="podiumSlots[1].user && handleUserClick(podiumSlots[1].user)">
						<template v-if="podiumSlots[1].user">
							<view class="crown-icon">
								<uni-icons type="medal-filled" size="48" color="#fbbf24"></uni-icons>
							</view>
							<view class="avatar-wrapper">
								<image class="avatar" :src="resolveAvatarUrl(podiumSlots[1].user.avatar, podiumSlots[1].user.user_id)" mode="aspectFill"></image>
								<view class="rank-badge gold">1</view>
							</view>
							<text class="nickname">{{ podiumSlots[1].user.nickname || '球手' }}</text>
							<text class="score">{{ podiumSlots[1].user.rank_score }}分</text>
						</template>
					</view>
					
					<!-- 第三名 -->
					<view class="podium-user third" :class="{ 'is-empty': !podiumSlots[2].user }" @click="podiumSlots[2].user && handleUserClick(podiumSlots[2].user)">
						<template v-if="podiumSlots[2].user">
							<view class="avatar-wrapper">
								<image class="avatar" :src="resolveAvatarUrl(podiumSlots[2].user.avatar, podiumSlots[2].user.user_id)" mode="aspectFill"></image>
								<view class="rank-badge bronze">3</view>
							</view>
							<text class="nickname">{{ podiumSlots[2].user.nickname || '球手' }}</text>
							<text class="score">{{ podiumSlots[2].user.rank_score }}分</text>
						</template>
					</view>
				</view>
		</view>

			<!-- 排行榜列表 -->
			<view class="ranking-list">
				<view 
					class="ranking-item" 
					v-for="item in rankList" 
					:key="item.user_id"
					@click="handleUserClick(item)"
				>
					<text class="item-rank">{{ item.rank }}</text>
					<view class="item-info">
						<image class="item-avatar" :src="resolveAvatarUrl(item.avatar, item.user_id)" mode="aspectFill"></image>
						<view class="item-details">
							<text class="item-nickname">{{ item.nickname || '球手' }}</text>
							<text class="item-rank-name">{{ item.rank_name }} | {{ item.rank_score }}分</text>
						</view>
					</view>
					<text class="item-winrate">胜率: {{ item.win_rate }}%</text>
				</view>
				</view>
			</scroll-view>
	</template>

			<!-- 我的排名卡片（固定在底部） -->
			<view class="my-ranking-card" v-if="myRanking">
				<text class="my-rank">{{ myRanking.rank > 0 ? myRanking.rank : '-' }}</text>
				<view class="my-info">
					<image class="my-avatar" :src="resolveAvatarUrl(myRanking.avatar, myRanking.user_id)" mode="aspectFill"></image>
					<view class="my-details">
						<text class="my-label">{{ myRanking.rank > 0 ? '我的排名' : '暂未上榜' }}</text>
						<text class="my-rank-name">{{ myRanking.rank > 0 ? myRanking.rank_name : '完成首场该模式对局后入榜' }}</text>
					</view>
				</view>
				<text class="my-score">{{ myRanking.rank > 0 ? `${myRanking.rank_score}分` : '未记录榜单分数' }}</text>
			</view>

		<!-- 选择比赛类型弹框（用于开始对战） -->
		<gameTypeModal
			v-model:visible="showGameTypeModal"
			:default-type="defaultGameType"
			@confirm="handleGameTypeConfirm"
		/>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { usePublicReadStore } from '@/store/publicRead.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { previewMatchInvite, startMatch } from '@/api/match.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'
import { buildStartMatchPayload } from '@/utils/start-match.js'
import { buildLeaderboardPodiumSlots } from '@/utils/ranking-podium.js'
import { readDefaultGameType, saveDefaultGameType } from '@/utils/game-type-preference.js'
import { scanAndResolveMatchCode } from '@/utils/match-scan.js'
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

// ========== 响应式数据 ==========
const userStore = useUserStore()
const publicReadStore = usePublicReadStore()
const leaderboardState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const isEmpty = computed(() => leaderboardState.value.status === ASYNC_PAGE_STATUS.EMPTY)
const isLoading = computed(() => leaderboardState.value.status === ASYNC_PAGE_STATUS.LOADING)
const isRefreshing = computed(() => (
	leaderboardState.value.status === ASYNC_PAGE_STATUS.REFRESHING && !isLoadingMore.value
))
const pageError = computed(() => {
	if (leaderboardState.value.status !== ASYNC_PAGE_STATUS.ERROR) return null
	return resolveAsyncPageErrorFeedback(leaderboardState.value.error, { resource: '排行榜' })
})
const refreshError = computed(() => (
	leaderboardState.value.refreshError
		? resolveAsyncPageErrorFeedback(leaderboardState.value.refreshError, { resource: '排行榜' })
		: null
))
const isLoadingMore = ref(false)
const hasMore = ref(true)

const topThree = ref([])
const myRanking = ref(null)
const rankList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)
const showGameTypeModal = ref(false)
const selectedGameType = ref(null)
const defaultGameType = ref(0)
const currentGameType = ref(3)
const gameTypeTabs = GAME_TYPE_TABS
const podiumSlots = computed(() => buildLeaderboardPodiumSlots(topThree.value))

onLoad((options) => {
	const gameType = Number(options?.game_type || 0)
	if (gameType > 0) {
		currentGameType.value = gameType
	}
})

// ========== 生命周期 ==========
onShow(() => {
	fetchLeaderboard()
})

// ========== 方法 ==========

/**
 * 获取排行榜数据
 */
const fetchLeaderboard = async (isRefresh = false, isLoadMore = false) => {
	if ((isLoading.value && !isRefresh) || isLoadingMore.value || (isLoadMore && !hasMore.value)) return
	const previousPage = currentPage.value
	const previousHasMore = hasMore.value
	const currentGeneration = userStore.authGeneration
	const nextState = beginAsyncPageLoad(leaderboardState.value, {
		authGeneration: currentGeneration,
		emptyData: []
	})
	const request = getAsyncPageRequest(nextState)
	const identityChanged = leaderboardState.value.authGeneration !== currentGeneration
	if (identityChanged) {
		topThree.value = []
		myRanking.value = null
		rankList.value = []
		total.value = 0
		hasMore.value = true
	}
	leaderboardState.value = nextState
	
	if (isRefresh) {
		currentPage.value = 1
		hasMore.value = true
	} else if (isLoadMore) {
		isLoadingMore.value = true
		currentPage.value++
	}
	
	try {
		const res = await publicReadStore.loadLeaderboard({
			game_type: currentGameType.value,
			page: currentPage.value,
			page_size: pageSize
		}, {
			force: isRefresh,
			identity: {
				userId: userStore.userId,
				authGeneration: userStore.authGeneration
			}
		})
		
		if (!res?.success) {
			throw createRequestError({
				message: res?.message || '获取排行榜失败',
				category: 'business'
			})
		}
		if (leaderboardState.value.authGeneration !== request.authGeneration || leaderboardState.value.requestId !== request.requestId) return

		// 前三名只在第一页或刷新时更新
		if (currentPage.value === 1 || isRefresh) {
			topThree.value = res.top_three || []
			myRanking.value = res.my_ranking
		}

		total.value = res.total || 0

		const list = res.list || []
		if (isRefresh || currentPage.value === 1) {
			rankList.value = list
		} else if (isLoadMore) {
			rankList.value = [...rankList.value, ...list]
		} else {
			rankList.value = list
		}

		// 判断是否还有更多
		hasMore.value = (topThree.value.length + rankList.value.length) < total.value
		leaderboardState.value = resolveAsyncPageLoad(leaderboardState.value, request, {
			data: [...topThree.value, ...rankList.value]
		})
	} catch (error) {
		const nextErrorState = rejectAsyncPageLoad(leaderboardState.value, request, error)
		if (nextErrorState !== leaderboardState.value && isLoadMore) {
			currentPage.value = previousPage
			hasMore.value = previousHasMore
		}
		leaderboardState.value = nextErrorState
	} finally {
		isLoadingMore.value = false
	}
}

const retryLeaderboard = () => fetchLeaderboard(true, false)

const handlePageErrorAction = () => {
	if (leaderboardState.value.error?.category === 'permission' || leaderboardState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryLeaderboard()
}

/**
 * 下拉刷新
 */
const onRefresh = () => {
	fetchLeaderboard(true, false)
}

/**
 * 上拉加载更多
 */
const onLoadMore = () => {
	fetchLeaderboard(false, true)
}

const handleGameTypeChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	currentPage.value = 1
	hasMore.value = true
	fetchLeaderboard(true, false)
}

/**
 * 点击用户查看详情
 */
const handleUserClick = (user) => {
	uni.showToast({
		title: '对方隐藏战绩，无法查看',
		icon: 'none',
		duration: 2000
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
	defaultGameType.value = readDefaultGameType(uni, userStore.userId)
	showGameTypeModal.value = true
}

/**
 * 处理比赛类型确认
 */
const handleGameTypeConfirm = ({ gameType, setAsDefault } = {}) => {
	if (setAsDefault) {
		defaultGameType.value = saveDefaultGameType(uni, userStore.userId, gameType)
	}
	selectedGameType.value = gameType
	// 开始扫码
	handleScanCode()
}

/**
 * 开始扫码
 */
const handleScanCode = async () => {
	const scanAction = await scanAndResolveMatchCode({
		uniApi: uni,
		previewMatchInvite
	})
	if (scanAction.type === 'cancelled') return
	if (scanAction.type === 'error') {
		uni.showToast({ title: scanAction.message || '扫码失败', icon: 'none' })
		return
	}
	if (scanAction.type !== 'start_match') {
		uni.showToast({ title: '请扫描匹配二维码发起对局', icon: 'none' })
		return
	}
	await handleMatchResult(scanAction)
}

/**
 * 处理匹配结果
 */
const handleMatchResult = async (scanAction) => {
	try {
		const opponentData = scanAction.opponent

		// 显示加载中
		uni.showLoading({ title: '匹配中...', mask: true })

		// 创建对局
		const res = await startMatch(buildStartMatchPayload({
			gameType: selectedGameType.value,
			opponent: opponentData,
			inviteToken: scanAction.inviteToken
		}))

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
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
