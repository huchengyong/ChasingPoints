<template>
	<view class="detail-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="detailError" class="loading-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="loading-text">{{ detailError.title }}</text>
			<text class="error-desc">{{ detailError.description }}</text>
			<button class="error-action" @tap="handleDetailErrorAction">{{ detailError.actionText }}</button>
		</view>

		<template v-else-if="achievement">
			<!-- 成就图标 -->
			<view class="hero-section" :class="{ unlocked: achievement.unlocked }">
				<view class="hero-icon">
					<image v-if="hasDetailIcon" :src="achievement.icon" mode="aspectFit" @error="handleDetailIconError"></image>
					<text v-else class="icon-emoji">{{ getFallbackEmoji(achievement) }}</text>
				</view>
				<text class="hero-name">{{ achievement.name }}</text>
				<view v-if="achievement.unlocked" class="unlock-badge">
					<uni-icons type="checkmarkempty" size="14" color="#fff"></uni-icons>
					<text>已解锁</text>
				</view>
			</view>

			<!-- 详情卡片 -->
			<view class="info-card">
				<view class="info-row condition-row">
					<text class="info-label">解锁条件</text>
					<text class="info-value">{{ achievement.description || '完成指定目标解锁此成就' }}</text>
				</view>
				<view class="info-row">
					<text class="info-label">分类</text>
					<text class="info-value">{{ getAchievementCategoryLabel(achievement.category) }}</text>
				</view>
				<view class="info-row">
					<text class="info-label">球种</text>
					<text class="info-value">{{ getAchievementGameTypeLabel(achievement.game_type) }}</text>
				</view>
				<view v-if="achievement.reward_title_name" class="info-row reward-row">
					<text class="info-label">奖励称号</text>
					<text class="info-value reward-title">{{ achievement.reward_title_name }}</text>
				</view>
			</view>

			<!-- 进度卡片 -->
			<view class="progress-card">
				<text class="progress-title">当前进度</text>
				<view class="progress-bar-large">
					<view
						class="progress-fill"
						:style="{ width: progressPercent + '%' }"
						:class="{ complete: achievement.unlocked }"
					></view>
				</view>
				<view class="progress-meta">
					<text class="progress-current">{{ achievement.progress || 0 }} / {{ achievement.threshold }}</text>
					<text class="progress-percent">{{ progressPercent }}%</text>
				</view>
			</view>

			<!-- 解锁时间 -->
			<view v-if="achievement.unlocked && achievement.unlocked_at" class="unlock-card">
				<uni-icons type="calendar" size="18" color="#E0AE12"></uni-icons>
				<text class="unlock-time">解锁于 {{ achievement.unlocked_at }}</text>
			</view>
		</template>

		<!-- 空状态 -->
		<view v-else class="empty-state">
			<text class="empty-text">成就不存在</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getAchievementDetail } from '@/api/achievement.js'
import { useUserStore } from '@/store/user.js'
import { getCachedAchievementDetail } from '@/utils/achievement-detail-cache.js'
import {
	getAchievementCategoryLabel,
	getAchievementFallbackEmoji,
	getAchievementGameTypeLabel
} from '@/utils/achievement-page.js'
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
const userStore = useUserStore()

const loading = ref(true)
const achievement = ref(null)
const achievementId = ref(0)
const iconFailed = ref(false)
const detailState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: null
}))
const detailError = computed(() => (
	detailState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(detailState.value.error, { resource: '成就详情' })
		: null
))

const progressPercent = computed(() => {
	if (!achievement.value) return 0
	if (achievement.value.unlocked) return 100
	const t = achievement.value.threshold || 1
	return Math.min(Math.round(((achievement.value.progress || 0) / t) * 100), 100)
})

const getFallbackEmoji = getAchievementFallbackEmoji
const hasDetailIcon = computed(() => Boolean(achievement.value?.icon && !iconFailed.value))
const handleDetailIconError = () => {
	iconFailed.value = true
}

const loadDetail = async () => {
	const nextState = beginAsyncPageLoad(detailState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: null
	})
	const pageRequest = getAsyncPageRequest(nextState)
	detailState.value = nextState
	loading.value = true
	try {
		const identity = {
			userId: userStore.userId,
			authGeneration: userStore.authGeneration
		}
		const cached = getCachedAchievementDetail(identity, achievementId.value)
		if (cached) {
			achievement.value = cached
			iconFailed.value = false
			detailState.value = resolveAsyncPageLoad(detailState.value, pageRequest, { data: cached, isEmpty: () => false })
			return
		}
		const res = await getAchievementDetail({ achievement_id: achievementId.value })
		if (detailState.value.requestId !== pageRequest.requestId || detailState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '加载成就详情失败', category: 'business' })
		achievement.value = res.achievement || null
		iconFailed.value = false
		detailState.value = resolveAsyncPageLoad(detailState.value, pageRequest, {
			data: achievement.value,
			isEmpty: value => !value
		})
	} catch (e) {
		detailState.value = rejectAsyncPageLoad(detailState.value, pageRequest, e)
	} finally {
		if (detailState.value.requestId === pageRequest.requestId && detailState.value.authGeneration === pageRequest.authGeneration) {
			loading.value = false
		}
	}
}

