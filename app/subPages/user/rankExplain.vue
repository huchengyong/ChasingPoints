<template>
	<view class="rank-explain-container" :class="{ 'dark-mode': isDarkMode }">
		<scroll-view class="content-scroll" scroll-y>
			<!-- 加载状态 -->
			<view class="loading-wrapper" v-if="isInitialLoading">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<template v-else>
				<view class="game-type-tabs-row">
					<scroll-view class="game-type-tabs" scroll-x :show-scrollbar="false" enable-flex>
						<view
							v-for="item in gameTypeTabs"
							:key="item.value"
							class="game-type-tab"
							:class="{ active: currentGameType === item.value, pending: isRefreshing && currentGameType === item.value }"
							@tap="handleGameTypeChange(item.value)"
						>
							<text>{{ item.label }}</text>
						</view>
					</scroll-view>
				</view>

				<!-- 当前段位卡片 -->
				<view class="current-rank-card">
					<view class="rank-main">
						<image class="rank-icon" :src="rankInfo.icon" mode="aspectFit"></image>
						<view class="rank-details">
							<text class="rank-label">当前段位</text>
							<text class="rank-name">{{ rankInfo.name }}</text>
							<text class="rank-score">当前排位分: {{ rankInfo.rank_score }}</text>
						</view>
						<view class="rank-main-refresh" v-if="isRefreshing">
							<uni-icons type="spinner-cycle" size="16" :color="isDarkMode ? '#94a3b8' : '#64748b'"></uni-icons>
						</view>
					</view>
					
					<!-- 晋级进度 -->
					<view class="progress-section" v-if="rankInfo.level < 5">
						<view class="progress-header">
							<text class="progress-title">晋级{{ rankInfo.next_name }}</text>
							<text class="progress-value">{{ rankInfo.rank_score }} / {{ rankInfo.next_score }}</text>
						</view>
						<view class="progress-bar">
							<view class="progress-fill" :style="{ width: rankInfo.progress + '%' }"></view>
						</view>
						<text class="progress-hint">还差 {{ rankInfo.next_score - rankInfo.rank_score }} 排位分即可晋级</text>
					</view>
					<view class="max-rank-badge" v-else>
						<uni-icons type="star-filled" size="16" color="#f59e0b"></uni-icons>
						<text class="max-rank-text">已达到最高段位</text>
					</view>
				</view>

				<!-- 提示语 -->
				<text class="tip-text">提升段位，证明你的台球实力</text>

				<view class="rank-list">
					<view class="rank-item">
						<view class="rank-item-header">
							<view class="rank-item-left">
								<text class="rank-item-name">段位分规则</text>
							</view>
						</view>
						<view class="rank-item-content" style="display: block;">
							<text class="promotion-condition">- 胜利基础分：+20</text>
							<text class="promotion-condition">- 失败基础分：-10</text>
							<text class="promotion-condition">- 失败时特殊战绩最多只能把扣分减免到 -2</text>
							<text class="promotion-condition">- 每个球种每天最多上涨 300 排位分</text>
							<text class="promotion-condition">- 同一对手前 2 场正常，第 3 场 80%，第 4-6 场 30%</text>
							<text class="promotion-condition">- 同一对手第 7 场起不计排位分，也不计胜率</text>
						</view>
					</view>
				</view>

				<!-- 段位列表 -->
				<view class="rank-list">
					<view 
						class="rank-item" 
						v-for="item in rankList" 
						:key="item.level"
						:class="{ 'is-current': item.is_current }"
						@click="toggleExpand(item.level)"
					>
						<view class="rank-item-header">
							<view class="rank-item-left">
								<image class="rank-item-icon" :src="item.icon" mode="aspectFit"></image>
								<text class="rank-item-name">{{ item.name }}</text>
							</view>
							<uni-icons 
								:type="expandedLevel === item.level ? 'up' : 'down'" 
								size="20" 
								:color="isDarkMode ? '#64748b' : '#94a3b8'"
							></uni-icons>
						</view>
						
						<!-- 展开内容 -->
						<view class="rank-item-content" v-if="expandedLevel === item.level">
							<text class="promotion-label">晋升条件:</text>
							<text class="promotion-condition">- 排位分达到: {{ item.min_score }}</text>
							<text class="promotion-condition" v-if="item.is_current">- 你当前处于这个段位</text>
						</view>
					</view>
				</view>
			</template>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { getUserRankInfo, getRankList } from '@/api/rank.js'
