<template>
	<view class="rules-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 搜索栏 -->
		<view class="search-bar">
			<uni-icons type="search" size="18" color="#94a3b8"></uni-icons>
			<input
				class="search-input"
				v-model="searchKeyword"
				placeholder="搜索规则、术语..."
				confirm-type="search"
				@confirm="handleSearch"
			/>
		</view>

		<!-- 搜索结果 -->
		<view v-if="showSearchResults" class="search-results">
			<view class="section-title">
				<text>搜索结果</text>
				<text class="clear-btn" @tap="clearSearch">清除</text>
			</view>
			<view v-if="searchLoading" class="loading-state">
				<uni-icons type="spinner-cycle" size="28" color="#E0AE12"></uni-icons>
			</view>
			<view v-else-if="searchResults.length > 0">
				<view
					v-for="item in searchResults"
					:key="item.id"
					class="result-item"
					@tap="goToDetail(item.category)"
				>
					<text class="result-title">{{ item.title }}</text>
					<text class="result-category">{{ getRuleCategoryLabel(item.category) }} · {{ getTypeLabel(item.content_type) }}</text>
				</view>
			</view>
			<view v-else class="empty-hint">
				<text>未找到相关内容</text>
			</view>
		</view>

		<!-- 球种卡片 -->
		<view v-else>
			<view class="section-title"><text>选择球种</text></view>
			<view class="game-cards">
				<view class="game-card chinese-eight" @tap="goToDetail('chinese_eight')">
					<text class="card-emoji">⚫</text>
					<view class="card-info">
						<text class="card-name">中式八球</text>
						<text class="card-desc">中国流行的花色球比赛</text>
					</view>
					<uni-icons type="right" size="18" color="#94a3b8"></uni-icons>
				</view>
				<view class="game-card nine-ball" @tap="goToDetail('nine_ball')">
					<text class="card-emoji">🟡</text>
					<view class="card-info">
						<text class="card-name">九球追分</text>
						<text class="card-desc">9球制，必须按顺序击球</text>
					</view>
					<uni-icons type="right" size="18" color="#94a3b8"></uni-icons>
				</view>
				<view class="game-card snooker" @tap="goToDetail('snooker')">
					<text class="card-emoji">🔴</text>
					<view class="card-info">
						<text class="card-name">斯诺克</text>
						<text class="card-desc">22球制台球，得分制比赛</text>
					</view>
					<uni-icons type="right" size="18" color="#94a3b8"></uni-icons>
				</view>
				<view class="game-card nine-ball" @tap="goToDetail('american_nine')">
					<text class="card-emoji">🟠</text>
					<view class="card-info">
						<text class="card-name">美式九球</text>
						<text class="card-desc">赛局制九球，支持普胜、小金和大金</text>
					</view>
					<uni-icons type="right" size="18" color="#94a3b8"></uni-icons>
				</view>
			</view>

			<!-- 术语词典入口 -->
			<view class="section-title"><text>更多</text></view>
			<view class="glossary-entry" @tap="goToGlossary">
				<view class="glossary-left">
					<text class="glossary-emoji">📚</text>
					<view class="glossary-info">
						<text class="glossary-name">术语词典</text>
						<text class="glossary-desc">台球专业术语大全</text>
					</view>
				</view>
				<uni-icons type="right" size="18" color="#94a3b8"></uni-icons>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { searchRules } from '@/api/rules.js'
import { getRuleCategoryLabel } from '@/utils/game-types.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const searchKeyword = ref('')
const searchResults = ref([])
const searchLoading = ref(false)
const showSearchResults = ref(false)

const getTypeLabel = (type) => {
	const map = { rule: '规则', foul: '犯规', glossary: '术语' }
	return map[type] || type
}

const handleSearch = async () => {
	if (!searchKeyword.value.trim()) return
	showSearchResults.value = true
	searchLoading.value = true
	try {
		const res = await searchRules({ keyword: searchKeyword.value.trim() })
		searchResults.value = res.list || res || []
	} catch (e) {
		console.error('搜索失败:', e)
	} finally {
		searchLoading.value = false
	}
}

const clearSearch = () => {
	searchKeyword.value = ''
	searchResults.value = []
	showSearchResults.value = false
}

const goToDetail = (category) => {
	uni.navigateTo({ url: '/subPages/rules/detail?category=' + category })
}

const goToGlossary = () => {
	uni.navigateTo({ url: '/subPages/rules/glossary' })
}
</script>

<style lang="scss" scoped>
.rules-page {
	min-height: 100vh;
	background: #f1f5f9;
	padding: 24rpx;
}

.search-bar {
	display: flex;
	align-items: center;
	background: #fff;
	border-radius: 16rpx;
	padding: 16rpx 24rpx;
	margin-bottom: 32rpx;
	gap: 12rpx;

	.search-input {
		flex: 1;
		font-size: 28rpx;
		color: #1e293b;
	}
}

.section-title {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 16rpx 8rpx;
	font-size: 28rpx;
	font-weight: 600;
	color: #64748b;

	.clear-btn {
		font-size: 24rpx;
		color: #C69200;
		font-weight: 400;
	}
}

.game-cards {
	.game-card {
		display: flex;
		align-items: center;
		background: #fff;
		border-radius: 16rpx;
		padding: 32rpx 28rpx;
		margin-bottom: 16rpx;
		gap: 20rpx;

		.card-emoji {
			font-size: 48rpx;
		}

		.card-info {
			flex: 1;
			.card-name {
				font-size: 30rpx;
				font-weight: 600;
				color: #1e293b;
				display: block;
				margin-bottom: 6rpx;
			}
			.card-desc {
				font-size: 24rpx;
				color: #94a3b8;
			}
		}
	}
}

.glossary-entry {
	display: flex;
	align-items: center;
	justify-content: space-between;
	background: #fff;
	border-radius: 16rpx;
	padding: 32rpx 28rpx;

	.glossary-left {
		display: flex;
		align-items: center;
		gap: 20rpx;

		.glossary-emoji {
			font-size: 48rpx;
		}
		.glossary-info {
			.glossary-name {
				font-size: 30rpx;
				font-weight: 600;
				color: #1e293b;
				display: block;
				margin-bottom: 6rpx;
			}
			.glossary-desc {
				font-size: 24rpx;
				color: #94a3b8;
			}
		}
	}
}

.search-results {
	.result-item {
		background: #fff;
		border-radius: 12rpx;
		padding: 24rpx 28rpx;
		margin-bottom: 12rpx;

		.result-title {
			font-size: 28rpx;
			color: #1e293b;
			display: block;
			margin-bottom: 8rpx;
		}
		.result-category {
			font-size: 22rpx;
			color: #94a3b8;
		}
	}
}

.loading-state {
	display: flex;
	justify-content: center;
	padding: 60rpx 0;
}

.empty-hint {
	text-align: center;
	padding: 60rpx 0;
	font-size: 28rpx;
	color: #94a3b8;
}

.rules-page.dark-mode {
	background: #141109;

	.search-bar,
	.game-cards .game-card,
	.glossary-entry,
	.search-results .result-item {
		background: #1e180d;
	}

	.search-input,
	.game-cards .game-card .card-info .card-name,
	.glossary-entry .glossary-info .glossary-name,
	.search-results .result-title {
		color: #fff7e1;
	}

	.section-title,
	.game-cards .game-card .card-info .card-desc,
	.glossary-entry .glossary-info .glossary-desc,
	.search-results .result-category,
	.empty-hint {
		color: #9f926e;
	}
}
</style>
