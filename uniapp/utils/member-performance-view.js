// 会员表现分析页的展示规则。所有数值都来自服务端 GET /api/member/performance，这里只负责文案与图表比例。

export const PERFORMANCE_SAMPLE_CAP = 200
export const PERFORMANCE_MIN_SUMMARY_SAMPLES = 5
export const PERFORMANCE_MIN_MONTH_SAMPLES = 3
export const PERFORMANCE_MONTH_FOOTNOTE = `每月满 ${PERFORMANCE_MIN_MONTH_SAMPLES} 场排位才显示表现分；读取上限内放不下的月份会标为数据不全。`
export const PERFORMANCE_NOTE = '表现分只反映这段时间打出的水平，不改变排位分。'

const toNumber = (value) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const BOUND_LABELS = { at_least: '至少', at_most: '至多' }

// 失效版本：对局结算（rank）或会员状态变化（member）后，下一次进入页面回源。
export const resolvePerformanceVersion = ({ rankVersion = 0, memberVersion = 0 } = {}) => `${toNumber(rankVersion)}:${toNumber(memberVersion)}`

export const formatSampleCount = (count) => {
  const value = Math.max(0, Math.trunc(toNumber(count)))
  return value >= PERFORMANCE_SAMPLE_CAP ? `${PERFORMANCE_SAMPLE_CAP}+` : String(value)
}

/**
 * 表现分的展示：精确值直接显示；全胜/全负按求解边界显示「至少 / 至多」；样本不足为空。
 */
export const resolveRatingView = (rating) => {
  if (!rating || !Number.isFinite(Number(rating.rating))) {
    return { available: false, value: '', qualifier: '', text: '—' }
  }
  const value = String(Math.round(Number(rating.rating)))
  const qualifier = BOUND_LABELS[rating.bound] || ''
  return { available: true, value, qualifier, text: qualifier ? `${qualifier}${value}` : value }
}

const formatSigned = (value, digits = 1) => {
  const rounded = Number(Math.abs(value).toFixed(digits))
  if (rounded === 0) return `${(0).toFixed(digits)}`
  return `${value > 0 ? '+' : '-'}${rounded.toFixed(digits)}`
}

/**
 * 90 天摘要：实际胜场与期望胜场的差值，以及一句口径明确的解读。
 */
export const resolveSummaryView = (summary, sampleCount = 0) => {
  const samples = Math.max(0, Math.trunc(toNumber(sampleCount)))
  if (!summary) {
    return { available: false, sampleText: formatSampleCount(samples), headline: '', diffText: '', tone: 'flat', rating: resolveRatingView(null) }
  }
  const diff = toNumber(summary.wins_diff)
  const tone = diff >= 0.5 ? 'up' : diff <= -0.5 ? 'down' : 'flat'
  const headline = tone === 'up'
    ? '近 90 天打得比对手实力预期更好'
    : tone === 'down'
      ? '近 90 天赢的场次略少于对手实力预期'
      : '近 90 天的战绩与对手实力预期基本一致'
  const rating = resolveRatingView(summary.rating)
  return {
    available: true,
    sampleText: formatSampleCount(samples),
    wins: Math.max(0, Math.trunc(toNumber(summary.wins))),
    expectedText: toNumber(summary.expected_wins).toFixed(1),
    diffText: formatSigned(diff),
    tone,
    headline,
    rating,
    ratingHint: rating.available ? '' : `近 90 天样本不足 ${PERFORMANCE_MIN_SUMMARY_SAMPLES} 场，打满后显示表现分`
  }
}

const monthLabel = (month) => {
  const monthNumber = Number(String(month || '').slice(5, 7))
  return monthNumber >= 1 && monthNumber <= 12 ? `${monthNumber}月` : String(month || '')
}

const toPercent = (value, min, max) => Math.min(100, Math.max(0, ((value - min) / (max - min)) * 100))

/**
 * 6 个月实力曲线：柱是当月表现分，点是月末排位分。柱与点共用同一个刻度，刻度按数据范围向外取整到 100。
 * 样本不足或被读取上限截断的月份不画柱，并说明原因，避免把残缺数据当成水平。
 */
