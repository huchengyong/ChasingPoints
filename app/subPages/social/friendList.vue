<template>
	<view class="friend-list-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="mode === 'pk-report'" class="page-banner">
			<uni-icons type="bars" size="18" color="#E0AE12"></uni-icons>
			<text>选择一位好友，基于你们的真实交锋记录生成 PK 报表。</text>
		</view>

		<!-- 顶部搜索 + 操作 -->
		<view class="header-bar">
			<view class="search-box" @tap="goToAddFriend">
				<uni-icons type="search" size="18" color="#94a3b8"></uni-icons>
				<text class="search-placeholder">搜索/添加好友</text>
			</view>
			<view class="header-actions">
				<view class="action-btn" @tap="goToRequests">
					<uni-icons type="person-filled" size="22" color="#E0AE12"></uni-icons>
					<view v-if="pendingCount > 0" class="action-badge">
						<text class="badge-num">{{ pendingCount > 99 ? '99+' : pendingCount }}</text>
					</view>
				</view>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 好友列表 -->
		<view v-else-if="friendList.length > 0" class="friend-list">
			<view
				v-for="item in friendList"
				:key="item.friend_user_id || item.id"
				class="friend-swipe-item"
			>
				<view class="swipe-actions">
					<view class="swipe-action-btn blacklist" @tap.stop="showBlacklistConfirm(item)">
						<text>拉黑</text>
					</view>
					<view class="swipe-action-btn delete" @tap.stop="showDeleteConfirm(item)">
						<text>删除</text>
					</view>
				</view>
				<view
					class="friend-item"
					:style="getFriendItemStyle(item)"
					@tap="handleFriendTap(item)"
					@touchstart="handleItemTouchStart(item, $event)"
					@touchmove="handleItemTouchMove(item, $event)"
					@touchend="handleItemTouchEnd(item)"
					@touchcancel="handleItemTouchEnd(item)"
				>
					<image
						class="friend-avatar"
						:src="resolveAvatarUrl(item.avatar, item.friend_user_id || item.user_id || item.id)"
						mode="aspectFill"
					/>
					<view class="friend-info">
						<view class="friend-primary">
							<text class="friend-name">{{ item.nickname || '球友' }}</text>
							<text v-if="item.rank_name" class="friend-rank">{{ item.rank_name }}</text>
						</view>
						<text class="friend-sub">ID: {{ item.friend_user_id }}</text>
					</view>
					<uni-icons type="right" size="16" color="#cbd5e1"></uni-icons>
				</view>
			</view>

			<!-- 加载更多 -->
			<view v-if="hasMore" class="load-more" @tap="loadMore">
				<text class="load-more-text">加载更多</text>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-else class="empty-state">
			<text class="empty-icon">👥</text>
			<text class="empty-title">还没有好友</text>
			<text class="empty-sub">去添加好友，一起打球吧！</text>
			<button class="add-btn" @tap="goToAddFriend">
				<text>添加好友</text>
			</button>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { getFriendList, getFriendRequests, deleteFriend, blacklistFriend } from '@/api/friend.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import {
	buildBlacklistFriendPayload,
	buildDeleteFriendPayload,
	buildFriendHomepageUrl,
	buildFriendPkReportUrl,
	normalizeFriendListItem,
	resolveFriendUserId
} from '@/utils/friend-entry.js'
import { clampFriendSwipeOffset, resolveFriendSwipeEndOffset } from '@/utils/friend-swipe.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()

const FRIEND_ACTION_WIDTH_RPX = 280

const createEmptyTouchState = () => ({
	friendUserId: 0,
	startX: 0,
	startY: 0,
	startOffset: 0,
	direction: ''
})

const loading = ref(true)
const friendList = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(false)
const pendingCount = ref(0)
const mode = ref('default')
const actionWidth = ref(140)
const openFriendUserId = ref(0)
const swipeOffsets = ref({})
const touchState = ref(createEmptyTouchState())
const friendRequestStore = useFriendRequestStore()

