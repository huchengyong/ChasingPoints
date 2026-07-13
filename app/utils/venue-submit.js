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

export const resolveVenueSubmitCopy = (missingRequiredLabels = []) => {
  if (missingRequiredLabels.length > 0) {
    return {
      title: `还有 ${missingRequiredLabels.length} 项未填写`,
      tip: `请先填写：${missingRequiredLabels.join('、')}`
    }
  }

  return {
    title: '确认常玩球馆后提交',
    tip: '首次有效补充并审核通过，送 1 个月会员。'
  }
}
