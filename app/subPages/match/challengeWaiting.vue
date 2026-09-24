<template>
	<view class="waiting-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-text">加载中…</text>
		</view>

		<view v-else-if="!challenge" class="state-block">
			<text class="state-text">暂时无法确认约球状态</text>
			<button class="secondary-btn" @tap="loadDetail">重新加载</button>
		</view>

		<view v-else class="main-block">
			<view class="vs-row">
				<view class="player">
					<image class="avatar" :src="resolveAvatarUrl(challenge.from_avatar, challenge.from_user_id)" mode="aspectFill"></image>
					<text class="player-state" :class="{ active: isWaiting(who.from) }">{{ isWaiting(who.from) ? '已进入' : '待进入' }}</text>
				</view>
				<text class="vs-label">VS</text>
				<view class="player">
					<image class="avatar" :src="resolveAvatarUrl(challenge.to_avatar, challenge.to_user_id)" mode="aspectFill"></image>
					<text class="player-state" :class="{ active: isWaiting(who.to) }">{{ isWaiting(who.to) ? '已进入' : '待进入' }}</text>
				</view>
			</view>
			<text class="meta-line">{{ gameTypeLabel }} · {{ matchModeLabel }}</text>
			<text class="meta-line">{{ scheduleLabel }}</text>
			<text class="meta-hint" v-if="expiredNow">已失效</text>
			<text class="meta-hint" v-else-if="challenge.status === 1">返回或锁屏后仍会等待；暂时不打，可以退出等待</text>
			<text class="meta-hint" v-else-if="challenge.status === 0 && expiryHint">{{ expiryHint }}</text>

			<view class="action-column">
				<template v-if="challenge.status === 0 && !expiredNow">
					<template v-if="challenge.to_user_id === myUserId">
						<button class="primary-btn" :disabled="acting" @tap="acceptInvite()"><text>接受约球</text></button>
						<button class="ghost-btn" :disabled="acting" @tap="rejectInvite"><text>这次不了</text></button>
					</template>
					<button v-else class="ghost-btn" :disabled="acting" @tap="cancelChallengeConfirm"><text>取消邀请</text></button>
				</template>
				<template v-else-if="challenge.status === 1 && !expiredNow">
					<template v-if="challenge.waiting_user_id === myUserId">
						<button class="primary-btn" @tap="leaveWaiting"><text>退出等待，保留约球</text></button>
						<button class="ghost-btn" @tap="cancelChallengeConfirm"><text>取消本次约球</text></button>
					</template>
					<template v-else-if="challenge.waiting_user_id === opponentUserId && challenge.waiting_user_id > 0">
						<button class="primary-btn" :disabled="acting" @tap="enterChallengeNow(true)"><text>进入对局</text></button>
						<button class="ghost-btn" :disabled="acting" @tap="abandonConfirm"><text>放弃本次约球</text></button>
					</template>
					<template v-else>
						<button class="primary-btn" :disabled="acting" @tap="enterChallengeNow(false)"><text>进入对局</text></button>
						<button class="ghost-btn" @tap="cancelChallengeConfirm"><text>取消本次约球</text></button>
					</template>
				</template>
				<template v-else-if="challenge.status === 6 && challenge.match_id > 0">
					<button class="primary-btn" @tap="goMatch()"><text>继续比赛</text></button>
				</template>
				<template v-else>
					<text class="state-text">{{ expiredNow ? '已失效' : statusLabel }}</text>
				</template>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onHide, onLoad, onShow, onUnload } from '@dcloudio/uni-app'
import { abandonChallenge, acceptChallenge, cancelChallenge, enterChallenge, getChallengeDetail, getChallengeSummary, leaveChallenge, rejectChallenge } from '@/api/challenge.js'
import { GAME_TYPE_LABEL_MAP } from '@/utils/game-types.js'
import { useUserStore } from '@/store/user.js'
import { useActivityStore } from '@/store/activity.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import { formatChallengeExpiry, formatChallengeSchedule } from '@/utils/challenge-view.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const activityStore = useActivityStore()
const challengeId = ref(0)
const challenge = ref(null)
const loading = ref(true)
const acting = ref(false)
const serverTime = ref('')
let pollTimer = null
let detailRequestId = 0
let pageUserId = 0
let pageAuthGeneration = -1
// 页面生命周期：隐藏/卸载后在途读写回调不得再改页面或导航。
let pageActive = false

