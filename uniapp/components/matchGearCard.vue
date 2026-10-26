<template>
	<view v-if="status !== 'hidden'" class="match-gear">
		<view class="match-gear-head">
			<text class="match-gear-title">本场用杆</text>
			<text v-if="canEdit" class="match-gear-edit" @tap="openEditor">修改</text>
		</view>

		<view v-if="status === 'idle' || status === 'loading'" class="match-gear-tip">
			<text>加载中...</text>
		</view>
		<view v-else-if="status === 'error'" class="match-gear-tip">
			<text>{{ errorMessage }}</text>
			<text class="match-gear-link" @tap="load">重试</text>
		</view>
		<view v-else class="match-gear-sides">
			<view v-for="side in sides" :key="side.key" class="match-gear-side">
				<text class="match-gear-side-label">{{ side.label }}</text>
				<view v-if="!side.view.recorded" class="match-gear-empty">
					<text class="match-gear-empty-text">未记录</text>
					<text v-if="side.mine && canEdit" class="match-gear-link" @tap="openEditor">补记</text>
				</view>
				<view v-for="row in side.view.rows" :key="row.key" class="match-gear-row">
					<view class="match-gear-thumb"><gearImage :subject="row.subject" fill /></view>
					<view class="match-gear-row-copy">
						<text class="match-gear-row-label">{{ row.label }}</text>
						<text class="match-gear-row-title">{{ row.title }}</text>
					</view>
				</view>
			</view>
		</view>
		<text v-if="status === 'ready' && canEdit && deadlineText" class="match-gear-foot">{{ deadlineText }}</text>

		<view v-if="editor.visible" class="match-gear-mask" @tap="closeEditor"></view>
		<view v-if="editor.visible" class="match-gear-sheet">
			<view class="sheet-head">
				<text class="sheet-title">修改本场用杆</text>
				<text class="sheet-close" @tap="closeEditor">取消</text>
			</view>
			<scroll-view scroll-y class="sheet-body">
				<view v-if="editor.itemsLoading" class="sheet-tip"><text>加载我的装备...</text></view>
				<view v-else-if="editor.itemsError" class="sheet-tip">
					<text>{{ editor.itemsError }}</text>
					<text class="match-gear-link" @tap="loadMyItems">重试</text>
				</view>
				<template v-else>
					<text class="sheet-section">球杆</text>
					<view class="option" :class="{ active: !editor.draft.cueItemId }" @tap="chooseCue(null)">
						<text class="option-title">不记录</text>
					</view>
					<view
						v-for="item in cueOptions"
						:key="item.id"
						class="option"
						:class="{ active: editor.draft.cueItemId === item.id }"
						@tap="chooseCue(item)"
					>
						<view class="option-thumb"><gearImage :subject="item" fill /></view>
						<text class="option-title">{{ itemTitle(item) }}</text>
						<text v-if="item.status === 2" class="option-tag">已退役</text>
					</view>
					<view v-if="!cueOptions.length" class="sheet-tip">
						<text>还没有球杆，先到「我的 → 装备」添加。</text>
					</view>

					<text class="sheet-section">皮头</text>
					<view class="part">
						<text class="part-value" :class="{ empty: !editor.draft.tipProduct }">{{ partText(editor.draft.tipProduct) }}</text>
						<text class="match-gear-link" @tap="pickPart('tip')">{{ editor.draft.tipProduct ? '更换' : '选择' }}</text>
						<text v-if="editor.draft.tipProduct" class="match-gear-link match-gear-link--muted" @tap="clearPart('tip')">清除</text>
					</view>

					<text class="sheet-section">巧克</text>
					<view class="part">
						<text class="part-value" :class="{ empty: !editor.draft.chalkProduct }">{{ partText(editor.draft.chalkProduct) }}</text>
						<text class="match-gear-link" @tap="pickPart('chalk')">{{ editor.draft.chalkProduct ? '更换' : '选择' }}</text>
						<text v-if="editor.draft.chalkProduct" class="match-gear-link match-gear-link--muted" @tap="clearPart('chalk')">清除</text>
					</view>

					<template v-if="data && data.supports_break">
						<text class="sheet-section">冲杆</text>
						<view class="option" :class="{ active: !editor.draft.breakItemId }" @tap="chooseBreak(null)">
							<text class="option-title">不记录</text>
						</view>
						<view
							v-for="item in breakOptions"
							:key="item.id"
							class="option"
							:class="{ active: editor.draft.breakItemId === item.id }"
							@tap="chooseBreak(item)"
						>
							<view class="option-thumb"><gearImage :subject="item" fill /></view>
							<text class="option-title">{{ itemTitle(item) }}</text>
							<text v-if="item.status === 2" class="option-tag">已退役</text>
						</view>
					</template>
				</template>
			</scroll-view>
			<button class="sheet-save" :disabled="editor.saving || editor.itemsLoading" @tap="save">
				{{ editor.saving ? '保存中...' : '保存' }}
			</button>
		</view>
	</view>
