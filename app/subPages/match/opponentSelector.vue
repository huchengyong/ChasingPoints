<template>
	<view class="opponent-selector" :class="{ 'dark-mode': isDarkMode }">
		<view class="selector-intro">
			<text class="selector-title">选择一位球友</text>
			<text class="selector-copy">好友优先展示，最近对手按最近对局排序。点头像直接进入约球发起页。</text>
		</view>

		<view v-if="loading" class="selector-state">
			<uni-icons type="spinner-cycle" size="38" color="#E0AE12"></uni-icons>
			<text>加载球友中...</text>
		</view>

		<view v-else-if="opponents.length === 0" class="selector-state">
			<text class="selector-empty-title">暂无可邀请的球友</text>
			<text>先添加好友或完成一场平台注册对局后再试。</text>
		</view>

		<view v-else class="opponent-list">
			<view
				v-for="item in opponents"
				:key="item.user_id"
				class="opponent-card"
				@tap="chooseOpponent(item)"
			>
				<image class="opponent-avatar" :src="resolveAvatarUrl(item.avatar, item.user_id)" mode="aspectFill" />
				<view class="opponent-copy">
					<text class="opponent-name">{{ item.nickname }}</text>
					<text class="opponent-source">{{ item.source === 'friend' ? '好友' : '最近对手' }}</text>
				</view>
				<uni-icons type="right" size="20" color="#9A8C67"></uni-icons>
			</view>
		</view>
	</view>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getOpponentCandidates } from '@/api/opponent.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const loading = ref(true)
const opponents = ref([])
const fromContext = ref('')
const gameTypePrefill = ref(0)
// 发起页「更换」进入时带回的表单状态；其他入口为空数组。
const formPrefill = ref([])

const loadOpponents = async () => {
	loading.value = true
	try {
		const response = await getOpponentCandidates({ limit: 30 })
		opponents.value = response?.success
			? (response.list || []).map((item) => ({
				...item,
				nickname: item.name || item.nickname || ''
			}))
			: []
	} catch (error) {
		console.error('加载PK对手失败', error)
		uni.showToast({ title: '加载球友失败', icon: 'none' })
	} finally {
		loading.value = false
	}
}

const chooseOpponent = (item) => {
	if (!item?.user_id) return
	const query = [
		`opponent_id=${item.user_id}`,
		`opponent_name=${encodeURIComponent(item.nickname || '')}`,
		`opponent_avatar=${encodeURIComponent(item.avatar || '')}`
	]
	if (gameTypePrefill.value > 0) query.push(`game_type=${gameTypePrefill.value}`)
	query.push(...formPrefill.value)
	uni.redirectTo({ url: `/subPages/match/challengeCompose?${query.join('&')}` })
}

onLoad((query = {}) => {
	if (query.from) fromContext.value = String(query.from)
	if (Number(query.game_type) > 0) gameTypePrefill.value = Number(query.game_type)
	if (query.from === 'challenge' && query.start_hour !== undefined) {
		formPrefill.value = [
			`match_mode=${query.match_mode || 'ranked'}`,
			`start_hour=${Number(query.start_hour) || 0}`,
			`end_hour=${Number(query.end_hour) || 1}`
		]
		if (query.scheduled_date) formPrefill.value.push(`scheduled_date=${query.scheduled_date}`)
		if (query.message) formPrefill.value.push(`message=${encodeURIComponent(String(query.message))}`)
	}
})

onMounted(loadOpponents)
</script>

<style lang="scss" scoped>
@import './opponentSelector.scss';
</style>