const myUserId = computed(() => Number(userStore.userId) || 0)
// 写请求与弹窗回调按发起时的身份隔离：切号后旧响应/旧确认不得继续操作。
const readIdentity = () => ({ userId: userStore.userId, authGeneration: userStore.authGeneration })
const isSameIdentity = (identity) => userStore.userId === identity.userId && userStore.authGeneration === identity.authGeneration
const who = computed(() => ({ from: challenge.value?.from_user_id || 0, to: challenge.value?.to_user_id || 0 }))
const opponentUserId = computed(() => (who.value.from === myUserId.value ? who.value.to : who.value.from))
const gameTypeLabel = computed(() => GAME_TYPE_LABEL_MAP[challenge.value?.game_type] || '台球')
const matchModeLabel = computed(() => (challenge.value?.match_mode === 'practice' ? '练习' : '排位'))
const scheduleLabel = computed(() => formatChallengeSchedule(challenge.value, serverTime.value))
const waitingHint = computed(() => {
	if (!challenge.value) return ''
	if (challenge.value.waiting_user_id === myUserId.value) return '等待对方进入'
	if (challenge.value.waiting_user_id > 0) return '对方已进入，等你一起开始'
	return '双方进入后正式开始'
})
const statusLabel = computed(() => {
	const map = { 2: '对方这次不了', 3: '已失效', 4: '约球已取消', 5: '对方已放弃，本次约球已结束', 7: '已完成', 8: '对局已取消' }
	return map[challenge.value?.status] || ''
})
// 服务端时间判断过期：worker 未运行时也能把到期约球显示为「已失效」（后端仍会拒绝操作）。
const expiredNow = computed(() => {
	if (!challenge.value) return false
	if (Number(challenge.value.status) === 3) return true
	if (Number(challenge.value.status) !== 0 && Number(challenge.value.status) !== 1) return false
	const expiry = String(challenge.value.expires_at || '')
	return Boolean(expiry && serverTime.value && expiry <= serverTime.value)
})
const expiryHint = computed(() => (challenge.value ? formatChallengeExpiry(challenge.value, serverTime.value) : ''))

const isWaiting = (userId) => challenge.value?.waiting_user_id === userId

const applyDetail = (detail) => {
	if (!detail) return false
	const previousWaiting = challenge.value?.waiting_user_id
	if (challenge.value && (challenge.value.status !== detail.status || previousWaiting !== detail.waiting_user_id || challenge.value.match_id !== detail.match_id)) {
		activityStore.markDirty()
	}
	challenge.value = detail
	if (previousWaiting !== undefined && detail.status === 6 && detail.match_id > 0) {
		goMatch(true)
		return true
	}
	return false
}

const loadDetail = async () => {
	if (!challengeId.value) return
	const requestId = ++detailRequestId
	const userId = userStore.userId
	const generation = userStore.authGeneration
	const isCurrent = () => detailRequestId === requestId && userStore.userId === userId && userStore.authGeneration === generation && pageActive
	loading.value = !challenge.value
	try {
		const [res, summary] = await Promise.all([
			getChallengeDetail({ id: challengeId.value }),
			getChallengeSummary().catch(() => null)
		])
		if (!isCurrent()) return
		if (summary?.success && summary.server_time) serverTime.value = summary.server_time
		if (res?.success && res.challenge) {
			applyDetail(res.challenge)
		} else if (!challenge.value) {
			challenge.value = null
		}
	} catch (error) {
		if (isCurrent() && !challenge.value) challenge.value = null
	} finally {
		if (isCurrent()) loading.value = false
	}
}

const startPoll = () => {
	stopPoll()
	pollTimer = setInterval(() => {
		if (!acting.value) loadDetail()
	}, 5000)
}
const stopPoll = () => {
	if (pollTimer) {
		clearInterval(pollTimer)
		pollTimer = null
	}
}

onLoad((query = {}) => {
	challengeId.value = Number(query.challenge_id || query.id) || 0
})

onShow(() => {
	if (pageUserId !== userStore.userId || pageAuthGeneration !== userStore.authGeneration) {
		challenge.value = null
		serverTime.value = ''
		pageUserId = userStore.userId
		pageAuthGeneration = userStore.authGeneration
	}
	pageActive = true
	loadDetail()
	startPoll()
})
const pausePage = () => {
	pageActive = false
	stopPoll()
}
onHide(pausePage)
onUnload(pausePage)