</template>

<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'
import gearImage from '@/components/gearImage.vue'
import { getGearMatch, updateGearMatch } from '@/api/gear.js'
import { useGearStore } from '@/store/gear.js'
import { useUserStore } from '@/store/user.js'
import { gearItemTitle, gearProductTitle } from '@/utils/gear-display.js'
import {
	applyCueToDraft,
	buildGearMatchDraft,
	buildGearMatchPayload,
	canEditGearMatch,
	createGearMatchLoader,
	describeGearMatchSide,
	formatGearEditDeadline,
	isGearMatchHiddenResponse,
	listMatchBreakOptions,
	listMatchCueOptions
} from '@/utils/gear-match.js'

// 本场用杆：双方在这场对局里用的球杆、皮头、巧克和冲杆。
// 读取失败只影响这张卡片，不影响页面其余内容；服务端不认你是这场的选手时整张卡片隐藏。
const props = defineProps({
	matchId: { type: [Number, String], default: 0 },
	// 由页面按「已完成且我是选手」决定是否展示，避免对裁判、观战者发起明知会被拒绝的请求
	enabled: { type: Boolean, default: false },
	myLabel: { type: String, default: '我' },
	opponentLabel: { type: String, default: '对手' }
})
const emit = defineEmits(['updated'])

const userStore = useUserStore()
const gearStore = useGearStore()

const status = ref('hidden')
const data = ref(null)
const errorMessage = ref('')
const nowTick = ref(Date.now())

const currentIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const loader = createGearMatchLoader({
	request: (params) => getGearMatch(params, { silent: true }),
	readIdentity: currentIdentity
})

const itemTitle = gearItemTitle
const partText = (product) => (product ? gearProductTitle(product) : '未设置')

const canEdit = computed(() => status.value === 'ready' && canEditGearMatch(data.value, nowTick.value))
const deadlineText = computed(() => formatGearEditDeadline(data.value?.edit_deadline))
const sides = computed(() => {
	if (!data.value) return []
	const list = [{ key: 'me', label: props.myLabel, mine: true, view: describeGearMatchSide(data.value.me) }]
	if (data.value.has_opponent) {
		list.push({ key: 'opponent', label: props.opponentLabel, mine: false, view: describeGearMatchSide(data.value.opponent) })
	}
	return list
})

const load = async () => {
	const matchId = Number(props.matchId) || 0
	if (!props.enabled || !matchId || !userStore.isLoggedIn || !userStore.userId) {
		loader.cancel()
		status.value = 'hidden'
		return
	}
	status.value = 'loading'
	errorMessage.value = ''
	try {
		const result = await loader.load(matchId)
		if (result.stale) return
		const response = result.response
		if (response?.success) {
			data.value = response
			nowTick.value = Date.now()
			status.value = 'ready'
		} else if (isGearMatchHiddenResponse(response)) {
			status.value = 'hidden'
		} else {
			errorMessage.value = response?.message || '本场用杆加载失败'
			status.value = 'error'
		}
	} catch (error) {
		errorMessage.value = '本场用杆加载失败，请检查网络后重试'
		status.value = 'error'
	}
}

watch(() => [props.matchId, props.enabled], load, { immediate: true })

// ========== 修改 ==========
const editor = ref({ visible: false, saving: false, draft: buildGearMatchDraft(null), items: [], itemsLoading: false, itemsError: '' })
const cueOptions = computed(() => listMatchCueOptions(editor.value.items))
const breakOptions = computed(() => listMatchBreakOptions(editor.value.items))

const loadMyItems = async () => {
	const identity = currentIdentity()
	editor.value = { ...editor.value, itemsLoading: true, itemsError: '' }
	try {
		const response = await gearStore.load(identity)
		if (!response || !editor.value.visible) return
		editor.value = { ...editor.value, items: response.items || [], itemsLoading: false }
	} catch (error) {
		if (!gearStore.isCurrent(identity)) return
		editor.value = { ...editor.value, itemsLoading: false, itemsError: '装备加载失败' }
	}
}

