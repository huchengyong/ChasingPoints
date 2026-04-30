<template>
	<view class="match-detail-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载状态 -->
		<view v-if="loading" class="loading-container">
			<uni-icons type="spinner-cycle" size="48" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<template v-else>
			<!-- 观战模式标识 -->
			<view v-if="isSpectateMode" class="spectate-badge">
				<uni-icons type="eye" size="16" color="#ffffff"></uni-icons>
				<text>观战模式</text>
			</view>

			<!-- 双方信息卡片 -->
			<view class="players-card">
				<!-- 双方头像 -->
				<view class="avatars-section">
					<view class="player-info">
						<view class="avatar-wrapper">
							<image 
								class="avatar" 
								:src="player1Info.avatar || '/static/images/default-avatar.png'" 
								mode="aspectFill" 
							/>
						</view>
						<text class="player-name">{{ player1Info.name || '玩家1' }}</text>
					</view>

					<view class="vs-section">
						<text class="vs-text">VS</text>
					</view>

					<view class="player-info">
						<view class="avatar-wrapper">
							<image 
								class="avatar" 
								:src="player2Info.avatar || '/static/images/default-avatar.png'" 
								mode="aspectFill" 
							/>
						</view>
						<text class="player-name">{{ player2Info.name || '玩家2' }}</text>
					</view>
				</view>

				<!-- 比分展示 -->
				<view class="score-display">
					<text class="score-text">{{ matchData.player1_score }} : {{ matchData.player2_score }}</text>
				</view>

				<!-- 对局信息 -->
				<view class="match-info-text">
					{{ gameTypeName }} · {{ formatDuration(matchData.duration_seconds) }}
				</view>

				<!-- 统计数据 -->
				<view class="stats-grid">
					<view class="stat-item">
						<text class="stat-label">总局数</text>
						<text class="stat-value">{{ matchData.total_rounds || 0 }}</text>
					</view>
					<view class="stat-item">
						<text class="stat-label">当前局</text>
						<text class="stat-value">{{ matchData.current_round || 1 }}</text>
					</view>
					<view class="stat-item">
						<text class="stat-label">状态</text>
						<text class="stat-value" :class="matchData.status === 2 ? 'status-finished' : 'status-ongoing'">{{ matchStatusText }}</text>
					</view>
				</view>
			</view>

			<!-- 局记录列表 -->
			<view class="round-history">
				<text class="section-title">局记录</text>
				
				<view v-if="roundRecords.length === 0" class="empty-rounds">
					<text class="empty-text">暂无已完成的局</text>
				</view>

				<view 
					v-for="(round, index) in roundRecords" 
					:key="index"
					:class="['round-card', getRoundResultClass(round)]"
				>
					<view class="round-info">
						<text class="round-date">第 {{ round.round_number }} 局</text>
						<text class="round-game-type">{{ gameTypeName }}</text>
					</view>
					<view class="round-result">
						<text class="round-score">{{ round.player1_score }} - {{ round.player2_score }}</text>
						<view class="result-badge">
							<text :class="['result-text', getRoundResultTone(round)]">
								{{ getRoundResultText(round) }}
							</text>
						</view>
						<uni-icons type="right" size="16" color="#64748b"></uni-icons>
					</view>
				</view>
			</view>
		</template>
	</view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { matchWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'
import { getMatchDetail, getPublicMatchDetail } from '@/api/match.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	normalizeMatchDetailPayload,
	shouldShowSpectateBadge,
	shouldUsePublicMatchDetail
} from '@/utils/match-detail.js'

// ========== 响应式数据 ==========
const loading = ref(true)
const matchId = ref(null)
const isSpectateMode = ref(false)
const detailSource = ref('')
const perspectiveUserId = ref(0)
const matchData = ref({
	player1_score: 0,
	player2_score: 0,
	game_type: 3,
	status: 0,
	duration_seconds: 0,
	current_round: 1,
	total_rounds: 0
})
const player1Info = ref({
	name: '',
	avatar: ''
})
const player2Info = ref({
	name: '',
	avatar: ''
})
const roundRecords = ref([])
let wsHandlersReady = false

