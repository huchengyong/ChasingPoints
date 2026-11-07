<template>
	<view class="gear-card" :class="{ 'gear-card--retired': retired }">
		<view class="gear-card-image">
			<gearImage :subject="item" fill />
			<text v-if="retired" class="gear-card-flag">已退役</text>
		</view>
		<view class="gear-card-body">
			<text class="gear-card-title">{{ title }}</text>
			<text v-if="subtitle" class="gear-card-subtitle">{{ subtitle }}</text>
			<view v-if="badges.length" class="gear-card-badges">
				<text v-for="badge in badges" :key="badge" class="gear-card-badge">{{ badge }}</text>
			</view>
			<text v-if="partsLine" class="gear-card-parts">{{ partsLine }}</text>
		</view>
	</view>
</template>

<script setup>
import { computed } from 'vue'
import gearImage from '@/components/gearImage.vue'
import { GEAR_ITEM_STATUS, gearItemSubtitle, gearItemTitle, gearProductTitle } from '@/utils/gear-display.js'

// 装备卡片：图、名称、副标题、主力标签，以及球杆当前的皮头与巧克。点击由使用方在外层处理。
const props = defineProps({
	item: { type: Object, required: true },
	// 这件装备在哪些球种里是主力，例如「中式八球 主力」
	badges: { type: Array, default: () => [] }
})

const retired = computed(() => props.item.status === GEAR_ITEM_STATUS.RETIRED)
const title = computed(() => gearItemTitle(props.item))
const subtitle = computed(() => gearItemSubtitle(props.item))
const partsLine = computed(() => {
	const parts = []
	if (props.item.tip_product) parts.push(`皮头 ${gearProductTitle(props.item.tip_product)}`)
	if (props.item.chalk_product) parts.push(`巧克 ${gearProductTitle(props.item.chalk_product)}`)
	return parts.join(' · ')
})
</script>

<style lang="scss" scoped>
.gear-card {
	display: flex;
	flex-direction: column;
	// 撑满所在的网格格子，同一行的卡片等高
	height: 100%;
	overflow: hidden;
	border-radius: var(--ui-radius-lg, 24rpx);
	background: var(--ui-surface-card, #FFFFFF);
	border: 1rpx solid var(--ui-border-default, #E9E2CF);
	box-sizing: border-box;
}

.gear-card--retired {
	opacity: 0.72;
}

// 图片区撑满卡片宽度：两列布局下不小于 160px，球杆细长，放小了只剩一条线。
.gear-card-image {
	position: relative;
	width: 100%;
	height: 340rpx;
	background: var(--ui-surface-subtle, #FAF8F2);
}

.gear-card-flag {
	position: absolute;
	top: 16rpx;
	left: 16rpx;
	padding: 4rpx 14rpx;
	border-radius: var(--ui-radius-pill, 999rpx);
	background: var(--ui-surface-card, #FFFFFF);
	border: 1rpx solid var(--ui-border-default, #E9E2CF);
	font-size: 20rpx;
	color: var(--ui-text-secondary, #6E6242);
}

.gear-card-body {
	display: flex;
	flex-direction: column;
	padding: 20rpx 20rpx 24rpx;
}

.gear-card-title {
	font-size: 28rpx;
	font-weight: 600;
	line-height: 1.35;
	color: var(--ui-text-primary, #231C0B);
	overflow-wrap: break-word;
}

.gear-card-subtitle {
	margin-top: 6rpx;
	font-size: 22rpx;
	color: var(--ui-text-muted, #9A8C67);
}

.gear-card-badges {
	display: flex;
	flex-wrap: wrap;
	gap: 8rpx;
	margin-top: 12rpx;
}

.gear-card-badge {
	padding: 4rpx 14rpx;
	border-radius: var(--ui-radius-pill, 999rpx);
	background: var(--ui-brand-tint, rgba(224, 174, 18, 0.14));
	font-size: 20rpx;
	color: var(--ui-brand-strong, #C69200);
}

.gear-card-parts {
	margin-top: 12rpx;
	font-size: 22rpx;
	line-height: 1.4;
	color: var(--ui-text-secondary, #6E6242);
}
</style>
