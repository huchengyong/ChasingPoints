<template>
	<view class="tournament-page">
		<view class="page-hero">
			<view class="hero-copy">
				<text class="hero-eyebrow">赛事情报</text>
				<text class="hero-title">最近的赛程、赛况和赛果集中看清楚</text>
				<text class="hero-desc">按球种和状态筛选，快速锁定正在进行、即将开始和刚更新结果的重点赛事。</text>
			</view>
			<view class="hero-stats">
				<view class="hero-stat">
					<text class="hero-stat-label">已收录</text>
					<text class="hero-stat-value">{{ totalCount }}</text>
				</view>
				<view class="hero-stat">
					<text class="hero-stat-label">进行中</text>
					<text class="hero-stat-value">{{ liveCount }}</text>
				</view>
				<view class="hero-stat">
					<text class="hero-stat-label">即将开始</text>
					<text class="hero-stat-value">{{ upcomingCount }}</text>
				</view>
			</view>
		</view>

		<view class="toolbar">
			<view class="filter-bar">
				<picker :range="gameTypes" range-key="label" @change="onGameTypeChange">
					<view class="filter-item">
						<view class="filter-copy">
							<text class="filter-label">球种</text>
							<text class="filter-value">{{ selectedGameType.label }}</text>
						</view>
						<uni-icons type="bottom" size="14" color="#64748b"></uni-icons>
					</view>
				</picker>
				<picker :range="statusList" range-key="label" @change="onStatusChange">
					<view class="filter-item">
						<view class="filter-copy">
							<text class="filter-label">状态</text>
							<text class="filter-value">{{ selectedStatus.label }}</text>
						</view>
						<uni-icons type="bottom" size="14" color="#64748b"></uni-icons>
					</view>
				</picker>
			</view>
			<view class="toolbar-tip">
				<text>统计基于当前已加载内容，继续下滑可补充更多赛事情报。</text>
			</view>
		</view>

		<view v-if="loading && page === 1 && list.length === 0" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<scroll-view
			v-else
			scroll-y
			class="tournament-scroll"
			@scrolltolower="loadMore"
			refresher-enabled
			:refresher-triggered="refreshing"
			@refresherrefresh="onRefresh"
		>
			<view v-if="list.length > 0" class="tournament-list">
				<view
					v-for="item in list"
					:key="item.id"
					class="tournament-card"
					@tap="goDetail(item.id)"
				>
					<view class="card-topline">
						<view class="card-chip game-chip">
							<text>{{ item.gameTypeText }}</text>
						</view>
						<view class="status-tag" :class="'status-' + item.status">
							<text>{{ item.statusText }}</text>
						</view>
					</view>
					<text class="card-title">{{ item.title }}</text>
					<text class="card-summary">{{ item.summary }}</text>

					<view class="card-meta">
						<view class="meta-item">
							<text class="meta-label">时间</text>
							<text class="meta-value">{{ item.timeText }}</text>
						</view>
						<view class="meta-item" v-if="item.locationText">
							<text class="meta-label">地点</text>
							<text class="meta-value">{{ item.locationText }}</text>
						</view>
						<view class="meta-item">
							<text class="meta-label">当前阶段</text>
							<text class="meta-value">{{ item.stageText }}</text>
						</view>
						<view class="meta-item">
							<text class="meta-label">最新赛果</text>
							<text class="meta-value">{{ item.resultText || item.summary }}</text>
						</view>
					</view>

					<view class="card-footer">
						<text class="footer-source">{{ item.sourceText || '手动录入' }}</text>
						<text class="footer-link">查看详情</text>
					</view>
				</view>
			</view>

			<view v-else class="empty-state">
				<view class="empty-badge">
					<uni-icons type="calendar" size="28" color="#15803d"></uni-icons>
				</view>
				<text class="empty-title">当前筛选下还没有赛事情报</text>
				<text class="empty-text">赛事正在更新中，稍后再来刷新，或者换个球种看看。</text>
			</view>

			<view v-if="list.length > 0 && !hasMore" class="no-more">
				<text>没有更多了</text>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { getEventNewsList } from '@/api/event-news.js'
