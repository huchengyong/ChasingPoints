export const MAX_RANK_LEVEL = 6

// 段位配置的静态缓存版本：配置新增了规则版本字段，升级版本号让旧缓存直接失效。
// 3：切换前拉取的缓存没有 rules_effective_at，升级后重新拉取，避免切换后仍按旧规则展示。
export const RANK_CONFIGS_STATIC_VERSION = 3

export const resolveRankExplainLoadingMode = ({
  hasLoadedOnce = false,
  isFetching = false
} = {}) => {
  if (!isFetching) return 'idle'
  return hasLoadedOnce ? 'refreshing' : 'initial'
}

export const shouldApplyRankExplainResponse = ({
  requestId = 0,
  latestRequestId = 0
} = {}) => requestId === latestRequestId

const GROWTH_STATE_TEXT = Object.freeze({
  current: '当前',
  reached: '已达成',
  pending: '待晋级'
})

const clampPercent = (value) => {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, Math.round(value)))
}

const normalizeRankConfigList = (rankList = []) => {
  return rankList
    .filter((item) => item && Number(item.level) > 0)
    .map((item) => ({
      level: Number(item.level),
      name: item.name || '',
      icon: item.icon || '',
      min_score: Number(item.min_score) || 0
    }))
    .sort((a, b) => a.level - b.level)
}

/**
 * 成长路径：按王者到青铜的顺序返回六级段位，
 * 每一级包含分数区间与“当前/已达成/待晋级”状态。
 */
export const buildRankGrowthPath = ({ rankList = [], currentLevel = 0, rankScore = 0 } = {}) => {
  const ascending = normalizeRankConfigList(rankList)
  const level = Number(currentLevel) || 0
  const score = Number(rankScore) || 0

  return ascending.map((item, index) => {
    const upper = ascending[index + 1]
    const rangeText = upper ? `${item.min_score}–${upper.min_score - 1} 分` : `${item.min_score} 分以上`
    let state = 'pending'
    if (item.level === level) {
      state = 'current'
    } else if (score >= item.min_score) {
      state = 'reached'
    }
    return { ...item, rangeText, state, stateText: GROWTH_STATE_TEXT[state] }
  }).reverse()
}

/**
 * 当前段位英雄卡展示模型：
 * 等级 1–5 计算级内进度、下一段位和剩余分数；等级 6 返回最高段位状态。
 */
export const buildRankHeroModel = ({ rankInfo = null, rankList = [] } = {}) => {
  if (!rankInfo) return null
  const level = Number(rankInfo.level) || 0
  const rankScore = Number(rankInfo.rank_score) || 0
  const base = {
    level,
    name: rankInfo.name || '',
    icon: rankInfo.icon || '',
    rankScore,
    placementRemaining: Math.max(0, Number(rankInfo.placement_remaining) || 0),
    isMax: level >= MAX_RANK_LEVEL
  }
  if (base.isMax) {
    return { ...base, progressPercent: 100 }
  }

  const currentConfig = normalizeRankConfigList(rankList).find((item) => item.level === level)
  const minScore = currentConfig ? currentConfig.min_score : 0
  const nextScore = Number(rankInfo.next_score) || 0
  const earned = Math.max(0, rankScore - minScore)
  const span = Math.max(0, nextScore - minScore)

  return {
    ...base,
    nextName: rankInfo.next_name || '',
    nextScore,
    earned,
    span,
    remaining: Math.max(0, nextScore - rankScore),
    progressPercent: span > 0 ? clampPercent((earned * 100) / span) : clampPercent(Number(rankInfo.progress))
  }
}

// ========== 计分规则展示文案（与服务端结算保持一致） ==========

const RULES_VERSION_LEGACY = 1
const RULES_VERSION_V2 = 2

// 旧规则：只在排位规则 V2 生效前展示。
export const LEGACY_RANK_RULES = Object.freeze({
  version: RULES_VERSION_LEGACY,
  summaryStats: Object.freeze([
    { value: '+8～40', label: '胜方基础分' },
    { value: '约一半', label: '败方基础扣分' },
    { value: '+500', label: '每日上涨上限' }
  ]),
  note: '实际结算还会受到完成局数、同一对手场次、对手剩余排位分和会员特殊战绩影响。',
  detailRows: Object.freeze([
    { title: '完成局数', description: '胜方基础分随完成局数递增，最低 +8，最高 +40。' },
    { title: '败方扣分', description: '败方扣胜方基础分的一半；排位分不会低于 0。' },
    { title: '同一对手', description: '当天同一对手前 2 场按 100% 结算，第 3 场按 80%，第 4～6 场按 30%（只缩减胜方加分），第 7 场起不计排位分和胜率。' },
    { title: '对手剩余分', description: '对手剩余排位分不足时，胜方实际基础分会相应降低，最低 +2。' },
    { title: '会员特殊战绩', description: '会员赢球时，特殊战绩按会员等级倍率计入排位分，单日最多 200 分；输球不计，普通用户只记录不计分。' },
    { title: '每日封顶', description: '每个球种每天最多上涨 500 排位分。' }
  ]),
  example: Object.freeze({
    title: '算分示例 · 5 局制胜利',
    formula: '基础分 +20；若这是当天与同一对手的第 3 场，则按 80% 结算为 +16。'
  })
})

