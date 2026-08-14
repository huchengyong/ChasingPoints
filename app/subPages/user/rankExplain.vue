<template>
	<view class="rank-explain-container" :class="{ 'dark-mode': isDarkMode }">
		<scroll-view class="content-scroll" scroll-y>
			<!-- 吸顶球种切换栏 -->
			<view class="game-tabs-wrap">
				<view class="game-tabs">
					<view
						v-for="item in gameTypeTabs"
						:key="item.value"
						class="game-tab"
						:class="{ active: currentGameType === item.value, pending: isRefreshing && currentGameType === item.value }"
						@tap="handleGameTypeChange(item.value)"
					>
						<text class="game-tab-text">{{ item.label }}</text>
					</view>
				</view>
			</view>

			<!-- 首次加载 -->
			<view class="loading-wrapper" v-if="isInitialLoading">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 首次加载失败 -->
			<view class="error-state" v-else-if="isInitialError">
				<view class="error-icon">
					<uni-icons type="info-filled" size="28" color="#b77908"></uni-icons>
				</view>
				<text class="error-title">段位数据加载失败</text>
				<text class="error-desc">请检查网络连接后重新加载</text>
				<button class="retry-button" hover-class="none" @tap="handleRetry">重新加载</button>
			</view>

			<template v-else-if="rankInfo && heroModel">
				<!-- 当前段位英雄卡 -->
				<view class="rank-hero" :class="`level-${heroModel.level}`">
					<view class="hero-orbit"></view>
					<view class="hero-main">
						<view class="hero-copy">
							<text class="hero-kicker">{{ currentGameTypeLabel }} · 当前段位</text>
							<text class="hero-name">{{ heroModel.name }}</text>
							<view class="hero-score">
								<text class="hero-score-value">{{ heroModel.rankScore }}</text>
								<text class="hero-score-unit">排位分</text>
							</view>
						</view>
						<image class="hero-badge" :src="heroModel.icon" mode="aspectFit"></image>
					</view>

					<!-- 晋级进度（等级 1–5） -->
					<view class="hero-progress" v-if="rankInfo.level < 6">
						<view class="progress-copy">
							<text class="progress-title">晋级{{ heroModel.nextName }}</text>
							<text class="progress-value">{{ heroModel.earned }} / {{ heroModel.span }}</text>
						</view>
						<view class="progress-track">
							<view class="progress-fill" :style="{ width: heroModel.progressPercent + '%' }"></view>
						</view>
						<view class="progress-footer">
							<text class="progress-hint">本段位进度 {{ heroModel.progressPercent }}%</text>
							<text class="next-pill">还差 {{ heroModel.remaining }} 分</text>
						</view>
					</view>
					<!-- 王者满级（等级 6） -->
					<view class="hero-progress" v-else>
						<view class="progress-copy">
							<text class="progress-title">最高段位成就</text>
							<text class="progress-value">MAX</text>
						</view>
						<view class="progress-track">
							<view class="progress-fill" style="width: 100%"></view>
						</view>
						<view class="progress-footer">
							<text class="progress-hint">已解锁全部段位成长节点</text>
							<text class="next-pill">已达最高段位</text>
						</view>
					</view>
				</view>

				<!-- 段位成长路径 -->
				<view class="section">
					<view class="section-heading">
						<view class="section-heading-copy">
							<text class="section-title">段位之路</text>
							<text class="section-subtitle">达到对应排位分即可晋级</text>
						</view>
						<view class="current-legend">
							<text class="current-legend-dot"></text>
							<text class="current-legend-text">我的段位</text>
						</view>
					</view>
					<view class="rank-path">
						<view
							v-for="item in growthPath"
							:key="item.level"
							class="rank-step"
							:class="{ 'is-current': item.state === 'current', 'is-reached': item.state === 'reached' }"
						>
							<view class="rank-node">
								<uni-icons v-if="item.state === 'reached'" type="checkmarkempty" size="10" color="#ffffff"></uni-icons>
							</view>
							<image class="step-icon" :src="item.icon" mode="aspectFit"></image>
							<view class="step-copy">
								<text class="step-name">{{ item.name }}</text>
								<text class="step-range">{{ item.rangeText }}</text>
							</view>
							<text class="step-state">{{ item.stateText }}</text>
						</view>
					</view>
				</view>

				<!-- 计分规则 -->
				<view class="section">
					<view class="section-heading">
						<view class="section-heading-copy">
							<text class="section-title">段位分怎么算？</text>
							<text class="section-subtitle">先看关键规则，再按需查看细节</text>
						</view>
					</view>
					<view class="rule-stats">
						<view class="rule-stat" v-for="item in ruleSummaryStats" :key="item.label">
							<text class="rule-stat-value">{{ item.value }}</text>
							<text class="rule-stat-label">{{ item.label }}</text>
						</view>
					</view>
					<view class="rule-note">
						<text class="rule-note-text">{{ ruleNote }}</text>
					</view>
					<button
						class="rule-toggle"
						hover-class="none"
						:aria-expanded="String(isRuleDetailsOpen)"
						@tap="toggleRuleDetails"
					>
						<text class="rule-toggle-text">{{ isRuleDetailsOpen ? '收起完整计分规则' : '查看完整计分规则' }}</text>
						<uni-icons :type="isRuleDetailsOpen ? 'up' : 'down'" size="14" :color="isDarkMode ? '#e8c469' : '#8a6309'"></uni-icons>
					</button>
					<view class="rule-details" v-if="isRuleDetailsOpen">
						<view class="rule-row" v-for="row in ruleDetailRows" :key="row.title">
							<text class="rule-row-title">{{ row.title }}</text>
							<text class="rule-row-desc">{{ row.description }}</text>
						</view>
						<view class="example-card">
							<text class="example-title">{{ ruleExample.title }}</text>
							<text class="example-formula">{{ ruleExample.formula }}</text>
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
import { getRankConfigs } from '@/api/rank.js'
import { usePublicReadStore } from '@/store/publicRead.js'
import { useRankStore } from '@/store/rank.js'
import { useUserStore } from '@/store/user.js'
import { GAME_TYPE_TABS, getGameTypeLabel } from '@/utils/game-types.js'
import {
	buildRankGrowthPath,
	buildRankHeroModel,
	RANK_RULE_SUMMARY_STATS,
	RANK_RULE_NOTE,
	RANK_RULE_DETAIL_ROWS,
	RANK_RULE_EXAMPLE,
	resolveRankExplainLoadingMode,
	shouldApplyRankExplainResponse
} from '@/utils/rank-explain.js'

