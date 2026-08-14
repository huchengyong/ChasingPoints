<template>
	<view class="challenges-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="page-tip">
			<uni-icons type="info" size="16" color="#E0AE12"></uni-icons>
			<text>这里记录的是线上 PK 邀约，只用于社交互动，不会直接生成真实对局。</text>
		</view>

		<view class="summary-grid" v-if="!loading">
			<view class="summary-card">
				<text class="summary-value">{{ receivedCount }}</text>
				<text class="summary-label">待我回应</text>
			</view>
			<view class="summary-card">
				<text class="summary-value">{{ sentCount }}</text>
				<text class="summary-label">等待对方</text>
			</view>
			<view class="summary-card">
				<text class="summary-value">{{ respondedCount }}</text>
				<text class="summary-label">已回应</text>
			</view>
		</view>

		<!-- 标签切换 -->
		<view class="tab-bar">
			<view class="tab-item" :class="{ active: tab === 'received' }" @tap="switchTab('received')">
				<text>收到</text>
				<view v-if="receivedCount > 0" class="tab-badge">
					<text>{{ receivedCount }}</text>
				</view>
			</view>
			<view class="tab-item" :class="{ active: tab === 'sent' }" @tap="switchTab('sent')">
				<text>发出</text>
			</view>
			<view class="tab-item" :class="{ active: tab === 'responded' }" @tap="switchTab('responded')">
				<text>已回应</text>
			</view>
		</view>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- PK 列表 -->
		<view v-else-if="list.length > 0" class="challenge-list">
			<view v-for="item in list" :key="item.id" class="challenge-card" @tap="handleChallengeCardTap(item)">
				<view class="card-top">
					<image
						class="challenge-avatar"
						:src="resolveAvatarUrl(item.avatar, item.opponent_id)"
						mode="aspectFill"
					></image>
					<view class="challenge-info">
						<text class="challenge-name">{{ item.nickname || '球友' }}</text>
							<text class="challenge-game">{{ getGameTypeLabel(item.game_type, '台球') }}</text>
						<text class="challenge-direction">{{ getDirectionText(item) }}</text>
					</view>
					<view class="challenge-status" :class="'status-' + item.status">
						<text>{{ getChallengeStatusText(item) }}</text>
					</view>
				</view>
				<view v-if="item.message" class="challenge-message">
					<text>"{{ item.message }}"</text>
				</view>
				<view class="card-time">
					<text>{{ item.relativeTime }}</text>
				</view>
				<!-- 操作按钮 -->
				<view v-if="tab === 'received' && item.status === 0" class="card-actions">
					<view class="action-btn ghost-btn" @tap.stop="openPkReport(item)">
						<text>看报表</text>
					</view>
					<view class="action-btn reject-btn" @tap.stop="handleReject(item)">
						<text>拒绝</text>
					</view>
					<view class="action-btn accept-btn" @tap.stop="handleAccept(item)">
						<text>回应PK</text>
					</view>
					</view>
				<view v-else-if="item.status === 1 && !item.match_id" class="card-actions">
					<view class="action-btn accept-btn" @tap.stop="openOfflineStart(item)">
						<text>线下扫码开局</text>
					</view>
				</view>
				<view v-else-if="item.status === 1 && item.match_id" class="card-actions">
					<view class="action-btn linked-btn" @tap.stop="openLinkedMatch(item)">
						<text>查看对局</text>
					</view>
				</view>
				<view v-else class="card-link" @tap.stop="openPkReport(item)">
					<text>查看PK报表</text>
				</view>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-else class="empty-state">
			<text class="empty-icon">⚔️</text>
			<text class="empty-text">{{ emptyText }}</text>
		</view>

		<!-- PK 发起弹窗 -->
		<view v-if="showChallengeModal" class="modal-overlay" @tap="closeChallengeModal">
			<view class="modal-container" @tap.stop>
				<text class="modal-title">发起PK邀约</text>
				<text class="modal-subtitle">向 {{ targetFriend.nickname || '好友' }} 发起数据 PK，不会直接开赛</text>
				<view class="game-type-options">
					<view
						v-for="gt in gameTypes"
						:key="gt.value"
						class="game-type-option"
						:class="{ selected: selectedGameType === gt.value }"
						@tap="selectedGameType = gt.value"
					>
						<text>{{ gt.label }}</text>
					</view>
				</view>
				<input
					class="message-input"
					v-model="challengeMessage"
					placeholder="附言（选填）"
					maxlength="50"
				/>
				<view class="modal-actions">
					<view class="modal-btn cancel-btn" @tap="closeChallengeModal">
						<text>取消</text>
					</view>
					<view class="modal-btn confirm-btn" @tap="submitChallenge">
						<text>发送PK</text>
					</view>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { getPendingChallenges, acceptChallenge, rejectChallenge, sendChallenge } from '@/api/challenge.js'
