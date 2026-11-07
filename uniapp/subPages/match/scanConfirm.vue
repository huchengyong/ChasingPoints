<template>
	<view class="confirm-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="stage === 'loading'" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-text">正在核验裁判码...</text>
		</view>

		<view v-else-if="stage === 'expired' || stage === 'error'" class="state-block">
			<text class="state-title">无法继续</text>
			<text class="state-text state-text--error">{{ errorMessage }}</text>
			<button class="primary-btn" @tap="handleRescan">重新扫码</button>
			<button class="ghost-btn" @tap="handleBack">返回</button>
		</view>

		<template v-else-if="mode === 'start'">
			<view class="opponent-card">
				<image class="opponent-avatar" :src="resolveAvatarUrl(opponent.avatar, opponent.id)" mode="aspectFill"></image>
				<view class="opponent-info">
					<text class="opponent-name">{{ opponent.nickname || '对手' }}</text>
					<text class="opponent-sub">已识别对手，选好球种和类型后开始对局</text>
				</view>
			</view>

			<view class="form-card">
				<text class="form-label">球种</text>
				<view class="chip-row">
					<view
						v-for="item in gameTypes"
						:key="item.value"
						class="chip"
						:class="{ active: gameType === item.value }"
						@tap="selectGameType(item.value)"
					>
						<text>{{ item.label }}</text>
						<text v-if="defaultGameType === item.value" class="chip-badge">默认</text>
					</view>
				</view>
				<view
					v-if="canSetDefault"
					class="default-choice"
					:class="{ active: setAsDefault }"
					@tap="setAsDefault = !setAsDefault"
				>
					<view class="default-choice-box">
						<uni-icons v-if="setAsDefault" type="checkmarkempty" size="14" color="#ffffff"></uni-icons>
					</view>
					<text>将当前选择设为默认对局类型</text>
				</view>

				<text class="form-label">对局类型</text>
				<view class="mode-list">
					<view
						v-for="(item, index) in modeOptions"
						:key="item.label"
						class="mode-option"
						:class="{ active: modeIndex === index }"
						@tap="modeIndex = index"
					>
						<view class="mode-radio"></view>
						<text class="mode-label">{{ item.label }}</text>
					</view>
				</view>
			</view>

			<text v-if="errorMessage" class="inline-error">{{ errorMessage }}</text>
			<button class="primary-btn" :disabled="!gameType || submitting" @tap="handleStart">{{ submitting ? '创建中...' : '开始对局' }}</button>
			<button class="ghost-btn" :disabled="submitting" @tap="handleRescan">重新扫码</button>
		</template>

		<template v-else>
			<view class="referee-card">
				<text class="referee-title">确认担任本场裁判</text>
				<text class="referee-desc">确认后，你将负责本场记分和结算，双方选手会切换为只读。</text>
				<view class="referee-body">
					<view class="referee-players">
						<view class="referee-player">
							<image :src="resolveAvatarUrl(refereePreview.player1_avatar, refereePreview.player1_id)" mode="aspectFill"></image>
							<text>{{ refereePreview.player1_name || '选手1' }}</text>
						</view>
						<text class="referee-vs">VS</text>
						<view class="referee-player">
							<image :src="resolveAvatarUrl(refereePreview.player2_avatar, refereePreview.player2_id)" mode="aspectFill"></image>
							<text>{{ refereePreview.player2_name || '选手2' }}</text>
						</view>
					</view>
					<text class="referee-score">{{ refereePreview.player1_score }} : {{ refereePreview.player2_score }}</text>
					<text class="referee-meta">{{ refereePreview.game_type_name || '对局' }} · {{ refereeModeText }}</text>
				</view>
			</view>

			<button class="primary-btn" :disabled="joining" @tap="handleJoin">{{ joining ? '确认中...' : '确认担任裁判' }}</button>
			<button class="ghost-btn" :disabled="joining" @tap="handleBack">暂不担任</button>
		</template>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { joinMatchReferee, previewMatchReferee, startMatch } from '@/api/match.js'
