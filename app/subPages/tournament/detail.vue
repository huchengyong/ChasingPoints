<template>
	<view class="detail-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

			<view v-else-if="eventView" class="detail-content">
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
					<text class="section-tip">{{ eventView.matchCountText }}</text>
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
					<text class="section-title">比赛结果</text>
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
										<view class="player-copy-top">
											<text v-if="match.homePlayerFlagEmoji" class="player-flag">{{ match.homePlayerFlagEmoji }}</text>
											<text v-if="match.homePlayerFirstName" class="player-first-name">{{ match.homePlayerFirstName }}</text>
										</view>
										<text class="player-last-name">{{ match.homePlayerLastName }}</text>
									</view>
								</view>
								<view class="score-pill">
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
										<view class="player-copy-top">
											<text v-if="match.awayPlayerFlagEmoji" class="player-flag">{{ match.awayPlayerFlagEmoji }}</text>
											<text v-if="match.awayPlayerFirstName" class="player-first-name">{{ match.awayPlayerFirstName }}</text>
										</view>
										<text class="player-last-name">{{ match.awayPlayerLastName }}</text>
									</view>
								</view>
							</view>
							<text class="story-meta">{{ match.metaText }}</text>
						</view>
					</view>
				</view>
				<view v-else class="story-empty">
					<text class="story-empty-title">比赛待更新</text>
					<text class="story-empty-text">当前赛事还没有录入逐场比赛结果。</text>
				</view>
			</view>

		</view>

		<view v-else class="empty-state">
			<view class="empty-icon">
				<uni-icons type="calendar" size="36" color="#E0AE12"></uni-icons>
			</view>
			<text class="empty-title">{{ errorMessage || '未找到该赛讯' }}</text>
			<text class="empty-text">可以返回上一页，或重试加载这场赛事的比赛结果。</text>
			<view class="empty-actions">
				<view class="empty-btn primary" @tap="fetchDetail">
					<text>重试</text>
				</view>
				<view class="empty-btn secondary" @tap="goBack">
					<text>返回上一页</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
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
	normalizeSaiXunCard
} from '@/utils/saixun.js'

const { isDarkMode } = usePageTheme()
const publicReadStore = usePublicReadStore()

const loading = ref(true)
const errorMessage = ref('')
const eventNewsId = ref(0)
const eventView = ref(null)
const rawEventPayload = ref(null)

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
		country: eventNews.country || tournament?.country || '',
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
	if (!eventNewsId.value) {
		errorMessage.value = '未找到该赛讯'
		rawEventPayload.value = null
		eventView.value = null
		loading.value = false
		return
	}

	loading.value = true
	errorMessage.value = ''
	try {
		const res = await publicReadStore.loadEventNewsView({ event_id: eventNewsId.value })
		if (!res.success) {
			throw new Error(res.message || '获取赛事详情失败')
		}
		const payload = pickEventNewsViewPayload(res)
		if (!payload) {
			throw new Error('赛事数据不存在')
		}
		rawEventPayload.value = payload
		renderEventView(Date.now())
		startEventViewRefreshTimer()
	} catch (e) {
		console.error('获取赛事详情失败', e)
		rawEventPayload.value = null
		eventView.value = null
		stopEventViewRefreshTimer()
		errorMessage.value = e?.responseData?.message || e?.message || '加载失败，请稍后重试'
	} finally {
		loading.value = false
	}
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
