<template>
	<view class="feedback-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="form-section">
			<!-- 反馈内容 -->
			<view class="form-item">
				<text class="form-label">反馈内容</text>
				<textarea
					class="feedback-textarea"
					v-model="feedbackContent"
					placeholder="请详细描述您遇到的问题或建议..."
					:maxlength="500"
				/>
				<text class="char-count">{{ feedbackContent.length }}/500</text>
			</view>

			<!-- 联系邮箱 -->
			<view class="form-item">
				<text class="form-label">联系邮箱 (选填)</text>
				<input
					class="email-input"
					type="text"
					v-model="contactEmail"
					placeholder="您的邮箱地址"
				/>
			</view>
		</view>

		<!-- 提交按钮 -->
		<view class="submit-section">
			<button class="submit-btn" @click="handleSubmit">提交</button>
		</view>
	</view>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 响应式数据 ==========
const feedbackContent = ref('')
const contactEmail = ref('')

// ========== 计算属性 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)

// ========== 生命周期 ==========
onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
	// 监听主题变化事件
	uni.$on(THEME_CHANGE_EVENT, handleThemeChange)
})

onHide(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

onUnmounted(() => {
	uni.$off(THEME_CHANGE_EVENT, handleThemeChange)
})

// ========== 方法 ==========

/**
 * 处理主题变化事件
 */
const handleThemeChange = () => {
	themeStore.applyNavigationBarTheme()
}


/**
 * 提交反馈
 */
const handleSubmit = () => {
	if (!feedbackContent.value.trim()) {
		uni.showToast({
			title: '请输入反馈内容',
			icon: 'none'
		})
		return
	}

	// 显示提交成功弹框
	uni.showToast({
		title: '提交成功',
		icon: 'success',
		duration: 2000
	})

	// 清空表单
	setTimeout(() => {
		feedbackContent.value = ''
		contactEmail.value = ''
		// 返回上一页
		uni.navigateBack()
	}, 1500)
}
</script>

<style lang="scss" scoped>
@import './feedback.scss';
</style>
