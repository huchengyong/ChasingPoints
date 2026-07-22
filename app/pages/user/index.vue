<template>
	<view class="user-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="main-content">
			<template v-if="!isLoggedIn">
				<view class="guest-hero">
					<text class="guest-eyebrow">{{ guestHeroCopy.eyebrow }}</text>
					<text class="guest-title">{{ guestHeroCopy.title }}</text>
					<text class="guest-description">{{ guestHeroCopy.description }}</text>
					<view class="guest-actions">
						<button class="hero-btn hero-btn-primary" @click="handleGoLogin">
							<text>{{ guestHeroCopy.primaryActionText }}</text>
						</button>
						<button class="hero-btn hero-btn-secondary" @click="handleGuestStartPK">
							<text>{{ guestHeroCopy.secondaryActionText }}</text>
						</button>
					</view>
				</view>

				<view class="guest-card">
					<text class="guest-card-title">登录后解锁</text>
					<view class="guest-benefits">
						<view v-for="item in guestBenefits" :key="item.title" class="guest-benefit">
							<view class="guest-benefit-icon">
								<uni-icons :type="item.icon" size="20" color="#E0AE12"></uni-icons>
							</view>
							<text class="guest-benefit-title">{{ item.title }}</text>
							<text class="guest-benefit-desc">{{ item.desc }}</text>
						</view>
					</view>
				</view>

				<view class="guest-card guest-links-card">
					<text class="guest-card-title">游客也能继续看</text>
					<view class="guest-link" @click="openRoute('/pages/ranking/index')">
						<view class="guest-link-copy">
							<text class="guest-link-title">排行榜</text>
							<text class="guest-link-desc">查看平台高手的段位与积分</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
					</view>
					<view class="guest-link" @click="openRoute('/pages/match/index', true)">
						<view class="guest-link-copy">
							<text class="guest-link-title">正在进行的对局</text>
							<text class="guest-link-desc">围观实时比赛与比分进展</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
					</view>
					<view class="guest-link" @click="openRoute('/pages/social/index', true)">
						<view class="guest-link-copy">
							<text class="guest-link-title">赛讯</text>
							<text class="guest-link-desc">查看赛事资讯和赛程更新</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
					</view>
					<view class="guest-link guest-support-link" @click="handleHelp">
						<view class="guest-link-copy">
							<text class="guest-link-title">帮助、投诉与举报</text>
							<text class="guest-link-desc">无需登录即可提交问题与反馈</text>
						</view>
						<uni-icons type="right" size="18" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
					</view>
				</view>
			</template>

			<template v-else>
				<view class="profile-header">
					<view class="profile-avatar" :class="{ 'has-rank-frame': rankAvatarFrame }" @click="handleEditProfile">
						<image class="profile-avatar-image" :src="userInfo.avatar" mode="aspectFill"></image>
						<image v-if="rankAvatarFrame" class="profile-avatar-frame" :src="rankAvatarFrame" mode="aspectFit"></image>
					</view>
					<view class="profile-copy">
						<text class="profile-name">{{ userInfo.nickname }}</text>
						<view class="profile-chips">
							<view class="profile-chip" @click="handleOpenMemberCenter">
								<text>{{ profileLevelText }}</text>
							</view>
							<view class="profile-chip" @click="handleOpenReputation">
								<text>{{ reputationEntryStatusText }}</text>
							</view>
						</view>
					</view>
					<view class="profile-actions">
						<button class="icon-btn" @click="handleNotificationCenter">
							<uni-icons type="notification-filled" size="20" :color="isDarkMode ? '#f7e7a8' : '#475569'"></uni-icons>
							<view v-if="pendingTotal > 0" class="icon-btn-badge">
								<text>{{ pendingBadgeText }}</text>
							</view>
						</button>
						<button class="icon-btn" @click="handleSettings">
							<uni-icons type="gear" size="21" :color="isDarkMode ? '#f7e7a8' : '#475569'"></uni-icons>
						</button>
					</view>
				</view>

				<view class="identity-card" :class="{ 'member-active': isActiveMember }">
					<view
						v-if="memberHeroStrip.visible"
						class="member-hero-strip"
						:class="`member-${memberHeroStrip.state}`"
						@click="handleOpenMemberCenter"
					>
						<view class="member-hero-copy">
							<text class="member-hero-title">{{ memberHeroStrip.title }}</text>
							<text class="member-hero-level">{{ memberHeroStrip.levelText }}</text>
						</view>
						<text class="member-hero-description">{{ memberHeroStrip.description }}</text>
					</view>

					<view v-if="!hasRankCache && (coreDataLoading || rankLoading)" class="rank-skeleton">
						<view class="skeleton-circle"></view>
						<view class="rank-skeleton-copy">
							<view class="skeleton-line skeleton-line-wide"></view>
							<view class="skeleton-line skeleton-line-short"></view>
						</view>
					</view>
					<view v-else class="rank-row" @click="handleRankExplain">
						<image class="rank-icon" :src="displayRank.icon" mode="aspectFit"></image>
						<view class="rank-copy">
							<text class="rank-title">{{ currentRankGameLabel }} · {{ displayRank.name }}</text>
							<text class="rank-meta">排位分 {{ displayRank.score }} · {{ rankProgressText }}</text>
						</view>
						<uni-icons type="right" size="18" color="#ffffff"></uni-icons>
					</view>

					<view class="rank-tabs">
						<view
							v-for="item in rankGameTabs"
							:key="item.value"
							class="rank-tab"
							:class="{ active: currentRankGameType === item.value }"
							@tap="handleRankGameTypeChange(item.value)"
						>
							<text>{{ item.label }}</text>
						</view>
					</view>
				</view>

				<view class="status-card" :class="{ ongoing: statusCard.type === 'ongoing', loading: statusCard.loading }">
					<template v-if="statusCard.loading">
						<view class="status-skeleton-head">
							<view class="skeleton-line skeleton-line-medium"></view>
							<view class="skeleton-pill"></view>
						</view>
						<view class="status-skeleton-body">
							<view class="skeleton-circle skeleton-player"></view>
							<view class="skeleton-score"></view>
							<view class="skeleton-circle skeleton-player"></view>
						</view>
						<view class="skeleton-button"></view>
					</template>

					<template v-else>
						<view class="status-heading">
							<text class="status-title">{{ statusCard.title }}</text>
							<view class="status-pill" :class="{ referee: homepageMode === 'ongoing-referee' }">
								<text>{{ statusCard.statusText }}</text>
							</view>
						</view>

						<template v-if="statusCard.type === 'ongoing'">
							<view class="match-participants">
								<view class="match-player">
									<image class="match-player-avatar" :src="statusPlayers[0].avatar" mode="aspectFill"></image>
									<text class="match-player-name">{{ statusPlayers[0].name }}</text>
								</view>
								<text class="match-score">{{ statusCard.scoreText }}</text>
								<view class="match-player match-player-right">
									<image class="match-player-avatar" :src="statusPlayers[1].avatar" mode="aspectFill"></image>
									<text class="match-player-name">{{ statusPlayers[1].name }}</text>
								</view>
							</view>
							<text class="status-description status-match-description">{{ statusCard.description }}</text>
							<button class="status-action-btn status-action-primary" @click="handleStatusAction(statusCard.action)">
								<text>{{ statusCard.actionText }}</text>
							</button>
						</template>

						<template v-else>
							<view class="empty-status-copy">
								<text class="status-eyebrow">{{ statusCard.eyebrow }}</text>
								<text class="status-description">{{ statusCard.description }}</text>
							</view>
							<view class="status-actions">
								<button
									v-if="showPrimaryStatusAction"
									class="status-action-btn status-action-primary"
									@click="handleStatusAction(statusCard.action)"
								>
									<text>{{ statusCard.actionText }}</text>
								</button>
								<button
									v-if="showSecondaryStatusAction"
									class="status-action-btn status-action-secondary"
									@click="handleStatusAction(statusCard.secondaryAction)"
								>
									<text>{{ statusCard.secondaryActionText }}</text>
								</button>
							</view>
						</template>
					</template>
				</view>

				<view class="metrics-card">
					<view v-for="item in metricCards" :key="item.key" class="metric-item">
						<text class="metric-value" :class="{ accent: item.key === 'win-rate' }">{{ item.value }}</text>
						<text class="metric-label">{{ item.label }}</text>
					</view>
				</view>

				<view class="shortcuts-card">
					<text class="section-title">{{ sectionTitles.quickActions }}</text>
					<view class="shortcut-grid">
						<view v-for="item in quickActions" :key="item.label" class="shortcut-item" @click="item.handler">
							<view class="shortcut-icon" :class="item.iconClass">
								<uni-icons :type="item.icon" size="22" :color="item.iconColor"></uni-icons>
							</view>
							<text class="shortcut-label">{{ item.label }}</text>
						</view>
					</view>
				</view>

				<view v-if="compactRewardEntry.inline" class="reward-entry">
					<view class="reward-entry-copy">
						<view class="reward-entry-heading">
							<text class="reward-entry-title">{{ compactRewardEntry.title }}</text>
							<view class="status-pill">
								<text>{{ compactRewardEntry.statusText }}</text>
							</view>
						</view>
						<text class="reward-entry-description">{{ compactRewardEntry.description }}</text>
					</view>
				</view>

				<view class="services-card">
					<view class="section-heading">
						<text class="section-title">{{ sectionTitles.secondaryServices }}</text>
						<text class="section-hint">按需查看</text>
					</view>
					<view class="service-grid">
						<view class="service-item" @click="handleStatsDetail"><text>竞技分析</text></view>
						<view class="service-item" @click="openRoute('/subPages/rules/index')"><text>规则百科</text></view>
						<view class="service-item" @click="openRoute('/subPages/season/index')"><text>赛季档案</text></view>
						<view class="service-item" @click="openRoute('/subPages/venue/index')"><text>附近球馆</text></view>
						<view class="service-item" @click="openRoute('/subPages/tournament/index')"><text>赛事情报</text></view>
					</view>
				</view>
			</template>
		</view>

		<view v-if="floatRewardVisible" class="float-reward-entry">
			<view class="float-reward-body" @click="handleFavoriteVenueRewardEntry">
				<view class="float-reward-copy">
					<text class="float-reward-title">{{ compactRewardEntry.title }}</text>
					<text class="float-reward-description">{{ compactRewardEntry.description }}</text>
				</view>
				<view class="float-reward-action">
					<text>{{ compactRewardEntry.actionText }}</text>
					<view class="float-reward-close" @click.stop="handleFloatRewardClose">
						<uni-icons type="closeempty" size="14" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
					</view>
				</view>
			</view>
		</view>

		<view v-if="showQrCodeModal" class="qrcode-modal-overlay" @click="closeQrCodeModal">
			<view class="qrcode-modal-container" @click.stop>
				<view class="qrcode-modal-header">
					<text class="qrcode-modal-title">{{ qrCodeModalCopy.title }}</text>
					<view class="qrcode-modal-close" @click="closeQrCodeModal">
						<uni-icons type="closeempty" size="24" :color="isDarkMode ? '#9f926e' : '#64748b'"></uni-icons>
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
						class="qrcode-preload-image"
						@load="onQRCodeLoad"
						@error="onQRCodeError"
					/>
				</view>
				<view class="qrcode-modal-footer">
					<text class="qrcode-hint">{{ qrCodeModalCopy.hint }}</text>
					<text class="qrcode-helper">{{ qrCodeModalCopy.helper }}</text>
				</view>
			</view>
		</view>

		<gameTypeModal v-model:visible="showGameTypeModal" @confirm="handleGameTypeConfirm" />
	</view>
