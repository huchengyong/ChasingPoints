<template>
	<image
		class="gear-image"
		:class="{ 'gear-image--fill': fill }"
		:src="currentSrc"
		mode="aspectFit"
		:style="fill ? {} : { width: `${size}rpx`, height: `${size}rpx` }"
		@error="handleError"
	></image>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { resolveGearImage } from '@/utils/gear-display.js'

// 装备图：优先用目录图，没有或加载失败时回退到按类别的默认图。
// subject 可以是装备（GearItemInfo）或目录型号（GearProductInfo）；图片只来自自有 CDN，不信任服务端返回的任意地址。
const props = defineProps({
	subject: { type: Object, default: null },
	// 方形边长，单位 rpx；fill 为 true 时撑满父容器
	size: { type: Number, default: 240 },
	fill: { type: Boolean, default: false }
})

const failed = ref(false)
const resolved = computed(() => resolveGearImage(props.subject))
const currentSrc = computed(() => (failed.value ? resolved.value.fallback : resolved.value.src))

watch(() => resolved.value.src, () => {
	failed.value = false
})

const handleError = () => {
	failed.value = true
}
</script>

<style lang="scss" scoped>
.gear-image {
	display: block;
	flex-shrink: 0;
}

.gear-image--fill {
	width: 100%;
	height: 100%;
}
</style>
