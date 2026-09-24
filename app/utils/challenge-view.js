/**
 * 约球展示纯逻辑：以服务端时间为基准生成「今天/昨天/月日」与失效文案。
 */

const pad = (value) => String(value).padStart(2, '0')

const parseServerTime = (serverTime = '') => {
  const match = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$/.exec(String(serverTime || '').trim())
  if (!match) return null
  return {
    year: Number(match[1]),
    month: Number(match[2]),
    day: Number(match[3]),
    hour: Number(match[4]),
    minute: Number(match[5])
  }
}

const dateKeyOf = (parts) => `${parts.year}-${pad(parts.month)}-${pad(parts.day)}`

/**
 * 相对日期文案：今天/明天/昨天/月日（跨年带年份）。
 * @param {string} scheduledDate YYYY-MM-DD 实际预约日期
 * @param {string} serverTime 服务端当前时间
 */
export const formatChallengeDayLabel = (scheduledDate, serverTime) => {
  const target = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(scheduledDate || '').trim())
  const now = parseServerTime(serverTime)
  if (!target || !now) {
    return String(scheduledDate || '').slice(0, 10)
  }
  const sameYear = Number(target[1]) === now.year
  const dayDiff = Math.round(
    (Date.UTC(Number(target[1]), Number(target[2]) - 1, Number(target[3])) -
      Date.UTC(now.year, now.month - 1, now.day)) / 86400000
  )
  if (dayDiff === 0) return '今天'
  if (dayDiff === 1) return '明天'
  if (dayDiff === -1) return '昨天'
  if (sameYear) return `${Number(target[2])}月${Number(target[3])}日`
  return `${target[1]}年${Number(target[2])}月${Number(target[3])}日`
}

/**
 * 卡片/详情的预约时间行：`今天16点—17点`。
 */
export const formatChallengeSchedule = (challenge = {}, serverTime = '') => {
  if (!challenge.scheduled_date) return ''
  const day = formatChallengeDayLabel(challenge.scheduled_date, serverTime)
  return `${day}${challenge.start_hour}点—${challenge.end_hour}点`
}

/**
 * 失效提示：未开局且未过期 → 「明天早晨7点失效」；过期 → 「已失效」。
 */
export const formatChallengeExpiry = (challenge = {}, serverTime = '') => {
  if (!challenge.expires_at) return ''
  const now = parseServerTime(serverTime)
  const expiry = parseServerTime(challenge.expires_at)
  if (!now || !expiry) return '已失效'
  const dayDiff = Math.round(
    (Date.UTC(expiry.year, expiry.month - 1, expiry.day) -
      Date.UTC(now.year, now.month - 1, now.day)) / 86400000
  )
  const nowMinutes = now.hour * 60 + now.minute
  const expiryMinutes = expiry.hour * 60 + expiry.minute
  if (dayDiff < 0 || (dayDiff === 0 && nowMinutes >= expiryMinutes)) return '已失效'
  if (dayDiff === 0 && expiryMinutes === 420) return '今天早晨7点失效'
  if (dayDiff === 1 && expiryMinutes === 420) return '明天早晨7点失效'
  return '已失效'
}
