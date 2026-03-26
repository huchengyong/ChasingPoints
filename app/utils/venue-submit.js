const trimValue = (value) => (typeof value === 'string' ? value.trim() : '')

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
