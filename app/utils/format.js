/**
 * 格式化工具函数
 */

/**
 * 格式化相对时间
 * @param {string|Date} dateTime 日期时间字符串或 Date 对象
 * @returns {string} 格式化后的相对时间字符串
 */
export const formatRelativeTime = (dateTime) => {
  if (!dateTime) return ''

  const now = new Date()
  const date = typeof dateTime === 'string' ? new Date(dateTime) : dateTime

  // 计算时间差（毫秒）
  const diff = now.getTime() - date.getTime()

  // 转换为各时间单位
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  const months = Math.floor(days / 30)
  const years = Math.floor(days / 365)

  if (seconds < 60) {
    return '刚刚'
  } else if (minutes < 60) {
    return `${minutes}分钟前`
  } else if (hours < 24) {
    return `${hours}小时前`
  } else if (days < 30) {
    return `${days}天前`
  } else if (months < 12) {
    return `${months}个月前`
  } else {
    return `${years}年前`
  }
}

/**
 * 格式化日期
 * @param {string|Date} dateTime 日期时间字符串或 Date 对象
 * @param {string} format 格式化模板，默认 'YYYY年MM月DD日'
 * @returns {string} 格式化后的日期字符串
 */
export const formatDate = (dateTime, format = 'YYYY年MM月DD日') => {
  if (!dateTime) return ''

  const date = typeof dateTime === 'string' ? new Date(dateTime) : dateTime

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  return format
    .replace('YYYY', year)
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
}

/**
 * 格式化完整日期时间
 * @param {string|Date} dateTime 日期时间字符串或 Date 对象
 * @returns {string} 格式化后的日期时间字符串
 */
export const formatDateTime = (dateTime) => {
  return formatDate(dateTime, 'YYYY年MM月DD日 HH:mm')
}

/**
 * 生成指定时区下的 YYYY-MM 月份键
 * @param {string|Date} dateTime
 * @param {string} timeZone
 * @returns {string}
 */
export const formatMonthKey = (dateTime, timeZone = 'Asia/Shanghai') => {
  if (!dateTime) return ''

  const date = typeof dateTime === 'string' ? new Date(dateTime) : dateTime
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return ''

  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone,
    year: 'numeric',
    month: '2-digit'
  })

  const parts = formatter.formatToParts(date)
  const year = parts.find(part => part.type === 'year')?.value || ''
  const month = String(parts.find(part => part.type === 'month')?.value || '').padStart(2, '0')
  if (!year || !month) return ''

  return `${year}-${month}`
}

export default {
  formatRelativeTime,
  formatDate,
  formatDateTime,
  formatMonthKey
}