const openEditor = () => {
	if (!data.value) return
	editor.value = {
		visible: true,
		saving: false,
		draft: buildGearMatchDraft(data.value.me),
		items: [],
		itemsLoading: true,
		itemsError: ''
	}
	loadMyItems()
}

const closeEditor = () => {
	editor.value = { ...editor.value, visible: false, saving: false }
}

const chooseCue = (item) => {
	editor.value = { ...editor.value, draft: applyCueToDraft(editor.value.draft, item) }
}

const chooseBreak = (item) => {
	editor.value = { ...editor.value, draft: { ...editor.value.draft, breakItemId: item?.id || 0 } }
}

const setPart = (category, product) => {
	const draft = { ...editor.value.draft }
	if (category === 'tip') draft.tipProduct = product
	else draft.chalkProduct = product
	editor.value = { ...editor.value, draft }
}

const clearPart = (category) => {
	setPart(category, null)
}

// 皮头与巧克沿用目录选择页，结果通过带 token 的全局事件回传；卸载时移除没触发的监听。
const pendingPickEvents = new Set()

const pickPart = (category) => {
	const token = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
	const eventName = `gear:pick:${token}`
	pendingPickEvents.add(eventName)
	uni.$once(eventName, (payload) => {
		pendingPickEvents.delete(eventName)
		if (editor.value.visible) setPart(category, payload?.product || null)
	})
	uni.navigateTo({ url: `/subPages/gear/add?mode=pick&category=${category}&token=${token}` })
}

const save = async () => {
	if (editor.value.saving || !data.value) return
	const identity = currentIdentity()
	const matchId = Number(props.matchId) || 0
	editor.value = { ...editor.value, saving: true }
	try {
		const payload = buildGearMatchPayload(matchId, editor.value.draft, { supportsBreak: Boolean(data.value.supports_break) })
		const response = await updateGearMatch(payload)
		if (!gearStore.isCurrent(identity)) return
		if (!response?.success) throw new Error(response?.message || '保存失败')
		closeEditor()
		uni.showToast({ title: '已保存', icon: 'none' })
		// 返回里带完整视图就直接用；只有「已保存」没有视图时重新读取。
		if (response.me) {
			data.value = response
			nowTick.value = Date.now()
			status.value = 'ready'
		} else {
			await load()
		}
		emit('updated')
	} catch (error) {
		if (!gearStore.isCurrent(identity)) return
		editor.value = { ...editor.value, saving: false }
		uni.showToast({ title: error?.message || '保存失败，请重试', icon: 'none' })
	}
}

onUnmounted(() => {
	loader.cancel()
	for (const eventName of pendingPickEvents) {
		uni.$off(eventName)
	}
	pendingPickEvents.clear()
})
</script>

