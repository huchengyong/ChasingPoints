<template>
	<view class="venue-page">
		<!-- 定位与城市 -->
		<view class="location-bar">
			<view class="location-info" @tap="getLocation">
				<uni-icons type="location" size="18" color="#18b05b"></uni-icons>
				<text class="location-text">{{ currentCity || '定位中...' }}</text>
				<uni-icons type="refreshempty" size="14" color="#94a3b8"></uni-icons>
			</view>
			<view class="submit-btn" @tap="goSubmit">
				<uni-icons type="plusempty" size="16" color="#fff"></uni-icons>
				<text class="submit-text">上传球馆</text>
			</view>
		</view>

		<!-- 模式切换 -->
		<view class="mode-tabs">
			<view class="tab-item" :class="{ active: mode === 'nearby' }" @tap="switchMode('nearby')">
				<text>附近球馆</text>
			</view>
			<view class="tab-item" :class="{ active: mode === 'list' }" @tap="switchMode('list')">
				<text>全部球馆</text>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="loading && page === 1" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 球馆列表 -->
		<scroll-view
			v-else
			scroll-y
			class="venue-scroll"
			@scrolltolower="loadMore"
			refresher-enabled
			:refresher-triggered="refreshing"
			@refresherrefresh="onRefresh"
		>
			<view v-if="list.length > 0" class="venue-list">
				<view
					v-for="item in list"
					:key="item.id"
					class="venue-card"
					@tap="goDetail(item.id)"
				>
					<view class="card-left">
						<view class="venue-img-wrap">
							<image
								v-if="item.images && item.images.length > 0"
								class="venue-img"
								:src="item.images[0]"
								mode="aspectFill"
							></image>
							<view v-else class="venue-img-placeholder">
								<text>🎱</text>
							</view>
						</view>
					</view>
					<view class="card-right">
						<text class="venue-name">{{ item.name }}</text>
						<text class="venue-address">{{ item.address }}</text>
						<view class="venue-meta">
							<text v-if="item.table_count" class="meta-item">{{ item.table_count }}台</text>
							<text v-if="item.price_range" class="meta-item">{{ item.price_range }}</text>
						</view>
						<view class="venue-bottom">
							<text v-if="item.distance" class="venue-distance">{{ formatDistance(item.distance) }}</text>
							<text v-if="item.checkin_count" class="venue-checkins">{{ item.checkin_count }}人签到</text>
						</view>
					</view>
				</view>
			</view>

			<!-- 空状态 -->
			<view v-else class="empty-state">
				<text class="empty-icon">🎱</text>
				<text class="empty-text">{{ mode === 'nearby' ? '附近暂无球馆' : '暂无球馆数据' }}</text>
				<view class="empty-btn" @tap="goSubmit">
					<text>上传球馆信息</text>
				</view>
			</view>

			<!-- 加载更多 -->
			<view v-if="list.length > 0 && !hasMore" class="no-more">
				<text>没有更多了</text>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getVenueList, getNearbyVenues } from '@/api/venue.js'

const mode = ref('nearby')
const list = ref([])
const loading = ref(false)
const refreshing = ref(false)
const page = ref(1)
const hasMore = ref(true)
const currentCity = ref('')
const latitude = ref(0)
const longitude = ref(0)

const formatDistance = (meters) => {
	if (meters < 1000) return Math.round(meters) + 'm'
	return (meters / 1000).toFixed(1) + 'km'
}

const getLocation = () => {
	uni.getLocation({
		type: 'gcj02',
		success: (res) => {
			latitude.value = res.latitude
			longitude.value = res.longitude
			currentCity.value = '已获取位置'
			// 重新加载列表
			page.value = 1
			fetchList(true)
		},
		fail: () => {
			uni.showToast({ title: '未获取定位，将展示全部球馆', icon: 'none' })
			currentCity.value = '未授权定位'
			mode.value = 'list'
			page.value = 1
			fetchList(true)
		}
	})
}