import { formatEventNewsTime, getEventNewsStatusText } from '@/utils/home-index.js'

const gameTypes = ref([
	{ label: '全部球种', value: 0 },
	{ label: '斯诺克', value: 1 },
	{ label: '中式八球', value: 3 },
	{ label: '中式九球', value: 2 }
])

const statusList = ref([
	{ label: '全部状态', value: -1 },
	{ label: '即将开始', value: 0 },
	{ label: '进行中', value: 1 },
	{ label: '已结束', value: 2 },
	{ label: '已取消', value: 3 }
])

const EVENT_GAME_TYPE_LABELS = {
	1: '斯诺克',
	2: '中式九球',
	3: '中式八球'
}

const selectedGameType = ref(gameTypes.value[0])
const selectedStatus = ref(statusList.value[0])

const list = ref([])
const loading = ref(false)
const refreshing = ref(false)
const page = ref(1)
const hasMore = ref(true)
const totalCount = ref(0)

const liveCount = computed(() => list.value.filter(item => item.status === 1).length)
const upcomingCount = computed(() => list.value.filter(item => item.status === 0).length)

const getGameTypeText = (gameType) => {
	return EVENT_GAME_TYPE_LABELS[Number(gameType)] || '台球'
}

const formatLocationText = (item) => {
	const city = typeof item.city === 'string' ? item.city.trim() : ''
	const venue = typeof item.venue === 'string' ? item.venue.trim() : ''
	if (city && venue) return `${city} · ${venue}`
	return city || venue || ''
}

const normalizeEventNewsItem = (item, now = Date.now()) => {
	const timeSource = item.start_time || item.sort_time || item.end_time || item.published_at || item.created_at
	return {
		...item,
		title: item.title || '赛事情报',
		summary: item.summary || item.result_text || '最新赛况持续更新中',
		statusText: getEventNewsStatusText(item.status, '未知状态'),
		gameTypeText: getGameTypeText(item.game_type),
		timeText: formatEventNewsTime(timeSource, now),
		locationText: formatLocationText(item),
		stageText: item.stage_text || '阶段待更新',
		resultText: item.result_text || '',
		sourceText: item.source_name || ''
	}
}

const fetchList = async (isRefresh = false) => {
	if (loading.value) return
	loading.value = true
	try {
		const params = {
			page: page.value,
			page_size: 10
		}
		if (selectedGameType.value.value > 0) params.game_type = selectedGameType.value.value
		if (selectedStatus.value.value >= 0) params.status = selectedStatus.value.value

		const res = await getEventNewsList(params)
		if (res.success) {
			const newList = (res.list || []).map((item) => normalizeEventNewsItem(item))
			if (isRefresh) {
				list.value = newList
			} else {
				list.value = [...list.value, ...newList]
			}
			totalCount.value = res.total || 0
			hasMore.value = list.value.length < (res.total || 0)
		}
	} catch (e) {
		console.error('获取赛事情报列表失败', e)
	} finally {
		loading.value = false
		refreshing.value = false
	}
}

const onRefresh = () => {
	if (loading.value) return
	refreshing.value = true
	page.value = 1
	fetchList(true)
}

const loadMore = () => {
	if (!hasMore.value || loading.value) return
	page.value++
	fetchList()
}

const onGameTypeChange = (e) => {
	if (loading.value) return
	selectedGameType.value = gameTypes.value[e.detail.value]
	page.value = 1
	list.value = []
	fetchList(true)
}

const onStatusChange = (e) => {
	if (loading.value) return
	selectedStatus.value = statusList.value[e.detail.value]
	page.value = 1
	list.value = []
	fetchList(true)
}

const goDetail = (id) => {
	uni.navigateTo({ url: '/subPages/tournament/detail?id=' + id })
}

onMounted(() => fetchList(true))
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
