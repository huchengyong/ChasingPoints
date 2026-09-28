<template>
	<view class="feed-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 顶部 Tab -->
		<view class="feed-tabs">
			<view
				v-for="tab in tabs"
				:key="tab.value"
				class="tab-item"
				:class="{ active: currentTab === tab.value }"
				@tap="switchTab(tab.value)"
			>
				<text>{{ tab.label }}</text>
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
				<view v-if="feedRefreshError" class="refresh-error-banner">
					<text>{{ feedRefreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryFeed">重试</text>
				</view>
				<view
					v-for="item in postList"
					:key="item.id"
					class="post-card"
				>
					<!-- 用户信息 -->
					<view class="post-header">
						<image
							class="post-avatar"
							:src="resolveAvatarUrl(item.avatar, item.user_id)"
							mode="aspectFill"
						/>
						<view class="post-user-info">
							<text class="post-nickname">{{ item.nickname || '球友' }}</text>
							<text class="post-time">{{ item.relativeTime }}</text>
						</view>
						<view v-if="item.is_mine" class="post-menu" @tap="handleDelete(item)">
							<uni-icons type="more-filled" size="20" color="#9A8C67"></uni-icons>
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
								:color="item.liked ? '#ef4444' : '#9A8C67'"
							></uni-icons>
							<text :class="{ liked: item.liked }">{{ formatActionCount(item.likes_count, '点赞') }}</text>
						</view>
						<view class="action-item" @tap="toggleComments(item)">
							<uni-icons type="chat" size="20" color="#9A8C67"></uni-icons>
							<text>{{ formatActionCount(item.comments_count, '评论') }}</text>
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

			<view v-else-if="feedPageError" class="empty-state error-state">
				<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
				<text class="empty-title">{{ feedPageError.title }}</text>
				<text class="empty-sub">{{ feedPageError.description }}</text>
				<button class="error-action" @tap="handleFeedErrorAction">{{ feedPageError.actionText }}</button>
			</view>

			<!-- 空状态 -->
			<view v-else-if="feedState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
				<text class="empty-icon">{{ emptyState.icon }}</text>
				<text class="empty-title">{{ emptyState.title }}</text>
				<text class="empty-sub">{{ emptyState.desc }}</text>
				<view v-if="emptyState.ctaText" class="empty-cta" @tap="handleEmptyStateCta">
					<text>{{ emptyState.ctaText }}</text>
				</view>
			</view>

			<!-- 加载更多指示 -->
			<view v-if="feedState.status !== ASYNC_PAGE_STATUS.ERROR && loadingMore" class="loading-more">
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
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { formatRelativeTime } from '@/utils/format.js'
import { filterReportPosts, resolveFeedTab, resolveSocialEmptyState, SOCIAL_TABS } from '@/utils/social-entry.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
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

const tabs = SOCIAL_TABS
const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

const currentTab = ref('recommend')
const loading = ref(true)
const refreshing = ref(false)
const loadingMore = ref(false)
const postList = ref([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const hasMore = ref(false)
const shouldRefreshOnShow = ref(false)
const feedState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))

const emptyState = computed(() => resolveSocialEmptyState({
	tab: currentTab.value,
	isLoggedIn: userStore.isLoggedIn
}))
const feedPageError = computed(() => (
	feedState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(feedState.value.error, { resource: '动态' })
		: null
))
const feedRefreshError = computed(() => (
	feedState.value.refreshError
		? resolveAsyncPageErrorFeedback(feedState.value.refreshError, { resource: '动态' })
		: null
))

const currentIdentityKey = () => `${userStore.userId}:${userStore.authGeneration}`

const switchTab = (tab) => {
	if (currentTab.value === tab) return
	currentTab.value = tab
	postList.value = []
	page.value = 1
	total.value = 0
	hasMore.value = false
	feedState.value = {
		...feedState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: feedState.value.requestId + 1,
		data: [],
		hasData: false,
		error: null,
		refreshError: null
	}
	loadData(true)
}

const loadData = async (isRefresh = false) => {
	if (feedState.value.status === ASYNC_PAGE_STATUS.LOADING || feedState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const requestIdentityKey = currentIdentityKey()
	const previousPage = page.value
	if (feedState.value.authGeneration !== userStore.authGeneration) {
		postList.value = []
		page.value = 1
		total.value = 0
		hasMore.value = false
		feedState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
	}
	if (isRefresh) {
		page.value = 1
	}
	const nextState = beginAsyncPageLoad(feedState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	feedState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const { list, total: resolvedTotal } = await loadPostsByTab(currentTab.value)
		if (currentIdentityKey() !== requestIdentityKey || feedState.value.requestId !== pageRequest.requestId || feedState.value.authGeneration !== pageRequest.authGeneration) return
		if (isRefresh) {
			postList.value = list
		} else {
			postList.value = [...postList.value, ...list]
		}
		total.value = resolvedTotal
		hasMore.value = postList.value.length < total.value
		feedState.value = resolveAsyncPageLoad(feedState.value, pageRequest, { data: postList.value })
	} catch (e) {
		if (currentIdentityKey() !== requestIdentityKey || feedState.value.requestId !== pageRequest.requestId || feedState.value.authGeneration !== pageRequest.authGeneration) return
		if (!isRefresh && previousPage > 1) {
			page.value = previousPage - 1
		}
		feedState.value = rejectAsyncPageLoad(feedState.value, pageRequest, e)
	} finally {
		if (currentIdentityKey() !== requestIdentityKey || feedState.value.requestId !== pageRequest.requestId || feedState.value.authGeneration !== pageRequest.authGeneration) return
		loading.value = false
		refreshing.value = false
		loadingMore.value = false
	}
}

const normalizePost = (item) => ({
	...item,
	liked: Boolean(item.liked ?? item.is_liked),
	images: parseImages(item.images),
	relativeTime: formatRelativeTime(item.created_at),
	showComments: false,
	commentList: [],
	commentText: '',
	commentLoading: false
})

const ensurePostListResponse = (res, message) => {
	if (!res || res.success === false) {
		throw createRequestError({ message: res?.message || message, category: 'business' })
	}
	return res
}

const loadPostsByTab = async (tab) => {
	if (tab === 'reports') {
		const requests = [
			getPublicPosts({ page: page.value, page_size: pageSize })
		]

		if (userStore.isLoggedIn) {
			requests.push(getPostList({ page: page.value, page_size: pageSize }))
		}

		const responses = await Promise.all(requests)
		const verifiedResponses = responses.map((res) => ensurePostListResponse(res, '加载战报动态失败'))
		const list = filterReportPosts(
			verifiedResponses.flatMap((res) => (res.list || []).map(normalizePost))
		).sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0))
		return {
			list,
			total: list.length
		}
	}

	const feedTab = resolveFeedTab(tab)
	if (feedTab === 'following' && !userStore.isLoggedIn) {
		return {
			list: [],
			total: 0
		}
	}

	const api = feedTab === 'following' ? getPostList : getPublicPosts
	const res = ensurePostListResponse(await api({ page: page.value, page_size: pageSize }), '加载动态失败')
	return {
		list: (res.list || []).map(normalizePost),
		total: res.total || (res.list || []).length
	}
}

const parseImages = (images) => {
	if (!images) return []
	if (Array.isArray(images)) return images
	try { return JSON.parse(images) } catch (e) { return [] }
}

const onRefresh = () => {
	if (feedState.value.status === ASYNC_PAGE_STATUS.LOADING || feedState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	refreshing.value = true
	loadData(true)
}

const loadMore = () => {
	if (!hasMore.value || loadingMore.value || currentTab.value === 'reports' || feedState.value.status === ASYNC_PAGE_STATUS.LOADING || feedState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	loadingMore.value = true
	page.value++
	loadData(false)
}

const retryFeed = () => loadData(true)

const handleFeedErrorAction = () => {
	if (feedState.value.error?.category === 'permission' || feedState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryFeed()
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
	shouldRefreshOnShow.value = true
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

const formatActionCount = (value, fallbackText) => {
	return value > 0 ? value : fallbackText
}

const handleEmptyStateCta = () => {
	if (currentTab.value === 'friends' && !userStore.isLoggedIn) {
		uni.navigateTo({ url: '/pages/login/login' })
	}
}

onLoad((options) => {
	const requestedTab = tabs.find((item) => item.value === options?.tab)?.value
	if (requestedTab) {
		currentTab.value = requestedTab
	}
	loadData(true)
})

onShow(() => {
	if (!loading.value && shouldRefreshOnShow.value) {
		shouldRefreshOnShow.value = false
		loadData(true)
	}
})
</script>

<style lang="scss" scoped>
.feed-page {
	min-height: 100vh;
	background: #FAF8F2;
	display: flex;
	flex-direction: column;

	&.dark-mode {
		background: #141109;

		.feed-tabs,
		.post-card {
			background: #1e180d;
			border-color: #3a2e16;
		}

		.tab-item,
		.post-time,
		.loading-text,
		.empty-sub,
		.loading-more-text,
		.no-more-text,
		.comment-text,
		.no-comments text,
		.comment-loading-text,
		.action-item text {
			color: #d7c89b;
		}

		.post-nickname,
		.post-text,
		.empty-title {
			color: #fff7e1;
		}

		.comment-input {
			background: #2b2316 !important;
			color: #fff7e1;
		}
	}
}

.feed-tabs {
	display: flex;
	background: #fff;
	padding: 0 32rpx;
	border-bottom: 1rpx solid #E9E2CF;

	.tab-item {
		flex: 1;
		text-align: center;
		padding: 24rpx 0;
		font-size: 30rpx;
		color: #6E6242;
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
		color: #9A8C67;
	}
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

.error-action {
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

.error-action::after {
	display: none;
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
	border: 1rpx solid #E9E2CF;
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
				color: #231C0B;
			}
			.post-time {
				font-size: 22rpx;
				color: #9A8C67;
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
			color: #6E6242;
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
		border-top: 1rpx solid #FAF8F2;
		padding-top: 16rpx;
		gap: 40rpx;

		.action-item {
			display: flex;
			align-items: center;
			gap: 8rpx;

			text {
				font-size: 24rpx;
				color: #9A8C67;

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

			text {
				font-size: 22rpx;
				font-weight: 500;
				color: #C69200;
			}
		}
	}

	.comment-section {
		margin-top: 16rpx;
		padding-top: 16rpx;
		border-top: 1rpx solid #FAF8F2;

		.comment-loading {
			padding: 16rpx 0;
			.comment-loading-text {
				font-size: 24rpx;
				color: #9A8C67;
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
				color: #6E6242;
			}
		}

		.no-comments {
			padding: 16rpx 0;
			text {
				font-size: 24rpx;
				color: #9A8C67;
			}
		}

		.comment-input-bar {
			display: flex;
			margin-top: 16rpx;
			gap: 12rpx;
			align-items: center;

			.comment-input {
				flex: 1;
				background: #FAF8F2;
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
					color: #231C0B;
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
		color: #231C0B;
		margin-bottom: 12rpx;
	}
	.empty-sub {
		font-size: 26rpx;
		color: #9A8C67;
	}
}

.empty-cta {
	margin-top: 20rpx;
	padding: 16rpx 32rpx;
	border-radius: 999rpx;
	background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);

	text {
		font-size: 24rpx;
		color: #ffffff;
	}
}

.loading-more, .no-more {
	display: flex;
	justify-content: center;
	padding: 24rpx;
	.loading-more-text, .no-more-text {
		font-size: 24rpx;
		color: #9A8C67;
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