// ========== 状态管理 ==========
const { isDarkMode } = usePageTheme()
const publicReadStore = usePublicReadStore()
const rankStore = useRankStore()
const userStore = useUserStore()
const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

// ========== 响应式数据 ==========
const isFetching = ref(true)
const hasLoadedOnce = ref(false)
const hasLoadError = ref(false)
const isRuleDetailsOpen = ref(false)
const currentGameType = ref(3)
const loadedGameType = ref(3)
const gameTypeTabs = GAME_TYPE_TABS
const latestRequestId = ref(0)
const loadingMode = computed(() => resolveRankExplainLoadingMode({
	hasLoadedOnce: hasLoadedOnce.value,
	isFetching: isFetching.value
}))
const isInitialLoading = computed(() => loadingMode.value === 'initial')
const isRefreshing = computed(() => loadingMode.value === 'refreshing')
const isInitialError = computed(() => !hasLoadedOnce.value && hasLoadError.value && !isFetching.value)

// 首次请求成功前不构造默认段位，避免把青铜数据渲染成真实业务结论
const rankInfo = ref(null)
const rankList = ref([])

const heroModel = computed(() => buildRankHeroModel({
	rankInfo: rankInfo.value,
	rankList: rankList.value
}))
const growthPath = computed(() => buildRankGrowthPath({
	rankList: rankList.value,
	currentLevel: rankInfo.value?.level || 0,
	rankScore: rankInfo.value?.rank_score || 0
}))
const currentGameTypeLabel = computed(() => getGameTypeLabel(loadedGameType.value))

const ruleSummaryStats = RANK_RULE_SUMMARY_STATS
const ruleNote = RANK_RULE_NOTE
const ruleDetailRows = RANK_RULE_DETAIL_ROWS
const ruleExample = RANK_RULE_EXAMPLE

onLoad((options) => {
	const gameType = Number(options?.game_type || 0)
	if (gameTypeTabs.some((item) => item.value === gameType)) {
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
const fetchData = async ({ force = false } = {}) => {
	const requestId = latestRequestId.value + 1
	const requestedGameType = currentGameType.value
	latestRequestId.value = requestId
	isFetching.value = true

	try {
		const [, listRes] = await Promise.all([
			force
				? rankStore.forceRefresh(getReadIdentity())
				: rankStore.ensureFresh(getReadIdentity()),
			publicReadStore.loadStatic('rank-configs', getRankConfigs, { force })
		])

		if (!shouldApplyRankExplainResponse({ requestId, latestRequestId: latestRequestId.value })) {
			return
		}

		const nextRankInfo = rankStore.rankInfoMap[requestedGameType]
		if (!nextRankInfo || !listRes?.success || !listRes.list) {
			throw new Error('段位数据加载失败')
		}

		rankInfo.value = nextRankInfo
		rankList.value = listRes.list
		loadedGameType.value = requestedGameType
		hasLoadedOnce.value = true
		hasLoadError.value = false
	} catch (error) {
		if (!shouldApplyRankExplainResponse({ requestId, latestRequestId: latestRequestId.value })) {
			return
		}

		console.error('获取段位数据失败:', error)
		if (hasLoadedOnce.value) {
			// 已有成功内容：保留旧内容，仅轻量提示刷新失败
			currentGameType.value = loadedGameType.value
			uni.showToast({
				title: error.message || '获取数据失败',
				icon: 'none'
			})
		} else {
			hasLoadError.value = true
		}
	} finally {
		if (shouldApplyRankExplainResponse({ requestId, latestRequestId: latestRequestId.value })) {
			isFetching.value = false
		}
	}
}

/**
 * 首次加载失败后重新加载
 */
const handleRetry = () => {
	fetchData({ force: true })
}

/**
 * 展开/收起完整计分规则
 */
const toggleRuleDetails = () => {
	isRuleDetailsOpen.value = !isRuleDetailsOpen.value
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
