<template>
	<view class="tournament-page">
		<view class="page-hero">
			<view class="hero-copy">
				<text class="hero-eyebrow">赛事大厅</text>
				<text class="hero-title">把正在报名和即将开打的赛事集中看清楚</text>
				<text class="hero-desc">筛选球种和状态，快速找到适合报名或围观的赛事。</text>
			</view>
			<view class="hero-stats">
				<view class="hero-stat">
					<text class="hero-stat-label">赛事总数</text>
					<text class="hero-stat-value">{{ totalCount }}</text>
				</view>
				<view class="hero-stat">
					<text class="hero-stat-label">报名中</text>
					<text class="hero-stat-value">{{ openCount }}</text>
				</view>
				<view class="hero-stat">
					<text class="hero-stat-label">进行中</text>
					<text class="hero-stat-value">{{ runningCount }}</text>
				</view>
			</view>
		</view>

		<view class="toolbar">
			<view class="filter-bar" style="flex: 1;">
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
			<!-- 赛事创建已关闭，仅管理员可在后台发布 -->
		</view>

		<view v-if="loading && page === 1" class="loading-state">
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
							<text>{{ getGameTypeLabel(item.game_type, '未知球种') }}</text>
						</view>
						<view class="status-tag" :class="'status-' + item.status">
							<text>{{ statusMap[item.status] || '未知状态' }}</text>
						</view>
					</view>
					<view class="card-header">
						<text class="card-name">{{ item.name }}</text>
						<text class="card-subtitle">{{ formatMap[item.format] || '未知赛制' }} · {{ item.max_players }} 人上限</text>
					</view>
					<view class="card-progress">
						<view class="progress-head">
							<text class="progress-label">报名进度</text>
							<text class="progress-value">{{ item.current_players || 0 }}/{{ item.max_players || 0 }}</text>
						</view>
						<view class="progress-track">
							<view
								class="progress-fill"
								:style="{ width: `${getProgressPercent(item)}%` }"
							></view>
						</view>
					</view>
					<view class="card-info">
						<view class="info-row">
							<uni-icons type="calendar" size="16" color="#64748b"></uni-icons>
							<view class="info-copy">
								<text class="info-label">开始时间</text>
								<text class="info-value">{{ formatDisplayTime(item.start_time) || '待定' }}</text>
							</view>
						</view>
						<view class="info-row" v-if="item.city">
							<uni-icons type="location" size="16" color="#64748b"></uni-icons>
							<view class="info-copy">
								<text class="info-label">举办城市</text>
								<text class="info-value">{{ item.city }}</text>
							</view>
						</view>
					</view>
					<view class="card-footer">
						<text class="footer-time">{{ formatRelativeTime(item.start_time) || '时间待定' }}</text>
						<text class="footer-link">查看详情</text>
					</view>
				</view>
			</view>

			<view v-else class="empty-state">
				<view class="empty-badge">
					<uni-icons type="calendar" size="28" color="#15803d"></uni-icons>
				</view>
				<text class="empty-title">当前筛选下还没有赛事</text>
				<text class="empty-text">赛事正在筹备中，请稍后再来查看。</text>
			</view>

			<view v-if="list.length > 0 && !hasMore" class="no-more">
				<text>没有更多了</text>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { getTournamentList } from '@/api/tournament.js'
import { formatRelativeTime } from '@/utils/format.js'
import { GAME_TYPE_FILTER_OPTIONS_WITH_ALL, getGameTypeLabel } from '@/utils/game-types.js'

const gameTypes = ref(GAME_TYPE_FILTER_OPTIONS_WITH_ALL)
const statusList = ref([
	{ label: '全部状态', value: -1 },
	{ label: '报名中', value: 0 },
	{ label: '进行中', value: 1 },
	{ label: '已结束', value: 2 }
])
const selectedGameType = ref(gameTypes.value[0])
const selectedStatus = ref(statusList.value[0])
const formatMap = { 1: '单败淘汰', 2: '双败淘汰', 3: '循环赛' }
const statusMap = { 0: '报名中', 1: '进行中', 2: '已结束', 3: '已取消' }

const list = ref([])
const loading = ref(false)
const refreshing = ref(false)
const page = ref(1)
const hasMore = ref(true)
const totalCount = ref(0)
const openCount = computed(() => list.value.filter(item => item.status === 0).length)
const runningCount = computed(() => list.value.filter(item => item.status === 1).length)

const fetchList = async (isRefresh = false) => {
	if (loading.value) return
	loading.value = true
	try {
		const params = { page: page.value, page_size: 10 }
		if (selectedGameType.value.value > 0) params.game_type = selectedGameType.value.value
		if (selectedStatus.value.value >= 0) params.status = selectedStatus.value.value
		const res = await getTournamentList(params)
		if (res.success) {
			const newList = res.list || []
			if (isRefresh) {
				list.value = newList
			} else {
				list.value = [...list.value, ...newList]
			}
			totalCount.value = res.total || 0
			hasMore.value = list.value.length < (res.total || 0)
		}
	} catch (e) {
		console.error('获取赛事列表失败', e)
	} finally {
		loading.value = false
		refreshing.value = false
	}
}

const onRefresh = () => {
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
	selectedGameType.value = gameTypes.value[e.detail.value]
	page.value = 1
	list.value = []
	fetchList(true)
}
const onStatusChange = (e) => {
	selectedStatus.value = statusList.value[e.detail.value]
	page.value = 1
	list.value = []
	fetchList(true)
}
const getProgressPercent = (item) => {
	const maxPlayers = Number(item.max_players) || 0
	if (!maxPlayers) return 0
	const currentPlayers = Number(item.current_players) || 0
	return Math.min(100, Math.round((currentPlayers / maxPlayers) * 100))
}
const formatDisplayTime = (value) => {
	if (!value) return ''
	const relative = formatRelativeTime(value)
	return relative === '刚刚' ? '今天' : relative
}
const goCreate = () => {
	uni.navigateTo({ url: '/subPages/tournament/create' })
}
const goDetail = (id) => {
	uni.navigateTo({ url: '/subPages/tournament/detail?id=' + id })
}

onMounted(() => fetchList(true))
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
