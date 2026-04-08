<template>
	<view class="playing-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="header" :style="{ paddingTop: statusBarHeight + 'px' }">
			<view class="header-content">
				<view class="header-left"></view>
				<view class="header-center">
					<text class="game-title">{{ gameTypeName }}</text>
					<text class="game-subtitle">{{ scoreboardSubtitle }}</text>
				</view>
				<view class="header-right"></view>
			</view>
		</view>

		<view class="main-content">
			<view class="scoreboard-card">
				<view class="scoreboard-compact">
					<view class="player-compact is-me">
						<view class="player-compact__main">
							<view class="player-compact__profile">
								<view class="avatar-wrapper">
									<image class="avatar" :src="myInfo.avatar || DEFAULT_AVATAR" mode="aspectFill"></image>
								</view>
								<view class="player-info">
									<view class="player-name-row">
										<view class="identity-tag is-me">{{ viewerUi.leftIdentity }}</view>
										<text class="player-name">{{ myInfo.nickname || '我' }}</text>
									</view>
									<view class="player-meta">
										<text class="player-meta__item">胜率 {{ myInfo.winRate || 0 }}%</text>
										<text class="player-meta__item">单杆 {{ myInfo.maxScore || 0 }}</text>
									</view>
								</view>
							</view>
							<text class="player-score is-me">{{ myScore }}</text>
						</view>
					</view>

					<view class="player-compact is-opponent">
						<view class="player-compact__main">
							<view class="player-compact__profile is-reverse">
								<view class="player-info is-right">
									<view class="player-name-row is-right">
										<text class="player-name">{{ opponentInfo.nickname || '对手' }}</text>
										<view class="identity-tag is-opponent">{{ viewerUi.rightIdentity }}</view>
									</view>
									<view class="player-meta is-right">
										<text class="player-meta__item">胜率 {{ opponentInfo.winRate || 0 }}%</text>
										<text class="player-meta__item">单杆 {{ opponentInfo.maxScore || 0 }}</text>
									</view>
								</view>
								<view class="avatar-wrapper">
									<image class="avatar" :src="opponentInfo.avatar || DEFAULT_AVATAR" mode="aspectFill"></image>
								</view>
							</view>
							<text class="player-score is-opponent">{{ opponentScore }}</text>
						</view>
					</view>
				</view>
				<view class="scoreboard-meta">
					<text class="scoreboard-meta__text">{{ currentRoundText }}</text>
					<text class="scoreboard-meta__divider">·</text>
					<view class="status-chip" :class="{ syncing: isSyncing }">
						<text class="status-chip__dot"></text>
						<text class="status-chip__text">{{ syncStatusText }}</text>
					</view>
				</view>
				<view class="viewer-banner">
					<text class="viewer-banner__role">{{ viewerUi.roleLabel }}</text>
					<text v-if="viewerUi.readonlyHint" class="viewer-banner__hint">{{ viewerUi.readonlyHint }}</text>
				</view>
				<view v-if="gameType === 1" class="snooker-frame-summary">
					<view class="snooker-frame-summary__label">
						<text>当前局比分</text>
						<text class="snooker-frame-summary__status">{{ currentFrameStarted ? '进行中' : '待开始' }}</text>
					</view>
					<text class="snooker-frame-summary__score">{{ currentFrameMyScore }} : {{ currentFrameOpponentScore }}</text>
				</view>
			</view>

			<view v-if="viewerUi.showActionPanel" class="action-panel">
					<view class="panel-header">
						<text class="panel-title">记分操作</text>
						<text class="panel-desc">{{ actionPanelDescription }}</text>
					</view>

				<view v-if="isRoundWinMode" class="action-grid is-three">
					<button
						v-for="item in roundWinActionOptions"
						:key="item.type"
						class="action-btn action-btn--opponent"
						@click="handleOpponentWin(item.type)"
					>
						<text class="action-btn__title">{{ item.title }}</text>
						<text class="action-btn__desc">{{ item.desc }}</text>
					</button>
				</view>

				<view v-if="gameType === 1" class="snooker-layout">
					<view class="snooker-section">
						<view class="snooker-section__head">
							<text class="action-column__title is-opponent">给对手计分</text>
							<text :class="['snooker-red-counter', { 'is-warning': snookerRedBallRemaining <= 2 && !isSnookerRedDisabled, 'is-finished': isSnookerRedDisabled }]">
								{{ snookerRedBallText }}
							</text>
						</view>
						<view class="snooker-stage is-score">
							<view class="snooker-stage__left">
								<view class="snooker-ball-row is-score-top">
									<button
										v-for="item in snookerScoreTopRow"
										:key="item.key"
										:class="['snooker-ball-btn', `is-${item.color}`, { 'is-disabled': isSnookerScoreDisabled(item.score) }]"
										:disabled="isSnookerScoreDisabled(item.score)"
										@click="handleAddScore(item.score)"
									>
										<view class="snooker-ball-btn__ball">
											<text class="snooker-ball-btn__score">{{ item.score }}</text>
										</view>
										<text class="snooker-ball-btn__label">{{ item.name }}</text>
										<text class="snooker-ball-btn__meta">{{ item.score }} 分</text>
									</button>
								</view>
								<view class="snooker-ball-row is-score-bottom">
									<button
										v-for="item in snookerScoreBottomRow"
										:key="item.key"
										:class="['snooker-ball-btn', `is-${item.color}`, { 'is-disabled': isSnookerScoreDisabled(item.score) }]"
										:disabled="isSnookerScoreDisabled(item.score)"
										@click="handleAddScore(item.score)"
									>
										<view class="snooker-ball-btn__ball">
											<text class="snooker-ball-btn__score">{{ item.score }}</text>
										</view>
										<text class="snooker-ball-btn__label">{{ item.name }}</text>
										<text class="snooker-ball-btn__meta">{{ item.score }} 分</text>
									</button>
								</view>
							</view>
							<button
								:class="['snooker-ball-btn', 'is-tall', 'is-primary', `is-${snookerPrimaryScore.color}`, { 'is-disabled': isSnookerScoreDisabled(snookerPrimaryScore.score) }]"
								:disabled="isSnookerScoreDisabled(snookerPrimaryScore.score)"
								@click="handleAddScore(snookerPrimaryScore.score)"
							>
								<view class="snooker-ball-btn__ball">
									<text class="snooker-ball-btn__score">{{ snookerPrimaryScore.score }}</text>
								</view>
								<text class="snooker-ball-btn__label">{{ snookerPrimaryScore.name }}</text>
								<text class="snooker-ball-btn__meta">{{ snookerPrimaryScore.score }} 分</text>
							</button>
						</view>
						<text class="snooker-hint">{{ snookerRedHintText }}</text>
					</view>

					<view class="snooker-section">
						<text class="action-column__title is-me">对手犯规，给我方加分</text>
						<view class="snooker-stage is-foul">
							<view class="snooker-stage__left">
								<view class="snooker-ball-row is-foul-top">
									<button
										v-for="item in snookerFoulTopRow"
										:key="item.key"
										:class="['snooker-ball-btn', 'is-foul', `is-${item.color}`]"
										@click="handleFoulByScore(item.score)"
									>
										<view class="snooker-ball-btn__ball">
											<text class="snooker-ball-btn__score">{{ item.score }}</text>
										</view>
										<text class="snooker-ball-btn__label">{{ item.name }}</text>
										<text class="snooker-ball-btn__meta">犯规 {{ item.score }} 分</text>
									</button>
								</view>
								<view class="snooker-ball-row is-foul-bottom">
									<button
										v-for="item in snookerFoulBottomRow"
										:key="item.key"
										:class="['snooker-ball-btn', 'is-foul', 'is-wide', `is-${item.color}`]"
										@click="handleFoulByScore(item.score)"
									>
										<view class="snooker-ball-btn__ball">
											<text class="snooker-ball-btn__score">{{ item.score }}</text>
										</view>
										<text class="snooker-ball-btn__label">{{ item.name }}</text>
										<text class="snooker-ball-btn__meta">犯规 {{ item.score }} 分</text>
									</button>
								</view>
							</view>
							<button
								:class="['snooker-ball-btn', 'is-foul', 'is-tall', 'is-primary', `is-${snookerPrimaryFoul.color}`]"
								@click="handleFoulByScore(snookerPrimaryFoul.score)"
							>
								<view class="snooker-ball-btn__ball">
									<text class="snooker-ball-btn__score">{{ snookerPrimaryFoul.score }}</text>
								</view>
								<text class="snooker-ball-btn__label">{{ snookerPrimaryFoul.name }}</text>
								<text class="snooker-ball-btn__meta">犯规 {{ snookerPrimaryFoul.score }} 分</text>
							</button>
						</view>
					</view>
				</view>

				<view v-if="gameType === 2" class="jiuqiu-layout">
					<view class="action-column">
						<text class="action-column__title is-opponent">记到对手</text>
						<button
							class="action-btn action-btn--opponent"
							@click="handleOpponentWin(jiuqiuNormalOption.type, jiuqiuNormalOption.score)"
						>
							<text class="action-btn__title">{{ jiuqiuNormalOption.title }}</text>
							<text class="action-btn__desc">{{ jiuqiuNormalOption.desc }}</text>
						</button>
						<view class="action-row-two">
							<button
								v-for="item in jiuqiuGoldOptions"
								:key="item.type"
								class="action-btn action-btn--opponent action-btn--compact"
								@click="handleOpponentWin(item.type, item.score)"
							>
								<text class="action-btn__title">{{ item.title }}</text>
								<text class="action-btn__desc">{{ item.desc }}</text>
							</button>
						</view>
					</view>
					<view class="action-column">
						<text class="action-column__title is-me">记到我方</text>
						<button class="action-btn action-btn--me" @click="handleFoul(2)">
							<text class="action-btn__title">对手犯规</text>
							<text class="action-btn__desc">我方 +1 分</text>
						</button>
					</view>
				</view>
			</view>
			<view v-else class="readonly-panel">
				<text class="readonly-panel__title">当前为只读观赛</text>
				<text class="readonly-panel__desc">{{ viewerUi.readonlyHint || '请等待裁判完成记分操作' }}</text>
			</view>
		</view>

		<view class="footer">
			<view class="footer-buttons">
				<button
					v-if="showInviteRefereeAction"
					class="footer-btn btn-secondary full-width"
					@click="openRefereeQrModal"
				>
					邀请裁判
				</button>
				<button
					v-if="gameType === 1 && viewerUi.showActionPanel"
					class="footer-btn btn-secondary full-width"
					@click="handleNextRound"
				>
					开始下一局
				</button>
				<button v-if="viewerUi.showFinishButton" class="footer-btn btn-primary full-width" @click="handleFinishMatch">结束本场对局</button>
			</view>
			<button v-if="viewerUi.showUndoButton" class="undo-btn" @click="handleUndo">撤销</button>
		</view>

		<view v-if="showRefereeQrModal" class="referee-modal-overlay" @click="closeRefereeQrModal">
			<view class="referee-modal-container" @click.stop>
				<text class="referee-modal-title">邀请裁判扫码</text>
				<text class="referee-modal-desc">让球童、助教或第三方扫码后接管本场记分</text>
				<view v-if="refereeQrcodeLoading" class="referee-modal-loading">
					<text>生成中...</text>
				</view>
				<image
					v-else-if="refereeQrcodeUrl"
					class="referee-qrcode-image"
					:src="refereeQrcodeUrl"
					mode="aspectFit"
				/>
				<text class="referee-modal-hint">裁判加入后，选手端会自动切换为只读比分视图。</text>
				<button class="referee-modal-close" @click="closeRefereeQrModal">关闭</button>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { onLoad, onShow, onHide } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { matchWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'
import { matchScore, endRound, startNextRound, matchFoul, matchUndo, finishMatch, getMatchDetail, getCurrentMatch, getMatchRefereeQRCode } from '@/api/match.js'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
import { consumeResultNavigationGuard, getMatchHistoryPageUrl, getMatchHistoryTabUrl, shouldLeavePlayingPage } from '@/utils/match-navigation.js'
import { buildMatchActionPayload } from '@/utils/match-action.js'
import { resolvePlayingViewerUi } from '@/utils/match-role-view.js'
import { resolveSnookerFinishMatchAction, resolveSnookerNextFrameAction } from '@/utils/snooker-frame.js'

// ========== 状态管理 ==========
const userStore = useUserStore()

// ========== 响应式数据 ==========
const matchId = ref(null)
const gameType = ref(3) // 1=斯诺克 2=九球追分 3=中式八球 4=美式九球
const myScore = ref(0)
const opponentScore = ref(0)
const currentFrameMyScore = ref(0)
const currentFrameOpponentScore = ref(0)
const currentFrameStarted = ref(true)
const currentRound = ref(1)
const serverRevision = ref(0)
const snookerRedBallCount = ref(0)
const snookerClearanceStarted = ref(false)
const snookerClearedColors = ref([])
const snookerExpectedClearanceScore = ref(0)
const snookerClearanceCompleted = ref(false)
const DEFAULT_AVATAR = '/static/default-avatar.png'
const myInfo = ref({ avatar: DEFAULT_AVATAR })
const opponentInfo = ref({ avatar: DEFAULT_AVATAR })
const viewerRole = ref('player1')
const refereeBound = ref(false)
const refereeUserId = ref(0)
const refereeName = ref('')
const canScore = ref(true)
const canUndo = ref(true)
const canFinish = ref(true)
const showRefereeQrModal = ref(false)
const refereeQrcodeLoading = ref(false)
const refereeQrcodeUrl = ref('')
const statusBarHeight = ref(0)
const isPlayer1 = ref(true) // 是否是对局创建者(用于视角判断)
const isSyncing = ref(false) // 比分同步中状态
const isLeavingPlayingPage = ref(false)
let syncTimeoutId = null // 同步超时定时器
let wsHandlersReady = false
const resultNavigationState = { hasNavigatedToResult: false }

const pageLog = (message, payload) => {
	if (payload === undefined) {
		console.log(`[MatchPlaying] ${message}`)
		return
	}
	console.log(`[MatchPlaying] ${message}`, payload)
}

const updateServerRevision = (revision) => {
	const normalizedRevision = Number(revision)
	if (!Number.isFinite(normalizedRevision) || normalizedRevision < 0) {
		return
	}
	serverRevision.value = Math.max(serverRevision.value, normalizedRevision)
}

const shouldApplyIncomingRevision = (revision) => {
	const normalizedRevision = Number(revision)
	if (!Number.isFinite(normalizedRevision) || normalizedRevision <= 0) {
		return true
	}
	if (normalizedRevision < serverRevision.value) {
		pageLog('忽略过期同步消息', {
			matchId: matchId.value,
			incomingRevision: normalizedRevision,
			localRevision: serverRevision.value
		})
		return false
	}
	return true
}

const buildActionRequest = (payload = {}) => buildMatchActionPayload(payload, serverRevision.value)

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 计算属性 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)

