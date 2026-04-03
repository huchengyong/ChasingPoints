<template>
	<view class="detail-page">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="eventView" class="detail-content">
			<view class="hero-card">
				<image class="hero-cover" :src="eventView.coverImage" mode="aspectFill"></image>
				<view class="hero-overlay"></view>
				<view class="hero-body">
					<view class="hero-topline">
						<view class="hero-chip">
							<text>{{ eventView.gameTypeText }}</text>
						</view>
						<view class="status-tag" :class="'status-' + eventView.status">
							<text>{{ eventView.statusText }}</text>
						</view>
					</view>
					<text class="hero-title">{{ eventView.title }}</text>
					<text class="hero-desc">{{ eventView.summary }}</text>
					<view class="hero-meta">
						<view class="hero-meta-item">
							<text class="hero-meta-label">日期</text>
							<text class="hero-meta-value">{{ eventView.dateText }}</text>
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

			<view class="info-card">
				<view class="section-head">
					<text class="section-title">赛事信息</text>
					<text class="section-tip">{{ eventView.matchCountText }}</text>
				</view>
				<view class="info-grid">
					<view class="grid-item">
						<text class="grid-label">当前轮次</text>
						<text class="grid-value">{{ eventView.currentRoundText }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">最新赛果</text>
						<text class="grid-value">{{ eventView.resultText || '等待更新' }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">国家 / 地区</text>
						<text class="grid-value">{{ eventView.country || '待补充' }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">更新时间</text>
						<text class="grid-value">{{ eventView.updatedAtText }}</text>
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
									<image class="player-avatar" :src="match.homePlayerAvatar" mode="aspectFill"></image>
									<text class="player-name">{{ match.homePlayerName }}</text>
								</view>
								<view class="score-pill">
									<text>{{ match.scoreText }}</text>
								</view>
								<view class="player-side player-side-right" :class="{ winner: match.winnerSide === 2 }">
									<image class="player-avatar" :src="match.awayPlayerAvatar" mode="aspectFill"></image>
									<text class="player-name">{{ match.awayPlayerName }}</text>
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

			<view class="source-card">
				<view class="section-head">
					<text class="section-title">来源信息</text>
					<text class="section-tip">{{ eventView.sourceUrl ? '可复制外链' : '暂无来源链接' }}</text>
				</view>
				<view class="source-box">
					<text class="source-name">{{ eventView.sourceName }}</text>
					<text class="source-url">{{ eventView.sourceUrl || '暂无来源链接' }}</text>
				</view>
				<button
					class="source-btn"
					:class="{ disabled: !eventView.sourceUrl }"
					:disabled="!eventView.sourceUrl"
					hover-class="none"
					@tap="handleCopySourceLink"
				>
					复制来源链接
				</button>
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
import { onLoad } from '@dcloudio/uni-app'
import { getEventNewsView } from '@/api/event-news.js'
import { pickEventNewsViewPayload } from '@/utils/event-news-response.js'
import { formatRelativeTime } from '@/utils/format.js'
import { getGameTypeLabel } from '@/utils/game-types.js'
import {
	DEFAULT_EVENT_COVER,
	buildEventLocationText,
	buildSaiXunDetailRounds,
	formatEventDateRange,
	formatEventTimeRange,
	normalizeSaiXunCard
} from '@/utils/saixun.js'

const loading = ref(true)
const errorMessage = ref('')
const eventNewsId = ref(0)
const eventView = ref(null)

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
		gameTypeText: getGameTypeLabel(mergedEvent.game_type, '台球'),
		dateText: formatEventDateRange(mergedEvent.start_date, mergedEvent.end_date),
		timeText: formatEventTimeRange(mergedEvent.start_time, mergedEvent.end_time, now),
		showTime: Boolean(mergedEvent.start_time || mergedEvent.end_time),
		locationText: buildEventLocationText(mergedEvent),
		coverImage: mergedEvent.cover_image || DEFAULT_EVENT_COVER,
		sourceName: eventNews.source_name || '追分官方',
		sourceUrl: eventNews.source_url || '',
		country: mergedEvent.country,
		updatedAtText: eventNews.updated_at ? formatRelativeTime(eventNews.updated_at) : '刚刚',
		rounds,
		resultText: eventNews.latest_result_text || eventCard.resultText || '',
		currentRoundText: eventNews.current_round_text || eventCard.currentRoundText
	}
}

const fetchDetail = async () => {
	if (!eventNewsId.value) {
		errorMessage.value = '未找到该赛讯'
		eventView.value = null
		loading.value = false
		return
	}

	loading.value = true
	errorMessage.value = ''
	try {
		const res = await getEventNewsView({ event_id: eventNewsId.value })
		if (!res.success) {
			throw new Error(res.message || '获取赛事详情失败')
		}
		const payload = pickEventNewsViewPayload(res)
		if (!payload) {
			throw new Error('赛事数据不存在')
		}
		eventView.value = normalizeEventView(payload)
	} catch (e) {
		console.error('获取赛事详情失败', e)
		eventView.value = null
		errorMessage.value = e?.responseData?.message || e?.message || '加载失败，请稍后重试'
	} finally {
		loading.value = false
	}
}

const handleCopySourceLink = () => {
	if (!eventView.value?.sourceUrl) {
		uni.showToast({
			title: '暂无来源链接',
			icon: 'none'
		})
		return
	}

	uni.setClipboardData({
		data: eventView.value.sourceUrl,
		success: () => {
			uni.showToast({
				title: '来源链接已复制',
				icon: 'none'
			})
		},
		fail: () => {
			uni.showToast({
				title: '复制失败，请稍后重试',
				icon: 'none'
			})
		}
	})
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
</script>

<style lang="scss" scoped>
@import './detail.scss';
</style>
