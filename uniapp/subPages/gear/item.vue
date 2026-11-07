<template>
	<view class="gear-item-page" :class="{ 'dark-mode': isDarkMode }">
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

		<view v-else-if="data && !item" class="state-block">
			<uni-icons type="info" size="48" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="state-title">装备不存在</text>
			<text class="state-desc">这件装备可能已经被删除。</text>
			<button class="primary-btn" @tap="goBack">返回</button>
		</view>

		<template v-else-if="item">
			<view v-if="refreshError" class="refresh-error-banner">
				<text>{{ refreshError.description }}</text>
				<text class="refresh-error-action" @tap="retry">重试</text>
			</view>

			<view v-if="justAdded && isCue" class="added-banner">
				<text>已添加。给这根杆设置现在装的皮头和巧克，对局会按它们记录。</text>
			</view>

			<view class="hero-card">
				<view class="hero-image"><gearImage :subject="item" fill /></view>
				<text class="hero-title">{{ title }}</text>
				<view class="hero-tags">
					<text v-if="subtitle" class="tag">{{ subtitle }}</text>
					<text v-if="retired" class="tag tag--muted">已退役</text>
				</view>
			</view>

			<view v-if="item.unlisted" class="notice-card">
				<text class="notice-title">未收录型号</text>
				<text class="notice-desc">你填写的名称只有你自己能看到，其他人会看到「未收录型号」。型号被收录后，这里会自动换成正式的名称、规格和图片。</text>
			</view>

			<view v-if="specs.length" class="card">
				<view class="card-head">
					<text class="card-title">官方规格</text>
					<text class="card-caption">以品牌公布为准</text>
				</view>
				<view v-for="spec in specs" :key="spec.key" class="spec-row">
					<text class="spec-label">{{ spec.label }}</text>
					<text class="spec-value">{{ spec.value }}</text>
				</view>
			</view>

			<view v-if="isCue" class="card">
				<view class="card-head">
					<text class="card-title">皮头与巧克</text>
					<text class="card-caption">对局会记录当时装的组合</text>
				</view>
				<view v-for="part in parts" :key="part.category" class="part-row">
					<view class="part-image"><gearImage :subject="part.subject" fill /></view>
					<view class="part-copy">
						<text class="part-label">{{ part.label }}</text>
						<text class="part-value" :class="{ empty: !part.product }">{{ part.text }}</text>
					</view>
					<view class="part-actions">
						<text class="part-link" @tap="pickPart(part.category)">{{ part.product ? '更换' : '选择' }}</text>
						<text v-if="part.product" class="part-link part-link--muted" @tap="clearPart(part.category)">清除</text>
					</view>
				</view>
				<text class="part-hint">找不到你用的皮头或巧克？<text class="part-hint-link" @tap="goFeedback">告诉我们</text>，我们会尽快收录。</text>
			</view>

			<view v-if="mainRows.length" class="card">
				<view class="card-head">
					<text class="card-title">设为主力</text>
					<text class="card-caption">对局完成时自动记录这件装备</text>
				</view>
				<view class="chip-row">
					<text
						v-for="row in mainRows"
						:key="row.gameType"
						class="chip"
						:class="{ active: row.active }"
						@tap="toggleMain(row)"
					>{{ row.label }}</text>
				</view>
			</view>
			<view v-else-if="retired && (isCue || isBreakCapable)" class="notice-card">
				<text class="notice-desc">退役的装备不能设为主力。恢复使用后可以重新设置。</text>
			</view>

			<view class="action-card">
				<button class="ghost-btn action-btn" :disabled="busy" @tap="toggleRetire">{{ retired ? '恢复使用' : '退役' }}</button>
				<button class="danger-btn" :disabled="busy" @tap="confirmDelete">删除装备</button>
				<text class="action-hint">退役后装备会收进「已退役」，历史对局的记录不受影响；删除后不再出现在装备列表里。</text>
			</view>
		</template>
	</view>
</template>

