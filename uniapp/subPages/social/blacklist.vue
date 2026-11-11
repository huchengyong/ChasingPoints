<template>
	<view class="blacklist-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="blacklist.length > 0" class="blacklist-list">
			<view v-if="refreshError" class="refresh-error-banner">
				<text>{{ refreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryBlacklist">重试</text>
			</view>
			<view
				v-for="item in blacklist"
				:key="item.friend_user_id"
				class="blacklist-item"
			>
				<memberAvatar
					class="item-avatar"
					:avatar="item.avatar"
					:user-id="item.friend_user_id"
					:size="88"
					reserve-frame
				/>
				<view class="item-info">
					<text class="item-name">{{ item.nickname || '球友' }}</text>
					<text class="item-time">{{ formatBlockedTime(item.created_at) }}</text>
				</view>
				<button
					class="remove-btn"
					:disabled="isRemoving(item.friend_user_id)"
					@tap="showRemoveConfirm(item)"
				>
					<text>移出</text>
				</button>
			</view>

			<view v-if="hasMore" class="load-more">
				<text class="load-more-text">上拉加载更多</text>
			</view>
		</view>

		<view v-else-if="pageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ pageError.title }}</text>
			<text class="empty-hint">{{ pageError.description }}</text>
			<button class="retry-btn" @tap="handleErrorAction">{{ pageError.actionText }}</button>
		</view>

		<view v-else-if="pageState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
			<text class="empty-icon">🚫</text>
			<text class="empty-text">黑名单是空的</text>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onReachBottom } from '@dcloudio/uni-app'
import { getFriendBlacklist, unblacklistFriend } from '@/api/friend.js'
import { useUserStore } from '@/store/user.js'
import { formatRelativeTime } from '@/utils/format.js'
import { usePageTheme } from '@/utils/page-theme.js'
import memberAvatar from '@/components/memberAvatar.vue'
import {
	buildUnblacklistConfirmCopy,
	buildUnblacklistPayload,
	normalizeFriendListItem,
	removeBlacklistItem,
	resolveFriendUserId
} from '@/utils/friend-entry.js'
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
const blacklist = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(false)
const removingIds = ref({})
const latestRequestId = ref(0)
const requestedIdentityKey = ref('')
const loadedIdentityKey = ref('')
const pageState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const pageError = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(pageState.value.error, { resource: '黑名单' })
		: null
))
const refreshError = computed(() => (
	pageState.value.refreshError
		? resolveAsyncPageErrorFeedback(pageState.value.refreshError, { resource: '黑名单' })
		: null
))

const currentIdentityKey = () => `${userStore.userId}:${userStore.authGeneration}`

const formatBlockedTime = (dateTime) => {
	if (!dateTime) return ''
	const parsed = new Date(dateTime)
	if (Number.isNaN(parsed.getTime())) return String(dateTime)
	return `${formatRelativeTime(parsed)}拉黑`
}

const isRemoving = (userId) => Boolean(removingIds.value[userId])

const setRemoving = (userId, removing) => {
	const next = { ...removingIds.value }
	if (removing) {
		next[userId] = true
	} else {
		delete next[userId]
	}
	removingIds.value = next
}

const loadData = async (isRefresh = false) => {
	const requestId = latestRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	const nextState = beginAsyncPageLoad(pageState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	latestRequestId.value = requestId
	requestedIdentityKey.value = requestIdentityKey
	pageState.value = nextState
	if (isRefresh) {
		page.value = 1
	}
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		const res = await getFriendBlacklist({ page: page.value, page_size: pageSize })
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey || pageState.value.requestId !== pageRequest.requestId || pageState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '加载黑名单失败', category: 'business' })
		const list = (res.list || []).map(normalizeFriendListItem)
		blacklist.value = isRefresh ? list : [...blacklist.value, ...list]
		total.value = res.total || blacklist.value.length
		hasMore.value = blacklist.value.length < total.value
		loadedIdentityKey.value = requestIdentityKey
		pageState.value = resolveAsyncPageLoad(pageState.value, pageRequest, { data: blacklist.value })
	} catch (e) {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		requestedIdentityKey.value = ''
		pageState.value = rejectAsyncPageLoad(pageState.value, pageRequest, e)
	} finally {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		loading.value = false
	}
}

