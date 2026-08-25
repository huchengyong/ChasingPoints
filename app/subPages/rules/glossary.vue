<template>
	<view class="glossary-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 搜索栏 -->
		<view class="search-bar">
			<uni-icons type="search" size="18" color="#9A8C67"></uni-icons>
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
		<view v-else-if="filteredList.length > 0" class="term-list">
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

		<view v-else-if="glossaryPageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ glossaryPageError.title }}</text>
			<text class="error-text">{{ glossaryPageError.description }}</text>
			<button class="retry-btn" @tap="handleGlossaryErrorAction">{{ glossaryPageError.actionText }}</button>
		</view>

		<!-- 空状态 -->
		<view v-else-if="glossaryState.status === ASYNC_PAGE_STATUS.EMPTY || filterText" class="empty-state">
			<text class="empty-text">{{ filterText ? '未找到匹配术语' : '暂无术语数据' }}</text>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getGlossary } from '@/api/rules.js'
import { usePublicReadStore } from '@/store/publicRead.js'
import { getRuleCategoryLabel } from '@/utils/game-types.js'
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
const publicReadStore = usePublicReadStore()

const loading = ref(true)
const glossaryList = ref([])
const filterText = ref('')
const glossaryState = ref(createAsyncPageState({ data: [] }))
const glossaryPageError = computed(() => (
	glossaryState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(glossaryState.value.error, { resource: '术语词典' })
		: null
))

const filteredList = computed(() => {
	if (!filterText.value.trim()) return glossaryList.value
	const kw = filterText.value.trim().toLowerCase()
	return glossaryList.value.filter(
		item => (item.title && item.title.toLowerCase().includes(kw)) ||
				(item.content && item.content.toLowerCase().includes(kw))
	)
})

const loadGlossary = async () => {
	const nextState = beginAsyncPageLoad(glossaryState.value, { emptyData: [] })
	const pageRequest = getAsyncPageRequest(nextState)
	glossaryState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const res = await publicReadStore.loadStatic('rules:glossary', () => getGlossary())
		if (glossaryState.value.requestId !== pageRequest.requestId) return
		if (!res || res.success === false) throw createRequestError({ message: res?.message || '加载术语失败', category: 'business' })
		glossaryList.value = res.list || []
		glossaryState.value = resolveAsyncPageLoad(glossaryState.value, pageRequest, { data: glossaryList.value })
	} catch (e) {
		if (glossaryState.value.requestId !== pageRequest.requestId) return
		glossaryState.value = rejectAsyncPageLoad(glossaryState.value, pageRequest, e)
	} finally {
		if (glossaryState.value.requestId !== pageRequest.requestId) return
		loading.value = false
	}
}

const retryGlossary = () => loadGlossary()

const handleGlossaryErrorAction = () => {
	if (glossaryState.value.error?.category === 'permission' || glossaryState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryGlossary()
}

onLoad(() => {
	loadGlossary()
})
</script>

<style lang="scss" scoped>
.glossary-page {
	min-height: 100vh;
	background: #FAF8F2;
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
		color: #231C0B;
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
			color: #231C0B;
			display: block;
			margin-bottom: 12rpx;
		}

		.term-content {
			font-size: 26rpx;
			color: #6E6242;
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
		color: #9A8C67;
	}
}

.error-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 16rpx;
	padding-left: 32rpx;
	padding-right: 32rpx;
}

.error-text {
	font-size: 24rpx;
	line-height: 1.7;
	color: #6E6242;
}

.retry-btn {
	min-width: 200rpx;
	height: 76rpx;
	line-height: 76rpx;
	margin: 12rpx 0 0;
	padding: 0 32rpx;
	border-radius: 38rpx;
	background: #E0AE12;
	color: #ffffff;
	font-size: 28rpx;
	font-weight: 600;
}

.retry-btn::after {
	display: none;
}

.glossary-page.dark-mode {
	background: #141109;

	.search-bar,
	.term-list .term-item {
		background: #1e180d;
	}

	.search-input,
	.term-list .term-item .term-title {
		color: #fff7e1;
	}

	.term-list .term-item .term-content,
	.empty-text,
	.error-text {
		color: #9f926e;
	}

	.term-list .term-item .term-tag {
		background: rgba(224, 174, 18, 0.18);
		color: #f7e7a8;
	}
}
</style>
