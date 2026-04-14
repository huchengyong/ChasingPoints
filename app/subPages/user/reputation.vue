<template>
	<view class="reputation-page" :class="{ 'dark-mode': isDarkMode }">
		<scroll-view
			class="reputation-scroll"
			scroll-y
			:refresher-enabled="true"
			:refresher-triggered="isRefreshing"
			@refresherrefresh="handleRefresh"
			@scrolltolower="handleLoadMore"
		>
			<view class="summary-card">
				<view class="summary-header">
					<view class="summary-copy">
						<text class="summary-eyebrow">当前信誉</text>
						<text class="summary-score">{{ displayScore }}</text>
					</view>
					<view class="summary-badge" :class="statusToneClass">
						<text>{{ displayStatusText }}</text>
					</view>
				</view>
				<text v-if="statusBanUntilText" class="summary-ban">禁赛至 {{ statusBanUntilText }}</text>
				<text class="summary-tip">异常比赛会扣减信誉，信誉会按规则恢复</text>
				<view v-if="statusError" class="summary-error">
					<text>{{ statusError }}</text>
					<text class="summary-retry" @click="handleRetryStatus">重新获取</text>
				</view>
			</view>

			<view class="section-block">
				<view class="section-header">
					<view>
						<text class="section-title">信誉记录</text>
						<text class="section-subtitle">只展示你自己的信誉变更流水</text>
					</view>
					<text class="section-total" v-if="total > 0">共 {{ total }} 条</text>
				</view>

				<view v-if="logsError && reputationLogs.length === 0" class="state-card">
					<uni-icons type="info" size="28" :color="isDarkMode ? '#fbbf24' : '#d97706'"></uni-icons>
					<text class="state-title">信誉记录暂时加载失败</text>
					<text class="state-description">{{ logsError }}</text>
					<button class="state-btn" @click="handleRetryLogs">
						<text>重试</text>
					</button>
				</view>

				<view v-else-if="logsLoading && reputationLogs.length === 0" class="state-card">
					<uni-icons type="spinner-cycle" size="32" :color="isDarkMode ? '#94a3b8' : '#64748b'"></uni-icons>
					<text class="state-title">正在加载信誉记录</text>
				</view>

				<view v-else-if="!logsLoading && reputationLogs.length === 0" class="state-card">
					<uni-icons type="list" size="32" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					<text class="state-title">暂时还没有信誉变更记录</text>
				</view>

				<view v-else class="log-list">
					<view v-if="logsError" class="inline-error">
						<text>{{ logsError }}</text>
						<text class="inline-error-retry" @click="handleRetryLogs">重试</text>
					</view>

					<view
						v-for="item in reputationLogs"
						:key="item.id"
						class="log-item"
					>
						<view class="log-main">
							<view class="log-head">
								<view class="log-title-row">
									<text class="log-title">{{ item.reason_text || '信誉变更' }}</text>
									<text class="log-type">{{ item.change_type_text || '记录' }}</text>
								</view>
								<text class="log-score" :class="getScoreClass(item.change_score)">
									{{ formatChangeScore(item.change_score) }}
								</text>
							</view>
							<view class="log-meta">
								<text class="log-meta-text">变更前 {{ item.before_score }}</text>
								<text class="log-meta-divider">→</text>
								<text class="log-meta-text">变更后 {{ item.after_score }}</text>
							</view>
							<view class="log-footer">
								<text class="log-time">{{ formatLogTime(item.created_at) }}</text>
								<text v-if="item.match_id" class="log-match">对局ID {{ item.match_id }}</text>
							</view>
						</view>
					</view>
				</view>

				<view v-if="reputationLogs.length > 0" class="load-more">
					<text v-if="isLoadingMore" class="load-more-text">加载更多...</text>
					<text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { getUserReputation, getUserReputationLogs } from '@/api/user.js'
import { formatDateTime, formatRelativeTime } from '@/utils/format.js'

const themeStore = useThemeStore()

const isDarkMode = computed(() => themeStore.isDarkMode)

const statusLoading = ref(false)
const statusError = ref('')
const reputationStatus = ref(null)

