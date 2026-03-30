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

export const buildH2HTrendItems = (history = []) => history.map((item = {}) => ({
  matchId: item.id || 0,
  result: item.result || 0,
  label: getResultLabel(item.result || 0)
}))

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
