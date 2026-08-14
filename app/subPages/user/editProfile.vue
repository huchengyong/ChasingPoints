<template>
	<view class="edit-profile-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="profile-header">
			<!-- #ifdef MP-WEIXIN -->
			<button
				class="avatar-picker avatar-picker-wechat"
				open-type="chooseAvatar"
				:disabled="isUploadingAvatar"
				@chooseavatar="handleChooseAvatar"
			>
				<image class="profile-avatar" :src="userAvatar" mode="aspectFill"></image>
				<view class="camera-badge">
					<uni-icons type="camera-filled" size="18" color="#ffffff"></uni-icons>
				</view>
			</button>
			<!-- #endif -->

			<!-- #ifndef MP-WEIXIN -->
			<view class="avatar-picker" @click="chooseAvatarSource">
				<image class="profile-avatar" :src="userAvatar" mode="aspectFill"></image>
				<view class="camera-badge">
					<uni-icons type="camera-filled" size="18" color="#ffffff"></uni-icons>
				</view>
			</view>
			<!-- #endif -->

			<text class="avatar-hint">{{ isUploadingAvatar ? '上传中...' : '点击更换头像' }}</text>
		</view>

		<view class="profile-card">
			<view class="profile-row" @click="openEditNicknameModal">
				<text class="row-label">昵称</text>
				<view class="row-value-wrap">
					<text class="row-value">{{ userNickname }}</text>
					<uni-icons type="right" size="20" :color="isDarkMode ? '#8d836c' : '#b8ae9a'"></uni-icons>
				</view>
			</view>
			<view class="profile-row" :class="{ actionable: canBindPhone }" @click="handlePhoneRow">
				<text class="row-label">手机号</text>
				<view class="row-value-wrap">
					<text class="row-value" :class="{ placeholder: !userPhone }">{{ displayPhoneText }}</text>
					<uni-icons v-if="canBindPhone" type="right" size="20" :color="isDarkMode ? '#8d836c' : '#b8ae9a'"></uni-icons>
				</view>
			</view>
		</view>

		<view v-if="showEditNicknameModal" class="modal-overlay" @click="closeEditNicknameModal">
			<view class="modal-card" @click.stop>
				<text class="modal-title">编辑昵称</text>
				<input
					class="modal-input"
					type="text"
					v-model="newNickname"
					placeholder="请输入昵称"
					maxlength="12"
				/>
				<view class="modal-actions">
					<button class="modal-button modal-button-secondary" @click="closeEditNicknameModal">取消</button>
					<button class="modal-button modal-button-primary" :disabled="isSaving" @click="handleSaveNickname">
						{{ isSaving ? '保存中...' : '保存' }}
					</button>
				</view>
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
import { computed, ref } from 'vue'
import { usePageTheme } from '@/utils/page-theme.js'
import { useUserStore } from '@/store/user.js'
import { getQiniuUploadToken, updateUserProfile } from '@/api/user.js'
import bindPhone from '@/components/bindPhone.vue'
import {
	prepareAvatarForUpload,
	buildQiniuAvatarObjectKey,
	formatSettingsPhone,
	getFileExtensionFromPath,
	normalizeQiniuUploadResult,
	normalizeQiniuUploadTokenResponse,
	uploadAvatarToQiniu
} from '@/utils/settings-profile.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const userNickname = computed(() => userStore.userInfo?.nickname || '用户')
const userPhone = computed(() => userStore.userInfo?.phone || '')
const userAvatar = computed(() => resolveAvatarUrl(userStore.userInfo?.avatar, userStore.userInfo?.id))
const displayPhoneText = computed(() => formatSettingsPhone(userPhone.value) || '未绑定')
const canBindPhone = computed(() => Boolean(userStore.needBindPhone || !userPhone.value))
const showEditNicknameModal = ref(false)
const showBindPhoneModal = ref(false)
const newNickname = ref('')
const isSaving = ref(false)
const isUploadingAvatar = ref(false)

const chooseAvatarSource = () => {
	if (isUploadingAvatar.value) return
	uni.showActionSheet({
		itemList: ['拍照', '从手机相册选择'],
		success: ({ tapIndex }) => {
			if (tapIndex === 0) pickAvatarImage(['camera'])
			if (tapIndex === 1) pickAvatarImage(['album'])
		}
	})
}

