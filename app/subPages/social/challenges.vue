<template>
	<view class="challenges-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="page-tip">
			<uni-icons type="info" size="16" color="#E0AE12"></uni-icons>
			<text>约好后双方各点一次进入即正式开局；未打成的约球不会产生比赛。</text>
		</view>

		<view class="tab-bar">
			<view class="tab-item" :class="{ active: tab === 'received' }" @tap="switchTab('received')">
				<text>收到</text>
				<view v-if="receivedCount > 0" class="tab-badge"><text>{{ receivedCount }}</text></view>
			</view>
			<view class="tab-item" :class="{ active: tab === 'sent' }" @tap="switchTab('sent')">
				<text>发出</text>
			</view>
			<view class="tab-item" :class="{ active: tab === 'history' }" @tap="switchTab('history')">
				<text>约球记录</text>
			</view>
		</view>

		<view v-if="tab === 'history' && activeFailureBanner" class="history-refresh-error" @tap="refresh">
			<text>{{ activeFailureBanner.title }}，点击重试</text>
		</view>

		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="pageError" class="empty-state error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="empty-text">{{ pageError.title }}</text>
			<text class="empty-text">{{ pageError.description }}</text>
			<button class="retry-btn" @tap="refresh">重试</button>
		</view>

		<view v-else-if="list.length > 0" class="challenge-list">
			<view v-if="tab === 'history' && historyRefreshError" class="history-refresh-error" @tap="refresh">
				<text>约球记录刷新失败，点击重试</text>
			</view>
			<view v-for="item in list" :key="item.id" class="challenge-card">
				<view class="card-top" @tap="openDetail(item)">
					<image class="challenge-avatar" :src="resolveAvatarUrl(opponentOf(item).avatar, opponentOf(item).id)" mode="aspectFill"></image>
					<view class="challenge-info">
						<text class="challenge-name">{{ opponentOf(item).nickname || '球友' }}</text>
						<text class="challenge-game">{{ gameTypeLabel(item) }} · {{ matchModeLabel(item) }}</text>
						<text class="challenge-time">{{ scheduleLabel(item) }}</text>
					</view>
					<view class="challenge-status" :class="statusClass(item)">
						<text>{{ statusLabel(item) }}</text>
					</view>
				</view>
				<view v-if="item.message" class="challenge-message"><text>"{{ item.message }}"</text></view>

				<!-- 收到待回应 -->
				<view v-if="tab === 'received' && item.status === 0" class="card-actions">
					<view class="action-btn ghost-btn" @tap.stop="handleReject(item)"><text>这次不了</text></view>
					<view class="action-btn accept-btn" @tap.stop="handleAccept(item)"><text>接受约球</text></view>
				</view>
				<!-- 本人发出待回应 -->
				<view v-else-if="isMineSentPending(item)" class="card-actions">
					<view class="action-btn ghost-btn" @tap.stop="handleCancel(item)"><text>取消约球</text></view>
				</view>
				<!-- 已接受未开局 -->
				<view v-else-if="item.status === 1" class="card-actions">
					<view class="action-btn accept-btn" @tap.stop="openWaiting(item)"><text>进入对局</text></view>
					<view class="action-btn ghost-btn" @tap.stop="handleCancel(item)"><text>取消约球</text></view>
				</view>
				<!-- 已开局 -->
				<view v-else-if="item.status === 6 && item.match_id" class="card-actions">
					<view class="action-btn accept-btn" @tap.stop="openMatch(item)"><text>查看对局</text></view>
				</view>
				<!-- 终态 -->
				<view v-else-if="isTerminal(item)" class="card-actions">
					<view class="action-btn ghost-btn" @tap.stop="rematch(item)"><text>再约一场</text></view>
				</view>
			</view>
			<view v-if="tab === 'history' && nextBeforeId > 0" class="load-more" @tap="loadMoreHistory">
				<text>加载更多</text>
			</view>
		</view>

		<view v-else class="empty-state">
			<text class="empty-icon">🎱</text>
			<text class="empty-text">{{ emptyText }}</text>
		</view>
	</view>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { onShow, onHide, onUnload } from '@dcloudio/uni-app'
