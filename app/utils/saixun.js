import { getGameTypeLabel } from './game-types.js'
import { formatEventNewsTime, getEventNewsStatusText } from './home-index.js'
import { DEFAULT_USER_AVATAR } from './user-profile.js'

export const DEFAULT_EVENT_COVER = '/static/images/default-event-cover.png'
export const DEFAULT_PLAYER_AVATAR = DEFAULT_USER_AVATAR

const MATCH_ACTIVE_STATUSES = new Set([0, 1])
const MATCH_LIVE_FALLBACK_WINDOW_MS = 12 * 60 * 60 * 1000
const CJK_PATTERN = /[\u3400-\u9fff]/
const YEAR_PATTERN = /\b(19|20)\d{2}\b/g
const ROUND_NAME_ALIASES = Object.freeze({
  quarter_final: '四分之一决赛',
  quarter_finals: '四分之一决赛',
  quarterfinal: '四分之一决赛',
  quarterfinals: '四分之一决赛',
  semi_final: '半决赛',
  semi_finals: '半决赛',
  semifinal: '半决赛',
  semifinals: '半决赛',
  final: '决赛'
})
const ORDINAL_WORD_TO_NUMBER = Object.freeze({
  one: '1',
  two: '2',
  three: '3',
  four: '4',
  five: '5',
  six: '6',
  seven: '7',
  eight: '8',
  nine: '9',
  ten: '10'
})
const COUNTRY_NAME_MAP = Object.freeze({
  england: '英格兰',
  scotland: '苏格兰',
  wales: '威尔士',
  'northern ireland': '北爱尔兰',
  china: '中国',
  germany: '德国',
  'saudi arabia': '沙特阿拉伯',
  thailand: '泰国',
  belgium: '比利时',
  australia: '澳大利亚',
  iran: '伊朗',
  india: '印度',
  pakistan: '巴基斯坦',
  malta: '马耳他',
  brazil: '巴西',
  canada: '加拿大',
  usa: '美国',
  'united states': '美国',
  japan: '日本',
  'south korea': '韩国',
  france: '法国',
  italy: '意大利',
  spain: '西班牙',
  netherlands: '荷兰',
  russia: '俄罗斯',
  switzerland: '瑞士',
  austria: '奥地利',
  poland: '波兰',
  norway: '挪威',
  sweden: '瑞典',
  denmark: '丹麦',
  finland: '芬兰',
  ireland: '爱尔兰',
  'hong kong': '中国香港',
  taiwan: '中国台湾',
  macau: '中国澳门'
})