const pickAvatarImage = (sourceType) => {
	uni.chooseImage({
		count: 1,
		sizeType: ['compressed'],
		sourceType,
		success: async ({ tempFilePaths }) => {
			if (tempFilePaths?.[0]) await uploadAvatar(tempFilePaths[0])
		}
	})
}

const handleChooseAvatar = (event) => {
	const avatarUrl = event?.detail?.avatarUrl
	if (avatarUrl) uploadAvatar(avatarUrl)
}

const uploadAvatar = async (filePath) => {
	if (isUploadingAvatar.value) return
	isUploadingAvatar.value = true
	try {
		const preparedFilePath = await prepareAvatarForUpload(filePath)
		const fileExt = getFileExtensionFromPath(preparedFilePath)
		const uploadTokenRes = await getQiniuUploadToken(fileExt)
		const uploadTokenPayload = normalizeQiniuUploadTokenResponse(uploadTokenRes)
		const uploadKey = uploadTokenPayload.key || buildQiniuAvatarObjectKey({
			userId: userStore.userInfo?.id,
			filePath: preparedFilePath
		})
		const uploadResult = await uploadAvatarToQiniu({
			filePath: preparedFilePath,
			uploadUrl: uploadTokenPayload.uploadUrl,
			uploadToken: uploadTokenPayload.uploadToken,
			key: uploadKey
		})
		const avatarUrl = normalizeQiniuUploadResult({
			domain: uploadTokenPayload.domain,
			key: uploadResult.key || uploadKey
		})
		if (!avatarUrl) throw new Error('头像地址生成失败')

		const res = await updateUserProfile({
			nickname: userNickname.value,
			avatar: avatarUrl
		})
		if (res?.success && res.user_info) {
			userStore.updateUserInfo(res.user_info)
		} else {
			userStore.updateUserInfo({ avatar: avatarUrl })
		}
		uni.showToast({ title: '头像更新成功', icon: 'success' })
	} catch (error) {
		uni.showToast({ title: error.message || '头像上传失败', icon: 'none' })
	} finally {
		isUploadingAvatar.value = false
	}
}

const openEditNicknameModal = () => {
	newNickname.value = userNickname.value
	showEditNicknameModal.value = true
}

const closeEditNicknameModal = () => {
	if (isSaving.value) return
	showEditNicknameModal.value = false
	newNickname.value = ''
}

const handleSaveNickname = async () => {
	const nickname = newNickname.value.trim()
	if (!nickname) {
		uni.showToast({ title: '请输入昵称', icon: 'none' })
		return
	}
	if (nickname.length < 2 || nickname.length > 12) {
		uni.showToast({ title: '昵称长度需要在2-12个字符之间', icon: 'none' })
		return
	}
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
		if (res?.user_info) {
			userStore.updateUserInfo(res.user_info)
		} else {
			userStore.updateNickname(nickname)
		}
		uni.showToast({ title: '更新成功', icon: 'success' })
		showEditNicknameModal.value = false
		newNickname.value = ''
	} catch (error) {
		uni.showToast({ title: error.message || '更新失败', icon: 'none' })
	} finally {
		isSaving.value = false
	}
}

const handlePhoneRow = () => {
	if (canBindPhone.value) showBindPhoneModal.value = true
}

const handleBindPhoneClose = () => {
	showBindPhoneModal.value = false
}

const handleBindPhoneSuccess = (payload) => {
	showBindPhoneModal.value = false
	if (payload?.action === 'relogin') {
		userStore.logout()
		uni.showToast({ title: payload.message || '账号已合并，请重新登录', icon: 'none' })
		uni.reLaunch({ url: '/pages/login/login' })
		return
	}
	if (!payload?.sessionReplaced) {
		userStore.bindPhoneSuccess(payload?.maskedPhone || payload?.phone || '')
	}
	uni.showToast({ title: payload?.message || '绑定成功', icon: 'success' })
}
</script>

<style lang="scss" scoped>
@import './editProfile.scss';
</style>
