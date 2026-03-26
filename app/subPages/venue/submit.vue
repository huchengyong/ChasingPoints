<template>
	<view class="submit-page">
		<view class="page-intro">
			<view class="intro-copy">
				<text class="intro-eyebrow">常玩球馆</text>
				<text class="intro-title">补充你常玩的球馆，后续约球和签到更方便</text>
				<text class="intro-desc">首次有效补充并审核通过后，送 1 个月会员。填写基础资料即可提交。</text>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">常玩球馆资料</text>
				<text class="section-tip">请填写球馆名称、城市和详细地址，审核通过后会自动发放会员。</text>
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

			<view class="form-group" :class="getFieldClass('city')">
				<view class="label-row">
					<text class="form-label">城市</text>
					<text :class="requiredFieldStatus.city ? 'label-complete' : 'label-required'">{{ requiredFieldStatus.city ? '已完成' : '必填' }}</text>
				</view>
				<input class="form-input" :class="{ 'input-missing': showValidation && !requiredFieldStatus.city }" v-model="form.city" placeholder="如：深圳" maxlength="20" />
				<text class="field-hint">填写后更方便大家按城市查找球馆。</text>
			</view>

			<view class="form-group">
				<view class="label-row">
					<text class="form-label">区域</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.district" placeholder="如：南山区" maxlength="20" />
				<text class="field-hint">填写商圈或区县后，用户更容易判断距离。</text>
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
import { ref, computed } from 'vue'
import { createVenue } from '@/api/venue.js'
import { buildVenueSubmitPayload, resolveVenueSubmitCopy } from '@/utils/venue-submit.js'

const form = ref({
	name: '',
	city: '',
	district: '',
	address: ''
})

const submitting = ref(false)
const showValidation = ref(false)
const requiredFieldOrder = ['name', 'city', 'address']
const requiredFieldLabelMap = {
	name: '球馆名称',
	city: '城市',
	address: '详细地址'
}
const requiredFieldStatus = computed(() => ({
	name: !!form.value.name.trim(),
	city: !!form.value.city.trim(),
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
	if (!requiredFieldStatus.value.city) return '请输入城市'
	if (!requiredFieldStatus.value.address) return '请输入详细地址'
	return ''
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
</script>

<style lang="scss" scoped>
@import './submit.scss';
</style>
