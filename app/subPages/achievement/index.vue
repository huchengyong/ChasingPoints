<template>
	<view class="honor-wall-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading && !loaded" class="state-panel full-state">
			<uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
			<text class="state-title">正在整理荣誉墙</text>
			<text class="state-description">生涯成就、赛季挑战与历届荣誉即将呈现</text>
		</view>

		<view v-else-if="sessionHandled && !loaded" class="state-panel full-state">
			<uni-icons type="info-filled" size="40" color="#E0AE12"></uni-icons>
			<text class="state-title">正在重新登录</text>
			<text class="state-description">登录状态已更新，即将返回登录页</text>
		</view>

		<view v-else-if="loadFailed && !loaded" class="state-panel full-state">
			<uni-icons type="info-filled" size="40" color="#E0AE12"></uni-icons>
			<text class="state-title">{{ loadErrorState?.title || '荣誉墙暂时没加载出来' }}</text>
			<text class="state-description">{{ loadErrorState?.description || '请稍后重试，已有荣誉不会受到影响' }}</text>
			<button v-if="loadErrorState?.showRetry" class="retry-button" @tap="loadData">
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

				<view class="equipped-title-card" :class="{ interactive: isSelf }" @tap="openTitleSelector">
					<view class="title-medal">
						<uni-icons type="medal-filled" size="25" color="#ffffff"></uni-icons>
					</view>
					<view class="title-copy">
						<text class="title-label">当前称号</text>
						<text class="title-name">{{ wall.equipped_title?.title_name || '暂未佩戴称号' }}</text>
						<text class="title-source">{{ equippedTitleSource }}</text>
					</view>
					<text v-if="isSelf" class="title-action">{{ wall.equipped_title ? '更换' : '选择' }}</text>
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
						<text class="section-description">永久累计，不随赛季清零；球种绝技独立展示</text>
					</view>
				</view>

				<view v-if="isSelf && !upcomingSection.hidden" class="content-card upcoming-card">
					<view class="section-heading compact-heading">
						<text class="section-title">即将达成</text>
						<text class="section-hint">通用 + {{ currentGameTypeLabel }}</text>
					</view>
					<view v-if="upcomingSection.items.length" class="upcoming-list">
						<view
							v-for="item in upcomingSection.items"
							:key="item.id"
							class="achievement-row upcoming-row interactive"
							@tap="goToDetail(item)"
						>
							<view class="achievement-icon locked">
								<image v-if="hasAchievementIcon(item)" :src="item.icon" mode="aspectFit" @error="handleAchievementIconError(item)"></image>
								<text v-else>{{ getFallbackEmoji(item) }}</text>
							</view>
							<view class="achievement-copy">
								<view class="achievement-name-row">
									<text class="achievement-name">{{ item.name }}</text>
									<text class="upcoming-percent">{{ item.progressPercent }}%</text>
								</view>
								<view class="progress-track">
									<view class="progress-fill" :style="{ width: `${item.progressPercent}%` }"></view>
								</view>
								<text class="progress-note">{{ item.progressText }} · {{ item.remainingText }}</text>
							</view>
							<uni-icons type="right" size="16" :color="isDarkMode ? '#9f926e' : '#94a3b8'"></uni-icons>
						</view>
					</view>
					<view v-else class="inline-empty upcoming-complete">
						<text>{{ upcomingEmptyText }}</text>
					</view>
				</view>

				<view class="section-heading page-section-heading career-section-heading">
					<view class="heading-copy">
						<text class="section-title">通用里程碑</text>
						<text class="section-description">对局、胜场、连胜与赛事历程跨球种累计</text>
					</view>
					<text class="section-count">{{ wall.summary.universal_unlocked || 0 }}/{{ wall.summary.universal_total || 0 }}</text>
				</view>
				<view v-if="universalGroups.length" class="achievement-groups">
					<view v-for="group in universalGroups" :key="group.key" class="content-card achievement-group">
						<view class="group-header">
							<view class="group-title-wrap">
								<text class="group-emoji">{{ getCategoryEmoji(group.key) }}</text>
								<text class="group-title">{{ group.label }}</text>
							</view>
							<text class="group-count">{{ group.list.length }} 项</text>
						</view>
						<view
							v-for="item in group.list"
							:key="item.id"
							class="achievement-row"
							:class="{ unlocked: item.unlocked, interactive: isSelf }"
							@tap="goToDetail(item)"
						>
							<view class="achievement-icon" :class="{ locked: !item.unlocked }">
								<image v-if="hasAchievementIcon(item)" :src="item.icon" mode="aspectFit" @error="handleAchievementIconError(item)"></image>
								<text v-else>{{ getFallbackEmoji(item) }}</text>
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
					<text class="state-title">暂无通用里程碑</text>
					<text class="state-description">完成有效比赛后，通用进度会自动更新</text>
				</view>

				<view class="section-heading page-section-heading career-section-heading">
					<view class="heading-copy">
						<text class="section-title">球种绝技</text>
						<text class="section-description">只计算当前球种，不要求跨球种集齐</text>
					</view>
					<text class="section-count">{{ wall.summary.specialty_unlocked || 0 }}/{{ wall.summary.specialty_total || 0 }}</text>
				</view>
				<view class="game-type-grid career-game-type-grid">
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
				<view v-if="specialtyAchievements.length" class="content-card specialty-card">
					<view class="group-header">
						<view class="group-title-wrap">
							<text class="group-emoji">{{ getFallbackEmoji(specialtyAchievements[0]) }}</text>
							<text class="group-title">{{ currentGameTypeLabel }}绝技</text>
						</view>
						<text class="group-count">{{ specialtyAchievements.length }} 项</text>
					</view>
					<view
						v-for="item in specialtyAchievements"
						:key="item.id"
						class="achievement-row"
						:class="{ unlocked: item.unlocked, interactive: isSelf }"
						@tap="goToDetail(item)"
					>
						<view class="achievement-icon" :class="{ locked: !item.unlocked }">
							<image v-if="hasAchievementIcon(item)" :src="item.icon" mode="aspectFit" @error="handleAchievementIconError(item)"></image>
							<text v-else>{{ getFallbackEmoji(item) }}</text>
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
				<view v-else class="content-card state-panel compact-state">
					<text class="state-title">暂无{{ currentGameTypeLabel }}绝技</text>
					<text class="state-description">{{ isSelf ? '完成该球种的特殊战绩后会在这里点亮' : 'TA 尚未解锁该球种绝技' }}</text>
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

				<view v-if="currentSeasonState.mode !== 'active'" class="content-card intermission-card">
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
								<view class="challenge-icon" :class="{ 'has-image': challenge.icon }">
									<image v-if="challenge.icon" :src="challenge.icon" mode="aspectFit"></image>
									<text v-else>{{ getChallengeEmoji(challenge.key) }}</text>
								</view>
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
								<view class="challenge-icon" :class="{ 'has-image': item.icon }">
									<image v-if="item.icon" :src="item.icon" mode="aspectFit"></image>
									<text v-else>{{ getChallengeEmoji(item.key) }}</text>
								</view>
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

			<view v-if="isSelf && showTitleSelector" class="title-selector-layer" @touchmove.stop>
				<view class="title-selector-mask" @tap="closeTitleSelector"></view>
				<view class="title-selector-panel" @tap.stop>
					<view class="title-selector-handle"></view>
					<view class="title-selector-header">
						<view class="title-selector-heading">
							<text class="title-selector-title">选择佩戴称号</text>
							<text class="title-selector-description">选择后立即生效，仅改变对外展示</text>
						</view>
						<view class="title-selector-close" @tap="closeTitleSelector">
							<uni-icons type="closeempty" size="20" :color="isDarkMode ? '#b9aa83' : '#64748b'"></uni-icons>
						</view>
					</view>

					<view v-if="titleListLoading" class="title-selector-state title-selector-loading">
						<uni-icons type="spinner-cycle" size="32" color="#E0AE12"></uni-icons>
						<text class="title-selector-state-title">正在加载已获称号</text>
					</view>

					<view v-else-if="titleListFailed" class="title-selector-state title-selector-failed">
						<uni-icons type="info-filled" size="32" color="#E0AE12"></uni-icons>
						<text class="title-selector-state-title">称号列表暂时没加载出来</text>
						<text class="title-selector-state-description">当前佩戴状态不会改变，可以重新加载后再选择</text>
						<button class="title-selector-retry" @tap="loadTitles">
							<text>重新加载称号</text>
						</button>
					</view>

					<view v-else-if="titleListLoaded && titleList.length === 0" class="title-selector-state title-selector-empty">
						<view class="title-selector-empty-icon"><text>🎖️</text></view>
						<text class="title-selector-state-title">暂无可佩戴称号</text>
						<text class="title-selector-state-description">解锁生涯成就或获得赛季、赛事荣誉后，称号会出现在这里</text>
						<view class="title-options empty-title-options">
							<view
								class="title-option none-option selected"
								:class="{ disabled: titleSubmitting }"
								@tap="selectNoTitle"
							>
								<view class="title-option-icon none"><text>—</text></view>
								<view class="title-option-copy">
									<text class="title-option-name">不佩戴称号</text>
									<text class="title-option-source">已获称号为空，当前没有对外展示的称号</text>
								</view>
								<view class="title-option-check">
									<uni-icons type="checkmarkempty" size="14" color="#ffffff"></uni-icons>
								</view>
							</view>
						</view>
					</view>

					<scroll-view v-else class="title-selector-list" scroll-y :show-scrollbar="false">
						<view class="title-options">
							<view
								v-for="item in titleOptions"
								:key="item.id"
								class="title-option"
								:class="{ selected: item.equipped, disabled: titleSubmitting }"
								@tap="selectTitle(item)"
							>
								<view class="title-option-icon" :class="getTitleSourceClass(item.source_type || item.source)">
									<uni-icons type="medal-filled" size="21" color="currentColor"></uni-icons>
								</view>
								<view class="title-option-copy">
									<text class="title-option-name">{{ item.title_name }}</text>
									<text class="title-option-source">{{ getTitleSourceDescription(item) }}</text>
								</view>
								<view class="title-option-check">
									<uni-icons v-if="item.equipped" type="checkmarkempty" size="14" color="#ffffff"></uni-icons>
								</view>
							</view>

							<view
								class="title-option none-option"
								:class="{ selected: currentTitleId === 0, disabled: titleSubmitting }"
								@tap="selectNoTitle"
							>
								<view class="title-option-icon none"><text>—</text></view>
								<view class="title-option-copy">
									<text class="title-option-name">不佩戴称号</text>
									<text class="title-option-source">隐藏对外展示，已获称号仍永久保留</text>
								</view>
								<view class="title-option-check">
									<uni-icons v-if="currentTitleId === 0" type="checkmarkempty" size="14" color="#ffffff"></uni-icons>
								</view>
							</view>
						</view>
					</scroll-view>
				</view>
			</view>
		</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onBackPress, onLoad, onPullDownRefresh, onReachBottom, onShow } from '@dcloudio/uni-app'
