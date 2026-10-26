<template>
	<view class="gear-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="isInitialLoading" class="state-block">
			<uni-icons type="spinner-cycle" size="36" color="#E0AE12"></uni-icons>
			<text class="state-title">加载装备中...</text>
		</view>

		<view v-else-if="pageError" class="state-block">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="state-title">{{ pageError.title }}</text>
			<text class="state-desc">{{ pageError.description }}</text>
			<button class="primary-btn" @tap="retry">{{ pageError.actionText }}</button>
		</view>

		<template v-else-if="data">
			<view v-if="refreshError" class="refresh-error-banner">
				<text>{{ refreshError.description }}</text>
				<text class="refresh-error-action" @tap="retry">重试</text>
			</view>

			<view class="card">
				<view class="card-head">
					<text class="card-title">主力装备</text>
					<text class="card-caption">对局完成时自动记下用杆</text>
				</view>
				<view v-for="row in defaultsView" :key="row.gameType" class="default-row">
					<text class="default-game">{{ row.label }}</text>
					<view class="default-slots">
						<view class="slot" @tap="openPicker(row, 'cue')">
							<text class="slot-label">球杆</text>
							<text class="slot-value" :class="{ empty: !row.cue }">{{ row.cue ? itemTitle(row.cue) : '未设置' }}</text>
						</view>
						<view v-if="row.supportsBreak" class="slot" @tap="openPicker(row, 'break')">
							<text class="slot-label">冲杆</text>
							<text class="slot-value" :class="{ empty: !row.breakCue }">{{ row.breakCue ? itemTitle(row.breakCue) : '未设置' }}</text>
						</view>
					</view>
				</view>
			</view>

			<view class="section-head">
				<view class="section-copy">
					<text class="section-title">我的装备</text>
					<text class="section-count">{{ itemCountText }}</text>
				</view>
				<button class="ghost-btn" :disabled="atLimit" @tap="goAdd('cue')">添加装备</button>
			</view>

			<view v-if="isEmpty" class="card empty-card">
				<text class="empty-title">还没有装备</text>
				<text class="empty-desc">登记你的球杆后，每场对局完成时会自动记下这场用的杆、皮头和巧克。</text>
				<button class="primary-btn" @tap="goAdd('cue')">添加球杆</button>
			</view>

			<template v-else>
				<view v-if="groups.cues.length" class="group">
					<text class="group-title">球杆</text>
					<view class="grid">
						<view v-for="item in groups.cues" :key="item.id" class="grid-cell" @tap="openItem(item)">
							<gearItemCard :item="item" :badges="mainLabels(item)" />
						</view>
					</view>
				</view>
				<view v-if="groups.breakJump.length" class="group">
					<text class="group-title">冲杆 / 跳杆</text>
					<view class="grid">
						<view v-for="item in groups.breakJump" :key="item.id" class="grid-cell" @tap="openItem(item)">
							<gearItemCard :item="item" :badges="mainLabels(item)" />
						</view>
					</view>
				</view>
				<view v-if="groups.retired.length" class="group">
					<text class="group-title">已退役</text>
					<view class="grid">
						<view v-for="item in groups.retired" :key="item.id" class="grid-cell" @tap="openItem(item)">
							<gearItemCard :item="item" />
						</view>
					</view>
				</view>
			</template>

			<text class="privacy-note">好友可以看到你在用的装备；未收录型号的名称只有你自己能看到。</text>
		</template>

		<view v-if="picker.visible" class="sheet-mask" @tap="closePicker"></view>
		<view v-if="picker.visible" class="sheet">
			<view class="sheet-head">
				<text class="sheet-title">{{ picker.title }}</text>
				<text class="sheet-close" @tap="closePicker">取消</text>
			</view>
			<scroll-view scroll-y class="sheet-body">
				<view v-if="!picker.candidates.length" class="picker-empty">
					<text class="picker-empty-text">{{ picker.emptyText }}</text>
					<button class="ghost-btn picker-empty-btn" @tap="goAddFromPicker">去添加</button>
				</view>
				<view
					v-for="item in picker.candidates"
					:key="item.id"
					class="picker-row"
					:class="{ active: item.id === picker.currentId }"
					@tap="choose(item.id)"
				>
					<view class="picker-image"><gearImage :subject="item" fill /></view>
					<view class="picker-copy">
						<text class="picker-title">{{ itemTitle(item) }}</text>
						<text class="picker-subtitle">{{ itemSubtitle(item) }}</text>
					</view>
					<text v-if="item.id === picker.currentId" class="picker-mark">当前</text>
				</view>
				<view v-if="picker.currentId" class="picker-clear" @tap="choose(0)">
					<text>不设主力</text>
				</view>
			</scroll-view>
		</view>
	</view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import gearImage from '@/components/gearImage.vue'
