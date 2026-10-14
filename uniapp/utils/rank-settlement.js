/**
 * 约球排位结算展示逻辑：排位结果在手动结束后延后生效，期间另一方可提出一次异议。
 * 状态与是否可异议均以服务端字段为准，这里只做展示文案与入口判定。
 */

export const RANK_SETTLEMENT_STATUS = {
  NONE: 'none',
  PENDING: 'pending',
  SETTLED: 'settled',
  VOIDED: 'voided'
}

// 与后端 model.RankDisputeWindow 保持一致，仅用于滑块说明文案。
export const RANK_DISPUTE_WINDOW_MINUTES = 10

const BEIJING_OFFSET_MS = 8 * 60 * 60 * 1000

const normalizeStatus = (value) => {
  switch (value) {
    case RANK_SETTLEMENT_STATUS.PENDING:
    case RANK_SETTLEMENT_STATUS.SETTLED:
    case RANK_SETTLEMENT_STATUS.VOIDED:
      return value
    default:
      return RANK_SETTLEMENT_STATUS.NONE
  }
}

/** 把服务端 RFC3339 时刻格式化为北京时间 HH:MM；无法解析返回空串。 */
export const formatSettlementClock = (value) => {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return ''
  const shifted = new Date(timestamp + BEIJING_OFFSET_MS)
  const hours = String(shifted.getUTCHours()).padStart(2, '0')
  const minutes = String(shifted.getUTCMinutes()).padStart(2, '0')
  return `${hours}:${minutes}`
}

const viewerUserIdOf = (match) => {
  if (!match || match.viewer_role === 'referee') return 0
  const id = match.is_player1 ? match.player1_id : match.player2_id
  return Number(id) > 0 ? Number(id) : 0
}

/**
 * 解析赛果页的排位结算视图。
 * @param {Object} match getMatchDetail 返回的 match
 * @returns {{
 *   visible: boolean, status: string, hideRankChange: boolean,
 *   statusText: string, rankValueText: string, rankNote: string,
 *   finisherText: string, canDispute: boolean, disputeHint: string
 * }}
 */
export const resolveRankSettlementView = (match) => {
  const status = normalizeStatus(match?.rank_settlement_status)
  const clock = formatSettlementClock(match?.rank_settlement_due_at)
  // 只有走过异议期的比赛（排位手动结束）才带有生效时刻；赛制达标自动结束即时结算，沿用普通排位展示。
  const hasWindow = Boolean(match?.rank_settlement_due_at) && status !== RANK_SETTLEMENT_STATUS.NONE
  const view = {
    visible: false,
    status,
    hideRankChange: false,
    statusText: '',
    rankValueText: '',
    rankNote: '',
    finisherText: '',
    canDispute: false,
    disputeHint: ''
  }
  if (!hasWindow) return view

  const viewerId = viewerUserIdOf(match)
  const finisherId = Number(match.completed_by_user_id) || 0
  let finisherName = ''
  if (finisherId > 0) {
    if (finisherId === viewerId) finisherName = '你'
    else finisherName = (viewerId > 0 ? match.opponent_name : '') || '对方'
  }
  view.visible = true
  view.finisherText = finisherName ? `${finisherName}滑动结束` : ''

  if (status === RANK_SETTLEMENT_STATUS.PENDING) {
    view.hideRankChange = true
    view.statusText = clock ? `${clock} 生效` : '待生效'
    view.rankValueText = '待生效'
    view.rankNote = clock ? `排位积分将于 ${clock} 生效。` : '排位积分将在异议期结束后生效。'
    view.canDispute = match.can_dispute_result === true
    if (view.canDispute) {
      view.disputeHint = `${finisherName || '对方'}结束了本场。对结果有异议，可在 ${clock || '异议期结束'} 前提出；提出后本场不计排位积分，比分与记录保留。`
    }
  } else if (status === RANK_SETTLEMENT_STATUS.VOIDED) {
    view.hideRankChange = true
    view.statusText = '有异议 · 不计排位积分'
    view.rankValueText = '不计分'
    view.rankNote = '本场已提出异议，不计排位积分。'
  } else {
    view.statusText = '已生效'
  }
  return view
}

/**
 * 滑动结束处的说明文案：写明无需对方确认；排位胜负结果还写明按当前比分结算及异议期。
 * @param {{ challengeId: number, matchMode: string, myScore: number, opponentScore: number }} input
 */
export const buildFinishSlideNote = ({ challengeId, matchMode, myScore, opponentScore } = {}) => {
  const base = '松手即结束，无需对方确认。'
  const decisive = Number.isFinite(myScore) && Number.isFinite(opponentScore) && myScore !== opponentScore
  if (!(Number(challengeId) > 0) || matchMode !== 'ranked' || !decisive) return base
  return `${base}按 ${myScore} : ${opponentScore} 结算排位，对方${RANK_DISPUTE_WINDOW_MINUTES}分钟内可提出异议。`
}
