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
				<view v-if="statusFeedback" class="summary-error">
					<text>{{ statusFeedback.description }}</text>
					<text class="summary-retry" @click="handleStatusErrorAction">{{ statusFeedback.actionText }}</text>
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

				<view v-if="logsPageError" class="state-card">
					<uni-icons type="info" size="28" :color="isDarkMode ? '#fbbf24' : '#d97706'"></uni-icons>
					<text class="state-title">{{ logsPageError.title }}</text>
					<text class="state-description">{{ logsPageError.description }}</text>
					<button class="state-btn" @click="handleLogsErrorAction">
						<text>{{ logsPageError.actionText }}</text>
					</button>
				</view>

				<view v-else-if="logsLoading && reputationLogs.length === 0" class="state-card">
					<uni-icons type="spinner-cycle" size="32" :color="isDarkMode ? '#9F926E' : '#6E6242'"></uni-icons>
					<text class="state-title">正在加载信誉记录</text>
				</view>

				<view v-else-if="logsState.status === ASYNC_PAGE_STATUS.EMPTY" class="state-card">
					<uni-icons type="list" size="32" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
					<text class="state-title">暂时还没有信誉变更记录</text>
				</view>

				<view v-else class="log-list">
					<view v-if="logsRefreshError" class="inline-error">
						<text>{{ logsRefreshError.description }}</text>
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
import { usePageTheme } from '@/utils/page-theme.js'
import { getUserReputationLogs } from '@/api/user.js'
import { useUserOverviewStore } from '@/store/userOverview.js'
import { useUserStore } from '@/store/user.js'
import { formatDateTime, formatRelativeTime } from '@/utils/format.js'
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
const userOverviewStore = useUserOverviewStore()
const userStore = useUserStore()

const statusLoading = ref(false)
const reputationStatus = ref(null)
const statusState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: null
}))

const logsLoading = ref(false)
const logsState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const reputationLogs = ref([])
const currentPage = ref(1)
const total = ref(0)
const logsLoaded = ref(false)

const PAGE_SIZE = 20
let statusRequestSeq = 0
let logsRequestSeq = 0
const statusFeedback = computed(() => (
	statusState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(statusState.value.error, { resource: '信誉状态' })
		: null
))
const logsPageError = computed(() => (
	logsState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(logsState.value.error, { resource: '信誉记录' })
		: null
))
const logsRefreshError = computed(() => (
	logsState.value.refreshError
		? resolveAsyncPageErrorFeedback(logsState.value.refreshError, { resource: '信誉记录' })
		: null
))

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
	void loadStatus()
	if (!logsLoaded.value) void loadLogs({ refresh: true })
})

const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const loadStatus = async ({ force = false } = {}) => {
	if (statusLoading.value && !force) {
		return
	}
	const requestSeq = ++statusRequestSeq
	const nextState = beginAsyncPageLoad(statusState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: null
	})
	if (statusState.value.authGeneration !== userStore.authGeneration) reputationStatus.value = null
	const pageRequest = getAsyncPageRequest(nextState)
	statusState.value = nextState
	statusLoading.value = true
	try {
		const overview = await userOverviewStore.fetch(getReadIdentity(), { force, silent: true })
		if (requestSeq !== statusRequestSeq || statusState.value.requestId !== pageRequest.requestId || statusState.value.authGeneration !== pageRequest.authGeneration) {
			return
		}
		if (!overview?.reputation?.success) throw createRequestError({ message: '信誉状态暂时不可用', category: 'business' })
		reputationStatus.value = overview.reputation
		statusState.value = resolveAsyncPageLoad(statusState.value, pageRequest, {
			data: reputationStatus.value,
			isEmpty: () => false
		})
	} catch (error) {
		if (requestSeq !== statusRequestSeq) {
			return
	}
	statusState.value = rejectAsyncPageLoad(statusState.value, pageRequest, error)
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
	if (append && !hasMore.value) return

	const requestSeq = ++logsRequestSeq
	const targetPage = refresh ? 1 : currentPage.value + 1
	const nextState = beginAsyncPageLoad(logsState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	if (logsState.value.authGeneration !== userStore.authGeneration) {
		reputationLogs.value = []
		currentPage.value = 1
		total.value = 0
		hasMore.value = true
		logsLoaded.value = false
	}
	const pageRequest = getAsyncPageRequest(nextState)
	logsState.value = nextState

	if (append) {
		isLoadingMore.value = true
	} else {
		logsLoading.value = true
	}

	try {
		const res = await getUserReputationLogs({
			page: targetPage,
			page_size: PAGE_SIZE
		})
		if (requestSeq !== logsRequestSeq || logsState.value.requestId !== pageRequest.requestId || logsState.value.authGeneration !== pageRequest.authGeneration) {
			return
		}
		if (!res?.success) throw createRequestError({ message: res?.message || '信誉记录加载失败，请稍后重试', category: 'business' })
		const list = Array.isArray(res?.list) ? res.list : []
		total.value = Number(res?.total) || 0
		currentPage.value = targetPage
		reputationLogs.value = append ? [...reputationLogs.value, ...list] : list
		hasMore.value = reputationLogs.value.length < total.value
		logsLoaded.value = true
		logsState.value = resolveAsyncPageLoad(logsState.value, pageRequest, { data: reputationLogs.value })
	} catch (error) {
		if (requestSeq !== logsRequestSeq) {
			return
	}
	logsState.value = rejectAsyncPageLoad(logsState.value, pageRequest, error)
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

const handleStatusErrorAction = () => {
	if (statusState.value.error?.category === 'permission' || statusState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	handleRetryStatus()
}

const handleLogsErrorAction = () => {
	if (logsState.value.error?.category === 'permission' || logsState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	handleRetryLogs()
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
