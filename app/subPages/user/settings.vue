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
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
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
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
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
						<uni-icons v-if="canBindPhone" type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
					</view>
				</view>
				<view class="menu-item theme-menu-item">
					<view class="menu-left">
						<view class="icon-wrapper gold">
							<uni-icons type="starhalf" size="24" color="#E0AE12"></uni-icons>
						</view>
						<text class="menu-text">主题模式</text>
					</view>
					<view class="theme-options">
						<view
							v-for="option in themeOptions"
							:key="option.value"
							class="theme-option"
							:class="{ active: themeMode === option.value }"
							@click="setThemeMode(option.value)"
						>
							<text>{{ option.label }}</text>
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
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="toggleHideMatch">
					<view class="menu-left">
						<view class="icon-wrapper amber">
							<uni-icons type="locked-filled" size="24" color="#c69200"></uni-icons>
						</view>
						<view class="menu-copy">
							<text class="menu-text">隐藏战绩</text>
							<text class="menu-description">控制是否在公开场景展示你的战绩</text>
						</view>
					</view>
					<view class="privacy-switch" :class="{ active: isHideMatch }">
						<view class="privacy-switch-thumb" :class="{ active: isHideMatch }"></view>
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
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
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
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
					</view>
				</view>
				<view class="menu-item" @click="handleHelp">
					<view class="menu-left">
						<view class="icon-wrapper blue">
							<uni-icons type="chat" size="24" color="#E0AE12"></uni-icons>
						</view>
						<view class="menu-copy">
							<text class="menu-text">帮助、投诉与举报</text>
							<text class="menu-description">提交问题建议、投诉举报或获取帮助</text>
						</view>
					</view>
					<view class="menu-right">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
					</view>
				</view>
			</view>

			<text class="section-label">账号与数据</text>
			<view class="menu-group account-data-group">
				<view class="menu-item" @click="handleExportPersonalData">
					<view class="menu-left">
						<view class="icon-wrapper blue">
							<uni-icons type="download" size="24" color="#E0AE12"></uni-icons>
						</view>
						<view class="menu-copy">
							<text class="menu-text">导出个人数据</text>
							<text class="menu-description">生成完整 JSON 文件，可保存或分享</text>
						</view>
					</view>
					<view class="menu-right">
						<text class="menu-value">{{ exportInProgress ? exportProgressText : '导出' }}</text>
						<uni-icons type="right" size="20" :color="isDarkMode ? '#c6b78c' : '#9A8C67'"></uni-icons>
					</view>
				</view>
				<view class="menu-item danger-menu-item" @click="openDeleteAccountDialog">
					<view class="menu-left">
						<view class="icon-wrapper rose">
							<uni-icons type="trash" size="24" color="#dc2626"></uni-icons>
						</view>
						<view class="menu-copy">
							<text class="menu-text danger-text">注销账号</text>
							<text class="menu-description">删除个人资料与关系，此操作不可恢复</text>
						</view>
					</view>
					<view class="menu-right">
						<uni-icons type="right" size="20" :color="isDarkMode ? '#fca5a5' : '#dc2626'"></uni-icons>
					</view>
				</view>
			</view>

			<text class="section-label">对局偏好</text>
			<view class="menu-group preference-group">
				<view class="menu-item preference-menu-item">
					<view class="menu-left">
						<view class="icon-wrapper gold">
							<uni-icons type="flag-filled" size="24" color="#E0AE12"></uni-icons>
						</view>
						<view class="menu-copy">
							<text class="menu-text">默认对局类型</text>
							<text class="menu-description">发起 PK 时自动预选，仍可临时切换</text>
						</view>
					</view>
					<view class="preference-options">
						<button
							v-for="option in defaultGameTypeOptions"
							:key="option.value"
							class="preference-option"
							:class="{ active: defaultGameType === option.value }"
							@click="setDefaultGameType(option.value)"
						>
							{{ option.label }}
						</button>
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

		<view v-if="showDeleteAccountDialog" class="deletion-mask" @click="closeDeleteAccountDialog">
			<view class="deletion-dialog" @click.stop>
				<text class="deletion-title">{{ deleteAccountStep === 'warning' ? '注销账号' : '身份复核' }}</text>
				<template v-if="deleteAccountStep === 'warning'">
					<text class="deletion-copy">注销后，账号资料、认证关联和社交数据会被删除；必要的历史比赛会匿名保留。此操作不可恢复。</text>
					<view class="deletion-actions">
						<button class="dialog-secondary-btn" @click="closeDeleteAccountDialog">取消</button>
						<button class="dialog-danger-btn" @click="continueDeleteAccount">我已知晓，继续</button>
					</view>
				</template>
				<template v-else>
					<text class="deletion-copy">{{ hasBoundPhone ? '请输入已绑定的完整手机号并完成注销验证码验证。' : '将重新发起第三方授权，仅用于确认你的注销请求。' }}</text>
					<view v-if="hasBoundPhone" class="deletion-sms-fields">
						<input v-model.trim="deletePhone" class="deletion-input" type="number" maxlength="11" placeholder="已绑定的完整手机号" />
						<view class="deletion-code-row">
							<input v-model.trim="deleteSmsCode" class="deletion-input deletion-code-input" type="number" maxlength="6" placeholder="注销验证码" />
							<button class="deletion-send-code-btn" :disabled="deleteSmsSending" @click="sendDeleteAccountSms">{{ deleteSmsSending ? '发送中…' : '获取验证码' }}</button>
						</view>
					</view>
					<view class="deletion-actions">
						<button class="dialog-secondary-btn" :disabled="deleteAccountSubmitting" @click="closeDeleteAccountDialog">取消</button>
						<button class="dialog-danger-btn" :disabled="deleteAccountSubmitting" @click="submitDeleteAccount">{{ deleteAccountSubmitting ? '处理中…' : '确认注销' }}</button>
					</view>
				</template>
			</view>
		</view>

	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { deleteAccount, exportPersonalData, getUserPrivacy, updateUserPrivacy } from '@/api/user.js'