const handleDetailErrorAction = () => {
	if (detailState.value.error?.category === 'permission' || detailState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	loadDetail()
}

onLoad((options) => {
	achievementId.value = options.id || 0
	loadDetail()
})
</script>

<style lang="scss" scoped>
.detail-page {
	min-height: 100vh;
	background: #FAF8F2;
	padding-bottom: 60rpx;
}

.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 300rpx;
	.loading-text {
		margin-top: 16rpx;
		font-size: 28rpx;
		color: #9A8C67;
	}
}

.error-desc {
	max-width: 560rpx;
	color: #6E6242;
	font-size: 24rpx;
	line-height: 1.6;
	text-align: center;
}

.error-action {
	min-width: 200rpx;
	height: 76rpx;
	line-height: 76rpx;
	margin: 16rpx 0 0;
	padding: 0 32rpx;
	border-radius: 38rpx;
	background: #E0AE12;
	color: #ffffff;
	font-size: 28rpx;
	font-weight: 600;
}

.error-action::after {
	display: none;
}

.hero-section {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 60rpx 32rpx 40rpx;
	background: linear-gradient(180deg, #E9E2CF, #FAF8F2);

	&.unlocked {
		background: linear-gradient(180deg, rgba(224, 174, 18, 0.18), #FAF8F2);
	}

	.hero-icon {
		width: 160rpx;
		height: 160rpx;
		border-radius: 50%;
		background: #fff;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.1);
		margin-bottom: 24rpx;

		image {
			width: 112rpx;
			height: 112rpx;
		}

		.icon-emoji {
			font-size: 72rpx;
		}
	}

	.hero-name {
		font-size: 36rpx;
		font-weight: 700;
		color: #231C0B;
		margin-bottom: 12rpx;
	}

	.unlock-badge {
		display: flex;
		align-items: center;
		gap: 8rpx;
		background: #E0AE12;
		color: #fff;
		padding: 8rpx 24rpx;
		border-radius: 24rpx;
		font-size: 24rpx;
	}
}

.info-card {
	margin: 24rpx;
	background: #fff;
	border-radius: 16rpx;
	padding: 8rpx 0;

	.info-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 24rpx 32rpx;
		border-bottom: 1rpx solid #FAF8F2;

		&:last-child {
			border-bottom: none;
		}

		.info-label {
			font-size: 28rpx;
			color: #6E6242;
		}
		.info-value {
			font-size: 28rpx;
			line-height: 1.6;
			color: #231C0B;
			text-align: right;
			flex: 1;
			margin-left: 32rpx;
			word-break: break-word;
		}

		.reward-title {
			font-weight: 700;
			color: #b17d00;
		}
	}
}

.progress-card {
	margin: 0 24rpx;
	background: #fff;
	border-radius: 16rpx;
	padding: 32rpx;

	.progress-title {
		font-size: 28rpx;
		font-weight: 600;
		color: #231C0B;
		margin-bottom: 20rpx;
	}

	.progress-bar-large {
		width: 100%;
		height: 20rpx;
		background: #E9E2CF;
		border-radius: 10rpx;
		overflow: hidden;
		margin-bottom: 16rpx;

		.progress-fill {
			height: 100%;
			background: #9A8C67;
			border-radius: 10rpx;
			transition: width 0.3s;

			&.complete {
				background: #E0AE12;
			}
		}
	}

	.progress-meta {
		display: flex;
		justify-content: space-between;
		.progress-current, .progress-percent {
			font-size: 24rpx;
			color: #6E6242;
		}
	}
}

.unlock-card {
	display: flex;
	align-items: center;
	gap: 12rpx;
	margin: 24rpx;
	background: rgba(224, 174, 18, 0.12);
	border-radius: 16rpx;
	padding: 24rpx 32rpx;

	.unlock-time {
		font-size: 26rpx;
		color: #C69200;
	}
}

.empty-state {
	display: flex;
	justify-content: center;
	padding-top: 300rpx;
	.empty-text {
		font-size: 28rpx;
		color: #9A8C67;
	}
}

.detail-page.dark-mode {
	background: #141109;

	.loading-text,
	.empty-text,
	.info-label,
	.progress-current,
	.progress-percent {
		color: #9f926e;
	}

	.hero-section {
		background: linear-gradient(180deg, #2a2110, #141109);

		&.unlocked {
			background: linear-gradient(180deg, rgba(224, 174, 18, 0.22), #141109);
		}

		.hero-icon {
			background: #1e180d;
		}
	}

	.hero-name,
	.info-value,
	.progress-title {
		color: #fff7e1;
	}

	.info-card,
	.progress-card {
		background: #1e180d;
	}

	.info-card .info-row {
		border-bottom-color: #3a2e16;
	}

	.progress-card .progress-bar-large {
		background: #3a2e16;
	}

	.unlock-card {
		background: rgba(224, 174, 18, 0.18);

		.unlock-time {
			color: #f7e7a8;
		}
	}
}
</style>
