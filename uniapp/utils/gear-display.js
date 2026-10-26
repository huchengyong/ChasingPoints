import { isCdnAssetPath, resolveCdnAssetUrl } from './cdn-asset.js'
import { ORDERED_GAME_TYPES } from './game-types.js'

// 装备展示规则：默认图选择、目录图校验、规格格式化、分组与主力设置视图。
// 全部是纯函数，不依赖 uni，页面与组件只负责渲染。

export const GEAR_CATEGORY = Object.freeze({
  CUE: 'cue',
  BREAK_JUMP: 'break_jump',
  TIP: 'tip',
  CHALK: 'chalk'
})

export const GEAR_ITEM_STATUS = Object.freeze({ ACTIVE: 1, RETIRED: 2, DELETED: 3 })

export const GEAR_SLOT = Object.freeze({ CUE: 'cue', BREAK: 'break' })

export const GEAR_MAX_ITEMS = 30

export const GEAR_CATEGORY_LABELS = Object.freeze({
  cue: '球杆',
  break_jump: '冲杆 / 跳杆',
  tip: '皮头',
  chalk: '巧克'
})

export const UNLISTED_MODEL_NAME = '未收录型号'

export const GEAR_DEFAULT_IMAGE_PATHS = Object.freeze({
  snookerCue: '/static/images/gear/gear-cue-snooker-default.png',
  poolCue: '/static/images/gear/gear-cue-pool-default.png',
  houseCue: '/static/images/gear/gear-house-cue-default.png',
  breakCue: '/static/images/gear/gear-break-cue-default.png',
  jumpCue: '/static/images/gear/gear-jump-cue-default.png',
  tip: '/static/images/gear/gear-tip-default.png',
  chalk: '/static/images/gear/gear-chalk-default.png'
})

// 后台只允许这个形态的目录图路径；客户端同样校验，不合规一律回退默认图，不信任服务端返回的任意地址。
const GEAR_IMAGE_PATH_PATTERN = /^\/static\/images\/gear\/[A-Za-z0-9][A-Za-z0-9._-]*\.(?:png|webp)$/

export const isGearImagePath = (path) => {
  return typeof path === 'string' && GEAR_IMAGE_PATH_PATTERN.test(path) && isCdnAssetPath(path)
}

const hasRole = (roles, role) => Array.isArray(roles) && roles.includes(role)

/**
 * 按类别选默认图：球房公杆用公杆图；其他球杆按杆型；冲跳装备带冲杆身份时用冲杆图，否则用跳杆图。
 */
export const resolveGearDefaultImagePath = ({ category, house = false, cueStyle = '', roles = [] } = {}) => {
  if (category === GEAR_CATEGORY.TIP) return GEAR_DEFAULT_IMAGE_PATHS.tip
  if (category === GEAR_CATEGORY.CHALK) return GEAR_DEFAULT_IMAGE_PATHS.chalk
  if (category === GEAR_CATEGORY.BREAK_JUMP) {
    return hasRole(roles, 'break') ? GEAR_DEFAULT_IMAGE_PATHS.breakCue : GEAR_DEFAULT_IMAGE_PATHS.jumpCue
  }
  if (house) return GEAR_DEFAULT_IMAGE_PATHS.houseCue
  return cueStyle === 'pool' ? GEAR_DEFAULT_IMAGE_PATHS.poolCue : GEAR_DEFAULT_IMAGE_PATHS.snookerCue
}

/**
 * 解析一件装备或一个目录型号的图片。subject 可以是 GearItemInfo（图片在 product.image_path）或 GearProductInfo。
 * 返回 { src, fallback }：src 是首选地址，加载失败时改用 fallback（按类别的默认图）。
 */
export const resolveGearImage = (subject) => {
  const source = subject || {}
  const product = source.product || (source.image_path !== undefined ? source : null)
  const fallbackPath = resolveGearDefaultImagePath({
    category: source.category,
    house: Boolean(source.house),
    cueStyle: source.cue_style,
    roles: source.roles
  })
  const imagePath = isGearImagePath(product?.image_path) ? product.image_path : ''
  return {
    src: resolveCdnAssetUrl(imagePath || fallbackPath),
    fallback: resolveCdnAssetUrl(fallbackPath)
  }
}

const positive = (value) => Number.isFinite(Number(value)) && Number(value) > 0

const trimNumber = (value) => String(Number(Number(value).toFixed(2)))

/**
 * 官方规格：品牌没有提供的项目不展示（后端用 0 表示未提供），重量同时给出克与盎司。
 */
export const formatGearSpecs = (product) => {
  if (!product) return []
  const specs = []
  if (positive(product.length_mm)) {
    specs.push({ key: 'length', label: '长度', value: `${Math.round(Number(product.length_mm))} mm` })
  }
  if (positive(product.weight_g)) {
    const grams = Math.round(Number(product.weight_g))
    specs.push({ key: 'weight', label: '重量', value: `${grams} g（${(grams / 28.3495).toFixed(1)} oz）` })
  }
  if (positive(product.tip_diameter_mm)) {
    specs.push({ key: 'tipDiameter', label: '皮头直径', value: `${trimNumber(product.tip_diameter_mm)} mm` })
  }
  return specs
}

