<template>
	<view class="honor-wall-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading && !loaded" class="state-panel full-state">
			<uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
			<text class="state-title">正在整理荣誉墙</text>
			<text class="state-description">生涯成就、赛季挑战与历届荣誉即将呈现</text>
		</view>

		<view v-else-if="loadFailed && !loaded" class="state-panel full-state">
			<uni-icons type="info-filled" size="40" color="#E0AE12"></uni-icons>
			<text class="state-title">荣誉墙暂时没加载出来</text>
			<text class="state-description">请检查网络后重试，已有荣誉不会受到影响</text>
			<button class="retry-button" @tap="loadData">
				<text>重新加载</text>
			</button>
		</view>

		<view v-else class="honor-wall-content">
			<view v-if="refreshing" class="refresh-indicator">
				<uni-icons type="spinner-cycle" size="16" color="#C69200"></uni-icons>
				<text>正在更新</text>
			</view>

			<view class="honor-hero-card">
				<view class="profile-row">
					<image class="profile-avatar" :src="profileAvatar" mode="aspectFill"></image>
					<view class="profile-copy">
						<text class="profile-kicker">{{ isSelf ? '我的荣誉墙' : '好友公开荣誉' }}</text>
						<text class="profile-name">{{ wall.profile.nickname || (isSelf ? '我的荣誉' : '球友') }}</text>
						<text class="profile-note">{{ isSelf ? '生涯永久累计，赛季挑战独立刷新' : '仅展示已获得成就与永久荣誉' }}</text>
					</view>
				</view>

				<view class="equipped-title-card" :class="{ interactive: isSelf }" @tap="goToTitles">
					<view class="title-medal">
						<uni-icons type="medal-filled" size="25" color="#ffffff"></uni-icons>
					</view>
					<view class="title-copy">
						<text class="title-label">当前称号</text>
						<text class="title-name">{{ wall.equipped_title?.title_name || '暂未装备称号' }}</text>
						<text class="title-source">{{ equippedTitleSource }}</text>
					</view>
					<uni-icons v-if="isSelf" type="right" size="18" color="#ffffff"></uni-icons>
				</view>

				<view class="summary-grid">
					<view v-for="item in summaryItems" :key="item.key" class="summary-item">
						<text class="summary-value">{{ item.value }}</text>
						<text class="summary-label">{{ item.label }}</text>
					</view>
				</view>

				<view class="recent-section">
					<view class="section-heading compact-heading">
						<text class="section-title">最近荣誉</text>
						<text class="section-hint">自动取最近 3 项</text>
					</view>
					<view v-if="recentHonors.length" class="recent-list">
						<view v-for="item in recentHonors" :key="`${item.source_type}-${item.id}`" class="recent-item">
							<view class="recent-icon"><text>{{ getHonorEmoji(item) }}</text></view>
							<view class="recent-copy">
								<view class="recent-title-row">
									<text class="recent-title">{{ item.name }}</text>
									<text class="recent-type">{{ item.sourceLabel }}</text>
								</view>
								<text class="recent-detail">{{ item.detailText }}</text>
								<text v-if="item.earned_at" class="recent-time">{{ item.earned_at }}</text>
							</view>
						</view>
					</view>
					<view v-else class="inline-empty">
						<text>完成第一项成就后，最近荣誉会出现在这里</text>
					</view>
				</view>
			</view>

			<view class="honor-tabs" :class="{ 'friend-tabs': !isSelf }">
				<view
					v-for="tab in tabs"
					:key="tab.key"
					class="honor-tab"
					:class="{ active: activeTab === tab.key }"
					@tap="switchTab(tab.key)"
				>
					<text>{{ tab.label }}</text>
				</view>
			</view>

			<view v-if="activeTab === 'career'" class="tab-panel">
				<view class="section-heading page-section-heading">
					<view class="heading-copy">
						<text class="section-title">生涯成就</text>
						<text class="section-description">永久累计，不随赛季清零</text>
					</view>
					<text class="section-count">{{ wall.summary.career_unlocked || 0 }}/{{ wall.summary.career_total || 0 }}</text>
				</view>

				<view v-if="achievementGroups.length" class="achievement-groups">
					<view v-for="group in achievementGroups" :key="group.key" class="content-card achievement-group">
						<view class="group-header">
							<view class="group-title-wrap">
								<text class="group-emoji">{{ getCategoryEmoji(group.key) }}</text>
								<text class="group-title">{{ group.label }}成就</text>
							</view>
							<text class="group-count">{{ group.list.length }} 项</text>
						</view>
						<view
							v-for="item in group.list"
							:key="item.id"
							class="achievement-row"
							:class="{ unlocked: item.unlocked, interactive: isSelf }"
							@tap="goToDetail(item.id)"
						>
							<view class="achievement-icon" :class="{ locked: !item.unlocked }">
								<image v-if="item.icon" :src="item.icon" mode="aspectFit"></image>
								<text v-else>{{ getCategoryEmoji(item.category) }}</text>
							</view>
							<view class="achievement-copy">
								<view class="achievement-name-row">
									<text class="achievement-name">{{ item.name }}</text>
									<text v-if="item.unlocked" class="unlocked-tag">已解锁</text>
								</view>
								<text class="achievement-description">{{ item.description || '完成目标即可永久点亮' }}</text>
								<view class="progress-track">
									<view class="progress-fill" :style="{ width: `${getCareerProgressPercent(item)}%` }"></view>
								</view>
								<text class="progress-note">{{ getCareerProgressText(item) }}</text>
							</view>
							<uni-icons v-if="isSelf" type="right" size="16" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
						</view>
					</view>
				</view>
				<view v-else class="content-card state-panel compact-state">
					<text class="state-title">暂无可展示的生涯成就</text>
					<text class="state-description">完成有效比赛后，成就进度会自动更新</text>
				</view>
			</view>

			<view v-else-if="activeTab === 'season'" class="tab-panel">
				<view class="section-heading page-section-heading">
					<view class="heading-copy">
						<text class="section-title">当前赛季</text>
						<text class="section-description">每个赛季、每种球类独立累计</text>
					</view>
				</view>

				<view class="game-type-grid">
					<view
						v-for="item in gameTypeTabs"
						:key="item.value"
						class="game-type-chip"
						:class="{ active: currentGameType === item.value }"
						@tap="handleGameTypeChange(item.value)"
					>
						<text>{{ item.label }}</text>
					</view>
				</view>

				<view v-if="currentSeasonState.mode === 'intermission'" class="content-card intermission-card">
					<view class="intermission-icon">
						<uni-icons type="calendar-filled" size="30" color="#C69200"></uni-icons>
					</view>
					<text class="intermission-title">{{ currentSeasonState.title }}</text>
					<text class="intermission-description">{{ currentSeasonState.description }}</text>
				</view>

				<template v-else>
					<view class="season-banner">
						<view class="season-banner-copy">
							<text class="season-kicker">进行中的赛季</text>
							<text class="season-name">{{ wall.current_season?.season_name }} · {{ currentGameTypeLabel }}</text>
							<text class="season-dates">{{ seasonDateText }}</text>
						</view>
						<view class="season-status"><text>独立累计</text></view>
					</view>

					<view class="content-card challenge-card">
						<view class="section-heading compact-heading">
							<text class="section-title">本赛季挑战</text>
							<text class="section-hint">固定 3 项</text>
						</view>
						<view class="challenge-list">
							<view v-for="challenge in currentChallenges" :key="challenge.key" class="challenge-row">
								<view class="challenge-icon"><text>{{ getChallengeEmoji(challenge.key) }}</text></view>
								<view class="challenge-copy">
									<view class="challenge-title-row">
										<text class="challenge-name">{{ challenge.name }}</text>
										<text class="challenge-progress" :class="{ complete: challenge.completed }">{{ challenge.progressText }}</text>
									</view>
									<text class="challenge-description">{{ challenge.description }}</text>
									<view class="progress-track season-progress-track">
										<view class="progress-fill" :style="{ width: `${challenge.progressPercent}%` }"></view>
									</view>
									<text class="progress-note">{{ challenge.remainingText }}</text>
								</view>
							</view>
						</view>
					</view>
				</template>
			</view>

			<view v-else class="tab-panel">
				<view class="section-heading page-section-heading">
					<view class="heading-copy">
						<text class="section-title">历届荣誉</text>
						<text class="section-description">赛季排名与赛事名次永久保留</text>
					</view>
					<text class="section-count">{{ wall.history.total || 0 }} 项</text>
				</view>

				<view v-if="isSelf && historySeasonOptions.length" class="content-card archive-selector-card">
					<view class="section-heading compact-heading">
						<text class="section-title">往届挑战记录</text>
						<text class="section-hint">只读快照</text>
					</view>
					<view class="archive-season-grid">
						<view
							v-for="season in historySeasonOptions"
							:key="season.id"
							class="archive-season-chip"
							:class="{ active: selectedHistorySeasonId === season.id }"
							@tap="selectHistorySeason(season.id)"
						>
							<text>{{ season.label }}</text>
						</view>
					</view>
					<view v-if="wall.history.challenge_season" class="archive-summary">
						<text class="archive-title">{{ wall.history.challenge_season.season_name }} · {{ currentGameTypeLabel }}</text>
						<view v-if="archivedChallenges.length" class="archive-challenge-list">
							<view v-for="item in archivedChallenges" :key="item.key" class="archive-challenge-row">
								<view class="archive-challenge-copy">
									<text class="archive-challenge-name">{{ item.name }}</text>
									<text class="archive-challenge-description">{{ item.description }}</text>
								</view>
								<text class="archive-challenge-progress" :class="{ complete: item.completed }">{{ item.progress }}/{{ item.threshold }}</text>
							</view>
						</view>
						<text v-else class="archive-empty-text">该赛季没有可回看的挑战记录</text>
					</view>
				</view>

				<view v-if="historyHonors.length" class="content-card history-card">
					<view class="history-list">
						<view v-for="item in historyHonors" :key="`${item.source_type}-${item.id}`" class="history-row">
							<view class="history-icon"><text>{{ getHonorEmoji(item) }}</text></view>
							<view class="history-copy">
								<text class="history-name">{{ item.name }}</text>
								<text class="history-description">{{ item.description || item.source_ref_name || '永久荣誉' }}</text>
								<text v-if="item.earned_at" class="history-time">{{ item.earned_at }}</text>
							</view>
							<text class="history-type">{{ item.source_type === 'season' ? '赛季' : '赛事' }}</text>
						</view>
					</view>
					<view v-if="hasMoreHistory" class="load-more" @tap="loadMoreHistory">
						<uni-icons v-if="historyLoading" type="spinner-cycle" size="16" color="#C69200"></uni-icons>
						<text>{{ historyLoading ? '加载中' : '加载更多荣誉' }}</text>
					</view>
				</view>
				<view v-else class="content-card state-panel compact-state">
					<text class="state-title">暂无历届荣誉</text>
					<text class="state-description">赛季排名或赛事名次产生后，会永久陈列在这里</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import { getHonorWall } from '@/api/achievement.js'