import { formatRelativeTime } from '@/utils/format.js'
import { GAME_TYPE_OPTIONS, getGameTypeLabel } from '@/utils/game-types.js'
import { useUserStore } from '@/store/user.js'
import { buildChallengePayload, buildChallengeStartContext, normalizeChallengeListItem } from '@/utils/challenge-entry.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()

const statusMap = { 0: '待回应', 1: '已回应', 2: '已拒绝', 3: '已过期' }
const gameTypes = GAME_TYPE_OPTIONS
const userStore = useUserStore()

const tab = ref('received')
const allChallenges = ref([])
const loading = ref(true)
const challengesDirty = ref(true)
const receivedCount = computed(() => allChallenges.value.filter(item => item.direction === 'received' && item.status === 0).length)
const sentCount = computed(() => allChallenges.value.filter(item => item.direction === 'sent' && item.status === 0).length)
const respondedCount = computed(() => allChallenges.value.filter(item => item.status !== 0).length)
const list = computed(() => {
	if (tab.value === 'received') return allChallenges.value.filter(item => item.direction === 'received' && item.status === 0)
	if (tab.value === 'sent') return allChallenges.value.filter(item => item.direction === 'sent' && item.status === 0)
	return allChallenges.value.filter(item => item.status !== 0)
})

// 挑战弹窗
const showChallengeModal = ref(false)
const targetFriend = ref({})
const selectedGameType = ref(GAME_TYPE_OPTIONS[0].value)
const challengeMessage = ref('')

const fetchList = async ({ force = false } = {}) => {
	if (!force && !challengesDirty.value) return
	loading.value = allChallenges.value.length === 0
	try {
		const res = await getPendingChallenges({ page: 1, page_size: 50 })
		if (res.success) {
			allChallenges.value = (res.list || []).map(item => ({
				...normalizeChallengeListItem(item, userStore.userId),
				relativeTime: formatRelativeTime(item.created_at)
			}))
			challengesDirty.value = false
		}
	} catch (e) {
		console.error('获取挑战列表失败', e)
	} finally {
		loading.value = false
	}
}

const refreshChallenges = () => {
	challengesDirty.value = true
	return fetchList()
}

const switchTab = (newTab) => {
	tab.value = newTab
}

const getDirectionText = (item) => {
	if (item.status !== 0) {
		return item.direction === 'sent' ? '我发起的 PK 已有结果' : '对方发起的 PK 已有结果'
	}
	return item.direction === 'sent' ? '等待对方回应' : '等待我来回应'
}

const getChallengeStatusText = (item) => {
	if (item.status === 1 && item.match_id) return '已关联对局'
	return statusMap[item.status] || '待处理'
}

const handleChallengeCardTap = (item) => {
	if (item.status === 1 && item.match_id) {
		openLinkedMatch(item)
		return
	}
	openPkReport(item)
}

const openPkReport = (item) => {
	const query = []
	const opponentId = item.opponent_id || 0
	const opponentName = item.opponent_name || item.nickname || ''

	if (opponentId) {
		query.push(`opponent_id=${opponentId}`)
	}
	if (opponentName) {
		query.push(`opponent_name=${encodeURIComponent(opponentName)}`)
	}
	if (item.avatar) {
		query.push(`opponent_avatar=${encodeURIComponent(item.avatar)}`)
	}

	uni.navigateTo({ url: `/subPages/social/pkReport?${query.join('&')}` })
}