import {
	acceptChallenge,
	cancelChallenge,
	getChallengeDetail,
	getChallengeHistory,
	getPendingChallenges,
	rejectChallenge
} from '@/api/challenge.js'
import { getChallengeSummary } from '@/api/challenge.js'
import { GAME_TYPE_LABEL_MAP } from '@/utils/game-types.js'
import { useUserStore } from '@/store/user.js'
import { useActivityStore } from '@/store/activity.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import { formatChallengeExpiry, formatChallengeSchedule } from '@/utils/challenge-view.js'
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
const userStore = useUserStore()
const activityStore = useActivityStore()

const tab = ref('received')
const activeItems = ref([])
const historyItems = ref([])
const nextBeforeId = ref(0)
const loading = ref(true)
const serverTime = ref('')
const focusId = ref(0)
const focused = ref(null)
const challengesDirty = ref(true)
// 历史缓存独立失效：其他 Tab 的写操作（取消/拒绝/接受）会改变历史内容。
const historyDirty = ref(true)
// 写入序号：历史响应只能消费自己读取之前发生的写入，读取期间的新写入不算被覆盖。
const historyDirtySeq = ref(0)
const markHistoryDirty = () => {
	historyDirty.value = true
	historyDirtySeq.value += 1
}
// 页面生命周期：离开后旧写响应不得再弹窗或导航。
let pageActive = true
const challengeState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
// 历史与活动列表分别维护状态：一方的失败不能把另一方伪装成空数据。
const historyState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const challengePageError = computed(() => (
	challengeState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(challengeState.value.error, { resource: '约球' })
		: null
))
const historyPageError = computed(() => (
	historyState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(historyState.value.error, { resource: '约球记录' })
		: null
))
const historyRefreshError = computed(() => (
	historyState.value.refreshError
		? resolveAsyncPageErrorFeedback(historyState.value.refreshError, { resource: '约球记录' })
		: null
))
// 当前 Tab 的错误只由对应资源决定：活动失败不能遮住已成功加载的历史。
const pageError = computed(() => (
	tab.value === 'history' ? historyPageError.value : challengePageError.value
))
const activeFailureBanner = computed(() => (
	challengeState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(challengeState.value.error, { resource: '约球状态' })
		: null
))

const receivedCount = computed(() => activeItems.value.filter((item) => isMineReceivedPending(item)).length)
const list = computed(() => {
	if (tab.value === 'received') return activeItems.value.filter((item) => isMineReceivedPending(item))
	if (tab.value === 'sent') {
		return activeItems.value.filter((item) => isMineSentPending(item) || item.status === 1 || item.status === 6)
	}
	return historyItems.value
})
const emptyText = computed(() => {
	if (tab.value === 'received') return '暂无待回应的约球'
	if (tab.value === 'sent') return '暂无进行中的约球'
	return '暂无约球记录'
})

const myUserId = computed(() => Number(userStore.userId) || 0)
const isMineReceivedPending = (item) => item.to_user_id === myUserId.value && item.status === 0
const isMineSentPending = (item) => item.from_user_id === myUserId.value && item.status === 0
const isTerminal = (item) => [2, 3, 4, 5, 7, 8].includes(Number(item.status))

const opponentOf = (item) => ({
	id: item.from_user_id === myUserId.value ? item.to_user_id : item.from_user_id,
	nickname: item.from_user_id === myUserId.value ? item.to_nickname : item.from_nickname,
	avatar: item.from_user_id === myUserId.value ? item.to_avatar : item.from_avatar
})
const gameTypeLabel = (item) => GAME_TYPE_LABEL_MAP[item.game_type] || '台球'
const matchModeLabel = (item) => (item.match_mode === 'practice' ? '练习' : '排位')
const scheduleLabel = (item) => formatChallengeSchedule(item, serverTime.value)
const expiryLabel = (item) => formatChallengeExpiry(item, serverTime.value)
const statusClass = (item) => `status-${item.status}`
const statusLabel = (item) => {
	const status = Number(item.status)
	if (status === 0) return item.from_user_id === myUserId.value ? '等待回应' : '待你回应'
	if (status === 1) {
		if (item.waiting_user_id === myUserId.value) return '等待对方进入'
		if (item.waiting_user_id > 0) return '对方已进入'
		return '已接受'
	}
	if (status === 6) return '对局进行中'
	const map = { 2: '已拒绝', 3: '已失效', 4: '已取消', 5: '已放弃', 7: '已完成', 8: '对局已取消' }
	const label = map[status]
	if (status >= 2 && item.close_reason && expiryLabel(item) && status === 3) {
		return `${label}`
	}
	return label || ''
}

