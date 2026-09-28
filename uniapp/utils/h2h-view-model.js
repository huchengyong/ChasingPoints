import {
  buildH2HTitle,
  buildH2HSubtitle,
  resolveAdvantageLevel,
  buildPrimaryActionText
} from './h2h-copywriting.js'

const getResultLabel = (result) => {
  if (result === 1) return '胜'
  if (result === 2) return '负'
  return '平'
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

const resolveCalendarMonth = (history = [], monthKey = '') => {
  const parsedMonth = parseMonthKey(monthKey)
  if (parsedMonth) return parsedMonth

  const latestMatch = history.find((item = {}) => {
    const date = new Date(item.match_time || '')
    return !Number.isNaN(date.getTime())
  })
  const latestDate = latestMatch ? new Date(latestMatch.match_time) : new Date()

  return {
    year: latestDate.getFullYear(),
    month: latestDate.getMonth() + 1
  }
}

const getRecentMomentumBadge = (history = []) => {
  const recent = history.slice(0, 5)
  if (recent.length === 0) return '等待首场交锋'

  const net = recent.reduce((acc, item = {}) => {
    if (item.result === 1) return acc + 1
    if (item.result === 2) return acc - 1
    return acc
  }, 0)

  if (net >= 2) return '近期强势'
  if (net <= -2) return '近期回落'
  return '五五开'
}

const getStreakBadge = (stats = {}) => {
  const streak = Number(stats.max_win_streak || 0)
  return streak > 0 ? `当前 ${streak} 连胜` : '等待连胜'
}

const getDiffBadge = (stats = {}) => {
  const diff = Number(stats.avg_score_diff || 0)
  return `平均分差 ${diff > 0 ? '+' : ''}${diff.toFixed(1)}`
}

export const buildH2HHeroViewModel = ({
  stats = {},
  opponent = {},
  history = [],
  subjectName = '你',
  subjectAvatar = ''
} = {}) => ({
  title: buildH2HTitle({
    totalMatches: stats.total_matches || 0,
    myWins: stats.my_wins || 0,
    opponentWins: stats.opponent_wins || 0,
    opponentName: opponent.name || '对手',
    subjectName
  }),
  subtitle: buildH2HSubtitle({
    myWins: stats.my_wins || 0,
    opponentWins: stats.opponent_wins || 0,
    recentHistory: history
  }),
  scoreText: `${stats.my_wins || 0} : ${stats.opponent_wins || 0}`,
  leftName: subjectName,
  leftAvatar: subjectAvatar,
  rightName: opponent.name || '对手',
  rightAvatar: opponent.avatar || '',
  badges: [
    getRecentMomentumBadge(history),
    getStreakBadge(stats),
    getDiffBadge(stats)
  ],
  advantageLevel: resolveAdvantageLevel({
    totalMatches: stats.total_matches || 0,
    myWins: stats.my_wins || 0,
    opponentWins: stats.opponent_wins || 0
  }),
  primaryActionText: buildPrimaryActionText(resolveAdvantageLevel({
    totalMatches: stats.total_matches || 0,
    myWins: stats.my_wins || 0,
    opponentWins: stats.opponent_wins || 0
  }))
})

export const buildH2HCalendarViewModel = (history = [], { monthKey = '' } = {}) => {
  const { year, month } = resolveCalendarMonth(history, monthKey)
  const currentMonthKey = `${year}-${padDatePart(month)}`
  const byDate = new Map()

  history.forEach((item = {}) => {
    const date = new Date(item.match_time || '')
    if (Number.isNaN(date.getTime()) || formatMonthKeyFromDate(date) !== currentMonthKey) return

    const dateKey = formatDateKey(date)
    const dayStats = byDate.get(dateKey) || {
      winCount: 0,
      lossCount: 0,
      drawCount: 0,
      matchCount: 0,
      matchIds: []
    }

    if (item.result === 1) {
      dayStats.winCount += 1
    } else if (item.result === 2) {
      dayStats.lossCount += 1
    } else if (item.result === 3) {
      dayStats.drawCount += 1
    }
    dayStats.matchCount += 1
    if (item.id) {
      dayStats.matchIds.push(item.id)
    }

    byDate.set(dateKey, dayStats)
  })

  const firstDay = new Date(year, month - 1, 1)
  const startDate = new Date(year, month - 1, 1 - firstDay.getDay())
  const cells = Array.from({ length: 42 }, (_, index) => {
    const date = new Date(startDate)
    date.setDate(startDate.getDate() + index)
    const dateKey = formatDateKey(date)
    const stats = byDate.get(dateKey) || {
      winCount: 0,
      lossCount: 0,
      drawCount: 0,
      matchCount: 0,
      matchIds: []
    }

    return {
      dateKey,
      day: date.getDate(),
      isCurrentMonth: formatMonthKeyFromDate(date) === currentMonthKey,
      hasMatches: stats.matchCount > 0,
      ...stats
    }
  })

  return {
    monthKey: currentMonthKey,
    title: `${year}年${month}月`,
    weekdays: ['日', '一', '二', '三', '四', '五', '六'],
    cells
  }
}

export const buildH2HMatchCards = (history = []) => history.map((item = {}) => {
  const diff = Number(item.my_score || 0) - Number(item.opponent_score || 0)
  let tag = '平局'
  if (item.result === 1) {
    tag = diff >= 4 ? '大胜' : '险胜'
  } else if (item.result === 2) {
    tag = diff <= -4 ? '被压制' : '惜败'
  }

  return {
    id: item.id || 0,
    result: item.result || 0,
    dateText: item.match_time || '',
    gameTypeText: item.game_type_name || '台球',
    scoreText: `${item.my_score || 0} : ${item.opponent_score || 0}`,
    resultText: getResultLabel(item.result || 0),
    diffText: `${diff > 0 ? '+' : ''}${diff}`,
    tags: [tag]
  }
})

export const resolveH2HPageStatus = ({
  statsLoaded = false,
  historyLoaded = false,
  totalMatches = 0,
  historyLength = 0,
  hasError = false
} = {}) => {
  if (hasError && (statsLoaded || historyLoaded)) {
    return 'partial'
  }

  if (hasError) {
    return 'error'
  }

  if (statsLoaded && historyLoaded) {
    return totalMatches <= 0 && historyLength <= 0 ? 'empty' : 'ready'
  }

  return 'loading'
}

export const shouldShowH2HSummaryCard = ({
  pageStatus = 'loading',
  statsLoaded = false
} = {}) => statsLoaded && pageStatus !== 'loading' && pageStatus !== 'error'
