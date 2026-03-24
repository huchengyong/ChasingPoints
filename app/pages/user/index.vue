<template>
	<view class="user-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="main-content">
			<template v-if="!isLoggedIn">
				<view class="guest-hero">
					<view class="guest-hero-copy">
						<text class="guest-eyebrow">{{ guestHeroCopy.eyebrow }}</text>
						<text class="guest-title">{{ guestHeroCopy.title }}</text>
						<text class="guest-description">{{ guestHeroCopy.description }}</text>
					</view>
					<view class="guest-hero-actions">
						<button class="hero-btn hero-btn-primary" @click="handleGoLogin">
							<text>{{ guestHeroCopy.primaryActionText }}</text>
						</button>
						<button class="hero-btn hero-btn-secondary" @click="openRoute('/pages/ranking/index')">
							<text>{{ guestHeroCopy.secondaryActionText }}</text>
						</button>
					</view>
				</view>

				<view class="section-block">
					<view class="section-header">
						<text class="section-title">登录后解锁</text>
					</view>
					<view class="benefits-grid">
						<view v-for="item in guestBenefits" :key="item.title" class="benefit-card">
							<view class="benefit-icon">
									<uni-icons :type="item.icon" size="20" color="#E0AE12"></uni-icons>
							</view>
							<text class="benefit-title">{{ item.title }}</text>
							<text class="benefit-desc">{{ item.desc }}</text>
						</view>
					</view>
				</view>

				<view class="section-block">
					<view class="section-header">
						<text class="section-title">登录后你会看到</text>
					</view>
					<view class="preview-card">
						<view class="preview-top">
							<view class="preview-badge">
								<text>示例</text>
							</view>
							<text class="preview-status">登录后解锁</text>
						</view>
						<text class="preview-title">你的段位、状态和待处理事项会在这里慢慢成型</text>
						<text class="preview-desc">登录后，你会看到进行中的对局、竞技概览、荣誉墙和待处理提醒，不再错过任何一场比赛。</text>
						<view class="preview-modules">
							<view v-for="item in previewModules" :key="item.title" class="preview-module">
								<text class="preview-module-title">{{ item.title }}</text>
								<text class="preview-module-desc">{{ item.desc }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="section-block">
					<view class="section-header">
						<text class="section-title">游客也能继续看</text>
					</view>
					<view class="explore-list">
						<view class="explore-item" @click="openRoute('/pages/ranking/index')">
							<view class="explore-copy">
								<text class="explore-title">排行榜</text>
								<text class="explore-desc">先看看平台高手的段位与积分</text>
							</view>
							<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
						</view>
						<view class="explore-item" @click="openRoute('/pages/match/index', true)">
							<view class="explore-copy">
								<text class="explore-title">正在进行中的对局</text>
								<text class="explore-desc">围观平台实时比赛和比分进展</text>
							</view>
							<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
						</view>
						<view class="explore-item" @click="openRoute('/pages/social/index', true)">
							<view class="explore-copy">
								<text class="explore-title">公开动态</text>
								<text class="explore-desc">看看大家分享的战绩与打球日常</text>
							</view>
							<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
						</view>
					</view>
				</view>
			</template>

			<template v-else>
				<view class="identity-hero">
					<view class="identity-top">
						<view class="identity-main">
							<view class="identity-avatar">
								<image :src="userInfo.avatar || '/static/images/default-avatar.png'" mode="aspectFill" />
							</view>
							<view class="identity-copy">
								<view class="identity-name-row">
									<text class="identity-name">{{ userInfo.nickname }}</text>
									<view class="identity-rank-chip" @click="handleRankExplain">
										<uni-icons type="star-filled" size="12" color="#f59e0b"></uni-icons>
										<text>当前段位</text>
									</view>
								</view>
								<text class="identity-id">ID: {{ userInfo.id }}</text>
								<text class="identity-summary">{{ identitySummary }}</text>
							</view>
						</view>
						<view class="identity-actions">
							<button class="icon-btn" @click="handleQrCode">
								<uni-icons fontFamily="CustomFont" size="16" color="#e2e8f0">{{ '\&#xe604;' }}</uni-icons>
							</button>
							<button class="icon-btn" @click="handleSettings">
								<uni-icons type="gear" size="20" color="#e2e8f0"></uni-icons>
							</button>
						</view>
					</view>
					<view class="identity-rank-row" @click="handleRankExplain">
						<image class="identity-rank-icon" :src="displayRank.icon" mode="aspectFit"></image>
						<view class="identity-rank-copy">
							<text class="identity-rank-title">{{ currentRankGameLabel }} · {{ displayRank.name }}</text>
							<text class="identity-rank-meta">当前积分 {{ displayRank.score }} · {{ rankProgressText }}</text>
						</view>
						<uni-icons type="right" size="18" color="#ffffff"></uni-icons>
					</view>
					<view class="identity-mode-tabs">
						<view
							v-for="item in rankGameTabs"
							:key="item.value"
							class="identity-mode-tab"
							:class="{ active: currentRankGameType === item.value }"
							@tap.stop="handleRankGameTypeChange(item.value)"
						>
							<text>{{ item.label }}</text>
						</view>
					</view>
				</view>

				<view class="status-card">
					<text class="status-eyebrow">{{ statusCard.eyebrow }}</text>
					<text class="status-title">{{ statusCard.title }}</text>
					<text class="status-description">{{ statusCard.description }}</text>
					<view class="status-actions">
						<button class="status-btn status-btn-primary" @click="handleStatusAction(statusCard.action)">
							<text>{{ statusCard.actionText }}</text>
						</button>
						<button
							v-if="statusCard.secondaryActionText"
							class="status-btn status-btn-secondary"
							@click="handleStatusAction(statusCard.secondaryAction)"
						>
							<text>{{ statusCard.secondaryActionText }}</text>
						</button>
					</view>
				</view>

				<view class="section-block">
					<view class="section-header">
						<text class="section-title">{{ sectionTitles.stats }}</text>
						<text class="section-hint">先看状态，再决定下一步</text>
					</view>
					<view class="metrics-grid">
						<view v-for="item in metricCards" :key="item.label" class="metric-card">
							<text class="metric-card-value" :class="{ accent: item.accent }">{{ item.value }}</text>
							<text class="metric-card-label">{{ item.label }}</text>
							<text class="metric-card-desc">{{ item.desc }}</text>
						</view>
					</view>
				</view>

				<view class="section-block">
					<view class="section-header">
						<text class="section-title">{{ sectionTitles.quickActions }}</text>
					</view>
					<view class="quick-grid">
						<view v-for="item in quickActions" :key="item.label" class="quick-card" @click="item.handler">
							<view class="quick-card-top">
								<view class="quick-icon" :class="item.iconClass">
									<uni-icons :type="item.icon" size="20" :color="item.iconColor"></uni-icons>
								</view>
								<view v-if="item.badge" class="quick-badge">
									<text>{{ item.badge }}</text>
								</view>
							</view>
							<text class="quick-title">{{ item.label }}</text>
							<text class="quick-desc">{{ item.desc }}</text>
						</view>
					</view>
				</view>

				<view class="section-block section-block-light">
					<view class="section-header">
						<text class="section-title">{{ sectionTitles.secondaryServices }}</text>
						<text class="section-hint">按需查看</text>
					</view>
					<view class="service-list">
						<view class="service-item" @click="openRoute('/subPages/tournament/index')">
							<text>赛事中心</text>
							<uni-icons type="right" size="16" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
						</view>
						<view class="service-item" @click="openRoute('/subPages/season/index')">
							<text>赛季档案</text>
							<uni-icons type="right" size="16" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
						</view>
						<view class="service-item" @click="openRoute('/subPages/venue/index')">
							<text>球房与签到</text>
							<uni-icons type="right" size="16" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
						</view>
					</view>
				</view>
			</template>

			<view class="settings-section">
				<view class="settings-card">
					<text class="settings-title">{{ sectionTitles.settings }}</text>
					<view v-if="isLoggedIn" class="settings-item settings-item-switch" @click="toggleHideMatch">
						<view class="settings-copy">
							<text class="settings-label">隐藏战绩</text>
							<text class="settings-desc">控制是否在公开场景中展示你的战绩</text>
						</view>
						<view class="switch-wrapper">
							<view class="switch-track" :class="{ active: isHideMatch }">
								<view class="switch-thumb" :class="{ active: isHideMatch }"></view>
							</view>
						</view>
					</view>
					<view v-if="isLoggedIn" class="settings-item" @click="handleSettings">
						<view class="settings-copy">
							<text class="settings-label">账号设置</text>
							<text class="settings-desc">修改昵称、查看账号信息与基础配置</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					</view>
					<view class="settings-item" @click="handleHelp">
						<view class="settings-copy">
							<text class="settings-label">帮助与反馈</text>
							<text class="settings-desc">提交问题、建议或获取使用帮助</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					</view>
					<view class="settings-item" @click="handlePrivacy">
						<view class="settings-copy">
							<text class="settings-label">隐私政策</text>
							<text class="settings-desc">查看平台隐私与数据使用说明</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					</view>
					<view class="settings-item" @click="handleAgreement">
						<view class="settings-copy">
							<text class="settings-label">用户协议</text>
							<text class="settings-desc">查看平台使用规则与服务条款</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					</view>
					<button v-if="isLoggedIn" class="logout-btn" @click="handleLogout">
						<text>退出登录</text>
					</button>
				</view>
			</view>
		</view>

		<view v-if="showQrCodeModal" class="qrcode-modal-overlay" @click="closeQrCodeModal">
			<view class="qrcode-modal-container" @click.stop>
				<view class="qrcode-modal-header">
					<text class="qrcode-modal-title">我的二维码</text>
					<view class="qrcode-modal-close" @click="closeQrCodeModal">
						<uni-icons type="closeempty" size="24" :color="isDarkMode ? '#94a3b8' : '#64748b'"></uni-icons>
					</view>
				</view>
				<view class="qrcode-modal-body">
					<view v-show="qrcodeLoading" class="qrcode-loading">
						<uni-icons type="spinner-cycle" size="48" color="#E0AE12"></uni-icons>
						<text class="qrcode-loading-text">生成中...</text>
					</view>
					<view v-show="!qrcodeLoading" class="qrcode-display">
						<image
							v-if="qrcodeUrl"
							:src="qrcodeUrl"
							mode="aspectFit"
							class="qrcode-image"
							@load="onQRCodeLoad"
							@error="onQRCodeError"
						/>
						<uni-icons v-else type="checkbox" size="160" color="#27272a"></uni-icons>
					</view>
					<image
						v-if="qrcodeUrl && qrcodeLoading"
						:src="qrcodeUrl"
						style="width: 0; height: 0; position: absolute; opacity: 0;"
						@load="onQRCodeLoad"
						@error="onQRCodeError"
					/>
				</view>
				<view class="qrcode-modal-footer">
					<text class="qrcode-hint">让对手扫描你的二维码进行匹配</text>
				</view>
			</view>
		</view>

		<gameTypeModal
			v-model:visible="showGameTypeModal"
			@confirm="handleGameTypeConfirm"
		/>
	</view>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
import { getUserStats } from '@/api/user.js'
import { getCurrentMatch, getMatchQRCode, startMatch } from '@/api/match.js'
import { getUserRankInfo } from '@/api/rank.js'
import { userWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'
import {
	resolveGuestHeroCopy,
	resolveSectionTitles,
	resolveStatusCardContent,
	resolveUserHomepageMode
} from '@/utils/user-homepage.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { useNotificationStore } from '@/store/notification.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'

const userStore = useUserStore()
const themeStore = useThemeStore()
const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()

const guestBenefits = [
	{ icon: 'flag-filled', title: '记录真实比分', desc: '每一场 PK 都能沉淀为你的个人竞技数据。' },
	{ icon: 'bars', title: '查看竞技画像', desc: '从胜率、连胜、对手记录观察你的状态变化。' },
	{ icon: 'chat', title: '接收待处理', desc: '挑战、好友申请和通知会集中提醒。' },
	{ icon: 'star-filled', title: '冲击更高段位', desc: '在个人主页里持续追踪段位和积分。' }
]

const previewModules = [
	{ title: '当前段位', desc: '段位、积分和下一段位进度会统一展示' },
	{ title: '竞技概览', desc: '胜率、连胜和最近状态会更清楚地反馈给你' },
	{ title: '待处理', desc: '消息、好友申请和挑战提醒会集中汇总' }
]

const guestHeroCopy = resolveGuestHeroCopy()
const sectionTitles = resolveSectionTitles()

const isDarkMode = computed(() => themeStore.isDarkMode)
const isHideMatch = ref(false)
const showGameTypeModal = ref(false)
const showQrCodeModal = ref(false)
const qrcodeLoading = ref(false)
const qrcodeUrl = ref('')
const selectedGameType = ref(null)
const currentRankGameType = ref(3)
const currentMatch = ref(null)
const rankInfo = ref(null)
const rankGameTabs = GAME_TYPE_TABS

const isLoggedIn = computed(() => userStore.isLoggedIn)
const pendingTotal = computed(() => notificationStore.unreadCount + friendRequestStore.pendingCount)
const pendingSummary = computed(() => {
	if (pendingTotal.value <= 0) return '当前没有未处理事项'

	const parts = []
	if (notificationStore.unreadCount > 0) {
		parts.push(`${notificationStore.unreadCount} 条消息`)
	}
	if (friendRequestStore.pendingCount > 0) {
		parts.push(`${friendRequestStore.pendingCount} 个好友申请`)
	}

	return parts.join(' · ')
})

const userInfo = computed(() => ({
	id: userStore.userInfo?.id || 0,
	phone: userStore.userInfo?.phone || '',
	nickname: userStore.userInfo?.nickname || '用户',
	avatar: userStore.userInfo?.avatar || ''
}))

const userStats = reactive({
	totalMatches: 0,
	wins: 0,
	losses: 0,
	winRate: 0,
	maxStreak: 0,
	loading: false
})

const displayRank = computed(() => ({
	name: rankInfo.value?.name || '未定级',
	icon: rankInfo.value?.icon || '/static/images/ranks/rank_bronze.png',
	score: rankInfo.value?.rank_score || 0,
	progress: rankInfo.value?.progress || 0,
	nextName: rankInfo.value?.next_name || '下一段位',
	nextScore: rankInfo.value?.next_score || 0
}))
const currentRankGameLabel = computed(() => rankGameTabs.find(item => item.value === currentRankGameType.value)?.label || '中式八球')

const hasRecentMatch = computed(() => userStats.totalMatches > 0)
const homepageMode = computed(() => resolveUserHomepageMode({
	isLoggedIn: isLoggedIn.value,
	hasCurrentMatch: Boolean(currentMatch.value),
	hasRecentMatch: hasRecentMatch.value
}))

const recentMatchCard = computed(() => {
	if (!hasRecentMatch.value) return null

	return {
		title: userStats.winRate >= 50 ? '最近手感不错，继续保持' : '上一场先放下，这一局重新打回来',
		description: `累计 ${userStats.totalMatches} 场对局 · 胜率 ${userStats.winRate}% · 最高连胜 ${userStats.maxStreak}`
	}
})

const statusCard = computed(() => resolveStatusCardContent({
	mode: homepageMode.value,
	currentMatch: currentMatch.value,
	recentMatch: recentMatchCard.value
}))

const identitySummary = computed(() => {
	if (homepageMode.value === 'ongoing' && currentMatch.value) {
		return `你和 ${currentMatch.value.opponent_name || '对手'} 的比赛还在继续，别让手感断掉`
	}
	if (hasRecentMatch.value) {
		return `近阶段完成 ${userStats.totalMatches} 场比赛，当前胜率 ${userStats.winRate}%`
	}
	return '这周还没开杆，来打一场找回手感'
})

const rankProgressText = computed(() => {
	if (rankInfo.value?.level >= 5) return '已达到最高段位'
	if (!rankInfo.value) return '完成首场比赛后开始计算积分'
	return `进度 ${displayRank.value.progress}% · 下一段位 ${displayRank.value.nextName}`
})

const metricCards = computed(() => ([
	{
		label: '胜率',
		value: `${userStats.winRate}%`,
		desc: hasRecentMatch.value ? '真实对局带来的即时反馈' : '完成首场比赛后开始统计',
		accent: true
	},
	{
		label: '总胜场',
		value: userStats.wins,
		desc: hasRecentMatch.value ? `总对局 ${userStats.totalMatches} 场` : '下一场会从这里开始',
		accent: false
	},
	{
		label: '最高连胜',
		value: userStats.maxStreak,
		desc: userStats.maxStreak > 0 ? '继续保持你的连胜手感' : '下一波连胜等你开启',
		accent: false
	},
	{
		label: '段位积分',
		value: displayRank.value.score,
		desc: rankInfo.value ? `当前处于 ${displayRank.value.name}` : '完成首场比赛后开始定级',
		accent: false
	}
]))

const quickActions = computed(() => ([
	{
		label: '比赛记录',
		desc: hasRecentMatch.value ? `累计 ${userStats.totalMatches} 场` : '查看历史对局',
		icon: 'list',
		iconColor: '#E0AE12',
		iconClass: 'blue',
		handler: handleMatchHistory
	},
	{
		label: '对手记录',
		desc: '回看你和不同对手的交锋结果',
		icon: 'contact',
		iconColor: '#f59e0b',
		iconClass: 'amber',
		handler: handleOpponentRecord
	},
	{
		label: '竞技分析',
		desc: '查看更完整的竞技画像',
		icon: 'bars',
		iconColor: '#E0AE12',
		iconClass: 'green',
		handler: handleStatsDetail
	},
	{
		label: '荣誉墙',
		desc: '查看你的成就与阶段里程碑',
		icon: 'medal',
		iconColor: '#8b5cf6',
		iconClass: 'purple',
		handler: handleAchievement
	},
	{
		label: '待处理',
		desc: pendingSummary.value,
		icon: 'chat',
		iconColor: '#ef4444',
		iconClass: 'rose',
		handler: handleNotificationCenter,
		badge: pendingTotal.value > 99 ? '99+' : (pendingTotal.value || '')
	},
	{
		label: '好友对局',
		desc: '管理好友，发起 PK 或查看报表对比',
		icon: 'person-filled',
		iconColor: '#0f766e',
		iconClass: 'teal',
		handler: handleFriendList
	}
]))

onMounted(() => {
	loadHideMatchPreference()
})

onShow(() => {
	if (isLoggedIn.value) {
		loadHomepageData()
		connectUserWS()
	} else {
		resetHomepageState()
		notificationStore.clearUnread()
		friendRequestStore.clearPendingCount()
	}

	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	uni.$on(THEME_CHANGE_EVENT, handleThemeChange)
})

onHide(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
	disconnectUserWS()
})

onUnmounted(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
	disconnectUserWS()
})

