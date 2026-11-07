import { getGameTypeLabel } from './game-types.js'
import { resolveCdnAssetUrl } from './cdn-asset.js'
import { formatEventNewsTime, getEventNewsStatusText } from './home-index.js'
import { DEFAULT_USER_AVATAR } from './user-profile.js'

export const DEFAULT_EVENT_COVER = resolveCdnAssetUrl('/static/images/default-event-cover.png')
export const DEFAULT_PLAYER_AVATAR = DEFAULT_USER_AVATAR
// 官方赛讯球员无头像时的中性占位图（WST 官方风格剪影），不用平台自定义台球头像
export const OFFICIAL_PLAYER_AVATAR_PLACEHOLDER = '/static/images/wst-player-placeholder.png'

const MATCH_LIVE_FALLBACK_WINDOW_MS = 12 * 60 * 60 * 1000
const DAY_MS = 24 * 60 * 60 * 1000
const WEEKDAY_LABELS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
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
  { pattern: /\bShenzhen Open\b/i, zh: '深圳公开赛' },
  { pattern: /\bChina Open\b/i, zh: '中国公开赛' },
  { pattern: /\bXi'an Grand Prix\b/i, zh: '西安大奖赛' },
  { pattern: /\bWorld Mixed Doubles\b/i, zh: '世界混双赛' },
  { pattern: /\bWorld Masters of Snooker\b/i, zh: '世界斯诺克大师赛' },
  { pattern: /\bAsia\s*[&-]\s*Oceania Q School\b/i, zh: '亚洲及大洋洲 Q School' },
  { pattern: /\bQ School\b/i, zh: 'Q School' },
  { pattern: /\bShoot Out\b/i, zh: '单局限时赛' },
  { pattern: /\bMasters\b/i, zh: '大师赛' }
])

// 资格赛 / 正赛阶段与主赛事同名，翻译时必须保留后缀，避免列表出现两张同名卡片
const TOURNAMENT_PHASE_SUFFIXES = Object.freeze([
  { pattern: /\s+Qualifiers?$/i, zh: '资格赛' },
  { pattern: /\s+Final Stages?$/i, zh: '决赛阶段' }
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
    .replace(/\bW(?:EE)?K\s*(\d+)\b/gi, '第$1周')
    .replace(/\bWinners Group\b/gi, '胜者组')
    .replace(/\bGroup\s+(\d+)\b/gi, '第$1组')
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

  const championshipLeagueMatch = title.match(/^(?:.+?\s+)?Championship League Snooker(?:\s+Invitational)?\s+(\d{4})\s+\(([^)]+)\)$/i)
  if (championshipLeagueMatch) {
    const [, year, suffix] = championshipLeagueMatch
    return toTitleWithYearAndSuffix('冠军联赛', year, localizeLeagueStageSuffix(suffix))
  }

  const groupMatch = title.match(/^(?:.+?\s+)?Championship League Group\s+(.+)$/i)
  if (groupMatch) {
    const groupNumber = normalizeOrdinalToken(groupMatch[1])
    return groupNumber ? `冠军联赛第${groupNumber}组` : '冠军联赛小组赛'
  }

  if (/^(?:.+?\s+)?Championship League Winners Group$/i.test(title)) {
    return '冠军联赛胜者组'
  }

  const qSchoolEventMatch = title.match(/^Q School\s+(\d{4})\s*-\s*Event\s+(\d+)$/i)
  if (qSchoolEventMatch) {
    const [, year, eventNo] = qSchoolEventMatch
    return `Q School ${year} - 第${eventNo}站`
  }

  const phase = TOURNAMENT_PHASE_SUFFIXES.find(({ pattern }) => pattern.test(title))
  const titleWithoutPhase = phase ? title.replace(phase.pattern, '') : title
  const year = extractTournamentYear(titleWithoutPhase)
  const titleWithoutYear = stripYearFromTitle(titleWithoutPhase, year)
  const seriesTitle = localizeTournamentSeries(titleWithoutYear)

  if (seriesTitle) {
    const localized = toTitleWithYearAndSuffix(seriesTitle, year, '')
    return phase ? `${localized} ${phase.zh}` : localized
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

const localizeRoundQualifier = (value) => {
  const text = String(value || '').trim()
  if (/^held over$/i.test(text)) return '保留赛'
  return localizeLeagueStageSuffix(text)
}

export const standardizeRoundText = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''

  const alias = ROUND_NAME_ALIASES[normalizeRoundKey(text)]
  if (alias) return alias

  const roundMatch = text.match(/^Round\s+(\d+)(?:\s*\(([^)]+)\))?$/i)
  if (roundMatch) {
    const [, roundNo, qualifier] = roundMatch
    return qualifier ? `第${roundNo}轮（${localizeRoundQualifier(qualifier)}）` : `第${roundNo}轮`
  }

  const lastMatch = text.match(/^Last\s+(\d+)$/i)
  if (lastMatch) return `${lastMatch[1]}强`

  const leagueMatch = text.match(/^League Phase\s*\(([^)]+)\)$/i)
  if (leagueMatch) return `联赛阶段（${localizeLeagueStageSuffix(leagueMatch[1])}）`

  if (CJK_PATTERN.test(text)) return text
  return localizeLeagueStageSuffix(text)
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

