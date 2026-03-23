<template>
	<view class="h2h-record-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 顶部统计卡片 -->
		<view class="h2h-summary-card" v-if="!isLoading">
			<!-- 头像对比 -->
			<view class="avatar-group">
				<view class="avatar-section">
					<image
						v-if="myAvatar"
						class="avatar"
						:src="myAvatar"
						mode="aspectFill"
					/>
					<view v-else class="avatar-placeholder">
						<text class="avatar-text">我</text>
					</view>
					<text class="name">你</text>
				</view>
				<view class="vs-section">
					<text class="vs-text">VS</text>
				</view>
				<view class="avatar-section">
					<image
						v-if="opponentData.avatar"
						class="avatar"
						:src="opponentData.avatar"
						mode="aspectFill"
					/>
					<view v-else class="avatar-placeholder">
						<text class="avatar-text">{{ getAvatarText(opponentData.name) }}</text>
					</view>
					<text class="name">{{ opponentData.name }}</text>
				</view>
			</view>

			<!-- 总战绩 -->
			<view class="total-score">
				<text class="score-text">{{ statsData.myWins }} : {{ statsData.opponentWins }}</text>
			</view>

			<!-- 胜率描述 -->
			<view class="win-rate-desc">
				<text class="desc-text">你对{{ opponentData.name }}的胜率是{{ statsData.winRate.toFixed(2) }}%</text>
			</view>

			<!-- 统计数据 -->
			<view class="stats-grid">
				<view class="stat-item">
					<text class="stat-label">总场次</text>
					<text class="stat-value">{{ statsData.totalMatches }}</text>
				</view>
				<view class="stat-item">
					<text class="stat-label">平均分差</text>
					<text class="stat-value">{{ statsData.avgScoreDiff > 0 ? '+' : '' }}{{ statsData.avgScoreDiff.toFixed(1) }}</text>
				</view>
				<view class="stat-item">
					<text class="stat-label">最长连胜</text>
					<text class="stat-value">{{ statsData.maxWinStreak }}</text>
				</view>
			</view>
		</view>

		<!-- 筛选按钮组 -->
		<view class="filter-bar" v-if="!isLoading">
			<view class="filter-buttons">
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 0 }"
					@click="handleFilterChange(0)"
				>
					<text class="btn-text">全部</text>
				</button>
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 1 }"
					@click="handleFilterChange(1)"
				>
					<text class="btn-text">胜利</text>
				</button>
				<button
					class="filter-btn"
					:class="{ active: currentFilter === 2 }"
					@click="handleFilterChange(2)"
				>
					<text class="btn-text">失败</text>
				</button>
			</view>
		</view>

		<!-- 历史列表标题 -->
		<view class="history-header" v-if="!isLoading && historyList.length > 0">
			<text class="header-text">比赛历史</text>
		</view>

		<!-- 历史对局列表 -->
		<scroll-view
			class="history-list"
			scroll-y
			@scrolltolower="onLoadMore"
		>
			<!-- 加载中状态 -->
			<view class="loading-wrapper" v-if="isLoading">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 空状态 -->
			<view class="empty-wrapper" v-else-if="historyList.length === 0">
				<uni-icons type="list" size="64" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="empty-text">暂无对局记录</text>
			</view>

			<!-- 历史记录列表 -->
			<view v-else>
				<view
					class="history-item"
					:class="getResultClass(item.result)"
					v-for="item in historyList"
					:key="item.id"
				>
					<view class="item-left">
						<text class="match-date">{{ formatMatchDate(item.match_time) }}</text>
						<text class="game-type">{{ item.game_type_name }}</text>
					</view>
					<view class="item-right">
						<text class="score">{{ item.my_score }} - {{ item.opponent_score }}</text>
						<view class="result-wrapper">
							<text class="result-text" :class="getResultClass(item.result)">{{ getResultText(item.result) }}</text>
						</view>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#64748b' : '#9ca3af'"></uni-icons>
					</view>
				</view>

				<!-- 底部加载状态 -->
				<view class="load-more" v-if="historyList.length > 0">
					<text v-if="isLoadingMore" class="load-more-text">加载中...</text>
					<text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { useUserStore } from '@/store/user.js'
import { getH2HStats, getH2HHistory } from '@/api/match.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()
const userStore = useUserStore()

// ========== 响应式数据 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)
const isLoading = ref(true)
const isLoadingMore = ref(false)
const hasMore = ref(true)

const opponentId = ref(0)
const opponentName = ref('')
const myAvatar = ref('')
const currentFilter = ref(0) // 0=全部, 1=胜利, 2=失败