<style lang="scss" scoped>
.match-gear {
	margin: 20rpx 0;
	padding: 28rpx;
	border-radius: var(--ui-radius-lg, 24rpx);
	background: var(--ui-surface-card, #FFFFFF);
	border: 1rpx solid var(--ui-border-default, #E9E2CF);
	box-sizing: border-box;
}

.match-gear-head {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 16rpx;

	.match-gear-title {
		font-size: 30rpx;
		font-weight: 700;
		color: var(--ui-text-primary, #231C0B);
	}

	.match-gear-edit {
		padding: 6rpx 8rpx;
		font-size: 26rpx;
		color: var(--ui-brand-strong, #C69200);
	}
}

.match-gear-tip {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 16rpx;
	padding: 12rpx 0;
	font-size: 26rpx;
	color: var(--ui-text-muted, #9A8C67);
}

.match-gear-link {
	flex-shrink: 0;
	padding: 4rpx 8rpx;
	font-size: 26rpx;
	color: var(--ui-brand-strong, #C69200);

	&.match-gear-link--muted {
		color: var(--ui-text-muted, #9A8C67);
	}
}

.match-gear-sides {
	display: flex;
	gap: 20rpx;
}

.match-gear-side {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	gap: 14rpx;
	padding: 18rpx;
	border-radius: var(--ui-radius-md, 18rpx);
	background: var(--ui-surface-subtle, #FAF8F2);
	border: 1rpx solid var(--ui-border-default, #E9E2CF);
	box-sizing: border-box;

	.match-gear-side-label {
		font-size: 24rpx;
		font-weight: 600;
		color: var(--ui-text-secondary, #6E6242);
	}
}

.match-gear-empty {
	display: flex;
	align-items: center;
	justify-content: space-between;

	.match-gear-empty-text {
		font-size: 26rpx;
		color: var(--ui-text-muted, #9A8C67);
	}
}

.match-gear-row {
	display: flex;
	align-items: center;
	gap: 14rpx;

	.match-gear-thumb {
		flex-shrink: 0;
		width: 84rpx;
		height: 84rpx;
		border-radius: var(--ui-radius-sm, 12rpx);
		background: var(--ui-surface-card, #FFFFFF);
	}

	.match-gear-row-copy {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}

	.match-gear-row-label {
		font-size: 20rpx;
		color: var(--ui-text-muted, #9A8C67);
	}

	.match-gear-row-title {
		margin-top: 2rpx;
		font-size: 24rpx;
		line-height: 1.35;
		color: var(--ui-text-primary, #231C0B);
		overflow-wrap: break-word;
	}
}

.match-gear-foot {
	display: block;
	margin-top: 16rpx;
	font-size: 22rpx;
	color: var(--ui-text-muted, #9A8C67);
}

.match-gear-mask {
	position: fixed;
	left: 0;
	right: 0;
	top: 0;
	bottom: 0;
	z-index: 900;
	background: rgba(0, 0, 0, 0.5);
}

.match-gear-sheet {
	position: fixed;
	left: 0;
	right: 0;
	bottom: 0;
	z-index: 910;
	max-height: 82vh;
	display: flex;
	flex-direction: column;
	padding: 28rpx 28rpx calc(24rpx + env(safe-area-inset-bottom));
	border-radius: var(--ui-radius-xl, 32rpx) var(--ui-radius-xl, 32rpx) 0 0;
	background: var(--ui-surface-card, #FFFFFF);
	box-sizing: border-box;

	.sheet-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 8rpx;
	}

	.sheet-title {
		font-size: 32rpx;
		font-weight: 700;
		color: var(--ui-text-primary, #231C0B);
	}

	.sheet-close {
		padding: 8rpx 12rpx;
		font-size: 26rpx;
		color: var(--ui-text-muted, #9A8C67);
	}

	.sheet-body {
		flex: 1;
		min-height: 0;
		max-height: 60vh;
	}

	.sheet-section {
		display: block;
		margin: 24rpx 0 8rpx;
		font-size: 24rpx;
		font-weight: 600;
		color: var(--ui-text-secondary, #6E6242);
	}

	.sheet-tip {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16rpx;
		padding: 16rpx 4rpx;
		font-size: 24rpx;
		color: var(--ui-text-muted, #9A8C67);
	}

	.option {
		display: flex;
		align-items: center;
		gap: 16rpx;
		padding: 14rpx 16rpx;
		margin-top: 10rpx;
		border-radius: var(--ui-radius-md, 18rpx);
		background: var(--ui-surface-subtle, #FAF8F2);
		border: 1rpx solid var(--ui-border-default, #E9E2CF);

		&.active {
			background: var(--ui-brand-tint, rgba(224, 174, 18, 0.14));
			border-color: var(--ui-brand-primary, #E0AE12);
		}
	}

	.option-thumb {
		flex-shrink: 0;
		width: 88rpx;
		height: 88rpx;
	}

	.option-title {
		flex: 1;
		min-width: 0;
		font-size: 26rpx;
		color: var(--ui-text-primary, #231C0B);
		overflow-wrap: break-word;
	}

	.option-tag {
		flex-shrink: 0;
		font-size: 20rpx;
		color: var(--ui-text-muted, #9A8C67);
	}

	.part {
		display: flex;
		align-items: center;
		gap: 12rpx;
		padding: 14rpx 16rpx;
		border-radius: var(--ui-radius-md, 18rpx);
		background: var(--ui-surface-subtle, #FAF8F2);
		border: 1rpx solid var(--ui-border-default, #E9E2CF);
	}

	.part-value {
		flex: 1;
		min-width: 0;
		font-size: 26rpx;
		color: var(--ui-text-primary, #231C0B);
		overflow-wrap: break-word;

		&.empty {
			color: var(--ui-text-muted, #9A8C67);
		}
	}

	.sheet-save {
		height: 88rpx;
		line-height: 88rpx;
		margin: 20rpx 0 0;
		border-radius: var(--ui-radius-pill, 999rpx);
		background: var(--ui-brand-primary, #E0AE12);
		color: #FFFFFF;
		font-size: 30rpx;
		font-weight: 600;

		&::after {
			display: none;
		}

		&[disabled] {
			opacity: 0.5;
		}
	}
}
</style>
