<template>
	<view class="match-history-container" :class="{ 'dark-mode': isDarkMode }">
		<!-- 需要登录状态 -->
		<view class="login-required" v-if="needLogin">
			<uni-icons type="locked" size="64" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
			<text class="hint-text">请登录后查看对局记录</text>
			<button class="login-btn" @click="goLogin">去登录</button>
		</view>

		<!-- 分栏切换 -->
		<view class="tab-bar" v-if="!needLogin">
			<view
				class="tab-item"
				:class="{ active: activeTab === 'player' }"
				@click="switchTab('player')"
			>
				<text>参赛</text>
			</view>
			<view
				class="tab-item"
				:class="{ active: activeTab === 'referee' }"
				@click="switchTab('referee')"
			>
				<text>执裁</text>
			</view>
		</view>

		<!-- 参赛列表（复用原列表） -->
		<scroll-view
			v-if="!needLogin && activeTab === 'player'"
			class="match-list"
			scroll-y
			:refresher-enabled="true"
			:refresher-triggered="isRefreshing"
			@refresherrefresh="onRefresh"
			@scrolltolower="onLoadMore"
		>
			<view class="loading-wrapper" v-if="isLoading && matchList.length === 0">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<view class="empty-wrapper" v-else-if="!isLoading && matchList.length === 0">
				<uni-icons type="list" size="64" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
				<text class="empty-text">暂无对局记录</text>
				<text class="empty-hint">快去发起一场PK吧！</text>
			</view>

			<view class="match-items" v-else>
				<view class="match-item" v-for="match in matchList" :key="'p-'+match.id" @click="handleMatchDetail(match)">
					<view class="avatar-wrapper">
						<image class="avatar" :src="resolveAvatarUrl(match.opponent_avatar, match.opponent_id)" mode="aspectFill"/>
					</view>
					<view class="match-info">
						<view class="name-row">
							<text class="opponent-name">{{ match.opponent_name }}</text>
							<view :class="['game-type-tag', getGameTypeClass(match.game_type)]">
								<text>{{ match.game_type_name }}</text>
							</view>
						</view>
						<text class="match-time">{{ formatMatchTime(match.match_time) }}</text>
					</view>
					<view class="match-result">
						<view class="result-badge" :class="getResultClass(match.result)">
							<text class="result-text">{{ getResultText(match.result) }}</text>
						</view>
						<text class="score">{{ match.my_score }} - {{ match.opponent_score }}</text>
					</view>
					<view class="arrow-wrapper">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
					</view>
				</view>
			</view>

			<view class="load-more" v-if="matchList.length > 0">
				<text v-if="isLoadingMore" class="load-more-text">加载中...</text>
				<text v-else-if="!hasMore" class="load-more-text">没有更多了</text>
			</view>
		</scroll-view>

		<!-- 执裁列表 -->
		<scroll-view
			v-if="!needLogin && activeTab === 'referee'"
			class="match-list"
			scroll-y
			:refresher-enabled="true"
			:refresher-triggered="isRefereeRefreshing"
			@refresherrefresh="onRefereeRefresh"
			@scrolltolower="onRefereeLoadMore"
		>
			<view class="loading-wrapper" v-if="isRefereeLoading && refereeList.length === 0">
				<uni-icons type="spinner-cycle" size="40" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
				<text class="loading-text">加载中...</text>
			</view>

			<!-- 继续执裁卡片 -->
			<view class="continue-referee-card" v-if="ongoingRefereeMatch" @click="handleContinueReferee">
				<view class="continue-left">
					<uni-icons type="refreshicon" size="36" color="#E0AE12"></uni-icons>
					<text class="continue-text">继续执裁</text>
				</view>
				<view class="continue-right">
					<text class="continue-players">{{ ongoingRefereeMatch.player1_name }} VS {{ ongoingRefereeMatch.player2_name }}</text>
					<text class="continue-score">{{ ongoingRefereeMatch.player1_score }} - {{ ongoingRefereeMatch.player2_score }}</text>
				</view>
			</view>

			<!-- 空状态 -->
			<view class="empty-wrapper" v-else-if="!isRefereeLoading && refereeList.length === 0 && !ongoingRefereeMatch">
				<uni-icons type="person" size="64" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
				<text class="empty-text">暂无执裁记录</text>
				<text class="empty-hint">去对局首页扫码担任裁判</text>
				<button class="go-match-btn" @click="goMatchHall">前往对局大厅</button>
			</view>

			<!-- 执裁记录列表 -->
			<view class="match-items" v-if="refereeList.length > 0">
				<view class="referee-item" v-for="item in refereeList" :key="'r-'+item.id" @click="handleRefereeDetail(item)">
					<view class="referee-players">
						<view class="referee-player">
							<image class="referee-avatar" :src="resolveAvatarUrl(item.player1_avatar, item.player1_id)" mode="aspectFill"/>
							<text class="referee-name">{{ item.player1_name }}</text>
						</view>
						<view class="referee-vs">
							<text class="vs-score">{{ item.player1_score }} - {{ item.player2_score }}</text>
							<text class="vs-label">VS</text>
						</view>
						<view class="referee-player">
							<image class="referee-avatar" :src="resolveAvatarUrl(item.player2_avatar, item.player2_id)" mode="aspectFill"/>
							<text class="referee-name">{{ item.player2_name }}</text>
						</view>
					</view>
					<view class="referee-meta">
						<view class="referee-tags">
							<view :class="['game-type-tag', getGameTypeClass(item.game_type)]">
								<text>{{ item.game_type_name }}</text>
							</view>
							<view :class="['status-tag', item.status === 3 ? 'cancelled' : 'completed']">
								<text>{{ item.status_text }}</text>
							</view>
						</view>
						<text class="referee-duration" v-if="item.referee_duration_seconds > 0">
							执裁 {{ formatDuration(item.referee_duration_seconds) }}
						</text>
					</view>
				</view>
			</view>

			<view class="load-more" v-if="refereeList.length > 0">
				<text v-if="isRefereeLoadingMore" class="load-more-text">加载中...</text>
				<text v-else-if="!hasRefereeMore" class="load-more-text">没有更多了</text>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { getMatchList, getCurrentMatch, getRefereeHistory } from '@/api/match.js'
