<template>
	<view class="create-page" :class="{ 'dark-mode': isDarkMode }">
		<view class="page-intro">
			<view class="intro-copy">
				<text class="intro-eyebrow">赛事创建</text>
				<text class="intro-title">把关键信息一次填清楚，报名体验会更顺</text>
				<text class="intro-desc">先完成基础信息，场地和补充说明可按实际情况完善。</text>
			</view>
			<view class="completion-card">
				<view class="completion-head">
					<text class="completion-title">必填进度</text>
					<text class="completion-value">{{ completedRequiredCount }}/{{ requiredFieldOrder.length }}</text>
				</view>
				<view class="completion-track">
					<view class="completion-fill" :style="{ width: `${progressPercent}%` }"></view>
				</view>
				<text class="completion-tip">
					{{ missingRequiredKeys.length ? `还差 ${missingRequiredKeys.length} 项必填信息` : '必填信息已完成，可以直接创建赛事' }}
				</text>
			</view>
			<view class="intro-stats">
				<view class="stat-card">
					<text class="stat-label">球种</text>
					<text class="stat-value">{{ form.game_type ? gameTypeLabel : '待选择' }}</text>
				</view>
				<view class="stat-card">
					<text class="stat-label">赛制</text>
					<text class="stat-value">{{ form.format ? formatLabel : '待选择' }}</text>
				</view>
				<view class="stat-card">
					<text class="stat-label">人数</text>
					<text class="stat-value">{{ form.max_players }}人</text>
				</view>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">基础信息</text>
				<text class="section-tip">填写这些信息后，参赛者会更容易了解赛事安排</text>
			</view>
			<view class="form-item" :class="getFieldClass('name')">
				<view class="label-row">
					<text class="form-label">赛事名称</text>
					<text :class="requiredFieldStatus.name ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.name ? '已完成' : '必填' }}</text>
				</view>
				<input
					class="form-input"
					:class="{ 'input-missing': showValidation && !requiredFieldStatus.name }"
					v-model="form.name"
					placeholder="例如：周末公开赛 · 第 3 站"
					maxlength="50"
				/>
				<view class="field-meta">
					<text class="field-hint">建议带上场次或主题，方便大家快速识别。</text>
					<text class="field-count">{{ form.name.length }}/50</text>
				</view>
			</view>
			<view class="form-item" :class="getFieldClass('game_type')">
				<view class="label-row">
					<text class="form-label">球种</text>
					<text :class="requiredFieldStatus.game_type ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.game_type ? '已完成' : '必填' }}</text>
				</view>
				<picker :range="gameTypes" range-key="label" @change="onGameType">
					<view class="form-picker form-picker-card" :class="{ 'input-missing': showValidation && !requiredFieldStatus.game_type }">
						<view class="picker-copy">
							<text class="picker-title" :class="{ placeholder: !form.game_type }">{{ form.game_type ? gameTypeLabel : '请选择球种' }}</text>
							<text class="picker-desc">方便大家快速了解赛事类型</text>
						</view>
						<uni-icons type="right" size="16" color="#9A8C67"></uni-icons>
					</view>
				</picker>
			</view>
			<view class="form-item" :class="getFieldClass('format')">
				<view class="label-row">
					<text class="form-label">赛制</text>
					<text :class="requiredFieldStatus.format ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.format ? '已完成' : '必填' }}</text>
				</view>
				<picker :range="formats" range-key="label" @change="onFormat">
					<view class="form-picker form-picker-card" :class="{ 'input-missing': showValidation && !requiredFieldStatus.format }">
						<view class="picker-copy">
							<text class="picker-title" :class="{ placeholder: !form.format }">{{ form.format ? formatLabel : '请选择赛制' }}</text>
							<text class="picker-desc">不同赛制会影响参赛人数和节奏</text>
						</view>
						<uni-icons type="right" size="16" color="#9A8C67"></uni-icons>
					</view>
				</picker>
			</view>
			<view class="form-item">
				<view class="label-row">
					<text class="form-label">人数上限</text>
					<text class="label-complete">已完成</text>
				</view>
				<picker :range="playerOptions" @change="onPlayers">
					<view class="form-picker form-picker-card">
						<view class="picker-copy">
							<text class="picker-title">{{ form.max_players }}人</text>
							<text class="picker-desc">建议与赛制和场地承载能力匹配</text>
						</view>
						<uni-icons type="right" size="16" color="#9A8C67"></uni-icons>
					</view>
				</picker>
			</view>
			<view class="form-item" :class="getFieldClass('start_time')">
				<view class="label-row">
					<text class="form-label">开始时间</text>
					<text :class="requiredFieldStatus.start_time ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.start_time ? '已完成' : '必填' }}</text>
				</view>
				<picker mode="date" @change="onDate">
					<view class="form-picker form-picker-card" :class="{ 'input-missing': showValidation && !requiredFieldStatus.start_time }">
						<view class="picker-copy">
							<text class="picker-title" :class="{ placeholder: !form.start_time }">{{ form.start_time || '请选择日期' }}</text>
							<text class="picker-desc">建议至少提前 1 天发布，方便报名和传播</text>
						</view>
						<uni-icons type="right" size="16" color="#9A8C67"></uni-icons>
					</view>
				</picker>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">场地信息</text>
				<text class="section-tip">补充举办城市和场馆后，参赛者更容易安排行程</text>
			</view>
			<view class="form-item">
				<view class="label-row">
					<text class="form-label">城市</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.city" placeholder="例如：上海" maxlength="20" />
				<text class="field-hint">填写后会展示在赛事信息中。</text>
			</view>
			<view class="form-item">
				<view class="label-row">
					<text class="form-label">场馆名称</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.venue_name" placeholder="例如：XX 台球俱乐部" maxlength="50" />
				<text class="field-hint">填清楚场馆，参赛者更容易判断出行距离。</text>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">补充说明</text>
				<text class="section-tip">写清报名规则、奖励或赛程提醒，能减少重复沟通</text>
			</view>
			<view class="form-item">
				<text class="form-label">赛事描述</text>
				<textarea
					class="form-textarea"
					v-model="form.description"
					placeholder="可填写报名要求、费用、奖励、赛程安排等"
					maxlength="500"
				/>
				<view class="field-meta">
					<text class="field-hint">写得更清楚，大家报名和了解规则会更方便。</text>
					<text class="field-count">{{ form.description.length }}/500</text>
				</view>
			</view>
		</view>

		<view class="submit-bar">
			<view class="submit-copy">
				<text class="submit-title">{{ missingRequiredKeys.length ? `还有 ${missingRequiredKeys.length} 项未填写` : '确认无误后发布赛事' }}</text>
				<text class="submit-tip">{{ missingRequiredKeys.length ? `请先填写：${missingRequiredLabels.join('、')}` : '信息确认后即可创建赛事，创建成功后会返回上一页。' }}</text>
			</view>
			<view class="submit-btn" :class="{ disabled: submitting, pending: missingRequiredKeys.length > 0 }" @tap="onSubmit">
				<text>{{ submitting ? '创建中...' : '创建赛事' }}</text>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { createTournament } from '@/api/tournament.js'
