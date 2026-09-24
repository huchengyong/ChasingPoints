<template>
	<view class="compose-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="opponent-card" v-if="opponent.id">
			<image class="opponent-avatar" :src="resolveAvatarUrl(opponent.avatar, opponent.id)" mode="aspectFill"></image>
			<view class="opponent-info">
				<text class="opponent-name">{{ opponent.nickname || '球友' }}</text>
				<text class="opponent-sub">约好后双方各点一次进入，直接开局</text>
			</view>
			<view class="opponent-change" @tap="chooseOpponent">
				<text>更换</text>
			</view>
		</view>
		<view class="opponent-card empty" v-else @tap="chooseOpponent">
			<text class="opponent-name">选择球友</text>
			<text class="opponent-sub">好友和最近对手都可以约</text>
		</view>

		<view class="form-card">
			<text class="form-label">打什么</text>
			<view class="chip-row">
				<view
					v-for="item in gameTypes"
					:key="item.value"
					class="chip"
					:class="{ active: gameType === item.value }"
					@tap="gameType = item.value"
				>
					<text>{{ item.label }}</text>
				</view>
			</view>

			<text class="form-label">比赛类型</text>
			<view class="chip-row">
				<view class="chip" :class="{ active: matchMode === 'ranked' }" @tap="matchMode = 'ranked'">
					<text>排位</text>
				</view>
				<view class="chip" :class="{ active: matchMode === 'practice' }" @tap="matchMode = 'practice'">
					<text>练习</text>
				</view>
			</view>
			<text class="form-hint">{{ matchModeHint }}</text>

			<text class="form-label">预计什么时候</text>
			<view class="chip-row">
				<view
					v-for="item in dayOptions"
					:key="item.value"
					class="chip"
					:class="{ active: activeDayOffset === item.value }"
					@tap="chooseDay(item.value)"
				>
					<text>{{ item.label }}</text>
				</view>
			</view>
			<view class="hour-row">
				<picker class="hour-picker" mode="selector" :range="startHourLabels" @change="onStartHourChange">
					<view class="hour-value"><text>{{ hourLabel(startHour) }}</text></view>
				</picker>
				<text class="hour-sep">到</text>
				<picker class="hour-picker" mode="selector" :range="endHourLabels" @change="onEndHourChange">
					<view class="hour-value"><text>{{ hourLabel(endHour) }}</text></view>
				</picker>
			</view>
			<text class="form-hint">预计时间仅作提醒，明天早晨7点失效</text>

			<text class="form-label">附言（选填）</text>
			<input class="message-input" v-model="message" placeholder="给对方捎句话" maxlength="50" />
		</view>

		<button class="submit-btn" :disabled="submitting" @tap="submit">
			<text>{{ submitting ? '发送中…' : '发送约球' }}</text>
		</button>
	</view>
</template>

<script setup>
import { onLoad, onShow, onHide, onUnload } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { sendChallenge, getChallengeSummary } from '@/api/challenge.js'
import { useActivityStore } from '@/store/activity.js'
import { useUserStore } from '@/store/user.js'
import { GAME_TYPE_OPTIONS } from '@/utils/game-types.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import {
	CHALLENGE_DAY_OFFSETS,
	availableStartHours,
	computeScheduledDate,
	dayOffsetOfScheduledDate,
	defaultChallengeSlot,
	endHourOptions,
	hourLabel
} from '@/utils/challenge-time.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const gameTypes = GAME_TYPE_OPTIONS

const opponent = ref({ id: 0, nickname: '', avatar: '' })
const gameType = ref(3)
const matchMode = ref('ranked')
const startHour = ref(16)
const endHour = ref(17)
const message = ref('')
const submitting = ref(false)
const serverTime = ref('')
// 发送结果与时间基准按发起身份/页面生命周期隔离：切号或离开后旧回调不得弹窗、标脏或导航。
let pageActive = false
// 页面世代：onHide/onUnload 递增，防止隐藏后再次 onShow 让旧提交链路重新获得执行资格。
let pageGeneration = 0
const readIdentity = () => ({ userId: userStore.userId, authGeneration: userStore.authGeneration })
const isSameIdentity = (identity) => userStore.userId === identity.userId && userStore.authGeneration === identity.authGeneration
// 实际预约日是唯一日期状态：相对日高亮与提交都由它派生；更换对手后原样带回。
const scheduledDate = ref('')
// 更换对手回来时保留已选时段；空则按服务端时间取默认下一可用整点。
const preservedSlot = ref(null)

const dayOptions = computed(() => {
	// 服务端时间不可用时退化为固定三天标签。
	if (!serverTime.value) return CHALLENGE_DAY_OFFSETS
	return CHALLENGE_DAY_OFFSETS.filter((item) => availableStartHours(serverTime.value, item.value).length > 0 || item.value > 0)
})