<script setup>
import { computed, onUnmounted, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import gearImage from '@/components/gearImage.vue'
import { deleteGearItem, setGearDefault, updateGearItem } from '@/api/gear.js'
import { useGearStore } from '@/store/gear.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	GEAR_CATEGORY,
	GEAR_ITEM_STATUS,
	GEAR_SLOT,
	buildGearDefaultsView,
	canBeMainBreakCue,
	canBeMainCue,
	formatGearSpecs,
	gearItemSubtitle,
	gearItemTitle,
	gearProductTitle
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

const itemId = ref(0)
const justAdded = ref(false)
const busy = ref(false)
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

const item = computed(() => (data.value?.items || []).find((entry) => entry.id === itemId.value) || null)
const defaults = computed(() => data.value?.defaults || [])
const retired = computed(() => item.value?.status === GEAR_ITEM_STATUS.RETIRED)
const isCue = computed(() => item.value?.category === GEAR_CATEGORY.CUE)
const isBreakCapable = computed(() => item.value?.category === GEAR_CATEGORY.BREAK_JUMP)
const title = computed(() => gearItemTitle(item.value))
const subtitle = computed(() => gearItemSubtitle(item.value))
const specs = computed(() => formatGearSpecs(item.value?.product))

const parts = computed(() => {
	const current = item.value
	if (!current) return []
	return [
		{ category: GEAR_CATEGORY.TIP, label: '皮头', product: current.tip_product },
		{ category: GEAR_CATEGORY.CHALK, label: '巧克', product: current.chalk_product }
	].map((part) => ({
		...part,
		subject: part.product || { category: part.category },
		text: part.product ? gearProductTitle(part.product) : '未设置'
	}))
})

// 能设为主力的球种：球杆有四个球种，冲杆没有斯诺克；退役装备不能设为主力。
const mainRows = computed(() => {
	const current = item.value
	if (!current) return []
	const accepts = isCue.value ? canBeMainCue(current) : canBeMainBreakCue(current)
	if (!accepts) return []
	const slot = isCue.value ? GEAR_SLOT.CUE : GEAR_SLOT.BREAK
	return buildGearDefaultsView(defaults.value, data.value?.items || [])
		.filter((row) => slot === GEAR_SLOT.CUE || row.supportsBreak)
		.map((row) => ({
			gameType: row.gameType,
			label: row.label,
			slot,
			active: (slot === GEAR_SLOT.CUE ? row.cue : row.breakCue)?.id === current.id
		}))
})

const currentIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const goLogin = () => {
	uni.navigateTo({ url: '/pages/login/login' })
}

const goBack = () => {
	uni.navigateBack({ fail: () => uni.redirectTo({ url: '/subPages/gear/index' }) })
}

const goFeedback = () => {
	uni.navigateTo({ url: '/subPages/help/feedback' })
}

const loadGear = async ({ force = false } = {}) => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		goLogin()
		return
	}
	const identity = currentIdentity()
	const cached = gearStore.peek(identity)
	const nextState = beginAsyncPageLoad({ ...pageState.value, data: cached, hasData: Boolean(cached) }, {
		authGeneration: identity.authGeneration,
		emptyData: null
	})
	const request = getAsyncPageRequest(nextState)
	pageState.value = nextState
	try {
		const response = await gearStore.load(identity, { force })
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

const refreshAfterWrite = async () => {
	gearStore.invalidate()
	await loadGear({ force: true })
}

// 统一的写操作外壳：身份变化后丢弃结果；业务失败用服务端文案，其余用兜底文案。
const runWrite = async (request, { fallback, success = '', refresh = true }) => {
	if (busy.value) return false
	const identity = currentIdentity()
	busy.value = true
	try {
		const res = await request()
		if (!gearStore.isCurrent(identity)) return false
		if (!res?.success) throw new Error(res?.message || fallback)
		// 删除后页面马上返回，只需让缓存失效，不要先刷新出「装备不存在」。
		if (refresh) {
			await refreshAfterWrite()
		} else {
			gearStore.invalidate()
		}
		if (success) uni.showToast({ title: success, icon: 'none' })
		return true
	} catch (error) {
		if (gearStore.isCurrent(identity)) {
			uni.showToast({ title: error?.message || fallback, icon: 'none' })
		}
		return false
	} finally {
		busy.value = false
	}
}

// ========== 皮头与巧克 ==========
// 选择页通过带 token 的全局事件回传结果；页面卸载时统一移除没触发的监听。
const pendingPickEvents = new Set()

const applyPart = (category, product) => {
	const payload = { item_id: itemId.value }
	if (category === GEAR_CATEGORY.TIP) {
		payload.update_tip = true
		payload.tip_product_id = product?.id || 0
	} else {
		payload.update_chalk = true
		payload.chalk_product_id = product?.id || 0
	}
	return runWrite(() => updateGearItem(payload), { fallback: '保存失败，请重试', success: '已保存' })
}

const pickPart = (category) => {
	const token = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
	const eventName = `gear:pick:${token}`
	const handler = (payload) => {
		pendingPickEvents.delete(eventName)
		applyPart(category, payload?.product || null)
	}
	pendingPickEvents.add(eventName)
	uni.$once(eventName, handler)
	uni.navigateTo({ url: `/subPages/gear/add?mode=pick&category=${category}&token=${token}` })
}

const clearPart = (category) => {
	applyPart(category, null)
}

// ========== 主力 ==========
const toggleMain = (row) => {
	return runWrite(
		() => setGearDefault({ game_type: row.gameType, slot: row.slot, item_id: row.active ? 0 : itemId.value }),
		{ fallback: '设置失败，请重试', success: row.active ? '已取消主力' : `已设为${row.label}主力` }
	)
}

// ========== 退役与删除 ==========
const toggleRetire = () => {
	const next = retired.value ? GEAR_ITEM_STATUS.ACTIVE : GEAR_ITEM_STATUS.RETIRED
	if (next === GEAR_ITEM_STATUS.ACTIVE) {
		runWrite(() => updateGearItem({ item_id: itemId.value, status: next }), { fallback: '操作失败，请重试', success: '已恢复使用' })
		return
	}
	uni.showModal({
		title: '退役这件装备？',
		content: '退役后它会收进「已退役」，并取消它的主力设置；历史对局的记录不受影响。',
		confirmText: '退役',
		success: (result) => {
			if (result.confirm) {
				runWrite(() => updateGearItem({ item_id: itemId.value, status: next }), { fallback: '操作失败，请重试', success: '已退役' })
			}
		}
	})
}

const confirmDelete = () => {
	uni.showModal({
		title: '删除这件装备？',
		content: '删除后不再出现在你的装备列表里；已经记录的历史对局仍会显示它。',
		confirmText: '删除',
		confirmColor: '#EF4444',
		success: async (result) => {
			if (!result.confirm) return
			const done = await runWrite(() => deleteGearItem({ item_id: itemId.value }), { fallback: '删除失败，请重试', success: '已删除', refresh: false })
			if (done) goBack()
		}
	})
}

onLoad((options) => {
	itemId.value = Number(options?.id) || 0
	justAdded.value = options?.added === '1'
})

onShow(() => {
	loadGear()
})

onUnmounted(() => {
	for (const eventName of pendingPickEvents) {
		uni.$off(eventName)
	}
	pendingPickEvents.clear()
})
</script>

<style lang="scss" scoped>
@import './item.scss';
</style>
