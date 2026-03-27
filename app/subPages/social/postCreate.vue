<template>
	<view class="post-create-page" :class="{ 'dark-mode': isDarkMode }">
		<!-- 文本输入 -->
		<view class="input-section">
			<textarea
				class="content-input"
				v-model="content"
				placeholder="分享你的台球时刻..."
				:maxlength="2000"
				auto-height
			/>
			<text class="char-count">{{ content.length }}/2000</text>
		</view>

		<!-- 图片上传 -->
		<view class="image-section">
			<view class="image-grid">
				<view
					v-for="(img, idx) in imageList"
					:key="idx"
					class="image-item"
				>
					<image :src="img" mode="aspectFill" class="preview-img" />
					<view class="remove-btn" @tap="removeImage(idx)">
						<uni-icons type="clear" size="20" color="#fff"></uni-icons>
					</view>
				</view>
				<view v-if="imageList.length < 9" class="image-add" @tap="chooseImage">
					<uni-icons type="plusempty" size="36" color="#cbd5e1"></uni-icons>
					<text class="add-text">{{ imageList.length }}/9</text>
				</view>
			</view>
		</view>

		<!-- 动态类型 -->
		<view class="type-section">
			<text class="section-label">类型</text>
			<view class="type-tags">
				<view
					class="type-tag"
					:class="{ active: postType === 3 }"
					@tap="postType = 3"
				>
					<text>日常</text>
				</view>
				<view
					class="type-tag"
					:class="{ active: postType === 1 }"
					@tap="postType = 1"
				>
					<text>🏆 战绩</text>
				</view>
				<view
					class="type-tag"
					:class="{ active: postType === 2 }"
					@tap="postType = 2"
				>
					<text>📍 打卡</text>
				</view>
			</view>
		</view>

		<!-- 发布按钮 -->
		<view class="publish-section">
			<button
				class="publish-btn"
				:disabled="!canPublish || publishing"
				@tap="handlePublish"
			>
				<text>{{ publishing ? '发布中...' : '发布动态' }}</text>
			</button>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { createPost } from '@/api/social.js'
import { useThemeStore } from '@/store/theme.js'
import { buildPostReviewSuccessCopy } from '@/utils/social-review.js'

const content = ref('')
const imageList = ref([])
const postType = ref(3)
const publishing = ref(false)
const themeStore = useThemeStore()

const isDarkMode = computed(() => themeStore.isDarkMode)

const canPublish = computed(() => {
	return content.value.trim().length > 0
})

onLoad((options) => {
	if (options.content) {
		content.value = decodeURIComponent(options.content)
	}
	if (options.post_type) {
		postType.value = Number(options.post_type) || 3
	}
})

onShow(() => {
	themeStore.syncTheme()
	themeStore.applyNavigationBarTheme()
})

const chooseImage = () => {
	const remaining = 9 - imageList.value.length
	uni.chooseImage({
		count: remaining,
		sizeType: ['compressed'],
		sourceType: ['album', 'camera'],
		success: (res) => {
			imageList.value = [...imageList.value, ...res.tempFilePaths].slice(0, 9)
		}
	})
}

const removeImage = (idx) => {
	imageList.value.splice(idx, 1)
}

const handlePublish = async () => {
	if (!canPublish.value || publishing.value) return

	publishing.value = true
	try {
		const data = {
			content: content.value.trim(),
			post_type: postType.value,
			images: imageList.value
		}

		const res = await createPost(data)
		if (res.success) {
			const successCopy = buildPostReviewSuccessCopy()
			uni.showToast({ title: successCopy.toast, icon: 'success' })
			setTimeout(() => {
				uni.redirectTo({ url: `/subPages/social/myPosts?hint=${encodeURIComponent(successCopy.hint)}` })
			}, 700)
		} else {
			uni.showToast({ title: res.message || '发布失败', icon: 'none' })
		}
	} catch (e) {
		console.error('发布动态失败:', e)
		uni.showToast({ title: '发布失败', icon: 'none' })
	} finally {
		publishing.value = false
	}
}
</script>

<style lang="scss" scoped>
.post-create-page {
	min-height: 100vh;
	background: #f1f5f9;
	padding: 20rpx 24rpx;

	&.dark-mode {
		background: #141109;

		.input-section,
		.image-section,
		.type-section {
			background: #1e180d;
		}

		.content-input,
		.type-tag.active {
			color: #fff7e1;
		}

		.char-count,
		.section-label,
		.add-text,
		.type-tag {
			color: #d7c89b;
		}

		.image-add,
		.type-tag {
			background: #2b2316;
			border-color: #3a2e16;
		}
	}
}

.input-section {
	background: #fff;
	border-radius: 20rpx;
	padding: 24rpx;
	margin-bottom: 20rpx;

	.content-input {
		width: 100%;
		min-height: 200rpx;
		font-size: 30rpx;
		color: #1e293b;
		line-height: 1.6;
	}

	.char-count {
		text-align: right;
		font-size: 22rpx;
		color: #94a3b8;
		margin-top: 12rpx;
	}
}

.image-section {
	background: #fff;
	border-radius: 20rpx;
	padding: 24rpx;
	margin-bottom: 20rpx;

	.image-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 12rpx;

		.image-item {
			position: relative;
			width: calc(33.33% - 8rpx);
			aspect-ratio: 1;

			.preview-img {
				width: 100%;
				height: 100%;
				border-radius: 12rpx;
			}

			.remove-btn {
				position: absolute;
				top: -10rpx;
				right: -10rpx;
				width: 40rpx;
				height: 40rpx;
				background: rgba(0, 0, 0, 0.5);
				border-radius: 50%;
				display: flex;
				align-items: center;
				justify-content: center;
			}
		}

		.image-add {
			width: calc(33.33% - 8rpx);
			aspect-ratio: 1;
			background: #f8fafc;
			border: 2rpx dashed #cbd5e1;
			border-radius: 12rpx;
			display: flex;
			flex-direction: column;
			align-items: center;
			justify-content: center;
			gap: 8rpx;

			.add-text {
				font-size: 22rpx;
				color: #94a3b8;
			}
		}
	}
}

.type-section {
	background: #fff;
	border-radius: 20rpx;
	padding: 24rpx;
	margin-bottom: 40rpx;

	.section-label {
		font-size: 26rpx;
		color: #64748b;
		margin-bottom: 16rpx;
	}

	.type-tags {
		display: flex;
		gap: 16rpx;

		.type-tag {
			padding: 12rpx 28rpx;
			border-radius: 32rpx;
			background: #f1f5f9;
			font-size: 26rpx;
			color: #64748b;

			&.active {
				background: #E0AE12;
				color: #1f2937;
			}
		}
	}
}

.publish-section {
	.publish-btn {
		width: 100%;
		background: linear-gradient(135deg, #E0AE12 0%, #F59E0B 100%);
		color: #1f2937;
		border-radius: 40rpx;
		height: 88rpx;
		line-height: 88rpx;
		font-size: 32rpx;
		font-weight: 500;
		border: none;

		&[disabled] {
			background: #cbd5e1;
			color: #94a3b8;
		}

		&::after {
			display: none;
		}
	}
}
</style>