import { equipTitle, getHonorWall, getUserTitles } from '@/api/achievement.js'
import { getNotificationList, markAsRead } from '@/api/notification.js'
import { useNotificationStore } from '@/store/notification.js'
import { useUserStore } from '@/store/user.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { cacheAchievementDetail } from '@/utils/achievement-detail-cache.js'
import {
	filterSpecialtyAchievements,
	getAchievementCategoryEmoji,
	getAchievementFallbackEmoji,
	getAchievementGameTypeLabel,
	getTitleSourceClass,
	getTitleSourceDescription,
	groupUniversalAchievements,
	resolveTitleSelection,
	sortTitleOptions
} from '@/utils/achievement-page.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import {
	buildCareerSummaryItems,
	buildChallengeViewModel,
	buildHistorySeasonOptions,
	buildHonorWallTabs,
	buildRecentHonorViewModel,
	buildUpcomingAchievementSection,
	createLatestRequestGuard,
	getProgressPercent,
	normalizeHonorWallOptions,
	presentLatestSeasonRollover,
	resolveCurrentSeasonState,
	resolveHonorWallLoadError
} from '@/utils/honor-wall.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const notificationStore = useNotificationStore()
const userStore = useUserStore()
const userDataInvalidationStore = useUserDataInvalidationStore()

