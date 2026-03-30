<template>
	<view class="opponent-record-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 需要登录状态 -->
			<view class="login-required" v-if="needLogin">
			<uni-icons type="locked" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
			<text class="hint-text">{{ viewModel.loginHint }}</text>
			<button class="login-btn" @click="goLogin">去登录</button>
		</view>

		<template v-else>
			<!-- 搜索框 -->
			<view class="search-section">
				<view class="search-box">
					<uni-icons type="search" size="20" color="#9ca3af"></uni-icons>
					<input
						type="text"
						v-model="searchKeyword"
						class="search-input"
						:placeholder="viewModel.searchPlaceholder"
						@confirm="handleSearch"
					/>
				</view>
			</view>

		<!-- 统计卡片 -->
		<view class="stats-card">
			<view class="stats-content">
				<view class="stat-item">
					<text class="stat-label">总对手数</text>
					<text class="stat-value">{{ statsData.totalOpponents }}</text>
				</view>
				<view class="stat-divider"></view>
				<view class="stat-item">
					<text class="stat-label">总胜场</text>
					<text class="stat-value">{{ statsData.totalWins }}</text>
				</view>
			</view>
			<view class="stats-icon">
				<uni-icons type="person" size="24" color="#93c5fd"></uni-icons>
			</view>
		</view>

		<!-- 对手列表 -->
		<scroll-view
			class="opponent-list"
			scroll-y
			:refresher-enabled="true"
			:refresher-triggered="isRefreshing"
			@refresherrefresh="onRefresh"
			@scrolltolower="onLoadMore"
		>
			<!-- 加载中状态 -->
			<view class="loading-wrapper" v-if="isLoading && opponentList.length === 0">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 空状态 -->
			<view class="empty-wrapper" v-else-if="!isLoading && opponentList.length === 0">
				<uni-icons type="contact" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="empty-text">{{ viewModel.emptyText }}</text>
				<text class="empty-hint">{{ viewModel.emptyHint }}</text>
			</view>

			<!-- 对手列表 -->
			<view class="opponent-items" v-else>
				<view
					class="opponent-item"
					v-for="opponent in opponentList"
					:key="opponent.name"
					@click="handleOpponentDetail(opponent)"
				>
					<view class="opponent-left">
						<!-- 头像 -->
						<view class="avatar-wrapper">
							<image
								v-if="opponent.avatar"
								class="avatar"
								:src="opponent.avatar"
								mode="aspectFill"
							/>
							<view v-else class="avatar-placeholder">
								<text class="avatar-text">{{ getAvatarText(opponent.name) }}</text>
							</view>
							<!-- 高胜率标记 -->
							<view class="star-badge" v-if="opponent.win_rate >= 60">
								<uni-icons type="star-filled" size="10" color="#ffffff"></uni-icons>
							</view>
						</view>
						<!-- 对手信息 -->
						<view class="opponent-info">
							<text class="opponent-name">{{ opponent.name }}</text>
							<text class="last-match">上次对局: {{ formatLastMatch(opponent.last_match_at) }}</text>
						</view>
					</view>
					<!-- 胜负统计 -->
					<view class="opponent-right">
						<view class="win-rate-row">
							<text class="win-rate" :class="getWinRateClass(opponent.win_rate)">{{ opponent.win_rate }}%</text>
							<view class="win-rate-badge" :class="getWinRateClass(opponent.win_rate)">
								<text>胜率</text>
							</view>
						</view>
						<text class="win-loss">{{ opponent.wins }} 胜 / {{ opponent.losses }} 负</text>
					</view>
				</view>
			</view>

			<!-- 底部加载状态 -->
			<view class="load-more" v-if="opponentList.length > 0">
				<text v-if="isLoadingMore" class="load-more-text">加载中...</text>
				<text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
			</view>
		</scroll-view>
		</template>
	</view>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { getOpponentList } from '@/api/match.js'
