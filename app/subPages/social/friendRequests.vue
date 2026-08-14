<template>
	<view class="requests-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 请求列表 -->
		<view v-else-if="requestList.length > 0" class="request-list">
			<view
				v-for="item in requestList"
				:key="item.id"
				class="request-item"
			>
				<view class="request-card-main">
					<image
						class="request-avatar"
						:src="resolveAvatarUrl(item.avatar, item.from_user_id || item.user_id || item.friend_user_id)"
						mode="aspectFill"
					/>
					<view class="request-info">
						<view class="request-head">
							<text class="request-name">{{ item.nickname || '球友' }}</text>
							<text class="request-time">{{ formatRequestTime(item.created_at) }}</text>
						</view>
						<view class="request-meta">
							<text class="request-message" :class="{ 'is-muted': !item.message }">
								{{ item.message || '请求添加你为好友' }}
							</text>
						</view>
					</view>
				</view>
				<view v-if="item.status === 0" class="request-actions">
					<button class="action-button action-reject" @tap="handleReject(item)">
						<text>拒绝</text>
					</button>
					<button class="action-button action-accept" @tap="handleAccept(item)">
						<text>接受</text>
					</button>
				</view>
				<view v-else class="request-status">
					<text class="status-text" :class="{ accepted: item.status === 1, rejected: item.status !== 1 }">
						{{ item.status === 1 ? '已接受' : '已拒绝' }}
					</text>
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
import { onLoad, onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { getFriendRequests, acceptFriendRequest, rejectFriendRequest } from '@/api/friend.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { useUserStore } from '@/store/user.js'
import { formatRelativeTime } from '@/utils/format.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()

const loading = ref(true)
const requestList = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(false)
const friendRequestStore = useFriendRequestStore()
const userStore = useUserStore()
const latestRequestId = ref(0)
const requestedIdentityKey = ref('')
const loadedIdentityKey = ref('')

const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const currentIdentityKey = () => {
	const identity = getReadIdentity()
	return `${identity.userId}:${identity.authGeneration}`
}

const formatRequestTime = (dateTime) => {
	if (!dateTime) return ''
	const parsed = new Date(dateTime)
	if (Number.isNaN(parsed.getTime())) return String(dateTime)
	return formatRelativeTime(parsed)
}

const loadData = async (isRefresh = false) => {
	const requestId = latestRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	latestRequestId.value = requestId
	requestedIdentityKey.value = requestIdentityKey
	if (isRefresh) {
		page.value = 1
		loading.value = true
	}
	try {
		const res = await getFriendRequests({ page: page.value, page_size: pageSize })
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		const list = res.list || res || []
		if (isRefresh) {
			requestList.value = list
		} else {
			requestList.value = [...requestList.value, ...list]
		}
		total.value = res.total || list.length
		friendRequestStore.setPendingCount(total.value, getReadIdentity())
		hasMore.value = requestList.value.length < total.value
		loadedIdentityKey.value = requestIdentityKey
	} catch (e) {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		requestedIdentityKey.value = ''
		console.error('加载好友请求失败:', e)
	} finally {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
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
	const requestIdentityKey = currentIdentityKey()
	try {
		const res = await acceptFriendRequest({ request_id: item.id })
		if (currentIdentityKey() !== requestIdentityKey) return
		if (res.success) {
			item.status = 1
			friendRequestStore.decrementPendingCount(getReadIdentity())
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
	const requestIdentityKey = currentIdentityKey()
	try {
		const res = await rejectFriendRequest({ request_id: item.id })
		if (currentIdentityKey() !== requestIdentityKey) return
		if (res.success) {
			item.status = 2
			friendRequestStore.decrementPendingCount(getReadIdentity())
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

onShow(() => {
	if (loadedIdentityKey.value !== currentIdentityKey() && requestedIdentityKey.value !== currentIdentityKey()) {
		requestList.value = []
		page.value = 1
		loading.value = false
		loadData(true)
	}
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
		flex-direction: column;
		background: #fff;
		border-radius: 24rpx;
		padding: 26rpx 24rpx 22rpx;
		margin-bottom: 18rpx;
		border: 1rpx solid rgba(224, 174, 18, 0.14);
		box-shadow: 0 12rpx 32rpx rgba(15, 23, 42, 0.06);
		box-sizing: border-box;

		.request-card-main {
			width: 100%;
			display: flex;
			align-items: center;
			gap: 20rpx;
			min-width: 0;
		}

		.request-avatar {
			width: 96rpx;
			height: 96rpx;
			border-radius: 50%;
			border: 2rpx solid rgba(224, 174, 18, 0.16);
			background: #f8fafc;
			flex-shrink: 0;
		}

		.request-info {
			flex: 1;
			min-width: 0;
		}

		.request-head {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 16rpx;
			min-width: 0;
		}

		.request-name {
			flex: 1;
			min-width: 0;
			font-size: 30rpx;
			font-weight: 600;
			color: #1e293b;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.request-time {
			font-size: 22rpx;
			color: #94a3b8;
			white-space: nowrap;
			flex-shrink: 0;
		}

		.request-meta {
			margin-top: 10rpx;
			min-width: 0;
		}

		.request-message {
			display: block;
			font-size: 24rpx;
			line-height: 1.5;
			color: #64748b;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;

			&.is-muted {
				color: #94a3b8;
			}
		}

		.request-actions {
			display: flex;
			justify-content: flex-end;
			gap: 12rpx;
			width: 100%;
			margin-top: 18rpx;

			.action-button {
				min-width: 128rpx;
				height: 64rpx;
				line-height: 64rpx;
				padding: 0 28rpx;
				border-radius: 999rpx;
				margin: 0;
				box-sizing: border-box;
				font-size: 24rpx;
				font-weight: 600;
				border: none;

				&::after {
					display: none;
				}
			}

			.action-reject {
				background: #f8fafc;
				color: #64748b;
				border: 1rpx solid rgba(148, 163, 184, 0.24);
			}

			.action-accept {
				background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
				color: #ffffff;
				box-shadow: 0 10rpx 24rpx rgba(224, 174, 18, 0.24);
			}
		}

		.request-status {
			display: flex;
			justify-content: flex-end;
			width: 100%;
			margin-top: 18rpx;

			.status-text {
				display: inline-flex;
				align-items: center;
				font-size: 22rpx;
				font-weight: 500;

				&.accepted {
					color: #0f766e;
				}

				&.rejected {
					color: #94a3b8;
				}
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

.requests-page.dark-mode {
	background: #141109;

	.request-list .request-item {
		background: #1e180d;
		border-color: #3a2e16;
		box-shadow: none;

		.request-avatar,
		.request-actions .action-reject,
		.request-status .status-text.rejected {
			background: #2a2110;
			border-color: #3a2e16;
		}

		.request-name {
			color: #fff7e1;
		}

		.request-message,
		.request-actions .action-reject {
			color: #d7c89b;
		}

		.request-time,
		.request-message.is-muted,
		.request-status .status-text.rejected {
			color: #9f926e;
		}
	}

	.loading-text,
	.load-more-text,
	.empty-text {
		color: #9f926e;
	}
}
</style>
