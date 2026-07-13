<template>
	<view class="achievement-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 当前称号 -->
		<view class="title-bar" @tap="goToTitles">
			<view class="title-info">
				<text class="title-label">当前称号</text>
				<text class="title-name">{{ equippedTitle || '未装备' }}</text>
			</view>
			<view class="title-arrow">
				<uni-icons type="right" size="16" color="#94a3b8"></uni-icons>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 成就列表 -->
		<view v-else class="achievement-sections">
			<view
				v-for="group in achievementGroups"
				:key="group.key"
				class="achievement-section"
			>
				<view class="section-header">
					<view class="section-title-wrap">
						<text class="section-emoji">{{ getCategoryEmoji(group.key) }}</text>
						<text class="section-title">{{ group.label }}成就</text>
					</view>
					<text class="section-count">{{ group.list.length }}项</text>
				</view>

				<view
					v-for="item in group.list"
					:key="item.id"
					class="achievement-row"
					:class="{ unlocked: item.unlocked }"
					@tap="goToDetail(item.id)"
				>
					<view class="row-icon" :class="{ locked: !item.unlocked }">
						<text class="icon-emoji">{{ getCategoryEmoji(item.category) }}</text>
					</view>
					<view
						class="row-main"
					>
						<text class="row-name">{{ item.name }}</text>
						<view class="progress-bar">
							<view
								class="progress-fill"
								:style="{ width: getProgress(item) + '%' }"
								:class="{ complete: item.unlocked }"
							></view>
						</view>
					</view>
					<text class="progress-text" :class="{ complete: item.unlocked }">{{ getProgressText(item) }}</text>
				</view>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-if="!loading && achievementGroups.length === 0" class="empty-state">
			<text class="empty-text">暂无成就数据</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getAchievementList, getUserTitles } from '@/api/achievement.js'
import {
	getAchievementCategoryEmoji,
	groupAchievementsByCategory
} from '@/utils/achievement-page.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const loading = ref(true)
const achievementList = ref([])
const equippedTitle = ref('')
const loaded = ref(false)

const achievementGroups = computed(() => {
	return groupAchievementsByCategory(achievementList.value)
})

const getCategoryEmoji = getAchievementCategoryEmoji

const getProgress = (item) => {
	if (item.unlocked) return 100
	if (!item.threshold || item.threshold === 0) return 0
	return Math.min(Math.round((item.progress / item.threshold) * 100), 100)
}

const getProgressText = (item) => {
	if (item.unlocked) return '已解锁'
	return (item.progress || 0) + '/' + (item.threshold || 0)
}

const goToDetail = (id) => {
	uni.navigateTo({ url: '/subPages/achievement/detail?id=' + id })
}

const goToTitles = () => {
	uni.navigateTo({ url: '/subPages/achievement/titles' })
}

const loadData = async () => {
	loading.value = true
	try {
		const achRes = await getAchievementList()
		achievementList.value = achRes.list || achRes || []
		await loadEquippedTitle()
		loaded.value = true
	} catch (e) {
		console.error('加载成就数据失败:', e)
	} finally {
		loading.value = false
	}
}

const loadEquippedTitle = async () => {
	try {
		const titleRes = await getUserTitles()
		const titles = titleRes.list || titleRes || []
		const equipped = titles.find(t => t.equipped)
		equippedTitle.value = equipped ? equipped.title_name : ''
	} catch (e) {
		console.error('加载当前称号失败:', e)
	}
}

onLoad(() => {
	loadData()
})

onShow(() => {
	if (loaded.value) {
		loadEquippedTitle()
	}
})
</script>

<style lang="scss" scoped>
.achievement-page {
	min-height: 100vh;
	background: #f1f5f9;
	padding-bottom: 40rpx;
}

.title-bar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin: 24rpx;
	padding: 28rpx 32rpx;
	background: linear-gradient(135deg, #E0AE12, #F59E0B);
	border-radius: 20rpx;
	color: #fff;

	.title-info {
		display: flex;
		flex-direction: column;
		.title-label {
			font-size: 24rpx;
			opacity: 0.8;
		}
		.title-name {
			font-size: 34rpx;
			font-weight: 600;
			margin-top: 8rpx;
		}
	}
}

.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 200rpx;
	.loading-text {
		margin-top: 16rpx;
		font-size: 28rpx;
		color: #94a3b8;
	}
}

.achievement-sections {
	padding: 0 24rpx;

	.achievement-section {
		margin-bottom: 28rpx;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin: 8rpx 4rpx 16rpx;

		.section-title-wrap {
			display: flex;
			align-items: center;
		}

		.section-emoji {
			font-size: 32rpx;
			margin-right: 12rpx;
		}

		.section-title {
			font-size: 30rpx;
			font-weight: 700;
			color: #1e293b;
		}

		.section-count {
			font-size: 24rpx;
			color: #94a3b8;
		}
	}

	.achievement-row {
		display: flex;
		align-items: center;
		width: 100%;
		box-sizing: border-box;
		margin-bottom: 16rpx;
		padding: 24rpx;
		background: #fff;
		border-radius: 18rpx;

		&.unlocked {
			border: 2rpx solid #E0AE12;
		}

		.row-icon {
			width: 76rpx;
			height: 76rpx;
			border-radius: 22rpx;
			background: rgba(224, 174, 18, 0.12);
			display: flex;
			align-items: center;
			justify-content: center;
			margin-right: 20rpx;
			flex-shrink: 0;

			&.locked {
				background: #f1f5f9;
				opacity: 0.5;
			}

			.icon-emoji {
				font-size: 38rpx;
			}
		}

		.row-main {
			flex: 1;
			min-width: 0;
			margin-right: 20rpx;
		}

		.row-name {
			display: block;
			font-size: 28rpx;
			font-weight: 600;
			color: #1e293b;
			margin-bottom: 14rpx;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		.progress-bar {
			width: 100%;
			height: 12rpx;
			background: #e2e8f0;
			border-radius: 6rpx;
			overflow: hidden;
			margin-bottom: 8rpx;

			.progress-fill {
				height: 100%;
				background: #94a3b8;
				border-radius: 6rpx;
				transition: width 0.3s;

				&.complete {
					background: #E0AE12;
				}
			}
		}

		.progress-text {
			min-width: 96rpx;
			font-size: 24rpx;
			text-align: right;
			color: #94a3b8;
			flex-shrink: 0;

			&.complete {
				color: #E0AE12;
				font-weight: 600;
			}
		}
	}
}

.empty-state {
	display: flex;
	justify-content: center;
	padding-top: 200rpx;
	.empty-text {
		font-size: 28rpx;
		color: #94a3b8;
	}
}
</style>
