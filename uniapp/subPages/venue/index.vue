<template>
	<view class="venue-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 定位与城市 -->
		<view class="location-bar">
			<view class="location-info" @tap="getLocation">
					<uni-icons type="location" size="18" color="#E0AE12"></uni-icons>
				<text class="location-text">{{ currentCity || '定位中...' }}</text>
				<uni-icons type="refreshempty" size="14" color="#9A8C67"></uni-icons>
			</view>
			<view class="submit-btn" @tap="goSubmit">
				<uni-icons type="plusempty" size="16" color="#fff"></uni-icons>
				<text class="submit-text">添加常玩球馆</text>
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
				<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
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
				<view v-if="venueRefreshError" class="refresh-error-banner">
					<text>{{ venueRefreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryVenueList">重试</text>
				</view>
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

			<view v-else-if="venuePageError" class="empty-state error-state">
				<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
				<text class="empty-text">{{ venuePageError.title }}</text>
				<text class="error-text">{{ venuePageError.description }}</text>
				<button class="retry-btn" @tap="handleVenueErrorAction">{{ venuePageError.actionText }}</button>
			</view>

			<!-- 空状态 -->
			<view v-else-if="venueState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
				<text class="empty-icon">🎱</text>
				<text class="empty-text">{{ mode === 'nearby' ? '附近暂无球馆' : '暂无球馆数据' }}</text>
				<view class="empty-btn" @tap="goSubmit">
					<text>添加常玩球馆</text>
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
import { computed, ref, onMounted } from 'vue'
import { getVenueList, getNearbyVenues } from '@/api/venue.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { getCurrentLocation } from '@/utils/permission-helper.js'
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

const mode = ref('nearby')
const list = ref([])
const loading = ref(false)
const refreshing = ref(false)
const page = ref(1)
const hasMore = ref(true)
const currentCity = ref('')
const latitude = ref(0)
const longitude = ref(0)
const venueState = ref(createAsyncPageState({ data: [] }))
const venuePageError = computed(() => (
	venueState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(venueState.value.error, { resource: '球馆列表' })
		: null
))
const venueRefreshError = computed(() => (
	venueState.value.refreshError
		? resolveAsyncPageErrorFeedback(venueState.value.refreshError, { resource: '球馆列表' })
		: null
))

const formatDistance = (meters) => {
	if (meters < 1000) return Math.round(meters) + 'm'
	return (meters / 1000).toFixed(1) + 'km'
}

const getLocation = async () => {
	const result = await getCurrentLocation({
		uniApi: uni,
		action: '查看附近球房',
		onPermissionDenied: (permission) => {
			if (permission.reason === 'denied') {
				uni.showToast({
					title: '定位权限未开启，将仅展示全部球馆',
					icon: 'none'
				})
			} else {
				uni.showToast({
					title: '定位失败，将仅展示全部球馆',
					icon: 'none'
				})
			}
		}
	})

	if (!result.success) {
		currentCity.value = result.permission?.reason === 'denied' ? '未授权定位' : '定位失败'
		if (mode.value === 'nearby') {
			mode.value = 'list'
		}
		return false
	}

	latitude.value = result.location.latitude
	longitude.value = result.location.longitude
	currentCity.value = '已获取位置'
	return true
}

const fetchList = async (isRefresh = false) => {
	if (venueState.value.status === ASYNC_PAGE_STATUS.LOADING || venueState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	const requestedPage = page.value
	const nextState = beginAsyncPageLoad(venueState.value, { emptyData: [] })
	const pageRequest = getAsyncPageRequest(nextState)
	venueState.value = nextState
	loading.value = nextState.status === ASYNC_PAGE_STATUS.LOADING
	try {
		if (mode.value === 'nearby' && (!latitude.value || !longitude.value)) {
			await getLocation()
		}

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
		if (venueState.value.requestId !== pageRequest.requestId) return
		if (!res?.success) throw createRequestError({ message: res?.message || '获取球馆列表失败', category: 'business' })
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
		venueState.value = resolveAsyncPageLoad(venueState.value, pageRequest, { data: list.value })
	} catch (e) {
		if (venueState.value.requestId !== pageRequest.requestId) return
		if (!isRefresh && requestedPage > 1) page.value = requestedPage - 1
		venueState.value = rejectAsyncPageLoad(venueState.value, pageRequest, e)
	} finally {
		if (venueState.value.requestId !== pageRequest.requestId) return
		loading.value = false
		refreshing.value = false
	}
}

const onRefresh = () => {
	if (venueState.value.status === ASYNC_PAGE_STATUS.LOADING || venueState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	refreshing.value = true
	page.value = 1
	fetchList(true)
}

const loadMore = () => {
	if (!hasMore.value || venueState.value.status === ASYNC_PAGE_STATUS.LOADING || venueState.value.status === ASYNC_PAGE_STATUS.REFRESHING) return
	page.value++
	fetchList()
}

const switchMode = (newMode) => {
	if (mode.value === newMode) return
	mode.value = newMode
	page.value = 1
	list.value = []
	hasMore.value = false
	resetVenueList()
	fetchList(true)
}

const resetVenueList = () => {
	venueState.value = {
		...venueState.value,
		status: ASYNC_PAGE_STATUS.IDLE,
		requestId: venueState.value.requestId + 1,
		data: [],
		hasData: false,
		error: null,
		refreshError: null
	}
}

const retryVenueList = () => fetchList(true)

const handleVenueErrorAction = () => {
	if (venueState.value.error?.category === 'permission' || venueState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryVenueList()
}

const goDetail = (id) => {
	uni.navigateTo({ url: '/subPages/venue/detail?id=' + id })
}

const goSubmit = () => {
	uni.navigateTo({ url: '/subPages/venue/submit' })
}

onMounted(() => {
	fetchList(true)
})
</script>

<style lang="scss" scoped>
.venue-page {
	min-height: 100vh;
	background: #FAF8F2;
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
		.location-text { font-size: 28rpx; color: #231C0B; font-weight: 500; }
	}
	.submit-btn {
		display: flex;
		align-items: center;
		gap: 6rpx;
		padding: 12rpx 24rpx;
			background: #e0ae12;
			border-radius: 12rpx;
			.submit-text { font-size: 26rpx; color: #ffffff; }
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
		background: #FAF8F2;
		font-size: 26rpx;
		color: #6E6242;
			&.active {
				background: #fff7dc;
				border: 1rpx solid rgba(224, 174, 18, 0.22);
				color: #c69200;
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
	.loading-text { font-size: 28rpx; color: #9A8C67; margin-top: 16rpx; }
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
				background: #FAF8F2;
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
		.venue-name { font-size: 30rpx; font-weight: 600; color: #231C0B; margin-bottom: 6rpx; }
		.venue-address { font-size: 24rpx; color: #9A8C67; margin-bottom: 8rpx; }
		.venue-meta {
			display: flex;
			gap: 16rpx;
			margin-bottom: 8rpx;
			.meta-item {
				font-size: 22rpx;
				color: #6E6242;
				padding: 4rpx 12rpx;
				background: #FAF8F2;
				border-radius: 6rpx;
			}
		}
		.venue-bottom {
			display: flex;
			gap: 20rpx;
				.venue-distance { font-size: 24rpx; color: #c69200; }
				.venue-checkins { font-size: 24rpx; color: #9A8C67; }
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
	.empty-text { font-size: 28rpx; color: #9A8C67; margin-bottom: 24rpx; }
		.empty-btn {
			padding: 16rpx 40rpx;
			background: #e0ae12;
			border-radius: 12rpx;
			color: #ffffff;
			font-size: 28rpx;
		}
	}
.no-more {
	text-align: center;
	padding: 32rpx;
	font-size: 24rpx;
	color: #9A8C67;
}

.refresh-error-banner { display: flex; align-items: center; justify-content: space-between; gap: 20rpx; margin-bottom: 20rpx; padding: 18rpx 22rpx; border-radius: 14rpx; background: rgba(224, 174, 18, 0.12); color: #8a5b00; font-size: 24rpx; }
.refresh-error-action { flex-shrink: 0; color: #a86f00; font-weight: 600; }
.error-state { gap: 16rpx; padding-left: 32rpx; padding-right: 32rpx; text-align: center; }
.error-text { font-size: 24rpx; line-height: 1.7; color: #6E6242; }
.retry-btn { min-width: 200rpx; height: 76rpx; line-height: 76rpx; margin: 12rpx 0 0; padding: 0 32rpx; border-radius: 38rpx; background: #E0AE12; color: #ffffff; font-size: 28rpx; font-weight: 600; }
.retry-btn::after { display: none; }

.venue-page.dark-mode {
	background: #141109;

	.location-bar,
	.mode-tabs,
	.venue-card {
		background: #1e180d;
	}

	.location-bar .location-info .location-text,
	.venue-card .card-right .venue-name {
		color: #fff7e1;
	}

	.mode-tabs .tab-item,
	.venue-card .card-left .venue-img-wrap .venue-img-placeholder,
	.venue-card .card-right .venue-meta .meta-item {
		background: #2a2110;
		color: #d7c89b;
	}

	.mode-tabs .tab-item.active {
		background: rgba(224, 174, 18, 0.18);
		border-color: rgba(224, 174, 18, 0.42);
		color: #f7e7a8;
	}

	.loading-text,
	.venue-address,
	.venue-checkins,
	.empty-text,
	.no-more {
		color: #9f926e;
	}
}
</style>
