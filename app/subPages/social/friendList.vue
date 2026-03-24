<template>
	<view class="friend-list-page">
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
				:key="item.friend_id"
				class="friend-item"
				@tap="goToH2H(item)"
				@longpress="showDeleteConfirm(item)"
			>
				<image
					class="friend-avatar"
					:src="item.avatar || '/static/images/default-avatar.png'"
					mode="aspectFill"
				/>
				<view class="friend-info">
					<view class="friend-name-row">
						<text class="friend-name">{{ item.nickname || '球友' }}</text>
						<text v-if="item.rank_name" class="friend-rank">{{ item.rank_name }}</text>
					</view>
					<text class="friend-sub">ID: {{ item.friend_id }}</text>
				</view>
				<uni-icons type="right" size="16" color="#cbd5e1"></uni-icons>
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
import { getFriendList, getFriendRequests, deleteFriend } from '@/api/friend.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'

const loading = ref(true)
const friendList = ref([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = ref(false)
const pendingCount = ref(0)
const mode = ref('default')
const friendRequestStore = useFriendRequestStore()

const loadData = async (isRefresh = false) => {
	if (isRefresh) {
		page.value = 1
		loading.value = true
	}
	try {
		const [friendRes, requestRes] = await Promise.all([
			getFriendList({ page: page.value, page_size: pageSize }),
			getFriendRequests({ page: 1, page_size: 1 }).catch(() => null)
		])
		const list = friendRes.list || friendRes || []
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

const goToH2H = (item) => {
	if (mode.value === 'pk-report') {
		uni.navigateTo({
			url: `/subPages/social/pkReport?opponent_id=${item.friend_id}&opponent_name=${encodeURIComponent(item.nickname || '球友')}&opponent_avatar=${encodeURIComponent(item.avatar || '')}`
		})
		return
	}
	uni.navigateTo({ url: '/subPages/user/h2hRecord?opponent_id=' + item.friend_id })
}

const showDeleteConfirm = (item) => {
	uni.showModal({
		title: '删除好友',
		content: `确定删除好友「${item.nickname || '球友'}」吗？`,
		success: async (res) => {
			if (res.confirm) {
				try {
					await deleteFriend({ friend_id: item.friend_id })
					friendList.value = friendList.value.filter(f => f.friend_id !== item.friend_id)
					uni.showToast({ title: '已删除', icon: 'success' })
				} catch (e) {
					uni.showToast({ title: '删除失败', icon: 'none' })
				}
			}
		}
	})
}

onLoad((options) => {
	mode.value = options.mode || 'default'
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

	.friend-item {
		display: flex;
		align-items: center;
		background: #fff;
		border-radius: 16rpx;
		padding: 24rpx;
		margin-bottom: 16rpx;

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

			.friend-name-row {
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
		}
}
</style>
