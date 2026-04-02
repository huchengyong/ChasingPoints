<template>
	<view class="detail-page">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="eventView" class="detail-content">
			<view class="hero-card">
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
				<view class="hero-stats">
					<view class="hero-stat">
						<text class="hero-stat-label">赛事时间</text>
						<text class="hero-stat-value">{{ eventView.dateRangeText }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">当前轮次</text>
						<text class="hero-stat-value">{{ eventView.currentRoundText }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">比赛数</text>
						<text class="hero-stat-value">{{ eventView.matchCountText }}</text>
					</view>
				</view>
			</view>

			<view class="info-card">
				<view class="section-head">
					<text class="section-title">赛事信息</text>
					<text class="section-tip">{{ eventView.locationText || '地点待补充' }}</text>
				</view>
				<view class="info-grid">
					<view class="grid-item">
						<text class="grid-label">状态</text>
						<text class="grid-value">{{ eventView.statusText }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">球种</text>
						<text class="grid-value">{{ eventView.gameTypeText }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">地点</text>
						<text class="grid-value">{{ eventView.locationText || '待补充' }}</text>
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
								<text class="story-badge">{{ match.statusText }}</text>
							</view>
							<view class="match-row">
								<text class="player-name" :class="{ winner: match.winnerSide === 1 }">{{ match.homePlayerName }}</text>
								<text class="score-pill">{{ match.scoreText }}</text>
								<text class="player-name right" :class="{ winner: match.winnerSide === 2 }">{{ match.awayPlayerName }}</text>
							</view>
							<text class="story-meta">{{ match.metaText }}</text>
						</view>
					</view>
				</view>
				<view v-else class="story-item">
					<text class="story-label">比赛待更新</text>
					<text class="story-value">当前赛事还没有录入逐场比赛结果。</text>
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
import { formatEventNewsTime, getEventNewsStatusText } from '@/utils/home-index.js'
import { getGameTypeLabel } from '@/utils/game-types.js'

const loading = ref(true)
const errorMessage = ref('')
const eventNewsId = ref(0)
const eventView = ref(null)

const formatLocationText = (eventInfo = {}, tournamentInfo = {}) => {
	const city = typeof eventInfo.city === 'string' && eventInfo.city.trim()
		? eventInfo.city.trim()
		: (typeof tournamentInfo.city === 'string' ? tournamentInfo.city.trim() : '')
	const venue = typeof eventInfo.venue === 'string' && eventInfo.venue.trim()
		? eventInfo.venue.trim()
		: (typeof tournamentInfo.venue_name === 'string' ? tournamentInfo.venue_name.trim() : '')

	if (city && venue) return `${city} · ${venue}`
	return city || venue || ''
}

const formatEventRelativeText = (dateTime, now = Date.now()) => {
	if (!dateTime) return '时间待定'

	const eventTime = new Date(dateTime).getTime()
	if (Number.isNaN(eventTime)) return '时间待定'

	const diff = eventTime - now
	const absSeconds = Math.floor(Math.abs(diff) / 1000)
	const absMinutes = Math.floor(absSeconds / 60)
	const absHours = Math.floor(absMinutes / 60)
	const absDays = Math.floor(absHours / 24)

	const prefix = diff >= 0 ? '还有' : '已过'
	if (absSeconds < 60) return diff >= 0 ? '即将开始' : '刚刚结束'
	if (absMinutes < 60) return `${prefix}${absMinutes}分钟`
	if (absHours < 24) return `${prefix}${absHours}小时`
	if (absDays < 30) return `${prefix}${absDays}天`
	return `${prefix}${Math.max(1, Math.floor(absDays / 30))}个月`
}

const buildMatchMetaText = (match) => {
	const parts = []
	if (match.best_of > 0) {
		parts.push(`Best of ${match.best_of}`)
	}
	if (match.is_placeholder) {
		parts.push('占位赛程')
	}
	return parts.join(' · ') || '等待比赛开始'
}

const normalizeMatch = (match, now = Date.now()) => {
	const status = Number(match.status || 0)
	const scoreReady = status === 1 || status === 2 || Number(match.home_score || 0) > 0 || Number(match.away_score || 0) > 0

	return {
		id: match.id,
		roundName: match.round_name || '轮次待更新',
		roundOrder: Number(match.round_order || 0),
		startTimeText: match.start_time ? formatEventNewsTime(match.start_time, now) : '时间待定',
		statusText: getEventNewsStatusText(status, '待更新'),
		status,
		homePlayerName: match.home_player_name || '待定',
		awayPlayerName: match.away_player_name || '待定',
		scoreText: scoreReady ? `${Number(match.home_score || 0)} : ${Number(match.away_score || 0)}` : '-',
		winnerSide: Number(match.winner_side || 0),
		metaText: buildMatchMetaText(match)
	}
}

const groupMatchesByRound = (matches = [], now = Date.now()) => {
	const roundMap = new Map()

	matches.forEach((item) => {
		const match = normalizeMatch(item, now)
		const key = `${match.roundOrder}-${match.roundName}`
		if (!roundMap.has(key)) {
			roundMap.set(key, {
				key,
				roundName: match.roundName,
				roundOrder: match.roundOrder,
				matches: []
			})
		}
		roundMap.get(key).matches.push(match)
	})

	return Array.from(roundMap.values())
		.sort((left, right) => left.roundOrder - right.roundOrder)
		.map((round) => ({
			...round,
			matches: round.matches.sort((left, right) => left.id - right.id)
		}))
}

const normalizeEventView = ({ eventNews, tournament, matches }, now = Date.now()) => {
	const startTime = eventNews.start_time || tournament?.start_time || eventNews.sort_time || eventNews.created_at
	const endTime = eventNews.end_time || tournament?.end_time || ''
	const rounds = groupMatchesByRound(matches, now)
	const matchCount = Array.isArray(matches) ? matches.length : 0

	return {
		id: eventNews.id,
		title: eventNews.tournament_name || eventNews.title || tournament?.name || '赛事情报',
		summary: eventNews.summary || eventNews.latest_result_text || '逐场比赛结果持续更新中',
		status: Number(eventNews.status || 0),
		statusText: getEventNewsStatusText(eventNews.status, '赛事情报'),
		gameTypeText: getGameTypeLabel(eventNews.game_type, '台球'),
		startTimeText: startTime ? formatEventNewsTime(startTime, now) : '待定',
		endTimeText: endTime ? formatEventNewsTime(endTime, now) : '待定',
		dateRangeText: endTime ? `${formatEventNewsTime(startTime, now)} - ${formatEventNewsTime(endTime, now)}` : formatEventNewsTime(startTime, now),
		relativeTimeText: startTime ? formatEventRelativeText(startTime, now) : '时间待定',
		locationText: formatLocationText(eventNews, tournament || {}),
		sourceName: eventNews.source_name || '手动录入',
		sourceUrl: eventNews.source_url || '',
		updatedAtText: eventNews.updated_at ? formatRelativeTime(eventNews.updated_at) : '刚刚',
		currentRoundText: eventNews.current_round_text || '轮次待更新',
		resultText: eventNews.latest_result_text || '',
		matchCountText: matchCount > 0 ? `${matchCount} 场比赛` : '比赛待更新',
		rounds
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