const pageLog = (message, payload) => {
	if (payload === undefined) {
		console.log(`[MatchDetail] ${message}`)
		return
	}
	console.log(`[MatchDetail] ${message}`, payload)
}

// ========== 状态管理 ==========
const { isDarkMode } = usePageTheme()

// ========== 计算属性 ==========

// ========== 计算属性 ==========
const gameTypeName = computed(() => {
	switch (matchData.value.game_type) {
		case 1: return '斯诺克'
		case 2: return '九球追分'
		case 3: return '中式八球'
		case 4: return '美式九球'
		default: return '未知'
	}
})

const matchStatusText = computed(() => {
	if (matchData.value.status === 2) return '已结束'
	if (matchData.value.status === 3) return '已取消'
	return '进行中'
})

// ========== 生命周期 ==========
onLoad((options) => {
	if (options.match_id) {
		matchId.value = parseInt(options.match_id)
	}
	detailSource.value = options.source || ''
	perspectiveUserId.value = Number(options.perspective_user_id || 0)
	isSpectateMode.value = shouldShowSpectateBadge(options)
	if (isSpectateMode.value) {
		// 设置导航栏标题
		uni.setNavigationBarTitle({
			title: '观战'
		})
	} else {
		uni.setNavigationBarTitle({
			title: '对局详情'
		})
	}
	
	loadMatchData()
})

onMounted(() => {
	// 注册WebSocket消息处理
	wsHandlersReady = true
	matchWS.on(WS_MESSAGE_TYPES.SCORE_UPDATE, handleScoreUpdate)
	matchWS.on(WS_MESSAGE_TYPES.ROUND_END, handleRoundEnd)
	matchWS.on(WS_MESSAGE_TYPES.MATCH_END, handleMatchEnd)
	matchWS.on(WS_MESSAGE_TYPES.SYNC, handleSync)
	if (matchData.value.status === 1) {
		connectWebSocket()
	}
})

onShow(() => {
	if (wsHandlersReady && matchId.value && matchData.value.status === 1) {
		if (matchWS.isConnected()) {
			matchWS.requestSync()
		} else {
			connectWebSocket()
		}
	}
})

onUnmounted(() => {
	wsHandlersReady = false
	matchWS.off(WS_MESSAGE_TYPES.SCORE_UPDATE, handleScoreUpdate)
	matchWS.off(WS_MESSAGE_TYPES.ROUND_END, handleRoundEnd)
	matchWS.off(WS_MESSAGE_TYPES.MATCH_END, handleMatchEnd)
	matchWS.off(WS_MESSAGE_TYPES.SYNC, handleSync)
	// 断开WebSocket
	matchWS.disconnect()
})

// ========== 方法 ==========

/**
 * 加载对局数据
 */
const loadMatchData = async () => {
	if (!matchId.value) {
		loading.value = false
		return
	}
	
	try {
		const apiCall = shouldUsePublicMatchDetail({
			mode: isSpectateMode.value ? 'spectate' : '',
			source: detailSource.value
		}) ? getPublicMatchDetail : getMatchDetail
		const res = await apiCall({ match_id: matchId.value })
		if (res.success && res.match) {
			const normalized = normalizeMatchDetailPayload(res.match, {
				perspectiveUserId: perspectiveUserId.value
			})
			matchData.value = normalized.matchData
			player1Info.value = normalized.player1Info
			player2Info.value = normalized.player2Info
			roundRecords.value = normalized.roundRecords
			if (wsHandlersReady && matchData.value.status === 1 && !matchWS.isConnected()) {
				connectWebSocket()
			}
		}
	} catch (error) {
		console.error('[MatchDetail] 加载对局详情失败', { matchId: matchId.value, error })
		uni.showToast({ title: '加载失败', icon: 'none' })
	} finally {
		loading.value = false
	}
}

/**
 * 连接WebSocket (观战模式)
 */