import gearItemCard from '@/components/gearItemCard.vue'
import { setGearDefault } from '@/api/gear.js'
import { useGearStore } from '@/store/gear.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	GEAR_MAX_ITEMS,
	GEAR_SLOT,
	buildGearDefaultsView,
	gearItemSubtitle,
	gearItemTitle,
	groupGearItems,
	listMainCandidates,
	listMainGameLabels
} from '@/utils/gear-display.js'
import {
	ASYNC_PAGE_STATUS,
	beginAsyncPageLoad,
	createAsyncPageState,
	getAsyncPageRequest,
	rejectAsyncPageLoad,
	resolveAsyncPageErrorFeedback,
	resolveAsyncPageLoad
} from '@/utils/async-page-state.js'

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const gearStore = useGearStore()

const pageState = ref(createAsyncPageState({ authGeneration: userStore.authGeneration, data: null }))

const data = computed(() => pageState.value.data)
const isInitialLoading = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.LOADING || pageState.value.status === ASYNC_PAGE_STATUS.IDLE
))
const pageError = computed(() => (
	pageState.value.status === ASYNC_PAGE_STATUS.ERROR
		? resolveAsyncPageErrorFeedback(pageState.value.error, { resource: '装备' })
		: null
))
const refreshError = computed(() => (
	pageState.value.refreshError
		? resolveAsyncPageErrorFeedback(pageState.value.refreshError, { resource: '装备' })
		: null
))

const items = computed(() => data.value?.items || [])
const defaults = computed(() => data.value?.defaults || [])
const groups = computed(() => groupGearItems(items.value))
const defaultsView = computed(() => buildGearDefaultsView(defaults.value, items.value))
const maxItems = computed(() => data.value?.max_items || GEAR_MAX_ITEMS)
const itemCountText = computed(() => `${items.value.length}/${maxItems.value}`)
const atLimit = computed(() => items.value.length >= maxItems.value)
const isEmpty = computed(() => items.value.length === 0)

const itemTitle = gearItemTitle
const itemSubtitle = gearItemSubtitle
const mainLabels = (item) => listMainGameLabels(defaults.value, item.id)

const currentIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const goLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}

const loadGear = async ({ force = false } = {}) => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}
	const identity = currentIdentity()
	// 同身份的缓存先撑住页面，没有缓存才显示加载态。
	const cached = gearStore.peek(identity)
	const nextState = beginAsyncPageLoad({ ...pageState.value, data: cached, hasData: Boolean(cached) }, {
		authGeneration: identity.authGeneration,
		emptyData: null
	})
	const request = getAsyncPageRequest(nextState)
	pageState.value = nextState
	try {
		const response = await gearStore.load(identity, { force })
		// null 表示读取期间身份已变化，旧身份的结果不能回写。
		if (!response) return
		const applied = resolveAsyncPageLoad(pageState.value, request, { data: response, isEmpty: () => false })
		if (applied === pageState.value) return
		pageState.value = applied
	} catch (error) {
		pageState.value = rejectAsyncPageLoad(pageState.value, request, error)
	}
}

const retry = () => {
	loadGear({ force: true })
}

const goAdd = (category) => {
	uni.navigateTo({ url: `/subPages/gear/add?category=${category}` })
}

const openItem = (item) => {
	uni.navigateTo({ url: `/subPages/gear/item?id=${item.id}` })
}

// ========== 主力选择抽屉 ==========
const picker = ref({ visible: false, gameType: 0, slot: GEAR_SLOT.CUE, title: '', candidates: [], currentId: 0, emptyText: '', saving: false })

const openPicker = (row, slot) => {
	const isBreak = slot === GEAR_SLOT.BREAK
	picker.value = {
		visible: true,
		gameType: row.gameType,
		slot,
		title: `${row.label} · 主力${isBreak ? '冲杆' : '球杆'}`,
		candidates: listMainCandidates(items.value, slot),
		currentId: (isBreak ? row.breakCue : row.cue)?.id || 0,
		emptyText: isBreak ? '还没有可选的冲杆，先添加一根带冲杆身份的装备。' : '还没有可选的球杆，先添加一根球杆。',
		saving: false
	}
}

const closePicker = () => {
	picker.value = { ...picker.value, visible: false }
}

const goAddFromPicker = () => {
	const category = picker.value.slot === GEAR_SLOT.BREAK ? 'break_jump' : 'cue'
	closePicker()
	goAdd(category)
}

const choose = async (itemId) => {
	if (picker.value.saving) return
	const identity = currentIdentity()
	const { gameType, slot } = picker.value
	picker.value = { ...picker.value, saving: true }
	try {
		const res = await setGearDefault({ game_type: gameType, slot, item_id: itemId })
		if (!gearStore.isCurrent(identity)) return
		if (!res?.success) throw new Error(res?.message || '设置失败')
		gearStore.invalidate()
		closePicker()
		await loadGear({ force: true })
	} catch (error) {
		picker.value = { ...picker.value, saving: false }
		uni.showToast({ title: error?.message || '设置失败，请重试', icon: 'none' })
	}
}

onShow(() => {
	loadGear()
})
</script>

<style lang="scss" scoped>
@import './index.scss';
</style>