const switchTab = (value) => {
	tab.value = value
	// 账号切换后不得继续展示旧账号缓存的历史。
	if (historyState.value.authGeneration !== userStore.authGeneration) {
		historyItems.value = []
		nextBeforeId.value = 0
	}
	syncLoading()
	if (value === 'history' && (historyState.value.status === ASYNC_PAGE_STATUS.IDLE || historyState.value.authGeneration !== userStore.authGeneration || historyDirty.value)) {
		fetchHistory({ force: true })
	}
}

const fetchActive = async () => {
	const res = await getPendingChallenges()
	if (!res?.success) throw createRequestError({ message: res?.message || '加载约球失败', category: 'business' })
	const summary = await getChallengeSummary()
	return { items: res.list || [], serverTime: summary?.success ? summary.server_time : '' }
}

const loadHistory = async () => {
	const res = await getChallengeHistory({ page_size: 20 })
	if (!res?.success) throw new Error(res?.message || '加载约球记录失败')
	return res
}

const loadMoreHistory = async () => {
	const cursor = nextBeforeId.value
	if (cursor <= 0) return
	const userId = userStore.userId
	const generation = userStore.authGeneration
	const res = await getChallengeHistory({ page_size: 20, before_id: cursor })
	if (userStore.userId !== userId || userStore.authGeneration !== generation || tab.value !== 'history' || nextBeforeId.value !== cursor) return
	if (res?.success) {
		historyItems.value = historyItems.value.concat(res.list || [])
		nextBeforeId.value = res.next_before_id || 0
	}
}

const syncLoading = () => {
	// 当前 Tab 的加载只由对应资源决定：历史读取不被活动请求阻塞或遮蔽。
	loading.value = tab.value === 'history'
		? historyState.value.status === ASYNC_PAGE_STATUS.LOADING
		: challengeState.value.status === ASYNC_PAGE_STATUS.LOADING
}

