<template>
	<view class="stats-page">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载统计数据...</text>
		</view>

		<template v-else>
			<!-- 分球种统计 -->
			<view class="section">
				<text class="section-title">分球种统计</text>
				<view class="game-tabs">
					<view
						v-for="tab in gameTabs"
						:key="tab.key"
						class="tab-item"
						:class="{ active: currentGame === tab.key }"
						@tap="currentGame = tab.key"
					>
						<text>{{ tab.label }}</text>
					</view>
				</view>
				<view v-if="currentGameStats" class="stats-grid">
					<view class="stat-item">
						<text class="stat-value highlight">{{ currentGameStats.win_rate || 0 }}%</text>
						<text class="stat-label">胜率</text>
					</view>
					<view class="stat-item">
						<text class="stat-value">{{ currentGameStats.total_matches || 0 }}</text>
						<text class="stat-label">总场次</text>
					</view>
					<view class="stat-item">
						<text class="stat-value green">{{ currentGameStats.wins || 0 }}</text>
						<text class="stat-label">胜场</text>
					</view>
					<view class="stat-item">
						<text class="stat-value red">{{ currentGameStats.losses || 0 }}</text>
						<text class="stat-label">负场</text>
					</view>
				</view>
				<view class="win-rate-bar">
					<view class="bar-fill" :style="{ width: (currentGameStats?.win_rate || 0) + '%' }"></view>
				</view>
			</view>

			<!-- 近期趋势 -->
			<view class="section">
				<text class="section-title">近期趋势</text>
				<view class="trend-summary">
					<text class="trend-rate">近{{ trendData.length }}场胜率：{{ trendWinRate }}%</text>
				</view>
				<view class="trend-dots">
					<view
						v-for="(match, index) in trendData.slice(-30)"
						:key="index"
						class="dot"
						:class="{ win: match.result === 1, loss: match.result === 2 }"
					></view>
				</view>
				<view class="trend-legend">
					<view class="legend-item"><view class="dot-sample win"></view><text>胜</text></view>
					<view class="legend-item"><view class="dot-sample loss"></view><text>负</text></view>
				</view>
			</view>

			<!-- 段位分变化 -->
			<view class="section">
				<text class="section-title">段位分变化</text>
				<view class="rank-summary">
					<view class="rank-current">
						<text class="rank-value">{{ currentRankScore }}</text>
						<text class="rank-label">当前分</text>
					</view>
					<view class="rank-meta">
						<view class="meta-item">
							<text class="meta-value">{{ peakRankScore }}</text>
							<text class="meta-label">最高分</text>
						</view>
						<view class="meta-item">
							<text class="meta-value">{{ lowestRankScore }}</text>
							<text class="meta-label">最低分</text>
						</view>
					</view>
				</view>
				<view v-if="rankTrendList.length > 0" class="rank-changes">
					<view
						v-for="(change, index) in rankTrendList.slice(0, 10)"
						:key="index"
						class="change-item"
					>
						<text class="change-date">{{ change.date || '' }}</text>
						<text class="change-delta" :class="{ up: change.delta > 0, down: change.delta < 0 }">
							{{ change.delta > 0 ? '+' : '' }}{{ change.delta || 0 }}
						</text>
					</view>
				</view>
				<view v-else class="empty-hint">
					<text>暂无段位变化记录</text>
				</view>
			</view>

			<!-- 单杆最高分 -->
			<view class="section">
				<text class="section-title">单杆最高分</text>
				<view class="high-score-card">
					<view class="score-main">
						<text class="score-number">{{ highScore.personal_best || 0 }}</text>
						<text class="score-label">个人最佳</text>
					</view>
					<view class="score-sub">
						<text class="score-month">本月最佳：{{ highScore.month_best || 0 }}</text>
					</view>
				</view>
			</view>

			<!-- 对局时长 -->
			<view class="section">
				<text class="section-title">对局时长</text>
				<view class="duration-grid">
					<view class="duration-item">
						<text class="duration-value">{{ formatDuration(durationData.average) }}</text>
						<text class="duration-label">平均</text>
					</view>
					<view class="duration-item">
						<text class="duration-value green">{{ formatDuration(durationData.fastest) }}</text>
						<text class="duration-label">最快</text>
					</view>
					<view class="duration-item">
						<text class="duration-value amber">{{ formatDuration(durationData.longest) }}</text>
						<text class="duration-label">最长</text>
					</view>
				</view>
			</view>

			<!-- 强弱对手分析 -->
			<view class="section">
				<text class="section-title">强弱对手分析</text>
				<view class="opponent-list">
					<view
						v-for="(tier, index) in opponentData"
						:key="index"
						class="opponent-item"
					>
						<text class="tier-name">{{ tier.tier_name || ('段位' + (index + 1)) }}</text>
						<view class="tier-bar">
							<view
								class="tier-fill"
								:style="{ width: (tier.win_rate || 0) + '%' }"
								:class="{ good: tier.win_rate >= 50, bad: tier.win_rate < 50 }"
							></view>
						</view>
						<text class="tier-rate" :class="{ good: tier.win_rate >= 50, bad: tier.win_rate < 50 }">{{ tier.win_rate || 0 }}%</text>
					</view>
				</view>
				<view v-if="opponentData.length === 0" class="empty-hint">
					<text>暂无对手分析数据</text>
				</view>
			</view>
		</template>
	</view>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import {
	getStatsByGameType,
	getRecentTrend,
	getRankScoreTrend,
	getSingleHighScore,
	getMatchDurationStats,
	getOpponentStrengthAnalysis
} from '@/api/stats.js'
import { GAME_TYPE_KEY_MAP, GAME_TYPE_STATS_TABS, GAME_TYPE_VALUE_MAP } from '@/utils/game-types.js'
import { formatMonthKey } from '@/utils/format.js'

