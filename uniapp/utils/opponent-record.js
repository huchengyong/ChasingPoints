import { formatRelativeTime } from './format.js'
import { resolveAdvantageLevel } from './h2h-copywriting.js'

const decodeRouteValue = (value = '') => {
  if (!value) return ''

  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export const normalizeOpponentRecordOptions = (options = {}) => ({
  targetUserId: Number(options.target_user_id || 0) || 0,
  targetName: decodeRouteValue(options.target_name || ''),
  targetAvatar: decodeRouteValue(options.target_avatar || '')
})

export const buildOpponentRecordRequestParams = ({
  page = 1,
  pageSize = 20,
  keyword = '',
  targetUserId = 0
} = {}) => {
  const params = {
    page,
    page_size: pageSize,
    keyword
  }

  if (Number(targetUserId) > 0) {
    params.target_user_id = Number(targetUserId)
  }

  return params
}

export const buildOpponentRecordViewModel = ({
  targetName = '',
  targetUserId = 0
} = {}) => {
  const isTargetMode = Number(targetUserId) > 0

  if (!isTargetMode) {
    return {
      isTargetMode: false,
      subjectName: '你',
      navigationTitle: '过往对手',
      loginHint: '请登录后查看过往对手',
      searchPlaceholder: '搜索对手',
      emptyText: '暂无过往对手',
      emptyHint: '快去发起一场PK吧！'
    }
  }

  const resolvedTargetName = targetName || '这位球友'

  return {
    isTargetMode: true,
    subjectName: resolvedTargetName,
    navigationTitle: '对方战绩',
    loginHint: '请登录后查看对方战绩',
    searchPlaceholder: '搜索 TA 的对手',
    emptyText: '暂无对方战绩',
    emptyHint: `还没看到${resolvedTargetName} 的历史过往对手`
  }
}

const buildRelationshipCopy = ({ advantageLevel = 'no_sample', subjectName = '你' } = {}) => {
  const map = {
    no_sample: `${subjectName}和他还没形成样本`,
    balanced: `${subjectName}和他势均力敌`,
    slight_lead: `${subjectName}略占上风`,
    slight_trail: `${subjectName}暂时落后`,
    clear_lead: `${subjectName}明显占优`,
    clear_trail: `${subjectName}最近被压制`
  }

  return map[advantageLevel] || map.no_sample
}

const buildRelationshipBadge = (advantageLevel = 'no_sample') => {
  const map = {
    no_sample: '待交锋',
    balanced: '均势',
    slight_lead: '领先',
    slight_trail: '落后',
    clear_lead: '压制',
    clear_trail: '复仇'
  }

  return map[advantageLevel] || '待交锋'
}

const buildToneClass = (advantageLevel = 'no_sample') => {
  if (advantageLevel === 'slight_lead' || advantageLevel === 'clear_lead') {
    return 'lead'
  }
  if (advantageLevel === 'slight_trail' || advantageLevel === 'clear_trail') {
    return 'trail'
  }
  if (advantageLevel === 'balanced') {
    return 'balanced'
  }
  return 'waiting'
}

export const buildOpponentStatsSummary = ({
  totalOpponents = 0,
  totalWins = 0,
  targetName = '',
  targetUserId = 0
} = {}) => {
  const isTargetMode = Number(targetUserId) > 0
  const title = isTargetMode ? `${targetName || '这位球友'}的对手档案` : '你的对手档案'
  const subtitle = totalOpponents > 0
    ? `已经形成 ${totalOpponents} 个对手样本，累计拿下 ${totalWins} 场胜利`
    : '先找到最值得点进复盘的那个对手'

  return { title, subtitle }
}

export const buildOpponentCardViewModels = ({
  opponents = [],
  subjectName = '你',
  currentUserId = 0,
  showWinRateLabel = false
} = {}) => opponents.map((opponent = {}) => {
  const wins = Number(opponent.wins || 0)
  const losses = Number(opponent.losses || 0)
  const totalMatches = Number(opponent.total_matches || 0) || (wins + losses)
  const isMeOpponent = Number(currentUserId) > 0 && Number(opponent.id) === Number(currentUserId)
  const advantageLevel = resolveAdvantageLevel({
    totalMatches,
    myWins: wins,
    opponentWins: losses
  })

  return {
    ...opponent,
    totalMatches,
    isMeOpponent,
    winRateText: `${Math.round(Number(opponent.win_rate || 0))}%${showWinRateLabel ? ' 胜率' : ''}`,
    recordText: `总交锋 ${wins} 胜 ${losses} 负`,
    sampleText: totalMatches > 0 ? `${totalMatches} 场交锋` : '等待首场交锋',
    lastMatchText: opponent.last_match_at ? `上次对局 ${formatRelativeTime(opponent.last_match_at)}` : '还没交过手',
    relationshipText: buildRelationshipCopy({ advantageLevel, subjectName }),
    relationshipBadge: buildRelationshipBadge(advantageLevel),
    toneClass: buildToneClass(advantageLevel)
  }
})

export const buildOpponentH2HUrl = ({
  opponent = {},
  targetUserId = 0,
  targetName = '',
  targetAvatar = ''
} = {}) => {
  const query = []

  if (Number(targetUserId) > 0) {
    query.push(`target_user_id=${Number(targetUserId)}`)
    query.push(`target_name=${encodeURIComponent(targetName || '球友')}`)
    query.push(`target_avatar=${encodeURIComponent(targetAvatar || '')}`)
  }

  if (Number(opponent.id) > 0) {
    query.push(`opponent_id=${Number(opponent.id)}`)
  }

  query.push(`opponent_name=${encodeURIComponent(opponent.name || '对手')}`)

  return `/subPages/user/h2hRecord?${query.join('&')}`
}
