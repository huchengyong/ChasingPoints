<template>
	<view class="user-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="main-content" :class="{ 'has-float-reward': floatRewardVisible }">
			<template v-if="!isLoggedIn">
				<view class="guest-card">
					<view class="guest-head">
						<view class="guest-avatar">
							<uni-icons type="person-filled" size="30" :color="mutedIconColor"></uni-icons>
						</view>
						<view class="profile-copy">
							<text class="profile-name">未登录</text>
							<text class="guest-sub">登录后同步你的段位、战绩和好友</text>
						</view>
					</view>
					<view class="guest-benefits">
						<view v-for="item in guestBenefits" :key="item" class="guest-benefit">
							<uni-icons type="checkmarkempty" size="16" :color="accentIconColor"></uni-icons>
							<text>{{ item }}</text>
						</view>
					</view>
					<button class="login-btn" @click="handleGoLogin">登录 / 注册</button>
				</view>

				<view class="list-card">
					<view v-for="item in guestLinks" :key="item.title" class="list-row" @click="item.handler">
						<view class="row-icon">
							<uni-icons :type="item.icon" size="20" :color="accentIconColor"></uni-icons>
						</view>
						<view class="row-copy">
							<text class="row-title">{{ item.title }}</text>
							<text class="row-desc">{{ item.desc }}</text>
						</view>
						<uni-icons type="right" size="16" :color="mutedIconColor"></uni-icons>
					</view>
				</view>
			</template>

			<template v-else>
				<view class="profile-header">
					<view class="profile-main" @click="handleEditProfile">
						<view class="profile-avatar" :class="{ 'has-rank-frame': rankAvatarFrame }">
							<image class="profile-avatar-image" :src="userInfo.avatar" mode="aspectFill"></image>
							<image v-if="rankAvatarFrame" class="profile-avatar-frame" :src="rankAvatarFrame" mode="aspectFit"></image>
						</view>
						<view class="profile-copy">
							<text class="profile-name">{{ userInfo.nickname }}</text>
							<view class="profile-chips">
								<view
									class="profile-chip"
									:class="{ 'profile-chip--vip': memberIdentity.showBadge }"
									@click.stop="handleOpenMemberCenter"
								>
									<text>{{ profileLevelText }}</text>
								</view>
								<view
									class="profile-chip"
									:class="{ 'profile-chip--danger': reputationChip.restricted }"
									@click.stop="handleOpenReputation"
								>
									<text>{{ reputationChip.text }}</text>
								</view>
							</view>
						</view>
					</view>
					<view class="profile-actions">
						<button class="icon-btn" aria-label="出示我的二维码" @click="showMyMatchQRCode">
							<view class="qr-glyph">
								<view class="qr-glyph-finder qr-glyph-finder--top-left"></view>
								<view class="qr-glyph-finder qr-glyph-finder--top-right"></view>
								<view class="qr-glyph-finder qr-glyph-finder--bottom-left"></view>
								<view class="qr-glyph-dots"></view>
							</view>
						</button>
						<button class="icon-btn" @click="handleSettings">
							<uni-icons type="gear" size="21" :color="secondaryIconColor"></uni-icons>
						</button>
					</view>
				</view>

				<view class="rank-card">
					<view v-if="rankGameView.tabs.length" class="rank-tabs">
						<view
							v-for="item in rankGameView.tabs"
							:key="item.value"
							class="rank-tab"
							:class="{ active: currentRankGameType === item.value }"
							@tap="handleRankGameTypeChange(item.value)"
						>
							<text>{{ item.label }}</text>
						</view>
					</view>
					<text v-else-if="hasRankCache" class="rank-game-label">{{ rankGameView.primaryLabel }}</text>

					<view v-if="!hasRankCache && (coreDataLoading || rankLoading)" class="rank-skeleton">
						<view class="skeleton-circle"></view>
						<view class="rank-skeleton-copy">
							<view class="skeleton-line skeleton-line-wide"></view>
							<view class="skeleton-line skeleton-line-short"></view>
						</view>
					</view>
					<view v-else class="rank-body" @click="handleRankExplain">
						<view class="rank-main">
							<image class="rank-icon" :src="rankView.icon" mode="aspectFit"></image>
							<view class="rank-copy">
								<text class="rank-name">{{ rankView.name }}</text>
								<text v-if="rankView.ranked" class="rank-score">排位分 {{ rankView.score }}</text>
							</view>
							<view class="rank-more">
								<text>段位说明</text>
								<uni-icons type="right" size="14" color="rgba(255, 255, 255, 0.72)"></uni-icons>
							</view>
						</view>
						<view v-if="rankView.ranked" class="rank-progress">
							<view class="rank-progress-fill" :style="{ width: `${rankView.progress}%` }"></view>
						</view>
						<text class="rank-hint">{{ rankView.hint }}</text>
					</view>
					<view v-if="showRankGamePicker" class="rank-picker">
						<text class="rank-picker-title">你主要打哪种？选好后发起 PK 也会默认选中</text>
						<view class="rank-picker-options">
							<view
								v-for="item in rankGameTabs"
								:key="item.value"
								class="rank-picker-option"
								@tap="handlePickRankGameType(item.value)"
							>
								<text>{{ item.label }}</text>
							</view>
						</view>
					</view>
				</view>

				<view class="me-section">
					<view class="section-header">
						<view class="section-title-row">
							<text class="section-title">生涯数据</text>
							<text class="section-caption">全部球种</text>
						</view>
						<view class="section-more" @click="handleStatsDetail">
							<text>竞技分析</text>
							<uni-icons type="right" size="14" :color="mutedIconColor"></uni-icons>
						</view>
					</view>
					<view v-if="careerState === 'empty'" class="career-empty">
						<text class="empty-title">还没有对局记录</text>
						<text class="empty-desc">打完第一场，场次、胜率和连胜都会记在这里</text>
					</view>
					<view v-else class="metrics-card" @click="handleStatsDetail">
						<view v-for="item in metricCards" :key="item.key" class="metric-item">
							<text class="metric-value" :class="{ accent: item.key === 'win-rate' }">{{ item.value }}</text>
							<text class="metric-label">{{ item.label }}</text>
						</view>
					</view>
					<view v-if="careerState === 'error'" class="career-error" @click="retryHomepageData">
						<text>生涯数据读取失败，点击重试</text>
					</view>
				</view>

				<view class="shortcuts-card">
					<view v-for="item in quickActions" :key="item.label" class="shortcut-item" @click="item.handler">
						<view class="shortcut-icon">
							<uni-icons :type="item.icon" size="22" :color="accentIconColor"></uni-icons>
						</view>
						<text class="shortcut-label">{{ item.label }}</text>
					</view>
				</view>

				<view v-if="compactRewardEntry.inline" class="reward-entry">
					<view class="reward-entry-heading">
						<text class="reward-entry-title">{{ compactRewardEntry.title }}</text>
						<text class="reward-entry-status">{{ compactRewardEntry.statusText }}</text>
					</view>
					<text class="reward-entry-description">{{ compactRewardEntry.description }}</text>
				</view>

				<view class="list-card">
					<view v-for="item in serviceLinks" :key="item.title" class="list-row" @click="item.handler">
						<view class="row-icon">
							<uni-icons :type="item.icon" size="20" :color="accentIconColor"></uni-icons>
						</view>
						<text class="row-title">{{ item.title }}</text>
						<text v-if="item.value" class="row-value">{{ item.value }}</text>
						<uni-icons type="right" size="16" :color="mutedIconColor"></uni-icons>
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
						<uni-icons type="closeempty" size="14" :color="mutedIconColor"></uni-icons>
					</view>
				</view>
			</view>
		</view>

		<view v-if="showMatchQrModal" class="qr-mask" @click="closeMatchQrModal">
			<view class="qr-panel" @click.stop>
				<view class="qr-header">
					<view>
						<text class="qr-title">出示我的匹配二维码</text>
						<text class="qr-subtitle">请让对手使用发起对局扫码入口识别</text>
					</view>
					<button class="qr-close" @click="closeMatchQrModal">
						<uni-icons type="closeempty" size="22" :color="mutedIconColor"></uni-icons>
					</button>
				</view>
				<view class="qr-body">
					<view v-if="matchQrLoading" class="qr-state">
						<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
						<text>二维码生成中...</text>
					</view>
					<view v-if="!matchQrLoading && matchQrFailed" class="qr-state">
						<text class="qr-error">二维码加载失败</text>
						<button class="qr-retry" @click="loadMyMatchQRCode">重试</button>
					</view>
					<canvas
						v-if="matchQrContent && !matchQrFailed"
						canvas-id="user-match-invite-qrcode"
						id="user-match-invite-qrcode"
						class="qr-image"
						:class="{ 'is-loading': matchQrLoading }"
						:width="240"
						:height="240"
					/>
				</view>
				<button class="qr-done" @click="closeMatchQrModal">关闭</button>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onUnmounted } from 'vue'
import { onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { useActivityStore } from '@/store/activity.js'
import { useUserOverviewStore } from '@/store/userOverview.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useMainTabBar, setMainTabBarVisible } from '@/utils/use-main-tab-bar.js'
import { useRankStore } from '@/store/rank.js'
import {
	resolveCompactRewardEntry,
	resolveCoreMetrics,
	resolveMemberEntry,
	resolveSelfMemberIdentity,
	resolveRankCardView,
	resolveRankGameTypeView,
	resolveReputationChip
} from '@/utils/user-homepage.js'
import { useNotificationStore } from '@/store/notification.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { markAsRead } from '@/api/notification.js'
import { getMatchQRCode } from '@/api/match.js'
import { buildHonorWallUrl, presentLatestSeasonRollover } from '@/utils/honor-wall.js'
import { readDefaultGameType, saveDefaultGameType } from '@/utils/game-type-preference.js'
import { renderLocalQRCode } from '@/utils/local-qrcode.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import {
	getFavoriteVenueRewardPopupStorageKey,
	getFavoriteVenueRewardFloatSnoozeKey,
	calcFavoriteVenueRewardFloatSnoozeUntil,
	isFavoriteVenueRewardFloatSnoozed,
	resolveFavoriteVenueRewardFloatSnoozeMigration
} from '@/utils/favorite-venue-reward.js'
import { resolveMemberGrowthCard } from '@/utils/member-center.js'

