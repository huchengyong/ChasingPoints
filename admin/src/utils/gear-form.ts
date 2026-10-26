// 装备目录后台的表单规则：与后端 admin_save_gear_product_logic.go / admin_save_gear_brand_logic.go 的校验逐条对应，
// 在提交前给出明确提示；最终仍以服务端校验为准。

export interface Option<T = string> {
  value: T
  label: string
}

export const GEAR_CATEGORY_OPTIONS: Option[] = [
  { value: 'cue', label: '球杆' },
  { value: 'break_jump', label: '冲杆 / 跳杆' },
  { value: 'tip', label: '皮头' },
  { value: 'chalk', label: '巧克' }
]

export const GEAR_CUE_STYLE_OPTIONS: Option[] = [
  { value: 'snooker', label: '斯诺克 / 中式杆' },
  { value: 'pool', label: '美式杆' }
]

export const GEAR_ROLE_OPTIONS: Option[] = [
  { value: 'break', label: '冲杆' },
  { value: 'jump', label: '跳杆' }
]

// 与用户端 UniApp 的 STATIC_ASSET_BASE_URL 一致：目录图上传到 CDN 后，这里只填路径。
export const GEAR_ASSET_BASE_URL = 'https://cdn.zhuifen.cn'

export const GEAR_BRAND_NAME_MAX = 64
export const GEAR_ALIASES_MAX = 255
export const GEAR_PRODUCT_NAME_MAX = 128
export const GEAR_LENGTH_MAX_MM = 3000
export const GEAR_WEIGHT_MAX_G = 2000
export const GEAR_TIP_DIAMETER_MAX_MM = 30

// 只接受自有 CDN 装备图目录下的 png 或 webp 文件名，不允许子目录和路径穿越。
const IMAGE_PATH_PATTERN = /^\/static\/images\/gear\/[A-Za-z0-9][A-Za-z0-9._-]*\.(png|webp)$/

export const isValidGearImagePath = (path: string): boolean => path === '' || IMAGE_PATH_PATTERN.test(path)

export const gearImageUrl = (path: string): string => (IMAGE_PATH_PATTERN.test(path) ? `${GEAR_ASSET_BASE_URL}${path}` : '')

const labelOf = (options: Option[], value: string): string => options.find((option) => option.value === value)?.label || value

export const gearCategoryLabel = (category: string): string => labelOf(GEAR_CATEGORY_OPTIONS, category)

export const formatGearRoles = (roles: string[] = []): string => {
  return GEAR_ROLE_OPTIONS.filter((option) => roles.includes(option.value)).map((option) => option.label).join(' · ')
}

export const formatGearCueStyle = (style: string): string => (style ? labelOf(GEAR_CUE_STYLE_OPTIONS, style) : '')

const charCount = (text: string): number => Array.from(text || '').length

export interface BrandForm {
  id: number
  name: string
  aliases: string
  sort: number
  status: number
}

export const defaultBrandForm = (): BrandForm => ({ id: 0, name: '', aliases: '', sort: 0, status: 1 })

// 别名统一分隔符、去掉空项与首尾空白，规则与后端 normalizeGearAliases 一致。
export const normalizeAliases = (raw: string): string => {
  return String(raw || '')
    .replace(/，/g, ',')
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
    .join(',')
}

export const validateBrandForm = (form: BrandForm): string => {
  const name = form.name.trim()
  if (!name || charCount(name) > GEAR_BRAND_NAME_MAX) return `品牌名称需为 1 到 ${GEAR_BRAND_NAME_MAX} 个字`
  if (charCount(normalizeAliases(form.aliases)) > GEAR_ALIASES_MAX) return `别名总长度不能超过 ${GEAR_ALIASES_MAX} 个字`
  if (form.status !== 0 && form.status !== 1) return '状态只能是启用或停用'
  return ''
}

export const buildBrandPayload = (form: BrandForm) => ({
  ...(form.id > 0 ? { id: form.id } : {}),
  name: form.name.trim(),
  aliases: normalizeAliases(form.aliases),
  sort: Number(form.sort) || 0,
  status: form.status
})

export interface ProductForm {
  id: number
  category: string
  // null 表示还没选品牌；0 表示球房公用条目，必须由管理员主动选择。
  brand_id: number | null
  name: string
  cue_style: string
  roles: string[]
  length_mm: number
  weight_g: number
  tip_diameter_mm: number
  image_path: string
  sort: number
  status: number
}

export const defaultProductForm = (category = 'cue'): ProductForm => ({
  id: 0, category, brand_id: null, name: '', cue_style: '', roles: [], length_mm: 0, weight_g: 0, tip_diameter_mm: 0,
  image_path: '', sort: 0, status: 1
})

// 品牌为 0 表示球房公用条目，只有球杆和巧克可以没有品牌。
export const allowsHouseEntry = (category: string): boolean => category === 'cue' || category === 'chalk'