const fetchActiveList = async ({ force = false } = {}) => {
	if (!force && !challengesDirty.value) return
	const nextState = beginAsyncPageLoad(challengeState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	const userId = userStore.userId
	const isCurrent = () => userStore.userId === userId && userStore.authGeneration === pageRequest.authGeneration &&
		challengeState.value.requestId === pageRequest.requestId && challengeState.value.authGeneration === pageRequest.authGeneration
	if (challengeState.value.authGeneration !== userStore.authGeneration) {
		activeItems.value = []
		historyItems.value = []
		nextBeforeId.value = 0
	}
	challengeState.value = nextState
	syncLoading()
	try {
		const active = await fetchActive()
		if (!isCurrent()) return
		activeItems.value = active.items
		if (active.serverTime) serverTime.value = active.serverTime
		challengesDirty.value = false
		challengeState.value = resolveAsyncPageLoad(challengeState.value, pageRequest, { data: activeItems.value })
	} catch (e) {
		if (isCurrent()) challengeState.value = rejectAsyncPageLoad(challengeState.value, pageRequest, e)
	} finally {
		if (isCurrent()) syncLoading()
	}
}

// 历史与活动列表分别维护状态：一方的失败不能把另一方伪装成空数据。
const fetchHistory = async ({ force = false } = {}) => {
	const nextState = beginAsyncPageLoad(historyState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	const userId = userStore.userId
	const isCurrent = () => userStore.userId === userId && userStore.authGeneration === pageRequest.authGeneration &&
		historyState.value.requestId === pageRequest.requestId && historyState.value.authGeneration === pageRequest.authGeneration
	if (historyState.value.authGeneration !== userStore.authGeneration) {
		historyItems.value = []
		nextBeforeId.value = 0
	}
	historyState.value = nextState
	syncLoading()
	const startDirtySeq = historyDirtySeq.value
	try {
		const res = await loadHistory()
		if (!isCurrent()) return
		if (res?.success) {
			historyItems.value = res.list || []
			nextBeforeId.value = res.next_before_id || 0
			// 读取期间发生的新写入不得被这份响应覆盖。
			historyDirty.value = historyDirtySeq.value > startDirtySeq
		} else {
			throw createRequestError({ message: res?.message || '加载约球记录失败', category: 'business' })
		}
		historyState.value = resolveAsyncPageLoad(historyState.value, pageRequest, { data: historyItems.value })
	} catch (e) {
		if (isCurrent()) historyState.value = rejectAsyncPageLoad(historyState.value, pageRequest, e)
	} finally {
		if (isCurrent()) syncLoading()
		// 读取期间发生过新写入（如另一设备/对方取消）时，当前 Tab 仍是历史则去重补读一次，
		// 不能让用户不切 Tab 就一直看到旧缓存；读取失败保留 dirty 交给重试入口，不自我重试。
		if (isCurrent() && historyDirty.value && historyDirtySeq.value > startDirtySeq && tab.value === 'history') {
			fetchHistory({ force: true })
		}
	}
}

const fetchList = async ({ force = false } = {}) => {
	const jobs = [fetchActiveList({ force })]
	if (tab.value === 'history') jobs.push(fetchHistory({ force }))
	await Promise.all(jobs)
}

const refresh = () => fetchList({ force: true })

// 约球写操作后同步标脏主 Tab 活动卡与历史缓存。
const markActivityDirty = () => {
	activityStore.markDirty()
	markHistoryDirty()
}

// 写请求与确认回调按发起时的身份隔离：切号后旧响应不得影响新账号或触发导航。
const readIdentity = () => ({ userId: userStore.userId, authGeneration: userStore.authGeneration })
const isSameIdentity = (identity) => userStore.userId === identity.userId && userStore.authGeneration === identity.authGeneration

const applyFocus = async () => {
	if (!focusId.value || focused.value) return
	const userId = userStore.userId
	const generation = userStore.authGeneration
	focused.value = true
	try {
		const res = await getChallengeDetail({ id: focusId.value })
		if (userStore.userId !== userId || userStore.authGeneration !== generation) return
		if (res?.success && res.challenge) {
			activeItems.value = activeItems.value.filter((item) => item.id !== res.challenge.id)
			activeItems.value.unshift(res.challenge)
		}
	} catch (error) {
		// 聚焦失败不阻塞列表。
	}
}

onMounted(() => {
	const pages = getCurrentPages()
	const current = pages[pages.length - 1]
	const options = current?.$page?.options || current?.options || {}
	if (['received', 'sent', 'history'].includes(options.tab)) tab.value = options.tab
	if (options.focus) focusId.value = Number(options.focus) || 0
	fetchList({ force: true }).then(applyFocus)
})

onShow(() => {
	pageActive = true
	// 切号后必须立即为新账号重新读取；首次加载由 onMounted 负责。
	if (challengeState.value.authGeneration !== userStore.authGeneration ||
		historyState.value.authGeneration !== userStore.authGeneration) {
		fetchList({ force: true })
		return
	}
	// 包括空列表在内都要恢复权威状态；历史可能已被另一设备/对方改变，标脏待切回时重读。
	markHistoryDirty()
	if (!loading.value) fetchList({ force: true })
})
onHide(() => {
	pageActive = false
})
onUnload(() => {
	pageActive = false
})

const requireConfirm = (title, content, confirmText, action) => {
	uni.showModal({ title, content, confirmText, cancelText: '暂不', success: ({ confirm }) => { if (confirm && pageActive) action() } })
}

const handleAccept = (item) => {
	const identity = readIdentity()
	acceptChallenge({ challenge_id: item.id }).then((res) => {
		if (!isSameIdentity(identity)) return
		if (res?.success) {
			markActivityDirty()
			if (!pageActive) return
			uni.showToast({ title: '已接受，双方进入后正式开局', icon: 'none' })
			uni.navigateTo({ url: `/subPages/match/challengeWaiting?challenge_id=${res.challenge_id || item.id}` })
			fetchList({ force: true })
			return
		}
		if (res?.need_confirm_close_own && res.pending_challenge_id > 0) {
			if (!pageActive) return
			uni.showModal({
				title: '接受这个约球？',
				content: '接受后会自动关闭你已发出的邀请。',
				confirmText: '确认接受',
				cancelText: '暂不接受',
				success: ({ confirm }) => {
					if (!confirm || !pageActive || !isSameIdentity(identity)) return
					acceptChallenge({ challenge_id: item.id, confirm_close_own_pending: true }).then((retry) => {
						if (!isSameIdentity(identity)) return
						if (retry?.success) {
							markActivityDirty()
							if (!pageActive) return
							uni.showToast({ title: '已接受', icon: 'success' })
							uni.navigateTo({ url: `/subPages/match/challengeWaiting?challenge_id=${retry.challenge_id || item.id}` })
							fetchList({ force: true })
						} else {
							if (!pageActive) return
							uni.showToast({ title: retry?.message || '接受失败', icon: 'none' })
						}
					})
				}
			})
			return
		}
		if (res?.type_conflict) {
			if (!pageActive) return
			uni.showModal({
				title: '类型不同，不能直接匹配',
				content: res.message || '你们发起的约球类型不同，可先取消已发出的邀请再接受。',
				confirmText: '查看发出的邀请',
				cancelText: '知道了',
				success: ({ confirm }) => {
					if (confirm && pageActive && isSameIdentity(identity)) {
						tab.value = 'sent'
					}
				}
			})
			return
		}
		if (res?.match_id > 0) {
			if (!pageActive) return
			uni.navigateTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
			return
		}
		if (!pageActive) return
		uni.showToast({ title: res?.message || '接受失败', icon: 'none' })
	}).catch(() => {
		if (pageActive && isSameIdentity(identity)) uni.showToast({ title: '操作失败', icon: 'none' })
	})
}

const handleReject = (item) => {
	const identity = readIdentity()
	rejectChallenge({ challenge_id: item.id }).then((res) => {
		if (!isSameIdentity(identity)) return
		if (res?.success) {
			markActivityDirty()
			if (!pageActive) return
			uni.showToast({ title: '已拒绝', icon: 'none' })
			fetchList({ force: true })
		} else {
			if (!pageActive) return
			uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
		}
	}).catch(() => {
		if (pageActive && isSameIdentity(identity)) uni.showToast({ title: '操作失败', icon: 'none' })
	})
}

const handleCancel = (item) => {
	const identity = readIdentity()
	requireConfirm('取消本次约球？', '取消后需要重新邀约才能再打。', '取消约球', () => {
		if (!isSameIdentity(identity)) return
		cancelChallenge({ challenge_id: item.id }).then((res) => {
			if (!isSameIdentity(identity)) return
			if (res?.success) {
				markActivityDirty()
				if (!pageActive) return
				uni.showToast({ title: '约球已取消', icon: 'none' })
				fetchList({ force: true })
			} else if (res?.match_id > 0) {
				if (!pageActive) return
				uni.navigateTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
			} else {
				if (!pageActive) return
				uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
			}
		}).catch(() => {
			if (pageActive && isSameIdentity(identity)) uni.showToast({ title: '操作失败', icon: 'none' })
		})
	})
}

const openWaiting = (item) => {
	uni.navigateTo({ url: `/subPages/match/challengeWaiting?challenge_id=${item.id}` })
}

const openMatch = (item) => {
	if (!item.match_id) return
	uni.navigateTo({ url: `/subPages/match/playing?match_id=${item.match_id}` })
}

const openDetail = (item) => {
	if (item.status === 1 || item.status === 6) {
		openWaiting(item)
	}
}

const rematch = (item) => {
	const opponent = opponentOf(item)
	const query = [
		`opponent_id=${opponent.id}`,
		`opponent_name=${encodeURIComponent(opponent.nickname || '')}`,
		`opponent_avatar=${encodeURIComponent(opponent.avatar || '')}`,
		`game_type=${item.game_type || 0}`,
		`match_mode=${item.match_mode === 'practice' ? 'practice' : 'ranked'}`
	]
	uni.navigateTo({ url: `/subPages/match/challengeCompose?${query.join('&')}` })
}
</script>

<style lang="scss" scoped>
.challenges-page {
	min-height: 100vh;
	background: #FAF8F2;
	padding-bottom: 32rpx;
}
.page-tip {
	margin: 20rpx 24rpx 0;
	padding: 18rpx 20rpx;
	border-radius: 20rpx;
	background: rgba(224, 174, 18, 0.12);
	display: flex;
	align-items: flex-start;
	gap: 12rpx;
	box-sizing: border-box;

	text {
		font-size: 24rpx;
		line-height: 1.6;
		color: #8A7340;
	}
}
.tab-bar {
	display: flex;
	margin: 20rpx 24rpx 0;
	background: #ffffff;
	border-radius: 20rpx;
	padding: 8rpx;

	.tab-item {
		flex: 1;
		text-align: center;
		padding: 16rpx 0;
		border-radius: 14rpx;
		position: relative;

		text {
			font-size: 26rpx;
			color: #9A8C67;
		}

		&.active {
			background: rgba(224, 174, 18, 0.14);

			text {
				color: #B8860B;
				font-weight: 600;
			}
		}

		.tab-badge {
			position: absolute;
			top: 4rpx;
			right: 20rpx;
			min-width: 30rpx;
			height: 30rpx;
			line-height: 30rpx;
			border-radius: 15rpx;
			background: #E05B4C;

			text {
				color: #ffffff;
				font-size: 20rpx;
			}
		}
	}
}
.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 16rpx;
	padding: 120rpx 0;
}
.loading-text {
	font-size: 26rpx;
	color: #9A8C67;
}
.challenge-list {
	margin: 20rpx 24rpx 0;
	display: flex;
	flex-direction: column;
	gap: 20rpx;
}
.history-refresh-error {
	padding: 16rpx 20rpx;
	border-radius: 16rpx;
	background: rgba(224, 91, 76, 0.12);

	text {
		font-size: 24rpx;
		color: #C0533F;
	}
}
.challenge-card {
	background: #ffffff;
	border-radius: 24rpx;
	padding: 26rpx;
	box-sizing: border-box;
}
.card-top {
	display: flex;
	align-items: center;
	gap: 20rpx;
}
.challenge-avatar {
	width: 88rpx;
	height: 88rpx;
	border-radius: 50%;
	background: #F1EADC;
}
.challenge-info {
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 6rpx;

	.challenge-name {
		font-size: 30rpx;
		font-weight: 500;
		color: #231C0B;
	}
	.challenge-game {
		font-size: 24rpx;
		color: #9A8C67;
	}
	.challenge-time {
		font-size: 24rpx;
		color: #9A8C67;
	}
}
.challenge-status {
	padding: 8rpx 20rpx;
	border-radius: 999rpx;
	background: #F5F0E4;

	text {
		font-size: 22rpx;
		color: #8A7340;
	}

	&.status-0 {
		background: rgba(224, 174, 18, 0.16);

		text {
			color: #B8860B;
		}
	}
	&.status-1, &.status-6 {
		background: rgba(59, 130, 246, 0.12);

		text {
			color: #2563EB;
		}
	}
}
.challenge-message {
	margin-top: 18rpx;
	padding: 16rpx 20rpx;
	border-radius: 14rpx;
	background: #FAF8F2;

	text {
		font-size: 24rpx;
		color: #6C6146;
	}
}
.card-actions {
	margin-top: 20rpx;
	display: flex;
	gap: 16rpx;

	.action-btn {
		flex: 1;
		height: 76rpx;
		line-height: 76rpx;
		border-radius: 999rpx;
		text-align: center;

		text {
			font-size: 26rpx;
		}
	}
	.accept-btn {
		background: #E0AE12;

		text {
			color: #ffffff;
			font-weight: 600;
		}
	}
	.ghost-btn {
		background: #F5F0E4;

		text {
			color: #6C6146;
		}
	}
}
.load-more {
	padding: 20rpx 0;
	text-align: center;

	text {
		font-size: 26rpx;
		color: #9A8C67;
	}
}
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 16rpx;
	padding: 120rpx 48rpx 0;
	box-sizing: border-box;

	.empty-icon {
		font-size: 60rpx;
	}
	.empty-text {
		font-size: 26rpx;
		color: #9A8C67;
		text-align: center;
	}
	.retry-btn {
		margin-top: 10rpx;
		height: 76rpx;
		line-height: 76rpx;
		padding: 0 60rpx;
		border-radius: 999rpx;
		background: #E0AE12;

		&::after {
			border: none;
		}

		text {
			color: #ffffff;
			font-size: 26rpx;
		}
	}
}
.challenges-page.dark-mode {
	background: #191413;

	.page-tip {
		background: rgba(224, 174, 18, 0.1);
	}
	.tab-bar,
	.challenge-card,
	.challenge-message {
		background: #241F1D;
	}
	.challenge-message {
		background: #191413;
	}
	.challenge-info .challenge-name,
	.challenge-time {
		color: #E8DFC9;
	}
	.challenge-status {
		background: #2F2823;

		text {
			color: #C6B78C;
		}
	}
	.card-actions .ghost-btn {
		background: #2F2823;

		text {
			color: #C6B78C;
		}
	}
}
</style>
