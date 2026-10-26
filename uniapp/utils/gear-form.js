// 登记未收录型号的表单校验与请求体构造（纯函数）。长度按字符数计，与后端按 rune 计数一致。
export const GEAR_BRAND_NAME_MAX = 32
export const GEAR_MODEL_NAME_MAX = 64

export const CUE_STYLE_OPTIONS = Object.freeze([
  { value: 'snooker', label: '斯诺克 / 中式杆' },
  { value: 'pool', label: '美式杆' }
])

export const ROLE_OPTIONS = Object.freeze([
  { value: 'break', label: '冲杆' },
  { value: 'jump', label: '跳杆' }
])

const charCount = (text) => Array.from(String(text || '')).length

const toggleValue = (list, value) => {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value]
}

export const toggleGearRole = (roles = [], role) => {
  const next = toggleValue(roles, role)
  // 固定顺序，和后端存储一致
  return ROLE_OPTIONS.map((option) => option.value).filter((value) => next.includes(value))
}

/**
 * form: { category, brandId, brandName, modelName, cueStyle, roles }
 * brandId > 0 表示选了目录品牌；否则必须填写自填品牌名。
 * 返回 { ok: true, payload } 或 { ok: false, message }。
 */
export const buildUnlistedGearPayload = (form = {}) => {
  const category = form.category
  if (category !== 'cue' && category !== 'break_jump') {
    return { ok: false, message: '装备类别无效' }
  }

  const modelName = String(form.modelName || '').trim()
  if (!modelName) return { ok: false, message: '请填写型号名称' }
  if (charCount(modelName) > GEAR_MODEL_NAME_MAX) {
    return { ok: false, message: `型号名称不能超过 ${GEAR_MODEL_NAME_MAX} 个字` }
  }

  const brandId = Number(form.brandId) > 0 ? Number(form.brandId) : 0
  const brandName = brandId ? '' : String(form.brandName || '').trim()
  if (!brandId) {
    if (!brandName) return { ok: false, message: '请选择品牌或填写品牌名称' }
    if (charCount(brandName) > GEAR_BRAND_NAME_MAX) {
      return { ok: false, message: `品牌名称不能超过 ${GEAR_BRAND_NAME_MAX} 个字` }
    }
  }

  const payload = { category, model_name: modelName }
  if (brandId) {
    payload.brand_id = brandId
  } else {
    payload.brand_name = brandName
  }

  if (category === 'cue') {
    if (!CUE_STYLE_OPTIONS.some((option) => option.value === form.cueStyle)) {
      return { ok: false, message: '请选择杆型' }
    }
    payload.cue_style = form.cueStyle
  } else {
    const roles = ROLE_OPTIONS.map((option) => option.value).filter((value) => (form.roles || []).includes(value))
    if (!roles.length) return { ok: false, message: '请选择冲杆或跳杆' }
    payload.roles = roles
  }
  return { ok: true, payload }
}
