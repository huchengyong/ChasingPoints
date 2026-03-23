<template>
	<view class="report-page">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#fff"></uni-icons>
		</view>

		<view v-else-if="report" class="report-content">
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

			<!-- 赛季总结卡片 -->
			<view class="report-card summary-card">
				<text class="card-title">赛季总结</text>
				<view class="summary-stats" v-if="report.record">
					<view class="stat-item">
						<text class="stat-value">{{ report.record.matches_played }}</text>
						<text class="stat-label">总场次</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ report.record.wins }}</text>
						<text class="stat-label">胜场</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ winRate }}%</text>
						<text class="stat-label">胜率</text>
					</view>
				</view>
			</view>

			<!-- 段位变化卡片 -->
			<view class="report-card rank-card" v-if="report.record">
				<text class="card-title">段位变化</text>
				<view class="rank-change">
					<view class="rank-from">
						<text class="rank-val">{{ report.record.start_rank_score }}</text>
						<text class="rank-label">赛季初始</text>
					</view>
					<view class="rank-arrow">
						<text>→</text>
					</view>
					<view class="rank-to">
						<text class="rank-val" :class="{ up: report.record.end_rank_score > report.record.start_rank_score }">{{ report.record.end_rank_score }}</text>
						<text class="rank-label">赛季结束</text>
					</view>
				</view>
				<view class="peak-score">
					<text>赛季峰值：{{ report.record.peak_rank_score }} 分</text>
				</view>
				<view class="final-rank" v-if="report.record.final_rank > 0">
					<text>最终排名：第 {{ report.record.final_rank }} 名</text>
				</view>
				<view class="peak-score" v-if="report.rank_trend && report.rank_trend.length > 0">
					<text>趋势记录：{{ report.rank_trend.join(' / ') }}</text>
				</view>
				<view class="peak-score" v-else>
					<text>本赛季暂无可展示的段位趋势</text>
				</view>
			</view>

			<!-- 球种胜率 -->
			<view class="report-card type-card" v-if="report.win_rate_by_type && Object.keys(report.win_rate_by_type).length > 0">
				<text class="card-title">各球种胜场</text>
				<view class="type-list">
					<view v-for="(wins, type) in report.win_rate_by_type" :key="type" class="type-item">
						<text class="type-name">{{ getGameTypeLabel(type, '球种' + type) }}</text>
						<view class="type-bar">
							<view class="type-fill" :style="{ width: Math.min(wins * 10, 100) + '%' }"></view>
						</view>
						<text class="type-wins">{{ wins }}胜</text>
					</view>
				</view>
			</view>

			<view class="report-card achievement-card" v-if="report.top_achievements && report.top_achievements.length > 0">
				<text class="card-title">本赛季解锁成就</text>
				<view class="achievement-list">
					<view
						v-for="achievement in report.top_achievements"
						:key="achievement.id"
						class="achievement-item"
					>
						<view class="achievement-icon">
							<image v-if="achievement.icon" :src="achievement.icon" mode="aspectFill"></image>
							<text v-else>奖</text>
						</view>
						<view class="achievement-meta">
							<text class="achievement-name">{{ achievement.name }}</text>
							<text class="achievement-desc">{{ achievement.description || '本赛季达成的关键里程碑' }}</text>
						</view>
						<text class="achievement-time">{{ formatUnlockTime(achievement.unlocked_at) }}</text>
					</view>
				</view>
			</view>

			<!-- 返回按钮 -->
			<view class="back-btn" @tap="goBack">
				<text>返回</text>
			</view>
		</view>

		<view v-else class="empty-state">
			<text class="empty-text">暂无赛季报告</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getSeasonReport } from '@/api/season.js'
import { GAME_TYPE_TABS, getGameTypeLabel } from '@/utils/game-types.js'

const gameTypeTabs = GAME_TYPE_TABS
const report = ref(null)
const loading = ref(true)
const seasonId = ref(0)
const currentGameType = ref(3)

const winRate = computed(() => {
	if (!report.value || !report.value.record || report.value.record.matches_played === 0) return 0
	return Math.round((report.value.record.wins / report.value.record.matches_played) * 100)
})

const formatUnlockTime = (value) => {
	if (!value) return '已解锁'
	return value.slice(0, 10)
}

const fetchReport = async () => {
	loading.value = true
	try {
		const res = await getSeasonReport({ season_id: seasonId.value, game_type: currentGameType.value })
		if (res.success && res.report) {
			report.value = res.report
		} else {
			report.value = null
		}
	} catch (e) {
		console.error('获取赛季报告失败', e)
	} finally {
		loading.value = false
	}
}

const goBack = () => { uni.navigateBack() }

const handleGameTypeChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	fetchReport()
}

