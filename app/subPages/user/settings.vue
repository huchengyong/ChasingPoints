<template>
	<view class="settings-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="main-content">
			<!-- 菜单列表 -->
			<view class="menu-group">
				<view class="menu-item avatar-item" @click="openEditProfile">
					<view class="menu-left">
						<view class="avatar-preview">
							<image class="avatar-image" :src="userAvatar" mode="aspectFill"></image>
						</view>
						<text class="menu-text">头像</text>
					</view>
					<view class="menu-right">
						<text class="menu-value">点击编辑</text>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="openEditProfile">
					<view class="menu-left">
						<view class="icon-wrapper blue">
							<uni-icons type="person" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">用户昵称</text>
					</view>
					<view class="menu-right">
						<text class="menu-value">{{ userNickname }}</text>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="handlePhoneRow">
					<view class="menu-left">
						<view class="icon-wrapper gold-soft">
							<uni-icons type="phone-filled" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">手机号</text>
					</view>
					<view class="menu-right">
						<text class="menu-value">{{ displayPhoneText }}</text>
						<uni-icons v-if="canBindPhone" type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="toggleDarkMode">
					<view class="menu-left">
						<view class="icon-wrapper gold">
							<uni-icons type="starhalf" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">深色模式</text>
					</view>
					<view class="menu-right">
						<view class="switch-wrapper">
							<view class="switch-track" :class="{ active: isDarkMode }">
								<view class="switch-thumb" :class="{ active: isDarkMode }"></view>
							</view>
						</view>
					</view>
				</view>
				<view class="menu-item" @click="handleNotifications">
					<view class="menu-left">
						<view class="icon-wrapper green">
							<uni-icons type="notification-filled" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">通知管理</text>
					</view>
					<view class="menu-right">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="handlePrivacy">
					<view class="menu-left">
						<view class="icon-wrapper rose">
							<uni-icons type="locked-filled" size="24" color="#d97706"></uni-icons>
						</view>
						<text class="menu-text">隐私政策</text>
					</view>
					<view class="menu-right">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="handleAgreement">
					<view class="menu-left">
						<view class="icon-wrapper gold-soft">
							<uni-icons type="flag-filled" size="24" color="#c69200"></uni-icons>
						</view>
						<text class="menu-text">用户协议</text>
					</view>
					<view class="menu-right">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
			</view>
			<view class="page-actions">
				<button class="logout-btn" @click="handleLogout">
					<text>退出登录</text>
				</button>
			</view>
		</view>

		<bindPhone
			:show="showBindPhoneModal"
			:closable="true"
			:is-dark-mode="isDarkMode"
			@close="handleBindPhoneClose"
			@success="handleBindPhoneSuccess"
		/>

	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { getUserInfo } from '@/api/user.js'
import bindPhone from '@/components/bindPhone.vue'
import { formatSettingsPhone } from '@/utils/settings-profile.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

// ========== 状态管理 ==========
const { isDarkMode, toggleTheme } = usePageTheme()
const userStore = useUserStore()

// ========== 响应式数据 ==========
const userNickname = computed(() => userStore.userInfo?.nickname || '用户')
const userPhone = computed(() => userStore.userInfo?.phone || '')
const userAvatar = computed(() => resolveAvatarUrl(userStore.userInfo?.avatar, userStore.userInfo?.id))
const canBindPhone = computed(() => Boolean(userStore.needBindPhone || !userPhone.value))
const displayPhoneText = computed(() => formatSettingsPhone(userPhone.value) || '未绑定')
const showBindPhoneModal = ref(false)

// ========== 生命周期 ==========
onShow(() => {
	fetchLatestUserInfo()
})

const fetchLatestUserInfo = async () => {
	try {
		const res = await getUserInfo()
		if (res?.success && res.user_info) {
			userStore.updateUserInfo(res.user_info)
		}
	} catch (error) {
		console.error('获取用户信息失败:', error)
	}
}

// ========== 方法 ==========

const openEditProfile = () => {
	uni.navigateTo({ url: '/subPages/user/editProfile' })
}

const toggleDarkMode = () => {
	toggleTheme()
}

const handlePhoneRow = () => {
	if (!canBindPhone.value) return
	openBindPhoneModal()
}

const openBindPhoneModal = () => {
	showBindPhoneModal.value = true
}

const handleBindPhoneClose = () => {
	showBindPhoneModal.value = false
}

const handleBindPhoneSuccess = (payload) => {
	showBindPhoneModal.value = false

	if (payload?.action === 'relogin') {
		userStore.logout()
		uni.showToast({
			title: payload.message || '账号已合并，请重新登录',
			icon: 'none'
		})
		uni.reLaunch({ url: '/pages/login/login' })
		return
	}

	if (!payload?.sessionReplaced) {
		userStore.bindPhoneSuccess(payload?.maskedPhone || payload?.phone || '')
	}
	uni.showToast({
		title: payload?.message || '绑定成功',
		icon: 'success'
	})
}

/**
 * 通知管理
 */
const handleNotifications = () => {
	uni.navigateTo({
		url: '/subPages/user/notification'
	})
}

/**
 * 隐私政策
 */
const handlePrivacy = () => {
	uni.navigateTo({
		url: '/subPages/agreement/privacyPolicy'
	})
}

/**
 * 用户协议
 */
const handleAgreement = () => {
	uni.navigateTo({
		url: '/subPages/agreement/userAgreement'
	})
}

const handleLogout = () => {
	uni.showModal({
		title: '提示',
		content: '确定要退出登录吗？',
		success: ({ confirm }) => {
			if (!confirm) return
			userStore.logout()
			uni.switchTab({ url: '/pages/user/index' })
			uni.showToast({
				title: '已退出登录',
				icon: 'success'
			})
		}
	})
}
</script>

<style lang="scss" scoped>
@import './settings.scss';
</style>
