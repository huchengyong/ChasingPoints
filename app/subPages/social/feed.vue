<template>
	<view class="feed-page">
		<!-- 顶部 Tab -->
		<view class="feed-tabs">
			<view
				class="tab-item"
				:class="{ active: currentTab === 'following' }"
				@tap="switchTab('following')"
			>
				<text>好友</text>
			</view>
			<view
				class="tab-item"
				:class="{ active: currentTab === 'public' }"
				@tap="switchTab('public')"
			>
				<text>推荐</text>
			</view>
			<view
				class="tab-item"
				:class="{ active: currentTab === 'reports' }"
				@tap="switchTab('reports')"
			>
				<text>战报</text>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 动态列表 -->
		<scroll-view
			v-else
			scroll-y
			class="feed-scroll"
			@scrolltolower="loadMore"
			refresher-enabled
			:refresher-triggered="refreshing"
			@refresherrefresh="onRefresh"
		>
			<view v-if="postList.length > 0" class="post-list">
				<view
					v-for="item in postList"
					:key="item.id"
					class="post-card"
				>
					<!-- 用户信息 -->
					<view class="post-header">
						<image
							class="post-avatar"
							:src="item.avatar || '/static/images/default-avatar.png'"
							mode="aspectFill"
						/>
						<view class="post-user-info">
							<text class="post-nickname">{{ item.nickname || '球友' }}</text>
							<text class="post-time">{{ item.relativeTime }}</text>
						</view>
						<view v-if="item.is_mine" class="post-menu" @tap="handleDelete(item)">
							<uni-icons type="more-filled" size="20" color="#94a3b8"></uni-icons>
						</view>
					</view>

					<!-- 内容 -->
					<view class="post-content">
						<text class="post-text">{{ item.content }}</text>
					</view>

					<!-- 图片网格 -->
					<view v-if="item.images && item.images.length > 0" class="post-images">
						<image
							v-for="(img, idx) in item.images.slice(0, 9)"
							:key="idx"
							class="post-image"
							:class="{ 'single': item.images.length === 1 }"
							:src="img"
							mode="aspectFill"
							@tap="previewImage(item.images, idx)"
						/>
					</view>

					<!-- 类型标签 -->
					<view v-if="item.post_type === 1" class="post-tag match-tag">
						<uni-icons type="trophy" size="14" color="#f59e0b"></uni-icons>
						<text>战绩分享</text>
					</view>
					<view v-else-if="item.post_type === 2" class="post-tag checkin-tag">
						<uni-icons type="location" size="14" color="#22c55e"></uni-icons>
						<text>打卡</text>
					</view>

					<!-- 互动栏 -->
					<view class="post-actions">
						<view class="action-item" @tap="toggleLike(item)">
							<uni-icons
								:type="item.liked ? 'heart-filled' : 'heart'"
								size="20"
								:color="item.liked ? '#ef4444' : '#94a3b8'"
							></uni-icons>
							<text :class="{ liked: item.liked }">{{ item.likes_count || '' }}</text>
						</view>
						<view class="action-item" @tap="toggleComments(item)">
							<uni-icons type="chat" size="20" color="#94a3b8"></uni-icons>
							<text>{{ item.comments_count || '' }}</text>
						</view>
					</view>

					<view v-if="item.post_type === 1" class="post-report-actions">
						<view class="report-link" @tap="openPkReport(item)">
							<text>查看PK报表</text>
						</view>
					</view>

					<!-- 评论区 -->
					<view v-if="item.showComments" class="comment-section">
						<view v-if="item.commentLoading" class="comment-loading">
							<text class="comment-loading-text">加载评论...</text>
						</view>
						<view v-else>
							<view
								v-for="comment in (item.commentList || [])"
								:key="comment.id"
								class="comment-item"
							>
								<text class="comment-user">{{ comment.nickname || '球友' }}</text>
								<text class="comment-text">{{ comment.content }}</text>
							</view>
							<view v-if="!item.commentList || item.commentList.length === 0" class="no-comments">
								<text>暂无评论</text>
							</view>
						</view>
						<!-- 评论输入 -->
						<view class="comment-input-bar">
							<input
								class="comment-input"
								v-model="item.commentText"
								placeholder="写评论..."
								confirm-type="send"
								@confirm="submitComment(item)"
							/>
							<view class="comment-send" @tap="submitComment(item)">
								<text>发送</text>
							</view>
						</view>
					</view>
				</view>
			</view>

			<!-- 空状态 -->
			<view v-else class="empty-state">
				<text class="empty-icon">{{ currentTab === 'following' ? '👀' : currentTab === 'reports' ? '🏆' : '📝' }}</text>
				<text class="empty-title">{{ emptyTitle }}</text>
				<text class="empty-sub">去发布第一条动态吧！</text>
			</view>

			<!-- 加载更多指示 -->
			<view v-if="loadingMore" class="loading-more">
				<text class="loading-more-text">加载更多...</text>
			</view>
			<view v-if="!hasMore && postList.length > 0" class="no-more">
				<text class="no-more-text">— 没有更多了 —</text>
			</view>
		</scroll-view>

		<!-- 发布按钮 FAB -->
		<view class="fab-btn" @tap="goToCreate">
			<uni-icons type="plusempty" size="28" color="#fff"></uni-icons>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { getPostList, getPublicPosts, likePost, unlikePost, commentPost, deletePost, getPostComments } from '@/api/social.js'