const TOURNAMENT_SERIES_TRANSLATIONS = Object.freeze([
  { pattern: /\bRiyadh Season Snooker Championship\b/i, zh: '利雅得狂欢季斯诺克锦标赛' },
  { pattern: /\bSaudi Arabia Snooker Masters\b/i, zh: '沙特阿拉伯斯诺克大师赛' },
  { pattern: /\bShanghai Masters\b/i, zh: '上海大师赛' },
  { pattern: /\bGerman Masters\b/i, zh: '德国大师赛' },
  { pattern: /\bTour Championship\b/i, zh: '斯诺克巡回锦标赛' },
  { pattern: /\bPlayers Championship\b/i, zh: '球员锦标赛' },
  { pattern: /\bChampion of Champions\b/i, zh: '冠中冠' },
  { pattern: /\bWorld Championship\b/i, zh: '世界锦标赛' },
  { pattern: /\bWorld Grand Prix\b/i, zh: '世界大奖赛' },
  { pattern: /\bInternational Championship\b/i, zh: '国际锦标赛' },
  { pattern: /\bBritish Open\b/i, zh: '英国公开赛' },
  { pattern: /\bUK Championship\b/i, zh: '英国锦标赛' },
  { pattern: /\bEnglish Open\b/i, zh: '英格兰公开赛' },
  { pattern: /\bNorthern Ireland Open\b/i, zh: '北爱尔兰公开赛' },
  { pattern: /\bScottish Open\b/i, zh: '苏格兰公开赛' },
  { pattern: /\bWelsh Open\b/i, zh: '威尔士公开赛' },
  { pattern: /\bWorld Open\b/i, zh: '世界公开赛' },
  { pattern: /\bWuhan Open\b/i, zh: '武汉公开赛' },
  { pattern: /\bChina Open\b/i, zh: '中国公开赛' },
  { pattern: /\bXi'an Grand Prix\b/i, zh: '西安大奖赛' },
  { pattern: /\bAsia\s*&\s*Oceania Q School\b/i, zh: '亚洲及大洋洲 Q School' },
  { pattern: /\bQ School\b/i, zh: 'Q School' },
  { pattern: /\bShoot Out\b/i, zh: '单局限时赛' },
  { pattern: /\bMasters\b/i, zh: '大师赛' }
])

const toTitleWithYearAndSuffix = (label, year, suffix) => {
  if (!label) return ''

  let result = label
  if (year) result += ` ${year}`
  if (suffix) result += `（${suffix}）`
  return result
}

const extractTournamentYear = (text) => {
  const matches = Array.from(String(text || '').matchAll(YEAR_PATTERN))
  return matches.length ? matches[matches.length - 1][0] : ''
}

const stripYearFromTitle = (text, year) => {
  if (!year) return String(text || '').trim()

  const index = String(text || '').lastIndexOf(year)
  if (index < 0) return String(text || '').trim()

  return `${text.slice(0, index)}${text.slice(index + year.length)}`
    .replace(/\s{2,}/g, ' ')
    .replace(/\s+([)\]-])/g, '$1')
    .replace(/([([\-])\s+/g, '$1')
    .trim()
}

const normalizeOrdinalToken = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''
  if (/^\d+$/.test(text)) return text

  const normalized = text.toLowerCase().replace(/[^a-z0-9]+/g, '')
  return ORDINAL_WORD_TO_NUMBER[normalized] || text
}

const localizeLeagueStageSuffix = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''

  return text
    .replace(/\bStage One\b/gi, '第一阶段')
    .replace(/\bStage Two\b/gi, '第二阶段')
    .replace(/\bStage Three\b/gi, '第三阶段')
    .replace(/\bWK\s*(\d+)\b/gi, '第$1周')
    .replace(/\s*&\s*Final\b/gi, '及决赛')
    .replace(/\s*\/\s*/g, ' / ')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

const localizeTournamentSeries = (title) => {
  const text = String(title || '').trim()
  if (!text) return ''

  const matchedSeries = TOURNAMENT_SERIES_TRANSLATIONS.find(({ pattern }) => pattern.test(text))
  return matchedSeries?.zh || ''
}

export const localizeTournamentTitle = (value) => {
  const title = String(value || '').trim()
  if (!title || CJK_PATTERN.test(title)) return title

  const championshipLeagueMatch = title.match(/^(?:.+?\s+)?Championship League Snooker\s+(\d{4})\s+\(([^)]+)\)$/i)
  if (championshipLeagueMatch) {
    const [, year, suffix] = championshipLeagueMatch
    return toTitleWithYearAndSuffix('冠军联赛', year, localizeLeagueStageSuffix(suffix))
  }

  const groupMatch = title.match(/^Championship League Group\s+(.+)$/i)
  if (groupMatch) {
    const groupNumber = normalizeOrdinalToken(groupMatch[1])
    return groupNumber ? `冠军联赛第${groupNumber}组` : '冠军联赛小组赛'
  }

  if (/^Championship League Winners Group$/i.test(title)) {
    return '冠军联赛胜者组'
  }

  const qSchoolEventMatch = title.match(/^Q School\s+(\d{4})\s*-\s*Event\s+(\d+)$/i)
  if (qSchoolEventMatch) {
    const [, year, eventNo] = qSchoolEventMatch
    return `Q School ${year} - 第${eventNo}站`
  }

  const year = extractTournamentYear(title)
  const titleWithoutYear = stripYearFromTitle(title, year)
  const seriesTitle = localizeTournamentSeries(titleWithoutYear)

  if (seriesTitle) {
    return toTitleWithYearAndSuffix(seriesTitle, year, '')
  }

  return title
}

