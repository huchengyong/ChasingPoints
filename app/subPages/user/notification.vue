<template>
	<view class="notification-container" :class="{ 'dark-mode': isDarkMode }">
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
					:disabled="isSaving"
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
					:disabled="isSaving"
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
					:disabled="isSaving"
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
					:disabled="isSaving"
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
					:disabled="isSaving"
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
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { getNotificationPreferences, saveNotificationPreferences } from '@/api/notification.js'
import { useThemeStore } from '@/store/theme.js'

const themeStore = useThemeStore()
const isDarkMode = computed(() => themeStore.isDarkMode)
const isSaving = ref(false)
const preferences = reactive({
	match_result_enabled: true,
	friend_request_enabled: true,
	challenge_enabled: true,
	tournament_enabled: true,
	follow_enabled: true
})

onMounted(() => {
	loadNotificationSettings()
})

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
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
	try {
		const res = await getNotificationPreferences()
		applyPreferences(res)
	} catch (error) {
		console.error('加载通知偏好失败:', error)
		uni.showToast({
			title: '加载通知设置失败',
			icon: 'none'
		})
	}
}

const togglePreference = async (key, event) => {
	if (isSaving.value) return

	const previousValue = preferences[key]
	preferences[key] = event.detail.value
	isSaving.value = true

	try {
		const res = await saveNotificationPreferences(buildPreferencesPayload())
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
