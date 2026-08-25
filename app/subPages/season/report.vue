<template>
	<view class="report-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" :color="isDarkMode ? '#fff' : '#E0AE12'"></uni-icons>
		</view>

		<view v-else-if="report" class="report-content">
			<view v-if="seasonReportRefreshError" class="refresh-error-banner">
				<text>{{ seasonReportRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retrySeasonReport">重试</text>
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

		<view v-else-if="seasonReportPageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ seasonReportPageError.title }}</text>
			<text class="error-text">{{ seasonReportPageError.description }}</text>
			<button class="retry-btn" @tap="handleSeasonReportErrorAction">{{ seasonReportPageError.actionText }}</button>
		</view>

		<view v-else-if="seasonReportState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
			<text class="empty-text">暂无赛季报告</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getSeasonReport } from '@/api/season.js'
import { GAME_TYPE_TABS, getGameTypeLabel } from '@/utils/game-types.js'
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
const gameTypeTabs = GAME_TYPE_TABS
const report = ref(null)
const loading = ref(true)
const seasonId = ref(0)
const currentGameType = ref(3)
const seasonReportState = ref(createAsyncPageState({ data: null }))
const seasonReportPageError = computed(() => (
	seasonReportState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(seasonReportState.value.error, { resource: '赛季报告' })
		: null
))
const seasonReportRefreshError = computed(() => (
	seasonReportState.value.refreshError
		? resolveAsyncPageErrorFeedback(seasonReportState.value.refreshError, { resource: '赛季报告' })
		: null
))

const winRate = computed(() => {
	if (!report.value || !report.value.record || report.value.record.matches_played === 0) return 0
	return Math.round((report.value.record.wins / report.value.record.matches_played) * 100)
})

const formatUnlockTime = (value) => {
	if (!value) return '已解锁'
	return value.slice(0, 10)
}

const fetchReport = async () => {
	const nextState = beginAsyncPageLoad(seasonReportState.value, { emptyData: null })
	const pageRequest = getAsyncPageRequest(nextState)
	seasonReportState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	if (!seasonId.value) {
		seasonReportState.value = rejectAsyncPageLoad(seasonReportState.value, pageRequest, createRequestError({
			message: '未找到赛季信息，请返回赛季页后重试。',
			category: 'not-found'
		}))
		loading.value = false
		return
	}
	try {
		const res = await getSeasonReport({ season_id: seasonId.value, game_type: currentGameType.value })
		if (seasonReportState.value.requestId !== pageRequest.requestId) return
		if (!res?.success) throw createRequestError({ message: res?.message || '获取赛季报告失败', category: 'business' })
		report.value = res.report || null
		seasonReportState.value = resolveAsyncPageLoad(seasonReportState.value, pageRequest, { data: report.value })
	} catch (e) {
		if (seasonReportState.value.requestId !== pageRequest.requestId) return
		seasonReportState.value = rejectAsyncPageLoad(seasonReportState.value, pageRequest, e)
	} finally {
		if (seasonReportState.value.requestId !== pageRequest.requestId) return
		loading.value = false
	}
}

const retrySeasonReport = () => fetchReport()

const handleSeasonReportErrorAction = () => {
	if (seasonReportState.value.error?.category === 'permission' || seasonReportState.value.error?.category === 'not-found') {
		goBack()
		return
	}
	retrySeasonReport()
}

const goBack = () => { uni.navigateBack() }

const handleGameTypeChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	report.value = null
	seasonReportState.value = {
		...seasonReportState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: seasonReportState.value.requestId + 1,
		data: null,
		hasData: false,
		error: null,
		refreshError: null
	}
	fetchReport()
}

onLoad((options) => {
	seasonId.value = parseInt(options?.id || 0)
	const gameType = parseInt(options?.game_type || 0)
	if (gameType > 0) {
		currentGameType.value = gameType
	}
	fetchReport()
})
</script>

