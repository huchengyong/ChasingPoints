import {
  buildEvidenceTextList,
  buildH2HTitle
} from './h2h-copywriting.js'

const countRecentResults = (history = []) => history.slice(0, 5).reduce((acc, item = {}) => {
  if (item.result === 1) acc.wins += 1
  else if (item.result === 2) acc.losses += 1
  else if (item.result === 3) acc.draws += 1
  return acc
}, { wins: 0, losses: 0, draws: 0 })

export const buildPkReportHero = ({
  stats = {},
  opponent = {},
  history = []
} = {}) => {
  const recent = countRecentResults(history)

  return {
    title: buildH2HTitle({
      totalMatches: stats.total_matches || 0,
      myWins: stats.my_wins || 0,
      opponentWins: stats.opponent_wins || 0,
      opponentName: opponent.name || '对手'
    }),
    scoreText: `${stats.my_wins || 0} : ${stats.opponent_wins || 0}`,
    metaText: `胜率 ${Math.round(stats.win_rate || 0)}% · 近 ${Math.min(history.length, 5)} 场 ${recent.wins} 胜 ${recent.losses} 负`
  }
}

export const buildPkEvidenceList = ({
  stats = {},
  history = []
} = {}) => buildEvidenceTextList({
  stats: {
    avg_score_diff: stats.avg_score_diff || 0,
    max_win_streak: stats.max_win_streak || 0
  },
  recentHistory: history
}).slice(0, 2)

export const buildPkPosterPayload = ({
  hero = {},
  evidenceList = [],
  stats = {},
  userName = '我',
  opponentName = '对手',
  gameTypeLabel = '真实交锋数据'
} = {}) => ({
  myName: userName,
  opponentName,
  myWins: stats.my_wins || 0,
  opponentWins: stats.opponent_wins || 0,
  totalMatches: stats.total_matches || 0,
  winRateLabel: `${Math.round(stats.win_rate || 0)}%`,
  avgScoreDiffLabel: `${Number(stats.avg_score_diff || 0) > 0 ? '+' : ''}${Number(stats.avg_score_diff || 0).toFixed(1)}`,
  maxWinStreak: stats.max_win_streak || 0,
  summaryText: [hero.title, ...evidenceList].filter(Boolean).join(' · '),
  gameTypeLabel
})

export const resolvePkReportStatus = ({
  statsLoaded = false,
  historyLoaded = false,
  totalMatches = 0,
  hasError = false
} = {}) => {
  if (statsLoaded && historyLoaded) {
    return totalMatches > 0 ? 'ready' : 'empty'
  }

  if (hasError) {
    return 'error'
  }

  return 'loading'
}

