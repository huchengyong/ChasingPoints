<template>
	<view class="notification-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="settings-list">
			<!-- 比赛更新 -->
			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper green">
						<uni-icons type="tune-filled" size="24" color="#22c55e"></uni-icons>
					</view>
					<text class="item-text">对局通知</text>
				</view>
				<switch
					:checked="matchUpdates"
					@change="toggleMatchUpdates"
					color="#18b05b"
					style="transform:scale(0.8)"
				/>
			</view>

			<!-- 好友请求 -->
			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper blue">
						<uni-icons type="staff-filled" size="24" color="#18b05b"></uni-icons>
					</view>
					<text class="item-text">对手请求</text>
				</view>
				<switch
					:checked="friendRequests"
					@change="toggleFriendRequests"
					color="#18b05b"
					style="transform:scale(0.8)"
				/>
			</view>

			<!-- 系统公告 -->
			<view class="settings-item">
				<view class="item-left">
					<view class="icon-wrapper purple">
						<uni-icons type="sound-filled" size="24" color="#6366f1"></uni-icons>
					</view>
					<text class="item-text">系统公告</text>
				</view>
				<switch
					:checked="systemAnnouncements"
					@change="toggleSystemAnnouncements"
					color="#18b05b"
					style="transform:scale(0.8)"
				/>
			</view>
		</view>

		<text class="description-text">
			管理您希望从应用接收的通知类型。您可以在任何时候更改这些设置。
		</text>
	</view>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useThemeStore } from '@/store/theme.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 响应式数据 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)
const matchUpdates = ref(true)
const friendRequests = ref(true)
const systemAnnouncements = ref(false)

// ========== 生命周期 ==========
onMounted(() => {
	// TODO: 从本地存储或服务器加载通知设置
	loadNotificationSettings()
})

onShow(() => {
	// 同步主题状态并更新导航栏
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
})

// ========== 方法 ==========

/**
 * 加载通知设置
 */
const loadNotificationSettings = () => {
	const settings = uni.getStorageSync('notification_settings')
	if (settings) {
		try {
			const parsed = JSON.parse(settings)
			matchUpdates.value = parsed.matchUpdates ?? true
			friendRequests.value = parsed.friendRequests ?? true
			systemAnnouncements.value = parsed.systemAnnouncements ?? false
		} catch (e) {
			console.error('Failed to parse notification settings', e)
		}
	}
}

/**
 * 保存通知设置
 */
const saveSettings = () => {
	const settings = {
		matchUpdates: matchUpdates.value,
		friendRequests: friendRequests.value,
		systemAnnouncements: systemAnnouncements.value
	}
	uni.setStorageSync('notification_settings', JSON.stringify(settings))
}

/**
 * 切换比赛更新通知
 */
const toggleMatchUpdates = (e) => {
	matchUpdates.value = e.detail.value
	saveSettings()
}

/**
 * 切换好友请求通知
 */
const toggleFriendRequests = (e) => {
	friendRequests.value = e.detail.value
	saveSettings()
}

/**
 * 切换系统公告通知
 */
const toggleSystemAnnouncements = (e) => {
	systemAnnouncements.value = e.detail.value
	saveSettings()
}
</script>

<style lang="scss" scoped>
@import './notification.scss';
</style>
