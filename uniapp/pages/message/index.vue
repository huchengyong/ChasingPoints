<template>
	<view class="message-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="!isLoggedIn" class="guest-state">
			<view class="guest-icon">
				<uni-icons type="chat" size="32" :color="mutedIconColor"></uni-icons>
			</view>
			<text class="guest-title">登录后查看消息</text>
			<text class="guest-desc">好友申请、约球邀请和比赛提醒都会在这里</text>
			<button class="login-btn" @tap="goLogin">登录 / 注册</button>
		</view>

		<template v-else>
			<!-- 待处理事项入口：数量取自共享的活动快照 -->
			<view class="entry-card">
				<view v-for="entry in entries" :key="entry.key" class="entry-item" @tap="openEntry(entry)">
					<view class="entry-icon">
						<uni-icons :type="entry.icon" size="22" :color="accentIconColor"></uni-icons>
					</view>
					<view class="entry-copy">
						<text class="entry-title">{{ entry.title }}</text>
						<text class="entry-desc">{{ entry.desc }}</text>
					</view>
					<view v-if="entry.countText" class="entry-count">
						<text>{{ entry.countText }}</text>
					</view>
					<uni-icons type="right" size="16" :color="mutedIconColor"></uni-icons>
				</view>
			</view>

			<!-- 顶部操作栏 -->
			<view class="top-bar">
				<text class="page-subtitle">全部消息</text>
				<text class="read-all-btn" @tap="handleReadAll">全部已读</text>
			</view>

			<!-- 通知列表 -->
			<view v-if="notificationList.length > 0" class="notification-list">
				<view v-if="notificationRefreshError" class="refresh-error-banner">
					<text>{{ notificationRefreshError.description }}</text>
					<text class="refresh-error-action" @tap="retryNotifications">重试</text>
				</view>
				<view
					v-for="item in notificationList"
					:key="item.id"
					class="notification-item"
					:class="{ unread: !item.is_read }"
					@tap="handleTapNotification(item)"
				>
					<view class="item-icon" :class="getTypeClass(item.type)">
						<text class="icon-text">{{ getTypeIcon(item.type) }}</text>
					</view>
					<view class="item-content">
						<view class="item-header">
							<text class="item-title">{{ item.title }}</text>
							<view v-if="!item.is_read" class="unread-dot"></view>
						</view>
						<text class="item-body">{{ item.content }}</text>
						<text class="item-time">{{ item.created_at }}</text>
					</view>
					<view class="item-delete" @tap.stop="handleDelete(item.id)">
						<uni-icons type="trash" size="18" color="#9A8C67"></uni-icons>
					</view>
				</view>
			</view>

			<view v-else-if="notificationPageError" class="empty-state error-state">
				<uni-icons type="info" size="48" :color="mutedIconColor"></uni-icons>
				<text class="empty-text">{{ notificationPageError.title }}</text>
				<text class="empty-hint">{{ notificationPageError.description }}</text>
				<button class="error-action" @tap="handleNotificationErrorAction">{{ notificationPageError.actionText }}</button>
			</view>

			<!-- 空状态 -->
			<view v-else-if="notificationState.status === ASYNC_PAGE_STATUS.EMPTY" class="empty-state">
				<text class="empty-icon">🔔</text>
				<text class="empty-text">暂无消息</text>
				<text class="empty-hint">有新消息时会在这里通知你</text>
			</view>

			<!-- 加载中 -->
			<view v-else-if="loading" class="loading-state">
				<uni-icons type="spinner-cycle" size="28" color="#E0AE12"></uni-icons>
			</view>

			<!-- 加载更多 -->
			<view v-if="notificationState.status !== ASYNC_PAGE_STATUS.ERROR && !loading && hasMore && notificationList.length > 0" class="load-more" @tap="loadMore">
				<text>加载更多</text>
			</view>
		</template>
	</view>
</template>

<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'
import { onHide, onPullDownRefresh, onShow } from '@dcloudio/uni-app'
import { getNotificationList, markAsRead, markAllAsRead, deleteNotification } from '@/api/notification.js'
import { useActivityStore } from '@/store/activity.js'
import { useFriendRequestStore } from '@/store/friendRequest.js'
import { useNotificationStore } from '@/store/notification.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useMainTabBar } from '@/utils/use-main-tab-bar.js'
import { useMessageCounts } from '@/utils/use-message-counts.js'
import {
	NOTIFICATION_REFRESH_DEBOUNCE_MS,
	buildMessageEntries,
	resolveNotificationReload
} from '@/utils/message-center.js'
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
useMainTabBar('message')

