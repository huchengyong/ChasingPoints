<template>
	<view class="notification-container" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="notificationSettingsState.status === ASYNC_PAGE_STATUS.IDLE || notificationSettingsState.status === ASYNC_PAGE_STATUS.LOADING" class="loading-state">
			<uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
			<text>正在加载通知偏好...</text>
		</view>

		<view v-else-if="notificationSettingsPageError" class="error-state">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="error-title">{{ notificationSettingsPageError.title }}</text>
			<text class="error-text">{{ notificationSettingsPageError.description }}</text>
			<button class="retry-btn" @tap="handleNotificationSettingsErrorAction">{{ notificationSettingsPageError.actionText }}</button>
		</view>

		<view v-else>
			<view v-if="notificationSettingsRefreshError" class="refresh-error-banner">
				<text>{{ notificationSettingsRefreshError.description }}</text>
				<text class="refresh-error-action" @tap="retryNotificationSettings">重试</text>
			</view>
			<view class="settings-list">
			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper green">
						<uni-icons type="tune-filled" size="24" color="#E0AE12"></uni-icons>
					</view>
					<text class="item-text">对局结果</text>
				</view>
				<switch
					:checked="preferences.match_result_enabled"
					:disabled="isSaving || notificationSettingsState.status !== ASYNC_PAGE_STATUS.READY"
					@change="(e) => togglePreference('match_result_enabled', e)"
					color="#E0AE12"
					style="transform:scale(0.8)"
				/>
			</view>

			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper blue">
						<uni-icons type="staff-filled" size="24" color="#E0AE12"></uni-icons>
					</view>
					<text class="item-text">好友申请</text>
				</view>
				<switch
					:checked="preferences.friend_request_enabled"
					:disabled="isSaving || notificationSettingsState.status !== ASYNC_PAGE_STATUS.READY"
					@change="(e) => togglePreference('friend_request_enabled', e)"
					color="#E0AE12"
					style="transform:scale(0.8)"
				/>
			</view>

			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper purple">
						<uni-icons type="notification-filled" size="24" color="#E0AE12"></uni-icons>
					</view>
					<text class="item-text">挑战提醒</text>
				</view>
				<switch
					:checked="preferences.challenge_enabled"
					:disabled="isSaving || notificationSettingsState.status !== ASYNC_PAGE_STATUS.READY"
					@change="(e) => togglePreference('challenge_enabled', e)"
					color="#E0AE12"
					style="transform:scale(0.8)"
				/>
			</view>

			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper orange">
						<uni-icons type="notification-filled" size="24" color="#E0AE12"></uni-icons>
					</view>
					<text class="item-text">赛事报名提醒</text>
				</view>
				<switch
					:checked="preferences.tournament_enabled"
					:disabled="isSaving || notificationSettingsState.status !== ASYNC_PAGE_STATUS.READY"
					@change="(e) => togglePreference('tournament_enabled', e)"
					color="#E0AE12"
					style="transform:scale(0.8)"
				/>
			</view>

			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper gold">
						<uni-icons type="notification-filled" size="24" color="#E0AE12"></uni-icons>
					</view>
					<text class="item-text">新关注提醒</text>
				</view>
				<switch
					:checked="preferences.follow_enabled"
					:disabled="isSaving || notificationSettingsState.status !== ASYNC_PAGE_STATUS.READY"
					@change="(e) => togglePreference('follow_enabled', e)"
					color="#E0AE12"
					style="transform:scale(0.8)"
				/>
			</view>
			</view>

			<text class="description-text">
				管理消息中心和推送共同使用的通知类型。关闭后，仅影响后续新消息，不清理历史消息。
			</text>
		</view>
	</view>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { getNotificationPreferences, saveNotificationPreferences } from '@/api/notification.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
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
const isSaving = ref(false)
const preferences = reactive({
	match_result_enabled: true,
	friend_request_enabled: true,
	challenge_enabled: true,
	tournament_enabled: true,
	follow_enabled: true
})
const notificationSettingsState = ref(createAsyncPageState({
	authGeneration: userStore.authGeneration,
	data: null
}))
const notificationSettingsPageError = computed(() => (
	notificationSettingsState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(notificationSettingsState.value.error, { resource: '通知偏好' })
		: null
))
const notificationSettingsRefreshError = computed(() => (
	notificationSettingsState.value.refreshError
		? resolveAsyncPageErrorFeedback(notificationSettingsState.value.refreshError, { resource: '通知偏好' })
		: null
))