export const formatGearRoles = (roles) => {
  const labels = []
  if (hasRole(roles, 'break')) labels.push('冲杆')
  if (hasRole(roles, 'jump')) labels.push('跳杆')
  return labels.join(' · ')
}

export const formatGearCueStyle = (cueStyle) => {
  if (cueStyle === 'pool') return '美式杆'
  if (cueStyle === 'snooker') return '斯诺克 / 中式杆'
  return ''
}

/**
 * 装备标题：品牌 + 型号；球房公用条目没有品牌，只显示型号名。
 */
export const gearItemTitle = (item) => {
  if (!item) return ''
  const parts = [item.brand_name, item.model_name].map((part) => String(part || '').trim()).filter(Boolean)
  return parts.join(' ') || '未命名装备'
}

/**
 * 装备副标题：球杆显示杆型，冲跳装备显示身份；未收录型号加一个提示。
 */
export const gearItemSubtitle = (item) => {
  if (!item) return ''
  const base = item.category === GEAR_CATEGORY.BREAK_JUMP
    ? formatGearRoles(item.roles)
    : (item.house ? '球房公用' : formatGearCueStyle(item.cue_style))
  return [base, item.unlisted ? '未收录' : ''].filter(Boolean).join(' · ')
}

export const gearProductTitle = (product) => {
  if (!product) return ''
  return [product.brand_name, product.name].map((part) => String(part || '').trim()).filter(Boolean).join(' ')
}

/**
 * 装备页分组：退役的单独一组，其余按球杆、冲杆跳杆分组；已删除的不展示。
 */
export const groupGearItems = (items = []) => {
  const groups = { cues: [], breakJump: [], retired: [] }
  for (const item of items) {
    if (!item || item.status === GEAR_ITEM_STATUS.DELETED) continue
    if (item.status === GEAR_ITEM_STATUS.RETIRED) {
      groups.retired.push(item)
    } else if (item.category === GEAR_CATEGORY.BREAK_JUMP) {
      groups.breakJump.push(item)
    } else {
      groups.cues.push(item)
    }
  }
  return groups
}

export const isGearItemActive = (item) => Boolean(item) && item.status === GEAR_ITEM_STATUS.ACTIVE

export const canBeMainBreakCue = (item) => {
  return isGearItemActive(item) && item.category === GEAR_CATEGORY.BREAK_JUMP && hasRole(item.roles, 'break')
}

export const canBeMainCue = (item) => {
  return isGearItemActive(item) && item.category === GEAR_CATEGORY.CUE
}

/**
 * 某个槽位可选的主力装备：只有在用的球杆能做主力球杆，只有在用且带冲杆身份的冲跳装备能做主力冲杆。
 */
export const listMainCandidates = (items = [], slot) => {
  const accepts = slot === GEAR_SLOT.BREAK ? canBeMainBreakCue : canBeMainCue
  return items.filter(accepts)
}

/**
 * 主力设置视图：按球种固定顺序列出，斯诺克没有冲杆槽位。
 */
export const buildGearDefaultsView = (defaults = [], items = []) => {
  const itemById = new Map(items.map((item) => [item.id, item]))
  const pick = (gameType, slot) => {
    const found = defaults.find((entry) => entry.game_type === gameType && entry.slot === slot)
    return found ? itemById.get(found.item_id) || null : null
  }
  return ORDERED_GAME_TYPES.map(({ value, label }) => {
    const supportsBreak = value !== 1
    return {
      gameType: value,
      label,
      supportsBreak,
      cue: pick(value, GEAR_SLOT.CUE),
      breakCue: supportsBreak ? pick(value, GEAR_SLOT.BREAK) : null
    }
  })
}

/**
 * 一件装备在哪些球种里是主力，用于装备卡片上的标签：球杆写「中式八球 主力」，冲杆写「中式八球 冲杆主力」。
 */
export const listMainGameLabels = (defaults = [], itemId) => {
  const labelByType = new Map(ORDERED_GAME_TYPES.map(({ value, label }) => [value, label]))
  return ORDERED_GAME_TYPES
    .flatMap(({ value }) => defaults
      .filter((entry) => entry.item_id === itemId && entry.game_type === value)
      .map((entry) => `${labelByType.get(value)} ${entry.slot === GEAR_SLOT.BREAK ? '冲杆主力' : '主力'}`))
}

/**
 * 好友装备页地址：user_id 必填，昵称只用于页面标题展示。
 */
export const buildFriendGearUrl = ({ userId, name = '' } = {}) => {
  const id = Number(userId) || 0
  const query = [`user_id=${id}`]
  const displayName = String(name || '').trim()
  if (displayName) query.push(`name=${encodeURIComponent(displayName)}`)
  return `/subPages/gear/friend?${query.join('&')}`
}