import { formatRelativeTime } from '@/utils/format.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import { formatDuration } from '@/utils/match-referee-view.js'
import { useUserStore } from '@/store/user.js'
import { useUserDataInvalidationStore } from '@/store/userDataInvalidation.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const userDataInvalidationStore = useUserDataInvalidationStore()

const currentReadIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const currentIdentityKey = () => {
	const identity = currentReadIdentity()
	return `${identity.userId}:${identity.authGeneration}`
}

// Tab
const activeTab = ref('player')

// Player state
const isLoading = ref(false)
const isRefreshing = ref(false)
const isLoadingMore = ref(false)
const hasMore = ref(true)
const needLogin = ref(false)
const matchList = ref([])
const currentPage = ref(1)
const pageSize = 20
const total = ref(0)

// Referee state
const isRefereeLoading = ref(false)
const isRefereeRefreshing = ref(false)
const isRefereeLoadingMore = ref(false)
const hasRefereeMore = ref(true)
const refereeList = ref([])
const refereePage = ref(1)
const refereeTotal = ref(0)
const ongoingRefereeMatch = ref(null)
const latestMatchRequestId = ref(0)
const latestRefereeRequestId = ref(0)
const latestCurrentMatchRequestId = ref(0)
const loadedIdentityKey = ref('')
const loadedHistoryScopeVersion = ref(0)

const resetHistoryForIdentity = () => {
	matchList.value = []
	currentPage.value = 1
	total.value = 0
	hasMore.value = true
	refereeList.value = []
	refereePage.value = 1
	refereeTotal.value = 0
	hasRefereeMore.value = true
	ongoingRefereeMatch.value = null
	isLoading.value = false
	isLoadingMore.value = false
	isRefereeLoading.value = false
	isRefereeLoadingMore.value = false
}

onShow(() => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		needLogin.value = true
		return
	}
	needLogin.value = false
	const identityKey = currentIdentityKey()
	const scopeVersion = userDataInvalidationStore.versionOf('history')
	if (loadedIdentityKey.value !== identityKey) resetHistoryForIdentity()
	if (activeTab.value === 'player' && (matchList.value.length === 0 || loadedHistoryScopeVersion.value !== scopeVersion) && !isLoading.value) {
		fetchMatchList()
	} else if (activeTab.value === 'referee' && (refereeList.value.length === 0 || loadedHistoryScopeVersion.value !== scopeVersion) && !isRefereeLoading.value) {
		fetchRefereeList()
	}
	// Always check for ongoing referee match
	checkOngoingRefereeMatch()
})

const switchTab = (tab) => {
	activeTab.value = tab
	if (tab === 'referee' && refereeList.value.length === 0 && !isRefereeLoading.value) {
		fetchRefereeList()
	}
	if (tab === 'player' && matchList.value.length === 0 && !isLoading.value) {
		fetchMatchList()
	}
}

