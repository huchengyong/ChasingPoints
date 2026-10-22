// 单场复盘页的展示规则。数据来自服务端 GET /api/member/match-replay，统一为「我 / 对手」视角，这里只负责文案与图表比例。
import { getGameTypeLabel } from './game-types.js'

const toNumber = (value) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const toCount = (value) => Math.max(0, Math.trunc(toNumber(value)))

export const GAME_TYPE_SNOOKER = 1
export const GAME_TYPE_NINE_BALL = 2
export const GAME_TYPE_CHINESE_EIGHT = 3
export const GAME_TYPE_AMERICAN_NINE = 4

const WIN_TYPE_LABELS = {
  normal: '普胜',
  break_clear: '炸清',
  continue_clear: '接清',
  small_gold: '小金',
  big_gold: '大金'
}

const KEY_TAG_LABELS = {
  decider: '决胜局',
  comeback: '逆转局'
}

export const REPLAY_MAX_TREND_COLUMNS = 60

const isPoolGame = (gameType) => gameType === GAME_TYPE_CHINESE_EIGHT || gameType === GAME_TYPE_AMERICAN_NINE

export const formatFrameDuration = (seconds) => {
  const total = toCount(seconds)
  if (!total) return ''
  const minutes = Math.floor(total / 60)
  const rest = total % 60
  if (minutes === 0) return `${rest}秒`
  return rest === 0 ? `${minutes}分` : `${minutes}分${rest}秒`
}

/**
 * 以 0 为轴的走势柱：正数（我方领先）向上，负数向下。高度按本局最大绝对分差归一，最小保留可见高度。
 */
export const buildTrendColumns = (trend = []) => {
  const points = (Array.isArray(trend) ? trend : []).slice(0, REPLAY_MAX_TREND_COLUMNS)
  const maxAbs = points.reduce((max, point) => Math.max(max, Math.abs(toNumber(point?.diff))), 0)
  return {
    hasData: points.length > 1 && maxAbs > 0,
    maxAbs,
    columns: points.map((point) => {
      const diff = Math.trunc(toNumber(point?.diff))
      return {
        step: Math.trunc(toNumber(point?.step)),
        diff,
        direction: diff > 0 ? 'up' : diff < 0 ? 'down' : 'flat',
        heightPercent: maxAbs > 0 ? (Math.abs(diff) / maxAbs) * 100 : 0
      }
    })
  }
}

const resolveFrameWinner = (winner) => (winner === 1 ? 'me' : winner === 2 ? 'opponent' : 'none')

const resolveFrameView = (frame, gameType, winTypeLabels) => {
  const winner = resolveFrameWinner(toNumber(frame?.winner))
  const tags = (Array.isArray(frame?.key_tags) ? frame.key_tags : [])
    .map((tag) => ({ key: tag, label: KEY_TAG_LABELS[tag] || '' }))
    .filter((tag) => tag.label)
  const pool = isPoolGame(gameType)
  const details = []
  const myRun = toCount(frame?.my_max_run)
  const opponentRun = toCount(frame?.opponent_max_run)
  if (myRun > 0 || opponentRun > 0) {
    details.push({
      label: gameType === GAME_TYPE_SNOOKER ? '单杆最高' : '最大连续得分',
      mine: String(myRun),
      opponent: String(opponentRun)
    })
  }
  const myFouls = toCount(frame?.my_foul_count)
  const opponentFouls = toCount(frame?.opponent_foul_count)
  if (gameType === GAME_TYPE_SNOOKER && (myFouls > 0 || opponentFouls > 0)) {
    details.push({
      label: '犯规（次 / 罚分）',
      mine: `${myFouls} / ${toCount(frame?.my_foul_points)}`,
      opponent: `${opponentFouls} / ${toCount(frame?.opponent_foul_points)}`
    })
  }
  const maxDeficit = toCount(frame?.max_deficit)
  const trend = buildTrendColumns(frame?.trend)
  return {
    frameNo: toCount(frame?.frame_no),
    title: `第 ${toCount(frame?.frame_no)} 局`,
    winner,
    winnerText: winner === 'me' ? '我赢' : winner === 'opponent' ? '对手赢' : '未分胜负',
    // 中式八球、美式九球每局只记胜负，没有比分可言。
    scoreText: pool ? '' : `${toCount(frame?.my_score)} : ${toCount(frame?.opponent_score)}`,
    winTypeText: frame?.win_type ? (winTypeLabels[frame.win_type] || WIN_TYPE_LABELS[frame.win_type] || '') : '',
    durationText: formatFrameDuration(frame?.duration_seconds),
    deficitText: maxDeficit > 0 && winner !== 'none' ? `${winner === 'me' ? '我' : '对手'}曾落后 ${maxDeficit} 分后赢下` : '',
    tags,
    isKey: tags.length > 0,
    details,
    trend
  }
}

