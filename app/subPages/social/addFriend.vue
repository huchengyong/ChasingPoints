<template>
	<view class="add-friend-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 搜索框 -->
		<view class="search-bar">
			<view class="search-input-wrap">
				<uni-icons type="search" size="18" color="#9A8C67"></uni-icons>
				<input
					class="search-input"
					v-model="keyword"
					placeholder="搜索昵称或手机号"
					confirm-type="search"
					@confirm="doSearch"
				/>
				<view v-if="keyword" class="clear-btn" @tap="clearSearch">
					<uni-icons type="clear" size="18" color="#9A8C67"></uni-icons>
				</view>
			</view>
			<view class="search-btn" @tap="doSearch">
				<text>搜索</text>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="searching" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">搜索中...</text>
		</view>

		<!-- 搜索结果 -->
		<view v-else-if="hasSearched" class="result-list">
			<view v-if="resultList.length === 0" class="empty-result">
				<text class="empty-text">未找到相关用户</text>
			</view>
			<view
				v-for="item in resultList"
				:key="item.user_id"
				class="user-item"
			>
				<image
					class="user-avatar"
					:src="resolveAvatarUrl(item.avatar, item.user_id)"
					mode="aspectFill"
				/>
				<view class="user-info">
					<text class="user-name">{{ item.nickname || '球友' }}</text>
					<text class="user-id">ID: {{ item.user_id }}</text>
				</view>
				<button
					v-if="item.status === 'friend'"
					class="status-btn friend"
					disabled
				>
					<text>已是好友</text>
				</button>
				<button
					v-else-if="item.status === 'pending'"
					class="status-btn pending"
					disabled
				>
					<text>已发送</text>
				</button>
				<button
					v-else
					class="status-btn add"
					@tap="handleAdd(item)"
				>
					<text>添加</text>
				</button>
			</view>
		</view>

		<!-- 初始引导 -->
		<view v-else class="guide-state">
			<text class="guide-icon">🔍</text>
			<text class="guide-text">输入昵称或手机号搜索用户</text>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { searchUser, sendFriendRequest } from '@/api/friend.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()

const keyword = ref('')
const searching = ref(false)
const hasSearched = ref(false)
const resultList = ref([])

const doSearch = async () => {
	const kw = keyword.value.trim()
	if (!kw) {
		uni.showToast({ title: '请输入搜索内容', icon: 'none' })
		return
	}
	searching.value = true
	hasSearched.value = true
	try {
		const res = await searchUser({ keyword: kw, page: 1, page_size: 20 })
		resultList.value = res.list || res || []
	} catch (e) {
		console.error('搜索用户失败:', e)
		resultList.value = []
	} finally {
		searching.value = false
	}
}

const clearSearch = () => {
	keyword.value = ''
	hasSearched.value = false
	resultList.value = []
}

const handleAdd = async (item) => {
	try {
		const res = await sendFriendRequest({ to_user_id: item.user_id, message: '' })
		if (res.success) {
			item.status = 'pending'
			uni.showToast({ title: '请求已发送', icon: 'success' })
		} else {
			uni.showToast({ title: res.message || '发送失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '发送失败', icon: 'none' })
	}
}
</script>

<style lang="scss" scoped>
.add-friend-page {
	min-height: 100vh;
	background: #FAF8F2;
}

.search-bar {
	display: flex;
	align-items: center;
	padding: 20rpx 24rpx;
	background: #fff;
	gap: 16rpx;

	.search-input-wrap {
		flex: 1;
		display: flex;
		align-items: center;
		background: #FAF8F2;
		border-radius: 36rpx;
		padding: 14rpx 20rpx;
		gap: 12rpx;

		.search-input {
			flex: 1;
			font-size: 28rpx;
			color: #231C0B;
		}
		.clear-btn {
			padding: 4rpx;
		}
	}

	.search-btn {
		padding: 14rpx 28rpx;
		background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
		border-radius: 36rpx;
		text {
			font-size: 28rpx;
			color: #ffffff;
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

.result-list {
	padding: 20rpx 24rpx;

	.empty-result {
		display: flex;
		justify-content: center;
		padding-top: 120rpx;
		.empty-text {
			font-size: 28rpx;
			color: #9A8C67;
		}
	}

	.user-item {
		display: flex;
		align-items: center;
		background: #fff;
		border-radius: 16rpx;
		padding: 24rpx;
		margin-bottom: 16rpx;

		.user-avatar {
			width: 88rpx;
			height: 88rpx;
			border-radius: 50%;
			margin-right: 20rpx;
			flex-shrink: 0;
		}

	.user-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		.user-name {
			font-size: 30rpx;
			font-weight: 500;
			color: #231C0B;
		}
		.user-id {
			font-size: 24rpx;
			color: #9A8C67;
			margin-top: 4rpx;
		}
	}

		.status-btn {
			padding: 0 28rpx;
			height: 60rpx;
			line-height: 60rpx;
			border-radius: 30rpx;
			font-size: 24rpx;
			border: none;
			margin: 0;

			&.add {
				background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
				color: #ffffff;
			}
			&.pending {
				background: #E9E2CF;
				color: #9A8C67;
			}
			&.friend {
				background: #f0fdf4;
				color: #22c55e;
			}
		}
	}
}

.guide-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 240rpx;

	.guide-icon {
		font-size: 80rpx;
		margin-bottom: 24rpx;
	}
	.guide-text {
		font-size: 28rpx;
		color: #9A8C67;
	}
}

.add-friend-page.dark-mode {
	background: #141109;

	.search-bar,
	.result-list .user-item {
		background: #1e180d;
	}

	.search-bar .search-input-wrap {
		background: #2a2110;

		.search-input {
			color: #fff7e1;
		}
	}

	.result-list .user-item {
		.user-name {
			color: #fff7e1;
		}

		.status-btn.pending {
			background: #3a2e16;
			color: #9f926e;
		}

		.status-btn.friend {
			background: rgba(34, 197, 94, 0.16);
		}
	}

	.loading-text,
	.empty-text,
	.user-id,
	.guide-text {
		color: #9f926e;
	}
}
</style>