export const validateProductForm = (form: ProductForm): string => {
  if (!GEAR_CATEGORY_OPTIONS.some((option) => option.value === form.category)) return '请选择装备类别'
  const name = form.name.trim()
  if (!name || charCount(name) > GEAR_PRODUCT_NAME_MAX) return `型号名称需为 1 到 ${GEAR_PRODUCT_NAME_MAX} 个字`
  if (form.status !== 0 && form.status !== 1) return '状态只能是启用或停用'
  if (form.brand_id === null) return '请选择品牌；球房公用条目请选「无品牌」'
  if (form.brand_id < 0) return '品牌无效'
  const house = form.brand_id === 0
  if (house && !allowsHouseEntry(form.category)) return '只有球杆和巧克可以是球房公用条目（不选品牌）'

  if (form.category === 'cue') {
    if (!house && !GEAR_CUE_STYLE_OPTIONS.some((option) => option.value === form.cue_style)) return '请选择球杆杆型'
    if (form.roles.length) return '球杆不能设置冲跳身份'
  } else if (form.category === 'break_jump') {
    if (!form.roles.length || !form.roles.every((role) => GEAR_ROLE_OPTIONS.some((option) => option.value === role))) {
      return '请至少选择冲杆或跳杆身份'
    }
    if (form.cue_style) return '冲杆和跳杆不设置杆型'
  } else if (form.cue_style || form.roles.length) {
    return '皮头和巧克不设置杆型与身份'
  }

  if (!(form.length_mm >= 0 && form.length_mm <= GEAR_LENGTH_MAX_MM)) return `长度需在 0 到 ${GEAR_LENGTH_MAX_MM} 毫米之间，0 表示未提供`
  if (!(form.weight_g >= 0 && form.weight_g <= GEAR_WEIGHT_MAX_G)) return `重量需在 0 到 ${GEAR_WEIGHT_MAX_G} 克之间，0 表示未提供`
  if (!(form.tip_diameter_mm >= 0 && form.tip_diameter_mm <= GEAR_TIP_DIAMETER_MAX_MM)) {
    return `皮头直径需在 0 到 ${GEAR_TIP_DIAMETER_MAX_MM} 毫米之间，0 表示未提供`
  }
  if (!isValidGearImagePath(form.image_path.trim())) return '目录图只能使用 /static/images/gear/ 下的 png 或 webp 文件'
  return ''
}

// 请求体只带与类别相关的字段：杆型只属于球杆，身份只属于冲跳装备。
export const buildProductPayload = (form: ProductForm) => ({
  ...(form.id > 0 ? { id: form.id } : {}),
  category: form.category,
  brand_id: form.brand_id as number, // 提交前已由 validateProductForm 保证不为 null
  name: form.name.trim(),
  ...(form.category === 'cue' && form.cue_style ? { cue_style: form.cue_style } : {}),
  ...(form.category === 'break_jump' ? { roles: GEAR_ROLE_OPTIONS.map((option) => option.value).filter((role) => form.roles.includes(role)) } : {}),
  length_mm: Math.round(Number(form.length_mm) || 0),
  weight_g: Math.round(Number(form.weight_g) || 0),
  tip_diameter_mm: Math.round((Number(form.tip_diameter_mm) || 0) * 100) / 100,
  image_path: form.image_path.trim(),
  sort: Number(form.sort) || 0,
  status: form.status
})

// 编辑时从列表行还原表单；类别不能改（已有用户装备引用它）。
export const productToForm = (product: {
  id: number; category: string; brand_id: number; name: string; cue_style: string; roles: string[]; length_mm: number;
  weight_g: number; tip_diameter_mm: number; image_path: string; sort: number; status: number
}): ProductForm => ({
  id: product.id,
  category: product.category,
  brand_id: product.brand_id,
  name: product.name,
  cue_style: product.cue_style || '',
  roles: [...(product.roles || [])],
  length_mm: product.length_mm || 0,
  weight_g: product.weight_g || 0,
  tip_diameter_mm: product.tip_diameter_mm || 0,
  image_path: product.image_path || '',
  sort: product.sort || 0,
  status: product.status
})

// 关联确认弹窗的提示：说明会改动多少位用户的装备，并提醒这是一次批量操作。
export const describeLinkConfirm = (group: { brand_name: string; model_name: string; count: number }, productTitle: string): string => {
  return `将「${group.brand_name} ${group.model_name}」的 ${group.count} 条用户登记关联到「${productTitle}」。关联后这些用户的装备会显示为该型号，历史对局也会随之更新。`
}

export const productTitle = (product: { brand_name: string; name: string }): string => {
  return [product.brand_name, product.name].map((part) => String(part || '').trim()).filter(Boolean).join(' ')
}
