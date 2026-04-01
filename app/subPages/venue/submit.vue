<template>
	<view class="submit-page">
		<view class="page-intro">
			<view class="intro-copy">
				<text class="intro-eyebrow">常玩球馆</text>
				<text class="intro-title">提交常玩的球馆，领取 1 个月会员</text>
				<text class="intro-desc">填写基础资料，要求真实信息，后台审核通过后会自动发放会员。虚假信息将不予通过。</text>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">常玩球馆资料</text>
				<text class="section-tip">请填写球馆名称、地区和详细地址，审核通过后会自动发放会员。</text>
			</view>
			<view class="form-group" :class="getFieldClass('name')">
				<view class="label-row">
					<text class="form-label">球馆名称</text>
					<text :class="requiredFieldStatus.name ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.name ? '已完成' : '必填' }}</text>
				</view>
				<input class="form-input" :class="{ 'input-missing': showValidation && !requiredFieldStatus.name }" v-model="form.name" placeholder="例如：星轨台球俱乐部" maxlength="50" />
				<view class="field-meta">
					<text class="field-hint">建议填写门店常用名称，方便大家识别。</text>
					<text class="field-count">{{ form.name.length }}/50</text>
				</view>
			</view>

			<view class="form-group" :class="getFieldClass('region')">
				<view class="label-row">
					<text class="form-label">地区</text>
					<text :class="requiredFieldStatus.region ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.region ? '已完成' : '必填' }}</text>
				</view>
				<picker
					mode="multiSelector"
					:range="areaColumns"
					range-key="name"
					:value="areaColumnIndexes"
					:disabled="submitting || (areaLoading && !areaReady)"
					@change="handleAreaConfirm"
					@columnchange="handleAreaColumnChange"
				>
					<view class="form-input form-picker" :class="{ 'input-missing': showValidation && !requiredFieldStatus.region, 'is-placeholder': !form.regionText }">
						<text class="picker-value">{{ form.regionText || (areaLoading && !areaReady ? '地区加载中...' : '请选择省 / 市 / 区') }}</text>
						<text class="picker-arrow">›</text>
					</view>
				</picker>
				<text class="field-hint">选择省、市、区后，审核和定位都会更准确。</text>
			</view>

			<view class="form-group" :class="getFieldClass('address')">
				<view class="label-row">
					<text class="form-label">详细地址</text>
					<text :class="requiredFieldStatus.address ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.address ? '已完成' : '必填' }}</text>
				</view>
				<input class="form-input" :class="{ 'input-missing': showValidation && !requiredFieldStatus.address }" v-model="form.address" placeholder="请输入门牌号、楼层等详细地址" maxlength="100" />
				<view class="field-meta">
					<text class="field-hint">建议包含楼层或门牌，首次到店更容易找到。</text>
					<text class="field-count">{{ form.address.length }}/100</text>
				</view>
			</view>
		</view>

			<view class="submit-bar">
			<view class="submit-copy">
				<text class="submit-title">{{ submitBarCopy.title }}</text>
				<text class="submit-tip">{{ submitBarCopy.tip }}</text>
			</view>
			<view class="submit-btn" :class="{ disabled: submitting, pending: missingRequiredKeys.length > 0 }" @tap="handleSubmit">
				<text>{{ submitting ? '提交中...' : '提交常玩球馆' }}</text>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { createVenue, getVenueAreaOptions } from '@/api/venue.js'
import {
	buildVenueRegionSelection,
	buildVenueSubmitPayload,
	resolveVenueSubmitCopy
} from '@/utils/venue-submit.js'

const createAreaPlaceholderOption = (name = '暂无数据') => ({
	area_id: 0,
	parent_id: 0,
	name,
	disabled: true
})

const areaOptionsCache = new Map()
let areaHydrateToken = 0

const form = ref({
	name: '',
	regionText: '',
	areaIds: [],
	city: '',
	district: '',
	address: ''
})

const submitting = ref(false)
const showValidation = ref(false)
const areaLoading = ref(false)
const areaReady = ref(false)
const areaColumns = ref([
	[createAreaPlaceholderOption('地区加载中...')],
	[createAreaPlaceholderOption('请选择城市')],
	[createAreaPlaceholderOption('请选择区域')]
])
const areaColumnIndexes = ref([0, 0, 0])
const requiredFieldOrder = ['name', 'region', 'address']
const requiredFieldLabelMap = {
	name: '球馆名称',
	region: '地区',
	address: '详细地址'
}
const requiredFieldStatus = computed(() => ({
	name: !!form.value.name.trim(),
	region: !!form.value.regionText.trim(),
	address: !!form.value.address.trim()
}))
const missingRequiredKeys = computed(() => requiredFieldOrder.filter(key => !requiredFieldStatus.value[key]))
const missingRequiredLabels = computed(() => missingRequiredKeys.value.map(key => requiredFieldLabelMap[key]))
const submitBarCopy = computed(() => resolveVenueSubmitCopy(missingRequiredLabels.value))