<style lang="scss" scoped>
.report-page {
	min-height: 100vh;
	background: linear-gradient(180deg, #FAF8F2 0%, #E9E2CF 100%);
	padding: 80rpx 24rpx 40rpx;

	&.dark-mode {
		background: linear-gradient(180deg, #241d10 0%, #141109 100%);

		.game-type-tab {
			background: rgba(255,255,255,0.08);
			border-color: rgba(255,255,255,0.14);

			text {
				color: #D7C89B;
			}

			&.active {
				border-color: transparent;

				text {
					color: #231c0b;
				}
			}
		}

		.report-card {
			background: rgba(255,255,255,0.08);
			border-color: transparent;

			.card-title {
				color: #9F926E;
			}
		}

		.summary-card .summary-stats .stat-item {
			.stat-value {
				color: #fff;
			}

			.stat-label {
				color: #9F926E;
			}
		}

		.rank-card {
			.rank-change {
				.rank-from,
				.rank-to {
					.rank-val {
						color: #D7C89B;

						&.up {
							color: #22c55e;
						}
					}

					.rank-label {
						color: #9F926E;
					}
				}

				.rank-arrow {
					color: #9F926E;
				}
			}

			.peak-score,
			.final-rank {
				color: #9F926E;
			}
		}

		.type-card .type-item {
			.type-name {
				color: #D7C89B;
			}

			.type-bar {
				background: rgba(255,255,255,0.1);
			}

			.type-wins {
				color: #9F926E;
			}
		}

		.achievement-card {
			.achievement-item {
				border-top-color: rgba(255,255,255,0.08);
			}

			.achievement-name {
				color: #fff;
			}

			.achievement-desc {
				color: #9F926E;
			}

			.achievement-time {
				color: #9F926E;
			}
		}

		.back-btn {
			background: rgba(255,255,255,0.1);
			border-color: transparent;
			color: #fff;
		}

		.empty-state .empty-text {
			color: #9F926E;
		}
	}
}
.loading-state {
	display: flex;
	justify-content: center;
	align-items: center;
	min-height: 60vh;
}
.refresh-error-banner { display: flex; align-items: center; justify-content: space-between; gap: 20rpx; padding: 18rpx 22rpx; border-radius: 14rpx; background: rgba(224, 174, 18, 0.12); color: #8a5b00; font-size: 24rpx; }
.refresh-error-action { flex-shrink: 0; color: #a86f00; font-weight: 600; }
.error-state { display: flex; flex-direction: column; align-items: center; gap: 16rpx; padding-left: 32rpx; padding-right: 32rpx; text-align: center; }
.error-text { font-size: 24rpx; line-height: 1.7; color: #6E6242; }
.retry-btn { min-width: 200rpx; height: 76rpx; line-height: 76rpx; margin: 12rpx 0 0; padding: 0 32rpx; border-radius: 38rpx; background: #E0AE12; color: #ffffff; font-size: 28rpx; font-weight: 600; }
.retry-btn::after { display: none; }
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
	background: rgba(255,255,255,0.92);
	border: 1rpx solid #E9E2CF;
	box-sizing: border-box;
	text-align: center;
	text {
		font-size: 24rpx;
		font-weight: 600;
		color: #6E6242;
	}
		&.active {
			background: linear-gradient(135deg, #e0ae12 0%, #c69200 100%);
			border-color: transparent;
			text {
				color: #231c0b;
			}
		}
}
.report-card {
	background: #ffffff;
	border: 1rpx solid #E9E2CF;
	border-radius: 20rpx;
	padding: 32rpx;
	backdrop-filter: blur(10px);
	.card-title {
		font-size: 30rpx;
		font-weight: 600;
		color: #231C0B;
		margin-bottom: 24rpx;
		display: block;
	}
}
.summary-card {
	.summary-stats {
		display: flex;
		justify-content: space-around;
		.stat-item { text-align: center;
			.stat-value { font-size: 48rpx; font-weight: 800; color: #231C0B; display: block; }
			.stat-label { font-size: 24rpx; color: #6E6242; }
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
			.rank-val { font-size: 40rpx; font-weight: 700; color: #231C0B; display: block; &.up { color: #16a34a; } }
			.rank-label { font-size: 22rpx; color: #6E6242; }
		}
		.rank-arrow { font-size: 40rpx; color: #9A8C67; }
	}
	.peak-score, .final-rank {
		font-size: 26rpx;
		color: #6E6242;
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
		.type-name { font-size: 26rpx; color: #231C0B; width: 140rpx; }
		.type-bar { flex: 1; height: 16rpx; background: #E9E2CF; border-radius: 8rpx; overflow: hidden;
				.type-fill { height: 100%; background: #e0ae12; border-radius: 8rpx; min-width: 10rpx; }
			}
		.type-wins { font-size: 24rpx; color: #6E6242; width: 80rpx; text-align: right; }
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
		border-top: 1rpx solid #E9E2CF;
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
		color: #231C0B;
	}
	.achievement-desc {
		font-size: 22rpx;
		color: #6E6242;
		line-height: 1.5;
	}
	.achievement-time {
		font-size: 22rpx;
		color: #9A8C67;
		flex-shrink: 0;
	}
}
.back-btn {
	text-align: center;
	padding: 24rpx;
	background: #ffffff;
	border: 1rpx solid #E9E2CF;
	border-radius: 16rpx;
	color: #231C0B;
	font-size: 28rpx;
	margin-top: 16rpx;
}
.empty-state {
	display: flex;
	justify-content: center;
	align-items: center;
	min-height: 60vh;
	.empty-text { font-size: 28rpx; color: #6E6242; }
}
</style>
