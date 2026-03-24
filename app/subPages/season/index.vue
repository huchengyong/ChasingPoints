<template>
	<view class="season-page">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
				<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else class="season-content">
			<!-- 赛季信息卡片 -->
			<view class="season-card" v-if="season">
				<view class="season-header">
					<text class="season-name">{{ season.name }}</text>
					<view class="season-status" :class="'status-' + season.status">
						<text>{{ statusMap[season.status] || '未知' }}</text>
					</view>
				</view>
				<view class="season-dates">
					<text>{{ season.start_date }} ~ {{ season.end_date }}</text>
				</view>
				<view class="season-progress" v-if="season.status === 1">
					<view class="progress-bar">
						<view class="progress-fill" :style="{ width: progressPercent + '%' }"></view>
					</view>
					<text class="progress-text">剩余 {{ remainDays }} 天</text>
				</view>
			</view>

			<view v-else class="no-season-card">
				<text class="no-season-icon">🏆</text>
				<text class="no-season-text">暂无进行中的赛季</text>
			</view>

			<view class="game-type-tabs">
				<view
					v-for="item in gameTypeTabs"
					:key="item.value"
					class="game-type-tab"
					:class="{ active: currentGameType === item.value }"
					@tap="handleGameTypeChange(item.value)"
				>
					<text>{{ item.label }}</text>
				</view>
			</view>

			<!-- 我的赛季数据 -->
			<view class="my-record-card" v-if="myRecord">
				<text class="card-title">我的赛季数据</text>
				<view class="record-stats">
					<view class="stat-item">
						<text class="stat-value">{{ myRecord.matches_played }}</text>
						<text class="stat-label">场次</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ myRecord.wins }}</text>
						<text class="stat-label">胜场</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ winRate }}%</text>
						<text class="stat-label">胜率</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ myRecord.peak_rank_score }}</text>
						<text class="stat-label">峰值分</text>
					</view>
				</view>
				<view class="record-rank" v-if="myRecord.final_rank > 0">
					<text>当前排名：第 {{ myRecord.final_rank }} 名</text>
				</view>
			</view>

			<!-- 赛季排行榜 -->
			<view class="leaderboard-card">
				<view class="card-header">
					<text class="card-title">赛季排行榜</text>
					<view class="report-btn" v-if="season && season.status === 2" @tap="goReport">
						<text>赛季报告</text>
					</view>
				</view>
				<view v-if="leaderboard.length > 0" class="leaderboard-list">
					<view
						v-for="(item, idx) in leaderboard"
						:key="item.user_id"
						class="rank-item"
					>
						<view class="rank-num" :class="{ 'top-1': idx === 0, 'top-2': idx === 1, 'top-3': idx === 2 }">
							<text>{{ idx + 1 }}</text>
						</view>
						<image
							class="rank-avatar"
							:src="item.avatar || '/static/default-avatar.png'"
							mode="aspectFill"
						></image>
						<view class="rank-info">
							<text class="rank-name">{{ item.nickname || '球友' }}</text>
							<text class="rank-score">{{ item.rank_score }} 分</text>
						</view>
						<view class="rank-stats">
							<text class="rank-wins">{{ item.wins }}胜</text>
						</view>
					</view>
				</view>
				<view v-else class="empty-list">
					<text>暂无排行数据</text>
				</view>

				<!-- 加载更多 -->
				<view v-if="leaderboard.length > 0 && hasMore" class="load-more" @tap="loadMoreLeaderboard">
					<text>加载更多</text>
				</view>
				<view v-if="leaderboard.length > 0 && !hasMore" class="no-more">
					<text>没有更多了</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getCurrentSeason, getSeasonLeaderboard, getMySeasonRecord } from '@/api/season.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'

const statusMap = { 0: '未开始', 1: '进行中', 2: '已结束' }
const gameTypeTabs = GAME_TYPE_TABS