const connectWebSocket = async () => {
	if (!matchId.value) return
	if (matchData.value.status !== 1) return
	
	try {
		await matchWS.connect(matchId.value, { allowAnonymous: isSpectateMode.value })
		pageLog('WebSocket连接成功', { matchId: matchId.value, spectate: isSpectateMode.value })
	} catch (error) {
		console.error('[MatchDetail] WebSocket连接失败', { matchId: matchId.value, spectate: isSpectateMode.value, error })
	}
}

/**
 * 处理分数更新消息
 */
const handleScoreUpdate = (data) => {
	matchData.value.player1_score = data.player1_score ?? data.my_score
	matchData.value.player2_score = data.player2_score ?? data.opponent_score
	matchData.value.current_round = data.current_round
	pageLog('收到比分更新', {
		matchId: data.match_id,
		currentRound: data.current_round,
		player1Score: data.player1_score ?? data.my_score,
		player2Score: data.player2_score ?? data.opponent_score
	})
}

/**
 * 处理局结束消息
 */
const handleRoundEnd = (data) => {
	matchData.value.player1_score = data.player1_score ?? data.my_score
	matchData.value.player2_score = data.player2_score ?? data.opponent_score
	matchData.value.current_round = data.current_round
	pageLog('收到单局结束', {
		matchId: data.match_id,
		roundNumber: data.round_number,
		winner: data.winner,
		currentRound: data.current_round
	})

	if (data.round_number) {
		roundRecords.value.push({
			round_number: data.round_number,
			player1_score: data.round_player1_score,
			player2_score: data.round_player2_score,
			winner: data.winner
		})
	} else {
		matchWS.requestSync()
	}
}

/**
 * 处理对局结束消息
 */
const handleMatchEnd = (data) => {
	pageLog('收到对局结束', {
		matchId: data.match_id,
		player1Score: data.player1_score ?? data.my_score,
		player2Score: data.player2_score ?? data.opponent_score,
		status: data.status
	})
	uni.showModal({
		title: '对局结束',
		content: `最终比分: ${data.player1_score ?? data.my_score} : ${data.player2_score ?? data.opponent_score}`,
		showCancel: false,
		success: () => {
			uni.navigateBack()
		}
	})
}

const handleSync = (data) => {
	matchData.value.player1_score = data.player1_score || 0
	matchData.value.player2_score = data.player2_score || 0
	matchData.value.current_round = data.current_round || 1
	matchData.value.total_rounds = data.total_rounds || 0
	if (Array.isArray(data.rounds)) {
		roundRecords.value = data.rounds
	}
	pageLog('收到同步快照', {
		matchId: data.match_id,
		currentRound: data.current_round,
		totalRounds: data.total_rounds,
		status: data.status
	})
}

/**
 * 格式化时长
 */
const formatDuration = (durationSeconds) => {
	if (!durationSeconds || durationSeconds < 0) return '0分'

	const totalMinutes = Math.floor(durationSeconds / 60)

	if (totalMinutes < 1) {
		return `${durationSeconds}秒`
	}

	if (totalMinutes < 60) {
		return `${totalMinutes}分`
	}

	const hours = Math.floor(totalMinutes / 60)
	const remainMinutes = totalMinutes % 60

	if (hours < 24) {
		if (remainMinutes > 0) {
			return `${hours}小时${remainMinutes}分`
		}
		return `${hours}小时`
	}

	const days = Math.floor(hours / 24)
	const remainHours = hours % 24

	if (remainHours > 0) {
		return `${days}天${remainHours}小时`
	}
	return `${days}天`
}

/**
 * 获取局结果样式类
 */
const getRoundResultClass = (round) => {
	if (round.result === 'win' || round.winner === 1) return 'win-round'
	return 'loss-round'
}

const getRoundResultTone = (round) => {
	return round.result === 'win' || round.winner === 1 ? 'win' : 'loss'
}

const getRoundResultText = (round) => {
	return round.resultText || (round.result === 'win' || round.winner === 1 ? '胜' : '负')
}
</script>

<style lang="scss" scoped>
@import './matchDetail.scss';
</style>