</template>

<script setup>
import { ref, reactive, computed, onUnmounted } from 'vue'
import { onShow, onHide, onPullDownRefresh } from '@dcloudio/uni-app'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { getFavoriteVenueRewardStatus, getUserReputation, getUserStats } from '@/api/user.js'
import { getMemberStatus } from '@/api/member.js'
import { getCurrentMatch, getMatchQRCode, startMatch } from '@/api/match.js'
import { useRankStore } from '@/store/rank.js'
import { userWS, WS_MESSAGE_TYPES } from '@/utils/websocket.js'
import {
	resolveGuestHeroCopy,
	resolveMemberRankAvatarFrame,
	resolveSectionTitles,
	resolveStatusActionVisibility,
	resolveUserHomepageModel
} from '@/utils/user-homepage.js'
import { resolveQrCodeModalCopy } from '@/utils/pk-entry-actions.js'
import gameTypeModal from '@/components/gameTypeModal.vue'
import { useNotificationStore } from '@/store/notification.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { getNotificationList, markAsRead } from '@/api/notification.js'
import { buildHonorWallUrl, presentLatestSeasonRollover } from '@/utils/honor-wall.js'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import {
	POST_LOGIN_ACTIONS,
	consumePostLoginIntent,
	setPostLoginIntent
} from '@/utils/post-login-intent.js'
import {
	getFavoriteVenueRewardPopupStorageKey,
	getFavoriteVenueRewardFloatSnoozeKey,
	calcFavoriteVenueRewardFloatSnoozeUntil,
	isFavoriteVenueRewardFloatSnoozed,
	resolveFavoriteVenueRewardFloatSnoozeMigration
} from '@/utils/favorite-venue-reward.js'
import { resolveMemberGrowthCard } from '@/utils/member-center.js'

