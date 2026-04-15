<template>
	<view class="match-history-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 需要登录状态 -->
		<view class="login-required" v-if="needLogin">
			<uni-icons type="locked" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
			<text class="hint-text">请登录后查看对局记录</text>
			<button class="login-btn" @click="goLogin">去登录</button>
		</view>

		<!-- 列表内容区 -->
		<scroll-view 
			v-else
			class="match-list" 
			scroll-y 
			:refresher-enabled="true"
			:refresher-triggered="isRefreshing"
			@refresherrefresh="onRefresh"
			@scrolltolower="onLoadMore"
		>
			<!-- 加载中状态 -->
			<view class="loading-wrapper" v-if="isLoading && matchList.length === 0">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 空状态 -->
			<view class="empty-wrapper" v-else-if="!isLoading && matchList.length === 0">
				<uni-icons type="list" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="empty-text">暂无对局记录</text>
				<text class="empty-hint">快去发起一场PK吧！</text>
			</view>

			<!-- 对局记录列表 -->
			<view class="match-items" v-else>
				<view 
					class="match-item" 
					v-for="match in matchList" 
					:key="match.id"
					@click="handleMatchDetail(match)"
				>
					<!-- 对手头像 -->
					<view class="avatar-wrapper">
						<image 
							class="avatar" 
							:src="match.opponent_avatar || '/static/images/default-avatar.png'" 
							mode="aspectFill"
						/>
					</view>

					<!-- 对局信息 -->
					<view class="match-info">
						<view class="name-row">
							<text class="opponent-name">{{ match.opponent_name }}</text>
							<view :class="['game-type-tag', getGameTypeClass(match.game_type)]">
								<text>{{ match.game_type_name }}</text>
							</view>
						</view>
						<text class="match-time">{{ formatMatchTime(match.match_time) }}</text>
					</view>

					<!-- 比分和结果 -->
					<view class="match-result">
						<view class="result-badge" :class="getResultClass(match.result)">
							<text class="result-text">{{ getResultText(match.result) }}</text>
						</view>
						<text class="score">{{ match.my_score }} - {{ match.opponent_score }}</text>
					</view>

					<!-- 右箭头 -->
					<view class="arrow-wrapper">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					</view>
				</view>
			</view>

			<!-- 底部加载状态 -->
			<view class="load-more" v-if="matchList.length > 0">
				<text v-if="isLoadingMore" class="load-more-text">加载中...</text>
				<text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { getMatchList } from '@/api/match.js'
import { formatRelativeTime } from '@/utils/format.js'

// ========== 状态管理 ==========
const { isDarkMode } = usePageTheme()

// ========== 响应式数据 ==========
const isLoading = ref(false)
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const needLogin = ref(false)

const matchList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

// ========== 生命周期 ==========
onShow(() => {
	// 同步主题状态并更新导航栏
	
	// 检查登录状态
	const token = uni.getStorageSync('token')
	if (!token) {
		needLogin.value = true
		return
	}
	
	// 已登录，重置状态并加载数据
	needLogin.value = false
	if (matchList.value.length === 0 && !isLoading.value) {
		fetchMatchList()
	}
})

// ========== 方法 ==========

/**
 * 获取对局列表
 */
const fetchMatchList = async (isRefresh = false, isLoadMore = false) => {
	if (isLoading.value || isLoadingMore.value) return
	
	if (isRefresh) {
		isRefreshing.value = true
		currentPage.value = 1
		hasMore.value = true
	} else if (isLoadMore) {
		if (!hasMore.value) return
		isLoadingMore.value = true
		currentPage.value++
	} else {
		isLoading.value = true
	}
	
	try {
		const res = await getMatchList({
			page: currentPage.value,
			page_size: pageSize
		})
		
		const list = res.list || []
		total.value = res.total || 0
		
		if (isRefresh) {
			matchList.value = list
		} else if (isLoadMore) {
			matchList.value = [...matchList.value, ...list]
		} else {
			matchList.value = list
		}
		
		// 判断是否还有更多数据
		hasMore.value = matchList.value.length < total.value
	} catch (error) {
		console.error('获取对局列表失败:', error)
		uni.showToast({
			title: error.message || '获取数据失败',
			icon: 'none'
		})
	} finally {
		isLoading.value = false
		isRefreshing.value = false
		isLoadingMore.value = false
	}
}

/**
 * 下拉刷新
 */
const onRefresh = () => {
	fetchMatchList(true, false)
}

/**
 * 上拉加载更多
 */
const onLoadMore = () => {
	fetchMatchList(false, true)
}

/**
 * 格式化对局时间
 */
const formatMatchTime = (time) => {
	return formatRelativeTime(time)
}

/**
 * 获取结果样式类名
 */
const getResultClass = (result) => {
	switch (result) {
		case 1:
			return 'win'
		case 2:
			return 'lose'
		case 3:
			return 'draw'
		default:
			return ''
	}
}

/**
 * 获取结果文本
 */
const getResultText = (result) => {
	switch (result) {
		case 1:
			return '胜利'
		case 2:
			return '失败'
		case 3:
			return '平局'
		default:
			return '未知'
	}
}

/**
 * 获取游戏类型样式类
 */
const getGameTypeClass = (gameType) => {
	switch (gameType) {
		case 1:
			return 'snooker'
		case 2:
		case 4:
			return 'american-9ball'
		case 3:
		default:
			return 'chinese-8ball'
	}
}

/**
 * 跳转登录页
 */
const goLogin = () => {
	uni.navigateTo({
		url: '/pages/login/login'
	})
}

/**
 * 查看对局详情
 */
const handleMatchDetail = (match) => {
	uni.navigateTo({
		url: `/subPages/match/matchResult?match_id=${match.id}&from=history`
	})
}
</script>

<style lang="scss" scoped>
@import './matchHistory.scss';
</style>
