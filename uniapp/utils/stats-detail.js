export const resolveStatsDetailLoadingMode = ({
  hasLoadedOnce = false,
  isFetching = false
} = {}) => {
  if (!isFetching) return 'idle'
  return hasLoadedOnce ? 'refreshing' : 'initial'
}

export const shouldApplyStatsDetailResponse = ({
  requestId = 0,
  latestRequestId = 0
} = {}) => requestId === latestRequestId

export const normalizeDurationStats = (payload = {}) => {
  const stats = payload?.stats || payload || {}
  return {
    average: stats.average_seconds || stats.average || 0,
    fastest: stats.fastest_seconds || stats.fastest || 0,
    longest: stats.longest_seconds || stats.longest || 0,
    total: stats.total_matches || stats.total || 0
  }
}

export const normalizeOpponentStrengthStats = (payload = {}) => {
  const list = Array.isArray(payload?.tiers) ? payload.tiers : Array.isArray(payload?.list) ? payload.list : Array.isArray(payload) ? payload : []
  return list.map((item = {}, index) => ({
    tier_name: item.tier_name || item.rank_range || `段位${index + 1}`,
    matches: item.matches || 0,
    wins: item.wins || 0,
    win_rate: item.win_rate || 0
  }))
}

const padDatePart = (value) => String(value).padStart(2, '0')

const formatDateKey = (date) => (
  `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())}`
)

const formatMonthKeyFromDate = (date) => (
  `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}`
)

const parseMonthKey = (monthKey = '') => {
  const matched = String(monthKey).match(/^(\d{4})-(\d{2})$/)
  if (!matched) return null

  const year = Number(matched[1])
  const month = Number(matched[2])
  if (month < 1 || month > 12) return null

  return { year, month }
}

const resolveMonth = (items = [], monthKey = '') => {
  const parsedMonth = parseMonthKey(monthKey)
  if (parsedMonth) return parsedMonth

  const firstItemWithDate = items.find((item = {}) => {
    const date = new Date(item.date || item.match_time || '')
    return !Number.isNaN(date.getTime())
  })
  const fallbackDate = firstItemWithDate ? new Date(firstItemWithDate.date || firstItemWithDate.match_time) : new Date()

  return {
    year: fallbackDate.getFullYear(),
    month: fallbackDate.getMonth() + 1
  }
}

const buildMonthCells = (year, month, byDate, createEmptyStats) => {
  const currentMonthKey = `${year}-${padDatePart(month)}`
  const firstDay = new Date(year, month - 1, 1)
  const startDate = new Date(year, month - 1, 1 - firstDay.getDay())

  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(startDate)
    date.setDate(startDate.getDate() + index)
    const dateKey = formatDateKey(date)
    const stats = byDate.get(dateKey) || createEmptyStats()

    return {
      dateKey,
      day: date.getDate(),
      isCurrentMonth: formatMonthKeyFromDate(date) === currentMonthKey,
      ...stats
    }
  })
}

export const shiftMonthKey = (monthKey = '', offset = 0) => {
  const { year, month } = parseMonthKey(monthKey) || resolveMonth()
  const date = new Date(year, month - 1 + offset, 1)
  return formatMonthKeyFromDate(date)
}

export const buildStatsTrendCalendarViewModel = (matches = [], { monthKey = '' } = {}) => {
  const { year, month } = resolveMonth(matches, monthKey)
  const currentMonthKey = `${year}-${padDatePart(month)}`
  const byDate = new Map()
  const summary = {
    matchCount: 0,
    winCount: 0,
    lossCount: 0,
    drawCount: 0,
    winRate: 0
  }

  matches.forEach((item = {}) => {
    const date = new Date(item.date || item.match_time || '')
    if (Number.isNaN(date.getTime()) || formatMonthKeyFromDate(date) !== currentMonthKey) return

    const dateKey = formatDateKey(date)
    const dayStats = byDate.get(dateKey) || {
      winCount: 0,
      lossCount: 0,
      drawCount: 0,
      matchCount: 0
    }

    if (item.result === 1) {
      dayStats.winCount += 1
      summary.winCount += 1
    } else if (item.result === 2) {
      dayStats.lossCount += 1
      summary.lossCount += 1
    } else {
      dayStats.drawCount += 1
      summary.drawCount += 1
    }
    dayStats.matchCount += 1
    summary.matchCount += 1
    byDate.set(dateKey, dayStats)
  })

  summary.winRate = summary.matchCount ? Math.round((summary.winCount / summary.matchCount) * 100) : 0

  const cells = buildMonthCells(year, month, byDate, () => ({
    winCount: 0,
    lossCount: 0,
    drawCount: 0,
    matchCount: 0
  })).map(cell => ({
    ...cell,
    hasMatches: cell.matchCount > 0
  }))

  return {
    monthKey: currentMonthKey,
    title: `${year}年${month}月`,
    weekdays: ['日', '一', '二', '三', '四', '五', '六'],
    summary,
    cells
  }
}

const normalizeRankChanges = (rankPoints = []) => {
  const list = Array.isArray(rankPoints?.list) ? rankPoints.list : Array.isArray(rankPoints) ? rankPoints : []
  return list.map((item = {}, index) => {
    const prev = list[index + 1]
    const prevScore = prev?.rank_score ?? item.rank_score ?? 0
    const score = item.rank_score || 0
    return {
      date: item.date || '',
      score,
      delta: score - prevScore
    }
  })
}

export const buildRankDeltaChartViewModel = (rankPoints = [], { monthKey = '' } = {}) => {
  const changes = normalizeRankChanges(rankPoints)
  const { year, month } = resolveMonth(changes, monthKey)
  const currentMonthKey = `${year}-${padDatePart(month)}`
  const daysInMonth = new Date(year, month, 0).getDate()
  const byDate = new Map()

  changes.forEach((item = {}) => {
    const date = new Date(item.date || '')
    if (Number.isNaN(date.getTime()) || formatMonthKeyFromDate(date) !== currentMonthKey) return

    const dateKey = formatDateKey(date)
    byDate.set(dateKey, (byDate.get(dateKey) || 0) + (item.delta || 0))
  })

  const deltas = Array.from(byDate.values())
  const maxAbsDelta = deltas.reduce((max, delta) => Math.max(max, Math.abs(delta)), 0)
  const days = Array.from({ length: daysInMonth }, (_, index) => {
    const day = index + 1
    const dateKey = `${currentMonthKey}-${padDatePart(day)}`
    const delta = byDate.get(dateKey) || 0
    const absDelta = Math.abs(delta)

    return {
      dateKey,
      day,
      label: `${day}日`,
      delta,
      hasChange: delta !== 0,
      isUp: delta > 0,
      isDown: delta < 0,
      heightPercent: maxAbsDelta > 0 ? Math.max(8, Math.round((absDelta / maxAbsDelta) * 100)) : 0,
      showTick: day === 1 || day === 10 || day === 20 || day === daysInMonth
    }
  })

  return {
    monthKey: currentMonthKey,
    title: `${year}年${month}月`,
    maxAbsDelta,
    summary: {
      netDelta: deltas.reduce((total, delta) => total + delta, 0),
      upDays: deltas.filter(delta => delta > 0).length,
      downDays: deltas.filter(delta => delta < 0).length,
      maxGain: deltas.reduce((max, delta) => Math.max(max, delta), 0),
      maxLoss: deltas.reduce((min, delta) => Math.min(min, delta), 0)
    },
    days
  }
}