const userStore = useUserStore()
const rankStore = useRankStore()
const { isDarkMode } = usePageTheme()
const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()

const guestBenefits = [
	{ icon: 'flag-filled', title: '真实比分', desc: '沉淀每一场 PK 记录' },
	{ icon: 'star-filled', title: '段位成长', desc: '持续追踪段位与积分' },
	{ icon: 'bars', title: '竞技画像', desc: '查看胜率、连胜和对手' },
	{ icon: 'chat', title: '集中提醒', desc: '不错过好友和比赛消息' }
]

const guestHeroCopy = resolveGuestHeroCopy()
const sectionTitles = resolveSectionTitles()
const qrCodeModalCopy = resolveQrCodeModalCopy()

const showGameTypeModal = ref(false)
const showQrCodeModal = ref(false)
const qrcodeLoading = ref(false)
const qrcodeUrl = ref('')
const selectedGameType = ref(null)
const currentRankGameType = ref(3)
const currentMatch = ref(null)
const rankInfo = computed(() => rankStore.rankInfoMap[currentRankGameType.value] || null)
const favoriteVenueRewardStatus = ref(null)
const memberStatus = ref(null)
const floatRewardVisible = ref(false)
const reputationStatus = ref(null)
const coreDataLoading = ref(false)
const homepageLoading = ref(false)
const rankLoading = computed(() => rankStore.loading)
const rankGameTabs = GAME_TYPE_TABS

