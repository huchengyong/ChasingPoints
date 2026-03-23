<template>
	<view class="requests-page">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 请求列表 -->
		<view v-else-if="requestList.length > 0" class="request-list">
			<view
				v-for="item in requestList"
				:key="item.id"
				class="request-item"
			>
				<image
					class="request-avatar"
					:src="item.avatar || '/static/images/default-avatar.png'"
					mode="aspectFill"
				/>
				<view class="request-info">
					<text class="request-name">{{ item.nickname || '球友' }}</text>
					<text v-if="item.message" class="request-message">{{ item.message }}</text>
					<text class="request-time">{{ item.created_at || '' }}</text>
				</view>
				<view v-if="item.status === 0" class="request-actions">
					<button class="action-accept" @tap="handleAccept(item)">
						<text>接受</text>
					</button>
					<button class="action-reject" @tap="handleReject(item)">
						<text>拒绝</text>
					</button>
				</view>
				<view v-else class="request-status">
					<text class="status-text">{{ item.status === 1 ? '已接受' : '已拒绝' }}</text>
				</view>
			</view>

			<!-- 加载更多 -->
			<view v-if="hasMore" class="load-more" @tap="loadMore">
				<text class="load-more-text">加载更多</text>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-else class="empty-state">
			<text class="empty-icon">📬</text>
			<text class="empty-text">暂无好友请求</text>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { getFriendRequests, acceptFriendRequest, rejectFriendRequest } from '@/api/friend.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'

const loading = ref(true)
const requestList = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(false)
const friendRequestStore = useFriendRequestStore()

const loadData = async (isRefresh = false) => {
	if (isRefresh) {
		page.value = 1
		loading.value = true
	}
	try {
		const res = await getFriendRequests({ page: page.value, page_size: pageSize })
		const list = res.list || res || []
		if (isRefresh) {
			requestList.value = list
		} else {
			requestList.value = [...requestList.value, ...list]
		}
		total.value = res.total || list.length
		friendRequestStore.pendingCount = total.value
		hasMore.value = requestList.value.length < total.value
	} catch (e) {
		console.error('加载好友请求失败:', e)
	} finally {
		loading.value = false
		uni.stopPullDownRefresh()
	}
}

const loadMore = () => {
	if (!hasMore.value) return
	page.value++
	loadData(false)
}

const handleAccept = async (item) => {
	try {
		const res = await acceptFriendRequest({ request_id: item.id })
		if (res.success) {
			item.status = 1
			friendRequestStore.decrementPendingCount()
			total.value = Math.max(total.value - 1, 0)
			uni.showToast({ title: '已接受', icon: 'success' })
		} else {
			uni.showToast({ title: res.message || '操作失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const handleReject = async (item) => {
	try {
		const res = await rejectFriendRequest({ request_id: item.id })
		if (res.success) {
			item.status = 2
			friendRequestStore.decrementPendingCount()
			total.value = Math.max(total.value - 1, 0)
			uni.showToast({ title: '已拒绝', icon: 'none' })
		} else {
			uni.showToast({ title: res.message || '操作失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

onLoad(() => {
	loadData(true)
})

onPullDownRefresh(() => {
	loadData(true)
})
</script>

<style lang="scss" scoped>
.requests-page {
	min-height: 100vh;
	background: #f1f5f9;
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

.request-list {
	padding: 20rpx 24rpx;

	.request-item {
		display: flex;
		align-items: center;
		background: #fff;
		border-radius: 16rpx;
		padding: 24rpx;
		margin-bottom: 16rpx;

		.request-avatar {
			width: 88rpx;
			height: 88rpx;
			border-radius: 50%;
			margin-right: 20rpx;
			flex-shrink: 0;
		}

		.request-info {
			flex: 1;
			overflow: hidden;

			.request-name {
				font-size: 30rpx;
				font-weight: 500;
				color: #1e293b;
			}
			.request-message {
				font-size: 24rpx;
				color: #64748b;
				margin-top: 4rpx;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}
			.request-time {
				font-size: 22rpx;
				color: #94a3b8;
				margin-top: 4rpx;
			}
		}

		.request-actions {
			display: flex;
			gap: 12rpx;
			flex-shrink: 0;

			.action-accept {
				background: #18b05b;
				color: #fff;
				font-size: 24rpx;
				padding: 0 24rpx;
				height: 56rpx;
				line-height: 56rpx;
				border-radius: 28rpx;
				border: none;
				margin: 0;
			}
			.action-reject {
				background: #f1f5f9;
				color: #64748b;
				font-size: 24rpx;
				padding: 0 24rpx;
				height: 56rpx;
				line-height: 56rpx;
				border-radius: 28rpx;
				border: none;
				margin: 0;
			}
		}

		.request-status {
			flex-shrink: 0;
			.status-text {
				font-size: 24rpx;
				color: #94a3b8;
			}
		}
	}

	.load-more {
		display: flex;
		justify-content: center;
		padding: 24rpx;
		.load-more-text {
			font-size: 26rpx;
			color: #64748b;
		}
	}
}

.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 240rpx;

	.empty-icon {
		font-size: 80rpx;
		margin-bottom: 24rpx;
	}
	.empty-text {
		font-size: 28rpx;
		color: #94a3b8;
	}
}
</style>