const acceptInvite = async (confirmed = false) => {
	if (acting.value) return
	acting.value = true
	const identity = readIdentity()
	try {
		const res = await acceptChallenge({ challenge_id: challengeId.value, confirm_close_own_pending: confirmed === true })
		if (!isSameIdentity(identity)) return
		if (res?.need_confirm_close_own) {
			if (!pageActive) return
			uni.showModal({
				title: '接受这个约球？',
				content: '接受后会自动关闭你已发出的邀请。',
				confirmText: '确认接受',
				success: ({ confirm }) => { if (confirm && pageActive && isSameIdentity(identity)) acceptInvite(true) }
			})
		} else if (res?.type_conflict) {
			if (!pageActive) return
			uni.showModal({
				title: '约球类型不同',
				content: res.message || '请先处理自己发出的邀请',
				confirmText: '查看已发出的邀请',
				cancelText: '知道了',
				success: ({ confirm }) => {
					if (confirm && pageActive && isSameIdentity(identity)) uni.navigateTo({ url: '/subPages/social/challenges?tab=sent' })
				}
			})
		} else if (res?.success) {
			activityStore.markDirty()
			if (!pageActive) return
			await loadDetail()
		} else {
			if (!pageActive) return
			uni.showToast({ title: res?.message || '接受失败', icon: 'none' })
			loadDetail()
		}
	} catch (error) {
		if (!pageActive) return
		uni.showToast({ title: '结果不确定，正在确认', icon: 'none' })
		loadDetail()
	} finally {
		acting.value = false
	}
}

const rejectInvite = async () => {
	if (acting.value) return
	acting.value = true
	const identity = readIdentity()
	try {
		const res = await rejectChallenge({ challenge_id: challengeId.value })
		if (!isSameIdentity(identity)) return
		if (res?.success) {
			activityStore.markDirty()
		} else if (pageActive) {
			uni.showToast({ title: res?.message || '拒绝失败', icon: 'none' })
		}
		if (pageActive) await loadDetail()
	} catch (error) {
		if (!pageActive) return
		uni.showToast({ title: '结果不确定，正在确认', icon: 'none' })
		loadDetail()
	} finally {
		acting.value = false
	}
}

const offerOngoingMatch = (res) => {
	if (res?.action !== 'self_ongoing' || !res.match_id) return false
	if (!pageActive) return true
	const identity = readIdentity()
	uni.showModal({
		title: '你有未结束的对局',
		content: res.message || '请先处理当前比赛',
		confirmText: '继续比赛',
		cancelText: '稍后处理',
		success: ({ confirm }) => {
			if (confirm && pageActive && isSameIdentity(identity)) uni.navigateTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
		}
	})
	return true
}

const enterChallengeNow = async (opponentWaiting) => {
	if (acting.value) return
	acting.value = true
	const identity = readIdentity()
	try {
		const res = await enterChallenge({ challenge_id: challengeId.value })
		if (!isSameIdentity(identity)) return
		if (!res?.success) {
			if (res?.action === 'confirm_early') {
				if (!pageActive) return
				uni.showModal({
					title: '现在进入吗？',
					content: `预计${scheduleLabel.value.split(' ')[0] || ''}开始。双方进入后才会正式开局。`,
					confirmText: '现在进入',
					cancelText: '稍后再打',
					success: ({ confirm }) => {
						if (confirm && pageActive && isSameIdentity(identity)) enterChallengeConfirmed()
					}
				})
				return
			}
			if (!pageActive) return
			uni.showToast({ title: res?.message || '进入失败', icon: 'none' })
			loadDetail()
			return
		}
		if (offerOngoingMatch(res)) return
		if (res.action === 'match_created' && res.match_id > 0) {
			activityStore.markDirty()
			if (!pageActive) return
			uni.redirectTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
			return
		}
		if (res.action === 'waiting') {
			activityStore.markDirty()
			if (!pageActive) return
			if (opponentWaiting) {
				uni.showToast({ title: '对方刚退出等待，你已成为等待方', icon: 'none' })
			} else {
				uni.showToast({ title: '已进入等待', icon: 'success' })
			}
			loadDetail()
		}
	} catch (error) {
		if (!pageActive) return
		uni.showToast({ title: '结果不确定，正在重新确认', icon: 'none' })
		loadDetail()
	} finally {
		acting.value = false
	}
}