const season = ref(null)
const myRecord = ref(null)
const leaderboard = ref([])
const loading = ref(true)
const page = ref(1)
const hasMore = ref(true)
const currentGameType = ref(3)

const winRate = computed(() => {
	if (!myRecord.value || myRecord.value.matches_played === 0) return 0
	return Math.round((myRecord.value.wins / myRecord.value.matches_played) * 100)
})

const remainDays = computed(() => {
	if (!season.value || !season.value.end_date) return 0
	const end = new Date(season.value.end_date)
	const now = new Date()
	const diff = Math.ceil((end - now) / (1000 * 60 * 60 * 24))
	return diff > 0 ? diff : 0
})

const progressPercent = computed(() => {
	if (!season.value || !season.value.start_date || !season.value.end_date) return 0
	const start = new Date(season.value.start_date).getTime()
	const end = new Date(season.value.end_date).getTime()
	const now = Date.now()
	if (end <= start) return 100
	const percent = ((now - start) / (end - start)) * 100
	return Math.min(Math.max(percent, 0), 100)
})

const fetchSeason = async () => {
	try {
		const res = await getCurrentSeason()
		if (res.success && res.season) {
			season.value = res.season
		}
	} catch (e) {
		console.error('获取赛季失败', e)
	}
}

const fetchMyRecord = async () => {
	if (!season.value) return
	try {
		const res = await getMySeasonRecord({ season_id: season.value.id, game_type: currentGameType.value })
		if (res.success && res.record) {
			myRecord.value = res.record
		} else {
			myRecord.value = null
		}
	} catch (e) {
		console.error('获取赛季记录失败', e)
	}
}

const fetchLeaderboard = async (isRefresh = false) => {
	if (!season.value) return
	try {
		const res = await getSeasonLeaderboard({
			season_id: season.value.id,
			game_type: currentGameType.value,
			page: page.value,
			page_size: 20
		})
		if (res.success) {
			const newList = res.list || []
			if (isRefresh) {
				leaderboard.value = newList
			} else {
				leaderboard.value = [...leaderboard.value, ...newList]
			}
			hasMore.value = leaderboard.value.length < (res.total || 0)
		}
	} catch (e) {
		console.error('获取排行榜失败', e)
	}
}

const loadMoreLeaderboard = () => {
	if (!hasMore.value) return
	page.value++
	fetchLeaderboard()
}

const goReport = () => {
	if (season.value) {
		uni.navigateTo({ url: `/subPages/season/report?id=${season.value.id}&game_type=${currentGameType.value}` })
	}
}

const handleGameTypeChange = async (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	page.value = 1
	hasMore.value = true
	leaderboard.value = []
	await Promise.all([fetchMyRecord(), fetchLeaderboard(true)])
}

onMounted(async () => {
	loading.value = true
	await fetchSeason()
	if (season.value) {
		await Promise.all([fetchMyRecord(), fetchLeaderboard(true)])
	}
	loading.value = false
})
</script>

