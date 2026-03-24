<template>
	<view class="detail-page">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="eventNews" class="detail-content">
			<view class="hero-card">
				<view class="hero-topline">
					<view class="hero-chip">
						<text>{{ eventNews.gameTypeText }}</text>
					</view>
					<view class="status-tag" :class="'status-' + eventNews.status">
						<text>{{ eventNews.statusText }}</text>
					</view>
				</view>
				<text class="hero-title">{{ eventNews.title }}</text>
				<text class="hero-desc">{{ eventNews.summary }}</text>
				<view class="hero-stats">
					<view class="hero-stat">
						<text class="hero-stat-label">开始时间</text>
						<text class="hero-stat-value">{{ eventNews.startTimeText }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">结束时间</text>
						<text class="hero-stat-value">{{ eventNews.endTimeText }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">相对时间</text>
						<text class="hero-stat-value">{{ eventNews.relativeTimeText }}</text>
					</view>
				</view>
			</view>

			<view class="info-card">
				<view class="section-head">
					<text class="section-title">赛事信息</text>
					<text class="section-tip">{{ eventNews.locationText || '地点待补充' }}</text>
				</view>
				<view class="info-grid">
					<view class="grid-item">
						<text class="grid-label">状态</text>
						<text class="grid-value">{{ eventNews.statusText }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">球种</text>
						<text class="grid-value">{{ eventNews.gameTypeText }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">地点</text>
						<text class="grid-value">{{ eventNews.locationText || '待补充' }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">更新时间</text>
						<text class="grid-value">{{ eventNews.updatedAtText }}</text>
					</view>
				</view>
			</view>

			<view class="story-card">
				<view class="section-head">
					<text class="section-title">赛况摘要</text>
					<text class="section-tip">{{ eventNews.sourceName || '信息来源' }}</text>
				</view>
				<view class="story-list">
					<view class="story-item">
						<text class="story-label">当前阶段</text>
						<text class="story-value">{{ eventNews.stageText }}</text>
					</view>
					<view class="story-item">
						<text class="story-label">最新赛果</text>
						<text class="story-value">{{ eventNews.resultText }}</text>
					</view>
					<view class="story-item">
						<text class="story-label">赛事地点</text>
						<text class="story-value">{{ eventNews.locationText || '待补充' }}</text>
					</view>
				</view>
			</view>

			<view class="source-card">
				<view class="section-head">
					<text class="section-title">来源信息</text>
					<text class="section-tip">{{ eventNews.sourceUrl ? '可复制外链' : '暂无来源链接' }}</text>
				</view>
				<view class="source-box">
					<text class="source-name">{{ eventNews.sourceName }}</text>
					<text class="source-url">{{ eventNews.sourceUrl || '暂无来源链接' }}</text>
				</view>
				<button
					class="source-btn"
					:class="{ disabled: !eventNews.sourceUrl }"
					:disabled="!eventNews.sourceUrl"
					hover-class="none"
					@tap="handleCopySourceLink"
				>
					复制来源链接
				</button>
			</view>
		</view>

		<view v-else class="empty-state">
			<view class="empty-icon">
				<uni-icons type="calendar" size="36" color="#15803d"></uni-icons>
			</view>
			<text class="empty-title">{{ errorMessage || '未找到该赛事情报' }}</text>
			<text class="empty-text">可以返回上一页，或重试加载这条赛事情报。</text>
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
import { getEventNewsDetail } from '@/api/event-news.js'
import { formatRelativeTime } from '@/utils/format.js'
import { formatEventNewsTime, getEventNewsStatusText } from '@/utils/home-index.js'
import { getGameTypeLabel } from '@/utils/game-types.js'

const loading = ref(true)
const errorMessage = ref('')
const eventNewsId = ref(0)
const eventNews = ref(null)

const formatLocationText = (item) => {
	const city = typeof item.city === 'string' ? item.city.trim() : ''
	const venue = typeof item.venue === 'string' ? item.venue.trim() : ''
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

const normalizeEventNews = (item, now = Date.now()) => {
	const startTime = item.start_time || item.sort_time || item.created_at
	const endTime = item.end_time || ''

	return {
		id: item.id,
		title: item.title || '赛事情报',
		summary: item.summary || item.result_text || '赛程赛况持续更新中',
		status: item.status,
		statusText: getEventNewsStatusText(item.status, '赛事情报'),
		gameTypeText: getGameTypeLabel(item.game_type, '台球'),
		startTimeText: startTime ? formatEventNewsTime(startTime, now) : '待定',
		endTimeText: endTime ? formatEventNewsTime(endTime, now) : '待定',
		relativeTimeText: item.start_time ? formatEventRelativeText(item.start_time, now) : '时间待定',
		locationText: formatLocationText(item),
		stageText: item.stage_text || '阶段待更新',
		resultText: item.result_text || '赛果待更新',
		sourceName: item.source_name || '手动录入',
		sourceUrl: item.source_url || '',
		updatedAtText: item.updated_at ? formatRelativeTime(item.updated_at) : '刚刚'
	}
}

const fetchDetail = async () => {
	if (!eventNewsId.value) {
		errorMessage.value = '未找到该赛事情报'
		eventNews.value = null
		loading.value = false
		return
	}

	loading.value = true
	errorMessage.value = ''
	try {
		const res = await getEventNewsDetail({ event_news_id: eventNewsId.value })
		eventNews.value = normalizeEventNews(res.event_news || res.eventNews)
		return
	} catch (e) {
		console.error('获取赛事情报详情失败', e)
		eventNews.value = null
		errorMessage.value = e?.responseData?.message || e?.message || '加载失败，请稍后重试'
	} finally {
		loading.value = false
	}
}

const handleCopySourceLink = () => {
	if (!eventNews.value?.sourceUrl) {
		uni.showToast({
			title: '暂无来源链接',
			icon: 'none'
		})
		return
	}

	uni.setClipboardData({
		data: eventNews.value.sourceUrl,
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