const loading = ref(true)
const currentGame = ref('chinese_eight')

const gameTypeStats = ref({})
const trendData = ref([])
const rankData = ref([])
const highScore = ref({})
const durationData = ref({})
const opponentData = ref([])

const gameTabs = GAME_TYPE_STATS_TABS
const gameTypeKeyMap = GAME_TYPE_KEY_MAP
const gameTypeValueMap = GAME_TYPE_VALUE_MAP

const currentGameStats = computed(() => {
	return gameTypeStats.value[currentGame.value] || {}
})

const trendWinRate = computed(() => {
	if (!trendData.value || trendData.value.length === 0) return 0
	const wins = trendData.value.filter(m => m.result === 1).length
	return Math.round((wins / trendData.value.length) * 100)
})

const rankTrendList = computed(() => {
	const list = Array.isArray(rankData.value?.list) ? rankData.value.list : Array.isArray(rankData.value) ? rankData.value : []
	return list.map((item, index) => {
		const prev = list[index + 1]
		const prevScore = prev?.rank_score ?? item.rank_score ?? 0
		return {
			date: item.date || '',
			score: item.rank_score || 0,
			delta: (item.rank_score || 0) - prevScore
		}
	})
})

const currentRankScore = computed(() => rankTrendList.value[0]?.score || 0)
const peakRankScore = computed(() => rankTrendList.value.length ? Math.max(...rankTrendList.value.map(item => item.score || 0)) : 0)
const lowestRankScore = computed(() => rankTrendList.value.length ? Math.min(...rankTrendList.value.map(item => item.score || 0)) : 0)

const formatDuration = (seconds) => {
	if (!seconds) return '0:00'
	const mins = Math.floor(seconds / 60)
	const secs = seconds % 60
	return mins + ':' + String(secs).padStart(2, '0')
}

const normalizeGameTypeStats = (payload) => {
	const source = Array.isArray(payload?.list) ? payload.list : Array.isArray(payload) ? payload : []
	const normalized = {}

	source.forEach(item => {
		const key = gameTypeKeyMap[item?.game_type]
		if (!key) return
		normalized[key] = item
	})

	return normalized
}