const handleThemeChange = () => {}

const loadHomepageData = async () => {
	await Promise.all([
		notificationStore.fetchUnreadCount(),
		friendRequestStore.fetchPendingCount(),
		loadUserStats(),
		loadRankInfo(),
		loadCurrentMatch()
	])
}

const loadUserStats = async () => {
	if (userStats.loading) return

	userStats.loading = true
	try {
		const res = await getUserStats()
		if (res.success) {
			userStats.totalMatches = res.total_matches || 0
			userStats.wins = res.wins || 0
			userStats.losses = res.losses || 0
			userStats.winRate = res.win_rate || 0
			userStats.maxStreak = res.max_win_streak || 0
		}
	} catch (error) {
		console.error('获取用户统计失败:', error)
	} finally {
		userStats.loading = false
	}
}

const loadRankInfo = async () => {
	try {
		const res = await getUserRankInfo({ game_type: currentRankGameType.value })
		rankInfo.value = res.success ? res.rank_info || null : null
	} catch (error) {
		console.error('获取段位信息失败:', error)
		rankInfo.value = null
	}
}

const loadCurrentMatch = async () => {
	try {
		const res = await getCurrentMatch()
		currentMatch.value = res.success && res.match ? res.match : null
	} catch (error) {
		console.error('获取当前对局失败:', error)
		currentMatch.value = null
	}
}