// 获取状态栏高度
const systemInfo = uni.getSystemInfoSync()
statusBarHeight.value = systemInfo.statusBarHeight || 20

// ========== 计算属性 ==========
const gameTypeName = computed(() => {
	switch (gameType.value) {
		case 1: return '斯诺克'
		case 2: return '九球追分'
		case 3: return '中式八球'
		case 4: return '美式九球'
		default: return '未知'
	}
})

const gameSubtitle = computed(() => {
	switch (gameType.value) {
		case 1: return '上方为整场 frame，下方记录当前局得分'
		case 2: return '区分我方得分与对手得分'
		case 3: return '明确判定对手的本局胜法'
		case 4: return '明确判定对手的本局胜法'
		default: return ''
	}
})

const viewerUi = computed(() => resolvePlayingViewerUi({
	viewerRole: viewerRole.value,
	canScore: canScore.value,
	canUndo: canUndo.value,
	canFinish: canFinish.value,
	refereeName: refereeName.value
}))

const showInviteRefereeAction = computed(() => viewerRole.value !== 'referee' && !refereeBound.value)

const scoreboardSubtitle = computed(() => `${gameSubtitle.value} · ${viewerUi.value.subtitleSuffix}`)

const currentRoundText = computed(() => {
	if (gameType.value === 1 && !currentFrameStarted.value) {
		return `待开始第 ${currentRound.value} 局`
	}
	return `第 ${currentRound.value} 局`
})
const snookerRedBallMax = 15
const snookerRedBallText = computed(() => `红球 ${snookerRedBallCount.value}/${snookerRedBallMax}`)
const snookerRedBallRemaining = computed(() => Math.max(snookerRedBallMax - snookerRedBallCount.value, 0))
const isSnookerRedDisabled = computed(() => gameType.value === 1 && (snookerRedBallCount.value >= snookerRedBallMax || !currentFrameStarted.value))
const snookerClearedColorNames = computed(() => snookerClearedColors.value.map((score) => getSnookerColorName(score)).filter(Boolean))
const nextSnookerClearanceColorName = computed(() => getSnookerColorName(snookerExpectedClearanceScore.value))
const snookerRedHintText = computed(() => {
	if (!currentFrameStarted.value) {
		return '当前局已结束，可开始下一局，或直接结束整场比赛'
	}
	if (snookerClearanceCompleted.value) {
		return '清彩阶段已完成，请确认本局胜方'
	}
	if (snookerClearanceStarted.value) {
		const clearedText = snookerClearedColorNames.value.length ? `，已完成：${snookerClearedColorNames.value.join('、')}` : ''
		const nextText = nextSnookerClearanceColorName.value ? `，下一颗：${nextSnookerClearanceColorName.value}` : ''
		return `已进入清彩阶段，彩球需按顺序计分${clearedText}${nextText}`
	}
	if (isSnookerRedDisabled.value) return '本局红球已打完，请先记录最后一颗红球后的彩球'
	if (snookerRedBallRemaining.value <= 2) return `本局红球还剩 ${snookerRedBallRemaining.value} 颗`
	return '斯诺克最多打完15套红球'
})

