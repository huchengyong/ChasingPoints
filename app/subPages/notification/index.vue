<template>
	<view class="notification-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 顶部操作栏 -->
		<view class="top-bar">
			<text class="page-subtitle">全部消息</text>
			<text class="read-all-btn" @tap="handleReadAll">全部已读</text>
		</view>

		<!-- 通知列表 -->
		<view v-if="notificationList.length > 0" class="notification-list">
			<view v-if="notificationRefreshError" class="refresh-error-banner">
				<text>{{ notificationRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryNotifications">重试</text>
			</view>
			<view
				v-for="item in notificationList"
				:key="item.id"
				class="notification-item"
				:class="{ unread: !item.is_read }"
				@tap="handleTapNotification(item)"
			>
				<view class="item-icon" :class="getTypeClass(item.type)">
					<text class="icon-text">{{ getTypeIcon(item.type) }}</text>
				</view>
				<view class="item-content">
					<view class="item-header">
						<text class="item-title">{{ item.title }}</text>
						<view v-if="!item.is_read" class="unread-dot"></view>
					</view>
					<text class="item-body">{{ item.content }}</text>
					<text class="item-time">{{ item.created_at }}</text>
				</view>
				<view class="item-delete" @tap.stop="handleDelete(item.id)">
					<uni-icons type="trash" size="18" color="#9A8C67"></uni-icons>
				</view>
			</view>
		</view>

		<view v-else-if="notificationPageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ notificationPageError.title }}</text>
			<text class="empty-hint">{{ notificationPageError.description }}</text>
			<button class="error-action" @tap="handleNotificationErrorAction">{{ notificationPageError.actionText }}</button>
		</view>

		<!-- 空状态 -->
		<view v-else-if="notificationState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
			<text class="empty-icon">🔔</text>
			<text class="empty-text">暂无消息</text>
			<text class="empty-hint">有新消息时会在这里通知你</text>
		</view>

		<!-- 加载中 -->
		<view v-else-if="loading" class="loading-state">
				<uni-icons type="spinner-cycle" size="28" color="#E0AE12"></uni-icons>
		</view>

		<!-- 加载更多 -->
		<view v-if="notificationState.status !== ASYNC_PAGE_STATUS.ERROR && !loading && hasMore" class="load-more" @tap="loadMore">
			<text>加载更多</text>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { getNotificationList, markAsRead, markAllAsRead, deleteNotification } from '@/api/notification.js'
import { useNotificationStore } from '@/store/notification.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { useUserStore } from '@/store/user.js'
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

const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()
const userStore = useUserStore()
const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})
const notificationList = ref([])
const loading = ref(false)
const notificationState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const hasMore = ref(true)
const page = ref(1)
const pageSize = 20
const latestRequestId = ref(0)
const requestedIdentityKey = ref('')
const loadedIdentityKey = ref('')
const notificationPageError = computed(() => (
	notificationState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(notificationState.value.error, { resource: '消息中心' })
		: null
))
const notificationRefreshError = computed(() => (
	notificationState.value.refreshError
		? resolveAsyncPageErrorFeedback(notificationState.value.refreshError, { resource: '消息中心' })
		: null
))

const currentIdentityKey = () => {
	const identity = getReadIdentity()
	return `${identity.userId}:${identity.authGeneration}`
}

const getTypeIcon = (type) => {
	const map = {
		challenge: '🎯',
		tournament: '🏆',
		friend_request: '👥',
		follow: '⭐',
		match_result: '🏁',
			rank_change: '📊',
			season_rollover: '🔄',
			system: '📢'
	}
	return map[type] || '🔔'
}

const getTypeClass = (type) => {
	return 'type-' + (type || 'system')
}

const parseNotificationData = (data) => {
	if (!data) return {}
	if (typeof data === 'object') return data
	try {
		return JSON.parse(data)
	} catch (error) {
		console.error('解析通知数据失败:', error)
		return {}
	}
}