import { getNotificationList, markAsRead } from '@/api/notification.js'
import { useNotificationStore } from '@/store/notification.js'
import { getAchievementCategoryEmoji, groupAchievementsByCategory } from '@/utils/achievement-page.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import {
	buildChallengeViewModel,
	buildHistorySeasonOptions,
	buildHonorWallTabs,
	buildRecentHonorViewModel,
	getProgressPercent,
	normalizeHonorWallOptions,
	presentLatestSeasonRollover,
	resolveCurrentSeasonState
} from '@/utils/honor-wall.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const notificationStore = useNotificationStore()

const createEmptyWall = () => ({
	viewer_scope: 'self',
	profile: {},
	equipped_title: null,
	summary: {
		career_unlocked: 0,
		career_total: 0,
		season_honors: 0,
		tournament_honors: 0
	},
	recent_honors: [],
	career_achievements: [],
	current_season: null,
	history: {
		total: 0,
		honors: [],
		challenge_season: null,
		challenge_records: []
	}
})

const normalizeWall = (payload = {}) => ({
	...createEmptyWall(),
	...payload,
	profile: { ...(payload.profile || {}) },
	summary: { ...createEmptyWall().summary, ...(payload.summary || {}) },
	recent_honors: Array.isArray(payload.recent_honors) ? payload.recent_honors : [],
	career_achievements: Array.isArray(payload.career_achievements) ? payload.career_achievements : [],
	history: {
		...createEmptyWall().history,
		...(payload.history || {}),
		honors: Array.isArray(payload.history?.honors) ? payload.history.honors : [],
		challenge_records: Array.isArray(payload.history?.challenge_records) ? payload.history.challenge_records : []
	}
})

