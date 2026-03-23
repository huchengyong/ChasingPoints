<template>
	<view class="submit-page">
		<view class="page-intro">
			<view class="intro-copy">
				<text class="intro-eyebrow">球馆提交</text>
				<text class="intro-title">完善球馆信息，方便大家了解和找到这里</text>
				<text class="intro-desc">球馆名称和详细地址为必填，其他信息可按实际情况补充。</text>
			</view>
			<view class="completion-card">
				<view class="completion-head">
					<text class="completion-title">提交准备度</text>
					<text class="completion-value">{{ completedRequiredCount }}/{{ requiredFieldOrder.length }}</text>
				</view>
				<view class="completion-track">
					<view class="completion-fill" :style="{ width: `${progressPercent}%` }"></view>
				</view>
				<text class="completion-tip">
					{{ missingRequiredKeys.length ? `还差 ${missingRequiredKeys.length} 项基础资料` : '基础资料已完成，可以提交并等待系统定位' }}
				</text>
			</view>
			<view class="intro-stats">
				<view class="stat-card">
					<text class="stat-label">城市</text>
					<text class="stat-value">{{ form.city || '待填写' }}</text>
				</view>
				<view class="stat-card">
					<text class="stat-label">定位</text>
					<text class="stat-value">系统解析</text>
				</view>
				<view class="stat-card">
					<text class="stat-label">状态</text>
					<text class="stat-value">异步整理</text>
				</view>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">基础资料</text>
				<text class="section-tip">先填写球馆名称和详细地址，方便大家准确找到这里。</text>
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

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">位置与联系</text>
				<text class="section-tip">系统会根据你填写的详细地址自动解析球馆位置，无需手动地图选点。</text>
			</view>
			<view class="form-group">
				<view class="label-row">
					<text class="form-label">定位方式</text>
					<text class="label-complete">自动处理</text>
				</view>
				<view class="location-picker">
					<view class="location-copy">
						<text class="location-title">系统将根据地址自动定位</text>
						<text class="location-subtitle">提交后后台会异步解析经纬度，并在整理完成后进入附近球馆。</text>
					</view>
					<uni-icons type="location" size="18" color="#0f766e"></uni-icons>
				</view>
			</view>

			<view class="form-group">
				<view class="label-row">
					<text class="form-label">联系电话</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.phone" placeholder="球馆联系电话" type="number" maxlength="15" />
				<text class="field-hint">建议填写前台或店长电话，方便到店前联系。</text>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">经营信息</text>
				<text class="section-tip">营业时间、球桌数量和价格信息能帮助大家更快做决定。</text>
			</view>
			<view class="form-group">
				<view class="label-row">
					<text class="form-label">营业时间</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.business_hours" placeholder="如：10:00-23:00" maxlength="30" />
				<text class="field-hint">尽量使用统一格式，查看起来更清楚。</text>
			</view>

			<view class="form-group">
				<view class="label-row">
					<text class="form-label">球桌数量</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.table_count" placeholder="球桌总数" type="number" />
				<text class="field-hint">填写后，大家更容易判断高峰期是否需要等位。</text>
			</view>

			<view class="form-group">
				<view class="label-row">
					<text class="form-label">台费范围</text>
					<text class="label-optional">选填</text>
				</view>
				<input class="form-input" v-model="form.price_range" placeholder="如：30-60元/小时" maxlength="30" />
				<text class="field-hint">可填写时段价或会员价区间，方便提前了解。</text>
			</view>
		</view>

		<view class="form-section">
			<view class="section-head">
				<text class="section-title">特色说明</text>
				<text class="section-tip">补充环境、服务或活动特色，方便大家提前了解球馆。</text>
			</view>
			<view class="form-group">
				<text class="form-label">球馆简介</text>
				<textarea
					class="form-textarea"
					v-model="form.description"
					placeholder="请简要介绍球馆特色..."
					maxlength="500"
					:auto-height="true"
				></textarea>
				<view class="field-meta">
					<text class="field-hint">可以写设备、包厢、停车、教学、赛事活动等亮点。</text>
					<text class="field-count">{{ form.description.length }}/500</text>
				</view>
			</view>
		</view>

		<view class="notice-card">
			<view class="notice-head">
				<uni-icons type="info" size="16" color="#0f766e"></uni-icons>
				<text class="notice-title">提交说明</text>
			</view>
			<text class="notice-text">提交后系统会先根据地址自动定位球馆位置，再整理资料并展示到球馆列表。请尽量保证名称、城市和详细地址准确一致。</text>
		</view>

		<view class="submit-bar">
			<view class="submit-copy">
				<text class="submit-title">{{ missingRequiredKeys.length ? `还有 ${missingRequiredKeys.length} 项未填写` : '确认信息后上传球馆' }}</text>
				<text class="submit-tip">{{ missingRequiredKeys.length ? `请先填写：${missingRequiredLabels.join('、')}` : '基础信息已完整，其他内容可以继续补充。' }}</text>
			</view>
			<view class="submit-btn" :class="{ disabled: submitting, pending: missingRequiredKeys.length > 0 }" @tap="handleSubmit">
				<text>{{ submitting ? '提交中...' : '上传球馆信息' }}</text>
			</view>
		</view>
	</view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { createVenue } from '@/api/venue.js'

const form = ref({
	name: '',
	city: '',
	district: '',
	address: '',
	phone: '',
	business_hours: '',
	table_count: '',
	price_range: '',
	description: ''
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
const completedRequiredCount = computed(() => requiredFieldOrder.length - missingRequiredKeys.value.length)
const progressPercent = computed(() => Math.round((completedRequiredCount.value / requiredFieldOrder.length) * 100))

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
		const data = {
			name: form.value.name.trim(),
			city: form.value.city.trim(),
			district: form.value.district.trim(),
			address: form.value.address.trim(),
			phone: form.value.phone.trim(),
			business_hours: form.value.business_hours.trim(),
			table_count: parseInt(form.value.table_count) || 0,
			price_range: form.value.price_range.trim(),
			description: form.value.description.trim()
		}
		const res = await createVenue(data)
		if (res.success) {
			uni.showToast({ title: res.message || '已提交，系统正在定位', icon: 'success' })
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
