import { formatRelativeTime } from './format.js'

const HIDDEN_TITLE = '对方已隐藏战绩'
const HIDDEN_DESC = '这位好友暂时没有公开自己的战绩列表'
const BATTLE_SECTION_TITLE = '对方战绩'
const BATTLE_SECTION_TIP = '直接查看 TA 最近的过往对手'

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
  const totalMatches = Number(
    stats.total_matches ??
    stats.totalMatches ??
    stats.total_opponents ??
    stats.totalOpponents ??
    0
  ) || 0
  const totalWins = Number(
    stats.total_wins ??
    stats.totalWins ??
    stats.my_wins ??
    stats.myWins ??
    0
  ) || 0
  const relativeTime = lastMatchAt ? relativeFormatter(lastMatchAt) : ''
  const recentMatchLabel = relativeTime || '暂无记录'
  const hasMatches = totalMatches > 0
  const heroTitle = hasMatches
    ? `这位球友最近有 ${totalMatches} 场真实对局`
    : '这位球友还没有公开的对局样本'
  const heroDesc = hasMatches
    ? '可以直接往下看 TA 的对手战绩列表，PK 报表里再看双方对比。'
    : '等 TA 完成更多真实对局后，这里会直接更新最新战绩列表。'

  return {
    hasMatches,
    heroTitle,
    heroDesc,
    totalMatchesText: `真实对局 ${totalMatches} 场`,
    recordText: `累计胜场 ${totalWins} 场`,
    recentMatchText: `最近更新 ${recentMatchLabel}`,
    emptyTitle: hasMatches ? '' : '暂时还没有可展示的战绩',
    emptyDesc: hasMatches ? '' : '等这位好友完成真实对局后，再回来看看',
    battleSectionTitle: BATTLE_SECTION_TITLE,
    battleSectionTip: BATTLE_SECTION_TIP,
    hiddenTitle: HIDDEN_TITLE,
    hiddenDesc: HIDDEN_DESC
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