export const buildEventRegionText = (item = {}) => {
  const parts = [localizeCountryName(item.country), item.city, item.venue]
    .map((value) => String(value || '').trim())
    .filter(Boolean)
  return Array.from(new Set(parts)).join(' · ')
}

const toTimeValue = (now) => (typeof now === 'string' ? new Date(now).getTime() : Number(now))

const diffLocalDays = (date, now) => {
  const nowDate = new Date(toTimeValue(now))
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  const today = new Date(nowDate.getFullYear(), nowDate.getMonth(), nowDate.getDate()).getTime()
  return Math.round((target - today) / DAY_MS)
}

export const formatEventCountdownText = (startDate, now = Date.now()) => {
  const start = parseDateOnly(startDate)
  if (!start) return '即将开始'

  const diffDays = diffLocalDays(start, now)
  if (diffDays < 0) return '即将开始'
  if (diffDays === 0) return '今天开赛'
  if (diffDays === 1) return '明天开赛'
  if (diffDays <= 30) return `${diffDays}天后开赛`
  return `${formatMonthDay(start)}开赛`
}

const SUMMARY_SCORE_PATTERN = /^(.+?)\s+(\d+)\s+-\s+(\d+)\s+(.+)$/
const SUMMARY_PAIR_PATTERN = /^(.+?)\s+-\s+(.+)$/

// 后端 latest_result_text 形如 "主队 9 - 6 客队"（已开赛）或 "主队 - 客队"（未开赛）
const parseMatchSummaryText = (value) => {
  const text = String(value || '').trim()
  const scored = text.match(SUMMARY_SCORE_PATTERN)
  if (scored) {
    return { home: scored[1], away: scored[4], homeScore: Number(scored[2]), awayScore: Number(scored[3]), hasScore: true }
  }
  const pair = text.match(SUMMARY_PAIR_PATTERN)
  if (pair) return { home: pair[1], away: pair[2], hasScore: false }
  return null
}

const buildSaiXunCardHighlight = (status, roundText, latestResultText) => {
  const summary = parseMatchSummaryText(latestResultText)
  if (!summary) return null

  if (status === 2) {
    if (roundText !== '决赛' || !summary.hasScore || summary.homeScore === summary.awayScore) return null
    const homeWon = summary.homeScore > summary.awayScore
    return {
      type: 'champion',
      label: '冠军',
      primary: homeWon ? summary.home : summary.away,
      secondary: `决赛 ${Math.max(summary.homeScore, summary.awayScore)}-${Math.min(summary.homeScore, summary.awayScore)} 胜 ${homeWon ? summary.away : summary.home}`
    }
  }

  if (status !== 0 && status !== 1) return null
  return {
    type: status === 1 ? 'live' : 'upcoming',
    label: roundText,
    primary: summary.hasScore
      ? `${summary.home} ${summary.homeScore} - ${summary.awayScore} ${summary.away}`
      : `${summary.home} vs ${summary.away}`,
    secondary: ''
  }
}

const SAIXUN_STATUS_TONES = Object.freeze({ 0: 'upcoming', 1: 'live', 2: 'ended', 3: 'canceled' })

export const normalizeSaiXunCard = (item = {}, now = Date.now()) => {
  const status = Number(item.status)
  const currentRoundText = standardizeRoundText(item.current_round_text)
  const matchCount = Number(item.match_count || 0)
  const timeText = formatEventTimeRange(item.start_time, item.end_time, now)
  const rawTitle = item.tournament_name || item.title || '赛事情报'
  const title = localizeTournamentTitle(rawTitle) || rawTitle
  const dateText = formatEventDateRange(item.start_date, item.end_date)
  const regionText = buildEventRegionText(item)

  return {
    ...item,
    id: item.id,
    title,
    originalTitle: title !== rawTitle ? rawTitle : '',
    summary: item.summary || item.latest_result_text || '官方赛讯持续更新中',
    statusText: getEventNewsStatusText(item.status, '赛讯更新中'),
    statusTone: SAIXUN_STATUS_TONES[status] || 'ended',
    statusLabel: status === 0 ? formatEventCountdownText(item.start_date, now) : getEventNewsStatusText(item.status, '赛讯更新中'),
    isFeatured: status === 0 || status === 1,
    gameTypeText: getGameTypeLabel(item.game_type, '台球'),
    dateText,
    timeText,
    showTime: Boolean(timeText),
    locationText: buildEventLocationText(item),
    regionText,
    metaText: [dateText, regionText].filter(Boolean).join(' · '),
    currentRoundText: currentRoundText || '轮次待更新',
    resultText: item.latest_result_text || '',
    highlight: buildSaiXunCardHighlight(status, currentRoundText, item.latest_result_text),
    matchCount,
    matchCountText: matchCount > 0 ? `${matchCount} 场比赛` : '比赛待更新',
    sourceText: typeof item.source_name === 'string' ? item.source_name.trim() : '',
    hasCover: Boolean(item.cover_image),
    coverImage: item.cover_image || DEFAULT_EVENT_COVER
  }
}