const enterChallengeConfirmed = async () => {
	if (acting.value) return
	acting.value = true
	const identity = readIdentity()
	try {
		const res = await enterChallenge({ challenge_id: challengeId.value, confirm_early: true })
		if (!isSameIdentity(identity)) return
		if (res?.success && offerOngoingMatch(res)) return
		if (res?.success && res.action === 'match_created' && res.match_id > 0) {
			activityStore.markDirty()
			if (!pageActive) return
			uni.redirectTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
			return
		}
		if (res?.success && res.action === 'waiting') {
			activityStore.markDirty()
			if (!pageActive) return
			uni.showToast({ title: '已进入等待', icon: 'success' })
			loadDetail()
		}
		if (!res?.success) {
			// 提前确认后的失败（如约球已失效）必须给出原因并立即回源。
			if (!pageActive) return
			uni.showToast({ title: res?.message || '进入失败', icon: 'none' })
			loadDetail()
		}
	} catch (error) {
		if (!pageActive) return
		uni.showToast({ title: '结果不确定，正在重新确认', icon: 'none' })
		loadDetail()
	} finally {
		acting.value = false
	}
}

const leaveWaiting = async () => {
	if (acting.value) return
	acting.value = true
	const identity = readIdentity()
	try {
		const res = await leaveChallenge({ challenge_id: challengeId.value })
		if (!isSameIdentity(identity)) return
		if (res?.success) {
			activityStore.markDirty()
			if (!pageActive) return
			uni.showToast({ title: '已退出等待，约球保留', icon: 'success' })
		} else if (res?.match_id > 0) {
			activityStore.markDirty()
			if (!pageActive) return
			uni.redirectTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
			return
		} else {
			if (!pageActive) return
			uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
		}
		loadDetail()
	} finally {
		acting.value = false
	}
}

const cancelChallengeConfirm = () => {
	const identity = readIdentity()
	uni.showModal({
		title: '取消本次约球？',
		content: '取消后需要重新邀约才能再打。',
		success: async ({ confirm }) => {
			if (!confirm || !pageActive || acting.value || !isSameIdentity(identity)) return
			acting.value = true
			try {
				const res = await cancelChallenge({ challenge_id: challengeId.value })
				if (!isSameIdentity(identity)) return
				if (res?.success) {
					activityStore.markDirty()
					if (!pageActive) return
					uni.showToast({ title: '约球已取消', icon: 'success' })
					loadDetail()
				} else if (res?.match_id > 0) {
					activityStore.markDirty()
					if (!pageActive) return
					uni.redirectTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
				} else {
					if (!pageActive) return
					uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
					loadDetail()
				}
			} finally {
				acting.value = false
			}
		}
	})
}

const abandonConfirm = () => {
	const identity = readIdentity()
	uni.showModal({
		title: '放弃本次约球？',
		content: '对方将退出等待，约球结束。不会产生比赛和战绩。',
		confirmText: '确认放弃',
		cancelText: '继续约球',
		success: async ({ confirm }) => {
			if (!confirm || !pageActive || acting.value || !isSameIdentity(identity)) return
			acting.value = true
			try {
				const res = await abandonChallenge({ challenge_id: challengeId.value })
				if (!isSameIdentity(identity)) return
				if (res?.success) {
					activityStore.markDirty()
					if (!pageActive) return
					uni.showToast({ title: '本次约球已结束', icon: 'none' })
					loadDetail()
				} else if (res?.match_id > 0) {
					activityStore.markDirty()
					if (!pageActive) return
					uni.redirectTo({ url: `/subPages/match/playing?match_id=${res.match_id}` })
				} else {
					if (!pageActive) return
					uni.showToast({ title: res?.message || '操作失败', icon: 'none' })
					loadDetail()
				}
			} finally {
				acting.value = false
			}
		}
	})
}

const goMatch = (replace = false) => {
	if (!challenge.value?.match_id) return
	const url = `/subPages/match/playing?match_id=${challenge.value.match_id}`
	if (replace) {
		uni.redirectTo({ url })
	} else {
		uni.navigateTo({ url })
	}
}
</script>

<style lang="scss" scoped>
@import './challengeWaiting.scss';
</style>