const fetchMatchList = async (isRefresh = false, isLoadMore = false) => {
	if (isLoading.value || isLoadingMore.value) return
	const requestId = latestMatchRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	const requestScopeVersion = userDataInvalidationStore.versionOf('history')
	latestMatchRequestId.value = requestId
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
		const res = await getMatchList({ page: currentPage.value, page_size: pageSize })
		if (requestId !== latestMatchRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		const list = res.list || []
		total.value = res.total || 0
		if (isRefresh) { matchList.value = list }
		else if (isLoadMore) { matchList.value = [...matchList.value, ...list] }
		else { matchList.value = list }
		hasMore.value = matchList.value.length < total.value
		loadedIdentityKey.value = requestIdentityKey
		loadedHistoryScopeVersion.value = requestScopeVersion
	} catch (error) {
		if (requestId !== latestMatchRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		console.error('获取对局列表失败:', error)
	} finally {
		if (requestId !== latestMatchRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		isLoading.value = false
		isRefreshing.value = false
		isLoadingMore.value = false
	}
}

const onRefresh = () => fetchMatchList(true, false)
const onLoadMore = () => fetchMatchList(false, true)

// Referee history
const fetchRefereeList = async (isRefresh = false, isLoadMore = false) => {
	if (isRefereeLoading.value || isRefereeLoadingMore.value) return
	const requestId = latestRefereeRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	const requestScopeVersion = userDataInvalidationStore.versionOf('history')
	latestRefereeRequestId.value = requestId
	if (isRefresh) {
		isRefereeRefreshing.value = true
		refereePage.value = 1
		hasRefereeMore.value = true
	} else if (isLoadMore) {
		if (!hasRefereeMore.value) return
		isRefereeLoadingMore.value = true
		refereePage.value++
	} else {
		isRefereeLoading.value = true
	}
	try {
		const res = await getRefereeHistory({ page: refereePage.value, page_size: pageSize })
		if (requestId !== latestRefereeRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		const list = res.list || []
		refereeTotal.value = res.total || 0
		if (isRefresh) { refereeList.value = list }
		else if (isLoadMore) { refereeList.value = [...refereeList.value, ...list] }
		else { refereeList.value = list }
		hasRefereeMore.value = refereeList.value.length < refereeTotal.value
		loadedIdentityKey.value = requestIdentityKey
		loadedHistoryScopeVersion.value = requestScopeVersion
	} catch (error) {
		if (requestId !== latestRefereeRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		console.error('获取执裁历史失败:', error)
	} finally {
		if (requestId !== latestRefereeRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		isRefereeLoading.value = false
		isRefereeRefreshing.value = false
		isRefereeLoadingMore.value = false
	}
}

const onRefereeRefresh = () => fetchRefereeList(true, false)
const onRefereeLoadMore = () => fetchRefereeList(false, true)

// Check for ongoing referee match
const checkOngoingRefereeMatch = async () => {
	const requestId = latestCurrentMatchRequestId.value + 1
	const requestIdentityKey = currentIdentityKey()
	latestCurrentMatchRequestId.value = requestId
	try {
		const res = await getCurrentMatch({ silent: true })
		if (requestId !== latestCurrentMatchRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		if (res?.match?.viewer_role === 'referee') {
			ongoingRefereeMatch.value = {
				match_id: res.match.id,
				player1_name: res.match.player1_name,
				player1_avatar: res.match.player1_avatar,
				player2_name: res.match.player2_name,
				player2_avatar: res.match.player2_avatar,
				player1_score: res.match.my_score,
				player2_score: res.match.opponent_score,
				game_type: res.match.game_type
			}
		}
	} catch (error) {
		if (requestId !== latestCurrentMatchRequestId.value || currentIdentityKey() !== requestIdentityKey) return
		// 静默处理，无进行中裁判对局
	}
}

const handleContinueReferee = () => {
	if (ongoingRefereeMatch.value?.match_id) {
		uni.navigateTo({ url: `/subPages/match/playing?match_id=${ongoingRefereeMatch.value.match_id}` })
	}
}

// Time and result helpers
const formatMatchTime = (time) => formatRelativeTime(time)

const getResultClass = (result) => {
	switch (result) {
		case 1: return 'win'
		case 2: return 'lose'
		case 3: return 'draw'
		default: return ''
	}
}

const getResultText = (result) => {
	switch (result) {
		case 1: return '胜利'
		case 2: return '失败'
		case 3: return '平局'
		default: return '未知'
	}
}

const getGameTypeClass = (gameType) => {
	switch (gameType) {
		case 1: return 'snooker'
		case 2:
		case 4: return 'american-9ball'
		case 3:
		default: return 'chinese-8ball'
	}
}

const goLogin = () => { uni.navigateTo({ url: '/pages/login/login' }) }
const goMatchHall = () => { uni.switchTab({ url: '/pages/index/index' }) }

const handleMatchDetail = (match) => {
	uni.navigateTo({ url: `/subPages/match/matchResult?match_id=${match.id}&from=history` })
}

const handleRefereeDetail = (item) => {
	uni.navigateTo({ url: `/subPages/match/matchResult?match_id=${item.id}&from=history&role=referee` })
}
</script>

<style lang="scss" scoped>
@import './matchHistory.scss';
</style>