const logsLoading = ref(false)
const logsError = ref('')
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const reputationLogs = ref([])
const currentPage = ref(1)
const total = ref(0)

const PAGE_SIZE = 20
let statusRequestSeq = 0
let logsRequestSeq = 0

const displayScore = computed(() => {
	const score = reputationStatus.value?.score
	return typeof score === 'number' ? String(score) : '--'
})

const displayStatusText = computed(() => {
	if (reputationStatus.value?.status_text) {
		return reputationStatus.value.status_text
	}
	if (statusLoading.value) {
		return '加载中'
	}
	return '暂不可用'
})

const statusBanUntilText = computed(() => {
	if (!reputationStatus.value?.ban_until) return ''
	return formatDateTime(reputationStatus.value.ban_until)
})

const statusToneClass = computed(() => {
	if (!reputationStatus.value) {
		return 'is-unknown'
	}
	return reputationStatus.value?.status === 'restricted' ? 'is-restricted' : 'is-good'
})

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	refreshPageData()
})

const refreshPageData = () => {
	void loadStatus()
	void loadLogs({ refresh: true })
}

const loadStatus = async ({ force = false } = {}) => {
	if (statusLoading.value && !force) {
		return
	}
	const requestSeq = ++statusRequestSeq
	statusLoading.value = true
	statusError.value = ''
	try {
		const res = await getUserReputation()
		if (requestSeq !== statusRequestSeq) {
			return
		}
		reputationStatus.value = res || null
	} catch (error) {
		if (requestSeq !== statusRequestSeq) {
			return
		}
		console.error('获取信誉状态失败:', error)
		statusError.value = error?.message || '信誉状态暂时不可用'
	} finally {
		if (requestSeq === statusRequestSeq) {
			statusLoading.value = false
		}
	}
}

const loadLogs = async ({ refresh = false, append = false, force = false } = {}) => {
	if ((logsLoading.value || isLoadingMore.value) && !force) {
		return
	}

	const requestSeq = ++logsRequestSeq
	const targetPage = refresh ? 1 : currentPage.value + 1
	logsError.value = ''

	if (append) {
		if (!hasMore.value) return
		isLoadingMore.value = true
	} else {
		logsLoading.value = true
	}

	try {
		const res = await getUserReputationLogs({
			page: targetPage,
			page_size: PAGE_SIZE
		})
		if (requestSeq !== logsRequestSeq) {
			return
		}
		const list = Array.isArray(res?.list) ? res.list : []
		total.value = Number(res?.total) || 0
		currentPage.value = targetPage
		reputationLogs.value = append ? [...reputationLogs.value, ...list] : list
		hasMore.value = reputationLogs.value.length < total.value
	} catch (error) {
		if (requestSeq !== logsRequestSeq) {
			return
		}
		console.error('获取信誉记录失败:', error)
		logsError.value = error?.message || '信誉记录加载失败，请稍后重试'
		if (append) {
			hasMore.value = true
		}
	} finally {
		if (requestSeq === logsRequestSeq) {
			logsLoading.value = false
			isLoadingMore.value = false
		}
	}
}

const handleRefresh = async () => {
	isRefreshing.value = true
	await Promise.allSettled([
		loadStatus({ force: true }),
		loadLogs({ refresh: true, force: true })
	])
	isRefreshing.value = false
}

const handleLoadMore = () => {
	if (!hasMore.value || logsLoading.value || isLoadingMore.value) {
		return
	}
	void loadLogs({ append: true })
}

const handleRetryStatus = () => {
	void loadStatus({ force: true })
}

const handleRetryLogs = () => {
	void loadLogs({ refresh: true, force: true })
}

const formatChangeScore = (score) => {
	const numericScore = Number(score) || 0
	return numericScore > 0 ? `+${numericScore}` : `${numericScore}`
}

const getScoreClass = (score) => {
	const numericScore = Number(score) || 0
	if (numericScore > 0) return 'is-positive'
	if (numericScore < 0) return 'is-negative'
	return 'is-neutral'
}

const formatLogTime = (value) => {
	return value ? formatRelativeTime(value) : '--'
}
</script>

<style lang="scss" scoped>
@import './reputation.scss';
</style>
