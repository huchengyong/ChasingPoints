<template>
	<view class="settings-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="main-content">
			<!-- 菜单列表 -->
			<view class="menu-group">
				<view class="menu-item" @click="openEditNicknameModal">
					<view class="menu-left">
						<view class="icon-wrapper blue">
							<uni-icons type="person" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">编辑资料</text>
					</view>
					<view class="menu-right">
						<text class="menu-value">{{ userNickname }}</text>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
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
				<view v-if="showBindPhoneEntry" class="menu-item" @click="openBindPhoneModal">
					<view class="menu-left">
						<view class="icon-wrapper gold-soft">
							<uni-icons type="phone-filled" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">绑定手机号</text>
					</view>
					<view class="menu-right">
						<text class="menu-value">{{ bindPhoneEntryText }}</text>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
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
				<view class="menu-item" @click="handleHelp">
					<view class="menu-left">
						<view class="icon-wrapper amber">
							<uni-icons type="help-filled" size="24" color="#b45309"></uni-icons>
						</view>
						<text class="menu-text">意见反馈</text>
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

		<!-- 编辑昵称弹框 -->
		<view class="modal-overlay" v-if="showEditNicknameModal" @click="closeEditNicknameModal">
			<view class="modal-container" @click.stop>
				<text class="modal-title">编辑昵称</text>
				<view class="modal-input-wrapper">
					<input
						type="text"
						v-model="newNickname"
						class="modal-input"
						placeholder="请输入昵称"
						maxlength="20"
					/>
				</view>
				<view class="modal-buttons">
					<button class="modal-btn cancel" @click="closeEditNicknameModal">
						<text>取消</text>
					</button>
					<button class="modal-btn confirm" @click="handleSaveNickname" :disabled="isSaving">
						<text>{{ isSaving ? '保存中...' : '保存' }}</text>
					</button>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
import { useUserStore } from '@/store/user.js'
import { updateNickname } from '@/api/user.js'
import bindPhone from '@/components/bindPhone.vue'

// ========== 状态管理 ==========
const themeStore = useThemeStore()
const userStore = useUserStore()

// ========== 响应式数据 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)
const userNickname = computed(() => userStore.userInfo?.nickname || '用户')
const userPhone = computed(() => userStore.userInfo?.phone || '')
const showBindPhoneEntry = computed(() => Boolean(userStore.needBindPhone || !userPhone.value))
const bindPhoneEntryText = computed(() => (userPhone.value ? '继续完善' : '立即绑定'))
const showEditNicknameModal = ref(false)
const showBindPhoneModal = ref(false)
const newNickname = ref('')
const isSaving = ref(false)

// ========== 生命周期 ==========
onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	uni.$on(THEME_CHANGE_EVENT, handleThemeChange)
})

onHide(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

/**
 * 处理主题变化
 */
const handleThemeChange = () => {
	// 主题变化后更新导航栏
	themeStore.applyNavigationBarTheme()
}

// ========== 方法 ==========

/**
 * 打开编辑昵称弹框
 */
const openEditNicknameModal = () => {
	newNickname.value = userNickname.value
	showEditNicknameModal.value = true
}

/**
 * 关闭编辑昵称弹框
 */
const closeEditNicknameModal = () => {
	showEditNicknameModal.value = false
	newNickname.value = ''
}

/**
 * 保存昵称
 */
const handleSaveNickname = async () => {
	const nickname = newNickname.value.trim()
	
	// 验证昵称
	if (!nickname) {
		uni.showToast({
			title: '请输入昵称',
			icon: 'none'
		})
		return
	}
	
	if (nickname.length < 2 || nickname.length > 20) {
		uni.showToast({
			title: '昵称长度需要在2-20个字符之间',
			icon: 'none'
		})
		return
	}
	
	// 如果昵称没有变化，直接关闭
	if (nickname === userNickname.value) {
		closeEditNicknameModal()
		return
	}
	
	isSaving.value = true
	
	try {
		const res = await updateNickname(nickname)
		
		// 更新 Store 中的用户信息
		if (res.user_info) {
			userStore.updateUserInfo(res.user_info)
		} else {
			userStore.updateNickname(nickname)
		}
		
		uni.showToast({
			title: '更新成功',
			icon: 'success'
		})
		
		closeEditNicknameModal()
	} catch (error) {
		uni.showToast({
			title: error.message || '更新失败',
			icon: 'none'
		})
	} finally {
		isSaving.value = false
	}
}

/**
 * 切换深色模式
 */
const toggleDarkMode = () => {
	themeStore.toggleTheme()
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

	userStore.bindPhoneSuccess(payload?.maskedPhone || payload?.phone || '')
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

/**
 * 帮助与反馈
 */
const handleHelp = () => {
	uni.navigateTo({
		url: '/subPages/help/feedback'
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