// 排位规则 V2：示例分值来自服务端同一公式（单局分差 2500 分为 10:1，K = 40·√N）。
export const RATING_V2_RANK_RULES = Object.freeze({
  version: RULES_VERSION_V2,
  summaryStats: Object.freeze([
    { value: '±45', label: '抢 5 同分胜负' },
    { value: '+68', label: '赢高 500 分的对手' },
    { value: '10 场', label: '定级赛涨跌加倍' }
  ]),
  note: '排位分只看胜负和对手实力：赢比你强的人加得多，输给比你强的人扣得少，赛制越长分值越大。会员身份和特殊战绩都不影响排位分。',
  detailRows: Object.freeze([
    { title: '赛前胜率', description: '按双方排位分算出赛前胜率：分差 500 分时，高分方单局胜率约 61%，抢 5 约 76%。结果越出乎意料，分数变化越大。' },
    { title: '赛制分值', description: '同分对手胜负：抢 1 为 ±20，抢 3 为 ±35，抢 5 为 ±45，抢 9 为 ±60，更长赛制按抢 9 计。5:0 与 5:4 分值相同；自由局按赢局较多一方的局数计。' },
    { title: '定级赛', description: '每个球种前 10 场为定级赛，涨跌加倍；已定级的球手遇到定级中的对手，涨跌减半。' },
    { title: '同一对手', description: '当天同一对手前 2 场按 100% 结算，第 3 场按 80%，第 4～6 场按 30%，双方同比例；第 7 场起不计排位分和胜率。' },
    { title: '每日封顶', description: '每个球种每天最多上涨 500 排位分；排位分不会低于 0。' },
    { title: '高光不计分', description: '炸清、接清、单杆等高光照常记录在战报和成就里，但不改变排位分，会员与普通用户规则相同。' },
    { title: '赛季排名', description: '赛季内该球种打满 5 场、遇到至少 3 位不同对手才参与赛季排名与称号；排位分跨赛季延续，不做重置。' }
  ]),
  example: Object.freeze({
    title: '算分示例 · 抢 5',
    formula: '两人同分时胜方 +45、负方 −45；你比对手低 500 分并获胜得 +68；输给比你高 500 分的对手只扣 21。'
  })
})

const parseEffectiveAt = (value) => {
  const time = Date.parse(String(value || ''))
  return Number.isFinite(time) ? time : 0
}

const formatEffectiveDate = (time) => {
  const date = new Date(time)
  // 生效时刻按北京时间零点下发，换算成北京时间日期展示。
  const shanghai = new Date(date.getTime() + 8 * 60 * 60 * 1000)
  return `${shanghai.getUTCFullYear()}年${shanghai.getUTCMonth() + 1}月${shanghai.getUTCDate()}日`
}

/**
 * 判断排位规则 V2 是否已生效或已排期：优先按生效时刻判断，避免静态配置缓存跨过切换点后仍按旧规则展示。
 */
export const resolveRankRulesState = ({ rulesVersion = 0, rulesEffectiveAt = '', now = Date.now() } = {}) => {
  const effectiveAt = parseEffectiveAt(rulesEffectiveAt)
  const active = effectiveAt > 0 ? now >= effectiveAt : Number(rulesVersion) === RULES_VERSION_V2
  return {
    active,
    scheduled: !active && effectiveAt > 0,
    effectiveDateText: effectiveAt > 0 ? formatEffectiveDate(effectiveAt) : ''
  }
}

/**
 * 选择要展示的计分规则与规则切换提示。
 */
export const resolveRankRules = (options = {}) => {
  const state = resolveRankRulesState(options)
  if (state.active) {
    return {
      ...RATING_V2_RANK_RULES,
      notice: state.effectiveDateText ? `新排位规则已于${state.effectiveDateText}生效，此前的赛季成绩与称号保持不变。` : ''
    }
  }
  return {
    ...LEGACY_RANK_RULES,
    notice: state.scheduled
      ? `新排位规则将于${state.effectiveDateText}起生效：排位分改为按对手实力计算，生效时用新规则重算你的历史排位确定初始分，前 10 场为定级赛。`
      : ''
  }
}
