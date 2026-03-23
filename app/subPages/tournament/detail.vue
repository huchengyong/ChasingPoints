<template>
	<view class="detail-page">
		<view v-if="loading" class="loading-state">
			<uni-icons type="spinner-cycle" size="36" color="#18b05b"></uni-icons>
			<text class="loading-text">加载中...</text>
		</view>

		<view v-else-if="tournament" class="detail-content">
			<view class="hero-card">
				<view class="hero-topline">
					<view class="hero-chip">
						<text>{{ getGameTypeLabel(tournament.game_type, '未知球种') }}</text>
					</view>
					<view class="status-tag" :class="'status-' + tournament.status">
						<text>{{ statusMap[tournament.status] }}</text>
					</view>
				</view>
				<text class="hero-title">{{ tournament.name }}</text>
				<text class="hero-desc">{{ tournament.description || '赛事创建者暂未补充详细说明，可先查看规则和参赛情况。' }}</text>
				<view class="hero-stats">
					<view class="hero-stat">
						<text class="hero-stat-label">赛制</text>
						<text class="hero-stat-value">{{ formatMap[tournament.format] || '未知' }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">参赛人数</text>
						<text class="hero-stat-value">{{ tournament.current_players }}/{{ tournament.max_players }}</text>
					</view>
					<view class="hero-stat">
						<text class="hero-stat-label">开始时间</text>
						<text class="hero-stat-value">{{ formatDisplayTime(tournament.start_time) || '待定' }}</text>
					</view>
				</view>
			</view>

			<view class="info-card">
				<view class="section-head">
					<text class="section-title">赛事信息</text>
					<text class="section-tip">{{ formatRelativeTime(tournament.start_time) || '待定时间' }}</text>
				</view>
				<view class="info-grid">
					<view class="grid-item">
						<text class="grid-label">球种</text>
						<text class="grid-value">{{ getGameTypeLabel(tournament.game_type, '未知') }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">赛制</text>
						<text class="grid-value">{{ formatMap[tournament.format] || '未知' }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">人数上限</text>
						<text class="grid-value">{{ tournament.max_players }}</text>
					</view>
					<view class="grid-item">
						<text class="grid-label">当前报名</text>
						<text class="grid-value">{{ tournament.current_players }}</text>
					</view>
					<view class="grid-item" v-if="tournament.city">
						<text class="grid-label">举办城市</text>
						<text class="grid-value">{{ tournament.city }}</text>
					</view>
					<view class="grid-item" v-if="tournament.venue_name">
						<text class="grid-label">场馆</text>
						<text class="grid-value">{{ tournament.venue_name }}</text>
					</view>
				</view>
				<view class="progress-panel">
					<view class="progress-head">
						<text class="progress-label">报名进度</text>
						<text class="progress-value">{{ tournament.current_players }}/{{ tournament.max_players }}</text>
					</view>
					<view class="progress-track">
						<view class="progress-fill" :style="{ width: `${progressPercent}%` }"></view>
					</view>
				</view>
				<view class="info-time" v-if="tournament.start_time">
					<uni-icons type="calendar" size="16" color="#64748b"></uni-icons>
					<text class="time-value">{{ tournament.start_time }}</text>
				</view>
			</view>

			<view class="action-bar" v-if="tournament.status <= 1">
				<view v-if="tournament.status === 1" class="action-btn bracket-btn" @tap="goBracket">
					<text>查看对阵图</text>
				</view>
				<view v-if="tournament.status === 0 && !isJoined" class="action-btn join-btn" @tap="onJoin">
					<text>报名参加</text>
				</view>
				<view v-if="tournament.status === 0 && isJoined" class="action-btn leave-btn" @tap="onLeave">
					<text>取消报名</text>
				</view>
			</view>

			<view class="section-card">
				<view class="section-head">
					<text class="section-title">参赛者（{{ participants.length }}）</text>
					<text class="section-tip">{{ participants.length > 0 ? '按当前报名顺序展示' : '还没有参赛者' }}</text>
				</view>
			</view>
			<view class="participant-list">
				<view v-for="p in participants" :key="p.user_id" class="participant-item">
					<image class="p-avatar" :src="p.avatar || '/static/images/default-avatar.png'" mode="aspectFill" />
					<view class="p-info">
						<text class="p-name">{{ p.nickname || '球友' }}</text>
						<text class="p-seed" v-if="p.seed > 0">种子 #{{ p.seed }}</text>
					</view>
					<text class="p-rank" v-if="p.final_rank > 0">第{{ p.final_rank }}名</text>
				</view>
				<view v-if="participants.length === 0" class="empty-participants">
					<text>暂无参赛者</text>
				</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { getTournamentDetail, joinTournament, leaveTournament } from '@/api/tournament.js'
import { formatRelativeTime } from '@/utils/format.js'
import { getGameTypeLabel } from '@/utils/game-types.js'

const formatMap = { 1: '单败淘汰', 2: '双败淘汰', 3: '循环赛' }
const statusMap = { 0: '报名中', 1: '进行中', 2: '已结束', 3: '已取消' }

const tournament = ref(null)
const participants = ref([])
const loading = ref(true)
const tournamentId = ref(0)

const isJoined = computed(() => {
	const userInfo = uni.getStorageSync('userInfo')
	if (!userInfo) return false
	return participants.value.some(p => p.user_id === userInfo.id)
})
const progressPercent = computed(() => {
	if (!tournament.value || !tournament.value.max_players) return 0
	return Math.min(100, Math.round(((tournament.value.current_players || 0) / tournament.value.max_players) * 100))
})

const fetchDetail = async () => {
	loading.value = true
	try {
		const res = await getTournamentDetail({ tournament_id: tournamentId.value })
		if (res.success) {
			tournament.value = res.tournament
			participants.value = res.participants || []
		}
	} catch (e) {
		console.error('获取赛事详情失败', e)
	} finally {
		loading.value = false
	}
}

const onJoin = async () => {
	try {
		const res = await joinTournament({ tournament_id: tournamentId.value })
		if (res.success) {
			uni.showToast({ title: '报名成功', icon: 'success' })
			fetchDetail()
		} else {
			uni.showToast({ title: res.message || '报名失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '报名失败', icon: 'none' })
	}
}

const onLeave = async () => {
	try {
		const res = await leaveTournament({ tournament_id: tournamentId.value })
		if (res.success) {
			uni.showToast({ title: '已取消报名', icon: 'success' })
			fetchDetail()
		} else {
			uni.showToast({ title: res.message || '操作失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '操作失败', icon: 'none' })
	}
}

const goBracket = () => {
	uni.navigateTo({ url: '/subPages/tournament/bracket?id=' + tournamentId.value })
}
const formatDisplayTime = (value) => {
	if (!value) return ''
	const relative = formatRelativeTime(value)
	return relative === '刚刚' ? '今天' : relative
}

onMounted(() => {
	const pages = getCurrentPages()
	const currentPage = pages[pages.length - 1]
	tournamentId.value = parseInt(currentPage.options.id || 0)
	if (tournamentId.value > 0) fetchDetail()
})
</script>

<style lang="scss" scoped>
@import './detail.scss';
</style>
