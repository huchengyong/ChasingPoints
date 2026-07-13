<template>
	<view v-if="show" class="agreement-sheet-overlay" @click="handleClose">
		<view class="agreement-sheet" :class="{ 'dark-mode': isDarkMode }" @click.stop>
			<view class="sheet-handle"></view>
			<text class="sheet-title">请阅读并同意以下条款</text>
			<text class="sheet-subtitle">
				继续使用前，请先阅读并同意《用户协议》和《隐私政策》。
			</text>
			<view class="sheet-links">
				<text class="sheet-link" @click="$emit('open-user')">用户协议</text>
				<text class="sheet-separator">和</text>
				<text class="sheet-link" @click="$emit('open-privacy')">隐私政策</text>
			</view>
			<button class="sheet-confirm-btn" @click="$emit('agree')">
				同意并继续
			</button>
		</view>
	</view>
</template>

<script setup>
defineProps({
	show: {
		type: Boolean,
		default: false
	},
	isDarkMode: {
		type: Boolean,
		default: false
	}
})

const emit = defineEmits(['close', 'agree', 'open-user', 'open-privacy'])

const handleClose = () => {
	emit('close')
}
</script>

<style lang="scss" scoped>
.agreement-sheet-overlay {
	position: fixed;
	inset: 0;
	z-index: 999;
	display: flex;
	align-items: flex-end;
	justify-content: center;
	background: rgba(9, 7, 4, 0.42);
}

.agreement-sheet {
	width: 100%;
	padding: 24rpx 32rpx calc(32rpx + env(safe-area-inset-bottom));
	border-radius: 32rpx 32rpx 0 0;
	background: rgba(255, 255, 255, 0.96);
	box-shadow: 0 -12rpx 48rpx rgba(35, 28, 11, 0.14);

	&.dark-mode {
		background: rgba(30, 24, 13, 0.98);
		box-shadow: 0 -12rpx 48rpx rgba(0, 0, 0, 0.26);
	}
}

.sheet-handle {
	width: 88rpx;
	height: 8rpx;
	margin: 0 auto 28rpx;
	border-radius: 999rpx;
	background: rgba(154, 140, 103, 0.35);
}

.sheet-title {
	display: block;
	color: #231c0b;
	font-size: 34rpx;
	font-weight: 700;
	text-align: center;
}

.dark-mode .sheet-title {
	color: #fff7e1;
}

.sheet-subtitle {
	display: block;
	margin-top: 16rpx;
	color: #6e6242;
	font-size: 27rpx;
	line-height: 1.6;
	text-align: center;
}

.dark-mode .sheet-subtitle {
	color: #d7c89b;
}

.sheet-links {
	display: flex;
	align-items: center;
	justify-content: center;
	flex-wrap: wrap;
	margin-top: 24rpx;
	font-size: 26rpx;
}

.sheet-link {
	color: #8a6510;
	text-decoration: underline;
}

.dark-mode .sheet-link {
	color: #f7e7a8;
}

.sheet-separator {
	margin: 0 10rpx;
	color: #9a8c67;
}

.sheet-confirm-btn {
	width: 100%;
	height: 96rpx;
	line-height: 96rpx; 
	margin-top: 32rpx;
	border-radius: 999rpx;
	background: linear-gradient(135deg, #f7d86a 0%, #e0ae12 48%, #c69200 100%);
	color: #ffffff;
	font-size: 32rpx;
	font-weight: 700;
	box-shadow: 0 16rpx 32rpx rgba(224, 174, 18, 0.22);

	&::after {
		display: none;
	}
}
</style>
