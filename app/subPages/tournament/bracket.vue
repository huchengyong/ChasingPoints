<template>
	<view class="bracket-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载对阵图...</text>
		</view>

		<view v-else-if="matches.length === 0" class="empty-state">
			<text class="empty-icon">🏆</text>
			<text class="empty-text">暂无对阵信息</text>
		</view>

		<scroll-view v-else scroll-x scroll-y class="bracket-scroll">
			<view class="bracket-container">
				<view v-for="round in totalRounds" :key="round" class="round-column">
					<view class="round-title">
						<text>{{ getRoundName(round) }}</text>
					</view>
					<view class="round-matches">
						<view
							v-for="match in getMatchesByRound(round)"
							:key="match.id"
							class="match-card"
							:class="{ completed: match.status === 2 }"
						>
							<view class="player-row" :class="{ winner: match.winner_id === match.player1_id && match.winner_id > 0 }">
								<text class="player-name">{{ match.player1_name || (match.player1_id > 0 ? '选手' : '轮空') }}</text>
								<text class="player-score" v-if="match.status === 2 && match.winner_id === match.player1_id">✓</text>
							</view>
							<view class="vs-line"></view>
							<view class="player-row" :class="{ winner: match.winner_id === match.player2_id && match.winner_id > 0 }">
								<text class="player-name">{{ match.player2_name || (match.player2_id > 0 ? '选手' : '轮空') }}</text>
								<text class="player-score" v-if="match.status === 2 && match.winner_id === match.player2_id">✓</text>
							</view>
							<view class="match-status">
								<text v-if="match.status === 0 && match.player1_id > 0 && match.player2_id > 0" class="status-pending">待开始</text>
								<text v-else-if="match.status === 1" class="status-live">进行中</text>
								<text v-else-if="match.status === 2" class="status-done">已结束</text>
							</view>
						</view>
					</view>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getTournamentBracket } from '@/api/tournament.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const matches = ref([])
const totalRounds = ref(0)
const loading = ref(true)
const tournamentId = ref(0)

const getMatchesByRound = (round) => {
	return matches.value.filter(m => m.round_number === round).sort((a, b) => a.match_order - b.match_order)
}

const getRoundName = (round) => {
	if (round === totalRounds.value) return '决赛'
	if (round === totalRounds.value - 1 && totalRounds.value > 1) return '半决赛'
	if (round === totalRounds.value - 2 && totalRounds.value > 2) return '1/4决赛'
	return `第${round}轮`
}

const fetchBracket = async () => {
	loading.value = true
	try {
		const res = await getTournamentBracket({ tournament_id: tournamentId.value })
		if (res.success) {
			matches.value = res.matches || []
			totalRounds.value = res.total_rounds || 0
		}
	} catch (e) {
		console.error('获取对阵图失败', e)
	} finally {
		loading.value = false
	}
}

onMounted(() => {
	const pages = getCurrentPages()
	const currentPage = pages[pages.length - 1]
	tournamentId.value = parseInt(currentPage.options.id || 0)
	if (tournamentId.value > 0) fetchBracket()
})
</script>

<style lang="scss" scoped>
.bracket-page {
	min-height: 100vh;
	background: #f1f5f9;
}
.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 60vh;
	.loading-text { font-size: 28rpx; color: #64748b; margin-top: 16rpx; }
}
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 60vh;
	.empty-icon { font-size: 80rpx; margin-bottom: 16rpx; }
	.empty-text { font-size: 28rpx; color: #64748b; }
}
.bracket-scroll {
	width: 100%;
	height: 100vh;
	white-space: nowrap;
}
.bracket-container {
	display: inline-flex;
	padding: 40rpx 24rpx;
	gap: 24rpx;
	min-height: 100vh;
	align-items: flex-start;
}
.round-column {
	display: flex;
	flex-direction: column;
	min-width: 300rpx;
	.round-title {
		text-align: center;
		padding: 16rpx;
		margin-bottom: 24rpx;
		font-size: 26rpx;
		font-weight: 600;
		color: #64748b;
		background: #ffffff;
		border-radius: 8rpx;
	}
}
.round-matches {
	display: flex;
	flex-direction: column;
	gap: 24rpx;
	justify-content: space-around;
	flex: 1;
}
.match-card {
	background: #ffffff;
	border-radius: 12rpx;
	padding: 16rpx 20rpx;
	border: 2rpx solid #e2e8f0;
	&.completed { border-color: #cbd5e1; }
	.player-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 10rpx 8rpx;
		border-radius: 6rpx;
		.player-name { font-size: 26rpx; color: #1e293b; }
		.player-score { font-size: 24rpx; color: #22c55e; font-weight: 700; }
		&.winner {
			background: rgba(34, 197, 94, 0.1);
			.player-name { color: #22c55e; font-weight: 600; }
		}
	}
	.vs-line {
		height: 2rpx;
		background: #e2e8f0;
		margin: 6rpx 0;
	}
	.match-status {
		text-align: center;
		margin-top: 8rpx;
		font-size: 20rpx;
		.status-pending { color: #64748b; }
		.status-live { color: #f59e0b; }
		.status-done { color: #64748b; }
	}
}

.bracket-page.dark-mode {
	background: #141109;

	.round-column .round-title {
		color: #d7c89b;
		background: rgba(255, 247, 225, 0.05);
	}

	.match-card {
		background: #1e180d;
		border-color: #3a2e16;

		&.completed {
			border-color: #4a3a1c;
		}

		.player-row .player-name {
			color: #fff7e1;
		}

		.player-row.winner .player-name {
			color: #22c55e;
		}

		.vs-line {
			background: #3a2e16;
		}

		.match-status .status-done {
			color: #9f926e;
		}
	}
}
</style>