const handleAccept = async (item) => {
	try {
		const res = await acceptChallenge({ challenge_id: item.id })
		if (res.success) {
			uni.showToast({ title: '已回应PK邀约', icon: 'success' })
			uni.showModal({
				title: '邀约已接受',
				content: '请双方线下见面后扫码确认在场，正式对局才会创建真实战绩。',
				confirmText: '去线下开局',
				cancelText: '稍后处理',
				success: ({ confirm }) => {
					if (confirm) openOfflineStart({ ...item, status: 1 })
				}
			})
			refreshChallenges()
		} else {
			uni.showToast({ title: res.msg || '接受失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const openOfflineStart = (item) => {
	const context = buildChallengeStartContext(item)
	if (!context?.challenge_id || !context?.opponent_id) {
		uni.showToast({ title: item.match_id ? '该邀约已关联对局' : '邀约上下文不完整', icon: 'none' })
		return
	}
	uni.setStorageSync('pending_match_challenge', JSON.stringify(context))
	uni.switchTab({ url: '/pages/match/index' })
}

const openLinkedMatch = (item) => {
	const matchId = Number(item.match_id || 0)
	if (!matchId) return
	uni.navigateTo({ url: `/subPages/match/matchDetail?match_id=${matchId}` })
}

const handleReject = async (item) => {
	try {
		const res = await rejectChallenge({ challenge_id: item.id })
		if (res.success) {
			uni.showToast({ title: '已拒绝', icon: 'success' })
			refreshChallenges()
		}
	} catch (e) {
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

// 打开发起挑战弹窗（由其他页面通过事件触发）
const openChallengeModal = (friend) => {
	targetFriend.value = friend
	selectedGameType.value = GAME_TYPE_OPTIONS[0].value
	challengeMessage.value = ''
	showChallengeModal.value = true
}

const closeChallengeModal = () => {
	showChallengeModal.value = false
}

const emptyText = computed(() => {
	if (tab.value === 'received') return '暂无待回应的 PK 邀约'
	if (tab.value === 'sent') return '暂无等待中的 PK 邀约'
	return '暂无已回应的 PK 记录'
})

const submitChallenge = async () => {
	try {
		const payload = buildChallengePayload({
			targetFriend: targetFriend.value,
			gameType: selectedGameType.value,
			message: challengeMessage.value
		})
		if (!payload.to_user_id) {
			uni.showToast({ title: '好友信息异常', icon: 'none' })
			return
		}
		const res = await sendChallenge(payload)
		if (res.success) {
			uni.showToast({ title: 'PK邀约已发送', icon: 'success' })
			closeChallengeModal()
			refreshChallenges()
		} else {
			uni.showToast({ title: res.msg || '发送失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '发送失败', icon: 'none' })
	}
}

onMounted(() => {
	fetchList()
	// 监听来自好友列表的挑战事件
	uni.$on('openChallenge', openChallengeModal)
})

// 页面卸载时取消事件监听
onUnmounted(() => {
	uni.$off('openChallenge', openChallengeModal)
})
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
		color: #7c5b12;
	}
}
.summary-grid {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	gap: 16rpx;
	margin: 20rpx 24rpx 0;
}
.summary-card {
	background: #fff;
	border-radius: 20rpx;
	padding: 20rpx 16rpx;
	text-align: center;

	.summary-value {
		display: block;
		font-size: 40rpx;
		font-weight: 700;
		color: #C69200;
	}

	.summary-label {
		display: block;
		margin-top: 8rpx;
		font-size: 22rpx;
		color: #6E6242;
	}
}
.tab-bar {
	display: flex;
	background: #fff;
	margin-top: 20rpx;
	padding: 0 24rpx;
	.tab-item {
		flex: 1;
		text-align: center;
		padding: 24rpx 0;
		font-size: 28rpx;
		color: #6E6242;
		position: relative;
		&.active {
			color: #C69200;
			font-weight: 600;
			&::after {
				content: '';
				position: absolute;
				bottom: 0;
				left: 30%;
				right: 30%;
				height: 4rpx;
				background: #E0AE12;
				border-radius: 2rpx;
			}
		}
		.tab-badge {
			position: absolute;
			top: 12rpx;
			right: 20%;
			background: #ef4444;
			border-radius: 20rpx;
			min-width: 32rpx;
			height: 32rpx;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 0 8rpx;
			font-size: 20rpx;
			color: #fff;
		}
	}
}
.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 50vh;
	.loading-text { font-size: 28rpx; color: #9A8C67; margin-top: 16rpx; }
}
.challenge-list {
	padding: 20rpx 24rpx;
}
.challenge-card {
	background: #fff;
	border-radius: 16rpx;
	padding: 24rpx;
	margin-bottom: 16rpx;
	.card-top {
		display: flex;
		align-items: center;
		.challenge-avatar {
			width: 80rpx;
			height: 80rpx;
			border-radius: 50%;
			margin-right: 16rpx;
			background: #E9E2CF;
		}
		.challenge-info {
			flex: 1;
			.challenge-name { font-size: 30rpx; font-weight: 500; color: #231C0B; display: block; }
			.challenge-game { font-size: 24rpx; color: #9A8C67; margin-top: 4rpx; }
			.challenge-direction { font-size: 22rpx; color: #9A8C67; margin-top: 4rpx; display: block; }
		}
		.challenge-status {
			padding: 6rpx 16rpx;
			border-radius: 8rpx;
			font-size: 22rpx;
			&.status-0 { background: #fef3c7; color: #d97706; }
			&.status-1 { background: #dcfce7; color: #16a34a; }
			&.status-2 { background: #fee2e2; color: #dc2626; }
			&.status-3 { background: #FAF8F2; color: #9A8C67; }
		}
	}
	.challenge-message {
		margin-top: 12rpx;
		padding: 12rpx 16rpx;
		background: #FAF8F2;
		border-radius: 8rpx;
		font-size: 26rpx;
		color: #6E6242;
		font-style: italic;
	}
	.card-time {
		margin-top: 12rpx;
		font-size: 22rpx;
		color: #9A8C67;
	}
	.card-link {
		margin-top: 16rpx;
		display: inline-flex;

		text {
			font-size: 22rpx;
			font-weight: 500;
			color: #C69200;
		}
	}
	.card-actions {
		display: flex;
		gap: 12rpx;
		margin-top: 16rpx;
		.action-btn {
			flex: 1;
			text-align: center;
			padding: 16rpx;
			border-radius: 10rpx;
			font-size: 28rpx;
		}
		.reject-btn { background: #FAF8F2; color: #6E6242; }
		.accept-btn { background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%); color: #231C0B; font-weight: 600; }
		.ghost-btn { background: rgba(224, 174, 18, 0.12); color: #C69200; }
		.linked-btn { background: #1E180D; color: #ffffff; font-weight: 600; }
	}
}
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 50vh;
	.empty-icon { font-size: 80rpx; margin-bottom: 16rpx; }
	.empty-text { font-size: 28rpx; color: #9A8C67; }
}
.modal-overlay {
	position: fixed;
	top: 0; left: 0; right: 0; bottom: 0;
	z-index: 999;
	background: rgba(0,0,0,0.5);
	display: flex;
	align-items: center;
	justify-content: center;
}
.modal-container {
	width: 600rpx;
	background: #fff;
	border-radius: 24rpx;
	padding: 40rpx;
	.modal-title { font-size: 34rpx; font-weight: 700; color: #231C0B; display: block; text-align: center; }
	.modal-subtitle { font-size: 26rpx; color: #9A8C67; display: block; text-align: center; margin: 12rpx 0 24rpx; }
	.game-type-options {
		display: flex;
		gap: 12rpx;
		margin-bottom: 20rpx;
		.game-type-option {
			flex: 1;
			text-align: center;
			padding: 16rpx;
			border-radius: 10rpx;
			background: #FAF8F2;
			font-size: 26rpx;
			color: #6E6242;
			&.selected { background: #E0AE12; color: #231C0B; }
		}
	}
	.message-input {
		width: 100%;
		padding: 16rpx;
		border: 2rpx solid #E9E2CF;
		border-radius: 10rpx;
		font-size: 28rpx;
		margin-bottom: 24rpx;
	}
	.modal-actions {
		display: flex;
		gap: 16rpx;
		.modal-btn {
			flex: 1;
			text-align: center;
			padding: 20rpx;
			border-radius: 12rpx;
			font-size: 28rpx;
		}
		.cancel-btn { background: #FAF8F2; color: #6E6242; }
		.confirm-btn { background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%); color: #231C0B; font-weight: 600; }
	}
}

.challenges-page.dark-mode {
	background: #141109;

	.page-tip {
		background: rgba(224, 174, 18, 0.18);

		text {
			color: #f7e7a8;
		}
	}

	.summary-card,
	.tab-bar,
	.challenge-card,
	.modal-container {
		background: #1e180d;
	}

	.summary-label,
	.tab-item,
	.challenge-game,
	.loading-text,
	.empty-text,
	.modal-subtitle {
		color: #9f926e;
	}

	.challenge-card {
		.challenge-avatar,
		.challenge-status.status-3,
		.reject-btn {
			background: #3a2e16;
		}

		.challenge-name {
			color: #fff7e1;
		}

		.challenge-message {
			background: #2a2110;
			color: #d7c89b;
		}
	}

	.modal-container {
		.modal-title {
			color: #fff7e1;
		}

		.game-type-option {
			background: #3a2e16;
			color: #d7c89b;
		}

		.message-input {
			background: #2a2110;
			border-color: #3a2e16;
			color: #fff7e1;
		}

		.cancel-btn {
			background: #3a2e16;
			color: #d7c89b;
		}
	}
}
</style>