const loadData = async (isRefresh = false) => {
	if (isRefresh) {
		page.value = 1
		loading.value = true
		resetSwipeState()
	}
	try {
		const [friendRes, requestRes] = await Promise.all([
			getFriendList({ page: page.value, page_size: pageSize }),
			getFriendRequests({ page: 1, page_size: 1 }).catch(() => null)
		])
		const list = (friendRes.list || friendRes || []).map(normalizeFriendListItem)
		if (isRefresh) {
			friendList.value = list
		} else {
			friendList.value = [...friendList.value, ...list]
		}
		total.value = friendRes.total || list.length
		hasMore.value = friendList.value.length < total.value
		if (requestRes) {
			pendingCount.value = requestRes.total || 0
			friendRequestStore.pendingCount = pendingCount.value
		}
	} catch (e) {
		console.error('加载好友列表失败:', e)
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

const goToAddFriend = () => {
	uni.navigateTo({ url: '/subPages/social/addFriend' })
}

const goToRequests = () => {
	uni.navigateTo({ url: '/subPages/social/friendRequests' })
}

const navigateToFriendDetail = (item) => {
	if (mode.value === 'pk-report') {
		uni.navigateTo({ url: buildFriendPkReportUrl(item) })
		return
	}
	uni.navigateTo({ url: buildFriendHomepageUrl(item) })
}

const resetSwipeState = () => {
	swipeOffsets.value = {}
	openFriendUserId.value = 0
	touchState.value = createEmptyTouchState()
}

const getTouchClient = (event) => {
	const touch = event?.touches?.[0] || event?.changedTouches?.[0]
	return {
		x: touch?.clientX ?? touch?.pageX ?? touch?.x ?? 0,
		y: touch?.clientY ?? touch?.pageY ?? touch?.y ?? 0
	}
}

const getFriendUserId = (item) => resolveFriendUserId(item)

const getFriendOffset = (friendUserId) => swipeOffsets.value[friendUserId] || 0

const setFriendOffset = (friendUserId, offset) => {
	swipeOffsets.value = {
		...swipeOffsets.value,
		[friendUserId]: offset
	}
}

const closeSwipeItem = (friendUserId) => {
	if (!friendUserId) return
	setFriendOffset(friendUserId, 0)
	if (openFriendUserId.value === friendUserId) {
		openFriendUserId.value = 0
	}
}

const closeOtherSwipeItems = (activeFriendUserId = 0) => {
	const nextOffsets = {}
	Object.keys(swipeOffsets.value).forEach((key) => {
		const currentId = Number(key)
		nextOffsets[currentId] = currentId === activeFriendUserId ? swipeOffsets.value[currentId] || 0 : 0
	})
	swipeOffsets.value = nextOffsets
	if (openFriendUserId.value && openFriendUserId.value !== activeFriendUserId) {
		openFriendUserId.value = 0
	}
}

const getFriendItemStyle = (item) => {
	const friendUserId = getFriendUserId(item)
	return {
		transform: `translate3d(${getFriendOffset(friendUserId)}px, 0, 0)`
	}
}

const handleFriendTap = (item) => {
	const friendUserId = getFriendUserId(item)
	if (!friendUserId) {
		uni.showToast({ title: '好友信息异常', icon: 'none' })
		return
	}

	if (openFriendUserId.value && openFriendUserId.value !== friendUserId) {
		closeSwipeItem(openFriendUserId.value)
		return
	}

	if (getFriendOffset(friendUserId) < 0) {
		closeSwipeItem(friendUserId)
		return
	}

	navigateToFriendDetail(item)
}

const handleItemTouchStart = (item, event) => {
	const friendUserId = getFriendUserId(item)
	if (!friendUserId) return

	closeOtherSwipeItems(friendUserId)
	const point = getTouchClient(event)
	touchState.value = {
		friendUserId,
		startX: point.x,
		startY: point.y,
		startOffset: getFriendOffset(friendUserId),
		direction: ''
	}
}

const handleItemTouchMove = (item, event) => {
	const friendUserId = getFriendUserId(item)
	const currentTouchState = touchState.value
	if (!friendUserId || currentTouchState.friendUserId !== friendUserId) return

	const point = getTouchClient(event)
	const deltaX = point.x - currentTouchState.startX
	const deltaY = point.y - currentTouchState.startY
	const absDeltaX = Math.abs(deltaX)
	const absDeltaY = Math.abs(deltaY)

	if (!currentTouchState.direction) {
		if (absDeltaY > absDeltaX && absDeltaY > 8) {
			touchState.value = {
				...currentTouchState,
				direction: 'vertical'
			}
			return
		}
		if (absDeltaX > 8) {
			touchState.value = {
				...currentTouchState,
				direction: 'horizontal'
			}
		}
	}

	if (touchState.value.direction !== 'horizontal') return

	const nextOffset = clampFriendSwipeOffset(currentTouchState.startOffset + deltaX, actionWidth.value)
	setFriendOffset(friendUserId, nextOffset)
}

const handleItemTouchEnd = (item) => {
	const friendUserId = getFriendUserId(item)
	const currentTouchState = touchState.value
	if (!friendUserId || currentTouchState.friendUserId !== friendUserId) return

	if (currentTouchState.direction === 'horizontal') {
		const nextOffset = resolveFriendSwipeEndOffset(getFriendOffset(friendUserId), actionWidth.value)
		setFriendOffset(friendUserId, nextOffset)
		openFriendUserId.value = nextOffset < 0 ? friendUserId : 0
	}

	touchState.value = createEmptyTouchState()
}

const removeFriendFromList = (friendUserId) => {
	friendList.value = friendList.value.filter(item => getFriendUserId(item) !== friendUserId)
	total.value = Math.max(0, total.value - 1)
	hasMore.value = friendList.value.length < total.value
	closeSwipeItem(friendUserId)
}

const showDeleteConfirm = (item) => {
	const friendUserId = getFriendUserId(item)
	if (!friendUserId) {
		uni.showToast({ title: '好友信息异常', icon: 'none' })
		return
	}

	closeSwipeItem(friendUserId)
	uni.showModal({
		title: '删除好友',
		content: `确定删除好友「${item.nickname || '球友'}」吗？`,
		success: async (res) => {
			if (res.confirm) {
				try {
					await deleteFriend(buildDeleteFriendPayload(item))
					removeFriendFromList(friendUserId)
					uni.showToast({ title: '已删除', icon: 'success' })
				} catch (e) {
					uni.showToast({ title: '删除失败', icon: 'none' })
				}
			}
		}
	})
}

const showBlacklistConfirm = (item) => {
	const friendUserId = getFriendUserId(item)
	if (!friendUserId) {
		uni.showToast({ title: '好友信息异常', icon: 'none' })
		return
	}

	closeSwipeItem(friendUserId)
	uni.showModal({
		title: '加入黑名单',
		content: `确定将「${item.nickname || '球友'}」加入黑名单吗？加入后会自动删除好友，并阻止彼此再次搜索和添加。`,
		confirmText: '确认拉黑',
		confirmColor: '#334155',
		success: async (res) => {
			if (res.confirm) {
				try {
					await blacklistFriend(buildBlacklistFriendPayload(item))
					removeFriendFromList(friendUserId)
					uni.showToast({ title: '已加入黑名单', icon: 'success' })
				} catch (e) {
					uni.showToast({ title: '操作失败', icon: 'none' })
				}
			}
		}
	})
}

onLoad((options) => {
	mode.value = options.mode || 'default'
	const resolvedWidth = typeof uni.upx2px === 'function' ? uni.upx2px(FRIEND_ACTION_WIDTH_RPX) : 140
	actionWidth.value = resolvedWidth > 0 ? resolvedWidth : 140
	loadData(true)
})

onPullDownRefresh(() => {
	loadData(true)
})
</script>

<style lang="scss" scoped>
.friend-list-page {
	min-height: 100vh;
	background: #f1f5f9;
}

.page-banner {
	display: flex;
	align-items: flex-start;
	gap: 12rpx;
	margin: 20rpx 24rpx 0;
	padding: 18rpx 20rpx;
	border-radius: 18rpx;
	background: rgba(224, 174, 18, 0.12);

	text {
		font-size: 24rpx;
		line-height: 1.6;
		color: #7c5b12;
	}
}

.header-bar {
	display: flex;
	align-items: center;
	padding: 20rpx 24rpx;
	background: #fff;
	gap: 20rpx;

	.search-box {
		flex: 1;
		display: flex;
		align-items: center;
		background: #f1f5f9;
		border-radius: 36rpx;
		padding: 16rpx 24rpx;
		gap: 12rpx;

		.search-placeholder {
			font-size: 28rpx;
			color: #94a3b8;
		}
	}

	.header-actions {
		.action-btn {
			position: relative;
			padding: 10rpx;

			.action-badge {
				position: absolute;
				top: 0;
				right: -4rpx;
				background: #ef4444;
				border-radius: 20rpx;
				min-width: 32rpx;
				height: 32rpx;
				display: flex;
				align-items: center;
				justify-content: center;
				padding: 0 8rpx;

				.badge-num {
					font-size: 20rpx;
					color: #fff;
					font-weight: 600;
				}
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

.friend-list {
	padding: 20rpx 24rpx;

	.friend-swipe-item {
		position: relative;
		margin-bottom: 16rpx;
		border-radius: 16rpx;
		overflow: hidden;
	}

	.swipe-actions {
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		width: 280rpx;
		display: flex;

		.swipe-action-btn {
			flex: 1;
			display: flex;
			align-items: center;
			justify-content: center;

			text {
				font-size: 26rpx;
				font-weight: 600;
				color: #ffffff;
			}

			&.blacklist {
				background: #475569;
			}

			&.delete {
				background: #ef4444;
			}
		}
	}

	.friend-item {
		position: relative;
		z-index: 1;
		display: flex;
		align-items: center;
		background: #fff;
		border-radius: 16rpx;
		padding: 24rpx;
		transition: transform 0.18s ease;

		.friend-avatar {
			width: 88rpx;
			height: 88rpx;
			border-radius: 50%;
			margin-right: 20rpx;
			flex-shrink: 0;
		}

		.friend-info {
			flex: 1;
			overflow: hidden;

			.friend-primary {
				display: flex;
				align-items: center;
				gap: 12rpx;

				.friend-name {
					font-size: 30rpx;
					font-weight: 500;
					color: #1e293b;
				}
				.friend-rank {
					font-size: 22rpx;
					color: #f59e0b;
					background: #fff7ed;
					padding: 2rpx 12rpx;
					border-radius: 16rpx;
				}
			}

			.friend-sub {
				font-size: 24rpx;
				color: #94a3b8;
				margin-top: 6rpx;
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
	padding-top: 200rpx;

	.empty-icon {
		font-size: 80rpx;
		margin-bottom: 24rpx;
	}
	.empty-title {
		font-size: 32rpx;
		font-weight: 500;
		color: #1e293b;
		margin-bottom: 12rpx;
	}
	.empty-sub {
		font-size: 26rpx;
		color: #94a3b8;
		margin-bottom: 40rpx;
	}

	.add-btn {
		background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
		color: #ffffff;
		border-radius: 40rpx;
		padding: 0 60rpx;
		height: 80rpx;
		line-height: 80rpx;
		font-size: 28rpx;
		border: none;

		&::after {
			border: none;
		}
	}
}

.friend-list-page.dark-mode {
	background: #141109;

	.page-banner {
		background: rgba(224, 174, 18, 0.18);

		text {
			color: #f7e7a8;
		}
	}

	.header-bar,
	.friend-list .friend-item {
		background: #1e180d;
	}

	.header-bar .search-box {
		background: #2a2110;
	}

	.friend-list .friend-item .friend-info .friend-primary {
		.friend-name {
			color: #fff7e1;
		}

		.friend-rank {
			background: rgba(224, 174, 18, 0.16);
			color: #f7e7a8;
		}
	}

	.loading-text,
	.search-placeholder,
	.friend-sub,
	.load-more-text,
	.empty-sub {
		color: #9f926e;
	}

	.empty-title {
		color: #fff7e1;
	}
}
</style>
