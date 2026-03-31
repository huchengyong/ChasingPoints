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

export const resolveMemberEntryCard = (memberStatus = {}, now = new Date()) => {
  const expiresAtText = typeof memberStatus.member_expires_at === 'string' ? memberStatus.member_expires_at.trim() : ''
  const expiresAt = parseUtc8Time(expiresAtText)
  const isActive = Boolean(memberStatus.is_active) && expiresAt && expiresAt.getTime() > now.getTime()

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

export const resolveMemberCenterSummary = (memberStatus = {}, now = new Date()) => {
  const expiresAtText = typeof memberStatus.member_expires_at === 'string' ? memberStatus.member_expires_at.trim() : ''
  const expiresAt = parseUtc8Time(expiresAtText)
  const isActive = Boolean(memberStatus.is_active) && expiresAt && expiresAt.getTime() > now.getTime()

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
