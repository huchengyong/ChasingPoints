<template>
	<view class="detail-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="skeleton">
			<view class="skeleton-cover"></view>
			<view class="skeleton-line skeleton-line--title"></view>
			<view class="skeleton-line"></view>
			<view class="skeleton-line skeleton-line--short"></view>
		</view>

		<view v-else-if="eventView" class="detail-content">
			<view v-if="eventDetailRefreshError" class="refresh-error-banner">
				<text>{{ eventDetailRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryEventDetail">重试</text>
			</view>

			<view class="event-header">
				<image class="event-cover" :src="eventView.coverImage" mode="aspectFill"></image>
				<view class="event-header-body">
					<view class="event-tags">
						<view class="status-badge" :class="'status-badge--' + eventView.statusTone">
							<view v-if="eventView.statusTone === 'live'" class="status-dot"></view>
							<text>{{ eventView.statusLabel }}</text>
						</view>
						<text class="event-tag">{{ eventView.gameTypeText }}</text>
						<text v-if="eventView.sourceText" class="event-tag">{{ eventView.sourceText }}</text>
					</view>
					<text class="event-title">{{ eventView.title }}</text>
					<text v-if="eventView.originalTitle" class="event-original-title">{{ eventView.originalTitle }}</text>
					<view class="event-facts">
						<view class="event-fact">
							<uni-icons type="calendar" size="15" :color="mutedIconColor"></uni-icons>
							<text class="event-fact-text">{{ eventView.showTime ? eventView.timeText : eventView.dateText }}</text>
						</view>
						<view v-if="eventView.regionText" class="event-fact">
							<uni-icons type="location" size="15" :color="mutedIconColor"></uni-icons>
							<text class="event-fact-text">{{ eventView.regionText }}</text>
						</view>
					</view>
				</view>
			</view>

			<view v-if="eventView.champion" class="champion-card">
				<view class="champion-avatar-frame">
					<image class="champion-avatar" :src="eventView.champion.avatar" mode="widthFix"></image>
				</view>
				<view class="champion-copy">
					<view class="champion-label">
						<uni-icons type="medal-filled" size="16" color="#F0C542"></uni-icons>
						<text class="champion-label-text">冠军</text>
					</view>
					<view class="champion-name-row">
						<image v-if="eventView.champion.flag.type === 'image'" class="flag-img" :src="eventView.champion.flag.value" mode="aspectFit"></image>
						<text v-else-if="eventView.champion.flag.type === 'emoji'" class="flag-emoji">{{ eventView.champion.flag.value }}</text>
						<text class="champion-name">{{ eventView.champion.name }}</text>
					</view>
					<text class="champion-desc">决赛 {{ eventView.champion.scoreText }} 胜 {{ eventView.champion.opponentName }}</text>
				</view>
			</view>

			<view class="schedule">
				<view class="schedule-head">
					<text class="schedule-title">赛程赛果</text>
					<text class="schedule-tip">{{ eventView.matchCountText }}</text>
				</view>

				<template v-if="eventView.rounds.length">
					<view id="round-tabs-anchor"></view>
					<view class="round-tabs-bar">
						<scroll-view
							scroll-x
							class="round-tabs"
							:scroll-into-view="roundTabIntoView"
							scroll-with-animation
							:show-scrollbar="false"
						>
							<view class="round-tabs-track">
								<view
									v-for="(round, index) in eventView.rounds"
									:id="'round-tab-' + index"
									:key="round.key"
									class="round-tab"
									:class="{ 'round-tab--active': activeRound && round.key === activeRound.key }"
									@tap="selectRound(round.key)"
								>
									<view v-if="round.hasLive" class="round-tab-live"></view>
									<text class="round-tab-text">{{ round.roundName }}</text>
								</view>
							</view>
						</scroll-view>
					</view>

					<view class="round-summary">
						<text>{{ activeRound.matchCount }} 场比赛</text>
						<text v-if="activeRound.metaText">{{ activeRound.metaText }}</text>
					</view>

					<view v-for="day in activeRound.days" :key="day.key" class="day-group">
						<text class="day-label">{{ day.label }}</text>
						<view
							v-for="match in day.matches"
							:key="match.id"
							class="match-card"
							:class="{ 'match-card--live': match.isLive }"
						>
							<view class="match-head">
								<text class="match-time">{{ match.clockText }}</text>
								<text v-if="match.metaText && !activeRound.metaText" class="match-format">{{ match.metaText }}</text>
								<view v-if="match.isLive" class="status-badge status-badge--live">
									<view class="status-dot"></view>
									<text>进行中</text>
								</view>
								<view v-else-if="match.status === 3" class="status-badge status-badge--canceled">
									<text>已取消</text>
								</view>
							</view>
							<view
								class="match-player"
								:class="{ 'is-winner': match.winnerSide === 1, 'is-loser': match.winnerSide === 2 }"
							>
								<view class="player-avatar-frame">
									<image class="player-avatar" :src="match.homePlayerAvatar" mode="widthFix" lazy-load></image>
								</view>
								<image v-if="match.homePlayerFlag.type === 'image'" class="flag-img" :src="match.homePlayerFlag.value" mode="aspectFit"></image>
								<text v-else-if="match.homePlayerFlag.type === 'emoji'" class="flag-emoji">{{ match.homePlayerFlag.value }}</text>
								<text class="player-name">{{ match.homePlayerName }}</text>
								<text v-if="match.showScore" class="player-score">{{ match.homeScore }}</text>
							</view>
							<view
								class="match-player"
								:class="{ 'is-winner': match.winnerSide === 2, 'is-loser': match.winnerSide === 1 }"
							>
								<view class="player-avatar-frame">
									<image class="player-avatar" :src="match.awayPlayerAvatar" mode="widthFix" lazy-load></image>
								</view>
								<image v-if="match.awayPlayerFlag.type === 'image'" class="flag-img" :src="match.awayPlayerFlag.value" mode="aspectFit"></image>
								<text v-else-if="match.awayPlayerFlag.type === 'emoji'" class="flag-emoji">{{ match.awayPlayerFlag.value }}</text>
								<text class="player-name">{{ match.awayPlayerName }}</text>
								<text v-if="match.showScore" class="player-score">{{ match.awayScore }}</text>
							</view>
						</view>
					</view>
				</template>

				<view v-else class="schedule-empty">
					<text class="schedule-empty-title">赛程待更新</text>
					<text class="schedule-empty-text">当前赛事还没有录入逐场赛程与赛果。</text>
				</view>
			</view>

			<text v-if="eventView.sourceText" class="source-note">数据来源：{{ eventView.sourceText }}</text>
		</view>

		<view v-else-if="eventDetailPageError" class="empty-state">
			<view class="empty-icon">
				<uni-icons type="calendar" size="32" color="#E0AE12"></uni-icons>
			</view>
			<text class="empty-title">{{ eventDetailPageError.title }}</text>
			<text class="empty-text">{{ eventDetailPageError.description }}</text>
			<view class="empty-actions">
				<button class="empty-btn primary" @tap="handleEventDetailErrorAction">{{ eventDetailPageError.actionText }}</button>
				<button class="empty-btn secondary" @tap="goBack">返回上一页</button>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { onHide, onLoad, onPullDownRefresh, onShow, onUnload } from '@dcloudio/uni-app'
import { usePublicReadStore } from '@/store/publicRead.js'
import { pickEventNewsViewPayload } from '@/utils/event-news-response.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	DEFAULT_EVENT_COVER,
	buildEventLocationText,
	buildEventRegionText,
	buildSaiXunDetailRounds,
	formatEventDateRange,
	formatEventTimeRange,
	localizeCountryName,
	normalizeSaiXunCard,
	resolveSaiXunChampion,
	resolveSaiXunFocusRoundKey
} from '@/utils/saixun.js'
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
const publicReadStore = usePublicReadStore()

const loading = ref(true)
const eventNewsId = ref(0)
const eventView = ref(null)
const rawEventPayload = ref(null)
const selectedRoundKey = ref('')
const roundTabIntoView = ref('')
const eventDetailState = ref(createAsyncPageState({ data: null }))
const eventDetailPageError = computed(() => (
	eventDetailState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(eventDetailState.value.error, { resource: '赛事详情' })
		: null
))
const eventDetailRefreshError = computed(() => (
	eventDetailState.value.refreshError
		? resolveAsyncPageErrorFeedback(eventDetailState.value.refreshError, { resource: '赛事详情' })
		: null
))
const mutedIconColor = computed(() => (isDarkMode.value ? '#9F926E' : '#9A8C67'))
// 定时重渲染或刷新后保留用户手动选择的轮次；该轮次消失时回到默认焦点轮次
const activeRound = computed(() => {
	const rounds = eventView.value?.rounds || []
	return rounds.find((round) => round.key === selectedRoundKey.value)
		|| rounds.find((round) => round.key === eventView.value.focusRoundKey)
		|| null
})

let eventViewRefreshTimer = null

const renderEventView = (now = Date.now()) => {
	if (!rawEventPayload.value) return
	eventView.value = normalizeEventView(rawEventPayload.value, now)
}

const normalizeEventView = ({ eventNews, tournament, matches }, now = Date.now()) => {
	const mergedEvent = {
		...eventNews,
		tournament_name: eventNews.tournament_name || tournament?.name || eventNews.title || '',
		cover_image: eventNews.cover_image || tournament?.cover_image || '',
		country: localizeCountryName(eventNews.country || tournament?.country || ''),
		city: eventNews.city || tournament?.city || '',
		venue: eventNews.venue || tournament?.venue_name || '',
		start_date: eventNews.start_date || tournament?.start_date || '',
		end_date: eventNews.end_date || tournament?.end_date || '',
		start_time: eventNews.start_time || tournament?.start_time || '',
		end_time: eventNews.end_time || tournament?.end_time || '',
		match_count: Array.isArray(matches) ? matches.length : Number(eventNews.match_count || 0)
	}
	const eventCard = normalizeSaiXunCard(mergedEvent, now)
	const rounds = buildSaiXunDetailRounds(Array.isArray(matches) ? matches : [], now)

	return {
		...eventCard,
		dateText: formatEventDateRange(mergedEvent.start_date, mergedEvent.end_date),
		timeText: formatEventTimeRange(mergedEvent.start_time, mergedEvent.end_time, now),
		showTime: Boolean(mergedEvent.start_time || mergedEvent.end_time),
		locationText: buildEventLocationText(mergedEvent),
		regionText: buildEventRegionText(mergedEvent),
		coverImage: mergedEvent.cover_image || DEFAULT_EVENT_COVER,
		rounds,
		focusRoundKey: resolveSaiXunFocusRoundKey(rounds),
		champion: resolveSaiXunChampion(rounds)
	}
}

const syncRoundTabIntoView = () => {
	const index = (eventView.value?.rounds || []).findIndex((round) => round.key === activeRound.value?.key)
	roundTabIntoView.value = index >= 0 ? `round-tab-${index}` : ''
}

// 切换轮次时若轮次栏已吸顶，把页面滚回轮次栏原位，避免新内容较短时停在页面底部。
// 锚点是轮次栏前的非吸顶元素，吸顶元素自身的 top 恒为 0，无法判断是否已滚过。
const keepRoundTabsInView = () => {
	const query = uni.createSelectorQuery()
	query.select('#round-tabs-anchor').boundingClientRect()
	query.selectViewport().scrollOffset()
	query.exec((results) => {
		const [anchorRect, viewport] = results || []
		if (!anchorRect || !viewport || anchorRect.top > 0) return
		uni.pageScrollTo({ scrollTop: viewport.scrollTop + anchorRect.top, duration: 0 })
	})
}

const selectRound = (key) => {
	if (key === activeRound.value?.key) return
	keepRoundTabsInView()
	selectedRoundKey.value = key
	syncRoundTabIntoView()
}

const stopEventViewRefreshTimer = () => {
	if (eventViewRefreshTimer) {
		clearInterval(eventViewRefreshTimer)
		eventViewRefreshTimer = null
	}
}

const startEventViewRefreshTimer = () => {
	stopEventViewRefreshTimer()
	if (!rawEventPayload.value) return

	eventViewRefreshTimer = setInterval(() => {
		renderEventView(Date.now())
	}, 30 * 1000)
}

const fetchDetail = async ({ force = false } = {}) => {
	const nextState = beginAsyncPageLoad(eventDetailState.value, { emptyData: null })
	const pageRequest = getAsyncPageRequest(nextState)
	eventDetailState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	if (!eventNewsId.value) {
		eventDetailState.value = rejectAsyncPageLoad(eventDetailState.value, pageRequest, createRequestError({
			message: '未找到该赛讯，请返回赛事列表后重试。',
			category: 'not-found'
		}))
		loading.value = false
		return
	}

	if (!nextState.hasData) {
		rawEventPayload.value = null
		eventView.value = null
		stopEventViewRefreshTimer()
	}
	try {
		const res = await publicReadStore.loadEventNewsView({ event_id: eventNewsId.value }, { force })
		if (eventDetailState.value.requestId !== pageRequest.requestId) return
		if (!res.success) {
			throw createRequestError({ message: res.message || '获取赛事详情失败', category: 'business' })
		}
		const payload = pickEventNewsViewPayload(res)
		if (!payload) {
			throw createRequestError({ message: '赛事数据不存在或已下线', category: 'not-found' })
		}
		rawEventPayload.value = payload
		renderEventView(Date.now())
		startEventViewRefreshTimer()
		eventDetailState.value = resolveAsyncPageLoad(eventDetailState.value, pageRequest, {
			data: payload,
			isEmpty: () => false
		})
		nextTick(syncRoundTabIntoView)
	} catch (e) {
		if (eventDetailState.value.requestId !== pageRequest.requestId) return
		eventDetailState.value = rejectAsyncPageLoad(eventDetailState.value, pageRequest, e)
		if (!eventDetailState.value.hasData) stopEventViewRefreshTimer()
	} finally {
		if (eventDetailState.value.requestId !== pageRequest.requestId) return
		loading.value = false
	}
}

const retryEventDetail = () => fetchDetail({ force: true })

const handleEventDetailErrorAction = () => {
	if (eventDetailState.value.error?.category === 'permission' || eventDetailState.value.error?.category === 'not-found') {
		goBack()
		return
	}
	retryEventDetail()
}

const goBack = () => {
	uni.navigateBack({
		delta: 1,
		fail: () => {
			uni.switchTab({
				url: '/pages/index/index'
			})
		}
	})
}

onLoad((options) => {
	eventNewsId.value = Number.parseInt(options?.id || '0', 10) || 0
	fetchDetail()
})

onShow(() => {
	renderEventView(Date.now())
	startEventViewRefreshTimer()
})

onHide(() => {
	stopEventViewRefreshTimer()
})

onUnload(() => {
	stopEventViewRefreshTimer()
})

onPullDownRefresh(async () => {
	await fetchDetail({ force: true })
	uni.stopPullDownRefresh()
})
</script>

<style lang="scss" scoped>
@import './detail.scss';
</style>
