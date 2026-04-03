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
				<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
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
					<image class="card-cover" :src="item.coverImage" mode="aspectFill"></image>
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
							<text class="meta-label">日期</text>
							<text class="meta-value">{{ item.dateText }}</text>
						</view>
						<view class="meta-item" v-if="item.showTime">
							<text class="meta-label">时间</text>
							<text class="meta-value">{{ item.timeText }}</text>
						</view>
						<view class="meta-item" v-if="item.locationText">
							<text class="meta-label">地点</text>
							<text class="meta-value">{{ item.locationText }}</text>
						</view>
					<view class="meta-item">
						<text class="meta-label">当前轮次</text>
						<text class="meta-value">{{ item.currentRoundText }}</text>
					</view>
					<view class="meta-item">
						<text class="meta-label">比赛数</text>
						<text class="meta-value">{{ item.matchCountText }}</text>
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
						<uni-icons type="calendar" size="28" color="#E0AE12"></uni-icons>
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
import { normalizeSaiXunCard } from '@/utils/saixun.js'

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

const buildQueryParams = (pageValue, gameTypeValue, statusValue) => {
	const params = {
		page: pageValue,
		page_size: 10
	}
	if (gameTypeValue > 0) params.game_type = gameTypeValue
	if (statusValue >= 0) params.status = statusValue
	return params
}

const fetchList = async ({
	pageValue = page.value,
	gameTypeValue = selectedGameType.value.value,
	statusValue = selectedStatus.value.value,
	replace = false,
	commitSelection = null
} = {}) => {
	if (loading.value) return
	loading.value = true
	try {
		const res = await getEventNewsList(buildQueryParams(pageValue, gameTypeValue, statusValue))
		// 显式检查API返回值
		if (!res.success) {
			console.error('获取赛事情报列表失败:', res.message)
			return false
		}
		if (!Array.isArray(res.list)) {
			console.error('获取赛事情报列表失败: 返回数据格式错误')
			return false
		}
		const newList = (res.list || []).map((item) => normalizeSaiXunCard(item))
		if (commitSelection) {
			if (commitSelection.gameType) selectedGameType.value = commitSelection.gameType
			if (commitSelection.status) selectedStatus.value = commitSelection.status
		}
		if (replace) {
			list.value = newList
		} else {
			list.value = [...list.value, ...newList]
		}
		page.value = pageValue
		totalCount.value = res.total || 0
		hasMore.value = list.value.length < (res.total || 0)
		return true
	} catch (e) {
		console.error('获取赛事情报列表失败', e)
		return false
	} finally {
		loading.value = false
		refreshing.value = false
	}
}

const onRefresh = async () => {
	if (loading.value) return
	refreshing.value = true
	await fetchList({
		pageValue: 1,
		replace: true
	})
}

const loadMore = async () => {
	if (!hasMore.value || loading.value) return
	await fetchList({
		pageValue: page.value + 1,
		replace: false
	})
}

const onGameTypeChange = async (e) => {
	if (loading.value) return
	const nextGameType = gameTypes.value[e.detail.value]
	await fetchList({
		pageValue: 1,
		gameTypeValue: nextGameType.value,
		statusValue: selectedStatus.value.value,
		replace: true,
		commitSelection: {
			gameType: nextGameType,
			status: selectedStatus.value
		}
	})
}

const onStatusChange = async (e) => {
	if (loading.value) return
	const nextStatus = statusList.value[e.detail.value]
	await fetchList({
		pageValue: 1,
		gameTypeValue: selectedGameType.value.value,
		statusValue: nextStatus.value,
		replace: true,
		commitSelection: {
			gameType: selectedGameType.value,
			status: nextStatus
		}
	})
}

const goDetail = (id) => {
	uni.navigateTo({ url: '/subPages/tournament/detail?id=' + id })
}

onMounted(() => fetchList({
	pageValue: 1,
	replace: true
}))
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
