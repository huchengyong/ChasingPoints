<template>
	<view class="detail-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 球种标题 -->
		<view class="category-header">
			<text class="category-emoji">{{ getCategoryEmoji() }}</text>
			<text class="category-name">{{ getCategoryLabel() }}</text>
		</view>

		<!-- 内容类型 Tab -->
		<view class="content-tabs">
			<view
				v-for="tab in contentTabs"
				:key="tab.key"
				class="tab-item"
				:class="{ active: currentType === tab.key }"
				@tap="switchType(tab.key)"
			>
				<text>{{ tab.label }}</text>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
		</view>

		<!-- 内容列表 -->
		<view v-else class="content-list">
			<view
				v-for="(item, index) in contentList"
				:key="item.id || index"
				class="content-item"
			>
				<view class="item-header" @tap="toggleItem(index)">
					<text class="item-index">{{ index + 1 }}</text>
					<text class="item-title">{{ item.title }}</text>
					<uni-icons
						:type="expandedIndex === index ? 'up' : 'down'"
						size="16"
						color="#9A8C67"
					></uni-icons>
				</view>
				<view v-if="expandedIndex === index" class="item-content">
					<text>{{ item.content }}</text>
				</view>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-if="!loading && contentList.length === 0" class="empty-state">
			<text class="empty-text">暂无内容</text>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getRuleContent } from '@/api/rules.js'
import { usePublicReadStore } from '@/store/publicRead.js'
import { getRuleCategoryLabel } from '@/utils/game-types.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()
const publicReadStore = usePublicReadStore()

const category = ref('snooker')
const currentType = ref('rule')
const contentList = ref([])
const loading = ref(true)
const expandedIndex = ref(-1)

const contentTabs = [
	{ key: 'rule', label: '规则' },
	{ key: 'foul', label: '犯规' },
	{ key: 'glossary', label: '术语' }
]

const resolveRuleCategory = (value) => {
	return value === 'american_nine' ? 'nine_ball' : value
}

const getCategoryEmoji = () => {
	const map = { snooker: '🔴', nine_ball: '🟡', chinese_eight: '⚫', american_nine: '🟠' }
	return map[category.value] || '🎱'
}

const getCategoryLabel = () => {
	return getRuleCategoryLabel(category.value, '规则')
}

const switchType = (type) => {
	currentType.value = type
	expandedIndex.value = -1
	loadContent()
}

const toggleItem = (index) => {
	expandedIndex.value = expandedIndex.value === index ? -1 : index
}

const loadContent = async () => {
	loading.value = true
	try {
		const resolvedCategory = resolveRuleCategory(category.value)
		const res = await publicReadStore.loadStatic(
			`rules:content:${resolvedCategory}:${currentType.value}`,
			() => getRuleContent({ category: resolvedCategory, content_type: currentType.value })
		)
		contentList.value = res.list || res || []
	} catch (e) {
		console.error('加载规则内容失败:', e)
		contentList.value = []
	} finally {
		loading.value = false
	}
}

onLoad((options) => {
	if (options.category) {
		category.value = options.category
	}
	// 动态设置导航栏标题
	uni.setNavigationBarTitle({ title: getCategoryLabel() + ' - 规则' })
	loadContent()
})
</script>

<style lang="scss" scoped>
.detail-page {
	min-height: 100vh;
	background: #FAF8F2;
}

.category-header {
	display: flex;
	align-items: center;
	gap: 16rpx;
	padding: 32rpx 32rpx 16rpx;

	.category-emoji {
		font-size: 48rpx;
	}
	.category-name {
		font-size: 36rpx;
		font-weight: 700;
		color: #231C0B;
	}
}

.content-tabs {
	display: flex;
	padding: 8rpx 24rpx 24rpx;
	gap: 16rpx;

	.tab-item {
		flex: 1;
		text-align: center;
		padding: 16rpx 0;
		border-radius: 12rpx;
		background: #fff;
		font-size: 28rpx;
		color: #6E6242;

		&.active {
			background: #E0AE12;
			color: #231C0B;
		}
	}
}

.loading-state {
	display: flex;
	justify-content: center;
	padding-top: 120rpx;
}

.content-list {
	padding: 0 24rpx;

	.content-item {
		background: #fff;
		border-radius: 16rpx;
		margin-bottom: 16rpx;
		overflow: hidden;

		.item-header {
			display: flex;
			align-items: center;
			padding: 28rpx 28rpx;
			gap: 16rpx;

			.item-index {
				width: 48rpx;
				height: 48rpx;
				border-radius: 50%;
				background: rgba(224, 174, 18, 0.12);
				color: #C69200;
				font-size: 24rpx;
				font-weight: 600;
				display: flex;
				align-items: center;
				justify-content: center;
				flex-shrink: 0;
			}

			.item-title {
				flex: 1;
				font-size: 28rpx;
				font-weight: 500;
				color: #231C0B;
			}
		}

		.item-content {
			padding: 0 28rpx 28rpx 92rpx;
			font-size: 26rpx;
			color: #6E6242;
			line-height: 1.8;
		}
	}
}

.empty-state {
	text-align: center;
	padding-top: 120rpx;
	.empty-text {
		font-size: 28rpx;
		color: #9A8C67;
	}
}

.detail-page.dark-mode {
	background: #141109;

	.category-name,
	.content-list .content-item .item-header .item-title {
		color: #fff7e1;
	}

	.content-tabs .tab-item,
	.content-list .content-item {
		background: #1e180d;
	}

	.content-tabs .tab-item {
		color: #d7c89b;

		&.active {
			background: #E0AE12;
			color: #231c0b;
		}
	}

	.content-list .content-item .item-content,
	.empty-text {
		color: #9f926e;
	}
}
</style>
