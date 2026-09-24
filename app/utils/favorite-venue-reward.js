const formatRewardDuration = (rewardDays = 30) => {
  if (rewardDays % 30 === 0) {
    const months = rewardDays / 30
    return `${months} 个月会员`
  }
  return `${rewardDays} 天会员`
}

const parseRewardTime = (rawValue) => {
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

const SNOOZE_DURATION_MS = 3 * 24 * 60 * 60 * 1000

export const getFavoriteVenueRewardFloatSnoozeKey = (userId, status) => `favorite-venue-reward-float-snoozed-until:${userId || 0}:${status || 'not_started'}`

export const calcFavoriteVenueRewardFloatSnoozeUntil = (now = new Date()) => new Date(now.getTime() + SNOOZE_DURATION_MS)

export const isFavoriteVenueRewardFloatSnoozed = (snoozeUntil, now = new Date()) => {
  if (!snoozeUntil) return false
  const until = snoozeUntil instanceof Date ? snoozeUntil : new Date(snoozeUntil)
  if (Number.isNaN(until.getTime())) return false
  return now.getTime() < until.getTime()
}

export const getFavoriteVenueRewardPopupStorageKey = (userId) => `favorite-venue-reward-popup-dismissed:${userId || 0}`

export const resolveFavoriteVenueRewardFloatSnoozeMigration = ({ userId, rewardStatus, oldPopupDismissed, getFloatSnoozeValue }) => {
  if (!userId || !rewardStatus) return null
  if (!oldPopupDismissed) return null
  const status = rewardStatus.status
  if (status !== 'not_started') return null
  const existingSnooze = getFloatSnoozeValue(userId, status)
  if (existingSnooze) return null
  return { key: getFavoriteVenueRewardFloatSnoozeKey(userId, status), snoozeUntil: calcFavoriteVenueRewardFloatSnoozeUntil() }
}

export const shouldShowFavoriteVenueRewardPopup = ({ userId, rewardStatus, popupDismissed }) => {
  if (!userId || !rewardStatus) return false
  if (popupDismissed) return false
  if (!rewardStatus.enabled || !rewardStatus.popup_enabled) return false
  return rewardStatus.status === 'not_started'
}

export const resolveFavoriteVenueRewardPopupCopy = (rewardStatus = {}) => {
  const rewardText = formatRewardDuration(rewardStatus.reward_days || 30)

  return {
    title: `补充常玩球馆，送 ${rewardText}`,
    description: '补充你常玩的球馆，审核通过后自动到账，不影响正常使用。',
    primaryText: '去领取',
    secondaryText: '暂不领取'
  }
}

export const resolveFavoriteVenueRewardTaskCard = (rewardStatus = {}) => {
  const rewardText = formatRewardDuration(rewardStatus.reward_days || 30)
  const status = rewardStatus.status || 'not_started'
  const submittedVenueName = typeof rewardStatus.submitted_venue_name === 'string' ? rewardStatus.submitted_venue_name.trim() : ''
  const rejectReason = typeof rewardStatus.reject_reason === 'string' ? rewardStatus.reject_reason.trim() : ''
  const memberExpiresAt = typeof rewardStatus.member_expires_at === 'string' ? rewardStatus.member_expires_at.trim() : ''

  if (!rewardStatus.enabled && status === 'not_started') {
    return {
      visible: false,
      title: '',
      description: '',
      actionText: '',
      statusText: ''
    }
  }

  if (status === 'pending_review') {
    return {
      visible: true,
      title: '常玩球馆审核中',
      description: submittedVenueName ? `${submittedVenueName} 正在审核中，通过后会自动发放会员。` : '球馆信息审核中，通过后会自动发放会员。',
      actionText: '',
      statusText: '审核中'
    }
  }

  if (status === 'reward_granted') {
    return {
      visible: false,
      title: '',
      description: '',
      actionText: '',
      statusText: ''
    }
  }

  if (status === 'rejected') {
    return {
      visible: true,
      title: '补充信息后可重新领取会员',
      description: rejectReason ? `本次未通过：${rejectReason}` : '本次提交未通过，补充完整后可重新提交。',
      actionText: '重新提交',
      statusText: '未通过'
    }
  }

  return {
    visible: true,
    title: `补充常玩球馆，送 ${rewardText}`,
    description: '首次有效补充并审核通过后，会员会自动到账。',
    actionText: '去领取',
    statusText: '常玩球馆奖励'
  }
}

export const resolveFavoriteVenueMemberCard = (rewardStatus = {}, now = new Date()) => {
  const rewardText = formatRewardDuration(rewardStatus.reward_days || 30).replace(/会员$/, '')
  const memberExpiresAt = typeof rewardStatus.member_expires_at === 'string' ? rewardStatus.member_expires_at.trim() : ''
  if (!memberExpiresAt) {
    return {
      visible: false,
      statusText: '',
      title: '',
      description: '',
      benefits: []
    }
  }

  const expiresAt = parseRewardTime(memberExpiresAt)
  const isExpired = expiresAt ? now.getTime() > expiresAt.getTime() : false

  if (isExpired) {
    return {
      visible: true,
      statusText: '已到期',
      title: '订阅会员已到期',
      description: `这次会员已于 ${memberExpiresAt} 到期，奖励记录会继续保留。`,
      benefits: [
        {
          title: '到期时间仍可回看',
          description: '方便你确认上一次会员奖励何时结束。'
        },
        {
          title: '奖励记录继续保留',
          description: '后台仍然可以核对这次常玩球馆奖励的发放记录。'
        },
        {
          title: '新的会员权益还会在这里展示',
          description: '如果后续再获得会员，这里会继续显示最新状态。'
        }
      ]
    }
  }

  return {
    visible: true,
    statusText: '会员中',
    title: '会员生效中',
    description: `有效期至 ${memberExpiresAt}`,
    benefits: []
  }
}
