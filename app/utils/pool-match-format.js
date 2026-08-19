export const POOL_MATCH_FORMAT_LEGACY = 'legacy'
export const POOL_MATCH_FORMAT_FREE = 'free'
export const POOL_MATCH_FORMAT_RACE_TO = 'race_to'
export const POOL_MATCH_MAX_ROUNDS = 129
export const POOL_MATCH_MAX_TARGET_WINS = 65
export const POOL_TARGET_OPTIONS = Array.from({ length: POOL_MATCH_MAX_TARGET_WINS }, (_, index) => index + 1)

export const isPoolMatchFormatGameType = (gameType) => [3, 4].includes(Number(gameType || 0))

export const normalizePoolMatchFormat = (payload = {}) => {
  const rawFormat = payload.match_format ?? payload.matchFormat ?? ''
  if (rawFormat === POOL_MATCH_FORMAT_LEGACY) {
    return { format: POOL_MATCH_FORMAT_LEGACY, targetWins: 0 }
  }
  const format = rawFormat === POOL_MATCH_FORMAT_RACE_TO
    ? POOL_MATCH_FORMAT_RACE_TO
    : POOL_MATCH_FORMAT_FREE
  const target = Number(payload.target_wins ?? payload.targetWins ?? 0) || 0
  return {
    format,
    targetWins: format === POOL_MATCH_FORMAT_RACE_TO
      ? Math.min(Math.max(target, 1), POOL_MATCH_MAX_TARGET_WINS)
      : 0
  }
}

export const getPoolMatchFormatLabel = (format, targetWins = 0) => {
  if (format === POOL_MATCH_FORMAT_LEGACY) return '传统赛制'
  return format === POOL_MATCH_FORMAT_RACE_TO
    ? `抢${Math.min(Math.max(Number(targetWins) || 1, 1), POOL_MATCH_MAX_TARGET_WINS)}局制`
    : '自由局数'
}

export const getPoolMatchFormatHint = (format, targetWins = 0) => {
  if (format === POOL_MATCH_FORMAT_LEGACY) return '按旧版手动结束规则进行'
  if (format !== POOL_MATCH_FORMAT_RACE_TO) return `不预设胜局，最多进行${POOL_MATCH_MAX_ROUNDS}局，允许平局`
  const target = Math.min(Math.max(Number(targetWins) || 1, 1), POOL_MATCH_MAX_TARGET_WINS)
  return `先达到${target}胜，最多可能进行${target * 2 - 1}局`
}

export const canFinishFreePoolMatch = ({ format, myScore, opponentScore } = {}) => {
  return format === POOL_MATCH_FORMAT_FREE && Number(myScore || 0) + Number(opponentScore || 0) >= 1
}

export const canContinuePoolMatch = ({ format, targetWins, myScore, opponentScore } = {}) => {
  const myWins = Number(myScore) || 0
  const opponentWins = Number(opponentScore) || 0
  if (format === POOL_MATCH_FORMAT_FREE) return myWins + opponentWins < POOL_MATCH_MAX_ROUNDS
  if (format === POOL_MATCH_FORMAT_RACE_TO) {
    const target = Math.min(Math.max(Number(targetWins) || 1, 1), POOL_MATCH_MAX_TARGET_WINS)
    return myWins < target && opponentWins < target
  }
  return true
}
