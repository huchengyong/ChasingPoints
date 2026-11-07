/**
 * 约球时间纯逻辑：以服务端时间为基准的北京时间三日整点区间。
 */

export const CHALLENGE_DAY_OFFSETS = [
  { value: 0, label: '今天' },
  { value: 1, label: '明天' },
  { value: 2, label: '后天' }
]

const pad = (value) => String(value).padStart(2, '0')

/**
 * 解析 "YYYY-MM-DD HH:mm:ss" 为北京时间分量（不依赖设备时区）。
 * 返回 { year, month, day, hour, minute } 或 null。
 */
export const parseServerTime = (serverTime = '') => {
  const match = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$/.exec(String(serverTime).trim())
  if (!match) return null
  return {
    year: Number(match[1]),
    month: Number(match[2]),
    day: Number(match[3]),
    hour: Number(match[4]),
    minute: Number(match[5])
  }
}

/**
 * 按服务端时间 + 相对天数计算实际预约日期（北京时间，按服务端日期起算）。
 */
export const computeScheduledDate = (serverTime, dayOffset) => {
  const parts = parseServerTime(serverTime)
  if (!parts) return ''
  const base = Date.UTC(parts.year, parts.month - 1, parts.day)
  const shifted = new Date(base + dayOffset * 24 * 60 * 60 * 1000)
  return `${shifted.getUTCFullYear()}-${pad(shifted.getUTCMonth() + 1)}-${pad(shifted.getUTCDate())}`
}

/**
 * 今天的可用整点：开始小时需满足区间未完全过去（结束时刻 24 点 = 次日 0 点仍在未来）。
 * 返回 0-23 升序数组；serverTime 缺失时返回全部小时。
 */
export const availableStartHours = (serverTime, dayOffset) => {
  const all = Array.from({ length: 24 }, (_, index) => index)
  const parts = parseServerTime(serverTime)
  if (!parts) return all
  if (dayOffset > 0) return all
  // 当天：结束晚于当前时刻才可用（结束至少是当前整点+1，即开始 >= 当前小时）。
  return all.filter((hour) => hour >= parts.hour)
}

/**
 * 默认选择：下一个可用整点的一小时区间；当天没有可用区间则展示明天。
 * 返回 { dayOffset, startHour, endHour }。
 */
export const defaultChallengeSlot = (serverTime) => {
  const offsets = [0, 1, 2]
  const parts = parseServerTime(serverTime)
  for (const dayOffset of offsets) {
    const hours = availableStartHours(serverTime, dayOffset)
    const candidates = parts && dayOffset === 0 ? hours.filter((hour) => hour > parts.hour || (hour === parts.hour && parts.minute === 0)) : hours
    const startHour = (candidates && candidates.length ? candidates : hours)[0]
    if (dayOffset === 0 && parts && startHour === parts.hour && parts.minute > 0 && !candidates.includes(startHour)) {
      continue
    }
    if (hours.length) {
      return { dayOffset, startHour, endHour: Math.min(startHour + 1, 24) }
    }
  }
  return { dayOffset: 1, startHour: 0, endHour: 1 }
}

/**
 * 实际预约日相对服务端今天的天数差（可为负）；无法解析时返回 null。
 */
export const dayOffsetOfScheduledDate = (serverTime, scheduledDate) => {
  const parts = parseServerTime(serverTime)
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(scheduledDate || '').trim())
  if (!parts || !match) return null
  const today = Date.UTC(parts.year, parts.month - 1, parts.day)
  const scheduled = Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
  return Math.round((scheduled - today) / 86400000)
}

/**
 * 结束小时选项（必须晚于开始，1-24）。
 */
export const endHourOptions = (startHour) => {
  if (!Number.isInteger(startHour) || startHour < 0 || startHour > 23) return []
  const options = []
  for (let hour = startHour + 1; hour <= 24; hour += 1) {
    options.push(hour)
  }
  return options
}

export const hourLabel = (hour) => `${hour}点`