const navigateByNotification = (item) => {
	const data = parseNotificationData(item.data)
	const target = data.target || data.url || ''

	if (target === '/subPages/match/matchResult' && data.match_id) {
		uni.navigateTo({ url: `${target}?match_id=${data.match_id}&from=history` })
		return
	}

	if (target) {
		uni.navigateTo({ url: target })
		return
	}

	if (item.type === 'friend_request') {
		uni.navigateTo({ url: '/subPages/social/friendRequests' })
	}
}

const handleTapNotification = async (item) => {
	const requestIdentityKey = currentIdentityKey()
	if (!item.is_read) {
		try {
			await markAsRead({ notification_id: item.id })
			if (currentIdentityKey() !== requestIdentityKey) return
			item.is_read = true
			notificationStore.decrementUnread(getReadIdentity())
			if (item.type === 'friend_request' && item.title === '收到好友申请') {
				friendRequestStore.fetchPendingCount(getReadIdentity())
			}
		} catch (e) {
			console.error('标记已读失败:', e)
		}
	}

	navigateByNotification(item)
}

const handleReadAll = async () => {
	const requestIdentityKey = currentIdentityKey()
	try {
		await markAllAsRead()
		if (currentIdentityKey() !== requestIdentityKey) return
		notificationList.value.forEach(item => {
			item.is_read = true
		})
		notificationStore.clearUnread(getReadIdentity())
		uni.showToast({ title: '已全部标记为已读', icon: 'success' })
	} catch (e) {
		console.error('全部已读失败:', e)
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const handleDelete = async (id) => {
	const requestIdentityKey = currentIdentityKey()
	try {
		await deleteNotification({ notification_id: id })
		if (currentIdentityKey() !== requestIdentityKey) return
		const idx = notificationList.value.findIndex(n => n.id === id)
		if (idx >= 0) {
			const item = notificationList.value[idx]
			if (!item.is_read) {
				notificationStore.decrementUnread(getReadIdentity())
			}
			notificationList.value.splice(idx, 1)
		}
		uni.showToast({ title: '已删除', icon: 'success' })
	} catch (e) {
		console.error('删除通知失败:', e)
		uni.showToast({ title: '删除失败', icon: 'none' })
	}
}

const loadNotifications = async (isRefresh = false) => {
	if (loading.value) return
	const requestId = latestRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	const nextState = beginAsyncPageLoad(notificationState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	latestRequestId.value = requestId
	requestedIdentityKey.value = requestIdentityKey
	notificationState.value = nextState
	loading.value = true

	if (isRefresh) {
		page.value = 1
		hasMore.value = true
	}

	try {
		const res = await getNotificationList({ page: page.value, page_size: pageSize })
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey || notificationState.value.requestId !== pageRequest.requestId || notificationState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '加载消息失败', category: 'business' })
		const list = res.list || []
		if (Object.prototype.hasOwnProperty.call(res || {}, 'unread_count')) {
			notificationStore.setUnreadCount(res.unread_count, getReadIdentity())
		}

		if (isRefresh) {
			notificationList.value = list
		} else {
			notificationList.value = [...notificationList.value, ...list]
		}

		hasMore.value = list.length >= pageSize
		loadedIdentityKey.value = requestIdentityKey
		notificationState.value = resolveAsyncPageLoad(notificationState.value, pageRequest, { data: notificationList.value })
	} catch (e) {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		requestedIdentityKey.value = ''
		notificationState.value = rejectAsyncPageLoad(notificationState.value, pageRequest, e)
	} finally {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		loading.value = false
		if (isRefresh) {
			uni.stopPullDownRefresh()
		}
	}
}

const loadMore = () => {
	if (!hasMore.value) return
	page.value++
	loadNotifications()
}

const retryNotifications = () => loadNotifications(true)

const handleNotificationErrorAction = () => {
	if (notificationState.value.error?.category === 'permission' || notificationState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryNotifications()
}

onLoad(() => {
	loadNotifications(true)
})

onShow(() => {
	if (loadedIdentityKey.value !== currentIdentityKey() && requestedIdentityKey.value !== currentIdentityKey()) {
		notificationList.value = []
		page.value = 1
		loading.value = false
		notificationState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
		loadNotifications(true)
	}
})

onPullDownRefresh(() => {
	loadNotifications(true)
})
</script>

<style lang="scss" scoped>
.notification-page {
	min-height: 100vh;
	background: #FAF8F2;
}

.top-bar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 24rpx 32rpx;
	background: #fff;
	border-bottom: 1rpx solid #FAF8F2;

	.page-subtitle {
		font-size: 28rpx;
		color: #6E6242;
	}
		.read-all-btn {
			font-size: 26rpx;
			color: #c69200;
		}
}

.notification-list {
	padding: 16rpx;

	.notification-item {
		display: flex;
		align-items: flex-start;
		background: #fff;
		border-radius: 16rpx;
		padding: 24rpx;
		margin-bottom: 12rpx;
		gap: 16rpx;

			&.unread {
				background: #FAF8F2;
				border-left: 6rpx solid #e0ae12;
			}

		.item-icon {
			width: 64rpx;
			height: 64rpx;
			border-radius: 50%;
			display: flex;
			align-items: center;
			justify-content: center;
			flex-shrink: 0;

			&.type-challenge { background: rgba(224, 174, 18, 0.14); }
			&.type-tournament { background: #fef3c7; }
			&.type-follow { background: rgba(251, 191, 36, 0.16); }
			&.type-match_result { background: rgba(14, 165, 233, 0.12); }
			&.type-rank_change { background: rgba(59, 130, 246, 0.12); }
			&.type-friend_request { background: #f3e8ff; }
			&.type-system { background: #FAF8F2; }

			.icon-text {
				font-size: 28rpx;
			}
		}

		.item-content {
			flex: 1;
			min-width: 0;

			.item-header {
				display: flex;
				align-items: center;
				gap: 8rpx;
				margin-bottom: 8rpx;

				.item-title {
					font-size: 28rpx;
					font-weight: 600;
					color: #231C0B;
				}
					.unread-dot {
						width: 12rpx;
						height: 12rpx;
						border-radius: 50%;
						background: #e0ae12;
						flex-shrink: 0;
					}
			}

			.item-body {
				font-size: 26rpx;
				color: #6E6242;
				display: block;
				margin-bottom: 8rpx;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}

			.item-time {
				font-size: 22rpx;
				color: #9A8C67;
			}
		}

		.item-delete {
			flex-shrink: 0;
			padding: 8rpx;
		}
	}
}

.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 260rpx;

	.empty-icon {
		font-size: 80rpx;
		margin-bottom: 24rpx;
	}
	.empty-text {
		font-size: 30rpx;
		color: #6E6242;
		margin-bottom: 8rpx;
	}
	.empty-hint {
		font-size: 24rpx;
		color: #9A8C67;
	}
}

.error-action {
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

.error-action::after {
	display: none;
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

.loading-state {
	display: flex;
	justify-content: center;
	padding: 40rpx 0;
}

	.load-more {
		text-align: center;
		padding: 32rpx 0;
		font-size: 26rpx;
		color: #c69200;
	}

.notification-page.dark-mode {
	background: #141109;

	.top-bar,
	.notification-list .notification-item {
		background: #1e180d;
	}

	.top-bar {
		border-bottom-color: #3a2e16;
	}

	.notification-list .notification-item {
		&.unread {
			background: #2a2110;
		}

		.item-icon.type-system {
			background: #3a2e16;
		}

		.item-content .item-title {
			color: #fff7e1;
		}

		.item-content .item-body {
			color: #d7c89b;
		}

		.item-content .item-time {
			color: #9f926e;
		}
	}

	.page-subtitle,
	.empty-text,
	.empty-hint {
		color: #9f926e;
	}
}
</style>