<style lang="scss" scoped>
.season-page {
	min-height: 100vh;
	background: #f1f5f9;
}
.loading-state {
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
	min-height: 60vh;
	.loading-text { font-size: 28rpx; color: #94a3b8; margin-top: 16rpx; }
}
.season-content {
	padding: 20rpx 24rpx;
}
.game-type-tabs {
	display: flex;
	gap: 12rpx;
	margin-bottom: 20rpx;
	overflow-x: auto;
}
.game-type-tab {
	flex-shrink: 0;
	min-width: 150rpx;
	padding: 14rpx 24rpx;
	border-radius: 999rpx;
	background: rgba(255, 255, 255, 0.86);
	border: 1rpx solid rgba(148, 163, 184, 0.24);
	box-sizing: border-box;
	text-align: center;
	text {
		font-size: 24rpx;
		font-weight: 600;
		color: #475569;
	}
	&.active {
		background: #fff7dc;
		border: 1rpx solid rgba(224, 174, 18, 0.22);
		text {
			color: #7c5b05;
		}
	}
	}
	.season-card {
		background: linear-gradient(135deg, #5f4306 0%, #b68108 100%);
	border-radius: 20rpx;
	padding: 32rpx;
	margin-bottom: 20rpx;
	.season-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12rpx;
		.season-name { font-size: 36rpx; font-weight: 700; color: #fff; }
		.season-status {
			padding: 6rpx 16rpx;
			border-radius: 8rpx;
			font-size: 22rpx;
			background: rgba(255,255,255,0.2);
			color: #fff;
			&.status-1 { background: rgba(34,197,94,0.3); }
			&.status-2 { background: rgba(148,163,184,0.3); }
		}
	}
	.season-dates { font-size: 24rpx; color: rgba(255,255,255,0.7); margin-bottom: 16rpx; }
	.season-progress {
		.progress-bar {
			height: 12rpx;
			background: rgba(255,255,255,0.2);
			border-radius: 6rpx;
			overflow: hidden;
			margin-bottom: 8rpx;
			.progress-fill { height: 100%; background: #22c55e; border-radius: 6rpx; min-width: 4rpx; }
		}
		.progress-text { font-size: 22rpx; color: rgba(255,255,255,0.8); }
	}
}
.no-season-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 60rpx 32rpx;
	margin-bottom: 20rpx;
	display: flex;
	flex-direction: column;
	align-items: center;
	.no-season-icon { font-size: 64rpx; margin-bottom: 16rpx; }
	.no-season-text { font-size: 28rpx; color: #94a3b8; }
}
.my-record-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 32rpx;
	margin-bottom: 20rpx;
	.record-stats {
		display: flex;
		justify-content: space-around;
		margin-top: 20rpx;
		.stat-item {
			text-align: center;
			.stat-value { font-size: 40rpx; font-weight: 700; color: #1e293b; display: block; }
			.stat-label { font-size: 22rpx; color: #94a3b8; }
		}
	}
		.record-rank {
			margin-top: 20rpx;
			text-align: center;
			font-size: 26rpx;
			color: #c69200;
			font-weight: 500;
		}
}
.leaderboard-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 32rpx;
	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 20rpx;
			.report-btn {
				padding: 8rpx 20rpx;
				background: #e0ae12;
				border-radius: 8rpx;
				font-size: 24rpx;
				color: #231c0b;
			}
		}
}
.card-title {
	font-size: 30rpx;
	font-weight: 600;
	color: #1e293b;
}
.leaderboard-list {
	display: flex;
	flex-direction: column;
}
.rank-item {
	display: flex;
	align-items: center;
	padding: 16rpx 0;
	border-bottom: 1rpx solid #f1f5f9;
	&:last-child { border-bottom: none; }
	.rank-num {
		width: 48rpx;
		height: 48rpx;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 24rpx;
		font-weight: 600;
		color: #94a3b8;
		background: #f1f5f9;
		margin-right: 16rpx;
		&.top-1 { background: #fef3c7; color: #d97706; }
		&.top-2 { background: #f1f5f9; color: #475569; }
		&.top-3 { background: #fed7aa; color: #c2410c; }
	}
	.rank-avatar {
		width: 72rpx;
		height: 72rpx;
		border-radius: 50%;
		margin-right: 16rpx;
		background: #e2e8f0;
	}
	.rank-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		.rank-name { font-size: 28rpx; color: #1e293b; font-weight: 500; }
		.rank-score { font-size: 22rpx; color: #94a3b8; margin-top: 4rpx; }
	}
		.rank-stats {
			.rank-wins { font-size: 24rpx; color: #c69200; font-weight: 500; }
		}
	}
.empty-list {
	text-align: center;
	padding: 40rpx;
	font-size: 28rpx;
	color: #94a3b8;
}
	.load-more {
		text-align: center;
		padding: 24rpx;
		font-size: 26rpx;
		color: #c69200;
	}
.no-more {
	text-align: center;
	padding: 24rpx;
	font-size: 24rpx;
	color: #cbd5e1;
}
</style>
