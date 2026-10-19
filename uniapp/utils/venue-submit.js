import { formatRewardDuration, isFavoriteVenueRewardClaimable } from './favorite-venue-reward.js'

const trimValue = (value) => (typeof value === 'string' ? value.trim() : '')
const normalizeAreaId = (value) => {
  const areaId = Number(value)
  return Number.isFinite(areaId) && areaId > 0 ? areaId : 0
}

export const buildVenueRegionSelection = (areaPath = []) => {
  const normalizedPath = Array.isArray(areaPath)
    ? areaPath.filter(item => item && trimValue(item.name))
    : []
  const names = normalizedPath.map(item => trimValue(item.name)).filter(Boolean)

  return {
    areaIds: normalizedPath.map(item => normalizeAreaId(item.area_id)).filter(Boolean),
    regionText: names.join(' '),
    city: names[1] || '',
    district: names[2] || ''
  }
}

export const buildVenueSubmitPayload = (form = {}) => ({
  name: trimValue(form.name),
  city: trimValue(form.city),
  district: trimValue(form.district),
  address: trimValue(form.address)
})

// 球馆页只有在常玩球馆奖励可领取时才提VIP，否则只说明普通审核流程。
export const resolveVenueSubmitIntro = (rewardStatus = {}) => {
  if (isFavoriteVenueRewardClaimable(rewardStatus)) {
    return {
      title: `免费领取 ${formatRewardDuration(rewardStatus.reward_days || 30)}`,
      desc: '填写球馆真实的位置信息，审核通过后会自动发放VIP。虚假信息将不予通过。',
      sectionTip: '请填写球馆名称、地区和详细地址，审核通过后会自动发放VIP。'
    }
  }

  return {
    title: '提交常去球馆',
    desc: '填写球馆真实的位置信息，审核通过后会展示给附近球友。虚假信息将不予通过。',
    sectionTip: '请填写球馆名称、地区和详细地址，审核通过后会展示给附近球友。'
  }
}

export const resolveVenueSubmitCopy = (missingRequiredLabels = [], rewardStatus = {}) => {
  if (missingRequiredLabels.length > 0) {
    return {
      title: `还有 ${missingRequiredLabels.length} 项未填写`,
      tip: `请先填写：${missingRequiredLabels.join('、')}`
    }
  }

  return {
    title: '确认常玩球馆后提交',
    tip: isFavoriteVenueRewardClaimable(rewardStatus)
      ? `首次有效补充并审核通过，送 ${formatRewardDuration(rewardStatus.reward_days || 30)}。`
      : '审核通过后会展示给附近球友。'
  }
}