const fetchList = async (isRefresh = false) => {
	if (loading.value) return
	loading.value = true
	try {
		let res
		if (mode.value === 'nearby' && latitude.value && longitude.value) {
			res = await getNearbyVenues({
				latitude: latitude.value,
				longitude: longitude.value,
				radius: 10000,
				limit: 20
			})
		} else {
			const params = { page: page.value, page_size: 10 }
			if (latitude.value && longitude.value) {
				params.latitude = latitude.value
				params.longitude = longitude.value
			}
			res = await getVenueList(params)
		}
		if (res.success) {
			const newList = res.list || []
			if (isRefresh) {
				list.value = newList
			} else {
				list.value = [...list.value, ...newList]
			}
			if (mode.value === 'nearby') {
				hasMore.value = false
			} else {
				hasMore.value = list.value.length < (res.total || 0)
			}
		}
	} catch (e) {
		console.error('获取球馆列表失败', e)
	} finally {
		loading.value = false
		refreshing.value = false
	}
}

const onRefresh = () => {
	refreshing.value = true
	page.value = 1
	fetchList(true)
}

const loadMore = () => {
	if (!hasMore.value || loading.value) return
	page.value++
	fetchList()
}

const switchMode = (newMode) => {
	if (mode.value === newMode) return
	mode.value = newMode
	page.value = 1
	list.value = []
	fetchList(true)
}

const goDetail = (id) => {
	uni.navigateTo({ url: '/subPages/venue/detail?id=' + id })
}

const goSubmit = () => {
	uni.navigateTo({ url: '/subPages/venue/submit' })
}

onMounted(() => {
	getLocation()
})
</script>

<style lang="scss" scoped>
.venue-page {
	min-height: 100vh;
	background: #f1f5f9;
}
.location-bar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 20rpx 24rpx;
	background: #fff;
	.location-info {
		display: flex;
		align-items: center;
		gap: 8rpx;
		.location-text { font-size: 28rpx; color: #1e293b; font-weight: 500; }
	}
	.submit-btn {
		display: flex;
		align-items: center;
		gap: 6rpx;
		padding: 12rpx 24rpx;
		background: #18b05b;
		border-radius: 12rpx;
		.submit-text { font-size: 26rpx; color: #fff; }
	}
}
.mode-tabs {
	display: flex;
	background: #fff;
	padding: 0 24rpx 20rpx;
	gap: 16rpx;
	.tab-item {
		padding: 12rpx 32rpx;
		border-radius: 20rpx;
		background: #f1f5f9;
		font-size: 26rpx;
		color: #64748b;
		&.active {
			background: #f0fdf4;
			color: #18b05b;
			font-weight: 500;
		}
	}
}
.loading-state {
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
	min-height: 50vh;
	.loading-text { font-size: 28rpx; color: #94a3b8; margin-top: 16rpx; }
}
.venue-scroll {
	height: calc(100vh - 200rpx);
}
.venue-list {
	padding: 20rpx 24rpx;
}
.venue-card {
	display: flex;
	background: #fff;
	border-radius: 16rpx;
	padding: 24rpx;
	margin-bottom: 16rpx;
	gap: 20rpx;
	.card-left {
		.venue-img-wrap {
			width: 160rpx;
			height: 160rpx;
			border-radius: 12rpx;
			overflow: hidden;
			.venue-img { width: 100%; height: 100%; }
			.venue-img-placeholder {
				width: 100%;
				height: 100%;
				background: #f1f5f9;
				display: flex;
				align-items: center;
				justify-content: center;
				font-size: 48rpx;
			}
		}
	}
	.card-right {
		flex: 1;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		.venue-name { font-size: 30rpx; font-weight: 600; color: #1e293b; margin-bottom: 6rpx; }
		.venue-address { font-size: 24rpx; color: #94a3b8; margin-bottom: 8rpx; }
		.venue-meta {
			display: flex;
			gap: 16rpx;
			margin-bottom: 8rpx;
			.meta-item {
				font-size: 22rpx;
				color: #64748b;
				padding: 4rpx 12rpx;
				background: #f1f5f9;
				border-radius: 6rpx;
			}
		}
		.venue-bottom {
			display: flex;
			gap: 20rpx;
			.venue-distance { font-size: 24rpx; color: #18b05b; }
			.venue-checkins { font-size: 24rpx; color: #94a3b8; }
		}
	}
}
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 50vh;
	.empty-icon { font-size: 80rpx; margin-bottom: 16rpx; }
	.empty-text { font-size: 28rpx; color: #94a3b8; margin-bottom: 24rpx; }
	.empty-btn {
		padding: 16rpx 40rpx;
		background: #18b05b;
		border-radius: 12rpx;
		color: #fff;
		font-size: 28rpx;
	}
}
.no-more {
	text-align: center;
	padding: 32rpx;
	font-size: 24rpx;
	color: #cbd5e1;
}
</style>
