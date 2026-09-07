<template>
	<view class="detail-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

			<view v-else-if="eventView" class="detail-content">
				<view v-if="eventDetailRefreshError" class="refresh-error-banner">
					<text>{{ eventDetailRefreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryEventDetail">重试</text>
				</view>
				<view class="hero-card">
					<image class="hero-cover" :src="eventView.coverImage" mode="aspectFill"></image>
					<view class="hero-overlay"></view>
					<view class="hero-body">
						<view class="hero-content-block">
							<text class="hero-title">{{ eventView.title }}</text>
							<text class="hero-desc">{{ eventView.summary }}</text>
						<view class="hero-meta">
							<view class="hero-meta-item">
								<text class="hero-meta-label">国家 / 地区</text>
								<text class="hero-meta-value">{{ eventView.country || '待补充' }}</text>
							</view>
							<view v-if="eventView.showTime" class="hero-meta-item">
								<text class="hero-meta-label">时间</text>
								<text class="hero-meta-value">{{ eventView.timeText }}</text>
							</view>
							<view class="hero-meta-item">
								<text class="hero-meta-label">地点</text>
								<text class="hero-meta-value">{{ eventView.locationText || '待补充' }}</text>
							</view>
						</view>
					</view>
				</view>
			</view>

			<view class="info-card">
				<view class="section-head">
					<text class="section-title">赛事信息</text>
				</view>
				<view class="info-grid">
					<view class="grid-item">
						<text class="grid-label">日期</text>
						<text class="grid-value">{{ eventView.dateText }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">当前轮次</text>
						<text class="grid-value">{{ eventView.currentRoundText }}</text>
					</view>
				</view>
			</view>

			<view class="story-card">
				<view class="section-head">
					<text class="section-title">赛程与赛果</text>
					<text class="section-tip">{{ eventView.matchCountText }}</text>
				</view>
				<view v-if="eventView.rounds.length" class="story-list">
					<view v-for="round in eventView.rounds" :key="round.key" class="round-group">
						<view class="round-head">
							<text class="round-title">{{ round.roundName }}</text>
							<text class="round-tip">{{ round.matches.length }} 场</text>
						</view>
						<view v-for="match in round.matches" :key="match.id" class="story-item">
							<view class="story-topline">
								<text class="story-label">{{ match.startTimeText }}</text>
								<text class="story-meta-inline">{{ match.metaText }}</text>
								<text class="story-badge" :class="'badge-' + match.status">{{ match.statusText }}</text>
							</view>
							<view class="match-row">
								<view class="player-side" :class="{ winner: match.winnerSide === 1 }">
									<text
										v-if="match.homeResultText"
										class="player-result-tag"
										:class="match.homeResultText === '胜' ? 'is-win' : 'is-lose'"
									>{{ match.homeResultText }}</text>
									<view class="player-portrait-frame" :class="{ winner: match.winnerSide === 1 }">
										<image class="player-avatar" :src="match.homePlayerAvatar" mode="aspectFit"></image>
									</view>
									<view class="player-copy">
										<text v-if="match.homePlayerFlagEmoji" class="player-flag">{{ match.homePlayerFlagEmoji }}</text>
										<text class="player-name-full">{{ match.homePlayerName }}</text>
									</view>
								</view>
								<view class="score-pill" :class="{ 'score-pill--muted': match.status === 0 }">
									<text class="score-pill-text">{{ match.scoreText }}</text>
								</view>
								<view class="player-side player-side-right" :class="{ winner: match.winnerSide === 2 }">
									<text
										v-if="match.awayResultText"
										class="player-result-tag"
										:class="match.awayResultText === '胜' ? 'is-win' : 'is-lose'"
									>{{ match.awayResultText }}</text>
									<view class="player-portrait-frame" :class="{ winner: match.winnerSide === 2 }">
										<image class="player-avatar" :src="match.awayPlayerAvatar" mode="aspectFit"></image>
									</view>
									<view class="player-copy">
										<text v-if="match.awayPlayerFlagEmoji" class="player-flag">{{ match.awayPlayerFlagEmoji }}</text>
										<text class="player-name-full">{{ match.awayPlayerName }}</text>
									</view>
								</view>
							</view>
						</view>
					</view>
				</view>
				<view v-else class="story-empty">
					<text class="story-empty-title">比赛待更新</text>
					<text class="story-empty-text">当前赛事还没有录入逐场赛程与赛果。</text>
				</view>
			</view>

		</view>

		<view v-else-if="eventDetailPageError" class="empty-state error-state">
			<view class="empty-icon">
				<uni-icons type="calendar" size="36" color="#E0AE12"></uni-icons>
			</view>
			<text class="empty-title">{{ eventDetailPageError.title }}</text>
			<text class="empty-text">{{ eventDetailPageError.description }}</text>
			<view class="empty-actions">
				<view class="empty-btn primary" @tap="handleEventDetailErrorAction">
					<text>{{ eventDetailPageError.actionText }}</text>
				</view>
				<view class="empty-btn secondary" @tap="goBack">
					<text>返回上一页</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onHide, onLoad, onShow, onUnload } from '@dcloudio/uni-app'
import { usePublicReadStore } from '@/store/publicRead.js'
import { pickEventNewsViewPayload } from '@/utils/event-news-response.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	DEFAULT_EVENT_COVER,
	buildEventLocationText,
	buildSaiXunDetailRounds,
	formatEventDateRange,
	formatEventTimeRange,
	localizeCountryName,
	normalizeSaiXunCard
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

let eventViewRefreshTimer = null

const renderEventView = (now = Date.now()) => {
	if (!rawEventPayload.value) return
	eventView.value = normalizeEventView(rawEventPayload.value, now)
}

const normalizeEventView = ({ eventNews, tournament, matches }, now = Date.now()) => {
	const mergedEvent = {
		...eventNews,
		tournament_name: eventNews.tournament_name || tournament?.name || eventNews.title || '',
		cover_image: eventNews.cover_image || tournament?.cover_image || DEFAULT_EVENT_COVER,
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
	const rounds = buildSaiXunDetailRounds(matches, now)

	return {
		...eventCard,
		dateText: formatEventDateRange(mergedEvent.start_date, mergedEvent.end_date),
		timeText: formatEventTimeRange(mergedEvent.start_time, mergedEvent.end_time, now),
		showTime: Boolean(mergedEvent.start_time || mergedEvent.end_time),
		locationText: buildEventLocationText(mergedEvent),
		coverImage: mergedEvent.cover_image || DEFAULT_EVENT_COVER,
		country: mergedEvent.country,
		rounds,
		currentRoundText: eventCard.currentRoundText
	}
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

const fetchDetail = async () => {
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
		const res = await publicReadStore.loadEventNewsView({ event_id: eventNewsId.value })
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
	} catch (e) {
		if (eventDetailState.value.requestId !== pageRequest.requestId) return
		eventDetailState.value = rejectAsyncPageLoad(eventDetailState.value, pageRequest, e)
		if (!eventDetailState.value.hasData) stopEventViewRefreshTimer()
	} finally {
		if (eventDetailState.value.requestId !== pageRequest.requestId) return
		loading.value = false
	}
}

const retryEventDetail = () => fetchDetail()

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
</script>

<style lang="scss" scoped>
@import './detail.scss';
</style>