const syncStatusText = computed(() => isSyncing.value ? '比分同步中' : '实时同步中')

const actionPanelDescription = computed(() => {
	if (!canScore.value) return viewerUi.value.readonlyHint || '当前由裁判负责记分'
	if (viewerRole.value === 'referee') {
		switch (gameType.value) {
			case 1:
				return '为选手2记录进球，为选手1记录对手犯规得分'
			case 2:
				return '左列为选手1得分，右列为选手2得分'
			case 3:
			case 4:
				return '以下按钮会判给对应选手本局胜'
			default:
				return '请按选手1 / 选手2 的实际结果记分'
		}
	}
	switch (gameType.value) {
		case 1:
			return '对手每进一个球，都在下方点击对应的球进行记分'
		case 2:
			return '左列记到我方，右列记到对手'
		case 3:
		case 4:
			return '以下按钮都会判给对手本局胜'
		default:
			return '请确认分数归属后再点击'
	}
})

const isRoundWinMode = computed(() => gameType.value === 3 || gameType.value === 4)

const roundWinActionOptions = computed(() => {
	if (gameType.value === 4) {
		return [
			{ type: 'normal', title: '对手普胜', desc: '本局计给对手' },
			{ type: 'small_gold', title: '对手小金', desc: '对手金球直接制胜' },
			{ type: 'big_gold', title: '对手大金', desc: '对手开球直接制胜' }
		]
	}

	return [
		{ type: 'normal', title: '对手普胜', desc: '本局计给对手' },
		{ type: 'break_clear', title: '对手炸清', desc: '对手直接清台获胜' },
		{ type: 'continue_clear', title: '对手接清', desc: '对手连续清台获胜' }
	]
})

