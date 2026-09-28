export const SNOOKER_BALLS = [
  { value: 1, key: 'red', label: '红球', color: 'red' },
  { value: 2, key: 'yellow', label: '黄球', color: 'yellow' },
  { value: 3, key: 'green', label: '绿球', color: 'green' },
  { value: 4, key: 'brown', label: '咖啡球', color: 'brown' },
  { value: 5, key: 'blue', label: '蓝球', color: 'blue' },
  { value: 6, key: 'pink', label: '粉球', color: 'pink' },
  { value: 7, key: 'black', label: '黑球', color: 'black' }
]

const BALL_VALUE_BY_KEY = Object.fromEntries(SNOOKER_BALLS.map(item => [item.key, item.value]))

export const isSnookerRulesV2 = (payload = {}) => Number(payload.snooker_rules_version || 0) === 2

export const normalizeSnookerV2State = (payload = {}) => ({
  rulesVersion: Number(payload.snooker_rules_version || 0),
  bestOfFrames: Number(payload.best_of_frames || 0),
  startingActor: Number(payload.starting_actor || 0),
  phase: payload.snooker_phase || '',
  ballOn: payload.snooker_ball_on || '',
  striker: Number(payload.snooker_striker || 0),
  visitNo: Number(payload.snooker_visit_no || 0),
  currentBreak: Number(payload.snooker_current_break || 0),
  redsRemaining: Math.max(0, Number(payload.snooker_reds_remaining || 0)),
  freeBallAvailable: !!payload.snooker_free_ball_available,
  cueBallInHand: !!payload.snooker_cue_ball_in_hand,
  missWarningActive: !!payload.snooker_miss_warning_active,
  respottedBlackPending: !!payload.snooker_respotted_black_pending,
  pendingConcessionActor: Number(payload.snooker_pending_concession_actor || 0),
  pendingConcessionScope: payload.snooker_pending_concession_scope || '',
  frameEndReason: payload.snooker_frame_end_reason || '',
  currentFrameStarted: payload.current_frame_started !== false,
  status: Number(payload.status || 1)
})

export const snookerBallOnValue = (state = {}, selectedColor = 0) => {
  if (state.ballOn === 'red') return 1
  if (state.ballOn === 'color_choice') {
    const value = Number(selectedColor)
    return value >= 2 && value <= 7 ? value : 0
  }
  return BALL_VALUE_BY_KEY[state.ballOn] || 0
}

export const getSnookerV2PotOptions = (state = {}) => {
  if (!state.currentFrameStarted || state.status === 2 || ['ended', 'respotted_black_pending'].includes(state.phase)) return []
  if (state.ballOn === 'red') {
    return state.redsRemaining > 0 ? [SNOOKER_BALLS[0]] : []
  }
  if (state.ballOn === 'color_choice') return SNOOKER_BALLS.slice(1)
  const value = snookerBallOnValue(state)
  return value ? SNOOKER_BALLS.filter(item => item.value === value) : []
}

export const getSnookerV2FreeBallOptions = (state = {}, ballOnValue = 0) => {
  if (!state.freeBallAvailable) return []
  const target = snookerBallOnValue(state, ballOnValue)
  if (!target) return []
  return SNOOKER_BALLS.slice(1).filter(item => item.value !== target)
}

export const getSnookerV2PhaseLabel = (state = {}) => {
  switch (state.phase) {
    case 'reds': return '红球阶段'
    case 'color_after_red': return '红球后的彩球'
    case 'colors': return '顺序清彩'
    case 'respotted_black_pending': return '等待重置黑球'
    case 'respotted_black': return '重置黑球决胜'
    case 'ended': return '本局已结束'
    default: return '等待同步局面'
  }
}

export const getSnookerV2BallOnLabel = (state = {}) => {
  if (state.ballOn === 'color_choice') return '自选彩球'
  return SNOOKER_BALLS.find(item => item.key === state.ballOn)?.label || '等待裁判确认'
}

export const buildSnookerV2PotPayload = ({ state = {}, actor, ballOnValue = 0, pottedReds = 0, freeBallValue = 0, freeBallPotted = false, ballOnPotted = false } = {}) => ({
  actor: Number(actor) || 0,
  outcome: 'pot',
  ball_on_value: snookerBallOnValue(state, ballOnValue),
  potted_reds: Math.max(0, Number(pottedReds) || 0),
  ball_on_potted: !!ballOnPotted,
  free_ball_value: Number(freeBallValue) || 0,
  free_ball_potted: !!freeBallPotted
})

export const buildSnookerV2NoScorePayload = ({ state = {}, actor, ballOnValue = 0, freeBallValue = 0 } = {}) => {
  const payload = {
    actor: Number(actor) || 0,
    outcome: 'no_score',
    ball_on_value: state.ballOn === 'color_choice' ? Number(ballOnValue) || 0 : snookerBallOnValue(state)
  }
  if (Number(freeBallValue) > 0) payload.free_ball_value = Number(freeBallValue)
  return payload
}

export const buildSnookerV2FoulPayload = ({ state = {}, actor, penalty = 4, ballOnValue = 0, redsRemoved = 0, resolution = 'incoming_plays', freeBallAwarded = false, foulAndMiss = false, missSequenceEligible = false, cueBallInHand = false } = {}) => ({
  actor: Number(actor) || 0,
  outcome: 'foul',
  ball_on_value: snookerBallOnValue(state, ballOnValue),
  penalty: Number(penalty) || 0,
  reds_removed: Math.max(0, Number(redsRemoved) || 0),
  foul_resolution: resolution,
  free_ball_awarded: resolution === 'incoming_plays' && !!freeBallAwarded,
  foul_and_miss: !!foulAndMiss,
  miss_sequence_eligible: resolution === 'offender_replays_original' && !!foulAndMiss && !!missSequenceEligible,
  cue_ball_in_hand: resolution !== 'offender_replays_original' && !!cueBallInHand
})

export const getSnookerV2MinimumPenalty = (state = {}, ballOnValue = 0) => Math.max(4, snookerBallOnValue(state, ballOnValue) || 0)

export const buildSnookerCountPages = (minimum, maximum, pageSize = 6) => {
  const min = Math.max(0, Number(minimum) || 0)
  const max = Math.max(min, Number(maximum) || 0)
  const size = Math.min(6, Math.max(1, Number(pageSize) || 6))
  const values = Array.from({ length: max - min + 1 }, (_, index) => min + index)
  const pages = []
  for (let index = 0; index < values.length; index += size) {
    pages.push(values.slice(index, index + size))
  }
  return pages
}
