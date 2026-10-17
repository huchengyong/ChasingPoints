const DEFAULT_PRESET = 'full_year'

const pad = (value) => String(value).padStart(2, '0')

const buildDateText = (year, month, day) => `${year}-${pad(month)}-${pad(day)}`

const getLastDayOfMonth = (year, month) => {
  return new Date(Date.UTC(year, month, 0)).getUTCDate()
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

const MIN_YEAR = 2023

export const buildYearOptions = (centerYear, span = 2, now = Date.now()) => {
  const year = Number(centerYear)
  if (!Number.isInteger(year) || year <= 0) return []

  const currentYear = new Date(now).getFullYear()
  const maxYear = Math.min(year + span, currentYear)
  const minYear = Math.max(year - span, MIN_YEAR)
  const options = []
  for (let value = minYear; value <= maxYear; value += 1) {
    options.push(value)
  }
  return options
}

export const buildYearBoundaryDate = (year, month, dayHint) => {
  const normalizedYear = Number(year)
  const normalizedMonth = Number(month)
  if (!Number.isInteger(normalizedYear) || !Number.isInteger(normalizedMonth)) return ''

  const maxDay = getLastDayOfMonth(normalizedYear, normalizedMonth)
  const normalizedDay = Math.min(Math.max(Number(dayHint) || 1, 1), maxDay)
  return buildDateText(normalizedYear, normalizedMonth, normalizedDay)
}