const snookerScoreBalls = computed(() => ([
	{ key: 'red', name: '红球', color: 'red', score: 1 },
	{ key: 'yellow', name: '黄球', color: 'yellow', score: 2 },
	{ key: 'green', name: '绿球', color: 'green', score: 3 },
	{ key: 'brown', name: '咖啡球', color: 'brown', score: 4 },
	{ key: 'blue', name: '蓝球', color: 'blue', score: 5 },
	{ key: 'pink', name: '粉球', color: 'pink', score: 6 },
	{ key: 'black', name: '黑球', color: 'black', score: 7 }
]))

const snookerPrimaryScore = computed(() => snookerScoreBalls.value[0])
const snookerScoreTopRow = computed(() => snookerScoreBalls.value.slice(1, 4))
const snookerScoreBottomRow = computed(() => snookerScoreBalls.value.slice(4))

const snookerFoulOptions = computed(() => ([
	{ key: 'foul4', name: '犯规 4 分', color: 'brown', score: 4 },
	{ key: 'foul5', name: '犯规 5 分', color: 'blue', score: 5 },
	{ key: 'foul6', name: '犯规 6 分', color: 'pink', score: 6 },
	{ key: 'foul7', name: '犯规 7 分', color: 'black', score: 7 }
]))

const snookerPrimaryFoul = computed(() => snookerFoulOptions.value[0])
const snookerFoulTopRow = computed(() => snookerFoulOptions.value.slice(1, 3))
const snookerFoulBottomRow = computed(() => snookerFoulOptions.value.slice(3))

const jiuqiuWinOptions = computed(() => [
	{ type: 'normal', score: 4, title: '对手普胜', desc: '对手 +4 分' },
	{ type: 'small_gold', score: 7, title: '对手小金', desc: '对手 +7 分' },
	{ type: 'big_gold', score: 10, title: '对手大金', desc: '对手 +10 分' }
])

const jiuqiuNormalOption = computed(() => jiuqiuWinOptions.value[0])

const jiuqiuGoldOptions = computed(() => jiuqiuWinOptions.value.slice(1))

// ========== 生命周期 ==========
onLoad((options) => {
	if (options.match_id) {
		matchId.value = parseInt(options.match_id)
	}
	if (options.game_type) {
		gameType.value = parseInt(options.game_type)
	}
	if (options.opponent_name) {
		opponentInfo.value.nickname = decodeURIComponent(options.opponent_name)
	}
	if (options.opponent_avatar) {
		opponentInfo.value.avatar = decodeURIComponent(options.opponent_avatar)
	}

	// 加载用户信息
	loadUserInfo()
})

onMounted(async () => {
	// 注册WebSocket消息处理
	wsHandlersReady = true
	matchWS.on(WS_MESSAGE_TYPES.SCORE_UPDATE, handleScoreUpdate)
	matchWS.on(WS_MESSAGE_TYPES.ROUND_END, handleRoundEnd)
	matchWS.on(WS_MESSAGE_TYPES.ROUND_START, handleRoundStart)
	matchWS.on(WS_MESSAGE_TYPES.MATCH_END, handleMatchEnd)
	matchWS.on(WS_MESSAGE_TYPES.MATCH_ROLE_CHANGED, handleRoleChanged)
	matchWS.on(WS_MESSAGE_TYPES.SYNC, handleSync)

	const canStayOnPlayingPage = await loadMatchInfo()
	if (canStayOnPlayingPage) {
		connectWebSocket()
	}
})

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	// 监听主题变化事件
	uni.$on(THEME_CHANGE_EVENT, handleThemeChange)
	if (wsHandlersReady && matchId.value) {
		resumeMatchIfStillActive()
	}
})

const resumeMatchIfStillActive = async () => {
	const canStayOnPlayingPage = await loadMatchInfo()
	if (!canStayOnPlayingPage || isLeavingPlayingPage.value) {
		return
	}

	if (wsHandlersReady && matchId.value) {
		if (matchWS.isConnected()) {
			matchWS.requestSync()
		} else {
			connectWebSocket()
		}
	}
}