const buildSummaryRows = (summary, gameType) => {
  if (!summary) return []
  const rows = [{
    label: '赢下局数',
    mine: String(toCount(summary.my_frame_wins)),
    opponent: String(toCount(summary.opponent_frame_wins))
  }]
  const myRun = toCount(summary.my_max_run)
  const opponentRun = toCount(summary.opponent_max_run)
  if (myRun > 0 || opponentRun > 0) {
    rows.push({ label: gameType === GAME_TYPE_SNOOKER ? '单杆最高' : '最大连续得分', mine: String(myRun), opponent: String(opponentRun) })
  }
  const myStreak = toCount(summary.my_longest_streak)
  const opponentStreak = toCount(summary.opponent_longest_streak)
  if (isPoolGame(gameType) && (myStreak > 0 || opponentStreak > 0)) {
    rows.push({ label: '最长连胜（局）', mine: String(myStreak), opponent: String(opponentStreak) })
  }
  return rows
}

/**
 * 中式八球、美式九球的逐局胜负序列（我赢 / 对手赢）。
 */
export const buildFrameSequence = (frames = []) => (
  (Array.isArray(frames) ? frames : []).map((frame) => ({
    frameNo: toCount(frame?.frame_no),
    winner: resolveFrameWinner(toNumber(frame?.winner))
  }))
)

/**
 * 复盘页的完整展示模型。locked 时只带局数，不含任何明细。
 */
export const resolveReplayView = (response, { canShowPurchase = false } = {}) => {
  const gameType = toCount(response?.game_type)
  const frameCount = toCount(response?.frame_count)
  const base = { gameType, gameTypeLabel: getGameTypeLabel(gameType), frameCount }
  if (response?.locked) {
    return {
      ...base,
      locked: true,
      showPurchase: Boolean(canShowPurchase),
      features: [
        '每局走势：谁领先、什么时候被追上',
        '单杆与连续得分、犯规统计',
        '逐局胜负序列与胜局方式',
        '决胜局与逆转局标注'
      ]
    }
  }
  const frames = Array.isArray(response?.frames) ? response.frames : []
  const winTypeLabels = Object.fromEntries(
    (Array.isArray(response?.summary?.win_type_stats) ? response.summary.win_type_stats : [])
      .filter((item) => item?.win_type && item?.label)
      .map((item) => [item.win_type, item.label])
  )
  const frameViews = frames.map((frame) => resolveFrameView(frame, gameType, winTypeLabels))
  const truncated = Boolean(response?.detail_truncated)
  const frameOnly = Boolean(response?.frame_only)
  return {
    ...base,
    locked: false,
    showPurchase: false,
    title: `${getGameTypeLabel(gameType)}复盘`,
    scoreText: `${toCount(response?.summary?.my_frame_wins)} : ${toCount(response?.summary?.opponent_frame_wins)}`,
    summaryRows: buildSummaryRows(response?.summary, gameType),
    winTypeRows: (Array.isArray(response?.summary?.win_type_stats) ? response.summary.win_type_stats : [])
      .map((item) => ({ label: item?.label || '', mine: String(toCount(item?.my_count)), opponent: String(toCount(item?.opponent_count)) }))
      .filter((row) => row.label),
    sequence: isPoolGame(gameType) ? buildFrameSequence(frames) : [],
    frames: frameViews,
    keyFrames: frameViews.filter((frame) => frame.isKey),
    notice: truncated
      ? '这场对局的记录太长，只显示每局结果。'
      : frameOnly
        ? '这场对局没有逐杆记录，只显示每局结果。'
        : '',
    hasDetail: !truncated && !frameOnly,
    empty: frameViews.length === 0
  }
}
