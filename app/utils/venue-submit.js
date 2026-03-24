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
    title: '确认基础资料后上传球馆',
    tip: '提交后系统会根据地址自动定位并等待审核。'
  }
}