import { formatRelativeTime } from '@/utils/format.js'

const currentTab = ref('public')
const loading = ref(true)
const refreshing = ref(false)
const loadingMore = ref(false)
const postList = ref([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const hasMore = ref(false)

const emptyTitle = computed(() => {
	if (currentTab.value === 'following') return '好友还没有动态'
	if (currentTab.value === 'reports') return '暂时还没有战报'
	return '推荐流还没有动态'
})

const switchTab = (tab) => {
	if (currentTab.value === tab) return
	currentTab.value = tab
	loadData(true)
}

const loadData = async (isRefresh = false) => {
	if (isRefresh) {
		page.value = 1
		loading.value = true
	}
	try {
		const api = currentTab.value === 'following' ? getPostList : getPublicPosts
		const res = await api({ page: page.value, page_size: pageSize })
		let list = (res.list || res || []).map(item => ({
			...item,
			images: parseImages(item.images),
			relativeTime: formatRelativeTime(item.created_at),
			showComments: false,
			commentList: [],
			commentText: '',
			commentLoading: false
		}))
		if (currentTab.value === 'reports') {
			list = list.filter(item => item.post_type === 1)
		}
		if (isRefresh) {
			postList.value = list
		} else {
			postList.value = [...postList.value, ...list]
		}
		total.value = res.total || postList.value.length
		hasMore.value = postList.value.length < total.value
	} catch (e) {
		console.error('加载动态失败:', e)
	} finally {
		loading.value = false
		refreshing.value = false
		loadingMore.value = false
	}
}

const parseImages = (images) => {
	if (!images) return []
	if (Array.isArray(images)) return images
	try { return JSON.parse(images) } catch (e) { return [] }
}

const onRefresh = () => {
	refreshing.value = true
	loadData(true)
}

const loadMore = () => {
	if (!hasMore.value || loadingMore.value) return
	loadingMore.value = true
	page.value++
	loadData(false)
}

const toggleLike = async (item) => {
	try {
		if (item.liked) {
			await unlikePost({ post_id: item.id })
			item.liked = false
			item.likes_count = Math.max((item.likes_count || 1) - 1, 0)
		} else {
			await likePost({ post_id: item.id })
			item.liked = true
			item.likes_count = (item.likes_count || 0) + 1
		}
	} catch (e) {
		console.error('点赞操作失败:', e)
	}
}

const toggleComments = async (item) => {
	item.showComments = !item.showComments
	if (item.showComments && (!item.commentList || item.commentList.length === 0)) {
		item.commentLoading = true
		try {
			const res = await getPostComments({ post_id: item.id, page: 1, page_size: 20 })
			item.commentList = res.list || res || []
		} catch (e) {
			console.error('加载评论失败:', e)
		} finally {
			item.commentLoading = false
		}
	}
}

const submitComment = async (item) => {
	const text = (item.commentText || '').trim()
	if (!text) {
		uni.showToast({ title: '请输入评论内容', icon: 'none' })
		return
	}
	try {
		const res = await commentPost({ post_id: item.id, content: text })
		if (res.success) {
			item.commentText = ''
			item.comments_count = (item.comments_count || 0) + 1
			// 重新加载评论
			const commRes = await getPostComments({ post_id: item.id, page: 1, page_size: 20 })
			item.commentList = commRes.list || commRes || []
			uni.showToast({ title: '评论成功', icon: 'success' })
		}
	} catch (e) {
		uni.showToast({ title: '评论失败', icon: 'none' })
	}
}

const handleDelete = (item) => {
	uni.showModal({
		title: '删除动态',
		content: '确定删除这条动态吗？',
		success: async (res) => {
			if (res.confirm) {
				try {
					await deletePost({ post_id: item.id })
					postList.value = postList.value.filter(p => p.id !== item.id)
					uni.showToast({ title: '已删除', icon: 'success' })
				} catch (e) {
					uni.showToast({ title: '删除失败', icon: 'none' })
				}
			}
		}
	})
}

const previewImage = (images, index) => {
	uni.previewImage({
		urls: images,
		current: index
	})
}

const goToCreate = () => {
	uni.navigateTo({ url: '/subPages/social/postCreate' })
}

const openPkReport = (item) => {
	const opponentId = item.opponent_id || item.target_id || item.user_id || 0
	const opponentName = item.opponent_name || item.nickname || ''
	const query = [`opponent_name=${encodeURIComponent(opponentName)}`]
	if (opponentId) {
		query.unshift(`opponent_id=${opponentId}`)
	}
	uni.navigateTo({ url: `/subPages/social/pkReport?${query.join('&')}` })
}

onLoad(() => {
	loadData(true)
})

onShow(() => {
	// 返回时刷新
	if (!loading.value) {
		loadData(true)
	}
})
</script>

<style lang="scss" scoped>
.feed-page {
	min-height: 100vh;
	background: #f1f5f9;
	display: flex;
	flex-direction: column;
}

.feed-tabs {
	display: flex;
	background: #fff;
	padding: 0 32rpx;
	border-bottom: 1rpx solid #e2e8f0;

	.tab-item {
		flex: 1;
		text-align: center;
		padding: 24rpx 0;
		font-size: 30rpx;
		color: #64748b;
		position: relative;

		&.active {
			color: #C69200;
			font-weight: 600;

			&::after {
				content: '';
				position: absolute;
				bottom: 0;
				left: 50%;
				transform: translateX(-50%);
				width: 48rpx;
				height: 6rpx;
				background: #E0AE12;
				border-radius: 3rpx;
			}
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

.feed-scroll {
	flex: 1;
	height: calc(100vh - 100rpx);
}

.post-list {
	padding: 20rpx 24rpx;
}

.post-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 28rpx;
	margin-bottom: 20rpx;

	.post-header {
		display: flex;
		align-items: center;
		margin-bottom: 20rpx;

		.post-avatar {
			width: 80rpx;
			height: 80rpx;
			border-radius: 50%;
			margin-right: 16rpx;
			flex-shrink: 0;
		}

		.post-user-info {
			flex: 1;
			.post-nickname {
				font-size: 28rpx;
				font-weight: 500;
				color: #1e293b;
			}
			.post-time {
				font-size: 22rpx;
				color: #94a3b8;
				margin-top: 4rpx;
			}
		}

		.post-menu {
			padding: 8rpx;
		}
	}

	.post-content {
		margin-bottom: 16rpx;
		.post-text {
			font-size: 28rpx;
			color: #334155;
			line-height: 1.6;
		}
	}

	.post-images {
		display: flex;
		flex-wrap: wrap;
		gap: 8rpx;
		margin-bottom: 16rpx;

		.post-image {
			width: calc(33.33% - 6rpx);
			aspect-ratio: 1;
			border-radius: 12rpx;

			&.single {
				width: 60%;
				aspect-ratio: 4/3;
			}
		}
	}

	.post-tag {
		display: inline-flex;
		align-items: center;
		gap: 6rpx;
		padding: 6rpx 16rpx;
		border-radius: 16rpx;
		font-size: 22rpx;
		margin-bottom: 16rpx;

		&.match-tag {
			background: #fef3c7;
			color: #d97706;
		}
		&.checkin-tag {
			background: #dcfce7;
			color: #16a34a;
		}
	}

	.post-actions {
		display: flex;
		border-top: 1rpx solid #f1f5f9;
		padding-top: 16rpx;
		gap: 40rpx;

		.action-item {
			display: flex;
			align-items: center;
			gap: 8rpx;

			text {
				font-size: 24rpx;
				color: #94a3b8;

				&.liked {
					color: #ef4444;
				}
			}
		}
	}

	.post-report-actions {
		margin-top: 16rpx;

		.report-link {
			display: inline-flex;
			align-items: center;
			padding: 10rpx 18rpx;
			border-radius: 999rpx;
			background: rgba(224, 174, 18, 0.12);

			text {
				font-size: 22rpx;
				color: #C69200;
			}
		}
	}

	.comment-section {
		margin-top: 16rpx;
		padding-top: 16rpx;
		border-top: 1rpx solid #f1f5f9;

		.comment-loading {
			padding: 16rpx 0;
			.comment-loading-text {
				font-size: 24rpx;
				color: #94a3b8;
			}
		}

		.comment-item {
			padding: 8rpx 0;
			.comment-user {
				font-size: 24rpx;
				font-weight: 500;
				color: #C69200;
				margin-right: 8rpx;
			}
			.comment-text {
				font-size: 24rpx;
				color: #475569;
			}
		}

		.no-comments {
			padding: 16rpx 0;
			text {
				font-size: 24rpx;
				color: #94a3b8;
			}
		}

		.comment-input-bar {
			display: flex;
			margin-top: 16rpx;
			gap: 12rpx;
			align-items: center;

			.comment-input {
				flex: 1;
				background: #f1f5f9;
				border-radius: 32rpx;
				padding: 12rpx 20rpx;
				font-size: 26rpx;
			}

			.comment-send {
				padding: 12rpx 24rpx;
				background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
				border-radius: 32rpx;
				text {
					font-size: 26rpx;
					color: #1f2937;
				}
			}
		}
	}
}

.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 200rpx;

	.empty-icon {
		font-size: 80rpx;
		margin-bottom: 24rpx;
	}
	.empty-title {
		font-size: 30rpx;
		font-weight: 500;
		color: #1e293b;
		margin-bottom: 12rpx;
	}
	.empty-sub {
		font-size: 26rpx;
		color: #94a3b8;
	}
}

.loading-more, .no-more {
	display: flex;
	justify-content: center;
	padding: 24rpx;
	.loading-more-text, .no-more-text {
		font-size: 24rpx;
		color: #94a3b8;
	}
}

.fab-btn {
	position: fixed;
	right: 40rpx;
	bottom: 120rpx;
	width: 100rpx;
	height: 100rpx;
	background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	box-shadow: 0 8rpx 24rpx rgba(224, 174, 18, 0.26);
}
</style>
