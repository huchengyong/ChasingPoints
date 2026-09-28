<template>
	<view class="my-posts-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="pageHint" class="page-hint">
			<text>{{ pageHint }}</text>
		</view>

		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="32" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<scroll-view
			v-else
			scroll-y
			class="post-scroll"
			refresher-enabled
			:refresher-triggered="refreshing"
			@refresherrefresh="refreshList"
		>
			<view v-if="postList.length" class="post-list">
				<view v-if="postRefreshError" class="refresh-error-banner">
					<text>{{ postRefreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryPosts">重试</text>
				</view>
				<view v-for="item in postList" :key="item.id" class="post-card">
					<view class="post-head">
						<view class="post-head-copy">
							<text class="post-time">{{ formatRelativeTime(item.created_at) }}</text>
							<view class="status-tag" :class="`status-${resolveMyPostReviewMeta(item.status, item.reject_reason).tagType}`">
								<text>{{ resolveMyPostReviewMeta(item.status, item.reject_reason).tagText }}</text>
							</view>
						</view>
						<view class="delete-btn" @tap="handleDelete(item)">
							<uni-icons type="trash" size="18" color="#9A8C67"></uni-icons>
						</view>
					</view>

					<text class="post-content">{{ item.content || '暂无内容' }}</text>

					<view v-if="item.images?.length" class="image-grid">
						<image
							v-for="(img, idx) in item.images.slice(0, 9)"
							:key="`${item.id}-${idx}`"
							class="post-image"
							:class="{ single: item.images.length === 1 }"
							:src="img"
							mode="aspectFill"
							@tap="previewImage(item.images, idx)"
						/>
					</view>

					<view
						v-if="resolveMyPostReviewMeta(item.status, item.reject_reason).reasonVisible"
						class="reject-reason"
					>
						<text class="reject-title">拒绝原因</text>
						<text class="reject-text">{{ resolveMyPostReviewMeta(item.status, item.reject_reason).reasonText }}</text>
					</view>

					<view class="post-meta">
						<text>{{ formatPostType(item.post_type) }}</text>
						<text>点赞 {{ item.likes_count || 0 }}</text>
						<text>评论 {{ item.comments_count || 0 }}</text>
					</view>
				</view>
			</view>

			<view v-else-if="postPageError" class="empty-state error-state">
				<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
				<text class="empty-title">{{ postPageError.title }}</text>
				<text class="empty-desc">{{ postPageError.description }}</text>
				<button class="retry-btn" @tap="handlePostsErrorAction">{{ postPageError.actionText }}</button>
			</view>

			<view v-else-if="postState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
				<text class="empty-icon">📝</text>
				<text class="empty-title">你还没有动态</text>
				<text class="empty-desc">发布后会在这里显示审核进度和发布记录。</text>
				<button class="empty-btn" @tap="goCreatePost">
					<text>去发动态</text>
				</button>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { deletePost, getMyPosts } from '@/api/social.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { formatRelativeTime } from '@/utils/format.js'
import { resolveMyPostReviewMeta } from '@/utils/social-review.js'
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
const refreshing = ref(false)
const postList = ref([])
const pageHint = ref('')
const hasLoadedOnce = ref(false)
const postState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const postPageError = computed(() => (
	postState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(postState.value.error, { resource: '我的动态' })
		: null
))
const postRefreshError = computed(() => (
	postState.value.refreshError
		? resolveAsyncPageErrorFeedback(postState.value.refreshError, { resource: '我的动态' })
		: null
))

const currentIdentityKey = () => `${userStore.userId}:${userStore.authGeneration}`

onLoad((options) => {
	pageHint.value = options.hint ? decodeURIComponent(options.hint) : ''
})

onShow(() => {
	if (!hasLoadedOnce.value || postState.value.authGeneration !== userStore.authGeneration) loadPosts()
})

onPullDownRefresh(() => {
	refreshList()
})

const loadPosts = async ({ force = false } = {}) => {
	if (hasLoadedOnce.value && !force) return
	const requestIdentityKey = currentIdentityKey()
	if (postState.value.authGeneration !== userStore.authGeneration) {
		postList.value = []
		hasLoadedOnce.value = false
		postState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
	}
	const nextState = beginAsyncPageLoad(postState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	postState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const res = await getMyPosts({ page: 1, page_size: 50 })
		if (currentIdentityKey() !== requestIdentityKey || postState.value.requestId !== pageRequest.requestId || postState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res || res.success === false) throw createRequestError({ message: res?.message || '加载我的动态失败', category: 'business' })
		postList.value = (res.list || []).map((item) => ({
			...item,
			images: Array.isArray(item.images) ? item.images : []
		}))
		hasLoadedOnce.value = true
		postState.value = resolveAsyncPageLoad(postState.value, pageRequest, { data: postList.value })
	} catch (error) {
		if (currentIdentityKey() !== requestIdentityKey || postState.value.requestId !== pageRequest.requestId || postState.value.authGeneration !== pageRequest.authGeneration) return
		postState.value = rejectAsyncPageLoad(postState.value, pageRequest, error)
	} finally {
		if (currentIdentityKey() !== requestIdentityKey || postState.value.requestId !== pageRequest.requestId || postState.value.authGeneration !== pageRequest.authGeneration) return
		loading.value = false
		refreshing.value = false
		uni.stopPullDownRefresh()
	}
}

const refreshList = async () => {
	if (postState.value.status === ASYNC_PAGE_STATUS.LOADING || postState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	refreshing.value = true
	await loadPosts({ force: true })
}

const retryPosts = () => loadPosts({ force: true })

const handlePostsErrorAction = () => {
	if (postState.value.error?.category === 'permission' || postState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryPosts()
}

const previewImage = (images, current) => {
	if (!images?.length) return
	uni.previewImage({
		urls: images,
		current
	})
}

const formatPostType = (postType) => {
	if (postType === 1) return '战绩分享'
	if (postType === 2) return '打卡动态'
	return '日常动态'
}

const goCreatePost = () => {
	uni.navigateTo({ url: '/subPages/social/postCreate' })
}

const handleDelete = (item) => {
	uni.showModal({
		title: '删除动态',
		content: '删除后不可恢复，确认删除这条动态吗？',
		success: async ({ confirm }) => {
			if (!confirm) return
			try {
				const res = await deletePost({ post_id: item.id })
				if (res.success) {
					uni.showToast({ title: '删除成功', icon: 'success' })
					postList.value = postList.value.filter((post) => post.id !== item.id)
					postState.value = {
						...postState.value,
						status: postList.value.length ? ASYNC_PAGE_STATUS.READY : ASYNC_PAGE_STATUS.EMPTY,
						data: postList.value,
						hasData: postList.value.length > 0,
						refreshError: null
					}
				} else {
					uni.showToast({ title: res.message || '删除失败', icon: 'none' })
				}
			} catch (error) {
				console.error('删除我的动态失败:', error)
				uni.showToast({ title: '删除失败', icon: 'none' })
			}
		}
	})
}
</script>

<style lang="scss" scoped>
.my-posts-page {
	min-height: 100vh;
	background: #FAF8F2;
	color: #231C0B;
	padding: 24rpx;
	box-sizing: border-box;

	&.dark-mode {
		background: #141109;
		color: #fff7e1;

		.page-hint,
		.post-card,
		.empty-state {
			background: #1e180d;
			border-color: #3a2e16;
		}

		.post-time,
		.post-meta,
		.reject-title,
		.loading-text,
		.empty-desc {
			color: #d7c89b;
		}

		.reject-reason {
			background: rgba(190, 24, 93, 0.14);
			border-color: rgba(251, 113, 133, 0.22);
		}
	}
}

.page-hint {
	background: #fff7e0;
	border: 1px solid #f5d68f;
	border-radius: 20rpx;
	padding: 18rpx 20rpx;
	margin-bottom: 20rpx;
	font-size: 24rpx;
	color: #7c5b00;
}

.loading-state,
.empty-state {
	background: #ffffff;
	border-radius: 24rpx;
	padding: 48rpx 32rpx;
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 16rpx;
}

.loading-text,
.empty-desc {
	font-size: 24rpx;
	color: #6E6242;
}

.refresh-error-banner {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 20rpx;
	margin-bottom: 20rpx;
	padding: 18rpx 22rpx;
	border-radius: 14rpx;
	background: rgba(224, 174, 18, 0.12);
	color: #8a5b00;
	font-size: 24rpx;
}

.refresh-error-action {
	flex-shrink: 0;
	color: #a86f00;
	font-weight: 600;
}

.retry-btn {
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

.retry-btn::after {
	display: none;
}

.empty-icon {
	font-size: 60rpx;
}

.empty-title {
	font-size: 32rpx;
	font-weight: 600;
}

.empty-btn {
	width: 100%;
	margin-top: 12rpx;
	background: #E0AE12;
	color: #ffffff;
	border-radius: 999rpx;

	&::after {
		display: none;
	}

	&[disabled] {
		opacity: 0.6;
	}
}

.post-scroll {
	height: calc(100vh - 48rpx);
}

.post-list {
	display: flex;
	flex-direction: column;
	gap: 20rpx;
	padding-bottom: 40rpx;
}

.post-card {
	background: #ffffff;
	border: 1px solid #E9E2CF;
	border-radius: 24rpx;
	padding: 24rpx;
}

.post-head {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16rpx;
	margin-bottom: 16rpx;
}

.post-head-copy {
	display: flex;
	flex-direction: column;
	gap: 12rpx;
}

.post-time {
	font-size: 22rpx;
	color: #6E6242;
}

.status-tag {
	display: inline-flex;
	align-items: center;
	font-size: 22rpx;
	font-weight: 500;
}

.status-pending {
	background: rgba(224, 174, 18, 0.14);
	color: #a16207;
}

.status-rejected {
	background: rgba(244, 63, 94, 0.14);
	color: #be123c;
}

.status-published {
	background: rgba(34, 197, 94, 0.12);
	color: #15803d;
}

.post-content {
	display: block;
	font-size: 30rpx;
	line-height: 1.7;
	margin-bottom: 18rpx;
}

.image-grid {
	display: flex;
	flex-wrap: wrap;
	gap: 12rpx;
	margin-bottom: 18rpx;
}

.post-image {
	width: calc((100% - 24rpx) / 3);
	height: 200rpx;
	border-radius: 18rpx;
	background: #E9E2CF;

	&.single {
		width: 100%;
		height: 360rpx;
	}
}

.reject-reason {
	display: flex;
	flex-direction: column;
	gap: 8rpx;
	margin-bottom: 18rpx;
	padding: 16rpx 18rpx;
	border-radius: 18rpx;
	background: #fff1f2;
	border: 1px solid #fecdd3;
}

.reject-title {
	font-size: 22rpx;
	color: #9f1239;
	font-weight: 600;
}

.reject-text {
	font-size: 24rpx;
	line-height: 1.6;
	color: #be123c;
}

.post-meta {
	display: flex;
	gap: 20rpx;
	font-size: 22rpx;
	color: #6E6242;
}
</style>
