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
					<text class="status-desc">这里直接看 TA 的公开战绩，想看你们之间的对比就去 PK 报表。</text>
				</view>

				<view class="hero-insight">
					<text class="insight-label">好友近况</text>
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

				<view class="hero-action-grid">
					<button class="primary-btn hero-primary-btn" @tap="goToPkReport">
						<text>PK 报表</text>
					</button>
					<button class="secondary-btn" @tap="goToHonorWall">
						<text>查看荣誉墙</text>
					</button>
				</view>
			</view>

			<view class="section-card">
				<view class="section-header">
					<text class="section-title">{{ summary.battleSectionTitle }}</text>
					<text class="section-tip">{{ summary.battleSectionTip }}</text>
				</view>

				<view v-if="battleListHidden" class="battle-empty-card">
					<text class="battle-empty-title">{{ summary.hiddenTitle }}</text>
					<text class="battle-empty-desc">{{ summary.hiddenDesc }}</text>
				</view>

				<view v-else-if="battleCardViewModels.length === 0" class="battle-empty-card">
					<text class="battle-empty-title">{{ summary.emptyTitle }}</text>
					<text class="battle-empty-desc">{{ summary.emptyDesc }}</text>
				</view>

				<view v-else class="battle-list">
					<view
						v-for="item in battleCardViewModels"
						:key="item.id || item.name"
						class="battle-item"
						:class="item.toneClass"
						@tap="goToBattleDetail(item)"
					>
						<view class="battle-left">
							<image
								v-if="item.avatar"
								class="battle-avatar"
								:src="item.avatar"
								mode="aspectFill"
							/>
							<view v-else class="battle-avatar battle-avatar-placeholder">
								<text>{{ getBattleAvatarText(item.name) }}</text>
							</view>
							<view class="battle-copy">
								<view class="battle-name-row">
									<text class="battle-name">{{ item.name || '对手' }}</text>
									<view v-if="item.isMeOpponent" class="me-badge">
										<text>我</text>
									</view>
									<view class="battle-badge" :class="item.toneClass">
										<text>{{ item.relationshipBadge }}</text>
									</view>
								</view>
								<text class="battle-desc">{{ item.relationshipText }}</text>
								<view class="battle-meta-row">
									<text class="battle-meta">{{ item.recordText }}</text>
									<text class="battle-meta">{{ item.lastMatchText }}</text>
								</view>
							</view>
						</view>

						<view class="battle-right">
							<text class="battle-win-rate" :class="item.toneClass">{{ item.winRateText }}</text>
							<text class="battle-sample">{{ item.sampleText }}</text>
							<text class="battle-link">查看交锋</text>
						</view>
					</view>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { getOpponentList } from '@/api/match.js'
import { buildFriendPkReportUrl } from '@/utils/friend-entry.js'
import { buildHonorWallUrl } from '@/utils/honor-wall.js'
import {
	buildOpponentCardViewModels,
	buildOpponentH2HUrl,
	buildOpponentRecordRequestParams
} from '@/utils/opponent-record.js'
import {
	buildFriendHomepageSummary,
	normalizeFriendHomepageOptions
} from '@/utils/friend-homepage.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

const loading = ref(true)
const loadFailed = ref(false)
const loadErrorText = ref('加载好友主页失败，请稍后再试。')
const lastMatchAt = ref('')
const battleListHidden = ref(false)
const battleOpponents = ref([])

const friendProfile = reactive({
	id: 0,
	name: '',
	avatar: '',
	rankName: ''
})

const stats = reactive({
	total_matches: 0,
	total_wins: 0
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

const battleCardViewModels = computed(() => buildOpponentCardViewModels({
	opponents: battleOpponents.value,
	subjectName: friendProfile.name || 'TA',
	currentUserId: userStore.userId,
	showWinRateLabel: true
}))

const getBattleAvatarText = (name = '') => {
	if (!name) return '友'
	return String(name).slice(0, 1).toUpperCase()
}

const loadData = async () => {
	if (!friendProfile.id) {
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
		const response = await getOpponentList(buildOpponentRecordRequestParams({
			page: 1,
			pageSize: 100,
			targetUserId: friendProfile.id
		}))
		const list = Array.isArray(response.list) ? response.list : []
		battleListHidden.value = Boolean(response.hidden || response.is_hidden || response.message === '对方已隐藏战绩')
		battleOpponents.value = battleListHidden.value ? [] : list
		stats.total_matches = list.reduce((sum, item = {}) => sum + Number(item.total_matches || 0), 0)
		stats.total_wins = Number(response.total_wins || 0)
		lastMatchAt.value = list[0]?.last_match_at || ''
	} catch (error) {
		loadFailed.value = true
		console.error('加载好友主页失败:', error)
		uni.showToast({ title: '加载好友主页失败', icon: 'none' })
	} finally {
		loading.value = false
	}
}

const goToBattleDetail = (item) => {
	uni.navigateTo({
		url: buildOpponentH2HUrl({
			opponent: item,
			targetUserId: friendProfile.id,
			targetName: friendProfile.name,
			targetAvatar: friendProfile.avatar
		})
	})
}

const goToPkReport = () => {
	uni.navigateTo({ url: buildFriendPkReportUrl(friendPayload.value) })
}

const goToHonorWall = () => {
	uni.navigateTo({
		url: buildHonorWallUrl({ userId: friendProfile.id, gameType: 3 })
	})
}

onLoad((options) => {
	const normalized = normalizeFriendHomepageOptions(options)
	friendProfile.id = normalized.opponentId
	friendProfile.name = normalized.opponentName
	friendProfile.avatar = normalized.opponentAvatar
	friendProfile.rankName = normalized.rankName
})

onShow(() => {
	loadData()
})
</script>

<style lang="scss" scoped>
@import './friendHomepage.scss';
</style>
