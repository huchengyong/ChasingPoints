<template>
	<view class="share-page">
		<!-- Canvas (隐藏，用于绘制) -->
		<canvas canvas-id="matchPoster" class="poster-canvas" style="width:750px;height:1320px;position:absolute;left:-9999px;"></canvas>

		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">生成海报中...</text>
		</view>

		<!-- 海报预览 -->
		<view v-else class="poster-preview">
			<image
				v-if="posterPath"
				:src="posterPath"
				mode="widthFix"
				class="poster-image"
			></image>
			<view v-else class="error-state">
				<text>海报生成失败</text>
			</view>
		</view>

		<!-- 操作按钮 -->
		<view class="action-bar">
			<view class="save-btn" @tap="handleSave">
				<uni-icons type="download" size="20" color="#fff"></uni-icons>
				<text class="save-text">保存到相册</text>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getMatchShareData } from '@/api/share.js'
import { generateMatchPoster, savePosterToAlbum } from '@/utils/posterGenerator.js'

const loading = ref(true)
const posterPath = ref('')
const matchId = ref(0)

const generate = async () => {
	loading.value = true
	try {
		const res = await getMatchShareData({ match_id: matchId.value })
		const shareData = normalizeShareData(res)
		if (shareData) {
			posterPath.value = await generateMatchPoster('matchPoster', shareData)
		} else {
			posterPath.value = await generateFallbackPoster()
		}
	} catch (e) {
		console.error('生成海报失败', e)
		posterPath.value = await generateFallbackPoster()
	} finally {
		loading.value = false
	}
}

const normalizeShareData = (payload) => {
	if (!payload) return null
	if (payload.data && typeof payload.data === 'object') return payload.data
	if (typeof payload === 'object') return payload
	return null
}

const generateFallbackPoster = () => {
	return generateMatchPoster('matchPoster', {
		game_type: 3,
		game_type_name: '中式八球',
		player1_name: '我',
		player2_name: '对手',
		my_name: '我',
		opponent_name: '对手',
		my_score: 0,
		opponent_score: 0,
		match_time: new Date().toLocaleDateString(),
		summary_highlights: [],
		summary_stats: []
	})
}

const resolveMatchId = (options = {}) => {
	const optionId = options.match_id || options.id
	if (optionId) {
		return parseInt(optionId, 10) || 0
	}

	const pages = getCurrentPages()
	const currentPage = pages[pages.length - 1]
	const pageOptions = currentPage?.options || {}
	return parseInt(pageOptions.match_id || pageOptions.id || 0, 10) || 0
}

const handleSave = async () => {
	if (!posterPath.value) {
		uni.showToast({ title: '海报未生成', icon: 'none' })
		return
	}
	try {
		await savePosterToAlbum(posterPath.value)
	} catch (e) {
		console.error('保存失败', e)
	}
}

onLoad((options) => {
	matchId.value = resolveMatchId(options)
	// 延迟等待 canvas 就绪
	setTimeout(() => generate(), 500)
})
</script>

<style lang="scss" scoped>
.share-page {
	min-height: 100vh;
	background: #0f172a;
	padding-bottom: 140rpx;
}
.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-height: 60vh;
	.loading-text { font-size: 28rpx; color: #94a3b8; margin-top: 16rpx; }
}
.poster-preview {
	padding: 40rpx 48rpx;
	.poster-image {
		width: 100%;
		border-radius: 16rpx;
		box-shadow: 0 8rpx 32rpx rgba(0,0,0,0.5);
	}
}
.error-state {
	display: flex;
	align-items: center;
	justify-content: center;
	min-height: 50vh;
	font-size: 28rpx;
	color: #94a3b8;
}
.action-bar {
	position: fixed;
	bottom: 0;
	left: 0;
	right: 0;
	padding: 20rpx 48rpx;
	padding-bottom: calc(20rpx + env(safe-area-inset-bottom));
	background: #1e293b;
	.save-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12rpx;
		background: #18b05b;
		border-radius: 12rpx;
		padding: 24rpx;
		.save-text { font-size: 30rpx; color: #fff; font-weight: 500; }
	}
}
</style>