const userStore = useUserStore()
const activityStore = useActivityStore()
const userOverviewStore = useUserOverviewStore()
const rankStore = useRankStore()
const { isDarkMode } = usePageTheme()
useMainTabBar('user')
const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()

const guestBenefits = ['记录每一局真实比分', '四个球种分别积累排位分', '查看胜率、连胜和过往对手']

const accentIconColor = computed(() => (isDarkMode.value ? '#F0C542' : '#C69200'))
const secondaryIconColor = computed(() => (isDarkMode.value ? '#D7C89B' : '#6E6242'))
const mutedIconColor = computed(() => (isDarkMode.value ? '#9F926E' : '#9A8C67'))

const rankGameTabs = GAME_TYPE_TABS
// 段位卡默认展示用户在设置里选的常用球种，没设时按排位场次自动选；页内切换只临时生效。
const preferredGameType = ref(readDefaultGameType(uni, userStore.userId))
let rankPreferenceOwnerId = Number(userStore.userId) || 0
const selectedRankGameType = ref(0)
const rankGameView = computed(() => resolveRankGameTypeView({
	preferredGameType: preferredGameType.value,
	rankInfoMap: hasRankCache.value ? rankStore.rankInfoMap : {}
}))
const currentRankGameType = computed(() => (
	rankGameView.value.tabs.some((item) => item.value === selectedRankGameType.value)
		? selectedRankGameType.value
		: rankGameView.value.primaryGameType
))
const showRankGamePicker = computed(() => hasRankCache.value && rankGameView.value.needsPick)
const rankInfo = computed(() => rankStore.rankInfoMap[currentRankGameType.value] || null)
const rankView = computed(() => resolveRankCardView(rankInfo.value))
const showMatchQrModal = ref(false)
const matchQrLoading = ref(false)
const matchQrFailed = ref(false)
const matchQrContent = ref('')
const favoriteVenueRewardStatus = ref(null)
const memberStatus = ref(null)
const floatRewardVisible = ref(false)
const reputationStatus = ref(null)
const coreDataLoading = ref(false)
const homepageLoading = ref(false)
const statsReady = ref(false)
const statsFailed = ref(false)
const rankLoading = computed(() => rankStore.loading)

const isLoggedIn = computed(() => userStore.isLoggedIn)
const hasRankCache = computed(() => (
	rankStore.loaded && rankStore.ownerUserId === (Number(userStore.userId) || 0)
))

const userInfo = computed(() => ({
	id: userStore.userInfo?.id || 0,
	nickname: userStore.userInfo?.nickname || '用户',
	avatar: resolveAvatarUrl(userStore.userInfo?.avatar, userStore.userInfo?.id)
}))

const userStats = reactive({
	totalMatches: 0,
	wins: 0,
	winRate: 0,
	maxStreak: 0
})

