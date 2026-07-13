<template>
	<view class="venue-detail-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="venue" class="detail-content">
			<view class="hero-card">
				<view class="hero-copy">
					<text class="hero-eyebrow">球馆档案</text>
					<text class="hero-title">{{ venue.name }}</text>
					<text class="hero-desc">{{ venue.description || '该球馆暂未填写简介，可先查看地址、营业时间和签到情况。' }}</text>
				</view>
				<view class="hero-stats">
					<view class="hero-stat">
						<text class="hero-stat-label">营业时间</text>
						<text class="hero-stat-value">{{ venue.business_hours || '暂未填写' }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">球桌数量</text>
						<text class="hero-stat-value">{{ venue.table_count ? `${venue.table_count} 台` : '暂未填写' }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">签到人数</text>
						<text class="hero-stat-value">{{ venue.checkin_count || 0 }}</text>
					</view>
				</view>
			</view>

			<view class="image-section" v-if="venue.images && venue.images.length > 0">
				<swiper class="image-swiper" indicator-dots circular autoplay>
					<swiper-item v-for="(img, idx) in venue.images" :key="idx">
						<image class="swiper-img" :src="img" mode="aspectFill"></image>
					</swiper-item>
				</swiper>
			</view>
			<view v-else class="image-placeholder">
				<view class="placeholder-badge">
					<uni-icons type="location" size="28" color="#0f766e"></uni-icons>
				</view>
				<text class="placeholder-title">暂无球馆图片</text>
				<text class="placeholder-text">先通过地址、营业信息和签到数据了解这个球馆。</text>
			</view>

			<view class="info-card">
				<view class="section-head">
					<text class="section-title">基础信息</text>
					<text class="section-tip">{{ venue.city || '未标注城市' }}</text>
				</view>
				<view class="info-row">
					<uni-icons type="location" size="16" color="#94a3b8"></uni-icons>
					<view class="info-copy">
						<text class="info-label">详细地址</text>
						<text class="info-text">{{ venue.address || '暂无地址' }}</text>
					</view>
				</view>
				<view class="info-row" v-if="venue.phone">
					<uni-icons type="phone" size="16" color="#94a3b8"></uni-icons>
					<view class="info-copy">
						<text class="info-label">联系电话</text>
						<text class="info-text">{{ venue.phone }}</text>
					</view>
				</view>
				<view class="info-row" v-if="venue.business_hours">
					<uni-icons type="calendar" size="16" color="#94a3b8"></uni-icons>
					<view class="info-copy">
						<text class="info-label">营业时间</text>
						<text class="info-text">{{ venue.business_hours }}</text>
					</view>
				</view>
				<view class="info-tags">
					<view class="tag" v-if="venue.table_count">
						<text>{{ venue.table_count }}台球桌</text>
					</view>
					<view class="tag" v-if="venue.price_range">
						<text>{{ venue.price_range }}</text>
					</view>
					<view class="tag" v-if="venue.city">
						<text>{{ venue.city }}</text>
					</view>
				</view>
			</view>

			<view class="checkin-card">
				<view class="section-head">
					<text class="section-title">签到动态</text>
					<text class="section-tip" v-if="venue.checkin_count">{{ venue.checkin_count }}人到访</text>
				</view>
				<view class="checkin-users" v-if="recentCheckins.length > 0">
					<view v-for="item in recentCheckins" :key="item.id" class="checkin-user">
						<image
							class="checkin-avatar"
							:src="item.avatar || '/static/default-avatar.png'"
							mode="aspectFill"
						></image>
					</view>
				</view>
				<text v-else class="empty-checkin">还没有人来签到，来当第一个打卡的人吧。</text>
			</view>

			<view class="action-bar">
				<view class="action-btn nav-btn" @tap="openNavigation">
					<uni-icons type="navigate" size="20" color="#E0AE12"></uni-icons>
					<text class="nav-text">去导航</text>
				</view>
				<view class="action-btn call-btn" @tap="callPhone" v-if="venue.phone">
					<uni-icons type="phone" size="20" color="#0f766e"></uni-icons>
					<text class="call-text">打电话</text>
				</view>
				<view class="action-btn checkin-btn" :class="{ disabled: hasCheckedIn }" @tap="doCheckin">
					<text class="checkin-text">{{ hasCheckedIn ? '今日已签到' : '签到打卡' }}</text>
				</view>
			</view>
		</view>

		<view v-else class="empty-state">
			<text>球馆不存在</text>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getVenueDetail, checkinVenue } from '@/api/venue.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const venue = ref(null)
const loading = ref(true)
const venueId = ref(0)
const hasCheckedIn = ref(false)
const recentCheckins = ref([])

const fetchDetail = async () => {
	loading.value = true
	try {
		const res = await getVenueDetail({ venue_id: venueId.value })
		if (res.success) {
			venue.value = res.venue || null
			recentCheckins.value = res.recent_checkins || []
			hasCheckedIn.value = res.has_checked_in || false
		}
	} catch (e) {
		console.error('获取球馆详情失败', e)
	} finally {
		loading.value = false
	}
}

const doCheckin = async () => {
	if (hasCheckedIn.value) {
		uni.showToast({ title: '今日已签到', icon: 'none' })
		return
	}
	try {
		const res = await checkinVenue({ venue_id: venueId.value })
		if (res.success) {
			hasCheckedIn.value = true
			uni.showToast({ title: '签到成功！', icon: 'success' })
			// 刷新详情
			fetchDetail()
		} else {
			uni.showToast({ title: res.msg || '签到失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '签到失败', icon: 'none' })
	}
}

const openNavigation = () => {
	if (!venue.value) return
	uni.openLocation({
		latitude: venue.value.latitude || 0,
		longitude: venue.value.longitude || 0,
		name: venue.value.name,
		address: venue.value.address,
		fail: () => {
			uni.showToast({ title: '打开导航失败', icon: 'none' })
		}
	})
}

const callPhone = () => {
	if (!venue.value || !venue.value.phone) return
	uni.makePhoneCall({ phoneNumber: venue.value.phone })
}

onLoad((options) => {
	venueId.value = Number.parseInt(options?.id || options?.venue_id || '0', 10) || 0
	if (venueId.value > 0) {
		fetchDetail()
	} else {
		loading.value = false
	}
})
</script>

<style lang="scss" scoped>
@import './detail.scss';
</style>
