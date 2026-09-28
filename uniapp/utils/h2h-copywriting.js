const countHistoryResults = (recentHistory = []) => recentHistory.reduce((acc, item = {}) => {
  if (item.result === 1) acc.wins += 1
  else if (item.result === 2) acc.losses += 1
  else if (item.result === 3) acc.draws += 1
  return acc
}, { wins: 0, losses: 0, draws: 0 })

export const resolveAdvantageLevel = ({
  totalMatches = 0,
  myWins = 0,
  opponentWins = 0
} = {}) => {
  if (totalMatches <= 0) return 'no_sample'
  if (myWins === opponentWins) return 'balanced'

  const lead = myWins - opponentWins
  const isLead = lead > 0

  if (totalMatches < 5) {
    return isLead ? 'slight_lead' : 'slight_trail'
  }

  if (Math.abs(lead) >= 3) {
    return isLead ? 'clear_lead' : 'clear_trail'
  }

  return isLead ? 'slight_lead' : 'slight_trail'
}

export const buildH2HTitle = ({
  totalMatches = 0,
  myWins = 0,
  opponentWins = 0,
  opponentName = '对手',
  subjectName = '你'
} = {}) => {
  const level = resolveAdvantageLevel({ totalMatches, myWins, opponentWins })
  const map = {
    no_sample: `${subjectName}和${opponentName}还没形成交锋样本`,
    balanced: `${subjectName}和${opponentName}目前势均力敌`,
    slight_lead: `${subjectName}对${opponentName}略占上风`,
    slight_trail: `${subjectName}对${opponentName}略处下风`,
    clear_lead: `${subjectName}对${opponentName}明显占优`,
    clear_trail: `${subjectName}最近被${opponentName}压制`
  }

  return map[level]
}

export const buildH2HSubtitle = ({
  myWins = 0,
  opponentWins = 0,
  recentHistory = []
} = {}) => {
  const recent = countHistoryResults(recentHistory.slice(0, 5))
  const parts = [`总交锋 ${myWins} 胜 ${opponentWins} 负`]

  if (recentHistory.length > 0) {
    const recentParts = [`近 ${Math.min(recentHistory.length, 5)} 场 ${recent.wins} 胜 ${recent.losses} 负`]
    if (recent.draws > 0) {
      recentParts.push(`${recent.draws} 平`)
    }
    parts.push(recentParts.join(' '))
  }

  return parts.join('，')
}

export const buildPrimaryActionText = (advantageLevel = 'no_sample') => {
  if (advantageLevel === 'clear_lead' || advantageLevel === 'slight_lead') {
    return '再约一场，扩大优势'
  }

  if (advantageLevel === 'clear_trail' || advantageLevel === 'slight_trail') {
    return '发起复仇局'
  }

  if (advantageLevel === 'balanced') {
    return '约一场决胜局'
  }

  return '约一场见真章'
}

export const buildEvidenceTextList = ({
  stats = {},
  recentHistory = []
} = {}) => {
  const recent = countHistoryResults(recentHistory.slice(0, 5))
  const items = []

  if (recentHistory.length > 0) {
    items.push(`近 ${Math.min(recentHistory.length, 5)} 场 ${recent.wins} 胜 ${recent.losses} 负`)
  }

  if (Number.isFinite(stats.max_win_streak) && stats.max_win_streak > 0) {
    items.push(`当前最长连胜 ${stats.max_win_streak} 场`)
  } else if (Number.isFinite(stats.avg_score_diff)) {
    const diff = Number(stats.avg_score_diff)
    items.push(`平均分差 ${diff > 0 ? '+' : ''}${diff.toFixed(1)}`)
  }

  return items.slice(0, 2)
}