import { GAME_TYPE_OPTIONS } from '@/utils/game-types.js'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

const gameTypes = ref(GAME_TYPE_OPTIONS)
const formats = ref([
	{ label: '单败淘汰', value: 1 },
	{ label: '循环赛', value: 3 }
])
const playerOptions = ref(['4', '8', '16', '32', '64'])

const form = ref({
	name: '',
	game_type: 0,
	format: 0,
	max_players: 8,
	start_time: '',
	city: '',
	venue_name: '',
	description: ''
})
const submitting = ref(false)
const showValidation = ref(false)
const requiredFieldOrder = ['name', 'game_type', 'format', 'start_time']
const requiredFieldLabelMap = {
	name: '赛事名称',
	game_type: '球种',
	format: '赛制',
	start_time: '开始时间'
}

const gameTypeLabel = computed(() => {
	const item = gameTypes.value.find(t => t.value === form.value.game_type)
	return item ? item.label : ''
})
const formatLabel = computed(() => {
	const item = formats.value.find(t => t.value === form.value.format)
	return item ? item.label : ''
})
const requiredFieldStatus = computed(() => ({
	name: !!form.value.name.trim(),
	game_type: !!form.value.game_type,
	format: !!form.value.format,
	start_time: !!form.value.start_time
}))
const missingRequiredKeys = computed(() => requiredFieldOrder.filter(key => !requiredFieldStatus.value[key]))
const missingRequiredLabels = computed(() => missingRequiredKeys.value.map(key => requiredFieldLabelMap[key]))
const completedRequiredCount = computed(() => requiredFieldOrder.length - missingRequiredKeys.value.length)
const progressPercent = computed(() => Math.round((completedRequiredCount.value / requiredFieldOrder.length) * 100))

const onGameType = (e) => { form.value.game_type = gameTypes.value[e.detail.value].value }
const onFormat = (e) => { form.value.format = formats.value[e.detail.value].value }
const onPlayers = (e) => { form.value.max_players = parseInt(playerOptions.value[e.detail.value]) }
const onDate = (e) => { form.value.start_time = e.detail.value }
const getFieldClass = (key) => ({
	'is-complete': requiredFieldStatus.value[key],
	'is-missing': showValidation.value && !requiredFieldStatus.value[key]
})
const validate = () => {
	showValidation.value = true
	if (!requiredFieldStatus.value.name) return '请输入赛事名称'
	if (!requiredFieldStatus.value.game_type) return '请选择球种'
	if (!requiredFieldStatus.value.format) return '请选择赛制'
	if (!requiredFieldStatus.value.start_time) return '请选择开始时间'
	return ''
}

const onSubmit = async () => {
	if (submitting.value) return
	const errorMessage = validate()
	if (errorMessage) return uni.showToast({ title: errorMessage, icon: 'none' })

	submitting.value = true
	try {
		const res = await createTournament(form.value)
		if (res.success) {
			uni.showToast({ title: '创建成功', icon: 'success' })
			setTimeout(() => {
				uni.navigateBack()
			}, 1000)
		} else {
			uni.showToast({ title: '创建失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '创建失败', icon: 'none' })
	} finally {
		submitting.value = false
	}
}
</script>

<style lang="scss" scoped>
@import './create.scss';
</style>
