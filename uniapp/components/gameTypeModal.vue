<template>
	<!-- 选择比赛类型弹框 -->
	<view v-if="visible" class="modal-overlay" :class="{ 'dark-mode': isDarkMode }" @click="handleCancel">
		<view class="modal-container" @click.stop>
			<!-- 弹框头部 -->
			<view class="modal-header">
				<text class="modal-title">选择比赛类型</text>
				<text class="modal-subtitle">请选择本次对局的比赛类型</text>
			</view>

			<!-- 比赛类型列表 -->
			<view class="game-type-list">
				<button 
					v-for="item in gameTypes" 
					:key="item.value"
					:class="['game-type-option', { selected: selectedType === item.value }]"
					@click="selectType(item.value)"
				>
					<text>{{ item.label }}</text>
					<text v-if="defaultType === item.value" class="default-badge">默认</text>
				</button>
			</view>

			<view
				v-if="selectedType && selectedType !== defaultType"
				class="default-choice"
				:class="{ active: saveAsDefault }"
				@click="saveAsDefault = !saveAsDefault"
			>
				<view class="default-choice-box">
					<uni-icons v-if="saveAsDefault" type="checkmarkempty" size="14" color="#ffffff"></uni-icons>
				</view>
				<text>将当前选择设为默认对局类型</text>
			</view>

			<!-- 按钮组 -->
			<view class="modal-buttons">
				<button class="modal-btn cancel" @click="handleCancel">
					<text>取消</text>
				</button>
				<button 
					class="modal-btn confirm" 
					:disabled="!selectedType"
					@click="handleConfirm"
				>
					<text>确认</text>
				</button>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useThemeStore } from '@/store/theme.js'
import { GAME_TYPE_OPTIONS } from '@/utils/game-types.js'

// ========== Props ==========
const props = defineProps({
	visible: {
		type: Boolean,
		default: false
	},
	defaultType: {
		type: Number,
		default: 0
	}
})

// ========== Emits ==========
const emit = defineEmits(['update:visible', 'confirm', 'cancel'])

// ========== 状态管理 ==========
const themeStore = useThemeStore()

// ========== 响应式数据 ==========
const selectedType = ref(null)
const saveAsDefault = ref(false)
const isDarkMode = computed(() => themeStore.isDarkMode)

// 比赛类型列表
const gameTypes = GAME_TYPE_OPTIONS

// ========== 监听器 ==========
// 弹框关闭时重置选择
watch(() => props.visible, (newVal) => {
	selectedType.value = newVal && gameTypes.some((item) => item.value === props.defaultType)
		? props.defaultType
		: null
	saveAsDefault.value = false
})

// ========== 方法 ==========

/**
 * 选择比赛类型
 */
const selectType = (type) => {
	selectedType.value = type
	saveAsDefault.value = false
}

/**
 * 取消
 */
const handleCancel = () => {
	emit('update:visible', false)
	emit('cancel')
}

/**
 * 确认
 */
const handleConfirm = () => {
	if (!selectedType.value) return
	
	emit('confirm', {
		gameType: selectedType.value,
		setAsDefault: saveAsDefault.value
	})
	emit('update:visible', false)
}
</script>

<style lang="scss" scoped>
// 主色调
$primary-color: #e0ae12;

// 浅色模式变量
$light-bg: #ffffff;
$light-container-bg: #f5f5f5;
$light-text-primary: #1a1a1a;
$light-text-secondary: #71717a;
$light-option-bg: #ffffff;
$light-cancel-bg: #e5e5e5;

// 深色模式变量
$dark-container-bg: #18181b;
$dark-text-primary: #ffffff;
$dark-text-secondary: #a1a1aa;
$dark-option-bg: #27272a;
$dark-cancel-bg: #27272a;

// ========== 选择比赛类型弹框 ==========
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	z-index: 100;
	background-color: rgba(0, 0, 0, 0.5);
	backdrop-filter: blur(8rpx);
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 48rpx;

	// 浅色模式样式
	.modal-container {
		width: 100%;
		max-width: 640rpx;
		display: flex;
		flex-direction: column;
		gap: 48rpx;
		padding: 48rpx;
		background-color: $light-container-bg;
		border-radius: 32rpx;
		box-shadow: 0 16rpx 64rpx rgba(0, 0, 0, 0.2);
	}

	// 弹框头部
	.modal-header {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;

		.modal-title {
			font-size: 40rpx;
			font-weight: 700;
			color: $light-text-primary;
			margin-bottom: 8rpx;
		}

		.modal-subtitle {
			font-size: 28rpx;
			color: $light-text-secondary;
		}
	}

	// 比赛类型列表
	.game-type-list {
		display: flex;
		flex-direction: column;
		gap: 24rpx;
	}

	// 比赛类型选项
	.game-type-option {
		width: 100%;
		height: 88rpx;
		line-height: 88rpx;
		margin: 0;
		padding: 0 32rpx;
		background-color: $light-option-bg;
		border-radius: 16rpx;
		border: 4rpx solid transparent;
		text-align: left;
		transition: all 0.2s ease;

		&::after {
			display: none;
		}

		&.selected {
			border-color: $primary-color;
			background-color: rgba($primary-color, 0.1);
		}

		text {
			font-size: 32rpx;
			font-weight: 500;
			color: $light-text-primary;
		}

		.default-badge {
			float: right;
			font-size: 22rpx;
			color: #8a6510;
		}
	}

	.default-choice {
		display: flex;
		align-items: center;
		gap: 16rpx;
		padding: 4rpx 8rpx;
		color: $light-text-secondary;
		font-size: 24rpx;

		.default-choice-box {
			width: 36rpx;
			height: 36rpx;
			border-radius: 10rpx;
			border: 2rpx solid rgba(113, 113, 122, 0.45);
			display: flex;
			align-items: center;
			justify-content: center;
			box-sizing: border-box;
		}

		&.active .default-choice-box {
			background: $primary-color;
			border-color: $primary-color;
		}
	}

	// 弹框按钮组
	.modal-buttons {
		display: flex;
		gap: 24rpx;

		.modal-btn {
			flex: 1;
			height: 96rpx;
			line-height: 96rpx;
			display: flex;
			align-items: center;
			justify-content: center;
			border-radius: 24rpx;
			font-size: 32rpx;
			font-weight: 700;
			transition: all 0.2s ease;

			&::after {
				display: none;
			}

			&.cancel {
				background-color: $light-cancel-bg;

				text {
					color: $light-text-primary;
				}
			}

				&.confirm {
					background-color: $primary-color;

					text {
						color: #ffffff;
					}

				&[disabled] {
					opacity: 0.5;
				}
			}
		}
	}

	// 深色模式样式
	&.dark-mode {
		.modal-container {
			background-color: $dark-container-bg;
			box-shadow: 0 16rpx 64rpx rgba(0, 0, 0, 0.4);
		}

		.modal-header {
			.modal-title {
				color: $dark-text-primary;
			}

			.modal-subtitle {
				color: $dark-text-secondary;
			}
		}

		.game-type-option {
			background-color: $dark-option-bg;

			text {
				color: $dark-text-primary;
			}
		}

		.default-choice {
			color: $dark-text-secondary;
		}

		.modal-buttons {
			.modal-btn.cancel {
				background-color: $dark-cancel-bg;

				text {
					color: $dark-text-primary;
				}
			}
		}
	}
}
</style>
