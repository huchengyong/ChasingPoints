import { resolveMemberRankingRightsCard } from './member-ranking-rights.js'

const MONTHLY_PRICE_FEN = 1900
const MEMBER_GROWTH_DAILY_CAP = 5
const MEMBER_GROWTH_LEVELS = [
  { level: 1, threshold: 0 },
  { level: 2, threshold: 10 },
  { level: 3, threshold: 60 },
  { level: 4, threshold: 260 },
  { level: 5, threshold: 760 }
]

const toSafeInteger = (value) => {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0
  }

  return Math.floor(parsed)
}

const parseUtc8Time = (rawValue) => {
  if (typeof rawValue !== 'string') return null
  const normalized = rawValue.trim()
  if (!normalized) return null

  const match = normalized.match(/^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/)
  if (!match) {
    return null
  }

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

const resolveMemberActivity = (memberStatus = {}, now = new Date()) => {
  const expiresAtText = typeof memberStatus.member_expires_at === 'string' ? memberStatus.member_expires_at.trim() : ''
  const expiresAt = parseUtc8Time(expiresAtText)
  const isActive = Boolean(memberStatus.is_active) && expiresAt && expiresAt.getTime() > now.getTime()

  return {
    expiresAtText,
    expiresAt,
    isActive
  }
}

const resolveMemberGrowthPoints = (memberStatus = {}) => {
  return toSafeInteger(
    memberStatus.growth_points
    ?? memberStatus.member_growth_points
    ?? memberStatus.total_growth_points
  )
}

const resolveTodayGrowthCount = (memberStatus = {}) => {
  const dailyCap = resolveGrowthDailyCap(memberStatus)
  return Math.min(
    dailyCap,
    toSafeInteger(
      memberStatus.today_growth_count
      ?? memberStatus.member_growth_today_count
      ?? memberStatus.daily_growth_count
    )
  )
}

const resolveGrowthDailyCap = (memberStatus = {}) => {
  return Math.max(
    1,
    toSafeInteger(memberStatus.growth_daily_cap ?? memberStatus.member_growth_daily_cap) || MEMBER_GROWTH_DAILY_CAP
  )
}

const resolveCurrentLevel = (growthPoints, explicitLevel = 0) => {
  const normalizedLevel = toSafeInteger(explicitLevel)
  if (normalizedLevel > 0) {
    const matched = MEMBER_GROWTH_LEVELS.find(item => item.level === normalizedLevel)
    if (matched) return matched
    return { level: normalizedLevel, threshold: growthPoints }
  }

  let current = MEMBER_GROWTH_LEVELS[0]

  for (const item of MEMBER_GROWTH_LEVELS) {
    if (growthPoints >= item.threshold) {
      current = item
      continue
    }
    break
  }

  return current
}

const resolveNextLevel = (currentLevel, memberStatus = {}) => {
  const explicitLevel = toSafeInteger(memberStatus.next_growth_level)
  if (explicitLevel > 0) {
    const matched = MEMBER_GROWTH_LEVELS.find(item => item.level === explicitLevel)
    if (matched) return matched
    return {
      level: explicitLevel,
      threshold: toSafeInteger(memberStatus.next_growth_level_points)
    }
  }
  return MEMBER_GROWTH_LEVELS.find(item => item.level === currentLevel.level + 1) || null
}

const resolveGrowthProgressPercent = (growthPoints, currentLevel, nextLevel) => {
  if (!nextLevel) return 100

  const segmentPoints = nextLevel.threshold - currentLevel.threshold
  if (segmentPoints <= 0) return 100

  const completed = Math.max(0, growthPoints - currentLevel.threshold)
  return Math.min(100, Math.round((completed / segmentPoints) * 100))
}

export const getMemberPayChannelOptions = () => ([
  {
    value: 'alipay',
    label: '支付宝',
    description: 'APP 端直接拉起支付宝完成支付'
  },
  {
    value: 'wechat',
    label: '微信支付',
    description: 'APP 端直接拉起微信完成支付'
  }
])

const resolveComplianceMemberEntryCard = (expiresAtText, isActive) => {
  if (isActive) {
    return {
      visible: true,
      eyebrow: '会员权益',
      statusText: '会员权益中',
      title: '会员权益已生效',
      description: '查看当前权益明细和有效期。',
      actionText: '查看权益',
      priceText: ''
    }
  }

  if (expiresAtText) {
    return {
      visible: true,
      eyebrow: '会员权益',
      statusText: '已结束',
      title: '获赠会员已结束',
      description: `上次获赠会员有效期到 ${expiresAtText}，新的权益到账后会继续展示。`,
      actionText: '查看权益',
      priceText: ''
    }
  }

  return {
    visible: true,
    eyebrow: '会员权益',
    statusText: '待发放',
    title: '会员权益待发放',
    description: '新用户注册奖励或活动奖励到账后，会直接展示在这里。',
    actionText: '查看权益',
    priceText: ''
  }
}

export const resolveMemberEntryCard = (memberStatus = {}, now = new Date(), options = {}) => {
  const { expiresAtText, isActive } = resolveMemberActivity(memberStatus, now)
  const complianceMode = Boolean(options.complianceMode)

  if (complianceMode) {
    return resolveComplianceMemberEntryCard(expiresAtText, isActive)
  }

  if (isActive) {
    return {
      visible: true,
      eyebrow: '会员中心',
      statusText: '会员中',
      title: `会员有效期至 ${expiresAtText}`,
      description: '当前会员已生效，可随时续费顺延时长。',
      actionText: '立即续费',
      priceText: '月卡 ¥19'
    }
  }

  if (expiresAtText) {
    return {
      visible: true,
      eyebrow: '会员中心',
      statusText: '已到期',
      title: '会员已到期',
      description: `上次会员有效期到 ${expiresAtText}，现在可以继续续费。`,
      actionText: '立即续费',
      priceText: '月卡 ¥19'
    }
  }

  return {
    visible: true,
    eyebrow: '会员中心',
    statusText: '未开通',
    title: '月卡会员',
    description: '开通后即可在这里查看会员状态，并随时继续订阅。',
    actionText: '立即开通',
    priceText: '月卡 ¥19'
  }
}

export const resolveMemberCenterSummary = (memberStatus = {}, now = new Date(), options = {}) => {
  const { expiresAtText, isActive } = resolveMemberActivity(memberStatus, now)
  const complianceMode = Boolean(options.complianceMode)
  const growthPoints = resolveMemberGrowthPoints(memberStatus)
  const growthDailyCap = resolveGrowthDailyCap(memberStatus)
  const todayGrowthCount = resolveTodayGrowthCount(memberStatus)
  const currentLevel = resolveCurrentLevel(growthPoints, memberStatus.growth_level)
  const growthStatus = `Lv${currentLevel.level}，今日已计入 ${todayGrowthCount}/${growthDailyCap}`
  const rankingRightsCard = resolveMemberRankingRightsCard(memberStatus, now)

  if (complianceMode) {
    if (isActive) {
      return {
        statusText: '会员权益中',
        title: '获赠会员权益已生效',
        description: `当前有效期至 ${expiresAtText}，新的系统奖励或活动奖励也会继续同步到这里。`,
        primaryActionText: '查看权益'
      }
    }

    if (expiresAtText) {
      return {
        statusText: '已结束',
        title: '获赠会员权益已结束',
        description: `上次获赠会员有效期到 ${expiresAtText}，如后续继续发放，会在这里展示最新状态。`,
        primaryActionText: '查看权益'
      }
    }

    return {
      statusText: '待发放',
      title: '会员权益等待发放',
      description: '新用户注册奖励或活动奖励到账后，无需订阅或支付，会自动更新这里的状态。',
      primaryActionText: '查看权益'
    }
  }

  if (isActive) {
    return {
      statusText: '会员中',
      title: '月卡会员生效中',
      description: `当前有效期至 ${expiresAtText}，会员成长 ${growthStatus}，高光排位按 ${rankingRightsCard.levelLabel} ${rankingRightsCard.currentPercentText} 计入。`,
      primaryActionText: '立即续费'
    }
  }

  if (expiresAtText) {
    const hasGrowthProgress = growthPoints > 0 || currentLevel.level > 1 || Boolean(memberStatus.growth_frozen)
    return {
      statusText: '已到期',
      title: '会员已到期',
      description: hasGrowthProgress
        ? `上次会员有效期到 ${expiresAtText}，当前成长 Lv${currentLevel.level} 已冻结，续开后恢复继续成长；高光排位权益会按 ${rankingRightsCard.levelLabel} ${rankingRightsCard.currentPercentText} 恢复。`
        : `上次会员有效期到 ${expiresAtText}，续费后会从当前时间重新开始计算。`,
      primaryActionText: '立即续费'
    }
  }

  return {
    statusText: '未开通',
    title: '开通月卡会员',
    description: '目前先开放月卡订阅，开通后完成真实对局可累计会员成长，高光会按会员等级计入排位。',
    primaryActionText: '立即开通'
  }
}

export const resolveMemberGrowthCard = (memberStatus = {}, now = new Date()) => {
  const { expiresAtText, isActive } = resolveMemberActivity(memberStatus, now)
  const growthPoints = resolveMemberGrowthPoints(memberStatus)
  const growthDailyCap = resolveGrowthDailyCap(memberStatus)
  const todayGrowthCount = resolveTodayGrowthCount(memberStatus)
  const currentLevel = resolveCurrentLevel(growthPoints, memberStatus.growth_level)
  const nextLevel = resolveNextLevel(currentLevel, memberStatus)
  const remainingPoints = toSafeInteger(memberStatus.remaining_growth_points) || (nextLevel ? Math.max(0, nextLevel.threshold - growthPoints) : 0)
  const frozen = !isActive && Boolean(expiresAtText)
  const hasGrowthProfile = frozen || growthPoints > 0 || todayGrowthCount > 0

  let statusText = '待开启'
  let description = '开通会员后，完成真实对局可累计成长。'
  let nextLevelText = '开通会员后，完成真实对局可累计成长'

  if (isActive) {
    statusText = '成长中'
    description = `当前已累计 ${growthPoints} 点成长值，今日已计入 ${todayGrowthCount}/${growthDailyCap} 场真实对局。`
    nextLevelText = nextLevel
      ? `距 Lv${nextLevel.level} 还差 ${remainingPoints} 场`
      : '当前已达到最高成长等级'
  } else if (frozen && hasGrowthProfile) {
    statusText = '成长已冻结'
    description = `会员已于 ${expiresAtText} 到期，当前成长已冻结，续开后恢复继续成长。`
    nextLevelText = nextLevel
      ? `续开后，再完成 ${remainingPoints} 场可升到 Lv${nextLevel.level}`
      : '续开后继续保留当前最高成长等级'
  }

  return {
    visible: true,
    statusText,
    level: currentLevel.level,
    levelLabel: `Lv${currentLevel.level}`,
    growthPoints,
    growthPointsText: `成长值 ${growthPoints}`,
    todayGrowthCount,
    dailyCap: growthDailyCap,
    todayProgressText: `今日已计入 ${todayGrowthCount}/${growthDailyCap}`,
    frozen,
    description,
    nextLevelText,
    nextLevel,
    progressPercent: resolveGrowthProgressPercent(growthPoints, currentLevel, nextLevel)
  }
}

export const resolveMemberPlanCards = (plans = [], selectedPlanCode = '') => {
  return plans.map((item) => ({
    ...item,
    selected: item.plan_code === selectedPlanCode
  }))
}

export const formatMemberPrice = (priceFen = MONTHLY_PRICE_FEN) => `¥${(priceFen / 100).toFixed(2)}`

export const createMemberPaymentRequest = ({ planCode, payChannel }) => ({
  plan_code: planCode,
  pay_channel: payChannel
})

export const shouldTreatMemberOrderAsPaid = (orderStatus = {}) => {
  return Boolean(orderStatus.success) && orderStatus.status === 'paid'
}