const getFieldClass = (key) => ({
	'is-complete': requiredFieldStatus.value[key],
	'is-missing': showValidation.value && !requiredFieldStatus.value[key]
})

const validate = () => {
	showValidation.value = true
	if (!requiredFieldStatus.value.name) return '请输入球馆名称'
	if (!requiredFieldStatus.value.region) return '请选择地区'
	if (!requiredFieldStatus.value.address) return '请输入详细地址'
	return ''
}

const normalizeAreaOptions = (list, emptyLabel) => {
	if (Array.isArray(list) && list.length > 0) {
		const normalizedList = list
			.map(item => ({
				area_id: Number(item.area_id || 0),
				parent_id: Number(item.parent_id || 0),
				name: typeof item.name === 'string' ? item.name.trim() : ''
			}))
			.filter(item => item.area_id > 0 && item.name)

		if (normalizedList.length > 0) {
			return normalizedList
		}
	}

	return [createAreaPlaceholderOption(emptyLabel)]
}

const clampAreaIndex = (index, options) => {
	if (!Array.isArray(options) || options.length === 0) return 0

	const normalizedIndex = Number(index)
	if (!Number.isInteger(normalizedIndex) || normalizedIndex < 0) {
		return 0
	}

	return Math.min(normalizedIndex, options.length - 1)
}

const fetchAreaOptions = async (parentId = 0) => {
	const cacheKey = String(parentId)
	if (areaOptionsCache.has(cacheKey)) {
		return areaOptionsCache.get(cacheKey)
	}

	const res = await getVenueAreaOptions({ parent_id: parentId })
	const list = normalizeAreaOptions(res?.list, parentId === 0 ? '暂无地区数据' : '暂无下级地区')
	areaOptionsCache.set(cacheKey, list)
	return list
}

const getCurrentAreaPath = (indexes = areaColumnIndexes.value) => indexes
	.map((index, column) => areaColumns.value[column]?.[index])
	.filter(item => item && item.area_id > 0)

const applyAreaSelection = (selection) => {
	form.value.regionText = selection.regionText
	form.value.areaIds = selection.areaIds
	form.value.city = selection.city
	form.value.district = selection.district
}

const hydrateAreaColumns = async (indexes = [0, 0, 0]) => {
	const currentToken = ++areaHydrateToken
	areaLoading.value = true

	try {
		const provinces = await fetchAreaOptions(0)
		const provinceIndex = clampAreaIndex(indexes[0], provinces)
		const province = provinces[provinceIndex]

		const cities = province?.area_id > 0
			? await fetchAreaOptions(province.area_id)
			: [createAreaPlaceholderOption('请选择城市')]
		const cityIndex = clampAreaIndex(indexes[1], cities)
		const city = cities[cityIndex]

		const districts = city?.area_id > 0
			? await fetchAreaOptions(city.area_id)
			: [createAreaPlaceholderOption('请选择区域')]
		const districtIndex = clampAreaIndex(indexes[2], districts)

		if (currentToken !== areaHydrateToken) return

		areaColumns.value = [provinces, cities, districts]
		areaColumnIndexes.value = [provinceIndex, cityIndex, districtIndex]
		areaReady.value = provinces.some(item => item.area_id > 0)
	} catch (error) {
		if (currentToken !== areaHydrateToken) return

		areaReady.value = false
		uni.showToast({ title: '地区数据加载失败', icon: 'none' })
	} finally {
		if (currentToken === areaHydrateToken) {
			areaLoading.value = false
		}
	}
}

const handleAreaColumnChange = async (event) => {
	const nextIndexes = [...areaColumnIndexes.value]
	nextIndexes[event.detail.column] = event.detail.value

	if (event.detail.column === 0) {
		nextIndexes[1] = 0
		nextIndexes[2] = 0
	}
	if (event.detail.column === 1) {
		nextIndexes[2] = 0
	}

	await hydrateAreaColumns(nextIndexes)
}

const handleAreaConfirm = async (event) => {
	await hydrateAreaColumns(event.detail.value)
	const selection = buildVenueRegionSelection(getCurrentAreaPath())

	if (!selection.city) {
		return uni.showToast({ title: '请选择完整地区', icon: 'none' })
	}

	applyAreaSelection(selection)
}

const handleSubmit = async () => {
	if (submitting.value) return
	const errorMessage = validate()
	if (errorMessage) return uni.showToast({ title: errorMessage, icon: 'none' })

	submitting.value = true
	try {
		const data = buildVenueSubmitPayload(form.value)
		const res = await createVenue(data)
		if (res.success) {
			uni.showToast({ title: res.message || '已提交，审核通过后会员将自动到账', icon: 'success' })
			setTimeout(() => {
				uni.navigateBack()
			}, 1500)
		} else {
			uni.showToast({ title: res.message || res.msg || '提交失败', icon: 'none' })
		}
	} catch (e) {
		uni.showToast({ title: '提交失败', icon: 'none' })
	} finally {
		submitting.value = false
	}
}

onMounted(() => {
	hydrateAreaColumns().catch(() => {})
})
</script>

<style lang="scss" scoped>
@import './submit.scss';
</style>
