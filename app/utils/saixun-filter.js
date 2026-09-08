const DEFAULT_PRESET = 'full_year'

const pad = (value) => String(value).padStart(2, '0')

const buildDateText = (year, month, day) => `${year}-${pad(month)}-${pad(day)}`

const parseDateText = (value) => {
  const text = String(value || '').trim()
  const match = text.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return null

  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const date = new Date(Date.UTC(year, month - 1, day))
  if (
    Number.isNaN(date.getTime()) ||
    date.getUTCFullYear() !== year ||
    date.getUTCMonth() + 1 !== month ||
    date.getUTCDate() !== day
  ) {
    return null
  }

  return { year, month, day }
}

const getLastDayOfMonth = (year, month) => {
  return new Date(Date.UTC(year, month, 0)).getUTCDate()
}

const formatDisplayDate = (value) => {
  const parsed = parseDateText(value)
  if (!parsed) return ''
  return `${parsed.year}.${pad(parsed.month)}.${pad(parsed.day)}`
}

const compareDateText = (left, right) => {
  if (!left && !right) return 0
  if (!left) return 1
  if (!right) return -1
  if (left === right) return 0
  return left < right ? -1 : 1
}

export const buildYearDateRange = (year) => {
  const normalizedYear = Number(year)
  if (!Number.isInteger(normalizedYear) || normalizedYear <= 0) {
    return { from: '', to: '' }
  }

  return {
    from: buildDateText(normalizedYear, 1, 1),
    to: buildDateText(normalizedYear, 12, 31)
  }
}

export const buildCurrentYearFilter = (now = Date.now()) => {
  const date = typeof now === 'string' ? new Date(now) : new Date(now)
  const year = Number.isNaN(date.getTime()) ? new Date().getFullYear() : date.getFullYear()
  const range = buildYearDateRange(year)

  return {
    year,
    from: range.from,
    to: range.to,
    preset: DEFAULT_PRESET
  }
}

export const buildPresetDateRange = (year, presetKey) => {
  const normalizedYear = Number(year)
  if (!Number.isInteger(normalizedYear) || normalizedYear <= 0) {
    return { from: '', to: '', preset: DEFAULT_PRESET }
  }

  switch (presetKey) {
    case 'q1':
      return { from: buildDateText(normalizedYear, 1, 1), to: buildDateText(normalizedYear, 3, 31), preset: 'q1' }
    case 'jan_apr':
      return { from: buildDateText(normalizedYear, 1, 1), to: buildDateText(normalizedYear, 4, 30), preset: 'jan_apr' }
    case 'jul_dec':
      return { from: buildDateText(normalizedYear, 7, 1), to: buildDateText(normalizedYear, 12, 31), preset: 'jul_dec' }
    case DEFAULT_PRESET:
    default: {
      const range = buildYearDateRange(normalizedYear)
      return { ...range, preset: DEFAULT_PRESET }
    }
  }
}

export const formatSaiXunFilterLabel = (filter = {}) => {
  const preset = String(filter.preset || '').trim()
  const from = String(filter.from || '').trim()
  const to = String(filter.to || '').trim()

  if (!from || !to) return '全年'
  if (preset === DEFAULT_PRESET) return '全年'
  if (preset === 'q1') return '1月-3月'
  if (preset === 'jan_apr') return '1月-4月'
  if (preset === 'jul_dec') return '7月-12月'

  const fromText = formatDisplayDate(from)
  const toText = formatDisplayDate(to)
  if (!fromText || !toText) return '全年'
  return `${fromText} - ${toText}`
}

export const buildEventNewsListParams = (filter = {}, page = 1, pageSize = 10) => {
  const params = {
    page,
    page_size: pageSize
  }

  const year = Number(filter.year || 0)
  const from = String(filter.from || '').trim()
  const to = String(filter.to || '').trim()

  if (from && to) {
    params.from = from
    params.to = to
  } else if (Number.isInteger(year) && year > 0) {
    params.year = year
  }

  const gameType = Number(filter.gameType || 0)
  if (gameType > 0) params.game_type = gameType

  const status = Number(filter.status)
  if (Number.isInteger(status) && status >= 0) params.status = status

  return params
}

export const buildYearOptions = (centerYear, span = 2) => {
  const year = Number(centerYear)
  if (!Number.isInteger(year) || year <= 0) return []

  const options = []
  for (let value = year - span; value <= year + span; value += 1) {
    options.push(value)
  }
  return options
}

export const buildCustomDateRange = (year, from, to) => {
  const normalizedYear = Number(year)
  const parsedFrom = parseDateText(from)
  const parsedTo = parseDateText(to)
  if (!parsedFrom || !parsedTo) {
    return buildPresetDateRange(normalizedYear, DEFAULT_PRESET)
  }

  const fromText = buildDateText(parsedFrom.year, parsedFrom.month, parsedFrom.day)
  const toText = buildDateText(parsedTo.year, parsedTo.month, parsedTo.day)
  const orderedRange = compareDateText(fromText, toText) <= 0
    ? { from: fromText, to: toText }
    : { from: toText, to: fromText }

  return {
    year: normalizedYear,
    from: orderedRange.from,
    to: orderedRange.to,
    preset: 'custom'
  }
}

export const buildYearBoundaryDate = (year, month, dayHint) => {
  const normalizedYear = Number(year)
  const normalizedMonth = Number(month)
  if (!Number.isInteger(normalizedYear) || !Number.isInteger(normalizedMonth)) return ''

  const maxDay = getLastDayOfMonth(normalizedYear, normalizedMonth)
  const normalizedDay = Math.min(Math.max(Number(dayHint) || 1, 1), maxDay)
  return buildDateText(normalizedYear, normalizedMonth, normalizedDay)
}

export const SAIXUN_DATE_PRESETS = Object.freeze([
  { key: DEFAULT_PRESET, label: '全年' },
  { key: 'q1', label: '1月-3月' },
  { key: 'jan_apr', label: '1月-4月' },
  { key: 'jul_dec', label: '7月-12月' },
  { key: 'custom', label: '自定义' }
])
