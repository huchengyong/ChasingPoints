<template>
	<view class="gear-friend-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="loading" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-title">加载装备中...</text>
		</view>

		<view v-else-if="failure" class="state-block">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="state-title">{{ failure.title }}</text>
			<text class="state-desc">{{ failure.description }}</text>
			<button class="primary-btn" @tap="failure.retryable ? load() : goBack()">{{ failure.retryable ? '重试' : '返回' }}</button>
		</view>

		<template v-else>
			<view class="owner-line">
				<text class="owner-name">{{ ownerTitle }}</text>
				<text class="owner-caption">只展示在用的装备</text>
			</view>

			<view v-if="mainRows.length" class="card">
				<view class="card-head">
					<text class="card-title">主力装备</text>
				</view>
				<view v-for="row in mainRows" :key="row.gameType" class="main-row">
					<text class="main-game">{{ row.label }}</text>
					<view class="main-values">
						<text class="main-value">球杆：{{ row.cue ? itemTitle(row.cue) : '未设置' }}</text>
						<text v-if="row.supportsBreak" class="main-value">冲杆：{{ row.breakCue ? itemTitle(row.breakCue) : '未设置' }}</text>
					</view>
				</view>
			</view>

			<view v-if="!items.length" class="card empty-card">
				<text class="empty-title">TA 还没有展示装备</text>
				<text class="empty-desc">TA 登记球杆后，会显示在这里。</text>
			</view>
			<template v-else>
				<view v-if="groups.cues.length" class="group">
					<text class="group-title">球杆</text>
					<view class="grid">
						<view v-for="item in groups.cues" :key="item.id" class="grid-cell">
							<gearItemCard :item="item" :badges="mainLabels(item)" />
						</view>
					</view>
				</view>
				<view v-if="groups.breakJump.length" class="group">
					<text class="group-title">冲杆 / 跳杆</text>
					<view class="grid">
						<view v-for="item in groups.breakJump" :key="item.id" class="grid-cell">
							<gearItemCard :item="item" :badges="mainLabels(item)" />
						</view>
					</view>
				</view>
			</template>
		</template>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import gearItemCard from '@/components/gearItemCard.vue'
import { getUserGear } from '@/api/gear.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	buildGearDefaultsView,
	gearItemTitle,
	groupGearItems,
	listMainGameLabels
} from '@/utils/gear-display.js'
import { resolveAsyncPageErrorFeedback } from '@/utils/async-page-state.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()

const targetUserId = ref(0)
const ownerName = ref('TA')
const loading = ref(true)
const failure = ref(null)
const items = ref([])
const defaults = ref([])

// 好友关系随时可能变化，所以好友装备不进缓存，每次进入页面都回源。
let requestSeq = 0

const groups = computed(() => groupGearItems(items.value))
// 默认的「TA」是英文占位，和「的装备」之间留空格；中文昵称直接连着写。
const ownerTitle = computed(() => (ownerName.value === 'TA' ? 'TA 的装备' : `${ownerName.value}的装备`))
const itemTitle = gearItemTitle
const mainLabels = (item) => listMainGameLabels(defaults.value, item.id)
// 一个主力都没设置时不展示这块，避免一排「未设置」。
const mainRows = computed(() => {
	const rows = buildGearDefaultsView(defaults.value, items.value)
	return rows.some((row) => row.cue || row.breakCue) ? rows : []
})

const goBack = () => {
	uni.navigateBack({ fail: () => uni.switchTab({ url: '/pages/user/index' }) })
}

const load = async () => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		uni.navigateTo({ url: '/pages/login/login' })
		return
	}
	if (!targetUserId.value) {
		failure.value = { title: '用户不存在', description: '请返回后重新进入。', retryable: false }
		loading.value = false
		return
	}
	const identity = { userId: userStore.userId, authGeneration: userStore.authGeneration }
	const seq = ++requestSeq
	loading.value = true
	failure.value = null
	try {
		const res = await getUserGear({ user_id: targetUserId.value })
		// 请求期间切号或又发起了新的读取：旧结果不能回写。
		if (seq !== requestSeq || identity.userId !== userStore.userId || identity.authGeneration !== userStore.authGeneration) return
		if (res?.success) {
			items.value = res.items || []
			defaults.value = res.defaults || []
		} else if (res?.reason === 'not_friend') {
			failure.value = { title: '仅好友可以查看', description: '加为好友后，就能看到 TA 的装备。', retryable: false }
		} else {
			failure.value = { title: '装备加载失败', description: res?.message || '请稍后重试', retryable: true }
		}
	} catch (error) {
		if (seq !== requestSeq) return
		const feedback = resolveAsyncPageErrorFeedback(error, { resource: '装备' })
		failure.value = { title: feedback.title, description: feedback.description, retryable: feedback.retryable }
	} finally {
		if (seq === requestSeq) loading.value = false
	}
}

onLoad((options) => {
	targetUserId.value = Number(options?.user_id) || 0
	if (options?.name) {
		try {
			ownerName.value = decodeURIComponent(options.name)
		} catch (error) {
			ownerName.value = 'TA'
		}
	}
	load()
})
</script>

<style lang="scss" scoped>
@import './friend.scss';
</style>
