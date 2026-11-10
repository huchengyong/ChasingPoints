// 九球追分得分构成的展示模型。数据固定按选手1 / 选手2 下发，这里只负责视角换边与文案。
const toNumber = (value) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const KIND_LABELS = {
  normal: '普胜',
  small_gold: '小金',
  big_gold: '大金',
  foul: '犯规',
  other: '其他'
}

const SUMMARY_ROWS = [
  { key: 'score', label: '总分' },
  { key: 'normal_wins', label: '普胜' },
  { key: 'small_golds', label: '小金' },
  { key: 'big_golds', label: '大金' },
  { key: 'foul_count', label: '犯规' },
  { key: 'foul_points_given', label: '犯规送分' },
  { key: 'other_points', label: '其他得分' }
]

// 服务端时间形如 2026-11-10 12:34:56，没有时间时为空串。
const toTimeText = (createdAt) => {
  const matched = typeof createdAt === 'string' ? createdAt.match(/(\d{2}:\d{2}:\d{2})/) : null
  return matched ? matched[1] : ''
}

/**
 * @param {object|null} breakdown 服务端 score_breakdown
 * @param {object} options swap=查看者是选手2；neutral=裁判 / 观战者，固定选手1 / 选手2 且不换边
 */
export const resolveChasingScoreView = (breakdown, { swap = false, neutral = false } = {}) => {
  if (!breakdown || typeof breakdown !== 'object' || Array.isArray(breakdown)) return null

  const available = Boolean(breakdown.available)
  const swapSides = Boolean(swap) && !neutral
  const leftLabel = neutral ? '选手1' : '我'
  const rightLabel = neutral ? '选手2' : '对手'
  const player1 = breakdown.player1 && typeof breakdown.player1 === 'object' ? breakdown.player1 : {}
  const player2 = breakdown.player2 && typeof breakdown.player2 === 'object' ? breakdown.player2 : {}
  const leftSide = swapSides ? player2 : player1
  const rightSide = swapSides ? player1 : player2

  const rows = []
  if (available) {
    SUMMARY_ROWS.forEach(({ key, label }) => {
      if (key === 'other_points' && toNumber(player1.other_points) <= 0 && toNumber(player2.other_points) <= 0) return
      rows.push({ key, label, left: String(toNumber(leftSide[key])), right: String(toNumber(rightSide[key])) })
    })
  } else {
    rows.push({ key: 'score', label: '总分', left: String(toNumber(leftSide.score)), right: String(toNumber(rightSide.score)) })
  }

  const events = available && Array.isArray(breakdown.events)
    ? breakdown.events
      .slice()
      .sort((a, b) => toNumber(b?.seq) - toNumber(a?.seq))
      .map((event) => {
        const kind = typeof event?.kind === 'string' ? event.kind : ''
        const kindLabel = KIND_LABELS[kind] || KIND_LABELS.other
        // 犯规条目的 scorer 是非犯规方即得分方。
        const scorer = toNumber(event?.scorer) === 2 ? 2 : 1
        const side = (scorer === 1) !== swapSides ? 'left' : 'right'
        const sideLabel = side === 'left' ? leftLabel : rightLabel
        const foulSideLabel = side === 'left' ? rightLabel : leftLabel
        // 事件分数与 rows 的左右一致。
        const eventLeftScore = toNumber(swapSides ? event?.player2_score : event?.player1_score)
        const eventRightScore = toNumber(swapSides ? event?.player1_score : event?.player2_score)
        return {
          seq: toNumber(event?.seq),
          kind,
          kindLabel,
          side,
          sideLabel,
          // 犯规的分记给对方，写明得分方，避免读成犯规方得分。
          pointsText: kind === 'foul' ? `${sideLabel} +${toNumber(event?.points)}` : `+${toNumber(event?.points)}`,
          scoreText: `${eventLeftScore} : ${eventRightScore}`,
          timeText: toTimeText(event?.created_at),
          desc: kind === 'foul'
            ? `${foulSideLabel}犯规`
            : kind === 'other'
              ? `${sideLabel} 得分`
              : `${sideLabel} ${kindLabel}`
        }
      })
    : []

  return {
    available,
    leftLabel,
    rightLabel,
    leftScore: toNumber(leftSide.score),
    rightScore: toNumber(rightSide.score),
    rows,
    events,
    eventsTruncated: Boolean(breakdown.events_truncated),
    notice: !available
      ? '记分明细暂不可用，仅显示总分'
      : breakdown.events_truncated
        ? '仅显示最近 200 条记分'
        : ''
  }
}
