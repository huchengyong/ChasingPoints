<template>
	<view class="glossary-page">
		<!-- 搜索栏 -->
		<view class="search-bar">
			<uni-icons type="search" size="18" color="#94a3b8"></uni-icons>
			<input
				class="search-input"
				v-model="filterText"
				placeholder="搜索术语..."
			/>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
		</view>

		<!-- 术语列表 -->
		<view v-else class="term-list">
			<view
				v-for="(item, index) in filteredList"
				:key="item.id || index"
				class="term-item"
			>
				<text class="term-title">{{ item.title }}</text>
				<text class="term-content">{{ item.content }}</text>
				<text v-if="item.category" class="term-tag">{{ getRuleCategoryLabel(item.category) }}</text>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-if="!loading && filteredList.length === 0" class="empty-state">
			<text class="empty-text">{{ filterText ? '未找到匹配术语' : '暂无术语数据' }}</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getGlossary } from '@/api/rules.js'
import { getRuleCategoryLabel } from '@/utils/game-types.js'

const loading = ref(true)
const glossaryList = ref([])
const filterText = ref('')

const filteredList = computed(() => {
	if (!filterText.value.trim()) return glossaryList.value
	const kw = filterText.value.trim().toLowerCase()
	return glossaryList.value.filter(
		item => (item.title && item.title.toLowerCase().includes(kw)) ||
				(item.content && item.content.toLowerCase().includes(kw))
	)
})

const loadGlossary = async () => {
	loading.value = true
	try {
		const res = await getGlossary()
		glossaryList.value = res.list || res || []
	} catch (e) {
		console.error('加载术语失败:', e)
	} finally {
		loading.value = false
	}
}

onLoad(() => {
	loadGlossary()
})
</script>

<style lang="scss" scoped>
.glossary-page {
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
	margin-bottom: 24rpx;
	gap: 12rpx;

	.search-input {
		flex: 1;
		font-size: 28rpx;
		color: #1e293b;
	}
}

.loading-state {
	display: flex;
	justify-content: center;
	padding-top: 200rpx;
}

.term-list {
	.term-item {
		background: #fff;
		border-radius: 16rpx;
		padding: 28rpx;
		margin-bottom: 16rpx;

		.term-title {
			font-size: 30rpx;
			font-weight: 600;
			color: #1e293b;
			display: block;
			margin-bottom: 12rpx;
		}

		.term-content {
			font-size: 26rpx;
			color: #64748b;
			line-height: 1.7;
			display: block;
			margin-bottom: 12rpx;
		}

		.term-tag {
			font-size: 22rpx;
			background: rgba(224, 174, 18, 0.12);
			color: #C69200;
			padding: 4rpx 16rpx;
			border-radius: 12rpx;
		}
	}
}

.empty-state {
	text-align: center;
	padding-top: 200rpx;
	.empty-text {
		font-size: 28rpx;
		color: #94a3b8;
	}
}
</style>