onHide(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

onUnmounted(() => {
	wsHandlersReady = false
	matchWS.off(WS_MESSAGE_TYPES.SCORE_UPDATE, handleScoreUpdate)
	matchWS.off(WS_MESSAGE_TYPES.ROUND_END, handleRoundEnd)
	matchWS.off(WS_MESSAGE_TYPES.ROUND_START, handleRoundStart)
	matchWS.off(WS_MESSAGE_TYPES.MATCH_END, handleMatchEnd)
	matchWS.off(WS_MESSAGE_TYPES.MATCH_ROLE_CHANGED, handleRoleChanged)
	matchWS.off(WS_MESSAGE_TYPES.SYNC, handleSync)
	// 断开WebSocket
	matchWS.disconnect()
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

// ========== 方法 ==========

/**
 * 处理主题变化事件
 */
const handleThemeChange = () => {
	themeStore.applyNavigationBarTheme()
}

/**
 * 加载用户信息
 */
const loadUserInfo = () => {
	if (userStore.userInfo) {
		myInfo.value = {
			nickname: userStore.userInfo.nickname || '我',
			avatar: userStore.userInfo.avatar,
			winRate: 0, // 将由 loadMatchInfo 从API获取
			maxScore: 0
		}
	}
}

const applyViewerCapabilities = (payload = {}) => {
	if (typeof payload.viewer_role === 'string' && payload.viewer_role) {
		viewerRole.value = payload.viewer_role
	}
	if (typeof payload.referee_bound === 'boolean') {
		refereeBound.value = payload.referee_bound
		if (payload.referee_bound) {
			showRefereeQrModal.value = false
		}
	}
	if (payload.referee_user_id !== undefined) {
		refereeUserId.value = Number(payload.referee_user_id || 0)
	}
	if (payload.referee_name !== undefined) {
		refereeName.value = payload.referee_name || ''
	}
	if (typeof payload.can_score === 'boolean') {
		canScore.value = payload.can_score
	}
	if (typeof payload.can_undo === 'boolean') {
		canUndo.value = payload.can_undo
	}
	if (typeof payload.can_finish === 'boolean') {
		canFinish.value = payload.can_finish
	}
}

/**
 * 加载对局信息
 */
const loadMatchInfo = async () => {
	if (!matchId.value) return
	
	try {
		const res = await getMatchDetail({ match_id: matchId.value })
			if (res.match) {
				// 设置视角标识
				isPlayer1.value = res.match.is_player1
				applyViewerCapabilities(res.match)
				updateServerRevision(res.match.server_revision)
				
				// 设置分数（API已根据视角返回正确的分数）
				myScore.value = res.match.my_score
			opponentScore.value = res.match.opponent_score
			
			pageLog('加载到对局详情', {
				matchId: matchId.value,
				isPlayer1: res.match.is_player1,
				gameType: res.match.game_type,
				myScore: res.match.my_score,
				opponentScore: res.match.opponent_score,
				currentFrameMyScore: res.match.current_frame_my_score,
				currentFrameOpponentScore: res.match.current_frame_opponent_score,
				redBallCount: res.match.red_ball_count
			})
			
			// 设置我的信息（使用API返回的胜率和单杆最高分）
			myInfo.value = {
				nickname: res.match.my_name || '我',
				avatar: res.match.my_avatar || DEFAULT_AVATAR,
				winRate: Math.round(res.match.my_win_rate || 0),
				maxScore: res.match.my_max_score || 0
			}
			
			// 设置对手信息（使用API返回的胜率和单杆最高分）
			opponentInfo.value = {
				nickname: res.match.opponent_name || '对手',
				avatar: res.match.opponent_avatar || DEFAULT_AVATAR,
				winRate: Math.round(res.match.opponent_win_rate || 0),
				maxScore: res.match.opponent_max_score || 0
			}
			
			// 设置游戏类型
			if (res.match.game_type) {
				gameType.value = res.match.game_type
			}
			currentFrameStarted.value = !!res.match.current_frame_started
			currentFrameMyScore.value = res.match.current_frame_my_score || 0
			currentFrameOpponentScore.value = res.match.current_frame_opponent_score || 0
			applySnookerRoundState(res.match)
		}

		const currentRes = await getCurrentMatch().catch(() => null)
		const currentMatchId = currentRes?.success ? currentRes.match?.id || 0 : undefined
		if (shouldLeavePlayingPage({
			pageMatchId: matchId.value,
			detailStatus: res?.match?.status,
			currentMatchId
		})) {
			pageLog('检测到对局已结束或当前对局已切换，返回比赛记录页', {
				matchId: matchId.value,
				detailStatus: res?.match?.status,
				currentMatchId
			})
			redirectToMatchHistory()
			return false
		}

			if (currentRes?.success && currentRes.match && currentRes.match.id === matchId.value) {
				updateServerRevision(currentRes.match.server_revision)
				currentRound.value = currentRes.match.current_round || currentRound.value
			}
	} catch (error) {
		console.error('[MatchPlaying] 加载对局信息失败', { matchId: matchId.value, error })
	}

	return true
}

const redirectToMatchHistory = () => {
	if (isLeavingPlayingPage.value) return
	isLeavingPlayingPage.value = true
	matchWS.disconnect()
	uni.switchTab({
		url: getMatchHistoryTabUrl(),
		success: () => {
			setTimeout(() => {
				uni.navigateTo({
					url: getMatchHistoryPageUrl(),
					fail: () => {
						isLeavingPlayingPage.value = false
					}
				})
			}, 120)
		},
		fail: () => {
			isLeavingPlayingPage.value = false
			uni.reLaunch({
				url: getMatchHistoryPageUrl()
			})
		}
	})
}

/**
 * 连接WebSocket
 */
const connectWebSocket = async () => {
	if (!matchId.value) return
	
	try {
		await matchWS.connect(matchId.value)
		pageLog('WebSocket连接成功', { matchId: matchId.value })
	} catch (error) {
		console.error('[MatchPlaying] WebSocket连接失败', { matchId: matchId.value, error })
	}
}

const openRefereeQrModal = async () => {
	if (!matchId.value || refereeBound.value || viewerRole.value === 'referee') return

	showRefereeQrModal.value = true
	refereeQrcodeLoading.value = true
	try {
		const res = await getMatchRefereeQRCode({ match_id: matchId.value })
		if (res?.success && res.qrcode_data) {
			refereeQrcodeUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=${encodeURIComponent(res.qrcode_data)}`
			return
		}
		uni.showToast({ title: res?.message || '生成裁判码失败', icon: 'none' })
	} catch (error) {
		console.error('[MatchPlaying] 生成裁判码失败', error)
		uni.showToast({ title: '生成裁判码失败', icon: 'none' })
	} finally {
		refereeQrcodeLoading.value = false
	}
}

const closeRefereeQrModal = () => {
	showRefereeQrModal.value = false
}

const ensureViewerCapability = (allowed, message) => {
	if (allowed) return true
	uni.showToast({
		title: message || viewerUi.value.readonlyHint || '当前由裁判负责记分',
		icon: 'none'
	})
	return false
}

const applyServerScores = (player1Score, player2Score) => {
	if (isPlayer1.value) {
		myScore.value = player1Score
		opponentScore.value = player2Score
	} else {
		myScore.value = player2Score
		opponentScore.value = player1Score
	}
}

const applyServerCurrentFrameScores = (player1Score, player2Score) => {
	if (isPlayer1.value) {
		currentFrameMyScore.value = player1Score || 0
		currentFrameOpponentScore.value = player2Score || 0
	} else {
		currentFrameMyScore.value = player2Score || 0
		currentFrameOpponentScore.value = player1Score || 0
	}
}

const applySnookerRoundState = (payload = {}) => {
	snookerRedBallCount.value = payload.red_ball_count || 0
	snookerClearanceStarted.value = !!payload.snooker_clearance_started
	snookerClearedColors.value = Array.isArray(payload.snooker_cleared_colors) ? payload.snooker_cleared_colors : []
	snookerExpectedClearanceScore.value = payload.snooker_expected_clearance_score || 0
	snookerClearanceCompleted.value = !!payload.snooker_clearance_completed
}

const applyMatchSnapshot = (snapshot = {}) => {
	if (!snapshot || typeof snapshot !== 'object') {
		return
	}
	applyViewerCapabilities(snapshot)
	updateServerRevision(snapshot.server_revision)
	if (typeof snapshot.my_score === 'number') {
		myScore.value = snapshot.my_score
	}
	if (typeof snapshot.opponent_score === 'number') {
		opponentScore.value = snapshot.opponent_score
	}
	if (typeof snapshot.current_frame_started === 'boolean') {
		currentFrameStarted.value = snapshot.current_frame_started
	}
	if (typeof snapshot.current_frame_my_score === 'number') {
		currentFrameMyScore.value = snapshot.current_frame_my_score
	}
	if (typeof snapshot.current_frame_opponent_score === 'number') {
		currentFrameOpponentScore.value = snapshot.current_frame_opponent_score
	}
	if (typeof snapshot.current_round === 'number' && snapshot.current_round > 0) {
		currentRound.value = snapshot.current_round
	}
	applySnookerRoundState(snapshot)
}

const applyWriteResponse = (payload = {}) => {
	if (payload?.snapshot) {
		applyMatchSnapshot(payload.snapshot)
		return
	}
	applyViewerCapabilities(payload)
	updateServerRevision(payload?.server_revision)
	if (typeof payload?.my_score === 'number') {
		myScore.value = payload.my_score
	}
	if (typeof payload?.opponent_score === 'number') {
		opponentScore.value = payload.opponent_score
	}
	if (typeof payload?.current_frame_started === 'boolean') {
		currentFrameStarted.value = payload.current_frame_started
	}
	if (typeof payload?.current_frame_my_score === 'number') {
		currentFrameMyScore.value = payload.current_frame_my_score
	}
	if (typeof payload?.current_frame_opponent_score === 'number') {
		currentFrameOpponentScore.value = payload.current_frame_opponent_score
	}
	applySnookerRoundState(payload)
}

const getSnookerColorName = (score) => {
	switch (score) {
		case 2: return '黄球'
		case 3: return '绿球'
		case 4: return '咖啡球'
		case 5: return '蓝球'
		case 6: return '粉球'
		case 7: return '黑球'
		default: return ''
	}
}

const isSnookerScoreDisabled = (score) => {
	if (gameType.value !== 1) return false
	if (!currentFrameStarted.value) return true
	if (score === 1) return isSnookerRedDisabled.value
	if (!snookerClearanceStarted.value) return false
	if (snookerExpectedClearanceScore.value > 0) {
		return score !== snookerExpectedClearanceScore.value
	}
	return false
}

const navigateToResultOnce = () => {
	if (!consumeResultNavigationGuard(resultNavigationState)) return
	pageLog('跳转对局结果页', { matchId: matchId.value })
	uni.navigateTo({
		url: `/subPages/match/matchResult?match_id=${matchId.value}`
	})
}

/**
 * 显示同步loading（带超时自动关闭）
 * @param timeout - 超时时间，默认5秒
 */
const showSyncLoading = (timeout = 5000) => {
	if (isSyncing.value) return
	isSyncing.value = true
	uni.showLoading({ title: '正在同步比分...', mask: true })
	// 设置超时自动关闭
	if (syncTimeoutId) clearTimeout(syncTimeoutId)
	syncTimeoutId = setTimeout(() => {
		if (isSyncing.value) {
			isSyncing.value = false
			uni.hideLoading()
			console.warn('[MatchPlaying] 同步超时，准备请求快照', { matchId: matchId.value })
			matchWS.requestSync()
		}
	}, timeout)
}

/**
 * 关闭同步loading
 */
const hideSyncLoading = () => {
	if (syncTimeoutId) {
		clearTimeout(syncTimeoutId)
		syncTimeoutId = null
	}
	if (isSyncing.value) {
		isSyncing.value = false
		uni.hideLoading()
	}
}

/**
 * 处理分数更新消息
 * WebSocket推送的是原始视角(player1)的分数，需要根据当前用户视角转换
 */
const handleScoreUpdate = (data) => {
	if (!shouldApplyIncomingRevision(data?.server_revision)) {
		return
	}
	updateServerRevision(data?.server_revision)
	// 如果当前没有在同步，说明是对方发起的操作，显示短暂loading
	if (!isSyncing.value) {
		uni.showLoading({ title: '正在同步比分...', mask: false })
		setTimeout(() => uni.hideLoading(), 300)
	} else {
		// 收到分数更新，关闭同步loading
		hideSyncLoading()
	}
	applyServerScores(data.player1_score ?? data.my_score, data.player2_score ?? data.opponent_score)
	applyServerCurrentFrameScores(data.current_frame_player1_score, data.current_frame_player2_score)
	currentFrameStarted.value = data.current_frame_started !== false
	currentRound.value = data.current_round
	applySnookerRoundState(data)
	pageLog('收到比分更新', {
		matchId: data.match_id,
		currentRound: data.current_round,
		actionType: data.action_type,
		player1Score: data.player1_score ?? data.my_score,
		player2Score: data.player2_score ?? data.opponent_score
	})
}

/**
 * 处理局结束消息
 */
const handleRoundEnd = (data) => {
	if (!shouldApplyIncomingRevision(data?.server_revision)) {
		return
	}
	updateServerRevision(data?.server_revision)
	// 如果当前没有在同步，说明是对方发起的操作，显示短暂loading
	if (!isSyncing.value) {
		uni.showLoading({ title: '正在同步比分...', mask: false })
		setTimeout(() => uni.hideLoading(), 300)
	} else {
		// 收到局结束消息，关闭同步loading
		hideSyncLoading()
	}
	applyServerScores(data.player1_score ?? data.my_score, data.player2_score ?? data.opponent_score)
	applyServerCurrentFrameScores(data.current_frame_player1_score, data.current_frame_player2_score)
	currentFrameStarted.value = data.current_frame_started !== false
	currentRound.value = data.current_round
	applySnookerRoundState(data)
	pageLog('收到单局结束', {
		matchId: data.match_id,
		roundNumber: data.round_number,
		winner: data.winner,
		currentRound: data.current_round
	})
	
	// 根据当前用户视角判断是否是自己赢
	let isMyWin = false
	if (isPlayer1.value) {
		isMyWin = data.actor === 'me'
	} else {
		isMyWin = data.actor === 'opponent'
	}
	
	uni.showToast({
		title: viewerRole.value === 'referee'
			? `本局已判给${data.winner === 1 ? '选手1' : '选手2'}`
			: (isMyWin ? '你赢得本局' : '对手赢得本局'),
		icon: 'none'
	})
}

/**
 * 处理新局开始消息
 */
const handleRoundStart = (data) => {
	if (!shouldApplyIncomingRevision(data?.server_revision)) {
		return
	}
	updateServerRevision(data?.server_revision)
	applyServerCurrentFrameScores(data.current_frame_player1_score, data.current_frame_player2_score)
	currentFrameStarted.value = data.current_frame_started !== false
	currentRound.value = data.current_round
	applySnookerRoundState(data)
	pageLog('收到下一局开始', {
		matchId: data.match_id,
		currentRound: data.current_round,
		totalRounds: data.total_rounds
	})
	uni.showToast({
		title: `第${data.current_round}局开始`,
		icon: 'none'
	})
}

const handleSync = (data) => {
	if (!shouldApplyIncomingRevision(data?.server_revision)) {
		hideSyncLoading()
		return
	}
	applyViewerCapabilities(data)
	updateServerRevision(data?.server_revision)
	applyServerScores(data.player1_score, data.player2_score)
	applyServerCurrentFrameScores(data.current_frame_player1_score, data.current_frame_player2_score)
	currentFrameStarted.value = data.current_frame_started !== false
	currentRound.value = data.current_round || 1
	applySnookerRoundState(data)
	hideSyncLoading()
	pageLog('收到同步快照', {
		matchId: data.match_id,
		currentRound: data.current_round,
		totalRounds: data.total_rounds,
		status: data.status
	})
	if (data.status === 2) {
		navigateToResultOnce()
	}
}

const handleRoleChanged = async () => {
	const canStayOnPlayingPage = await loadMatchInfo()
	if (!canStayOnPlayingPage || isLeavingPlayingPage.value) {
		return
	}
	if (matchWS.isConnected()) {
		matchWS.requestSync()
	}
	uni.showToast({
		title: canScore.value ? '你已接管本场记分' : (viewerUi.value.readonlyHint || '本场已切换裁判记分'),
		icon: 'none'
	})
}

/**
 * 处理对局结束消息
 */
const handleMatchEnd = (data) => {
	if (!shouldApplyIncomingRevision(data?.server_revision)) {
		return
	}
	updateServerRevision(data?.server_revision)
	pageLog('收到对局结束消息', {
		matchId: data.match_id,
		status: data.status,
		player1Score: data.player1_score ?? data.my_score,
		player2Score: data.player2_score ?? data.opponent_score
	})
	navigateToResultOnce()
}

/**
 * 转换actor参数
 * 后端存储从player1视角：actor=1是player1，actor=2是player2
 * 前端操作从当前用户视角："我"的操作应该发送正确的actor
 */
const convertActor = (uiActor) => {
	// uiActor: 前端UI视角的actor，1=我方，2=对手方
	if (isPlayer1.value) {
		// 当前用户是player1，不需要转换
		return uiActor
	} else {
		// 当前用户是player2，需要反转
		// UI中的"我方"(actor=1) 实际上是后端的"对手方"(actor=2)
		// UI中的"对手方"(actor=2) 实际上是后端的"我方"(actor=1)
		return uiActor === 1 ? 2 : 1
	}
}

/**
 * 加分 (斯诺克模式 - 固定给对手加分)
 */
const handleAddScore = async (score) => {
	if (!ensureViewerCapability(canScore.value, '当前只有裁判可以记分')) return
	if (!matchId.value || isSyncing.value) return
	if (!currentFrameStarted.value) {
		uni.showToast({ title: '请先开始下一局', icon: 'none' })
		return
	}
	if (score === 1 && isSnookerRedDisabled.value) {
		uni.showToast({ title: '本局红球已打完，请改为彩球计分', icon: 'none' })
		return
	}
	if (score > 1 && isSnookerScoreDisabled(score)) {
		const nextText = nextSnookerClearanceColorName.value ? `请先记 ${nextSnookerClearanceColorName.value}` : `${getSnookerColorName(score)}当前不可计分`
		uni.showToast({ title: nextText, icon: 'none' })
		return
	}
	
		try {
			showSyncLoading()
			const res = await matchScore(buildActionRequest({
				match_id: matchId.value,
				actor: convertActor(2), // 固定给对手加分
				score: score
			}))
			if (!res?.success) {
				applyWriteResponse(res)
				hideSyncLoading()
				uni.showToast({ title: res?.message || '比分已更新，请重试', icon: 'none' })
				return
			}
			applyWriteResponse(res)
			hideSyncLoading()
			pageLog('加分HTTP成功', { matchId: matchId.value, score, serverRevision: res?.server_revision ?? res?.snapshot?.server_revision })
			uni.showToast({ title: `已给对手 +${score} 分`, icon: 'none' })
		} catch (error) {
		console.error('[MatchPlaying] 加分失败', { matchId: matchId.value, score, error })
		hideSyncLoading()
		uni.showToast({ title: error?.message || '操作失败', icon: 'none' })
	}
}

/**
 * 犯规 (九球模式)
 * @param actor - UI视角的actor，1=我方犯规，2=对手犯规
 */
const handleFoul = async (actor) => {
	if (!ensureViewerCapability(canScore.value, '当前只有裁判可以记分')) return
	if (!matchId.value || isSyncing.value) return
	
		try {
			showSyncLoading()
			const res = await matchFoul(buildActionRequest({
				match_id: matchId.value,
				actor: convertActor(actor)
			}))
			if (!res?.success) {
				applyWriteResponse(res)
				hideSyncLoading()
				uni.showToast({ title: res?.message || '比分已更新，请重试', icon: 'none' })
				return
			}
			applyWriteResponse(res)
			hideSyncLoading()
			pageLog('犯规HTTP成功', { matchId: matchId.value, actor, serverRevision: res?.server_revision ?? res?.snapshot?.server_revision })
			uni.showToast({ title: '对手犯规，我方 +1 分', icon: 'none' })
		} catch (error) {
		console.error('[MatchPlaying] 犯规失败', { matchId: matchId.value, actor, error })
		hideSyncLoading()
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const handleFoulByScore = async (score) => {
	if (!ensureViewerCapability(canScore.value, '当前只有裁判可以记分')) return
	if (!matchId.value || isSyncing.value) return

		try {
			showSyncLoading()
			const res = await matchFoul(buildActionRequest({
				match_id: matchId.value,
				actor: convertActor(2),
				score
			}))
			if (!res?.success) {
				applyWriteResponse(res)
				hideSyncLoading()
				uni.showToast({ title: res?.message || '比分已更新，请重试', icon: 'none' })
				return
			}
			applyWriteResponse(res)
			hideSyncLoading()
			pageLog('犯规加分HTTP成功', { matchId: matchId.value, score, serverRevision: res?.server_revision ?? res?.snapshot?.server_revision })
			uni.showToast({ title: `对手犯规，我方 +${score} 分`, icon: 'none' })
		} catch (error) {
		console.error('[MatchPlaying] 犯规加分失败', { matchId: matchId.value, score, error })
		hideSyncLoading()
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}



/**
 * 对手获胜
 */
const handleOpponentWin = async (winType, score = 1, winnerActor = 2) => {
	if (!ensureViewerCapability(canScore.value, '当前只有裁判可以记分')) return
	if (!matchId.value || isSyncing.value) return
	
		try {
			showSyncLoading()
			// 直接调用 endRound，传入获胜分数
			// 后端会根据 score 参数加分并记录操作日志
			const res = await endRound(buildActionRequest({
				match_id: matchId.value,
				winner: convertActor(winnerActor),
				win_type: winType,
				score: score // 获胜得分
			}))
			if (!res?.success) {
				applyWriteResponse(res)
				hideSyncLoading()
				uni.showToast({ title: res?.message || '比分已更新，请重试', icon: 'none' })
				return
			}
			applyWriteResponse(res)
			hideSyncLoading()
			pageLog('结束单局HTTP成功', { matchId: matchId.value, winType, score, roundNo: res.round_no, serverRevision: res?.server_revision ?? res?.snapshot?.server_revision })
			uni.showToast({ title: winnerActor === 1 ? '已判给我方' : '已判给对手', icon: 'none' })
		} catch (error) {
		console.error('[MatchPlaying] 结束单局失败', { matchId: matchId.value, winType, score, error })
		hideSyncLoading()
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

/**
 * 开始下一局
 */
const handleNextRound = async () => {
	if (!ensureViewerCapability(canScore.value, '当前只有裁判可以开始下一局')) return
	if (!matchId.value) return

	const nextFrameAction = resolveSnookerNextFrameAction({
		currentFrameStarted: currentFrameStarted.value,
		myFrameScore: currentFrameMyScore.value,
		opponentFrameScore: currentFrameOpponentScore.value
	})
	if (nextFrameAction.action === 'blocked') {
		uni.showToast({ title: nextFrameAction.message, icon: 'none' })
		return
	}

	const modalContent = nextFrameAction.action === 'settle_and_start_next_round'
		? `将按当前局比分 ${currentFrameMyScore.value}:${currentFrameOpponentScore.value} 自动判定${nextFrameAction.winner === 1 ? '我方' : '对手'}赢下本局，并开始下一局。`
		: '确定要开始新的一局吗？'

	uni.showModal({
		title: '开始下一局',
		content: modalContent,
		success: async (res) => {
			if (res.confirm) {
				try {
						if (nextFrameAction.action === 'settle_and_start_next_round') {
							const settleRes = await endRound(buildActionRequest({
								match_id: matchId.value,
								winner: convertActor(nextFrameAction.winner),
								win_type: 'normal',
								score: 1
							}))
							if (!settleRes?.success) {
								applyWriteResponse(settleRes)
								uni.showToast({ title: settleRes?.message || '本局结算失败', icon: 'none' })
								return
							}
							applyWriteResponse(settleRes)
						}

						const result = await startNextRound(buildActionRequest({
							match_id: matchId.value
						}))
						if (!result?.success) {
							applyWriteResponse(result)
							uni.showToast({ title: result?.message || '操作失败', icon: 'none' })
							return
						}
						applyWriteResponse(result)
						pageLog('开始下一局HTTP成功', { matchId: matchId.value, roundNo: result.round_no, serverRevision: result?.server_revision ?? result?.snapshot?.server_revision })
						uni.showToast({
							title: `第${result.round_no}局开始`,
							icon: 'none'
						})
					} catch (error) {
						console.error('[MatchPlaying] 开始下一局失败', { matchId: matchId.value, error })
						uni.showToast({ title: '操作失败', icon: 'none' })
				}
			}
		}
	})
}

/**
 * 撤销
 */
const handleUndo = async () => {
	if (!ensureViewerCapability(canUndo.value, '当前只有裁判可以撤销操作')) return
	if (!matchId.value) return
	
	try {
			const res = await matchUndo(buildActionRequest({
				match_id: matchId.value
			}))
			if (!res?.success) {
				applyWriteResponse(res)
				uni.showToast({ title: res?.message || '撤销失败', icon: 'none' })
				return
			}
			applyWriteResponse(res)
			matchWS.requestSync()
			pageLog('撤销HTTP成功', { matchId: matchId.value, serverRevision: res?.server_revision ?? res?.snapshot?.server_revision })
			uni.showToast({ title: res.message || '已撤销', icon: 'none' })
		} catch (error) {
		console.error('[MatchPlaying] 撤销失败', { matchId: matchId.value, error })
		uni.showToast({ title: '撤销失败', icon: 'none' })
	}
}

/**
 * 结束对局
 */
const handleFinishMatch = () => {
	if (!ensureViewerCapability(canFinish.value, '当前只有裁判可以结束对局')) return
	const finishAction = gameType.value === 1
		? resolveSnookerFinishMatchAction({
			currentFrameStarted: currentFrameStarted.value,
			myFrameScore: currentFrameMyScore.value,
			opponentFrameScore: currentFrameOpponentScore.value
		})
		: { action: 'finish_match' }
	if (finishAction.action === 'blocked') {
		uni.showToast({ title: finishAction.message, icon: 'none' })
		return
	}

	const finishContent = finishAction.action === 'settle_and_finish_match'
		? `确定要结束本次对局吗？将按当前局比分 ${currentFrameMyScore.value}:${currentFrameOpponentScore.value} 自动判定${finishAction.winner === 1 ? '我方' : '对手'}赢下本局后，再结束整场对局。`
		: '确定要结束本次对局吗？'
	uni.showModal({
		title: '确认结束',
		content: finishContent,
		success: async (res) => {
			if (res.confirm) {
				try {
					showSyncLoading()
					if (finishAction.action === 'settle_and_finish_match') {
						const settleRes = await endRound(buildActionRequest({
							match_id: matchId.value,
							winner: convertActor(finishAction.winner),
							win_type: 'normal',
							score: 1
						}))
						if (!settleRes?.success) {
							applyWriteResponse(settleRes)
							hideSyncLoading()
							uni.showToast({ title: settleRes?.message || '本局结算失败', icon: 'none' })
							return
						}
						applyWriteResponse(settleRes)
					}

					const res = await finishMatch(buildActionRequest({
						match_id: matchId.value
					}))
					if (!res?.success) {
						applyWriteResponse(res)
						hideSyncLoading()
						uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
						return
					}
					applyWriteResponse(res)
					hideSyncLoading()
					pageLog('结束对局HTTP成功', { matchId: matchId.value })
					navigateToResultOnce()
				} catch (error) {
					hideSyncLoading()
					console.error('[MatchPlaying] 结束对局失败', { matchId: matchId.value, error })
					uni.showToast({ title: '操作失败', icon: 'none' })
				}
			}
		}
	})
}

/**
 * 返回
 */
const handleBack = () => {
	uni.showModal({
		title: '提示',
		content: '对局进行中，确定要退出吗？',
		success: (res) => {
			if (res.confirm) {
				uni.navigateBack()
			}
		}
	})
}

/**
 * 显示菜单
 */
const showMenu = () => {
	uni.showActionSheet({
		itemList: ['对局设置', '放弃对局'],
		success: (res) => {
			if (res.tapIndex === 1) {
				handleFinishMatch()
			}
		}
	})
}
</script>

<style lang="scss" scoped>
@import './playing.scss';
</style>