const userStore = useUserStore()
const activityStore = useActivityStore()
const notificationStore = useNotificationStore()
const friendRequestStore = useFriendRequestStore()
const userDataInvalidationStore = useUserDataInvalidationStore()
const { friendRequestCount, challengeCount } = useMessageCounts()

const isLoggedIn = computed(() => userStore.isLoggedIn)
const mutedIconColor = computed(() => (isDarkMode.value ? '#9F926E' : '#9A8C67'))
const accentIconColor = computed(() => (isDarkMode.value ? '#F0C542' : '#C69200'))
const entries = computed(() => buildMessageEntries({
	friendRequestCount: friendRequestCount.value,
	challengeCount: challengeCount.value
}))

const getReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})
const notificationList = ref([])
const loading = ref(false)
const notificationState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: []
}))
const hasMore = ref(true)
const page = ref(1)
const pageSize = 20
const latestRequestId = ref(0)
const requestedIdentityKey = ref('')
const loadedIdentityKey = ref('')
const loadedScopeVersion = ref(-1)
const notificationPageError = computed(() => (
	notificationState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(notificationState.value.error, { resource: '消息中心' })
		: null
))
const notificationRefreshError = computed(() => (
	notificationState.value.refreshError
		? resolveAsyncPageErrorFeedback(notificationState.value.refreshError, { resource: '消息中心' })
		: null
))

const currentIdentityKey = () => {
	const identity = getReadIdentity()
	return `${identity.userId}:${identity.authGeneration}`
}

const getTypeIcon = (type) => {
	const map = {
		challenge: '🎯',
		tournament: '🏆',
		friend_request: '👥',
		follow: '⭐',
		match_result: '🏁',
		rank_change: '📊',
		season_rollover: '🔄',
		system: '📢'
	}
	return map[type] || '🔔'
}

const getTypeClass = (type) => {
	return 'type-' + (type || 'system')
}

const parseNotificationData = (data) => {
	if (!data) return {}
	if (typeof data === 'object') return data
	try {
		return JSON.parse(data)
	} catch (error) {
		console.error('解析通知数据失败:', error)
		return {}
	}
}

const navigateByNotification = (item) => {
	const data = parseNotificationData(item.data)
	const target = data.target || data.url || ''

	if (target === '/subPages/match/matchResult' && data.match_id) {
		uni.navigateTo({ url: `${target}?match_id=${data.match_id}&from=history` })
		return
	}

	if (target) {
		uni.navigateTo({ url: target })
		return
	}

	// 约球通知：只打开权威详情，不替用户进入或开局。
	if (item.type === 'challenge' && data.challenge_id) {
		uni.navigateTo({ url: `/subPages/match/challengeWaiting?challenge_id=${data.challenge_id}` })
		return
	}

	if (item.type === 'friend_request') {
		uni.navigateTo({ url: '/subPages/social/friendRequests' })
	}
}

const goLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}

const openEntry = (entry) => {
	uni.navigateTo({ url: entry.url })
}

const handleTapNotification = async (item) => {
	const requestIdentityKey = currentIdentityKey()
	if (!item.is_read) {
		try {
			await markAsRead({ notification_id: item.id })
			if (currentIdentityKey() !== requestIdentityKey) return
			item.is_read = true
			notificationStore.decrementUnread(getReadIdentity())
			if (item.type === 'friend_request' && item.title === '收到好友申请') {
				friendRequestStore.fetchPendingCount(getReadIdentity())
			}
		} catch (e) {
			console.error('标记已读失败:', e)
		}
	}

	navigateByNotification(item)
}

