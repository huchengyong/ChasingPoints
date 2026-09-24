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
				<view v-for="item in postList" :key="item.id" class="post-card">
					<view class="post-head">
						<view class="post-head-copy">
							<text class="post-time">{{ formatRelativeTime(item.created_at) }}</text>
							<view class="status-tag" :class="`status-${resolveMyPostReviewMeta(item.status, item.reject_reason).tagType}`">
								<text>{{ resolveMyPostReviewMeta(item.status, item.reject_reason).tagText }}</text>
							</view>
						</view>
						<view class="delete-btn" @tap="handleDelete(item)">
							<uni-icons type="trash" size="18" color="#94a3b8"></uni-icons>
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

			<view v-else class="empty-state">
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
import { formatRelativeTime } from '@/utils/format.js'
import { resolveMyPostReviewMeta } from '@/utils/social-review.js'

const { isDarkMode } = usePageTheme()

const loading = ref(true)
const refreshing = ref(false)
const postList = ref([])
const pageHint = ref('')

onLoad((options) => {
	pageHint.value = options.hint ? decodeURIComponent(options.hint) : ''
})

onShow(() => {
	loadPosts()
})

onPullDownRefresh(() => {
	refreshList()
})

const loadPosts = async () => {
	try {
		loading.value = true
		const res = await getMyPosts({ page: 1, page_size: 50 })
		postList.value = (res.list || []).map((item) => ({
			...item,
			images: Array.isArray(item.images) ? item.images : []
		}))
	} catch (error) {
		console.error('加载我的动态失败:', error)
		uni.showToast({ title: '加载失败', icon: 'none' })
	} finally {
		loading.value = false
		refreshing.value = false
		uni.stopPullDownRefresh()
	}
}

const refreshList = async () => {
	refreshing.value = true
	await loadPosts()
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
	background: #f8fafc;
	color: #0f172a;
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
	color: #64748b;
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
	border: 1px solid #e2e8f0;
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
	color: #64748b;
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
	background: #e2e8f0;

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
	color: #64748b;
}
</style>
