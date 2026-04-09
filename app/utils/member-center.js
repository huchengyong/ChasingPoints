const MONTHLY_PRICE_FEN = 1900

const parseUtc8Time = (rawValue) => {
  if (typeof rawValue !== 'string') return null
  const normalized = rawValue.trim()
  if (!normalized) return null

  const match = normalized.match(/^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/)
  if (!match) {
    return null
  }

  const [, year, month, day, hour, minute, second] = match
  const parsed = new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    Number(second)
  )

  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return parsed
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
  const expiresAtText = typeof memberStatus.member_expires_at === 'string' ? memberStatus.member_expires_at.trim() : ''
  const expiresAt = parseUtc8Time(expiresAtText)
  const isActive = Boolean(memberStatus.is_active) && expiresAt && expiresAt.getTime() > now.getTime()
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
  const expiresAtText = typeof memberStatus.member_expires_at === 'string' ? memberStatus.member_expires_at.trim() : ''
  const expiresAt = parseUtc8Time(expiresAtText)
  const isActive = Boolean(memberStatus.is_active) && expiresAt && expiresAt.getTime() > now.getTime()
  const complianceMode = Boolean(options.complianceMode)

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
      description: `当前有效期至 ${expiresAtText}，续费会在现有到期时间基础上顺延。`,
      primaryActionText: '立即续费'
    }
  }

  if (expiresAtText) {
    return {
      statusText: '已到期',
      title: '会员已到期',
      description: `上次会员有效期到 ${expiresAtText}，续费后会从当前时间重新开始计算。`,
      primaryActionText: '立即续费'
    }
  }

  return {
    statusText: '未开通',
    title: '开通月卡会员',
    description: '目前先开放月卡订阅，支付成功后立即更新会员状态。',
    primaryActionText: '立即开通'
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