const handleReadAll = async () => {
	const requestIdentityKey = currentIdentityKey()
	try {
		await markAllAsRead()
		if (currentIdentityKey() !== requestIdentityKey) return
		notificationList.value.forEach(item => {
			item.is_read = true
		})
		notificationStore.clearUnread(getReadIdentity())
		uni.showToast({ title: '已全部标记为已读', icon: 'success' })
	} catch (e) {
		console.error('全部已读失败:', e)
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const handleDelete = async (id) => {
	const requestIdentityKey = currentIdentityKey()
	try {
		await deleteNotification({ notification_id: id })
		if (currentIdentityKey() !== requestIdentityKey) return
		const idx = notificationList.value.findIndex(n => n.id === id)
		if (idx >= 0) {
			const item = notificationList.value[idx]
			if (!item.is_read) {
				notificationStore.decrementUnread(getReadIdentity())
			}
			notificationList.value.splice(idx, 1)
		}
		uni.showToast({ title: '已删除', icon: 'success' })
	} catch (e) {
		console.error('删除通知失败:', e)
		uni.showToast({ title: '删除失败', icon: 'none' })
	}
}

// 加载进行中又收到刷新请求时先记下，当前请求结束后补一次，避免漏掉期间到达的新通知。
let reloadQueued = false

const loadNotifications = async (isRefresh = false) => {
	if (loading.value) {
		if (isRefresh) reloadQueued = true
		return
	}
	const requestId = latestRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	const requestScopeVersion = userDataInvalidationStore.versionOf('notification')
	const nextState = beginAsyncPageLoad(notificationState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: []
	})
	const pageRequest = getAsyncPageRequest(nextState)
	latestRequestId.value = requestId
	requestedIdentityKey.value = requestIdentityKey
	notificationState.value = nextState
	loading.value = true

	if (isRefresh) {
		page.value = 1
		hasMore.value = true
	}

	try {
		const res = await getNotificationList({ page: page.value, page_size: pageSize })
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey || notificationState.value.requestId !== pageRequest.requestId || notificationState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res?.success) throw createRequestError({ message: res?.message || '加载消息失败', category: 'business' })
		const list = res.list || []
		if (Object.prototype.hasOwnProperty.call(res || {}, 'unread_count')) {
			notificationStore.setUnreadCount(res.unread_count, getReadIdentity())
		}

		if (isRefresh) {
			notificationList.value = list
			// 以首屏请求开始时的失效版本为准：加载期间到达的新通知会让下一次显示再刷新
			loadedScopeVersion.value = requestScopeVersion
		} else {
			notificationList.value = [...notificationList.value, ...list]
		}

		hasMore.value = list.length >= pageSize
		loadedIdentityKey.value = requestIdentityKey
		notificationState.value = resolveAsyncPageLoad(notificationState.value, pageRequest, { data: notificationList.value })
	} catch (e) {
		if (requestId !== latestRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		requestedIdentityKey.value = ''
		notificationState.value = rejectAsyncPageLoad(notificationState.value, pageRequest, e)
	} finally {
		if (requestId === latestRequestId.value) {
			loading.value = false
			if (reloadQueued) {
				reloadQueued = false
				loadNotifications(true)
			}
		}
	}
}

const loadMore = () => {
	if (!hasMore.value) return
	page.value++
	loadNotifications()
}

const retryNotifications = () => loadNotifications(true)

const handleNotificationErrorAction = () => {
	if (notificationState.value.error?.category === 'permission' || notificationState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryNotifications()
}

// 切号或退出：清空列表并作废在途请求，旧账号的迟到响应不再写入。
const resetList = () => {
	latestRequestId.value += 1
	notificationList.value = []
	page.value = 1
	hasMore.value = true
	loading.value = false
	reloadQueued = false
	requestedIdentityKey.value = ''
	loadedIdentityKey.value = ''
	loadedScopeVersion.value = -1
	notificationState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: [] })
}

let active = false
let refreshTimer = null

const clearRefreshTimer = () => {
	if (refreshTimer) clearTimeout(refreshTimer)
	refreshTimer = null
}

onShow(() => {
	active = true
	if (!userStore.isLoggedIn) {
		if (loadedIdentityKey.value || requestedIdentityKey.value || notificationList.value.length) resetList()
		return
	}
	const identityKey = currentIdentityKey()
	const decision = resolveNotificationReload({
		identityKey,
		loadedIdentityKey: loadedIdentityKey.value,
		requestedIdentityKey: requestedIdentityKey.value,
		loadedScopeVersion: loadedScopeVersion.value,
		scopeVersion: userDataInvalidationStore.versionOf('notification')
	})
	if (decision === 'load') {
		if (loadedIdentityKey.value !== identityKey) resetList()
		loadNotifications(true)
	}
	// 两个待处理数量取自共享活动快照，由 TTL / dirty 决定是否回源，不在这里强制刷新
	activityStore.fetch(getReadIdentity(), { silent: true })
})

onHide(() => {
	active = false
	clearRefreshTimer()
})

onUnmounted(() => {
	active = false
	clearRefreshTimer()
})

// 页面可见期间收到新通知：短时间内的多次失效合并为一次首屏刷新。
watch(() => userDataInvalidationStore.versionOf('notification'), () => {
	if (!active || !userStore.isLoggedIn) return
	clearRefreshTimer()
	refreshTimer = setTimeout(() => {
		refreshTimer = null
		if (active && userStore.isLoggedIn) loadNotifications(true)
	}, NOTIFICATION_REFRESH_DEBOUNCE_MS)
})

onPullDownRefresh(async () => {
	if (!userStore.isLoggedIn) {
		uni.stopPullDownRefresh()
		return
	}
	try {
		await Promise.allSettled([
			loadNotifications(true),
			activityStore.fetch(getReadIdentity(), { force: true, silent: true })
		])
	} finally {
		uni.stopPullDownRefresh()
	}
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
