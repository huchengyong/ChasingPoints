<template>
	<view class="titles-page">
		<!-- 加载中 -->
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<!-- 称号列表 -->
		<view v-else-if="titleList.length > 0" class="title-list">
			<view
				v-for="item in titleList"
				:key="item.id"
				class="title-item"
				:class="{ equipped: item.equipped }"
			>
				<view class="title-left">
					<text class="title-name">{{ item.title_name }}</text>
					<view class="title-source">
						<text class="source-badge" :class="getSourceClass(item.source)">{{ item.source }}</text>
					</view>
				</view>
				<button
					class="equip-btn"
					:class="{ 'equipped-btn': item.equipped }"
					@tap="handleEquip(item)"
					:disabled="equipLoading"
				>
					<text>{{ item.equipped ? '卸下' : '装备' }}</text>
				</button>
			</view>
		</view>

		<!-- 空状态 -->
		<view v-else class="empty-state">
			<text class="empty-icon">🎖️</text>
			<text class="empty-text">暂无称号</text>
			<text class="empty-hint">解锁成就或参加赛事可获得称号</text>
		</view>
	</view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getUserTitles, equipTitle } from '@/api/achievement.js'

const loading = ref(true)
const equipLoading = ref(false)
const titleList = ref([])

const getSourceClass = (source) => {
	const map = { '成就': 'achievement', '赛季': 'season', '赛事': 'tournament' }
	return map[source] || 'default'
}

const handleEquip = async (item) => {
	equipLoading.value = true
	try {
		await equipTitle({ title_id: item.id, equip: !item.equipped })
		// 刷新列表
		await loadTitles()
		uni.showToast({
			title: item.equipped ? '已卸下称号' : '已装备称号',
			icon: 'success'
		})
	} catch (e) {
		console.error('操作称号失败:', e)
		uni.showToast({ title: '操作失败', icon: 'none' })
	} finally {
		equipLoading.value = false
	}
}

const loadTitles = async () => {
	try {
		const res = await getUserTitles()
		titleList.value = res.list || res || []
	} catch (e) {
		console.error('加载称号列表失败:', e)
	} finally {
		loading.value = false
	}
}

onLoad(() => {
	loadTitles()
})
</script>

<style lang="scss" scoped>
.titles-page {
	min-height: 100vh;
	background: #f1f5f9;
	padding: 24rpx;
}

.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 300rpx;
	.loading-text {
		margin-top: 16rpx;
		font-size: 28rpx;
		color: #94a3b8;
	}
}

.title-list {
	.title-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: #fff;
		border-radius: 16rpx;
		padding: 28rpx 32rpx;
		margin-bottom: 16rpx;

		&.equipped {
			border: 2rpx solid #18b05b;
			background: #f0fdf4;
		}

		.title-left {
			flex: 1;
			.title-name {
				font-size: 30rpx;
				font-weight: 600;
				color: #1e293b;
				display: block;
				margin-bottom: 8rpx;
			}
			.title-source {
				.source-badge {
					font-size: 22rpx;
					padding: 4rpx 16rpx;
					border-radius: 16rpx;

					&.achievement {
						background: #fef3c7;
						color: #d97706;
					}
					&.season {
						background: #dcfce7;
						color: #15803d;
					}
					&.tournament {
						background: #dcfce7;
						color: #16a34a;
					}
					&.default {
						background: #f1f5f9;
						color: #64748b;
					}
				}
			}
		}

		.equip-btn {
			min-width: 120rpx;
			height: 60rpx;
			line-height: 60rpx;
			text-align: center;
			border-radius: 30rpx;
			font-size: 26rpx;
			background: #18b05b;
			color: #fff;
			border: none;
			padding: 0 24rpx;
			margin: 0;

			&.equipped-btn {
				background: #fff;
				color: #64748b;
				border: 2rpx solid #e2e8f0;
			}

			&::after {
				border: none;
			}
		}
	}
}

.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding-top: 260rpx;

	.empty-icon {
		font-size: 80rpx;
		margin-bottom: 24rpx;
	}
	.empty-text {
		font-size: 30rpx;
		color: #64748b;
		margin-bottom: 8rpx;
	}
	.empty-hint {
		font-size: 24rpx;
		color: #94a3b8;
	}
}
</style>