const activeDayOffset = computed(() => dayOffsetOfScheduledDate(serverTime.value, scheduledDate.value))
const startHourChoices = computed(() => availableStartHours(serverTime.value, Math.max(activeDayOffset.value, 0)))
const startHourLabels = computed(() => startHourChoices.value.map(hourLabel))
const endHourLabels = computed(() => endHourOptions(startHour.value).map(hourLabel))
const matchModeHint = computed(() => (matchMode.value === 'ranked' ? '排位赛公开展示并计入段位，双方进入后单方滑动即可结束' : '练习赛不计段位，仅记录战绩'))

const applyDefaultSlot = () => {
	const slot = preservedSlot.value || defaultChallengeSlot(serverTime.value)
	startHour.value = slot.startHour
	endHour.value = slot.endHour
	// 实际预约日只保留一份：未回填时按服务端今天 + 默认相对日计算。
	if (!scheduledDate.value) {
		scheduledDate.value = computeScheduledDate(serverTime.value, slot.dayOffset || 0)
	}
}

let serverTimeRequestSeq = 0
const fetchServerTime = async () => {
	// 只允许最新一次读取更新时间基准：乱序返回的旧响应不得回退日期。
	const requestId = ++serverTimeRequestSeq
	let updated = false
	try {
		const res = await getChallengeSummary()
		if (requestId === serverTimeRequestSeq && res?.success && res.server_time) {
			serverTime.value = res.server_time
			updated = true
		}
	} catch (error) {
		// 保留旧基准；提交时由服务端校验兑底。
	}
	// 被替换的旧响应不得做任何事（包括默认时段初始化）。
	if (requestId !== serverTimeRequestSeq) return false
	if (!updated && !serverTime.value) {
		const now = new Date(Date.now() + 8 * 60 * 60 * 1000)
		serverTime.value = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}-${String(now.getUTCDate()).padStart(2, '0')} ${String(now.getUTCHours()).padStart(2, '0')}:${String(now.getUTCMinutes()).padStart(2, '0')}:00`
	}
	return true
}

let slotInitialized = false
const loadServerTime = async () => {
	const current = await fetchServerTime()
	// 默认时段只在首次进入落一次，且必须基于最新有效的时间基准。
	if (current && !slotInitialized) {
		slotInitialized = true
		applyDefaultSlot()
	}
}

onLoad((query = {}) => {
	if (query.opponent_id) {
		opponent.value = {
			id: Number(query.opponent_id) || 0,
			nickname: decodeURIComponent(query.opponent_name || ''),
			avatar: decodeURIComponent(query.opponent_avatar || '')
		}
	}
	if (query.game_type && Number(query.game_type) > 0) {
		gameType.value = Number(query.game_type)
	}
	if (query.match_mode === 'practice' || query.match_mode === 'ranked') {
		matchMode.value = query.match_mode
	}
	if (query.message) {
		message.value = String(query.message)
	}
	const startHourValue = Number(query.start_hour)
	const endHourValue = Number(query.end_hour)
	if (Number.isInteger(startHourValue) && startHourValue >= 0 && startHourValue <= 23 &&
		Number.isInteger(endHourValue) && endHourValue > startHourValue && endHourValue <= 24) {
		preservedSlot.value = {
			startHour: startHourValue,
			endHour: endHourValue
		}
	}
	if (query.scheduled_date) {
		scheduledDate.value = String(query.scheduled_date)
	}
})

onShow(() => {
	pageActive = true
	// 回到前台刷新服务端时间：保留实际预约日与用户输入，只重算相对展示与有效性。
	loadServerTime()
})
onHide(() => {
	pageActive = false
	pageGeneration += 1
})
onUnload(() => {
	pageActive = false
	pageGeneration += 1
})

// 提交前刷新时间基准：页面保持打开跨过午夜时服务端日期会前进，旧基准派生的相对日会被拒绝。
// 仅当本次读取是最新且成功才返回 true；失败或被替换时终止提交，不得把旧基准当作刷新成功。
const refreshServerTimeForSubmit = async () => {
	const requestId = ++serverTimeRequestSeq
	let updated = false
	try {
		const res = await getChallengeSummary()
		if (requestId === serverTimeRequestSeq && res?.success && res.server_time) {
			serverTime.value = res.server_time
			updated = true
		}
	} catch (error) {
		// 读取失败：本次提交终止，不把旧基准当作刷新成功。
	}
	return updated && requestId === serverTimeRequestSeq
}

const chooseOpponent = () => {
	// 选人后回填时保留已选比赛类型、实际预约日、时段与附言，只替换对手。
	const query = [
		'from=challenge',
		`game_type=${gameType.value}`,
		`match_mode=${matchMode.value}`,
		`start_hour=${startHour.value}`,
		`end_hour=${endHour.value}`,
		`scheduled_date=${scheduledDate.value}`
	]
	if (message.value) query.push(`message=${encodeURIComponent(message.value)}`)
	uni.navigateTo({ url: `/subPages/match/opponentSelector?${query.join('&')}` })
}

const chooseDay = (value) => {
	// 直接更新实际预约日：高亮与提交使用同一份，回填值不会覆盖用户新选择。
	const date = computeScheduledDate(serverTime.value, value)
	if (date) scheduledDate.value = date
	const hours = availableStartHours(serverTime.value, value)
	startHour.value = hours[0]
	endHour.value = Math.min(startHour.value + 1, 24)
}

const onStartHourChange = ({ detail }) => {
	const hour = startHourChoices.value[Number(detail.value)] ?? startHour.value
	startHour.value = hour
	if (endHour.value <= hour) {
		endHour.value = Math.min(hour + 1, 24)
	}
}

const onEndHourChange = ({ detail }) => {
	const choices = endHourOptions(startHour.value)
	endHour.value = choices[Number(detail.value)] ?? endHour.value
}

const submit = async () => {
	if (submitting.value) return
	if (!opponent.value.id) {
		uni.showToast({ title: '请先选择球友', icon: 'none' })
		return
	}
	// 相对日由实际预约日按服务端今天派生，跨午夜不静默滚日。
	if (!scheduledDate.value) {
		uni.showToast({ title: '时间选择无效，请重试', icon: 'none' })
		return
	}
	// 提交锁覆盖包括时间预读在内的整个异步流程；身份与页面世代必须在首个 await 前捕获。
	submitting.value = true
	const identity = readIdentity()
	const generation = pageGeneration
	try {
		// 页面保持打开跨过午夜时，旧基准派生的相对日会被服务端拒绝：提交前先刷新基准。
		const refreshed = await refreshServerTimeForSubmit()
		// 预读期间切号、离开页面或读取失败/被替换：终止本次提交，不得改用新账号或旧基准发送。
		if (!refreshed || !isSameIdentity(identity) || generation !== pageGeneration) {
			if (pageActive && isSameIdentity(identity) && generation === pageGeneration) {
				uni.showToast({ title: '时间校验失败，请重试', icon: 'none' })
			}
			return
		}
		const dayOffsetValue = dayOffsetOfScheduledDate(serverTime.value, scheduledDate.value)
		if (dayOffsetValue === null) {
			uni.showToast({ title: '时间选择无效，请重试', icon: 'none' })
			return
		}
		const payload = {
			to_user_id: opponent.value.id,
			game_type: gameType.value,
			match_mode: matchMode.value,
			visibility: matchMode.value === 'practice' ? 'private' : 'public',
			match_format: 'free',
			target_wins: 0,
			message: message.value,
			day_offset: dayOffsetValue,
			scheduled_date: scheduledDate.value,
			start_hour: startHour.value,
			end_hour: endHour.value
		}
		if (gameType.value === 1) {
			payload.snooker_rules_version = 2
			payload.snooker_format = 'free'
			payload.snooker_target_wins = 0
		} else if (gameType.value === 2) {
			delete payload.match_format
			delete payload.target_wins
		}
		const res = await sendChallenge(payload)
		// 发送结果按发起时的身份与页面世代隔离：切号或离开后旧响应不弹窗、不标脏、不导航。
		if (!isSameIdentity(identity) || generation !== pageGeneration) return
		if (!res?.success) {
			if (res?.pending_challenge_id > 0) {
				if (!pageActive) return
				uni.showModal({
					title: '约球未发送',
					content: res.message || '你发出的约球还在等待回应，请先处理',
					confirmText: '查看并取消',
					cancelText: '稍后再说',
					success: ({ confirm }) => {
						if (confirm && pageActive && isSameIdentity(identity)) {
							// 直接打开该约球详情：列表 Tab 过滤可能隐藏本人发出的邀请。
							uni.navigateTo({ url: `/subPages/match/challengeWaiting?challenge_id=${res.pending_challenge_id}` })
						}
					}
				})
			} else {
				if (!pageActive) return
				uni.showToast({ title: res?.message || '发送失败', icon: 'none' })
			}
			return
		}
		if (pageActive) uni.showToast({ title: '约球已发送', icon: 'success' })
		// 主 Tab 活动卡可能已挂载，标脏让 onShow 回源。
		useActivityStore().markDirty()
		setTimeout(() => {
			if (!pageActive || !isSameIdentity(identity)) return
			uni.switchTab({ url: '/pages/match/index' })
		}, 600)
	} catch (error) {
		if (pageActive && isSameIdentity(identity) && generation === pageGeneration) uni.showToast({ title: '发送失败', icon: 'none' })
	} finally {
		submitting.value = false
	}
}
</script>

<style lang="scss" scoped>
@import './challengeCompose.scss';
</style>