onLoad((options) => {
	seasonId.value = parseInt(options?.id || 0)
	const gameType = parseInt(options?.game_type || 0)
	if (gameType > 0) {
		currentGameType.value = gameType
	}
	if (seasonId.value > 0) fetchReport()
	else loading.value = false
})
</script>

<style lang="scss" scoped>
.report-page {
	min-height: 100vh;
	background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
	padding: 80rpx 24rpx 40rpx;
}
.loading-state {
	display: flex;
	justify-content: center;
	align-items: center;
	min-height: 60vh;
}
.report-content {
	display: flex;
	flex-direction: column;
	gap: 24rpx;
}
.game-type-tabs {
	display: flex;
	gap: 12rpx;
	overflow-x: auto;
}
.game-type-tab {
	flex-shrink: 0;
	min-width: 150rpx;
	padding: 14rpx 24rpx;
	border-radius: 999rpx;
	background: rgba(255,255,255,0.08);
	border: 1rpx solid rgba(255,255,255,0.14);
	box-sizing: border-box;
	text-align: center;
	text {
		font-size: 24rpx;
		font-weight: 600;
		color: #cbd5e1;
	}
	&.active {
		background: linear-gradient(135deg, #18b05b 0%, #15803d 100%);
		border-color: transparent;
		text {
			color: #fff;
		}
	}
}
.report-card {
	background: rgba(255,255,255,0.08);
	border-radius: 20rpx;
	padding: 32rpx;
	backdrop-filter: blur(10px);
	.card-title {
		font-size: 30rpx;
		font-weight: 600;
		color: #94a3b8;
		margin-bottom: 24rpx;
		display: block;
	}
}
.summary-card {
	.summary-stats {
		display: flex;
		justify-content: space-around;
		.stat-item { text-align: center;
			.stat-value { font-size: 48rpx; font-weight: 800; color: #fff; display: block; }
			.stat-label { font-size: 24rpx; color: #94a3b8; }
		}
	}
}
.rank-card {
	.rank-change {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 32rpx;
		margin-bottom: 20rpx;
		.rank-from, .rank-to { text-align: center;
			.rank-val { font-size: 40rpx; font-weight: 700; color: #cbd5e1; display: block; &.up { color: #22c55e; } }
			.rank-label { font-size: 22rpx; color: #64748b; }
		}
		.rank-arrow { font-size: 40rpx; color: #64748b; }
	}
	.peak-score, .final-rank {
		font-size: 26rpx;
		color: #94a3b8;
		text-align: center;
		margin-top: 8rpx;
	}
}
.type-card {
	.type-list {
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}
	.type-item {
		display: flex;
		align-items: center;
		gap: 16rpx;
		.type-name { font-size: 26rpx; color: #cbd5e1; width: 140rpx; }
		.type-bar { flex: 1; height: 16rpx; background: rgba(255,255,255,0.1); border-radius: 8rpx; overflow: hidden;
			.type-fill { height: 100%; background: #18b05b; border-radius: 8rpx; min-width: 10rpx; }
		}
		.type-wins { font-size: 24rpx; color: #94a3b8; width: 80rpx; text-align: right; }
	}
}
.achievement-card {
	.achievement-list {
		display: flex;
		flex-direction: column;
		gap: 18rpx;
	}
	.achievement-item {
		display: flex;
		align-items: center;
		gap: 18rpx;
		padding: 20rpx 0;
		border-top: 1rpx solid rgba(255,255,255,0.08);
		&:first-child {
			border-top: none;
			padding-top: 0;
		}
	}
	.achievement-icon {
		width: 72rpx;
		height: 72rpx;
		border-radius: 18rpx;
		background: rgba(250, 204, 21, 0.16);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		image {
			width: 100%;
			height: 100%;
			border-radius: 18rpx;
		}
		text {
			font-size: 28rpx;
			color: #facc15;
			font-weight: 700;
		}
	}
	.achievement-meta {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 6rpx;
	}
	.achievement-name {
		font-size: 28rpx;
		font-weight: 600;
		color: #fff;
	}
	.achievement-desc {
		font-size: 22rpx;
		color: #94a3b8;
		line-height: 1.5;
	}
	.achievement-time {
		font-size: 22rpx;
		color: #64748b;
		flex-shrink: 0;
	}
}
.back-btn {
	text-align: center;
	padding: 24rpx;
	background: rgba(255,255,255,0.1);
	border-radius: 16rpx;
	color: #fff;
	font-size: 28rpx;
	margin-top: 16rpx;
}
.empty-state {
	display: flex;
	justify-content: center;
	align-items: center;
	min-height: 60vh;
	.empty-text { font-size: 28rpx; color: #64748b; }
}
</style>
