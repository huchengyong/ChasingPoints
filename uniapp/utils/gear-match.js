import { GEAR_CATEGORY, GEAR_ITEM_STATUS, gearItemSubtitle, gearItemTitle, gearProductTitle } from './gear-display.js'

// 「本场用杆」卡片的纯逻辑：是否请求、展示行、编辑草稿、换杆预填、请求体与读取器。
// 访问控制以服务端为准：这里的判断只用来避免对旁观者、裁判发起明知会被拒绝的请求。

// 服务端用这些原因拒绝：不是选手 / 对局未结束 / 对局不存在。卡片直接隐藏，不当作错误。
export const GEAR_MATCH_HIDDEN_REASONS = Object.freeze(['not_participant', 'not_completed', 'not_found'])

export const GEAR_MATCH_COMPLETED_STATUS = 2

/**
 * 只有已完成对局的选手才有「本场用杆」；观战、裁判视角不请求。
 */
export const shouldShowMatchGear = ({ status, viewerRole = '', isPlayer1 = false, isSpectate = false } = {}) => {
  if (isSpectate) return false
  if (Number(status) !== GEAR_MATCH_COMPLETED_STATUS) return false
  return Boolean(isPlayer1) || viewerRole === 'player1' || viewerRole === 'player2'
}

export const isGearMatchHiddenResponse = (response) => {
  return Boolean(response) && response.success === false && GEAR_MATCH_HIDDEN_REASONS.includes(response.reason)
}

/**
 * 把一方的记录整理成展示行。每一行都是「球杆 / 皮头 / 巧克 / 冲杆」之一，没有的不展示。
 */
export const describeGearMatchSide = (side) => {
  if (!side || !side.recorded) return { recorded: false, rows: [] }
  const rows = []
  if (side.cue) {
    rows.push({ key: 'cue', label: '球杆', title: gearItemTitle(side.cue), subtitle: gearItemSubtitle(side.cue), subject: side.cue })
  }
  if (side.tip_product) {
    rows.push({ key: 'tip', label: '皮头', title: gearProductTitle(side.tip_product), subtitle: '', subject: side.tip_product })
  }
  if (side.chalk_product) {
    rows.push({ key: 'chalk', label: '巧克', title: gearProductTitle(side.chalk_product), subtitle: '', subject: side.chalk_product })
  }
  if (side.break_cue) {
    rows.push({ key: 'break', label: '冲杆', title: gearItemTitle(side.break_cue), subtitle: gearItemSubtitle(side.break_cue), subject: side.break_cue })
  }
  return { recorded: true, rows }
}

const pad2 = (value) => String(value).padStart(2, '0')

/**
 * 修改截止的提示文案：「5月3日前可修改」。无法解析时返回空串，由调用方决定是否展示。
 */
export const formatGearEditDeadline = (deadline) => {
  const date = new Date(deadline)
  if (!deadline || Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}月${date.getDate()}日前可修改`
}

/**
 * 当前能不能改：服务端给出 editable，同时客户端按截止时间再核对一次，页面长时间停留后不会误导。
 */
export const canEditGearMatch = (response, now = Date.now()) => {
  if (!response?.success || !response.editable) return false
  const deadline = new Date(response.edit_deadline).getTime()
  return Number.isNaN(deadline) ? true : now < deadline
}

// ========== 编辑草稿 ==========

export const buildGearMatchDraft = (side) => ({
  cueItemId: side?.cue?.id || 0,
  tipProduct: side?.tip_product || null,
  chalkProduct: side?.chalk_product || null,
  breakItemId: side?.break_cue?.id || 0
})

/**
 * 换球杆时，用这根杆当前的皮头与巧克预填，用户可以再改。选「不记录」（item 为空）则一并清空。
 */
export const applyCueToDraft = (draft, item) => ({
  ...draft,
  cueItemId: item?.id || 0,
  tipProduct: item?.tip_product || null,
  chalkProduct: item?.chalk_product || null
})

export const isGearMatchDraftEmpty = (draft) => {
  return !draft.cueItemId && !draft.tipProduct && !draft.chalkProduct && !draft.breakItemId
}

/**
 * 修改请求体。球种不记录冲杆时不会带冲杆。
 */
export const buildGearMatchPayload = (matchId, draft, { supportsBreak = true } = {}) => ({
  match_id: Number(matchId) || 0,
  cue_item_id: draft.cueItemId || 0,
  tip_product_id: draft.tipProduct?.id || 0,
  chalk_product_id: draft.chalkProduct?.id || 0,
  break_item_id: supportsBreak ? (draft.breakItemId || 0) : 0
})

const isSelectable = (item, category) => {
  return Boolean(item) && item.category === category && item.status !== GEAR_ITEM_STATUS.DELETED
}

/**
 * 补记历史对局时可以选的球杆：未删除的球杆，已退役的也可以（当年可能就是用它打的）。
 */
export const listMatchCueOptions = (items = []) => items.filter((item) => isSelectable(item, GEAR_CATEGORY.CUE))

export const listMatchBreakOptions = (items = []) => {
  return items.filter((item) => isSelectable(item, GEAR_CATEGORY.BREAK_JUMP) && Array.isArray(item.roles) && item.roles.includes('break'))
}

// ========== 读取器 ==========

const sameIdentity = (a, b) => a.userId === b.userId && a.authGeneration === b.authGeneration

/**
 * 带身份校验的读取器：同一时刻只认最新的一次读取；读取期间切号、退出或又发起新读取，旧响应一律丢弃。
 * load 返回 { stale: true } 表示结果不能使用；请求失败会抛出。
 */
export const createGearMatchLoader = ({ request, readIdentity }) => {
  let sequence = 0
  return {
    async load(matchId) {
      const identity = readIdentity()
      const current = ++sequence
      const response = await request({ match_id: matchId })
      if (current !== sequence || !sameIdentity(identity, readIdentity())) return { stale: true }
      return { stale: false, response }
    },
    // 页面卸载或本场对局变化时调用：进行中的读取不再回写。
    cancel() {
      sequence += 1
    }
  }
}