import { formatRelativeTime } from '@/utils/format.js'
import {
	buildOpponentH2HUrl,
	buildOpponentRecordRequestParams,
	buildOpponentRecordViewModel,
	normalizeOpponentRecordOptions
} from '@/utils/opponent-record.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 响应式数据 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)
const isLoading = ref(false)
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const searchKeyword = ref('')
const needLogin = ref(false)
const targetUserId = ref(0)
const targetName = ref('')
const targetAvatar = ref('')

const opponentList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

const statsData = reactive({
	totalOpponents: 0,
	totalWins: 0
})

const viewModel = computed(() => buildOpponentRecordViewModel({
	targetName: targetName.value,
	targetUserId: targetUserId.value
}))

// ========== 生命周期 ==========
// ========== 生命周期 ==========
onLoad((options) => {
	const normalized = normalizeOpponentRecordOptions(options)
	targetUserId.value = normalized.targetUserId
	targetName.value = normalized.targetName
	targetAvatar.value = normalized.targetAvatar

	uni.setNavigationBarTitle({
		title: buildOpponentRecordViewModel({
			targetName: normalized.targetName,
			targetUserId: normalized.targetUserId
		}).navigationTitle
	})
})

onShow(() => {
	// 同步主题状态并更新导航栏
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	
	// 检查登录状态
	const token = uni.getStorageSync('token')
	if (!token) {
		needLogin.value = true
		return
	}
	
	// 已登录，重置状态并加载数据
	needLogin.value = false
	// 列表为空或者需要刷新时加载数据
	if (opponentList.value.length === 0) {
		fetchOpponentList()
	}
})

// ========== 方法 ==========

/**
 * 获取对手列表
 */
const fetchOpponentList = async (isRefresh = false, isLoadMore = false) => {
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
		const res = await getOpponentList(buildOpponentRecordRequestParams({
			page: currentPage.value,
			pageSize,
			keyword: searchKeyword.value,
			targetUserId: targetUserId.value
		}))

		const list = res.list || []
		total.value = res.total || 0
		statsData.totalOpponents = res.total_opponents || 0
		statsData.totalWins = res.total_wins || 0

		if (isRefresh) {
			opponentList.value = list
		} else if (isLoadMore) {
			opponentList.value = [...opponentList.value, ...list]
		} else {
			opponentList.value = list
		}

		// 判断是否还有更多数据
		hasMore.value = opponentList.value.length < total.value
	} catch (error) {
		console.error('获取对手列表失败:', error)
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
 * 搜索
 */
const handleSearch = () => {
	currentPage.value = 1
	hasMore.value = true
	opponentList.value = [] // 清空列表
	fetchOpponentList()
}

/**
 * 下拉刷新
 */
const onRefresh = () => {
	searchKeyword.value = ''
	fetchOpponentList(true, false)
}

/**
 * 上拉加载更多
 */
const onLoadMore = () => {
	fetchOpponentList(false, true)
}

/**
 * 格式化上次对局时间
 */
const formatLastMatch = (time) => {
	return formatRelativeTime(time)
}

/**
 * 获取头像占位符文本
 */
const getAvatarText = (name) => {
	if (!name) return '?'
	return name.substring(0, 2).toUpperCase()
}

/**
 * 获取胜率样式类
 */
const getWinRateClass = (winRate) => {
	if (winRate >= 60) return 'high'
	if (winRate >= 40) return 'medium'
	return 'low'
}

/**
 * 查看对手详情
 */
const handleOpponentDetail = (opponent) => {
	uni.navigateTo({
		url: buildOpponentH2HUrl({
			opponent,
			targetUserId: targetUserId.value,
			targetName: targetName.value,
			targetAvatar: targetAvatar.value
		})
	})
}
/**
 * 跳转登录页
 */
const goLogin = () => {
	uni.navigateTo({
		url: '/pages/login/login'
	})
}
</script>

<style lang="scss" scoped>
@import './opponentRecord.scss';
</style>