const loadMore = () => {
	if (!hasMore.value || loading.value) return
	page.value++
	loadData(false)
}

const retryBlacklist = () => loadData(true)

const handleErrorAction = () => {
	if (pageState.value.error?.category === 'permission' || pageState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryBlacklist()
}

const showRemoveConfirm = (item) => {
	const userId = resolveFriendUserId(item)
	if (!userId || isRemoving(userId)) return
	uni.showModal({
		title: '移出黑名单',
		content: buildUnblacklistConfirmCopy(item.nickname),
		success: async ({ confirm }) => {
			if (!confirm) return
			await handleRemove(item, userId)
		}
	})
}

const handleRemove = async (item, userId) => {
	if (isRemoving(userId)) return
	const requestIdentityKey = currentIdentityKey()
	setRemoving(userId, true)
	try {
		const res = await unblacklistFriend(buildUnblacklistPayload(item))
		if (currentIdentityKey() !== requestIdentityKey) return
		if (!res?.success) {
			uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
			return
		}
		blacklist.value = removeBlacklistItem(blacklist.value, userId)
		total.value = Math.max(total.value - 1, 0)
		hasMore.value = blacklist.value.length < total.value
		pageState.value = {
			...pageState.value,
			status: blacklist.value.length ? ASYNC_PAGE_STATUS.READY : ASYNC_PAGE_STATUS.EMPTY,
			data: blacklist.value,
			hasData: blacklist.value.length > 0
		}
		uni.showToast({ title: '已移出黑名单', icon: 'none' })
	} catch (e) {
		if (currentIdentityKey() !== requestIdentityKey) return
		uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
	} finally {
		setRemoving(userId, false)
	}
}

onLoad(() => {
	loadData(true)
})

onShow(() => {
	if (loadedIdentityKey.value !== currentIdentityKey() && requestedIdentityKey.value !== currentIdentityKey()) {
		blacklist.value = []
		page.value = 1
		total.value = 0
		hasMore.value = false
		pageState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
		loadData(true)
	}
})

onReachBottom(() => {
	loadMore()
})
</script>

<style lang="scss" scoped>
.blacklist-page {
	min-height: 100vh;
	background: #F7F4EC;
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

.blacklist-list {
	padding: 20rpx 24rpx;

	.blacklist-item {
		display: flex;
		align-items: center;
		gap: 20rpx;
		background: #fff;
		border-radius: 24rpx;
		padding: 24rpx;
		margin-bottom: 18rpx;
		border: 1rpx solid rgba(224, 174, 18, 0.14);
		box-shadow: 0 12rpx 32rpx rgba(15, 23, 42, 0.06);
		box-sizing: border-box;
	}

	.item-avatar {
		flex-shrink: 0;
	}

	.item-info {
		flex: 1;
		min-width: 0;
	}

	.item-name {
		display: block;
		font-size: 30rpx;
		font-weight: 600;
		color: #231C0B;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.item-time {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		color: #9A8C67;
	}

	.remove-btn {
		flex-shrink: 0;
		min-width: 128rpx;
		height: 64rpx;
		line-height: 64rpx;
		margin: 0;
		padding: 0 28rpx;
		border-radius: 999rpx;
		background: #FAF8F2;
		border: 1rpx solid rgba(148, 163, 184, 0.24);
		box-sizing: border-box;

		&::after {
			display: none;
		}

		text {
			font-size: 24rpx;
			font-weight: 600;
			color: #6E6242;
		}

		&[disabled] {
			opacity: 0.6;
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

	.empty-hint {
		margin-top: 12rpx;
		font-size: 24rpx;
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

.blacklist-page.dark-mode {
	background: #141109;

	.blacklist-list .blacklist-item {
		background: #1e180d;
		border-color: #3a2e16;
		box-shadow: none;

		.item-name {
			color: #fff7e1;
		}

		.remove-btn {
			background: #2a2110;
			border-color: #3a2e16;

			text {
				color: #d7c89b;
			}
		}
	}

	.loading-text,
	.item-time,
	.load-more-text,
	.empty-text,
	.empty-hint {
		color: #9f926e;
	}
}
</style>
