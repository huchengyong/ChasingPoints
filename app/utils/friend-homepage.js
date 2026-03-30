import { formatRelativeTime } from './format.js'

const EMPTY_SUMMARY_DESC = '先积累一场真实对局，再来看战绩和 PK 报表'
const EMPTY_HERO_DESC = '先约一场真实对局，这里就会慢慢留下你们的战绩。'

const decodeQueryValue = (value = '') => {
  if (!value) return ''

  try {
    return decodeURIComponent(value)
  } catch {
    return String(value)
  }
}

export const normalizeFriendHomepageOptions = (options = {}) => ({
  opponentId: Number(options.opponent_id || 0),
  opponentName: decodeQueryValue(options.opponent_name),
  opponentAvatar: decodeQueryValue(options.opponent_avatar),
  rankName: decodeQueryValue(options.rank_name)
})

export const buildFriendHomepageSummary = ({
  stats = {},
  lastMatchAt = '',
  formatRelativeTime: relativeFormatter = formatRelativeTime
} = {}) => {
  const totalMatches = Number(stats.total_matches ?? stats.totalMatches ?? 0) || 0
  const myWins = Number(stats.my_wins ?? stats.myWins ?? 0) || 0
  const opponentWins = Number(stats.opponent_wins ?? stats.opponentWins ?? 0) || 0
  const relativeTime = lastMatchAt ? relativeFormatter(lastMatchAt) : ''
  const recentMatchLabel = relativeTime || '暂无记录'
  const hasMatches = totalMatches > 0
  const heroTitle = !hasMatches
    ? '你们还没正式交手'
    : myWins > opponentWins
      ? '你暂时领先'
      : myWins < opponentWins
        ? '他暂时领先'
        : '你们目前势均力敌'
  const heroDesc = !hasMatches
    ? EMPTY_HERO_DESC
    : myWins > opponentWins
      ? `你们已经认真交手 ${totalMatches} 场，你暂时领先。`
      : myWins < opponentWins
        ? '这位球友最近手感更好一点，你还有机会下一场追回来。'
        : '你们最近打得很胶着，暂时谁也没拉开差距。'

  return {
    hasMatches,
    heroTitle,
    heroDesc,
    totalMatchesText: `${totalMatches} 场交手`,
    recordText: `你 ${myWins} 胜 ${opponentWins} 负`,
    recentMatchText: `最近一次交手 ${recentMatchLabel}`,
    emptyTitle: hasMatches ? '' : '你们还没有正式交手',
    emptyDesc: hasMatches ? '' : EMPTY_SUMMARY_DESC
  }
}

export const fetchFriendHomepageData = async ({
  params = {},
  getStats,
  getHistory
} = {}) => {
  if (typeof getStats !== 'function' || typeof getHistory !== 'function') {
    throw new TypeError('fetchFriendHomepageData requires getStats and getHistory functions')
  }

  const [statsRes, historyRes] = await Promise.all([
    getStats(params),
    getHistory({ ...params, page: 1, page_size: 1 })
  ])

  return {
    statsRes: statsRes || {},
    historyRes: historyRes || {}
  }
}