import { GAME_TYPE_TABS } from '@/utils/game-types.js'
import { resolveRankExplainLoadingMode, shouldApplyRankExplainResponse } from '@/utils/rank-explain.js'

// ========== 状态管理 ==========
const { isDarkMode } = usePageTheme()

// ========== 响应式数据 ==========
const isFetching = ref(true)
const hasLoadedOnce = ref(false)
const expandedLevel = ref(0)
const currentGameType = ref(3)
const gameTypeTabs = GAME_TYPE_TABS
const latestRequestId = ref(0)
const loadingMode = computed(() => resolveRankExplainLoadingMode({
	hasLoadedOnce: hasLoadedOnce.value,
	isFetching: isFetching.value
}))
const isInitialLoading = computed(() => loadingMode.value === 'initial')
const isRefreshing = computed(() => loadingMode.value === 'refreshing')

const rankInfo = ref({
	level: 1,
	name: '青铜球手',
	icon: '/static/images/ranks/rank_bronze.png',
	rank_score: 0,
	total_wins: 0,
	total_losses: 0,
	max_streak: 0,
	next_level: 2,
	next_name: '白银球手',
	next_score: 500,
	progress: 0
})

const rankList = ref([])

onLoad((options) => {
	const gameType = Number(options?.game_type || 0)
	if (gameType > 0) {
		currentGameType.value = gameType
	}
})

// ========== 生命周期 ==========
onMounted(() => {
	fetchData()
})

onShow(() => {
	// 同步主题状态并更新导航栏
})

// ========== 方法 ==========

/**
 * 获取数据
 */
const fetchData = async () => {
	const requestId = latestRequestId.value + 1
	latestRequestId.value = requestId
	isFetching.value = true
	
	try {
		// 并行请求用户段位信息和段位列表
		const [infoRes, listRes] = await Promise.all([
			getUserRankInfo({ game_type: currentGameType.value }),
			getRankList({ game_type: currentGameType.value })
		])
		
		if (!shouldApplyRankExplainResponse({ requestId, latestRequestId: latestRequestId.value })) {
			return
		}

		if (infoRes.success && infoRes.rank_info) {
			rankInfo.value = infoRes.rank_info
			// 默认展开当前段位
			expandedLevel.value = infoRes.rank_info.level
		}
		
		if (listRes.success && listRes.list) {
			rankList.value = listRes.list
		}

		hasLoadedOnce.value = true
	} catch (error) {
		if (!shouldApplyRankExplainResponse({ requestId, latestRequestId: latestRequestId.value })) {
			return
		}

		console.error('获取段位数据失败:', error)
		uni.showToast({
			title: error.message || '获取数据失败',
			icon: 'none'
		})
	} finally {
		if (shouldApplyRankExplainResponse({ requestId, latestRequestId: latestRequestId.value })) {
			isFetching.value = false
		}
	}
}

/**
 * 切换展开状态
 */
const toggleExpand = (level) => {
	if (expandedLevel.value === level) {
		expandedLevel.value = 0
	} else {
		expandedLevel.value = level
	}
}

const handleGameTypeChange = (gameType) => {
	if (currentGameType.value === gameType) return
	currentGameType.value = gameType
	fetchData()
}
</script>

<style lang="scss" scoped>
@import './rankExplain.scss';
</style>