import { useUserStore } from '@/store/user.js'
import { GAME_TYPE_OPTIONS } from '@/utils/game-types.js'
import { readDefaultGameType, saveDefaultGameType } from '@/utils/game-type-preference.js'
import { buildPlayingRoute, resolveStartMatchGuardAction } from '@/utils/ongoing-match-guard.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { startQuickScan, takeScanHandoff } from '@/utils/quick-scan-entry.js'
import { START_MATCH_MODE_OPTIONS, buildStartMatchPayload } from '@/utils/start-match.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

const gameTypes = GAME_TYPE_OPTIONS
const modeOptions = START_MATCH_MODE_OPTIONS

// loading：核验裁判码；ready：可确认；expired：没有可用的扫码信息；error：核验或确认失败。
const stage = ref('loading')
const mode = ref('')
const errorMessage = ref('')
const opponent = ref({})
const gameType = ref(0)
const defaultGameType = ref(0)
const setAsDefault = ref(false)
const modeIndex = ref(0)
const submitting = ref(false)
const refereePreview = ref({})
const joining = ref(false)

// 凭证只放在这里，不进响应式状态和路由；页面只在发起扫码的身份下可用。
let inviteToken = ''
let refereeJoin = null
let rematch = null
let pageIdentity = null

const canSetDefault = computed(() => gameType.value > 0 && gameType.value !== defaultGameType.value)
const refereeModeText = computed(() => (refereePreview.value.match_mode === 'ranked' ? '排位赛' : '练习赛'))

const isCurrentIdentity = () => (
	Boolean(pageIdentity)
	&& userStore.isLoggedIn
	&& userStore.userId === pageIdentity.userId
	&& userStore.authGeneration === pageIdentity.authGeneration
)

const showExpired = (message) => {
	stage.value = 'expired'
	errorMessage.value = message
	inviteToken = ''
	refereeJoin = null
}

const resolveModeIndex = (preset) => {
	if (!preset) return 0
	const index = modeOptions.findIndex((item) => item.matchMode === preset.matchMode && item.visibility === preset.visibility)
	return index >= 0 ? index : 0
}

const loadRefereePreview = async () => {
	const identity = pageIdentity
	stage.value = 'loading'
	try {
		const res = await previewMatchReferee(refereeJoin)
		if (pageIdentity !== identity || !isCurrentIdentity()) return
		if (!res?.success || !res.preview) {
			errorMessage.value = res?.message || '裁判码已失效，请重新扫码'
			stage.value = 'error'
			return
		}
		refereePreview.value = res.preview
		stage.value = 'ready'
	} catch (error) {
		console.error('预览裁判对局失败:', error)
		if (pageIdentity !== identity || !isCurrentIdentity()) return
		errorMessage.value = '裁判码已失效，请重新扫码'
		stage.value = 'error'
	}
}

onLoad(() => {
	const identity = { userId: userStore.userId, authGeneration: userStore.authGeneration }
	const handoff = userStore.isLoggedIn ? takeScanHandoff(identity) : null
	if (!handoff) {
		showExpired('扫码信息已失效，请重新扫码')
		return
	}
	pageIdentity = identity
	rematch = handoff.rematch
	mode.value = handoff.mode

	if (handoff.mode === 'referee') {
		uni.setNavigationBarTitle({ title: '确认执裁' })
		refereeJoin = handoff.refereeJoin
		loadRefereePreview()
		return
	}

	uni.setNavigationBarTitle({ title: '确认对局' })
	inviteToken = handoff.inviteToken
	opponent.value = handoff.opponent
	defaultGameType.value = readDefaultGameType(uni, identity.userId)
	gameType.value = handoff.preset?.gameType || defaultGameType.value
	modeIndex.value = resolveModeIndex(handoff.preset)
	stage.value = 'ready'
})

