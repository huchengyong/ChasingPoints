<template>
	<view class="settings-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="main-content">
			<!-- 菜单列表 -->
			<view class="menu-group">
				<view class="menu-item avatar-item" @click="chooseAvatarSource">
					<view class="menu-left">
						<view class="avatar-preview">
							<image class="avatar-image" :src="userAvatar" mode="aspectFill"></image>
						</view>
						<text class="menu-text">头像</text>
					</view>
					<view class="menu-right">
						<text class="menu-value">{{ isUploadingAvatar ? '上传中...' : '点击更换' }}</text>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#94a3b8'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="openEditNicknameModal">
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

		<!-- 编辑昵称弹框 -->
		<view class="modal-overlay" v-if="showEditNicknameModal" @click="closeEditNicknameModal">
			<view class="modal-container" @click.stop>
				<text class="modal-title">用户昵称</text>
				<view class="modal-input-wrapper">
					<input
						type="text"
						v-model="newNickname"
						class="modal-input"
						placeholder="请输入昵称"
						maxlength="12"
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
import { getQiniuUploadToken, getUserInfo, updateUserProfile } from '@/api/user.js'
import bindPhone from '@/components/bindPhone.vue'
import {
	buildQiniuAvatarObjectKey,
	formatSettingsPhone,
	getFileExtensionFromPath,
	normalizeQiniuUploadResult,
	normalizeQiniuUploadTokenResponse,
	uploadAvatarToQiniu
} from '@/utils/settings-profile.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()
const userStore = useUserStore()

// ========== 响应式数据 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)
const userNickname = computed(() => userStore.userInfo?.nickname || '用户')
const userPhone = computed(() => userStore.userInfo?.phone || '')
const userAvatar = computed(() => userStore.userInfo?.avatar || '/static/images/default-avatar.png')
const canBindPhone = computed(() => Boolean(userStore.needBindPhone || !userPhone.value))
const displayPhoneText = computed(() => formatSettingsPhone(userPhone.value) || '未绑定')
const showEditNicknameModal = ref(false)
const showBindPhoneModal = ref(false)
const newNickname = ref('')
const isSaving = ref(false)
const isUploadingAvatar = ref(false)

// ========== 生命周期 ==========
onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	uni.$on(THEME_CHANGE_EVENT, handleThemeChange)
	fetchLatestUserInfo()
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
	
	if (nickname.length < 2 || nickname.length > 12) {
		uni.showToast({
			title: '昵称长度需要在2-12个字符之间',
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
		const res = await updateUserProfile({
			nickname,
			avatar: userStore.userInfo?.avatar || ''
		})
		
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

const handlePhoneRow = () => {
	if (!canBindPhone.value) return
	openBindPhoneModal()
}

const chooseAvatarSource = () => {
	if (isUploadingAvatar.value) return

	uni.showActionSheet({
		itemList: ['拍照', '从手机相册选择'],
		success: ({ tapIndex }) => {
			if (tapIndex === 0) {
				pickAvatarFromCamera()
				return
			}
			if (tapIndex === 1) {
				pickAvatarFromAlbum()
			}
		}
	})
}

const pickAvatarFromCamera = () => {
	pickAvatarImage(['camera'])
}

const pickAvatarFromAlbum = () => {
	pickAvatarImage(['album'])
}

const pickAvatarImage = (sourceType) => {
	uni.chooseImage({
		count: 1,
		sizeType: ['compressed'],
		sourceType,
		success: async ({ tempFilePaths }) => {
			const filePath = tempFilePaths?.[0]
			if (!filePath) return
			await uploadAvatar(filePath)
		}
	})
}

const uploadAvatar = async (filePath) => {
	isUploadingAvatar.value = true

	try {
		const fileExt = getFileExtensionFromPath(filePath)
		const uploadTokenRes = await getQiniuUploadToken(fileExt)
		const uploadTokenPayload = normalizeQiniuUploadTokenResponse(uploadTokenRes)
		const uploadKey = uploadTokenPayload.key || buildQiniuAvatarObjectKey({
			userId: userStore.userInfo?.id,
			filePath
		})

		const uploadResult = await uploadAvatarToQiniu({
			filePath,
			uploadUrl: uploadTokenPayload.uploadUrl,
			uploadToken: uploadTokenPayload.uploadToken,
			key: uploadKey
		})

		const avatarUrl = normalizeQiniuUploadResult({
			domain: uploadTokenPayload.domain,
			key: uploadResult.key || uploadKey
		})

		if (!avatarUrl) {
			throw new Error('头像地址生成失败')
		}

		const res = await updateUserProfile({
			nickname: userNickname.value,
			avatar: avatarUrl
		})

		if (res?.success && res.user_info) {
			userStore.updateUserInfo(res.user_info)
		} else {
			userStore.updateUserInfo({ avatar: avatarUrl })
		}

		uni.showToast({
			title: '头像更新成功',
			icon: 'success'
		})
	} catch (error) {
		uni.showToast({
			title: error.message || '头像上传失败',
			icon: 'none'
		})
	} finally {
		isUploadingAvatar.value = false
	}
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
