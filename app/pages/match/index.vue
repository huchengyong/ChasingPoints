<template>
	<view class="match-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 主内容区 -->
		<view class="main-content">
			<!-- 加载状态 -->
			<view v-if="showPageLoading" class="loading-container">
					<uni-icons type="spinner-cycle" size="48" color="#E0AE12"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 空数据状态 -->
			<view v-else-if="!currentMatch && ongoingMatches.length === 0" class="empty-state">
				<uni-icons type="medal" size="128" color="#6b7280" class="empty-icon"></uni-icons>
				<text class="empty-title">暂无正在进行的对局</text>
				<text class="empty-subtitle">是时候展现你的台球技巧了！</text>
				<button class="start-button" @click="handleStartMatch">
					<text>发起PK</text>
				</button>
				<button class="secondary-link" @click="handleScanAsReferee">
					<text>专业比赛需要裁判？扫码进入</text>
				</button>
			</view>

			<!-- 对局列表 -->
			<view v-else class="match-list">
				<view v-if="userStore.isLoggedIn" class="quick-actions">
					<button class="quick-action quick-action--primary" @click="handleStartMatch">
						<text>发起PK</text>
					</button>
					<button class="quick-link" @click="handleScanAsReferee">
						<text>扫码担任裁判</text>
					</button>
				</view>

				<!-- 进行中的对局 -->
				<view v-if="currentMatch" class="match-card my-match" @click="handleContinueMatch(currentMatch)">
					<!-- MY MATCH 标签 -->
					<view class="my-match-badge">我的对局</view>
					<view v-if="currentMatch.viewer_role === 'referee'" class="referee-match-info">
						<text class="referee-match-info__title">你正在担任本场裁判</text>
						<text class="referee-match-info__score">{{ currentMatch.my_score }} : {{ currentMatch.opponent_score }}</text>
						<text class="referee-match-info__hint">点击进入裁判记分页</text>
					</view>
					<view v-else class="match-info">
						<!-- 我方玩家 -->
						<view class="player">
							<view class="avatar me-avatar" :class="{ winner: currentMatch.my_score > currentMatch.opponent_score }">
								<image :src="userAvatar || '/static/images/default-avatar.png'" mode="aspectFill" />
								<view class="me-badge">我</view>
							</view>
							<text class="name">{{ userName }}</text>
						</view>

						<!-- 比分 -->
						<view class="score-area">
							<text class="score">{{ currentMatch.my_score }} : {{ currentMatch.opponent_score }}</text>
							<text class="vs-text">VS</text>
						</view>

						<!-- 对手玩家 -->
						<view class="player">
							<view class="avatar" :class="{ winner: currentMatch.opponent_score > currentMatch.my_score }">
								<image src="/static/images/default-avatar.png" mode="aspectFill" />
							</view>
							<text class="name">{{ currentMatch.opponent_name }}</text>
						</view>
					</view>

					<!-- 底部信息 -->
					<view class="match-footer">
						<view :class="['game-type-tag', getGameTypeClass(currentMatch.game_type)]">
							{{ currentMatch.game_type_name }}
						</view>
						<view class="match-status">
							<uni-icons type="circle" size="14" color="#22c55e"></uni-icons>
							<text>进行中 {{ formatDuration(currentMatch.duration_seconds) }}</text>
						</view>
					</view>
				</view>

				<!-- 平台正在进行的对局列表 -->
				<view
					v-for="match in ongoingMatches"
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
						<view class="match-status">
							<uni-icons type="circle" size="14" color="#22c55e"></uni-icons>
							<text>进行中 {{ formatDuration(match.duration_seconds) }}</text>
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
	</view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

import { onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { getCurrentMatch, getOngoingMatches, joinMatchReferee, startMatch } from '@/api/match.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { shouldShowMatchPageLoading } from '@/utils/match-page.js'
import { buildPlayingRoute, resolveMatchScanAction, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'

// ========== 状态管理 ==========
const userStore = useUserStore()
const { isDarkMode } = usePageTheme()

// ========== 响应式数据 ==========
const loading = ref(false)
const refreshing = ref(false)
const currentMatch = ref(null)
const ongoingMatches = ref([]) // 平台正在进行的对局列表
const showGameTypeModal = ref(false)
const selectedGameType = ref(null)
const scanIntent = ref('start')
const showPageLoading = computed(() => shouldShowMatchPageLoading(loading.value, refreshing.value))

// 用户信息
const userName = computed(() => userStore.userInfo?.nickname || '我')
const userAvatar = computed(() => userStore.userInfo?.avatar || '')
const userId = computed(() => userStore.userInfo?.id || 0)

/**
 * 判断是否是当前用户参与的对局
 */
const isMyMatch = (match) => {
	if (!userStore.isLoggedIn || !userId.value) return false
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
			getOngoingMatches({ page: 1, page_size: 20 }).catch(() => ({ success: false, list: [] }))
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

			// 设置平台正在进行的对局列表（过滤掉自己的对局）
			const ongoingRes = results[1]
			if (ongoingRes.success && ongoingRes.list) {
				// 如果有自己的对局，从列表中过滤掉
				if (currentMatch.value) {
					ongoingMatches.value = ongoingRes.list.filter(
						m => m.id !== currentMatch.value.id
					)
				} else {
					ongoingMatches.value = ongoingRes.list
				}
			}
		} else {
			currentMatch.value = null
			// 设置平台正在进行的对局列表
			const ongoingRes = results[0]
			if (ongoingRes.success && ongoingRes.list) {
				ongoingMatches.value = ongoingRes.list
			}
		}
	} catch (error) {
		console.error('加载对局数据失败:', error)
	} finally {
		loading.value = false
	}
}

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

/**
 * 格式化对局时长（使用服务端返回的秒数）
 * @param {number} durationSeconds 对局持续秒数
 */
const formatDuration = (durationSeconds) => {
	if (!durationSeconds || durationSeconds < 0) return '00:00'

	const minutes = Math.floor(durationSeconds / 60)
	const seconds = durationSeconds % 60

	return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
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
	if (isPlayer1Me(match)) {
		return match.player2_name || '对手'
	}
	return match.player1_name || '对手'
}

/**
 * 获取对手头像（根据当前用户判断）
 */
const getOpponentAvatar = (match) => {
	if (isPlayer1Me(match)) {
		return match.player2_avatar || ''
	}
	return match.player1_avatar || ''
}

/**
 * 点击对局卡片
 */
const handleMatchClick = (match) => {
	if (isMyMatch(match)) {
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