const memberEntry = computed(() => resolveMemberEntry(memberStatus.value || {}, new Date()))
const memberLevelText = computed(() => resolveMemberGrowthCard(memberStatus.value || {}, new Date()).levelLabel || 'Lv1')
// 已有缓存时刷新不闪「同步中」，只有首次读取才显示占位文案。
const profileLevelText = computed(() => {
	if (!memberStatus.value && coreDataLoading.value) return '等级同步中'
	const identity = memberIdentity.value
	return identity.showBadge ? `${identity.badgeText} · ${identity.tierLabel}` : memberLevelText.value
})
const reputationChip = computed(() => resolveReputationChip(reputationStatus.value, coreDataLoading.value))
const compactRewardEntry = computed(() => resolveCompactRewardEntry(favoriteVenueRewardStatus.value || {}))
const memberIdentity = computed(() => resolveSelfMemberIdentity({
	isLoggedIn: isLoggedIn.value,
	memberStatus: memberStatus.value,
	rankInfo: rankInfo.value,
	rankLoading: rankLoading.value && !hasRankCache.value,
	now: new Date()
}))
const rankAvatarFrame = computed(() => memberIdentity.value.frameUrl)

const careerState = computed(() => {
	if (statsReady.value) return userStats.totalMatches > 0 ? 'ready' : 'empty'
	return statsFailed.value && !coreDataLoading.value ? 'error' : 'loading'
})
const metricCards = computed(() => resolveCoreMetrics({
	stats: userStats,
	isLoading: !statsReady.value
}))

const quickActions = [
	{ label: '比赛记录', icon: 'list', handler: () => openRoute('/subPages/user/matchHistory') },
	{ label: '过往对手', icon: 'contact', handler: () => openRoute('/subPages/user/opponentRecord') },
	{ label: '荣誉墙', icon: 'medal', handler: () => handleAchievement() },
	{ label: '好友', icon: 'person', handler: () => openRoute('/subPages/social/friendList') }
]

const serviceLinks = computed(() => ([
	{ title: '约球记录', icon: 'list', value: '', handler: () => openRoute('/subPages/social/challenges?tab=history') },
	{ title: '会员中心', icon: 'vip', value: memberEntry.value.text, handler: handleOpenMemberCenter },
	{ title: '赛季档案', icon: 'calendar', value: '', handler: () => openRoute('/subPages/season/index') },
	{ title: '规则百科', icon: 'help', value: '', handler: () => openRoute('/subPages/rules/index') },
	{ title: '附近球馆', icon: 'location', value: '', handler: () => openRoute('/subPages/venue/index') }
]))

const guestLinks = [
	{ title: '排行榜', desc: '各球种高手的段位与积分', icon: 'medal', handler: () => openRoute('/pages/ranking/index') },
	{ title: '规则百科', desc: '常见台球规则与术语', icon: 'help', handler: () => openRoute('/subPages/rules/index') },
	{ title: '帮助、投诉与举报', desc: '无需登录即可提交问题与反馈', icon: 'chatbubble', handler: () => handleHelp() }
]

onShow(() => {
	syncRankGamePreference()
	if (isLoggedIn.value) {
		Promise.resolve(loadHomepageData()).finally(showLatestSeasonRollover)
	} else {
		resetHomepageState()
		notificationStore.clearUnread()
		friendRequestStore.clearPendingCount()
	}
})

const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const applyUserOverview = (overview = {}) => {
	const stats = overview.stats?.success ? overview.stats : null
	statsReady.value = Boolean(stats)
	statsFailed.value = !stats
	userStats.totalMatches = stats?.total_matches || 0
	userStats.wins = stats?.wins || 0
	userStats.winRate = stats?.win_rate || 0
	userStats.maxStreak = stats?.max_win_streak || 0
	reputationStatus.value = overview.reputation?.success ? overview.reputation : null
	favoriteVenueRewardStatus.value = overview.favoriteVenueRewardStatus?.success
		? overview.favoriteVenueRewardStatus
		: null
	memberStatus.value = overview.memberStatus?.success ? overview.memberStatus : null
}

const loadHomepageData = async ({ force = false } = {}) => {
	if (homepageLoading.value) return

	homepageLoading.value = true
	coreDataLoading.value = true
	const identity = getReadIdentity()
	try {
		const [, overviewResult] = await Promise.allSettled([
			activityStore.fetch(identity, { force, silent: true }),
			userOverviewStore.fetch(identity, { force, silent: true }),
			rankStore.ensureFresh(identity)
		])
		if (overviewResult.status === 'fulfilled') {
			applyUserOverview(overviewResult.value)
		} else {
			applyUserOverview(userOverviewStore.snapshot())
		}
	} finally {
		coreDataLoading.value = false
		homepageLoading.value = false
	}

	migrateFavoriteVenueRewardFloatSnooze()
	syncFloatRewardVisibility()
}