const loading = ref(true)
const refreshing = ref(false)
const historyLoading = ref(false)
const loadFailed = ref(false)
const loaded = ref(false)
const targetUserId = ref(0)
const currentGameType = ref(3)
const activeTab = ref('career')
const selectedHistorySeasonId = ref(0)
const historyPage = ref(1)
const historyPageSize = 20
const wall = ref(createEmptyWall())

const isSelf = computed(() => wall.value.viewer_scope !== 'friend')
const tabs = computed(() => buildHonorWallTabs(wall.value.viewer_scope))
const gameTypeTabs = GAME_TYPE_TABS
const currentGameTypeLabel = computed(() => (
	gameTypeTabs.find(item => item.value === currentGameType.value)?.label || '中式八球'
))
const profileAvatar = computed(() => resolveAvatarUrl(wall.value.profile.avatar, wall.value.profile.user_id))
const equippedTitleSource = computed(() => {
	const title = wall.value.equipped_title
	if (!title) return isSelf.value ? '可前往称号管理装备' : 'TA 暂未装备称号'
	if (title.source_ref_name) return `来自 ${title.source_ref_name}`
	return title.source_type === 'season' ? '赛季荣誉称号' : title.source_type === 'tournament' ? '赛事荣誉称号' : '生涯成就称号'
})
const summaryItems = computed(() => ([
	{ key: 'career', label: '生涯成就', value: `${wall.value.summary.career_unlocked || 0}/${wall.value.summary.career_total || 0}` },
	{ key: 'season', label: '赛季荣誉', value: wall.value.summary.season_honors || 0 },
	{ key: 'tournament', label: '赛事荣誉', value: wall.value.summary.tournament_honors || 0 }
]))
const recentHonors = computed(() => wall.value.recent_honors.map(buildRecentHonorViewModel))
const achievementGroups = computed(() => groupAchievementsByCategory(wall.value.career_achievements))
const currentSeasonState = computed(() => resolveCurrentSeasonState(wall.value.current_season))
const currentChallenges = computed(() => (
	(wall.value.current_season?.challenges || []).map(buildChallengeViewModel)
))
const archivedChallenges = computed(() => wall.value.history.challenge_records.map(buildChallengeViewModel))
const historyHonors = computed(() => wall.value.history.honors || [])
const historySeasonOptions = computed(() => buildHistorySeasonOptions(historyHonors.value))
const hasMoreHistory = computed(() => historyHonors.value.length < Number(wall.value.history.total || 0))
const seasonDateText = computed(() => {
	const season = wall.value.current_season
	if (!season) return ''
	return `${formatDate(season.start_date)} - ${formatDate(season.end_date)}`
})