const createEmptyWall = () => ({
	viewer_scope: 'self',
	profile: {},
	equipped_title: null,
	summary: {
		career_unlocked: 0,
		career_total: 0,
		universal_unlocked: 0,
		universal_total: 0,
		specialty_game_type: 3,
		specialty_unlocked: 0,
		specialty_total: 0,
		season_honors: 0,
		tournament_honors: 0
	},
	recent_honors: [],
	career_achievements: [],
	season_state: 'not_started',
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
const loadErrorState = ref(null)
const sessionHandled = ref(false)
const loaded = ref(false)
const targetUserId = ref(0)
const currentGameType = ref(3)
const activeTab = ref('career')
const selectedHistorySeasonId = ref(0)
const historyPage = ref(1)
const historyPageSize = 20
const wall = ref(createEmptyWall())
const showTitleSelector = ref(false)
const titleList = ref([])
const titleListLoading = ref(false)
const titleListLoaded = ref(false)
const titleListFailed = ref(false)
const titleSubmitting = ref(false)
const brokenAchievementIcons = ref({})
const wallRequestGuard = createLatestRequestGuard()
const requestedIdentityKey = ref('')
const requestedHonorScopeVersion = ref(0)
const loadedIdentityKey = ref('')
const loadedHonorScopeVersion = ref(0)

const currentReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const currentIdentityKey = () => {
	const identity = currentReadIdentity()
	return `${identity.userId}:${identity.authGeneration}`
}

const isSelf = computed(() => wall.value.viewer_scope !== 'friend')
const tabs = computed(() => buildHonorWallTabs(wall.value.viewer_scope))
const currentTitleId = computed(() => Number(wall.value.equipped_title?.id || 0))
const titleOptions = computed(() => sortTitleOptions(titleList.value))
const gameTypeTabs = GAME_TYPE_TABS
const currentGameTypeLabel = computed(() => (
	gameTypeTabs.find(item => item.value === currentGameType.value)?.label || getAchievementGameTypeLabel(currentGameType.value)
))
const profileAvatar = computed(() => resolveAvatarUrl(wall.value.profile.avatar, wall.value.profile.user_id))
const equippedTitleSource = computed(() => {
	const title = wall.value.equipped_title
	if (!title) return isSelf.value ? '选择一个已获称号进行展示' : 'TA 暂未佩戴称号'
	return getTitleSourceDescription(title)
})
const summaryItems = computed(() => buildCareerSummaryItems(
	wall.value.summary,
	currentGameType.value,
	currentGameTypeLabel.value
))
const recentHonors = computed(() => wall.value.recent_honors.map(buildRecentHonorViewModel))
const universalGroups = computed(() => groupUniversalAchievements(wall.value.career_achievements))
const specialtyAchievements = computed(() => filterSpecialtyAchievements(
	wall.value.career_achievements,
	currentGameType.value,
	{ unlockedOnly: !isSelf.value }
))
const upcomingSection = computed(() => buildUpcomingAchievementSection(
	wall.value.career_achievements,
	currentGameType.value,
	wall.value.viewer_scope
))
const upcomingEmptyText = computed(() => (
	upcomingSection.value.completed
		? `通用成就与${currentGameTypeLabel.value}绝技已全部达成`
		: '暂无可追踪的成就目标'
))
const currentSeasonState = computed(() => resolveCurrentSeasonState(
	wall.value.current_season,
	wall.value.season_state
))
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
const getFallbackEmoji = getAchievementFallbackEmoji
const achievementIconKey = (item = {}) => String(item.id || item.key || '')
const hasAchievementIcon = (item = {}) => Boolean(item.icon && !brokenAchievementIcons.value[achievementIconKey(item)])
const handleAchievementIconError = (item = {}) => {
	const key = achievementIconKey(item)
	if (!key) return
	brokenAchievementIcons.value = { ...brokenAchievementIcons.value, [key]: true }
}
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
	if (appendHistory && (historyLoading.value || loading.value || refreshing.value)) return
	const requestId = wallRequestGuard.next()
	const requestIdentityKey = currentIdentityKey()
	const requestScopeVersion = userDataInvalidationStore.versionOf('honor')
	requestedIdentityKey.value = requestIdentityKey
	requestedHonorScopeVersion.value = requestScopeVersion
	if (!appendHistory && loadedIdentityKey.value && loadedIdentityKey.value !== requestIdentityKey) {
		wall.value = createEmptyWall()
		titleList.value = []
		titleListLoaded.value = false
		titleListLoading.value = false
		loaded.value = false
	}
	const requestParams = buildRequestParams()
	if (appendHistory) {
		historyLoading.value = true
	} else if (!loaded.value) {
		loading.value = true
	} else {
		refreshing.value = true
	}
	loadFailed.value = false
	loadErrorState.value = null
	sessionHandled.value = false

	try {
		const response = await getHonorWall(requestParams)
		if (!response?.success) throw new Error(response?.message || '荣誉墙加载失败')
		if (!wallRequestGuard.isLatest(requestId) || currentIdentityKey() !== requestIdentityKey) return
		const normalized = normalizeWall(response)
		if (appendHistory) {
			normalized.history.honors = [...historyHonors.value, ...normalized.history.honors]
		}
		wall.value = normalized
		loaded.value = true
		const allowedTabs = tabs.value.map(item => item.key)
		if (!allowedTabs.includes(activeTab.value)) activeTab.value = 'career'
		if (isSelf.value && !appendHistory) showLatestRollover()
		loadedIdentityKey.value = requestIdentityKey
		loadedHonorScopeVersion.value = requestScopeVersion
	} catch (error) {
		if (!wallRequestGuard.isLatest(requestId) || currentIdentityKey() !== requestIdentityKey) return
		console.error('加载荣誉墙失败:', error)
		const loadError = resolveHonorWallLoadError({ error })
		if (loadError.kind === 'superseded') {
			loadFailed.value = true
			loadErrorState.value = loadError
			if (appendHistory) historyPage.value = Math.max(1, historyPage.value - 1)
			return
		}
		if (loadError.kind === 'session') {
			// 全局会话失效流程已接管清理/提示/导航，页面不展示失败态、不重复提示
			sessionHandled.value = true
			return
		}
		loadFailed.value = true
		loadErrorState.value = loadError
		if (appendHistory) historyPage.value = Math.max(1, historyPage.value - 1)
		if (loaded.value) {
			uni.showToast({ title: error.message || '荣誉墙更新失败', icon: 'none' })
		}
	} finally {
		if (!wallRequestGuard.isLatest(requestId) || currentIdentityKey() !== requestIdentityKey) return
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
		onRead: () => notificationStore.fetchUnreadCount({
			userId: userStore.userId,
			authGeneration: userStore.authGeneration
		})
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

const goToDetail = (achievement) => {
	const id = Number(achievement?.id || 0)
	if (!isSelf.value || !id) return
	cacheAchievementDetail({
		userId: userStore.userId,
		authGeneration: userStore.authGeneration
	}, achievement)
	uni.navigateTo({ url: `/subPages/achievement/detail?id=${id}` })
}

const syncEquippedTitleFromList = () => {
	const equipped = titleList.value.find(item => item.equipped) || null
	wall.value.equipped_title = equipped ? { ...equipped } : null
}

const loadTitles = async () => {
	if (titleListLoading.value) return
	const requestIdentityKey = currentIdentityKey()
	titleListLoading.value = true
	titleListFailed.value = false

	try {
		const response = await getUserTitles()
		if (currentIdentityKey() !== requestIdentityKey) return
		const list = Array.isArray(response?.list) ? response.list : Array.isArray(response) ? response : []
		titleList.value = list
		titleListLoaded.value = true
		syncEquippedTitleFromList()
	} catch (error) {
		if (currentIdentityKey() !== requestIdentityKey) return
		console.error('加载称号列表失败:', error)
		titleListFailed.value = true
	} finally {
		if (currentIdentityKey() === requestIdentityKey) titleListLoading.value = false
	}
}

const openTitleSelector = async () => {
	if (!isSelf.value) return
	showTitleSelector.value = true
	if (titleListLoaded.value) {
		syncEquippedTitleFromList()
		return
	}
	await loadTitles()
}

const closeTitleSelector = () => {
	showTitleSelector.value = false
}

const submitTitleSelection = async (item = null) => {
	if (titleSubmitting.value) return
	const selection = resolveTitleSelection({
		currentTitleId: currentTitleId.value,
		selectedTitleId: item?.id || 0
	})
	if (selection.action === 'close') {
		closeTitleSelector()
		return
	}

	titleSubmitting.value = true
	try {
		await equipTitle({
			title_id: selection.titleId,
			equip: selection.action === 'equip'
		})
		if (selection.action === 'equip') {
			titleList.value = titleList.value.map(title => ({
				...title,
				equipped: Number(title.id) === selection.titleId
			}))
			const equipped = titleList.value.find(title => title.equipped)
			wall.value.equipped_title = equipped ? { ...equipped } : null
			closeTitleSelector()
			uni.showToast({ title: `已佩戴「${equipped?.title_name || ''}」`, icon: 'none' })
			return
		}

		titleList.value = titleList.value.map(title => ({ ...title, equipped: false }))
		wall.value.equipped_title = null
		closeTitleSelector()
		uni.showToast({ title: '已停止展示称号', icon: 'none' })
	} catch (error) {
		console.error('切换称号失败:', error)
		uni.showToast({ title: error.message || '称号切换失败', icon: 'none' })
	} finally {
		titleSubmitting.value = false
	}
}

const selectTitle = (item) => submitTitleSelection(item)
const selectNoTitle = () => submitTitleSelection()

onLoad((options) => {
	const normalized = normalizeHonorWallOptions(options)
	targetUserId.value = normalized.userId
	currentGameType.value = normalized.gameType
	activeTab.value = normalized.activeTab
	selectedHistorySeasonId.value = normalized.historySeasonId
})

onShow(() => {
	const identityKey = currentIdentityKey()
	const scopeVersion = userDataInvalidationStore.versionOf('honor')
	if (requestedIdentityKey.value === identityKey && requestedHonorScopeVersion.value === scopeVersion && (loading.value || refreshing.value)) return
	if (loaded.value && requestedIdentityKey.value === identityKey && requestedHonorScopeVersion.value === scopeVersion && loadedIdentityKey.value === identityKey && loadedHonorScopeVersion.value === scopeVersion) return
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

onBackPress(() => {
	if (!showTitleSelector.value) return false
	closeTitleSelector()
	return true
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
