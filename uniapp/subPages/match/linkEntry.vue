<template>
	<view class="entry-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="stage === 'loading'" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-text">正在打开邀请…</text>
		</view>

		<view v-else-if="stage === 'expired'" class="state-block">
			<text class="state-title">邀请已失效</text>
			<text class="state-text state-text--error">{{ errorMessage }}</text>
			<button class="primary-btn" @tap="handleBackHome">返回首页</button>
		</view>

		<view v-else-if="stage === 'error'" class="state-block">
			<text class="state-title">暂时无法打开邀请</text>
			<text class="state-text state-text--error">{{ errorMessage }}</text>
			<button class="primary-btn" @tap="handleRetry">重试</button>
			<button class="text-btn" @tap="handleBackHome">返回首页</button>
		</view>

		<template v-else-if="stage === 'preview'">
			<view class="inviter-card">
				<image
					v-if="inviter.avatar"
					class="inviter-avatar"
					:src="resolveAvatarUrl(inviter.avatar, 0)"
					mode="aspectFill"
				></image>
				<view v-else class="inviter-avatar inviter-avatar--empty">
					<uni-icons type="person" size="28" color="#9A8C67"></uni-icons>
				</view>
				<text class="inviter-name">{{ inviter.name }}</text>
				<text class="inviter-desc">邀请你打一局</text>
				<text v-if="remainingText" class="inviter-meta">{{ remainingText }}</text>
			</view>

			<button class="primary-btn" @tap="handleLogin">登录并开局</button>
			<text class="hint">登录后直接进入对局确认，无需再扫码</text>
		</template>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow, onUnload } from '@dcloudio/uni-app'
import { previewPublicMatchInvite, resolveMatchInviteCode } from '@/api/match.js'
import { useUserStore } from '@/store/user.js'
import { buildSpectateRoute, decodeLaunchQuery, parseEntryLink } from '@/utils/entry-link.js'
import { createLinkEntryFlow } from '@/utils/link-entry-flow.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { handoffResolvedScan } from '@/utils/quick-scan-entry.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const LOGIN_ROUTE = '/pages/login/login'
const HOME_ROUTE = '/pages/index/index'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

// loading：解析或兑换中；preview：未登录预览；expired：短码失效；error：网络或服务异常，可重试。
const stage = ref('loading')
const errorMessage = ref('')
const inviter = ref({ name: '', avatar: '' })
const expiresAt = ref('')
const now = ref(Date.now())

let timer = null
let started = false

const remainingText = computed(() => {
	if (!expiresAt.value) return ''
	const remaining = new Date(expiresAt.value).getTime() - now.value
	if (!Number.isFinite(remaining) || remaining <= 0) return ''
	const minutes = Math.ceil(remaining / 60000)
	return `${minutes} 分钟内有效`
})

const readIdentity = () => ({
	isLoggedIn: userStore.isLoggedIn,
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const flow = createLinkEntryFlow({
	getSession: readIdentity,
	previewPublicMatchInvite,
	resolveMatchInviteCode,
	handoffResolved: handoffResolvedScan
})

const stopTimer = () => {
	if (timer) clearInterval(timer)
	timer = null
}

const handleAction = (action) => {
	if (action.type === 'await_login') {
		inviter.value = action.inviter
		expiresAt.value = action.expiresAt
		stage.value = 'preview'
		stopTimer()
		// 短码 5 分钟有效，剩余时间每 30 秒刷新一次即可。
		timer = setInterval(() => { now.value = Date.now() }, 30000)
		return
	}
	if (action.type === 'confirm') {
		stopTimer()
		uni.redirectTo({ url: action.route })
		return
	}
	if (action.type === 'expired' || action.type === 'error') {
		stopTimer()
		errorMessage.value = action.message
		stage.value = action.type
	}
}

const handleRetry = async () => {
	stage.value = 'loading'
	const action = await flow.retry()
	// 被忽略（已有请求在途或登录态刚变化）时退回错误态，不让页面停在加载中。
	if (action.type === 'ignore') {
		if (stage.value === 'loading') stage.value = 'error'
		return
	}
	handleAction(action)
}

const handleBackHome = () => {
	uni.switchTab({ url: HOME_ROUTE })
}

const handleLogin = () => {
	uni.navigateTo({ url: LOGIN_ROUTE })
}

onLoad(async (options = {}) => {
	const parsed = parseEntryLink(decodeLaunchQuery(options.q))
	if (parsed.type === 'match') {
		uni.redirectTo({ url: buildSpectateRoute(parsed.matchId) })
		return
	}
	if (parsed.type !== 'invite') {
		handleBackHome()
		return
	}
	started = true
	handleAction(await flow.begin(parsed.inviteCode))
})

onShow(async () => {
	// 登录返回后才会真正兑换；进入时的首次 onShow 由 onLoad 负责。
	if (!started) return
	handleAction(await flow.resume())
})

onUnload(stopTimer)
</script>

<style lang="scss" scoped>
.entry-page {
	min-height: 100vh;
	padding: 120rpx 48rpx 60rpx;
	background: #f7f4ec;
	display: flex;
	flex-direction: column;
	align-items: center;
	box-sizing: border-box;

	&.dark-mode {
		background: #141109;
	}
}

.state-block,
.inviter-card {
	display: flex;
	flex-direction: column;
	align-items: center;
	width: 100%;
}

.state-block {
	padding-top: 160rpx;
}

.state-title {
	font-size: 36rpx;
	font-weight: 600;
	color: #231c0b;
	margin-bottom: 16rpx;

	.dark-mode & {
		color: #fff7e1;
	}
}

.state-text {
	font-size: 28rpx;
	color: #6e6242;
	margin-top: 24rpx;

	.dark-mode & {
		color: #d7c89b;
	}

	&--error {
		text-align: center;
	}
}

.inviter-card {
	background: #ffffff;
	border-radius: 24rpx;
	padding: 56rpx 40rpx;
	margin-bottom: 64rpx;

	.dark-mode & {
		background: #1e180d;
	}
}

.inviter-avatar {
	width: 160rpx;
	height: 160rpx;
	border-radius: 80rpx;
	background: #faf8f2;

	&--empty {
		display: flex;
		align-items: center;
		justify-content: center;
	}
}

.inviter-name {
	margin-top: 28rpx;
	font-size: 36rpx;
	font-weight: 600;
	color: #231c0b;
	max-width: 100%;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;

	.dark-mode & {
		color: #fff7e1;
	}
}

.inviter-desc {
	margin-top: 12rpx;
	font-size: 28rpx;
	color: #6e6242;

	.dark-mode & {
		color: #d7c89b;
	}
}

.inviter-meta {
	margin-top: 20rpx;
	font-size: 24rpx;
	color: #9a8c67;

	.dark-mode & {
		color: #9f926e;
	}
}

.primary-btn {
	width: 100%;
	height: 96rpx;
	line-height: 96rpx;
	border-radius: 48rpx;
	background: #e0ae12;
	color: #ffffff;
	font-size: 32rpx;
	font-weight: 600;
	border: none;
}

.text-btn {
	margin-top: 24rpx;
	background: transparent;
	color: #6e6242;
	font-size: 28rpx;
	border: none;

	&::after {
		border: none;
	}

	.dark-mode & {
		color: #d7c89b;
	}
}

.hint {
	margin-top: 28rpx;
	font-size: 24rpx;
	color: #9a8c67;

	.dark-mode & {
		color: #9f926e;
	}
}
</style>
