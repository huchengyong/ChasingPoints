<template>
	<view class="notification-page">
		<!-- 顶部操作栏 -->
		<view class="top-bar">
			<text class="page-subtitle">全部消息</text>
			<text class="read-all-btn" @tap="handleReadAll">全部已读</text>
		</view>

		<!-- 通知列表 -->
		<view v-if="notificationList.length > 0" class="notification-list">
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
					<uni-icons type="trash" size="18" color="#94a3b8"></uni-icons>
				</view>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-if="!loading && notificationList.length === 0" class="empty-state">
			<text class="empty-icon">🔔</text>
			<text class="empty-text">暂无消息</text>
			<text class="empty-hint">有新消息时会在这里通知你</text>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
				<uni-icons type="spinner-cycle" size="28" color="#E0AE12"></uni-icons>
		</view>

		<!-- 加载更多 -->
		<view v-if="!loading && hasMore" class="load-more" @tap="loadMore">
			<text>加载更多</text>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { getNotificationList, markAsRead, markAllAsRead, deleteNotification } from '@/api/notification.js'
import { useNotificationStore } from '@/store/notification.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'

const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()
const notificationList = ref([])
const loading = ref(false)
const hasMore = ref(true)
const page = ref(1)
const pageSize = 20

const getTypeIcon = (type) => {
	const map = {
		challenge: '🎯',
		tournament: '🏆',
		friend_request: '👥',
		follow: '⭐',
		match_result: '🏁',
		rank_change: '📊',
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
	const target = data.target || ''

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
	if (!item.is_read) {
		try {
			await markAsRead({ notification_id: item.id })
			item.is_read = true
			notificationStore.decrementUnread()
			if (item.type === 'friend_request' && item.title === '收到好友申请') {
				friendRequestStore.fetchPendingCount()
			}
		} catch (e) {
			console.error('标记已读失败:', e)
		}
	}

	navigateByNotification(item)
}

const handleReadAll = async () => {
	try {
		await markAllAsRead()
		notificationList.value.forEach(item => {
			item.is_read = true
		})
		notificationStore.clearUnread()
		uni.showToast({ title: '已全部标记为已读', icon: 'success' })
	} catch (e) {
		console.error('全部已读失败:', e)
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const handleDelete = async (id) => {
	try {
		await deleteNotification({ notification_id: id })
		const idx = notificationList.value.findIndex(n => n.id === id)
		if (idx >= 0) {
			const item = notificationList.value[idx]
			if (!item.is_read) {
				notificationStore.decrementUnread()
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
	loading.value = true

	if (isRefresh) {
		page.value = 1
		hasMore.value = true
	}

	try {
		const res = await getNotificationList({ page: page.value, page_size: pageSize })
		const list = res.list || res || []

		if (isRefresh) {
			notificationList.value = list
		} else {
			notificationList.value = [...notificationList.value, ...list]
		}

		hasMore.value = list.length >= pageSize
	} catch (e) {
		console.error('加载通知失败:', e)
	} finally {
		loading.value = false
		if (isRefresh) {
			uni.stopPullDownRefresh()
		}
	}
}

const loadMore = () => {
	page.value++
	loadNotifications()
}

onLoad(() => {
	loadNotifications(true)
	notificationStore.fetchUnreadCount()
})

onPullDownRefresh(() => {
	loadNotifications(true)
})
</script>

<style lang="scss" scoped>
.notification-page {
	min-height: 100vh;
	background: #f1f5f9;
}

.top-bar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 24rpx 32rpx;
	background: #fff;
	border-bottom: 1rpx solid #f1f5f9;

	.page-subtitle {
		font-size: 28rpx;
		color: #64748b;
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
				background: #f8fafc;
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
			&.type-system { background: #f1f5f9; }

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
					color: #1e293b;
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
				color: #64748b;
				display: block;
				margin-bottom: 8rpx;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}

			.item-time {
				font-size: 22rpx;
				color: #94a3b8;
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
		color: #64748b;
		margin-bottom: 8rpx;
	}
	.empty-hint {
		font-size: 24rpx;
		color: #94a3b8;
	}
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
</style>
