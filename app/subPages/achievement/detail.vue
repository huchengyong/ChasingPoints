<template>
	<view class="detail-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<template v-else-if="achievement">
			<!-- 成就图标 -->
			<view class="hero-section" :class="{ unlocked: achievement.unlocked }">
				<view class="hero-icon">
					<text class="icon-emoji">{{ getCategoryEmoji(achievement.category) }}</text>
				</view>
				<text class="hero-name">{{ achievement.name }}</text>
				<view v-if="achievement.unlocked" class="unlock-badge">
					<uni-icons type="checkmarkempty" size="14" color="#fff"></uni-icons>
					<text>已解锁</text>
				</view>
			</view>

			<!-- 详情卡片 -->
			<view class="info-card">
				<view class="info-row">
					<text class="info-label">描述</text>
					<text class="info-value">{{ achievement.description || '完成指定目标解锁此成就' }}</text>
				</view>
				<view class="info-row">
					<text class="info-label">分类</text>
					<text class="info-value">{{ getAchievementCategoryLabel(achievement.category) }}</text>
				</view>
				<view class="info-row">
					<text class="info-label">解锁条件</text>
					<text class="info-value">累计达成 {{ achievement.threshold }} 次</text>
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
import { getAchievementList } from '@/api/achievement.js'
import { getAchievementCategoryEmoji, getAchievementCategoryLabel } from '@/utils/achievement-page.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const loading = ref(true)
const achievement = ref(null)
const achievementId = ref(0)

const progressPercent = computed(() => {
	if (!achievement.value) return 0
	if (achievement.value.unlocked) return 100
	const t = achievement.value.threshold || 1
	return Math.min(Math.round(((achievement.value.progress || 0) / t) * 100), 100)
})

const getCategoryEmoji = getAchievementCategoryEmoji

const loadDetail = async () => {
	loading.value = true
	try {
		const res = await getAchievementList()
		const list = res.list || res || []
		achievement.value = list.find(a => String(a.id) === String(achievementId.value)) || null
	} catch (e) {
		console.error('加载成就详情失败:', e)
	} finally {
		loading.value = false
	}
}

onLoad((options) => {
	achievementId.value = options.id || 0
	loadDetail()
})
</script>

<style lang="scss" scoped>
.detail-page {
	min-height: 100vh;
	background: #f1f5f9;
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
		color: #94a3b8;
	}
}

.hero-section {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 60rpx 32rpx 40rpx;
	background: linear-gradient(180deg, #e2e8f0, #f1f5f9);

	&.unlocked {
		background: linear-gradient(180deg, rgba(224, 174, 18, 0.18), #f1f5f9);
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

		.icon-emoji {
			font-size: 72rpx;
		}
	}

	.hero-name {
		font-size: 36rpx;
		font-weight: 700;
		color: #1e293b;
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
		border-bottom: 1rpx solid #f1f5f9;

		&:last-child {
			border-bottom: none;
		}

		.info-label {
			font-size: 28rpx;
			color: #64748b;
		}
		.info-value {
			font-size: 28rpx;
			color: #1e293b;
			text-align: right;
			flex: 1;
			margin-left: 32rpx;
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
		color: #1e293b;
		margin-bottom: 20rpx;
	}

	.progress-bar-large {
		width: 100%;
		height: 20rpx;
		background: #e2e8f0;
		border-radius: 10rpx;
		overflow: hidden;
		margin-bottom: 16rpx;

		.progress-fill {
			height: 100%;
			background: #94a3b8;
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
			color: #64748b;
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
		color: #94a3b8;
	}
}
</style>