const normalizeHighScoreStats = (payload) => {
	const list = Array.isArray(payload?.list) ? payload.list : []
	const personalBest = list.length ? (list[0]?.score || 0) : 0
	const currentMonth = formatMonthKey(new Date())
	const monthBest = list.reduce((best, item) => {
		if (!item?.date || !String(item.date).startsWith(currentMonth)) return best
		return Math.max(best, item.score || 0)
	}, 0)
	return {
		personal_best: personalBest,
		month_best: monthBest
	}
}

const loadAllStats = async () => {
	loading.value = true
	const gameType = gameTypeValueMap[currentGame.value] || 3
	try {
		const results = await Promise.allSettled([
			getStatsByGameType(),
			getRecentTrend({ limit: 30, game_type: gameType }),
			getRankScoreTrend({ game_type: gameType }),
			getSingleHighScore({ game_type: gameType }),
			getMatchDurationStats({ game_type: gameType }),
			getOpponentStrengthAnalysis()
		])

		if (results[0].status === 'fulfilled') {
			const data = results[0].value
			gameTypeStats.value = normalizeGameTypeStats(data)
		}
		if (results[1].status === 'fulfilled') {
			const data = results[1].value
			trendData.value = data.list || data.matches || data || []
		}
		if (results[2].status === 'fulfilled') {
			const data = results[2].value
			rankData.value = data?.list ? data : (Array.isArray(data) ? data : [])
		}
		if (results[3].status === 'fulfilled') {
			highScore.value = normalizeHighScoreStats(results[3].value)
		}
		if (results[4].status === 'fulfilled') {
			durationData.value = results[4].value || {}
		}
		if (results[5].status === 'fulfilled') {
			const data = results[5].value
			opponentData.value = data.tiers || data.list || data || []
		}
	} catch (e) {
		console.error('加载统计数据失败:', e)
	} finally {
		loading.value = false
	}
}

onLoad((options) => {
	const gameType = Number(options?.game_type || 0)
	if (gameTypeKeyMap[gameType]) {
		currentGame.value = gameTypeKeyMap[gameType]
	}
	loadAllStats()
})

watch(currentGame, () => {
	loadAllStats()
})
</script>

<style lang="scss" scoped>
.stats-page {
	min-height: 100vh;
	background: #f1f5f9;
	padding-bottom: 60rpx;
	padding-top: 24rpx;
	box-sizing: border-box;
}

.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 300rpx;
	.loading-text {
		margin-top: 16rpx;
		font-size: 28rpx;
		color: #94a3b8;
	}
}

.section {
	margin: 0 24rpx 24rpx;
	background: #fff;
	border-radius: 16rpx;
	padding: 28rpx;

	.section-title {
		font-size: 30rpx;
		font-weight: 700;
		color: #1e293b;
		margin-bottom: 20rpx;
		display: block;
	}
}

.game-tabs {
	display: flex;
	gap: 12rpx;
	margin-bottom: 20rpx;

	.tab-item {
		flex: 1;
		text-align: center;
		padding: 12rpx 0;
		border-radius: 10rpx;
		background: #f1f5f9;
		font-size: 26rpx;
		color: #64748b;

		&.active {
			background: #18b05b;
			color: #fff;
		}
	}
}