// 在确认页期间账号发生变化（会话失效后重新登录等）：旧凭证不再可用。
onShow(() => {
	if (pageIdentity && !isCurrentIdentity() && stage.value !== 'expired') {
		showExpired('登录状态已变更，请重新扫码')
	}
})

const selectGameType = (value) => {
	gameType.value = value
	setAsDefault.value = false
}

const redirectToPlaying = (match) => {
	uni.redirectTo({ url: buildPlayingRoute(match) })
}

const handleStartOutcome = (outcome) => {
	if (outcome.type === 'navigate' || outcome.type === 'resume') {
		redirectToPlaying(outcome.match)
		return
	}
	if (outcome.type === 'prompt_self_ongoing') {
		uni.showModal({
			title: '你有未结束的对局',
			content: outcome.message,
			confirmText: '进入该对局',
			cancelText: '稍后处理',
			success: ({ confirm }) => {
				if (confirm) redirectToPlaying(outcome.match)
			}
		})
		return
	}
	if (outcome.type === 'toast_opponent_ongoing') {
		uni.showToast({ title: outcome.message, icon: 'none' })
		return
	}
	errorMessage.value = outcome.message || '创建对局失败'
}

const handleStart = async () => {
	if (!gameType.value || submitting.value) return
	if (!isCurrentIdentity()) {
		showExpired('登录状态已变更，请重新扫码')
		return
	}
	const identity = pageIdentity
	const selectedGameType = gameType.value
	const option = modeOptions[modeIndex.value] || modeOptions[0]
	if (setAsDefault.value) {
		defaultGameType.value = saveDefaultGameType(uni, identity.userId, selectedGameType)
		setAsDefault.value = false
	}
	submitting.value = true
	errorMessage.value = ''
	try {
		const res = await startMatch(buildStartMatchPayload({
			gameType: selectedGameType,
			opponent: opponent.value,
			matchMode: option.matchMode,
			visibility: option.visibility,
			inviteToken
		}))
		// 等待期间切号或会话失效，旧账号的开局结果不再跳转
		if (pageIdentity !== identity || !isCurrentIdentity()) return
		handleStartOutcome(resolveStartMatchGuardAction({
			response: res,
			selectedGameType,
			scannedOpponent: opponent.value
		}))
	} catch (error) {
		console.error('扫码开局失败:', error)
		if (pageIdentity === identity && isCurrentIdentity()) errorMessage.value = '匹配失败，请重试'
	} finally {
		submitting.value = false
	}
}

const handleJoin = async () => {
	if (joining.value) return
	if (!isCurrentIdentity()) {
		showExpired('登录状态已变更，请重新扫码')
		return
	}
	const identity = pageIdentity
	const join = refereeJoin
	joining.value = true
	try {
		const res = await joinMatchReferee(join)
		if (pageIdentity !== identity || !isCurrentIdentity()) return
		if (!res?.success) {
			errorMessage.value = res?.message || '裁判码已失效，请重新扫码'
			stage.value = 'error'
			return
		}
		redirectToPlaying({
			match_id: res.match_id || join.match_id,
			game_type: res.match?.game_type || 3,
			opponent_name: res.match?.opponent_name || '选手2',
			opponent_avatar: res.match?.opponent_avatar || ''
		})
	} catch (error) {
		console.error('确认担任裁判失败:', error)
		if (pageIdentity === identity && isCurrentIdentity()) {
			errorMessage.value = '加入裁判失败，请重新扫码'
			stage.value = 'error'
		}
	} finally {
		joining.value = false
	}
}

// 替换当前页而不是再压一页；取消扫码时停留在原状态。
const handleRescan = () => {
	startQuickScan({ rematch, replace: true })
}

const handleBack = () => {
	uni.navigateBack({
		delta: 1,
		fail: () => uni.switchTab({ url: '/pages/index/index' })
	})
}
</script>

<style lang="scss" scoped>
@import './scanConfirm.scss';
</style>
