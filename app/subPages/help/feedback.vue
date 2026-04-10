<template>
	<view class="feedback-container" :class="{ 'dark-mode': isDarkMode }">
		<view class="form-section">
			<view class="form-tip">
				<text>此入口受理使用问题、投诉与举报。涉及账号、内容或线下纠纷时，请尽量补充对象、时间和证据线索。</text>
			</view>

			<view class="form-item">
				<text class="form-label">提交类型</text>
				<picker mode="selector" :range="categoryLabels" :value="categoryIndex" @change="handleCategoryChange">
					<view class="category-picker">
						<text>{{ selectedCategory.label }}</text>
						<uni-icons type="right" size="16" :color="isDarkMode ? '#64748b' : '#94a3b8'"></uni-icons>
					</view>
				</picker>
			</view>

			<view class="form-item">
				<text class="form-label">投诉举报或反馈内容</text>
				<textarea
					class="feedback-textarea"
					v-model="feedbackContent"
					placeholder="请详细描述您遇到的问题、投诉或举报事项..."
					:maxlength="contentMaxLength"
				/>
				<text class="char-count">{{ feedbackContent.length }}/{{ contentMaxLength }}</text>
			</view>

			<view class="form-item">
				<text class="form-label">联系方式 (选填)</text>
				<input
					class="contact-input"
					type="text"
					v-model="contactInfo"
					placeholder="手机号或邮箱，便于处理人员联系您"
				/>
			</view>
		</view>

		<view class="submit-section">
			<button class="submit-btn" :disabled="isSubmitting" @click="handleSubmit">
				{{ isSubmitting ? '提交中...' : '提交投诉举报与反馈' }}
			</button>
		</view>
	</view>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { onShow, onHide } from '@dcloudio/uni-app'
import { useThemeStore, THEME_CHANGE_EVENT } from '@/store/theme.js'
import { createUserFeedbackTicket } from '@/api/feedback.js'
import {
	FEEDBACK_CATEGORY_OPTIONS,
	buildFeedbackTicketPayload
} from '@/utils/feedback-ticket.js'

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 响应式数据 ==========
const feedbackContent = ref('')
const contactInfo = ref('')
const categoryIndex = ref(0)
const isSubmitting = ref(false)
const contentMaxLength = 1000

// ========== 计算属性 ==========
const isDarkMode = computed(() => themeStore.isDarkMode)
const categoryLabels = computed(() => FEEDBACK_CATEGORY_OPTIONS.map((item) => item.label))
const selectedCategory = computed(() => FEEDBACK_CATEGORY_OPTIONS[categoryIndex.value] || FEEDBACK_CATEGORY_OPTIONS[0])

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

const handleCategoryChange = (event) => {
	categoryIndex.value = Number(event.detail.value) || 0
}

/**
 * 提交投诉举报与反馈
 */
const handleSubmit = async () => {
	if (isSubmitting.value) return

	const payload = buildFeedbackTicketPayload({
		category: selectedCategory.value.value,
		content: feedbackContent.value,
		contact: contactInfo.value
	})

	if (!payload.content) {
		uni.showToast({
			title: '请输入投诉举报或反馈内容',
			icon: 'none'
		})
		return
	}

	isSubmitting.value = true
	try {
		const res = await createUserFeedbackTicket(payload)
		if (!res.success) {
			throw new Error(res.message || '提交失败')
		}
		uni.showToast({
			title: '提交成功',
			icon: 'success',
			duration: 2000
		})
		feedbackContent.value = ''
		contactInfo.value = ''
		categoryIndex.value = 0
		setTimeout(() => {
			uni.navigateBack()
		}, 1200)
	} catch (error) {
		console.error('提交投诉举报与反馈失败:', error)
		if (!error._isHandled) {
			uni.showToast({
				title: error.message || '提交失败，请稍后重试',
				icon: 'none'
			})
		}
	} finally {
		isSubmitting.value = false
	}
}
</script>

<style lang="scss" scoped>
@import './feedback.scss';
</style>