const parseDateOnly = (value) => {
  if (!value) return null
  const parsed = new Date(`${value}T00:00:00+08:00`)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const parseDateTime = (value) => {
  if (!value) return null

  const normalized = typeof value === 'string' && value.includes('T')
    ? value
    : `${String(value).replace(' ', 'T')}+08:00`
  const parsed = new Date(normalized)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const formatMonthDay = (date) => {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return '日期待定'
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const normalizeRoundKey = (value) => {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

export const standardizeRoundText = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''

  return ROUND_NAME_ALIASES[normalizeRoundKey(text)] || text
}

export const formatEventDateRange = (startDate, endDate) => {
  const start = parseDateOnly(startDate)
  const end = parseDateOnly(endDate || startDate)

  if (!start && !end) return '日期待定'
  if (start && end && start.getTime() === end.getTime()) return formatMonthDay(start)
  if (!start) return formatMonthDay(end)
  if (!end) return formatMonthDay(start)
  return `${formatMonthDay(start)} - ${formatMonthDay(end)}`
}

export const formatEventTimeRange = (startTime, endTime, now = Date.now()) => {
  const start = parseDateTime(startTime)
  const end = parseDateTime(endTime)

  if (!start && !end) return ''
  if (start && end) {
    return `${formatEventNewsTime(start.getTime(), now)} - ${formatEventNewsTime(end.getTime(), now)}`
  }
  return formatEventNewsTime((start || end).getTime(), now)
}

export const buildEventLocationText = (item = {}) => {
  const city = typeof item.city === 'string' ? item.city.trim() : ''
  const venue = typeof item.venue === 'string' ? item.venue.trim() : ''

  if (city && venue) return `${city} · ${venue}`
  return city || venue || ''
}

export const normalizeSaiXunCard = (item = {}, now = Date.now()) => {
  const currentRoundText = standardizeRoundText(item.current_round_text)
  const matchCount = Number(item.match_count || 0)
  const timeText = formatEventTimeRange(item.start_time, item.end_time, now)
  const rawTitle = item.tournament_name || item.title || '赛事情报'

  return {
    ...item,
    id: item.id,
    title: localizeTournamentTitle(rawTitle) || rawTitle,
    summary: item.summary || item.latest_result_text || '官方赛讯持续更新中',
    statusText: getEventNewsStatusText(item.status, '赛讯更新中'),
    gameTypeText: getGameTypeLabel(item.game_type, '台球'),
    dateText: formatEventDateRange(item.start_date, item.end_date),
    timeText,
    showTime: Boolean(timeText),
    locationText: buildEventLocationText(item),
    currentRoundText: currentRoundText || '轮次待更新',
    resultText: item.latest_result_text || '',
    matchCount,
    matchCountText: matchCount > 0 ? `${matchCount} 场比赛` : '比赛待更新',
    sourceText: typeof item.source_name === 'string' ? item.source_name.trim() : '',
    coverImage: item.cover_image || DEFAULT_EVENT_COVER
  }
}

export const buildSaiXunHeroStats = (list = []) => ({
  totalCount: list.length,
  liveCount: list.filter((item) => Number(item?.status) === 1).length,
  upcomingCount: list.filter((item) => Number(item?.status) === 0).length
})

const splitDisplayName = (name) => {
  const text = String(name || '').trim()
  if (!text || text === '待定') {
    return { firstName: '', lastName: text || '待定' }
  }

  const parts = text.split(/\s+/).filter(Boolean)
  if (parts.length <= 1) {
    return { firstName: '', lastName: text }
  }

  return {
    firstName: parts.slice(0, -1).join(' '),
    lastName: parts[parts.length - 1]
  }
}

const resolvePlayerTextParts = (firstName, lastName, fullName) => {
  const normalizedFirstName = String(firstName || '').trim()
  const normalizedLastName = String(lastName || '').trim()

  if (normalizedFirstName || normalizedLastName) {
    return {
      firstName: normalizedFirstName,
      lastName: normalizedLastName || fullName || '待定'
    }
  }

  return splitDisplayName(fullName)
}

const buildMatchMetaText = (match) => {
  const parts = []
  if (Number(match.best_of || 0) > 0) parts.push(`Best of ${Number(match.best_of || 0)}`)
  if (match.is_placeholder) parts.push('占位赛程')
  return parts.join(' · ') || '等待比赛开始'
}

const buildScoreText = (match) => {
  const status = Number(match.status || 0)
  const scoreReady = status === 1 || status === 2 || Number(match.home_score || 0) > 0 || Number(match.away_score || 0) > 0
  return scoreReady ? `${Number(match.home_score || 0)} : ${Number(match.away_score || 0)}` : '-'
}

const buildPlayerResultText = (winnerSide, side) => {
  const normalizedWinnerSide = Number(winnerSide || 0)
  const normalizedSide = Number(side || 0)
  if (normalizedWinnerSide !== 1 && normalizedWinnerSide !== 2) return ''
  if (normalizedSide !== 1 && normalizedSide !== 2) return ''
  return normalizedWinnerSide === normalizedSide ? '胜' : '败'
}

const resolveEffectiveMatchStatus = (match = {}, startAt, now = Date.now()) => {
  const status = Number(match.status || 0)
  if (status !== 0) return status

  const nowTime = typeof now === 'string' ? new Date(now).getTime() : Number(now)
  const startTime = startAt?.getTime?.() ?? Number.POSITIVE_INFINITY
  const hasWinner = Number(match.winner_side || 0) > 0
  const hasScore = Number(match.home_score || 0) > 0 || Number(match.away_score || 0) > 0

  if (hasWinner && hasScore) return 2
  if (Number.isFinite(startTime) && Number.isFinite(nowTime) && startTime <= nowTime) {
    return nowTime - startTime <= MATCH_LIVE_FALLBACK_WINDOW_MS ? 1 : 2
  }
  return status
}

const normalizeSaiXunMatch = (match = {}, now = Date.now()) => {
  const startAt = parseDateTime(match.start_time)
  const effectiveStatus = resolveEffectiveMatchStatus(match, startAt, now)
  const homePlayerParts = resolvePlayerTextParts(match.home_player_first_name, match.home_player_last_name, match.home_player_name || '待定')
  const awayPlayerParts = resolvePlayerTextParts(match.away_player_first_name, match.away_player_last_name, match.away_player_name || '待定')
  return {
    id: match.id,
    roundName: standardizeRoundText(match.round_name) || '轮次待更新',
    roundOrder: Number(match.round_order || 0),
    matchOrder: Number(match.match_order || 0),
    status: effectiveStatus,
    statusText: getEventNewsStatusText(effectiveStatus, '待更新'),
    startAt,
    startTimeText: startAt ? formatEventNewsTime(startAt.getTime(), now) : '时间待定',
    homePlayerName: match.home_player_name || '待定',
    homePlayerFirstName: homePlayerParts.firstName,
    homePlayerLastName: homePlayerParts.lastName,
    homePlayerFlagEmoji: sanitizeFlagEmoji(match.home_player_flag_emoji),
    homePlayerCountryCode: (match.home_player_country_code || '').trim().toLowerCase(),
    homePlayerFlag: resolvePlayerFlag(
      sanitizeFlagEmoji(match.home_player_flag_emoji),
      (match.home_player_country_code || '').trim().toLowerCase()
    ),
    homePlayerAvatar: match.home_player_avatar || DEFAULT_PLAYER_AVATAR,
    homeResultText: buildPlayerResultText(match.winner_side, 1),
    awayPlayerName: match.away_player_name || '待定',
    awayPlayerFirstName: awayPlayerParts.firstName,
    awayPlayerLastName: awayPlayerParts.lastName,
    awayPlayerFlagEmoji: sanitizeFlagEmoji(match.away_player_flag_emoji),
    awayPlayerCountryCode: (match.away_player_country_code || '').trim().toLowerCase(),
    awayPlayerFlag: resolvePlayerFlag(
      sanitizeFlagEmoji(match.away_player_flag_emoji),
      (match.away_player_country_code || '').trim().toLowerCase()
    ),
    awayPlayerAvatar: match.away_player_avatar || DEFAULT_PLAYER_AVATAR,
    awayResultText: buildPlayerResultText(match.winner_side, 2),
    scoreText: buildScoreText(match),
    winnerSide: Number(match.winner_side || 0),
    metaText: buildMatchMetaText(match),
    isPlaceholder: Boolean(match.is_placeholder)
  }
}

const compareSaiXunMatches = (left, right) => {
  const leftStatus = Number(left.status || 0)
  const rightStatus = Number(right.status || 0)

  if (leftStatus !== rightStatus) {
    const priority = { 1: 0, 0: 1, 2: 2, 3: 3 }
    return (priority[leftStatus] ?? 9) - (priority[rightStatus] ?? 9)
  }

  const leftTime = left.startAt?.getTime() ?? Number.POSITIVE_INFINITY
  const rightTime = right.startAt?.getTime() ?? Number.POSITIVE_INFINITY

  if (MATCH_ACTIVE_STATUSES.has(leftStatus)) {
    if (leftTime !== rightTime) return leftTime - rightTime
  } else if (leftStatus === 2 || leftStatus === 3) {
    if (leftTime !== rightTime) return rightTime - leftTime
  }

  if (left.roundOrder !== right.roundOrder) return right.roundOrder - left.roundOrder
  if (left.matchOrder !== right.matchOrder) return left.matchOrder - right.matchOrder
  return Number(left.id || 0) - Number(right.id || 0)
}

export const buildSaiXunDetailRounds = (matches = [], now = Date.now()) => {
  const normalized = matches.map((item) => normalizeSaiXunMatch(item, now))
  const activeRoundOrders = normalized
    .filter((item) => MATCH_ACTIVE_STATUSES.has(item.status))
    .map((item) => item.roundOrder)
    .filter((value) => value > 0)
  const maxVisibleRoundOrder = activeRoundOrders.length ? Math.min(...activeRoundOrders) : Number.POSITIVE_INFINITY
  const visibleMatches = normalized
    .filter((item) => item.roundOrder <= maxVisibleRoundOrder)
    .sort(compareSaiXunMatches)

  const roundMap = new Map()
  visibleMatches.forEach((item) => {
    const key = `${item.roundOrder}-${item.roundName}`
    if (!roundMap.has(key)) {
      roundMap.set(key, {
        key,
        roundName: item.roundName,
        roundOrder: item.roundOrder,
        matches: []
      })
    }
    roundMap.get(key).matches.push(item)
  })

  return Array.from(roundMap.values())
}

export const localizeCountryName = (name) => {
  const text = String(name || '').trim()
  if (!text) return ''
  if (CJK_PATTERN.test(text)) return text
  return COUNTRY_NAME_MAP[text.toLowerCase()] || text
}

export const sanitizeFlagEmoji = (emoji) => {
  const text = String(emoji || '').trim()
  if (!text) return ''
  // Black flag base (U+1F3F4) — all subdivision flags are handled via local images
  if (text.codePointAt(0) === 0x1F3F4) return ''
  return text
}

const SUBDIVISION_FLAG_IMAGES = {
  eng: '/static/flags/eng.png',
  sct: '/static/flags/sco.png',
  wls: '/static/flags/wls.png'
}

const SUBDIVISION_LABELS = {
  eng: 'ENG',
  sct: 'SCO',
  wls: 'WLS'
}

/**
 * @param {string} flagEmoji
 * @param {string} countryCode
 * @returns {{ type: 'image'|'emoji'|'text'|'none', value: string }}
 */
export const resolvePlayerFlag = (flagEmoji, countryCode) => {
  const code = (countryCode || '').trim().toLowerCase()
  if (code === 'gb-eng' || code === 'eng') {
    return { type: 'image', value: SUBDIVISION_FLAG_IMAGES.eng }
  }
  if (code === 'gb-sct' || code === 'sct') {
    return { type: 'image', value: SUBDIVISION_FLAG_IMAGES.sct }
  }
  if (code === 'gb-wls' || code === 'wls') {
    return { type: 'image', value: SUBDIVISION_FLAG_IMAGES.wls }
  }
  const emoji = (flagEmoji || '').trim()
  if (emoji) {
    return { type: 'emoji', value: emoji }
  }
  return { type: 'none', value: '' }
}