const formatDate = (value = '') => String(value || '').slice(0, 10).replace(/-/g, '.')
const getCategoryEmoji = getAchievementCategoryEmoji
const getHonorEmoji = (item = {}) => {
	if (item.source_type === 'season') return '👑'
	if (item.source_type === 'tournament') return '🏆'
	return '🏅'
}
const getChallengeEmoji = (key = '') => {
	if (key.includes('wins')) return '🔥'
	if (key.includes('tournament')) return '🏆'
	return '🎱'
}
const getCareerProgressPercent = (item) => getProgressPercent({
	progress: item.progress,
	threshold: item.threshold,
	unlocked: item.unlocked
})
const getCareerProgressText = (item) => {
	if (item.unlocked) return item.unlocked_at ? `已解锁 · ${item.unlocked_at}` : '已解锁'
	return `${Number(item.progress || 0)}/${Number(item.threshold || 0)}`
}

const buildRequestParams = () => {
	const params = {
		game_type: currentGameType.value,
		history_page: historyPage.value,
		history_page_size: historyPageSize
	}
	if (targetUserId.value > 0) params.user_id = targetUserId.value
	if (selectedHistorySeasonId.value > 0) params.history_season_id = selectedHistorySeasonId.value
	return params
}

const loadData = async ({ appendHistory = false } = {}) => {
	if (appendHistory ? historyLoading.value : refreshing.value) return
	if (appendHistory) {
		historyLoading.value = true
	} else if (!loaded.value) {
		loading.value = true
	} else {
		refreshing.value = true
	}
	loadFailed.value = false

	try {
		const response = await getHonorWall(buildRequestParams())
		if (!response?.success) throw new Error(response?.message || '荣誉墙加载失败')
		const normalized = normalizeWall(response)
		if (appendHistory) {
			normalized.history.honors = [...historyHonors.value, ...normalized.history.honors]
		}
		wall.value = normalized
		loaded.value = true
		const allowedTabs = tabs.value.map(item => item.key)
		if (!allowedTabs.includes(activeTab.value)) activeTab.value = 'career'
		if (isSelf.value && !appendHistory) showLatestRollover()
	} catch (error) {
		console.error('加载荣誉墙失败:', error)
		loadFailed.value = true
		if (appendHistory) historyPage.value = Math.max(1, historyPage.value - 1)
		if (loaded.value) {
			uni.showToast({ title: error.message || '荣誉墙更新失败', icon: 'none' })
		}
	} finally {
		loading.value = false
		refreshing.value = false
		historyLoading.value = false
		uni.stopPullDownRefresh()
	}
}