const opponentData = reactive({
	id: 0,
	name: '',
	avatar: ''
})

const statsData = reactive({
	totalMatches: 0,
	myWins: 0,
	opponentWins: 0,
	winRate: 0,
	avgScoreDiff: 0,
	maxWinStreak: 0
})

const historyList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

// ========== 生命周期 ==========
onLoad((options) => {
	if (options.opponent_id) {
		opponentId.value = parseInt(options.opponent_id) || 0
	}
	if (options.opponent_name) {
		opponentName.value = decodeURIComponent(options.opponent_name)
	}
	
	// 设置页面标题
	uni.setNavigationBarTitle({
		title: `交锋记录 - ${opponentName.value || '对手'}`
	})
})

onMounted(() => {
	loadUserInfo()
	fetchData()
})

onShow(() => {
	// 同步主题状态并更新导航栏
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
})

// ========== 方法 ==========

/**
 * 加载用户信息获取头像
 */
const loadUserInfo = () => {
	if (userStore.userInfo && userStore.userInfo.avatar) {
		myAvatar.value = userStore.userInfo.avatar
	}
}

/**
 * 获取数据
 */
const fetchData = async () => {
	isLoading.value = true
	
	try {
		// 同时获取统计和历史数据
		await Promise.all([
			fetchStats(),
			fetchHistory(false, false)
		])
	} catch (error) {
		console.error('获取数据失败:', error)
	} finally {
		isLoading.value = false
	}
}

/**
 * 获取交锋统计
 */
const fetchStats = async () => {
	try {
		const params = {}
		if (opponentId.value > 0) {
			params.opponent_id = opponentId.value
		} else if (opponentName.value) {
			params.opponent_name = opponentName.value
		}
		
		const res = await getH2HStats(params)
		
		if (res.opponent) {
			opponentData.id = res.opponent.id || 0
			opponentData.name = res.opponent.name || opponentName.value
			opponentData.avatar = res.opponent.avatar || ''
		} else {
			opponentData.name = opponentName.value
		}
		
		if (res.stats) {
			statsData.totalMatches = res.stats.total_matches || 0
			statsData.myWins = res.stats.my_wins || 0
			statsData.opponentWins = res.stats.opponent_wins || 0
			statsData.winRate = res.stats.win_rate || 0
			statsData.avgScoreDiff = res.stats.avg_score_diff || 0
			statsData.maxWinStreak = res.stats.max_win_streak || 0
		}
	} catch (error) {
		console.error('获取交锋统计失败:', error)
		// 使用传入的对手名称
		opponentData.name = opponentName.value
		throw error
	}
}

/**
 * 获取交锋历史
 */
const fetchHistory = async (isRefresh = false, isLoadMore = false) => {
	if (isLoadingMore.value) return

	if (isRefresh) {
		currentPage.value = 1
		hasMore.value = true
	} else if (isLoadMore) {
		if (!hasMore.value) return
		isLoadingMore.value = true
		currentPage.value++
	}

	try {
		const params = {
			opponent_id: opponentId.value || opponentData.id,
			page: currentPage.value,
			page_size: pageSize
		}
		
		// 添加筛选条件
		if (currentFilter.value > 0) {
			params.result = currentFilter.value
		}

		const res = await getH2HHistory(params)

		const list = res.list || []
		total.value = res.total || 0

		if (isRefresh || !isLoadMore) {
			historyList.value = list
		} else {
			historyList.value = [...historyList.value, ...list]
		}

		hasMore.value = historyList.value.length < total.value
	} catch (error) {
		console.error('获取交锋历史失败:', error)
		if (!isLoadMore) {
			historyList.value = []
		}
	} finally {
		isLoadingMore.value = false
	}
}

/**
 * 筛选切换
 */
const handleFilterChange = (filter) => {
	if (currentFilter.value === filter) return
	currentFilter.value = filter
	currentPage.value = 1
	hasMore.value = true
	historyList.value = []
	fetchHistory(true, false)
}

/**
 * 加载更多
 */
const onLoadMore = () => {
	fetchHistory(false, true)
}

/**
 * 获取头像占位文字
 */
const getAvatarText = (name) => {
	if (!name) return '?'
	return name.substring(0, 1).toUpperCase()
}

/**
 * 格式化对局日期
 */
const formatMatchDate = (dateStr) => {
	if (!dateStr) return ''
	const date = new Date(dateStr)
	const year = date.getFullYear()
	const month = date.getMonth() + 1
	const day = date.getDate()
	return `${year}年${month}月${day}日`
}

/**
 * 获取结果样式类
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
</script>

<style lang="scss" scoped>
@import './h2hRecord.scss';
</style>
