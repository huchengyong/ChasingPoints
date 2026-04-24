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

		<!-- 分类 Tab -->
		<scroll-view scroll-x class="category-tabs" :show-scrollbar="false">
			<view
				v-for="tab in categoryTabs"
				:key="tab.key"
				class="tab-item"
				:class="{ active: currentCategory === tab.key }"
				@tap="switchCategory(tab.key)"
			>
				<text>{{ tab.label }}</text>
			</view>
		</scroll-view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 成就列表 -->
		<view v-else class="achievement-grid">
			<view
				v-for="item in filteredList"
				:key="item.id"
				class="achievement-card"
				:class="{ unlocked: item.unlocked }"
				@tap="goToDetail(item.id)"
			>
				<view class="card-icon" :class="{ locked: !item.unlocked }">
					<text class="icon-emoji">{{ getCategoryEmoji(item.category) }}</text>
				</view>
				<text class="card-name">{{ item.name }}</text>
				<view class="progress-bar">
					<view
						class="progress-fill"
						:style="{ width: getProgress(item) + '%' }"
						:class="{ complete: item.unlocked }"
					></view>
				</view>
				<text class="progress-text">{{ item.unlocked ? '已解锁' : item.progress + '/' + item.threshold }}</text>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-if="!loading && filteredList.length === 0" class="empty-state">
			<text class="empty-text">暂无成就数据</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getAchievementList, getUserTitles } from '@/api/achievement.js'
import {
	ACHIEVEMENT_CATEGORY_TABS,
	filterAchievementsByCategory,
	getAchievementCategoryEmoji
} from '@/utils/achievement-page.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const loading = ref(true)
const achievementList = ref([])
const equippedTitle = ref('')
const currentCategory = ref('all')
const loaded = ref(false)
const categoryTabs = ACHIEVEMENT_CATEGORY_TABS

const filteredList = computed(() => {
	return filterAchievementsByCategory(achievementList.value, currentCategory.value)
})

const getCategoryEmoji = getAchievementCategoryEmoji

const getProgress = (item) => {
	if (item.unlocked) return 100
	if (!item.threshold || item.threshold === 0) return 0
	return Math.min(Math.round((item.progress / item.threshold) * 100), 100)
}

const switchCategory = (key) => {
	currentCategory.value = key
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

.category-tabs {
	white-space: nowrap;
	padding: 0 24rpx 20rpx;

	.tab-item {
		display: inline-block;
		padding: 12rpx 28rpx;
		margin-right: 16rpx;
		border-radius: 32rpx;
		background: #fff;
		font-size: 26rpx;
		color: #64748b;

		text {
			color: inherit;
		}

		&.active {
			background: #E0AE12;
			color: #ffffff;
		}

		&.active text {
			color: #ffffff;
		}
	}

	&::-webkit-scrollbar {
		display: none;
		width: 0;
		height: 0;
	}

	scrollbar-width: none;
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

.achievement-grid {
	display: flex;
	flex-wrap: wrap;
	padding: 0 16rpx;

	.achievement-card {
		width: calc(50% - 24rpx);
		margin: 8rpx 12rpx;
		background: #fff;
		border-radius: 16rpx;
		padding: 28rpx 20rpx;
		display: flex;
		flex-direction: column;
		align-items: center;

		&.unlocked {
			border: 2rpx solid #E0AE12;
		}

		.card-icon {
			width: 96rpx;
			height: 96rpx;
			border-radius: 50%;
			background: rgba(224, 174, 18, 0.12);
			display: flex;
			align-items: center;
			justify-content: center;
			margin-bottom: 16rpx;

			&.locked {
				background: #f1f5f9;
				opacity: 0.5;
			}

			.icon-emoji {
				font-size: 44rpx;
			}
		}

		.card-name {
			font-size: 26rpx;
			font-weight: 500;
			color: #1e293b;
			margin-bottom: 12rpx;
			text-align: center;
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
			font-size: 22rpx;
			color: #94a3b8;
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
