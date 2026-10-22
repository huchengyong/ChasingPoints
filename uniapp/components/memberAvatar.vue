<template>
	<view class="member-avatar" :style="{ width: `${metrics.boxSize}rpx`, height: `${metrics.boxSize}rpx` }">
		<view class="member-avatar-core" :style="{ width: `${metrics.avatarSize}rpx`, height: `${metrics.avatarSize}rpx` }">
			<image class="member-avatar-image" :src="avatarSrc" :style="ringStyle" mode="aspectFill"></image>
			<image
				v-if="identity.frameUrl"
				class="member-avatar-frame"
				:src="identity.frameUrl"
				mode="aspectFit"
				:style="{ width: `${metrics.frameSize}rpx`, height: `${metrics.frameSize}rpx` }"
			></image>
			<view
				v-if="showBadge && identity.showBadge"
				class="member-avatar-badge"
				:class="[`member-avatar-badge--${identity.tier}`, { 'member-avatar-badge--corner': badgeCorner }]"
				:style="{
					...cornerOffsetStyle,
					height: `${metrics.badgeHeight}rpx`,
					lineHeight: `${metrics.badgeHeight}rpx`,
					padding: `0 ${metrics.badgePaddingX}rpx`,
					fontSize: `${metrics.badgeFontSize}rpx`
				}"
			>
				<text class="member-avatar-badge-text">{{ identity.badgeText }}</text>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed } from 'vue'
import { resolveMemberAvatarMetrics, resolveMemberIdentity } from '@/utils/member-identity.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

// 会员身份头像：头像 + 段位头像框 + VIP 标识。会员字段缺失按非会员处理，不影响头像本身。
const props = defineProps({
	avatar: { type: String, default: '' },
	userId: { type: [Number, String], default: 0 },
	// 有效会员的成长等级 1–5，0 或缺失表示非会员
	vipLevel: { type: [Number, String], default: 0 },
	// 当前球种的段位等级 1–6，用于选择头像框
	rankLevel: { type: [Number, String], default: 0 },
	// 头像直径，单位 rpx
	size: { type: Number, default: 96 },
	// 按头像框预留布局盒，会员与非会员占位一致；四周有空白的位置保持 false，框图向外溢出
	reserveFrame: { type: Boolean, default: false },
	// size 视为含头像框的总占位：有头像框时头像缩小到框内，框图不再向外溢出
	fitFrame: { type: Boolean, default: false },
	// 没有头像框时给头像加一圈色环（框本身就是会员的环，不再叠加）
	ringColor: { type: String, default: '' },
	ringWidth: { type: Number, default: 8 },
	showBadge: { type: Boolean, default: true },
	// 标识放在右下角而不是底部居中，用于底部已有其他徽标的位置
	badgeCorner: { type: Boolean, default: false }
})

const identity = computed(() => resolveMemberIdentity({ vipLevel: props.vipLevel, rankLevel: props.rankLevel }))
const metrics = computed(() => resolveMemberAvatarMetrics(props.size, {
	reserveFrame: props.reserveFrame,
	fitFrame: props.fitFrame,
	hasFrame: Boolean(identity.value.frameUrl)
}))
// 右下角标识贴着布局盒的右缘，而不是缩进后的头像右缘。
const cornerOffsetStyle = computed(() => (
	props.badgeCorner ? { right: `${-(Math.round((metrics.value.boxSize - metrics.value.avatarSize) / 2) + 12)}rpx` } : {}
))
const ringStyle = computed(() => (
	props.ringColor && !identity.value.frameUrl
		? { border: `${props.ringWidth}rpx solid ${props.ringColor}`, boxSizing: 'border-box' }
		: {}
))
const avatarSrc = computed(() => resolveAvatarUrl(props.avatar, props.userId))
</script>

<style lang="scss" scoped>
.member-avatar {
	position: relative;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
}

.member-avatar-core {
	position: relative;
}

.member-avatar-image {
	width: 100%;
	height: 100%;
	border-radius: 50%;
	background: var(--ui-surface-subtle, #FAF8F2);
}

.member-avatar-frame {
	position: absolute;
	top: 50%;
	left: 50%;
	transform: translate(-50%, -50%);
	z-index: 2;
	pointer-events: none;
}

.member-avatar-badge {
	position: absolute;
	bottom: -8rpx;
	left: 50%;
	transform: translateX(-50%);
	z-index: 3;
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 999rpx;
	font-weight: 800;
	white-space: nowrap;
	box-sizing: border-box;
	pointer-events: none;
}

.member-avatar-badge--corner {
	left: auto;
	right: -12rpx;
	bottom: -4rpx;
	transform: none;
}

.member-avatar-badge-text {
	font-size: inherit;
	line-height: inherit;
	font-weight: inherit;
	color: inherit;
}

// 标准：金色浅底胶囊。叠在卡片底色上保持不透明，避免压在头像上读不清。
.member-avatar-badge--standard {
	color: var(--ui-brand-strong, #C69200);
	border: 1rpx solid rgba(224, 174, 18, 0.55);
	background:
		linear-gradient(var(--ui-brand-tint, rgba(224, 174, 18, 0.14)), var(--ui-brand-tint, rgba(224, 174, 18, 0.14))),
		var(--ui-surface-card, #FFFFFF);
}

// 进阶：金色实底胶囊，白字（与品牌主按钮一致）。
.member-avatar-badge--advanced {
	color: #FFFFFF;
	border: 1rpx solid rgba(187, 132, 0, 0.6);
	background: linear-gradient(135deg, var(--ui-brand-gradient-start, #D9A617), var(--ui-brand-gradient-end, #BB8400));
}

// 尊享：暖黑金底配金字，亮暗主题下都保持同一外观。
.member-avatar-badge--premium {
	color: #F0C542;
	border: 1rpx solid #C69200;
	background: linear-gradient(135deg, #3C2B08, #6B4A07);
}
</style>