const retryHomepageData = () => {
	loadHomepageData({ force: true })
}

const resetHomepageState = () => {
	activityStore.clear()
	userOverviewStore.clear()
	rankStore.clear()
	favoriteVenueRewardStatus.value = null
	memberStatus.value = null
	reputationStatus.value = null
	floatRewardVisible.value = false
	coreDataLoading.value = false
	homepageLoading.value = false
	statsReady.value = false
	statsFailed.value = false
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

const openRoute = (url) => {
	uni.navigateTo({ url })
}

const handleGoLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}

const showMyMatchQRCode = async () => {
	showMatchQrModal.value = true
	setMainTabBarVisible(false)
	await loadMyMatchQRCode()
}

const loadMyMatchQRCode = async () => {
	const authGeneration = userStore.authGeneration
	matchQrLoading.value = true
	matchQrFailed.value = false
	matchQrContent.value = ''
	try {
		const res = await getMatchQRCode()
		// 弹窗已关闭或身份已变化时丢弃迟到的二维码
		if (!showMatchQrModal.value || authGeneration !== userStore.authGeneration) return
		if (!res?.success || !res.qrcode_data) {
			matchQrLoading.value = false
			matchQrFailed.value = true
			return
		}
		matchQrContent.value = res.qrcode_data
		await nextTick()
		renderLocalQRCode({
			canvasId: 'user-match-invite-qrcode',
			content: matchQrContent.value,
			size: 240,
			uniApi: uni
		})
		matchQrLoading.value = false
	} catch (error) {
		if (!showMatchQrModal.value) return
		matchQrLoading.value = false
		matchQrFailed.value = true
	}
}

const closeMatchQrModal = () => {
	showMatchQrModal.value = false
	matchQrLoading.value = false
	matchQrFailed.value = false
	matchQrContent.value = ''
	setMainTabBarVisible(true)
}

onUnmounted(() => {
	if (showMatchQrModal.value) setMainTabBarVisible(true)
})

const handleRankExplain = () => {
	uni.navigateTo({ url: `/subPages/user/rankExplain?game_type=${currentRankGameType.value}` })
}

const handleAchievement = () => {
	uni.navigateTo({ url: buildHonorWallUrl({ gameType: currentRankGameType.value }) })
}

const showLatestSeasonRollover = () => {
	presentLatestSeasonRollover({
		notification: activityStore.latestSeasonRollover,
		markAsRead,
		showModal: options => uni.showModal(options),
		onRead: () => {
			activityStore.setLatestSeasonRollover(null)
			notificationStore.decrementUnread(getReadIdentity())
		},
		navigateTo: url => uni.navigateTo({ url })
	}).catch(error => console.error('展示换季结果失败:', error))
}

const handleStatsDetail = () => {
	uni.navigateTo({ url: `/subPages/user/statsDetail?game_type=${currentRankGameType.value}` })
}

const handleRankGameTypeChange = (gameType) => {
	if (currentRankGameType.value === gameType) return
	selectedRankGameType.value = gameType
}

// 设置页或开局弹窗改了常用球种、或换了账号时，段位卡回到新的常用球种。
const syncRankGamePreference = () => {
	const ownerId = Number(userStore.userId) || 0
	const preferred = readDefaultGameType(uni, ownerId)
	if (ownerId === rankPreferenceOwnerId && preferred === preferredGameType.value) return
	rankPreferenceOwnerId = ownerId
	preferredGameType.value = preferred
	selectedRankGameType.value = 0
}

// 段位卡里直接选的常用球种与设置页、开局弹窗共用同一偏好。
const handlePickRankGameType = (gameType) => {
	preferredGameType.value = saveDefaultGameType(uni, userStore.userId, gameType)
	selectedRankGameType.value = 0
	uni.showToast({ title: '已设为常用球种', icon: 'none' })
}

onPullDownRefresh(async () => {
	if (!isLoggedIn.value) {
		uni.stopPullDownRefresh()
		return
	}
	try {
		await Promise.allSettled([
			activityStore.fetch(getReadIdentity(), { force: true, silent: true }),
			userOverviewStore.fetch(getReadIdentity(), { force: true, silent: true }),
			rankStore.forceRefresh(getReadIdentity())
		])
		applyUserOverview(userOverviewStore.snapshot())
	} catch (error) {
		console.error('刷新个人主页失败:', error)
	} finally {
		uni.stopPullDownRefresh()
	}
})

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
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
