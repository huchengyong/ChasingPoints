export const SNOOKER_FORMAT_FREE = 'free'
export const SNOOKER_FORMAT_RACE_TO = 'race_to'
export const SNOOKER_MAX_FRAMES = 49
export const SNOOKER_TARGET_OPTIONS = Array.from({ length: 25 }, (_, index) => index + 1)

export const normalizeSnookerMatchFormat = (payload = {}) => {
  const format = payload.snooker_format === SNOOKER_FORMAT_RACE_TO
    ? SNOOKER_FORMAT_RACE_TO
    : SNOOKER_FORMAT_FREE
  const target = Number(payload.snooker_target_wins) || 0
  return {
    format,
    targetWins: format === SNOOKER_FORMAT_RACE_TO
      ? Math.min(Math.max(target, 1), 25)
      : 0
  }
}

export const getSnookerFormatLabel = (format, targetWins = 0) => {
  return format === SNOOKER_FORMAT_RACE_TO
    ? `抢${Math.min(Math.max(Number(targetWins) || 1, 1), 25)}局制`
    : '自由局数'
}

export const getSnookerFormatHint = (format, targetWins = 0) => {
  if (format !== SNOOKER_FORMAT_RACE_TO) return `不预设胜局，最多进行${SNOOKER_MAX_FRAMES}局`
  const target = Math.min(Math.max(Number(targetWins) || 1, 1), 25)
  return `先达到${target}胜，最多可能进行${target * 2 - 1}局`
}

export const canFinishFreeSnookerMatch = ({ format, currentFrameStarted, myScore, opponentScore } = {}) => {
  return format === SNOOKER_FORMAT_FREE && !currentFrameStarted && Number(myScore || 0) + Number(opponentScore || 0) >= 1
}

export const canStartNextSnookerFrame = ({ format, targetWins, myScore, opponentScore } = {}) => {
  const myWins = Number(myScore) || 0
  const opponentWins = Number(opponentScore) || 0
  if (format === SNOOKER_FORMAT_FREE) return myWins + opponentWins < SNOOKER_MAX_FRAMES
  const target = Math.min(Math.max(Number(targetWins) || 1, 1), 25)
  return myWins < target && opponentWins < target
}