const resetHomepageState = () => {
	currentMatch.value = null
	rankInfo.value = null
	userStats.totalMatches = 0
	userStats.wins = 0
	userStats.losses = 0
	userStats.winRate = 0
	userStats.maxStreak = 0
}

const connectUserWS = async () => {
	try {
		await userWS.connect()
		userWS.off(WS_MESSAGE_TYPES.MATCH_START, handleMatchStart)
		userWS.off(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
		userWS.on(WS_MESSAGE_TYPES.MATCH_START, handleMatchStart)
		userWS.on(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
	} catch (error) {
		console.error('[UserPage] 用户WS连接失败:', error)
	}
}

const disconnectUserWS = () => {
	userWS.off(WS_MESSAGE_TYPES.MATCH_START, handleMatchStart)
	userWS.off(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
	userWS.disconnect()
}

const handleNotificationUpdate = () => {
	notificationStore.fetchUnreadCount()
	friendRequestStore.fetchPendingCount()
}

const handleMatchStart = (data) => {
	if (showQrCodeModal.value) {
		showQrCodeModal.value = false
	}

	uni.showToast({
		title: `${data.opponent_name || '对手'} 向你发起了对局`,
		icon: 'none',
		duration: 2000
	})

	setTimeout(() => {
		uni.navigateTo({
			url: `/subPages/match/playing?match_id=${data.match_id}&game_type=${data.game_type}&opponent_name=${encodeURIComponent(data.opponent_name || '对手')}&opponent_avatar=${encodeURIComponent(data.opponent_avatar || '')}`
		})
	}, 1000)
}

const openRoute = (url, isTab = false) => {
	if (isTab) {
		uni.switchTab({ url })
		return
	}
	uni.navigateTo({ url })
}

const handleStatusAction = (action) => {
	if (action === 'continue') {
		handleContinueCurrentMatch()
		return
	}
	if (action === 'login') {
		handleGoLogin()
		return
	}
	if (action === 'ranking') {
		openRoute('/pages/ranking/index')
		return
	}
	if (action === 'history' || action === 'recent') {
		handleMatchHistory()
		return
	}
	handleStartPK()
}

const handleGoLogin = () => {
	uni.navigateTo({
		url: '/pages/login/login'
	})
}

const handleStartPK = () => {
	if (!isLoggedIn.value) {
		handleGoLogin()
		return
	}
	showGameTypeModal.value = true
}

const handleGameTypeConfirm = (gameType) => {
	selectedGameType.value = gameType
	handleScanCode()
}

const handleScanCode = () => {
	// #ifdef APP-PLUS || APP-HARMONY
	uni.scanCode({
		scanType: ['qrCode'],
		success: (res) => {
			handleMatchResult(res.result)
		},
		fail: (err) => {
			console.error('扫码失败:', err)
		}
	})
	// #endif
}

const handleMatchResult = async (scanResult) => {
	try {
		const opponentData = JSON.parse(scanResult)

		if (!opponentData.user_id) {
			uni.showToast({ title: '无效的二维码', icon: 'none' })
			return
		}

		uni.showLoading({ title: '匹配中...', mask: true })

		const res = await startMatch({
			game_type: selectedGameType.value,
			opponent_id: opponentData.user_id,
			opponent_name: opponentData.nickname || '对手'
		})

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

const handleContinueCurrentMatch = () => {
	if (!currentMatch.value) {
		handleStartPK()
		return
	}

	navigateToPlayingMatch({
		match_id: currentMatch.value.id,
		game_type: currentMatch.value.game_type || 3,
		opponent_name: currentMatch.value.opponent_name || '对手',
		opponent_avatar: currentMatch.value.opponent_avatar || ''
	})
}

const handleMatchHistory = () => {
	uni.navigateTo({ url: '/subPages/user/matchHistory' })
}

const handleOpponentRecord = () => {
	uni.navigateTo({ url: '/subPages/user/opponentRecord' })
}

const handleRankExplain = () => {
	uni.navigateTo({ url: `/subPages/user/rankExplain?game_type=${currentRankGameType.value}` })
}

const handleAchievement = () => {
	uni.navigateTo({ url: '/subPages/achievement/index' })
}

const handleFriendList = () => {
	uni.navigateTo({ url: '/subPages/social/friendList' })
}

const handleStatsDetail = () => {
	uni.navigateTo({ url: `/subPages/user/statsDetail?game_type=${currentRankGameType.value}` })
}

const handleRankGameTypeChange = (gameType) => {
	if (currentRankGameType.value === gameType) return
	currentRankGameType.value = gameType
	loadRankInfo()
}

const handleNotificationCenter = () => {
	uni.navigateTo({ url: '/subPages/notification/index' })
}

const loadHideMatchPreference = () => {
	const savedHideMatch = uni.getStorageSync('user_hide_match')
	if (savedHideMatch !== '' && savedHideMatch !== undefined) {
		isHideMatch.value = savedHideMatch
	}
}

const toggleHideMatch = () => {
	isHideMatch.value = !isHideMatch.value
	uni.setStorageSync('user_hide_match', isHideMatch.value)
	userStore.setHideMatch && userStore.setHideMatch(isHideMatch.value)
}

const handlePrivacy = () => {
	uni.navigateTo({ url: '/subPages/agreement/privacyPolicy' })
}

const handleAgreement = () => {
	uni.navigateTo({ url: '/subPages/agreement/userAgreement' })
}

const handleHelp = () => {
	uni.navigateTo({ url: '/subPages/help/feedback' })
}

const handleSettings = () => {
	uni.navigateTo({ url: '/subPages/user/settings' })
}

const handleQrCode = () => {
	showQrCodeModal.value = true
	generateQRCode()
}

const closeQrCodeModal = () => {
	showQrCodeModal.value = false
}

const generateQRCode = async () => {
	if (!isLoggedIn.value) return

	qrcodeLoading.value = true
	try {
		const res = await getMatchQRCode()
		if (res.success && res.qrcode_data) {
			qrcodeUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(res.qrcode_data)}`
		} else {
			qrcodeLoading.value = false
		}
	} catch (error) {
		console.error('生成二维码失败:', error)
		qrcodeLoading.value = false
	}
}

const onQRCodeLoad = () => {
	qrcodeLoading.value = false
}

const onQRCodeError = () => {
	console.error('二维码图片加载失败')
	qrcodeLoading.value = false
}

const handleLogout = () => {
	uni.showModal({
		title: '提示',
		content: '确定要退出登录吗？',
		success: (res) => {
			if (res.confirm) {
				userStore.logout()
				resetHomepageState()
				uni.showToast({
					title: '已退出登录',
					icon: 'success'
				})
			}
		}
	})
}
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
