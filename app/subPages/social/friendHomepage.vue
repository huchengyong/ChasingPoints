<template>
	<view class="friend-homepage-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载好友主页中...</text>
		</view>

		<view v-else-if="loadFailed" class="error-state">
			<uni-icons type="info-filled" size="38" color="#E0AE12"></uni-icons>
			<text class="error-title">好友主页暂时没加载出来</text>
			<text class="error-text">{{ loadErrorText }}</text>
			<button class="retry-btn" @tap="loadData">
				<text>重新加载</text>
			</button>
		</view>

		<scroll-view v-else scroll-y class="page-scroll">
			<view class="hero-card">
				<text class="hero-kicker">好友名片</text>

				<view class="identity-row">
					<image
						v-if="friendProfile.avatar"
						class="avatar"
						:src="friendProfile.avatar"
						mode="aspectFill"
					/>
					<view v-else class="avatar avatar-placeholder">
						<text>{{ avatarText }}</text>
					</view>
					<view class="identity-content">
						<view class="name-row">
							<text class="friend-name">{{ friendProfile.name || '球友' }}</text>
							<text v-if="friendProfile.rankName" class="friend-rank">{{ friendProfile.rankName }}</text>
						</view>
						<text class="friend-id">ID: {{ friendProfile.id || '--' }}</text>
					</view>
				</view>

				<view class="status-row">
					<view class="status-chip">
						<text>已是好友</text>
					</view>
					<text class="status-desc">你们已经是好友了，接下来的每一场都会慢慢写进这份战绩。</text>
				</view>

				<view class="hero-insight">
					<text class="insight-label">关系里的真实战绩</text>
					<text class="insight-title">{{ summary.heroTitle }}</text>
					<text class="insight-desc">{{ summary.heroDesc }}</text>
				</view>

				<view class="hero-facts">
					<view class="fact-chip">
						<text>{{ summary.totalMatchesText }}</text>
					</view>
					<view class="fact-chip">
						<text>{{ summary.recordText }}</text>
					</view>
					<view class="fact-chip">
						<text>{{ summary.recentMatchText }}</text>
					</view>
				</view>
			</view>

			<view class="section-card">
				<view class="section-header">
					<text class="section-title">继续查看</text>
					<text class="section-tip">查完整记录 or 看对比</text>
				</view>

				<view class="action-list">
					<button class="primary-btn" @tap="goToH2H">
						<text>查看对方战绩</text>
					</button>
					<button class="secondary-btn" @tap="goToPkReport">
						<text>PK 报表</text>
					</button>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'
import { getH2HHistory, getH2HStats } from '@/api/match.js'
import { buildFriendOpponentRecordUrl, buildFriendPkReportUrl } from '@/utils/friend-entry.js'
import {
	buildFriendHomepageSummary,
	fetchFriendHomepageData,
	normalizeFriendHomepageOptions
} from '@/utils/friend-homepage.js'

const themeStore = useThemeStore()

const isDarkMode = computed(() => themeStore.isDarkMode)
const loading = ref(true)
const loadFailed = ref(false)
const loadErrorText = ref('加载好友主页失败，请稍后再试。')
const lastMatchAt = ref('')

const friendProfile = reactive({
	id: 0,
	name: '',
	avatar: '',
	rankName: ''
})

const stats = reactive({
	total_matches: 0,
	my_wins: 0,
	opponent_wins: 0
})

const summary = computed(() => buildFriendHomepageSummary({
	stats,
	lastMatchAt: lastMatchAt.value
}))

const avatarText = computed(() => {
	if (!friendProfile.name) return '友'
	return friendProfile.name.slice(0, 1).toUpperCase()
})

const friendPayload = computed(() => ({
	user_id: friendProfile.id,
	nickname: friendProfile.name || '球友',
	avatar: friendProfile.avatar || '',
	rank_name: friendProfile.rankName || ''
}))

const resolveQueryParams = () => {
	const params = {}

	if (friendProfile.id > 0) {
		params.opponent_id = friendProfile.id
	} else if (friendProfile.name) {
		params.opponent_name = friendProfile.name
	}

	return params
}

const loadData = async () => {
	const params = resolveQueryParams()
	if (!params.opponent_id && !params.opponent_name) {
		loadFailed.value = true
		loadErrorText.value = '好友信息异常，请返回好友列表后重试。'
		loading.value = false
		uni.showToast({ title: '好友信息异常', icon: 'none' })
		return
	}

	loading.value = true
	loadFailed.value = false
	loadErrorText.value = '加载好友主页失败，请稍后再试。'

	try {
		const { statsRes, historyRes } = await fetchFriendHomepageData({
			params,
			getStats: getH2HStats,
			getHistory: getH2HHistory
		})

		if (statsRes.opponent) {
			friendProfile.id = statsRes.opponent.id || friendProfile.id
			friendProfile.name = statsRes.opponent.name || friendProfile.name
			friendProfile.avatar = statsRes.opponent.avatar || friendProfile.avatar
			friendProfile.rankName = statsRes.opponent.rank_name || friendProfile.rankName
		}

		if (statsRes.stats) {
			stats.total_matches = statsRes.stats.total_matches || 0
			stats.my_wins = statsRes.stats.my_wins || 0
			stats.opponent_wins = statsRes.stats.opponent_wins || 0
		}

		lastMatchAt.value = historyRes.list?.[0]?.match_time || ''
	} catch (error) {
		loadFailed.value = true
		console.error('加载好友主页失败:', error)
		uni.showToast({ title: '加载好友主页失败', icon: 'none' })
	} finally {
		loading.value = false
	}
}

const goToH2H = () => {
	uni.navigateTo({ url: buildFriendOpponentRecordUrl(friendPayload.value) })
}

const goToPkReport = () => {
	uni.navigateTo({ url: buildFriendPkReportUrl(friendPayload.value) })
}

onLoad((options) => {
	const normalized = normalizeFriendHomepageOptions(options)
	friendProfile.id = normalized.opponentId
	friendProfile.name = normalized.opponentName
	friendProfile.avatar = normalized.opponentAvatar
	friendProfile.rankName = normalized.rankName
})

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	loadData()
})
</script>

<style lang="scss" scoped>
@import './friendHomepage.scss';
</style>