.stats-grid {
	display: flex;
	flex-wrap: wrap;

	.stat-item {
		width: 50%;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 16rpx 0;

		.stat-value {
			font-size: 40rpx;
			font-weight: 700;
			color: #1e293b;

			&.highlight { color: #18b05b; }
			&.green { color: #22c55e; }
			&.red { color: #ef4444; }
		}
		.stat-label {
			font-size: 24rpx;
			color: #94a3b8;
			margin-top: 4rpx;
		}
	}
}

.win-rate-bar {
	height: 16rpx;
	background: #fee2e2;
	border-radius: 8rpx;
	overflow: hidden;
	margin-top: 12rpx;

	.bar-fill {
		height: 100%;
		background: #22c55e;
		border-radius: 8rpx;
	}
}

.trend-summary {
	margin-bottom: 16rpx;
	.trend-rate {
		font-size: 26rpx;
		color: #64748b;
	}
}

.trend-dots {
	display: flex;
	flex-wrap: wrap;
	gap: 8rpx;

	.dot {
		width: 20rpx;
		height: 20rpx;
		border-radius: 4rpx;
		background: #e2e8f0;

		&.win { background: #22c55e; }
		&.loss { background: #ef4444; }
	}
}

.trend-legend {
	display: flex;
	gap: 24rpx;
	margin-top: 16rpx;

	.legend-item {
		display: flex;
		align-items: center;
		gap: 8rpx;
		font-size: 22rpx;
		color: #94a3b8;

		.dot-sample {
			width: 16rpx;
			height: 16rpx;
			border-radius: 4rpx;
			&.win { background: #22c55e; }
			&.loss { background: #ef4444; }
		}
	}
}

.rank-summary {
	display: flex;
	align-items: center;
	gap: 32rpx;
	margin-bottom: 20rpx;

	.rank-current {
		display: flex;
		flex-direction: column;
		align-items: center;

		.rank-value {
			font-size: 56rpx;
			font-weight: 800;
			color: #18b05b;
		}
		.rank-label {
			font-size: 24rpx;
			color: #94a3b8;
		}
	}

	.rank-meta {
		flex: 1;
		display: flex;
		gap: 32rpx;

		.meta-item {
			display: flex;
			flex-direction: column;
			align-items: center;

			.meta-value {
				font-size: 32rpx;
				font-weight: 600;
				color: #1e293b;
			}
			.meta-label {
				font-size: 22rpx;
				color: #94a3b8;
			}
		}
	}
}

.rank-changes {
	.change-item {
		display: flex;
		justify-content: space-between;
		padding: 12rpx 0;
		border-bottom: 1rpx solid #f1f5f9;

		&:last-child { border-bottom: none; }

		.change-date {
			font-size: 24rpx;
			color: #94a3b8;
		}
		.change-delta {
			font-size: 26rpx;
			font-weight: 600;
			&.up { color: #22c55e; }
			&.down { color: #ef4444; }
		}
	}
}

.high-score-card {
	display: flex;
	align-items: center;
	gap: 32rpx;

	.score-main {
		display: flex;
		flex-direction: column;
		align-items: center;

		.score-number {
			font-size: 64rpx;
			font-weight: 800;
			color: #f59e0b;
		}
		.score-label {
			font-size: 24rpx;
			color: #94a3b8;
		}
	}
	.score-sub {
		.score-month {
			font-size: 26rpx;
			color: #64748b;
		}
	}
}

.duration-grid {
	display: flex;

	.duration-item {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;

		.duration-value {
			font-size: 36rpx;
			font-weight: 700;
			color: #1e293b;

			&.green { color: #22c55e; }
			&.amber { color: #f59e0b; }
		}
		.duration-label {
			font-size: 24rpx;
			color: #94a3b8;
			margin-top: 4rpx;
		}
	}
}

.opponent-list {
	.opponent-item {
		display: flex;
		align-items: center;
		gap: 16rpx;
		margin-bottom: 16rpx;

		.tier-name {
			width: 120rpx;
			font-size: 24rpx;
			color: #64748b;
			flex-shrink: 0;
		}

		.tier-bar {
			flex: 1;
			height: 20rpx;
			background: #f1f5f9;
			border-radius: 10rpx;
			overflow: hidden;

			.tier-fill {
				height: 100%;
				border-radius: 10rpx;
				&.good { background: #22c55e; }
				&.bad { background: #ef4444; }
			}
		}

		.tier-rate {
			width: 80rpx;
			text-align: right;
			font-size: 24rpx;
			font-weight: 600;
			flex-shrink: 0;
			&.good { color: #22c55e; }
			&.bad { color: #ef4444; }
		}
	}
}

.empty-hint {
	text-align: center;
	padding: 32rpx 0;
	font-size: 26rpx;
	color: #94a3b8;
}
</style>