const showLatestRollover = () => {
	presentLatestSeasonRollover({
		getNotificationList,
		markAsRead,
		showModal: options => uni.showModal(options),
		onRead: () => notificationStore.fetchUnreadCount()
	}).catch(error => console.error('展示换季结果失败:', error))
}

const switchTab = (tab) => {
	if (activeTab.value === tab) return
	activeTab.value = tab
}

const handleGameTypeChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	historyPage.value = 1
	loadData()
}

const selectHistorySeason = (seasonId) => {
	if (selectedHistorySeasonId.value === seasonId) return
	selectedHistorySeasonId.value = seasonId
	historyPage.value = 1
	loadData()
}

const loadMoreHistory = () => {
	if (!hasMoreHistory.value || historyLoading.value) return
	historyPage.value += 1
	loadData({ appendHistory: true })
}

const goToDetail = (id) => {
	if (!isSelf.value || !id) return
	uni.navigateTo({ url: `/subPages/achievement/detail?id=${id}` })
}

const goToTitles = () => {
	if (!isSelf.value) return
	uni.navigateTo({ url: '/subPages/achievement/titles' })
}

onLoad((options) => {
	const normalized = normalizeHonorWallOptions(options)
	targetUserId.value = normalized.userId
	currentGameType.value = normalized.gameType
	activeTab.value = normalized.activeTab
	selectedHistorySeasonId.value = normalized.historySeasonId
})

onShow(() => {
	historyPage.value = 1
	loadData()
})

onPullDownRefresh(() => {
	historyPage.value = 1
	loadData()
})

onReachBottom(() => {
	if (activeTab.value === 'history') loadMoreHistory()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
