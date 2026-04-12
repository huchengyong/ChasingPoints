const MEMBER_RANKING_DAILY_CAP = 200

const MEMBER_RANKING_LEVELS = [
  { level: 1, multiplier: 1, percentText: '100%' },
  { level: 2, multiplier: 1.1, percentText: '110%' },
  { level: 3, multiplier: 1.2, percentText: '120%' },
  { level: 4, multiplier: 1.3, percentText: '130%' },
  { level: 5, multiplier: 1.4, percentText: '140%' }
]

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

const parseUtc8Time = (rawValue) => {
  if (typeof rawValue !== 'string') return null
  const normalized = rawValue.trim()
  if (!normalized) return null

  const match = normalized.match(/^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/)
  if (!match) return null

  const [, year, month, day, hour, minute, second] = match
  const parsed = new Date(Date.UTC(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour) - 8,
    Number(minute),
    Number(second)
  ))

  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return parsed
}

const resolveMemberActiveState = (memberStatus = {}, now = new Date()) => {
  const expiresAtText = typeof memberStatus.member_expires_at === 'string' ? memberStatus.member_expires_at.trim() : ''
  const expiresAt = parseUtc8Time(expiresAtText)
  return Boolean(memberStatus.is_active) && expiresAt && expiresAt.getTime() > now.getTime()
}

const resolveGrowthLevel = (memberStatus = {}) => {
  const explicitLevel = toSafeInteger(memberStatus.growth_level)
  if (explicitLevel > 0) {
    return explicitLevel
  }

  const growthPoints = Math.max(0, toSafeInteger(memberStatus.growth_points))
  let level = 1

  for (const item of MEMBER_RANKING_LEVELS) {
    if (growthPoints >= resolveGrowthThreshold(item.level)) {
      level = item.level
    }
  }

  return level
}

const resolveGrowthThreshold = (level) => {
  switch (level) {
    case 5:
      return 760
    case 4:
      return 260
    case 3:
      return 60
    case 2:
      return 10
    case 1:
    default:
      return 0
  }
}

const resolveRankingLevelConfig = (level) => {
  return MEMBER_RANKING_LEVELS.find(item => item.level === level) || MEMBER_RANKING_LEVELS[0]
}

export const resolveMemberRankingRightsCard = (memberStatus = {}, now = new Date()) => {
  const active = resolveMemberActiveState(memberStatus, now)
  const level = resolveGrowthLevel(memberStatus)
  const levelConfig = resolveRankingLevelConfig(level)
  const frozen = !active && Boolean(typeof memberStatus.member_expires_at === 'string' && memberStatus.member_expires_at.trim())

  let statusText = '未生效'
  let headline = '普通用户高光只记录，不计入排位'
  let description = '开通会员后，特殊战绩会按当前会员等级倍率计入排位分。'

  if (active) {
    statusText = '权益生效中'
    headline = `当前按 Lv${level} ${levelConfig.percentText} 计入特殊战绩排位分`
    description = `普通用户高光会保留记录但不计入排位；会员用户会按 Lv${level} 的 ${levelConfig.percentText} 倍率结算高光分，单日最高计入 200 分。`
  } else if (frozen) {
    statusText = '权益已冻结'
    headline = `当前冻结在 Lv${level} ${levelConfig.percentText}`
    description = `会员已到期，高光仍会记录但不会计入排位；续开后会按 Lv${level} 的 ${levelConfig.percentText} 倍率恢复结算。`
  }

  return {
    statusText,
    headline,
    description,
    level,
    levelLabel: `Lv${level}`,
    currentPercentText: levelConfig.percentText,
    dailyCap: MEMBER_RANKING_DAILY_CAP,
    levels: MEMBER_RANKING_LEVELS.map(item => ({
      ...item,
      levelLabel: `Lv${item.level}`,
      active: item.level === level,
      enabled: active && item.level <= level
    })),
    ruleItems: [
      '普通用户高光会正常记录到战报，但不计入排位分。',
      '会员用户按 Lv1-Lv5 的 100% / 110% / 120% / 130% / 140% 倍率计入特殊战绩分。',
      `会员特殊战绩排位分仅看每日上限，当日最多计入 ${MEMBER_RANKING_DAILY_CAP} 分。`
    ]
  }
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
  return /(会员特殊战绩分|成就奖励|特殊战绩)/.test(label)
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
        ? `本场记录了 ${recordText}，其中 ${countedScore} 分特殊战绩已计入排位，另有 ${capReduction} 分受到会员特殊战绩单日上限影响。`
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
    description: `本场记录了 ${recordText}，但这场排位未计入高光分。普通用户高光会保留记录，会员用户才会按等级倍率计入。`,
    tone: 'blue',
    countedInRanking: false,
    capped: false,
    recordedAchievementsText: recordText
  }
}

export {
  MEMBER_RANKING_DAILY_CAP,
  MEMBER_RANKING_LEVELS
}