import { sendSms } from '@/api/auth.js'
import bindPhone from '@/components/bindPhone.vue'
import { formatSettingsPhone } from '@/utils/settings-profile.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'
import { buildHuaweiOAuthReauthPayload } from '@/utils/oauth-login.js'
import { buildPersonalDataExportFilename, exportPersonalDataToWriter } from '@/utils/personal-data-export.js'
import {
	createPersonalDataExportFileWriter,
	getPersonalDataExportCapability,
	openPersonalDataExportFile
} from '@/utils/personal-data-export-file.js'
import {
	DEFAULT_GAME_TYPE_OPTIONS,
	readDefaultGameType,
	saveDefaultGameType
} from '@/utils/game-type-preference.js'

// ========== 状态管理 ==========
const { isDarkMode, themeMode, setThemeMode } = usePageTheme()
const userStore = useUserStore()
const themeOptions = [
	{ value: 'system', label: '跟随系统' },
	{ value: 'light', label: '浅色' },
	{ value: 'dark', label: '深色' }
]
const defaultGameTypeOptions = DEFAULT_GAME_TYPE_OPTIONS

// ========== 响应式数据 ==========
const userNickname = computed(() => userStore.userInfo?.nickname || '用户')
const userPhone = computed(() => userStore.userInfo?.phone || '')
const userAvatar = computed(() => resolveAvatarUrl(userStore.userInfo?.avatar, userStore.userInfo?.id))
const canBindPhone = computed(() => Boolean(userStore.needBindPhone || !userPhone.value))
const hasBoundPhone = computed(() => Boolean(userPhone.value))
const displayPhoneText = computed(() => formatSettingsPhone(userPhone.value) || '未绑定')
const showBindPhoneModal = ref(false)
const defaultGameType = ref(0)
const isHideMatch = ref(false)
const hideMatchLoading = ref(false)
const privacyLoadedAt = ref(0)
const privacyDirty = ref(true)
const PRIVACY_CACHE_TTL = 5 * 60 * 1000
const exportInProgress = ref(false)
const exportProgressText = ref('')
const showDeleteAccountDialog = ref(false)
const deleteAccountStep = ref('warning')
const deletePhone = ref('')
const deleteSmsCode = ref('')
const deleteSmsSending = ref(false)
const deleteAccountSubmitting = ref(false)

