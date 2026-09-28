<template>
	<view class="bracket-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载对阵图...</text>
		</view>

		<view v-else-if="bracketPageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ bracketPageError.title }}</text>
			<text class="error-text">{{ bracketPageError.description }}</text>
			<button class="retry-btn" @tap="handleBracketErrorAction">{{ bracketPageError.actionText }}</button>
		</view>

		<view v-else-if="bracketState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
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
import { computed, ref, onMounted } from 'vue'
import { getTournamentBracket } from '@/api/tournament.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	ASYNC_PAGE_STATUS,
	beginAsyncPageLoad,
	createAsyncPageState,
	getAsyncPageRequest,
	rejectAsyncPageLoad,
	resolveAsyncPageErrorFeedback,
	resolveAsyncPageLoad
} from '@/utils/async-page-state.js'
import { createRequestError } from '@/utils/request-errors.js'

const { isDarkMode } = usePageTheme()

const matches = ref([])
const totalRounds = ref(0)
const loading = ref(true)
const tournamentId = ref(0)
const bracketState = ref(createAsyncPageState({ data: [] }))
const bracketPageError = computed(() => (
	bracketState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(bracketState.value.error, { resource: '赛事对阵图' })
		: null
))

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
	const nextState = beginAsyncPageLoad(bracketState.value, { emptyData: [] })
	const pageRequest = getAsyncPageRequest(nextState)
	bracketState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	if (!tournamentId.value) {
		bracketState.value = rejectAsyncPageLoad(bracketState.value, pageRequest, createRequestError({
			message: '未找到赛事信息，请返回赛事列表后重试。',
			category: 'not-found'
		}))
		loading.value = false
		return
	}
	try {
		const res = await getTournamentBracket({ tournament_id: tournamentId.value })
		if (bracketState.value.requestId !== pageRequest.requestId) return
		if (!res?.success) throw createRequestError({ message: res?.message || '获取对阵图失败', category: 'business' })
		matches.value = res.matches || []
		totalRounds.value = res.total_rounds || 0
		bracketState.value = resolveAsyncPageLoad(bracketState.value, pageRequest, { data: matches.value })
	} catch (e) {
		if (bracketState.value.requestId !== pageRequest.requestId) return
		bracketState.value = rejectAsyncPageLoad(bracketState.value, pageRequest, e)
	} finally {
		if (bracketState.value.requestId !== pageRequest.requestId) return
		loading.value = false
	}
}

const retryBracket = () => fetchBracket()

const handleBracketErrorAction = () => {
	if (bracketState.value.error?.category === 'permission' || bracketState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryBracket()
}

onMounted(() => {
	const pages = getCurrentPages()
	const currentPage = pages[pages.length - 1]
	tournamentId.value = parseInt(currentPage.options.id || 0)
	fetchBracket()
})
</script>

<style lang="scss" scoped>
.bracket-page {
	min-height: 100vh;
	background: #FAF8F2;
}
.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 60vh;
	.loading-text { font-size: 28rpx; color: #6E6242; margin-top: 16rpx; }
}
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 60vh;
	.empty-icon { font-size: 80rpx; margin-bottom: 16rpx; }
	.empty-text { font-size: 28rpx; color: #6E6242; }
}
.error-state { gap: 16rpx; padding: 0 32rpx; text-align: center; }
.error-text { font-size: 24rpx; line-height: 1.7; color: #6E6242; }
.retry-btn { min-width: 200rpx; height: 76rpx; line-height: 76rpx; margin: 12rpx 0 0; padding: 0 32rpx; border-radius: 38rpx; background: #E0AE12; color: #ffffff; font-size: 28rpx; font-weight: 600; }
.retry-btn::after { display: none; }
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
		color: #6E6242;
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
	border: 2rpx solid #E9E2CF;
	&.completed { border-color: #E9E2CF; }
	.player-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 10rpx 8rpx;
		border-radius: 6rpx;
		.player-name { font-size: 26rpx; color: #231C0B; }
		.player-score { font-size: 24rpx; color: #22c55e; font-weight: 700; }
		&.winner {
			background: rgba(34, 197, 94, 0.1);
			.player-name { color: #22c55e; font-weight: 600; }
		}
	}
	.vs-line {
		height: 2rpx;
		background: #E9E2CF;
		margin: 6rpx 0;
	}
	.match-status {
		text-align: center;
		margin-top: 8rpx;
		font-size: 20rpx;
		.status-pending { color: #6E6242; }
		.status-live { color: #f59e0b; }
		.status-done { color: #6E6242; }
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

	.error-text,
	.empty-text {
		color: #9f926e;
	}
}
</style>