export const buildMonthChart = (months = []) => {
  const list = Array.isArray(months) ? months : []
  const values = []
  for (const item of list) {
    const rating = resolveRatingView(item?.rating)
    if (rating.available && !item?.incomplete) values.push(Number(rating.value))
    if (toNumber(item?.end_rank_score) > 0) values.push(toNumber(item.end_rank_score))
  }
  if (!values.length) {
    return { hasData: false, scaleMin: 0, scaleMax: 0, bars: [] }
  }
  let scaleMin = Math.floor((Math.min(...values) - 50) / 100) * 100
  let scaleMax = Math.ceil((Math.max(...values) + 50) / 100) * 100
  if (scaleMax - scaleMin < 200) scaleMax = scaleMin + 200
  scaleMin = Math.max(0, scaleMin)

  const bars = list.map((item, index) => {
    const rating = resolveRatingView(item?.rating)
    const incomplete = Boolean(item?.incomplete)
    const endRankScore = Math.max(0, Math.trunc(toNumber(item?.end_rank_score)))
    const sampleCount = Math.max(0, Math.trunc(toNumber(item?.sample_count)))
    const ready = rating.available && !incomplete
    return {
      month: String(item?.month || ''),
      label: monthLabel(item?.month),
      isCurrent: index === list.length - 1,
      status: ready ? 'ready' : incomplete ? 'incomplete' : 'insufficient',
      sampleCount,
      ratingText: ready ? rating.text : '—',
      qualifier: ready ? rating.qualifier : '',
      heightPercent: ready ? toPercent(Number(rating.value), scaleMin, scaleMax) : 0,
      endRankScore,
      dotPercent: endRankScore > 0 ? toPercent(endRankScore, scaleMin, scaleMax) : null,
      note: ready ? '' : incomplete ? '数据不全' : '样本不足'
    }
  })
  return { hasData: true, scaleMin, scaleMax, bars }
}

/**
 * 最佳胜利 / 爆冷失利：展示对手、赛前对手排位分与赛前整场胜率。
 */
export const resolveNotableMatchView = (item, kind) => {
  if (!item) return null
  const rate = toNumber(item.expected_win_rate)
  const rateText = `${Number.isInteger(rate) ? rate : rate.toFixed(1)}%`
  const name = item.opponent_name || '对手'
  const win = kind === 'best_win'
  return {
    kind,
    title: win ? '最佳胜利' : '爆冷失利',
    matchId: Math.trunc(toNumber(item.match_id)),
    opponentName: name,
    opponentAvatar: item.opponent_avatar || '',
    headline: win ? `击败 ${name}` : `负于 ${name}`,
    detail: win
      ? `对手赛前排位分 ${toNumber(item.opponent_before_score)}，赛前你的胜率只有 ${rateText}`
      : `对手赛前排位分 ${toNumber(item.opponent_before_score)}，赛前你的胜率有 ${rateText}`,
    playedAt: item.played_at || ''
  }
}

const FULL_ANALYSIS_FEATURES = [
  '近 90 天实际胜场与期望胜场对比',
  '近 90 天表现分',
  '最佳胜利与爆冷失利',
  '最近 6 个月实力曲线'
]

/**
 * 锁定态（非有效会员）：只展示权益说明、服务端给出的样本数与本月表现分预览。
 * 开通入口是否出现完全由合规开关决定，页面不自行判断。
 */
export const resolveLockedView = (response, { canShowPurchase = false } = {}) => {
  const samples = Math.max(0, Math.trunc(toNumber(response?.sample_count)))
  const preview = response?.month_preview || null
  const previewRating = resolveRatingView(preview?.rating)
  const previewSamples = Math.max(0, Math.trunc(toNumber(preview?.sample_count)))
  const monthText = preview?.month ? `${Number(String(preview.month).slice(5, 7))}月` : '本月'
  return {
    sampleText: formatSampleCount(samples),
    hasSamples: samples > 0,
    features: FULL_ANALYSIS_FEATURES,
    preview: {
      monthText,
      available: previewRating.available,
      ratingText: previewRating.text,
      qualifier: previewRating.qualifier,
      value: previewRating.value,
      sampleCount: previewSamples,
      hint: previewRating.available
        ? `基于${monthText}已打的 ${previewSamples} 场排位`
        : `${monthText}已打 ${previewSamples} 场排位，满 ${PERFORMANCE_MIN_MONTH_SAMPLES} 场后显示表现分`
    },
    showPurchase: Boolean(canShowPurchase)
  }
}
