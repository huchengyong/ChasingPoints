const ACHIEVEMENT_RECORD_DEFS = [
  { key: 'break_clear', label: '炸清' },
  { key: 'continue_clear', label: '接清' },
  { key: 'golden_break', label: '小金' },
  { key: 'nine_on_break', label: '大金' },
  { key: 'break_50', label: '单杆50+' },
  { key: 'break_100', label: '单杆100+' },
  { key: 'break_147', label: '147' }
]

const toSafeInteger = (value) => {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return 0
  }

  return Math.trunc(parsed)
}

export const resolveAchievementRecords = (achievements = {}) => {
  return ACHIEVEMENT_RECORD_DEFS
    .map((item) => ({
      ...item,
      count: Math.max(0, toSafeInteger(achievements[item.key]))
    }))
    .filter(item => item.count > 0)
}

const isAchievementRankDetail = (detail = {}) => {
  const label = typeof detail.label === 'string' ? detail.label : ''
  return /(VIP特殊战绩分|成就奖励|特殊战绩)/.test(label)
}

const isDailyCapDetail = (detail = {}) => {
  const label = typeof detail.label === 'string' ? detail.label : ''
  return /每日封顶/.test(label)
}

const formatAchievementRecordText = (records = []) => {
  if (!records.length) return ''
  return records.map(item => `${item.label}x${item.count}`).join('、')
}

export const resolveMatchRankingRightsSummary = (match = {}) => {
  const achievementRecords = resolveAchievementRecords(match.achievements || {})
  if (Number(match.rank_rules_version) === 2) {
    const recordText = formatAchievementRecordText(achievementRecords)
    return recordText
      ? {
        badgeText: '高光不计分',
        title: '本场高光已记录',
        description: `本场记录了 ${recordText}，高光会保留在战报和成就里；排位分只看胜负与对手实力，会员与普通用户规则相同。`,
        tone: 'blue',
        countedInRanking: false,
        capped: false,
        recordedAchievementsText: recordText
      }
      : {
        badgeText: '只看胜负',
        title: '本场排位只看胜负与对手实力',
        description: '排位分由赛前胜率、赛制长度和胜负决定，会员与普通用户规则相同。',
        tone: 'slate',
        countedInRanking: false,
        capped: false,
        recordedAchievementsText: ''
      }
  }
  const rankDetails = Array.isArray(match.my_rank_details) ? match.my_rank_details : []
  const countedScore = rankDetails
    .filter(isAchievementRankDetail)
    .reduce((sum, item) => sum + Math.max(0, toSafeInteger(item.value)), 0)
  const capReduction = rankDetails
    .filter(isDailyCapDetail)
    .reduce((sum, item) => sum + Math.abs(Math.min(0, toSafeInteger(item.value))), 0)
  const recordText = formatAchievementRecordText(achievementRecords)

  if (!achievementRecords.length) {
    return {
      badgeText: '无高光排位分',
      title: '本场没有可计入排位的特殊战绩',
      description: '本场排位只按基础分结算。',
      tone: 'slate',
      countedInRanking: false,
      capped: false,
      recordedAchievementsText: ''
    }
  }

  if (countedScore > 0) {
    const capped = capReduction > 0
    return {
      badgeText: capped ? '已计入并触发封顶' : '高光已计入排位',
      title: capped ? '本场高光已部分计入排位' : '本场高光已计入排位',
      description: capped
        ? `本场记录了 ${recordText}，其中 ${countedScore} 分特殊战绩已计入排位，另有 ${capReduction} 分受到VIP特殊战绩单日上限影响。`
        : `本场记录了 ${recordText}，特殊战绩分已计入排位，具体结算以排位变化明细为准。`,
      tone: capped ? 'gold' : 'emerald',
      countedInRanking: true,
      capped,
      recordedAchievementsText: recordText
    }
  }

  return {
    badgeText: '高光仅记录',
    title: '本场高光未计入排位',
    description: `本场记录了 ${recordText}，但这场排位未计入高光分。普通用户高光会保留记录，VIP用户才会按等级倍率计入。`,
    tone: 'blue',
    countedInRanking: false,
    capped: false,
    recordedAchievementsText: recordText
  }
}