const resolveSaiXunSection = (item = {}) => {
  const status = Number(item.status)
  if (status === 1) return { key: 'live', title: '进行中', caption: '' }
  if (status === 0) return { key: 'upcoming', title: '即将开始', caption: '' }
  if (status === 3) return { key: 'canceled', title: '已取消', caption: '' }

  const start = parseDateOnly(item.start_date)
  if (!start) return { key: 'ended', title: '已结束', caption: '' }
  return { key: `ended-${start.getFullYear()}-${start.getMonth() + 1}`, title: `${start.getMonth() + 1}月`, caption: '已结束' }
}

// 列表按后端顺序（进行中 → 即将开始 → 已结束按时间倒序）切分分组，只合并相邻同组项
export const buildSaiXunListSections = (list = []) => {
  const sections = []
  list.forEach((item) => {
    const section = resolveSaiXunSection(item)
    const last = sections[sections.length - 1]
    if (last && last.groupKey === section.key) {
      last.items.push(item)
      return
    }
    sections.push({
      key: `${section.key}-${sections.length}`,
      groupKey: section.key,
      title: section.title,
      caption: section.caption,
      items: [item]
    })
  })
  return sections
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

// 中文赛制习惯写作"9局5胜"；偶数局制（如冠军联赛小组赛）可能打平，只标总局数
export const formatBestOfText = (bestOf) => {
  const frames = Number(bestOf || 0)
  if (!Number.isInteger(frames) || frames <= 0) return ''
  if (frames % 2 === 0) return `共${frames}局`
  return `${frames}局${Math.ceil(frames / 2)}胜`
}

// 只按源数据状态判断比分是否可信，按开赛时间推断出的进行中不展示 0:0
const isMatchScoreReady = (match) => {
  const status = Number(match.status || 0)
  return status === 1 || status === 2 || Number(match.home_score || 0) > 0 || Number(match.away_score || 0) > 0
}

const buildScoreText = (match) => {
  return isMatchScoreReady(match) ? `${Number(match.home_score || 0)} : ${Number(match.away_score || 0)}` : '-'
}

const pad2 = (value) => String(value).padStart(2, '0')

const buildDayKey = (date) => `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`

const formatMatchDayLabel = (date, now) => {
  const base = `${date.getMonth() + 1}月${date.getDate()}日 ${WEEKDAY_LABELS[date.getDay()]}`
  const diffDays = diffLocalDays(date, now)
  if (diffDays === 0) return `${base} · 今天`
  if (diffDays === 1) return `${base} · 明天`
  if (diffDays === -1) return `${base} · 昨天`
  return base
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
  // 官方赛事球员不用平台自定义风格头像，缺头像时用中性官方占位图
  const playerAvatarFallback = match.source_type === 'official'
    ? OFFICIAL_PLAYER_AVATAR_PLACEHOLDER
    : DEFAULT_PLAYER_AVATAR
  const showScore = isMatchScoreReady(match)
  return {
    id: match.id,
    roundName: standardizeRoundText(match.round_name) || '轮次待更新',
    roundOrder: Number(match.round_order || 0),
    matchOrder: Number(match.match_order || 0),
    status: effectiveStatus,
    statusText: getEventNewsStatusText(effectiveStatus, '待更新'),
    isLive: effectiveStatus === 1,
    startAt,
    startTimeText: startAt ? formatEventNewsTime(startAt.getTime(), now) : '时间待定',
    clockText: startAt ? `${pad2(startAt.getHours())}:${pad2(startAt.getMinutes())}` : '时间待定',
    dayKey: startAt ? buildDayKey(startAt) : 'unscheduled',
    dayLabel: startAt ? formatMatchDayLabel(startAt, now) : '时间待定',
    homePlayerName: match.home_player_name || '待定',
    homePlayerFirstName: homePlayerParts.firstName,
    homePlayerLastName: homePlayerParts.lastName,
    homePlayerFlagEmoji: sanitizeFlagEmoji(match.home_player_flag_emoji),
    homePlayerCountryCode: (match.home_player_country_code || '').trim().toLowerCase(),
    homePlayerFlag: resolvePlayerFlag(
      sanitizeFlagEmoji(match.home_player_flag_emoji),
      (match.home_player_country_code || '').trim().toLowerCase()
    ),
    homePlayerAvatar: match.home_player_avatar || playerAvatarFallback,
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
    awayPlayerAvatar: match.away_player_avatar || playerAvatarFallback,
    awayResultText: buildPlayerResultText(match.winner_side, 2),
    showScore,
    homeScore: Number(match.home_score || 0),
    awayScore: Number(match.away_score || 0),
    scoreText: buildScoreText(match),
    winnerSide: Number(match.winner_side || 0),
    metaText: formatBestOfText(match.best_of),
    isPlaceholder: Boolean(match.is_placeholder)
  }
}

const getMatchStartValue = (match) => match.startAt?.getTime() ?? Number.POSITIVE_INFINITY

// 轮次内按开赛时间排列，便于按比赛日分组浏览
const compareSaiXunMatches = (left, right) => {
  const leftTime = getMatchStartValue(left)
  const rightTime = getMatchStartValue(right)
  if (leftTime !== rightTime) return leftTime - rightTime
  if (left.matchOrder !== right.matchOrder) return left.matchOrder - right.matchOrder
  return Number(left.id || 0) - Number(right.id || 0)
}

const compareSaiXunRounds = (left, right) => {
  if (left.roundOrder !== right.roundOrder) return left.roundOrder - right.roundOrder
  return getMatchStartValue(left.matches[0]) - getMatchStartValue(right.matches[0])
}

const groupMatchesByDay = (matches) => {
  const days = []
  matches.forEach((match) => {
    const last = days[days.length - 1]
    if (last && last.key === match.dayKey) {
      last.matches.push(match)
      return
    }
    days.push({ key: match.dayKey, label: match.dayLabel, matches: [match] })
  })
  return days
}

const resolveRoundMetaText = (matches) => {
  const formats = new Set(matches.map((match) => match.metaText).filter(Boolean))
  return formats.size === 1 ? Array.from(formats)[0] : ''
}

// 按轮次先后（含 round_order 相同的保留赛）返回全部轮次；全部为占位对阵的轮次没有可看信息，不展示
export const buildSaiXunDetailRounds = (matches = [], now = Date.now()) => {
  const roundMap = new Map()
  matches.map((item) => normalizeSaiXunMatch(item, now)).forEach((item) => {
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
    .filter((round) => round.matches.some((match) => !match.isPlaceholder))
    .map((round) => {
      const sortedMatches = [...round.matches].sort(compareSaiXunMatches)
      return {
        ...round,
        matches: sortedMatches,
        matchCount: sortedMatches.length,
        hasLive: sortedMatches.some((match) => match.isLive),
        metaText: resolveRoundMetaText(sortedMatches),
        days: groupMatchesByDay(sortedMatches)
      }
    })
    .sort(compareSaiXunRounds)
}

// 默认展示最值得看的轮次：有直播的最新轮次 → 下一轮待赛 → 最后一轮（通常是决赛）
export const resolveSaiXunFocusRoundKey = (rounds = []) => {
  if (!rounds.length) return ''

  const liveRound = [...rounds].reverse().find((round) => round.hasLive)
  if (liveRound) return liveRound.key

  const upcomingRound = rounds.find((round) => round.matches.some((match) => match.status === 0 && !match.isPlaceholder))
  if (upcomingRound) return upcomingRound.key

  return rounds[rounds.length - 1].key
}

export const resolveSaiXunChampion = (rounds = []) => {
  const finalRound = rounds.find((round) => round.roundName === '决赛')
  const finalMatch = finalRound?.matches.find((match) => match.status === 2 && (match.winnerSide === 1 || match.winnerSide === 2))
  if (!finalMatch) return null

  const homeWon = finalMatch.winnerSide === 1
  return {
    name: homeWon ? finalMatch.homePlayerName : finalMatch.awayPlayerName,
    avatar: homeWon ? finalMatch.homePlayerAvatar : finalMatch.awayPlayerAvatar,
    flag: homeWon ? finalMatch.homePlayerFlag : finalMatch.awayPlayerFlag,
    opponentName: homeWon ? finalMatch.awayPlayerName : finalMatch.homePlayerName,
    scoreText: `${Math.max(finalMatch.homeScore, finalMatch.awayScore)}-${Math.min(finalMatch.homeScore, finalMatch.awayScore)}`
  }
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
  if (code === 'gb-nir' || code === 'nir') {
    return { type: 'image', value: '/static/flags/nir.png' }
  }
  const emoji = (flagEmoji || '').trim()
  if (emoji) {
    return { type: 'emoji', value: emoji }
  }
  return { type: 'none', value: '' }
}