const isLoggedIn = computed(() => userStore.isLoggedIn)
const hasRankCache = computed(() => (
	rankStore.loaded && rankStore.ownerUserId === (Number(userStore.userId) || 0)
))
const pendingTotal = computed(() => notificationStore.unreadCount + friendRequestStore.pendingCount)
const pendingBadgeText = computed(() => (pendingTotal.value > 99 ? '99+' : String(pendingTotal.value || '')))

const userInfo = computed(() => ({
	id: userStore.userInfo?.id || 0,
	nickname: userStore.userInfo?.nickname || '用户',
	avatar: resolveAvatarUrl(userStore.userInfo?.avatar, userStore.userInfo?.id)
}))

const userStats = reactive({
	totalMatches: 0,
	wins: 0,
	winRate: 0,
	maxStreak: 0,
	loading: false
})

const memberGrowthCard = computed(() => resolveMemberGrowthCard(memberStatus.value || {}, new Date()))
const memberLevelText = computed(() => memberGrowthCard.value.levelLabel || 'Lv1')
const profileLevelText = computed(() => coreDataLoading.value ? '等级同步中' : memberLevelText.value)
const reputationEntryStatusText = computed(() => {
	if (reputationStatus.value?.status_text) return reputationStatus.value.status_text
	return coreDataLoading.value ? '信誉同步中' : '信誉暂不可用'
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

const homepageModel = computed(() => resolveUserHomepageModel({
	isLoggedIn: isLoggedIn.value,
	isLoading: coreDataLoading.value,
	currentMatch: currentMatch.value,
	currentUser: userInfo.value,
	stats: userStats,
	rankInfo: rankInfo.value,
	memberStatus: memberStatus.value,
	memberLevelText: memberLevelText.value,
	rewardStatus: favoriteVenueRewardStatus.value
}))
const homepageMode = computed(() => homepageModel.value.mode)
const statusCard = computed(() => homepageModel.value.statusCard)
const metricCards = computed(() => homepageModel.value.metrics)
const memberHeroStrip = computed(() => homepageModel.value.memberHeroStrip)
const compactRewardEntry = computed(() => homepageModel.value.rewardEntry)
const isActiveMember = computed(() => memberHeroStrip.value.state === 'active')
const rankAvatarFrame = computed(() => resolveMemberRankAvatarFrame({
	isLoggedIn: isLoggedIn.value,
	memberStatus: memberStatus.value,
	rankInfo: rankInfo.value,
	rankLoading: rankLoading.value && !hasRankCache.value,
	now: new Date()
}))
const statusPlayers = computed(() => statusCard.value.players.map(player => ({
	...player,
	avatar: resolveAvatarUrl(player.avatar, player.id)
})))
const statusActionVisibility = computed(() => resolveStatusActionVisibility({ statusCard: statusCard.value }))
const showPrimaryStatusAction = computed(() => statusActionVisibility.value.showPrimary)
const showSecondaryStatusAction = computed(() => statusActionVisibility.value.showSecondary)

const rankProgressText = computed(() => {
	if (rankInfo.value?.level >= 6) return '已达到最高段位'
	if (!rankInfo.value) return '完成首场比赛后开始定级'
	return `进度 ${displayRank.value.progress}% · 下一段位 ${displayRank.value.nextName}`
})

const quickActions = computed(() => ([
	{ label: '比赛记录', icon: 'list', iconColor: '#b77908', iconClass: 'gold', handler: handleMatchHistory },
	{ label: '过往对手', icon: 'contact', iconColor: '#2563eb', iconClass: 'blue', handler: handleOpponentRecord },
	{ label: '荣誉墙', icon: 'medal', iconColor: '#7c3aed', iconClass: 'purple', handler: handleAchievement },
	{ label: '好友', icon: 'person-filled', iconColor: '#0f766e', iconClass: 'teal', handler: handleFriendList }
]))

onShow(() => {
	if (isLoggedIn.value) {
		Promise.resolve(loadHomepageData()).finally(showLatestSeasonRollover)
		connectUserWS()
		handlePendingPostLoginIntent()
	} else {
		resetHomepageState()
		notificationStore.clearUnread()
		friendRequestStore.clearPendingCount()
	}
})

onHide(() => {
	unsubscribeUserWS()
})

onUnmounted(() => {
	unsubscribeUserWS()
})

const loadHomepageData = async () => {
	if (homepageLoading.value) return

	homepageLoading.value = true
	coreDataLoading.value = true
	const secondaryRequests = Promise.allSettled([
		notificationStore.fetchUnreadCount(),
		friendRequestStore.fetchPendingCount(),
		loadReputationStatus(),
		loadFavoriteVenueRewardStatus(),
		loadMemberStatus()
	])

	try {
		await Promise.allSettled([
			loadUserStats(),
			loadRankInfo(),
			loadCurrentMatch()
		])
	} finally {
		coreDataLoading.value = false
	}

	await secondaryRequests
	migrateFavoriteVenueRewardFloatSnooze()
	syncFloatRewardVisibility()
	homepageLoading.value = false
}

const loadUserStats = async () => {
	if (userStats.loading) return

	userStats.loading = true
	try {
		const res = await getUserStats()
		if (res.success) {
			userStats.totalMatches = res.total_matches || 0
			userStats.wins = res.wins || 0
			userStats.winRate = res.win_rate || 0
			userStats.maxStreak = res.max_win_streak || 0
		}
	} catch (error) {
		console.error('获取用户统计失败:', error)
	} finally {
		userStats.loading = false
	}
}

const loadReputationStatus = async () => {
	try {
		const res = await getUserReputation()
		reputationStatus.value = res?.success ? res : null
	} catch (error) {
		console.error('获取信誉状态失败:', error)
		reputationStatus.value = null
	}
}

const loadRankInfo = async () => {
	try {
		await rankStore.ensureFresh(userStore.userId)
	} catch (error) {
		console.error('获取段位信息失败:', error)
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

const loadFavoriteVenueRewardStatus = async () => {
	try {
		const res = await getFavoriteVenueRewardStatus()
		favoriteVenueRewardStatus.value = res.success ? res : null
	} catch (error) {
		console.error('获取常玩球馆奖励状态失败:', error)
		favoriteVenueRewardStatus.value = null
	}
}

const loadMemberStatus = async () => {
	try {
		const res = await getMemberStatus()
		memberStatus.value = res.success ? res : null
	} catch (error) {
		console.error('获取会员状态失败:', error)
		memberStatus.value = null
	}
}

const resetHomepageState = () => {
	currentMatch.value = null
	rankStore.clear()
	favoriteVenueRewardStatus.value = null
	memberStatus.value = null
	reputationStatus.value = null
	floatRewardVisible.value = false
	coreDataLoading.value = false
	homepageLoading.value = false
	userStats.totalMatches = 0
	userStats.wins = 0
	userStats.winRate = 0
	userStats.maxStreak = 0
}

const getFloatSnoozeStorageValue = (userId, status) => {
	const key = getFavoriteVenueRewardFloatSnoozeKey(userId, status)
	const raw = uni.getStorageSync(key)
	if (!raw) return null
	const parsed = new Date(raw)
	return Number.isNaN(parsed.getTime()) ? null : parsed
}

const writeFloatSnooze = (userId, status) => {
	const snoozeUntil = calcFavoriteVenueRewardFloatSnoozeUntil()
	uni.setStorageSync(getFavoriteVenueRewardFloatSnoozeKey(userId, status), snoozeUntil.toISOString())
}

const migrateFavoriteVenueRewardFloatSnooze = () => {
	const userId = userInfo.value.id
	if (!userId) return
	const oldDismissed = !!uni.getStorageSync(getFavoriteVenueRewardPopupStorageKey(userId))
	const migration = resolveFavoriteVenueRewardFloatSnoozeMigration({
		userId,
		rewardStatus: favoriteVenueRewardStatus.value,
		oldPopupDismissed: oldDismissed,
		getFloatSnoozeValue: (uid, status) => getFloatSnoozeStorageValue(uid, status)
	})
	if (migration) {
		uni.setStorageSync(migration.key, migration.snoozeUntil.toISOString())
		uni.removeStorageSync(getFavoriteVenueRewardPopupStorageKey(userId))
	} else if (oldDismissed) {
		uni.removeStorageSync(getFavoriteVenueRewardPopupStorageKey(userId))
	}
}

const syncFloatRewardVisibility = () => {
	const status = favoriteVenueRewardStatus.value
	if (!status || !status.enabled || !userInfo.value.id) {
		floatRewardVisible.value = false
		return
	}
	const state = status.status
	if (state !== 'not_started' && state !== 'rejected') {
		floatRewardVisible.value = false
		return
	}
	const snoozeUntil = getFloatSnoozeStorageValue(userInfo.value.id, state)
	if (isFavoriteVenueRewardFloatSnoozed(snoozeUntil)) {
		floatRewardVisible.value = false
		return
	}
	if (snoozeUntil) {
		uni.removeStorageSync(getFavoriteVenueRewardFloatSnoozeKey(userInfo.value.id, state))
	}
	floatRewardVisible.value = true
}

const handleFavoriteVenueRewardAction = () => {
	uni.navigateTo({ url: '/subPages/venue/submit' })
}

const handleFavoriteVenueRewardEntry = () => {
	if (!compactRewardEntry.value.action) return
	handleFavoriteVenueRewardAction()
}

const handleFloatRewardClose = () => {
	const status = favoriteVenueRewardStatus.value
	if (!status || !userInfo.value.id) return
	writeFloatSnooze(userInfo.value.id, status.status)
	floatRewardVisible.value = false
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

const unsubscribeUserWS = () => {
	userWS.off(WS_MESSAGE_TYPES.MATCH_START, handleMatchStart)
	userWS.off(WS_MESSAGE_TYPES.NOTIFICATION_UPDATE, handleNotificationUpdate)
}

const handleNotificationUpdate = (data = {}) => {
	if (data?.category === 'match_result') {
		rankStore.invalidate(userStore.userId)
	}
	notificationStore.fetchUnreadCount()
	friendRequestStore.fetchPendingCount()
}

const handleMatchStart = (data) => {
	if (showQrCodeModal.value) showQrCodeModal.value = false

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
	if (action === 'show_pk_code') {
		handleQrCode()
		return
	}
	handleStartPK()
}

const handleGoLogin = (postLoginAction = '') => {
	setPostLoginIntent(postLoginAction)
	uni.navigateTo({ url: '/pages/login/login' })
}

const handleGuestStartPK = () => {
	handleGoLogin(POST_LOGIN_ACTIONS.START_PK)
}

const handlePendingPostLoginIntent = () => {
	const pendingIntent = consumePostLoginIntent()

	if (pendingIntent === POST_LOGIN_ACTIONS.START_PK) {
		handleStartPK()
		return
	}

	if (pendingIntent === POST_LOGIN_ACTIONS.SHOW_PK_CODE) handleQrCode()
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
		success: (res) => handleMatchResult(res.result),
		fail: (err) => console.error('扫码失败:', err)
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
	uni.navigateTo({ url: buildPlayingRoute(match) })
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
				if (confirm) navigateToPlayingMatch(outcome.match)
			}
		})
		return
	}

	uni.showToast({ title: outcome.message || '创建对局失败', icon: 'none' })
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
	uni.navigateTo({ url: buildHonorWallUrl({ gameType: currentRankGameType.value }) })
}

const showLatestSeasonRollover = () => {
	presentLatestSeasonRollover({
		getNotificationList,
		markAsRead,
		showModal: options => uni.showModal(options),
		onRead: () => notificationStore.fetchUnreadCount()
	}).catch(error => console.error('展示换季结果失败:', error))
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
}

onPullDownRefresh(async () => {
	if (!isLoggedIn.value) {
		uni.stopPullDownRefresh()
		return
	}
	try {
		await rankStore.forceRefresh(userStore.userId)
	} catch (error) {
		console.error('刷新段位信息失败:', error)
	} finally {
		uni.stopPullDownRefresh()
	}
})

const handleNotificationCenter = () => {
	uni.navigateTo({ url: '/subPages/notification/index' })
}

const handleHelp = () => {
	uni.navigateTo({ url: '/subPages/help/feedback' })
}

const handleSettings = () => {
	uni.navigateTo({ url: '/subPages/user/settings' })
}

const handleEditProfile = () => {
	uni.navigateTo({ url: '/subPages/user/editProfile' })
}

const handleOpenMemberCenter = () => {
	uni.navigateTo({ url: '/subPages/user/memberCenter' })
}

const handleOpenReputation = () => {
	uni.navigateTo({ url: '/subPages/user/reputation' })
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
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
