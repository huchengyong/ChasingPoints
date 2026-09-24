<template>
	<view class="opponent-selector" :class="{ 'dark-mode': isDarkMode }">
		<view class="selector-intro">
			<text class="selector-title">选择一位球友发起 PK</text>
			<text class="selector-copy">好友优先展示，最近对手按最近对局排序。发送邀约不会直接创建正式对局。</text>
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
				:class="{ selected: selectedUserId === item.user_id }"
				@tap="selectedUserId = item.user_id"
			>
				<image class="opponent-avatar" :src="resolveAvatarUrl(item.avatar, item.user_id)" mode="aspectFill" />
				<view class="opponent-copy">
					<text class="opponent-name">{{ item.nickname }}</text>
					<text class="opponent-source">{{ item.source === 'friend' ? '好友' : '最近对手' }}</text>
				</view>
				<uni-icons
					:type="selectedUserId === item.user_id ? 'checkbox-filled' : 'circle'"
					size="22"
					:color="selectedUserId === item.user_id ? '#E0AE12' : '#cbd5e1'"
				></uni-icons>
			</view>
		</view>

		<view v-if="opponents.length > 0" class="selector-footer">
			<text class="game-type-title">选择球种</text>
			<view class="game-type-grid">
				<button
					v-for="item in gameTypes"
					:key="item.value"
					class="game-type-button"
					:class="{ selected: selectedGameType === item.value }"
					@tap="selectedGameType = item.value"
				>
					{{ item.label }}
				</button>
			</view>
			<button class="send-button" :disabled="!selectedOpponent || sending" @tap="submitChallenge">
				{{ sending ? '发送中...' : '发送PK邀约' }}
			</button>
		</view>
	</view>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { getFriendList } from '@/api/friend.js'
import { getMatchList } from '@/api/match.js'
import { sendChallenge } from '@/api/challenge.js'
import { buildChallengePayload } from '@/utils/challenge-entry.js'
import { GAME_TYPE_OPTIONS } from '@/utils/game-types.js'
import { buildOpponentSelectionList } from '@/utils/opponent-selector.js'
import { usePageTheme } from '@/utils/page-theme.js'
import { resolveAvatarUrl } from '@/utils/user-profile.js'

const { isDarkMode } = usePageTheme()
const loading = ref(true)
const sending = ref(false)
const opponents = ref([])
const selectedUserId = ref(0)
const selectedGameType = ref(GAME_TYPE_OPTIONS[0].value)
const gameTypes = GAME_TYPE_OPTIONS
const selectedOpponent = computed(() => opponents.value.find(item => item.user_id === selectedUserId.value) || null)

const loadOpponents = async () => {
	loading.value = true
	try {
		const [friendRes, matchRes] = await Promise.all([
			getFriendList({ page: 1, page_size: 100 }),
			getMatchList({ page: 1, page_size: 100 })
		])
		opponents.value = buildOpponentSelectionList({
			friends: friendRes.list || friendRes || [],
			matches: matchRes.list || []
		})
	} catch (error) {
		console.error('加载PK对手失败', error)
		uni.showToast({ title: '加载球友失败', icon: 'none' })
	} finally {
		loading.value = false
	}
}

const submitChallenge = async () => {
	if (!selectedOpponent.value || sending.value) return
	sending.value = true
	try {
		const res = await sendChallenge(buildChallengePayload({
			targetFriend: selectedOpponent.value,
			gameType: selectedGameType.value
		}))
		if (!res?.success) {
			uni.showToast({ title: res?.message || res?.msg || '发送失败', icon: 'none' })
			return
		}
		uni.showModal({
			title: 'PK邀约已发送',
			content: '对方接受后，双方线下扫码才会创建正式对局。',
			confirmText: '查看PK记录',
			cancelText: '返回',
			success: ({ confirm }) => {
				if (confirm) {
					uni.navigateTo({ url: '/subPages/social/challenges' })
					return
				}
				uni.navigateBack()
			}
		})
	} catch (error) {
		uni.showToast({ title: '发送失败', icon: 'none' })
	} finally {
		sending.value = false
	}
}

onMounted(loadOpponents)
</script>

<style lang="scss" scoped>
@import './opponentSelector.scss';
</style>