// ========== 生命周期 ==========
onShow(() => {
	defaultGameType.value = readDefaultGameType(uni, userStore.userId)
	loadUserPrivacy()
})

const loadUserPrivacy = async ({ force = false } = {}) => {
	if (!force && !privacyDirty.value && Date.now() - privacyLoadedAt.value < PRIVACY_CACHE_TTL) return
	try {
		const res = await getUserPrivacy()
		isHideMatch.value = Boolean(res?.success && res.hide_match_record)
		privacyLoadedAt.value = Date.now()
		privacyDirty.value = false
	} catch (error) {
		console.error('获取用户隐私设置失败:', error)
	}
}

// ========== 方法 ==========

const openEditProfile = () => {
	uni.navigateTo({ url: '/subPages/user/editProfile' })
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

const setDefaultGameType = (gameType) => {
	defaultGameType.value = saveDefaultGameType(uni, userStore.userId, gameType)
	uni.showToast({
		title: defaultGameType.value > 0 ? '默认对局类型已更新' : '已改为每次询问',
		icon: 'none'
	})
}

const toggleHideMatch = async () => {
	if (hideMatchLoading.value) return

	const previousValue = isHideMatch.value
	const nextValue = !previousValue
	isHideMatch.value = nextValue
	hideMatchLoading.value = true

	try {
		const res = await updateUserPrivacy({ hide_match_record: nextValue })
		if (!res.success) {
			throw new Error(res.message || '更新隐私设置失败')
		}

		isHideMatch.value = Boolean(res.hide_match_record)
		privacyLoadedAt.value = Date.now()
		privacyDirty.value = false
		uni.showToast({
			title: res.hide_match_record ? '已隐藏战绩' : '已公开战绩',
			icon: 'none'
		})
	} catch (error) {
		console.error('更新隐藏战绩失败:', error)
		isHideMatch.value = previousValue
		uni.showToast({
			title: error.message || '更新失败',
			icon: 'none'
		})
	} finally {
		hideMatchLoading.value = false
	}
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

const handleHelp = () => {
	uni.navigateTo({ url: '/subPages/help/feedback' })
}

const handleExportPersonalData = async () => {
	if (exportInProgress.value) return
	const capability = getPersonalDataExportCapability()
	if (!capability.supported) {
		uni.showToast({ title: capability.message, icon: 'none' })
		return
	}

	exportInProgress.value = true
	exportProgressText.value = '准备中…'
	try {
		const fileName = buildPersonalDataExportFilename(userStore.userId)
		const writer = await createPersonalDataExportFileWriter({ fileName })
		const result = await exportPersonalDataToWriter({
			userId: userStore.userId,
			fileName,
			writer,
			requestPage: async (cursor) => {
				const response = await exportPersonalData(cursor)
				if (!response?.success) throw new Error(response?.message || '导出分段失败')
				return response
			},
			onProgress: ({ itemCount, complete }) => {
				exportProgressText.value = complete ? '正在完成…' : `已整理 ${itemCount} 条`
			}
		})
		exportProgressText.value = '已完成'
		uni.showModal({
			title: '导出完成',
			content: `已生成 ${result.itemCount} 条个人数据，是否打开文件？`,
			success: ({ confirm }) => {
				if (!confirm || !result.saved?.filePath) return
				openPersonalDataExportFile({ filePath: result.saved.filePath }).catch(() => {
					uni.showToast({ title: '文件已生成，可在设备文件中查看', icon: 'none' })
				})
			}
		})
	} catch (error) {
		uni.showToast({ title: error.message || '导出失败，请重试', icon: 'none' })
	} finally {
		exportInProgress.value = false
		exportProgressText.value = ''
	}
}

const openDeleteAccountDialog = () => {
	if (deleteAccountSubmitting.value) return
	deleteAccountStep.value = 'warning'
	deletePhone.value = ''
	deleteSmsCode.value = ''
	showDeleteAccountDialog.value = true
}

const closeDeleteAccountDialog = ({ force = false } = {}) => {
	if (deleteAccountSubmitting.value && !force) return
	showDeleteAccountDialog.value = false
	deleteAccountStep.value = 'warning'
}

const continueDeleteAccount = () => {
	deleteAccountStep.value = 'verify'
}

const sendDeleteAccountSms = async () => {
	if (deleteSmsSending.value) return
	if (!/^1[3-9]\d{9}$/.test(deletePhone.value)) {
		uni.showToast({ title: '请输入已绑定的完整手机号', icon: 'none' })
		return
	}
	deleteSmsSending.value = true
	try {
		const response = await sendSms(deletePhone.value, 'delete_account')
		if (!response?.success) throw new Error(response?.message || '验证码发送失败')
		uni.showToast({ title: '注销验证码已发送', icon: 'none' })
	} catch (error) {
		uni.showToast({ title: error.message || '验证码发送失败', icon: 'none' })
	} finally {
		deleteSmsSending.value = false
	}
}

const requestHuaweiDeletionCredential = async () => {
	if (typeof uni.login !== 'function') {
		throw new Error('当前设备暂不支持第三方身份验证')
	}
	const loginResult = await new Promise((resolve, reject) => {
		uni.login({ provider: 'huawei', success: resolve, fail: reject })
	})
	const platform = uni.getSystemInfoSync?.()?.uniPlatform || 'app-plus'
	return buildHuaweiOAuthReauthPayload({ loginResult, platform })
}

const requestOAuthDeletionCredential = async () => {
	const platform = uni.getSystemInfoSync?.()?.uniPlatform || ''
	if (platform === 'mp-weixin') {
		if (typeof uni.login !== 'function') throw new Error('当前设备暂不支持第三方身份验证')
		const loginResult = await new Promise((resolve, reject) => {
			uni.login({ success: resolve, fail: reject })
		})
		const credential = String(loginResult?.code || '').trim()
		if (!credential) throw new Error('微信授权凭据缺失，请重试')
		return {
			provider: 'weixin_mini_program',
			credential,
			credential_type: 'login_code',
			platform: 'mp-weixin'
		}
	}
	return requestHuaweiDeletionCredential()
}

const submitDeleteAccount = async () => {
	if (deleteAccountSubmitting.value) return
	let verification
	if (hasBoundPhone.value) {
		if (!/^1[3-9]\d{9}$/.test(deletePhone.value) || !/^\d{6}$/.test(deleteSmsCode.value)) {
			uni.showToast({ title: '请填写完整手机号和注销验证码', icon: 'none' })
			return
		}
		verification = { verify_type: 'sms', sms_code: deleteSmsCode.value }
	} else {
		try {
			verification = { verify_type: 'oauth', ...(await requestOAuthDeletionCredential()) }
		} catch (error) {
			uni.showToast({ title: error.message || '第三方身份验证失败', icon: 'none' })
			return
		}
	}

	deleteAccountSubmitting.value = true
	try {
		const response = await deleteAccount({ confirm_text: '注销账号', ...verification })
		if (!response?.success) throw new Error(response?.message || '注销失败')
		closeDeleteAccountDialog({ force: true })
		userStore.logout()
		uni.showToast({ title: '账号已注销', icon: 'success' })
		uni.reLaunch({ url: '/pages/login/login' })
	} catch (error) {
		uni.showToast({ title: error.message || '注销失败，请重试', icon: 'none' })
	} finally {
		deleteAccountSubmitting.value = false
	}
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
