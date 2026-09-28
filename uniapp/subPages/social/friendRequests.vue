<template>
	<view class="requests-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 请求列表 -->
		<view v-else-if="requestList.length > 0" class="request-list">
			<view v-if="requestRefreshError" class="refresh-error-banner">
				<text>{{ requestRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryRequests">重试</text>
			</view>
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

		<view v-else-if="requestPageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ requestPageError.title }}</text>
			<text class="empty-hint">{{ requestPageError.description }}</text>
			<button class="retry-btn" @tap="handleRequestErrorAction">{{ requestPageError.actionText }}</button>
		</view>

		<!-- 空状态 -->
		<view v-else-if="requestState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
			<text class="empty-icon">📬</text>
			<text class="empty-text">暂无好友请求</text>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onPullDownRefresh } from '@dcloudio/uni-app'
import { getFriendRequests, acceptFriendRequest, rejectFriendRequest } from '@/api/friend.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { useUserStore } from '@/store/user.js'
import { formatRelativeTime } from '@/utils/format.js'
import { usePageTheme } from '@/utils/page-theme.js'
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

const { isDarkMode } = usePageTheme()

const loading = ref(true)
const requestList = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(false)
const friendRequestStore = useFriendRequestStore()
const userStore = useUserStore()
const requestState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const latestRequestId = ref(0)
const requestedIdentityKey = ref('')
const loadedIdentityKey = ref('')
const requestPageError = computed(() => (
	requestState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(requestState.value.error, { resource: '好友申请' })
		: null
))
const requestRefreshError = computed(() => (
	requestState.value.refreshError
		? resolveAsyncPageErrorFeedback(requestState.value.refreshError, { resource: '好友申请' })
		: null
))

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
	const nextState = beginAsyncPageLoad(requestState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	latestRequestId.value = requestId
	requestedIdentityKey.value = requestIdentityKey
	requestState.value = nextState
	if (isRefresh) {
		page.value = 1
	}
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const res = await getFriendRequests({ page: page.value, page_size: pageSize })
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey || requestState.value.requestId !== pageRequest.requestId || requestState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '加载好友申请失败', category: 'business' })
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
		requestState.value = resolveAsyncPageLoad(requestState.value, pageRequest, { data: requestList.value })
	} catch (e) {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		requestedIdentityKey.value = ''
		requestState.value = rejectAsyncPageLoad(requestState.value, pageRequest, e)
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

const retryRequests = () => loadData(true)

const handleRequestErrorAction = () => {
	if (requestState.value.error?.category === 'permission' || requestState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryRequests()
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
		requestState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
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
	background: #FAF8F2;
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
			background: #FAF8F2;
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
			color: #231C0B;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.request-time {
			font-size: 22rpx;
			color: #9A8C67;
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
			color: #6E6242;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;

			&.is-muted {
				color: #9A8C67;
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
				background: #FAF8F2;
				color: #6E6242;
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
					color: #9A8C67;
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
			color: #6E6242;
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
		color: #9A8C67;
	}
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
