export const MAX_RANK_LEVEL = 6

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

// ========== 计分规则展示文案（与当前结算策略保持一致） ==========

export const RANK_RULE_SUMMARY_STATS = Object.freeze([
  { value: '+8～40', label: '胜方基础分' },
  { value: '约一半', label: '败方基础扣分' },
  { value: '+500', label: '每日上涨上限' }
])

export const RANK_RULE_NOTE = '实际结算还会受到完成局数、同一对手场次、对手剩余排位分和特殊战绩分影响。'

export const RANK_RULE_DETAIL_ROWS = Object.freeze([
  { title: '完成局数', description: '胜方基础分随完成局数递增，最低 +8，最高 +40。' },
  { title: '失败减免', description: '败方基础扣分约为胜方基础分的一半，特殊战绩最多把失败扣分减免到 -2。' },
  { title: '同一对手', description: '当天同一对手前 2 场按 100% 结算，第 3 场按 80%，第 4～6 场按 30%，第 7 场起不计排位分和胜率。' },
  { title: '对手剩余分', description: '对手剩余排位分不足时，胜方实际基础分会相应降低。' },
  { title: '每日封顶', description: '每个球种每天最多上涨 500 排位分。' }
])

export const RANK_RULE_EXAMPLE = Object.freeze({
  title: '算分示例 · 5 局制胜利',
  formula: '基础分 +20；若这是当天与同一对手的第 3 场，则按 80% 结算为 +16。'
})