const currentIdentityKey = () => `${userStore.userId}:${userStore.authGeneration}`

onMounted(() => {
	loadNotificationSettings()
})

onShow(() => {
	if (notificationSettingsState.value.authGeneration !== userStore.authGeneration) {
		loadNotificationSettings()
	}
})

const applyPreferences = (payload = {}) => {
	preferences.match_result_enabled = payload.match_result_enabled ?? true
	preferences.friend_request_enabled = payload.friend_request_enabled ?? true
	preferences.challenge_enabled = payload.challenge_enabled ?? true
	preferences.tournament_enabled = payload.tournament_enabled ?? true
	preferences.follow_enabled = payload.follow_enabled ?? true
}

const buildPreferencesPayload = () => {
	return {
		match_result_enabled: preferences.match_result_enabled,
		friend_request_enabled: preferences.friend_request_enabled,
		challenge_enabled: preferences.challenge_enabled,
		tournament_enabled: preferences.tournament_enabled,
		follow_enabled: preferences.follow_enabled
	}
}

const loadNotificationSettings = async () => {
	const requestIdentityKey = currentIdentityKey()
	if (notificationSettingsState.value.authGeneration !== userStore.authGeneration) {
		applyPreferences()
		notificationSettingsState.value = createAsyncPageState({ authGeneration: userStore.authGeneration, data: null })
	}
	const nextState = beginAsyncPageLoad(notificationSettingsState.value, {
		authGeneration: userStore.authGeneration,
		emptyData: null
	})
	const pageRequest = getAsyncPageRequest(nextState)
	notificationSettingsState.value = nextState
	try {
		const res = await getNotificationPreferences()
		if (currentIdentityKey() !== requestIdentityKey || notificationSettingsState.value.requestId !== pageRequest.requestId || notificationSettingsState.value.authGeneration !== pageRequest.authGeneration) return
		if (!res || res.success === false) throw createRequestError({ message: res?.message || '加载通知设置失败', category: 'business' })
		applyPreferences(res)
		notificationSettingsState.value = resolveAsyncPageLoad(notificationSettingsState.value, pageRequest, {
			data: res,
			isEmpty: () => false
		})
	} catch (error) {
		if (currentIdentityKey() !== requestIdentityKey || notificationSettingsState.value.requestId !== pageRequest.requestId || notificationSettingsState.value.authGeneration !== pageRequest.authGeneration) return
		notificationSettingsState.value = rejectAsyncPageLoad(notificationSettingsState.value, pageRequest, error)
	}
}

const retryNotificationSettings = () => loadNotificationSettings()

const handleNotificationSettingsErrorAction = () => {
	if (notificationSettingsState.value.error?.category === 'permission' || notificationSettingsState.value.error?.category === 'not-found') {
		uni.navigateBack({ delta: 1 })
		return
	}
	retryNotificationSettings()
}

const togglePreference = async (key, event) => {
	if (isSaving.value) return

	const previousValue = preferences[key]
	preferences[key] = event.detail.value
	isSaving.value = true

	try {
		const res = await saveNotificationPreferences(buildPreferencesPayload())
		if (!res || res.success === false) throw createRequestError({ message: res?.message || '保存失败，请重试', category: 'business' })
		applyPreferences(res)
	} catch (error) {
		preferences[key] = previousValue
		console.error('保存通知偏好失败:', error)
		uni.showToast({
			title: '保存失败，请重试',
			icon: 'none'
		})
	} finally {
		isSaving.value = false
	}
}
</script>

<style lang="scss" scoped>
@import './notification.scss';
</style>
