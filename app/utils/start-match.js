export const validateStartMatchPayload = (payload = {}) => {
  const opponentId = Number(payload.opponent_id || payload.opponentId || 0)
  if (!Number.isFinite(opponentId) || opponentId <= 0) {
    return '请选择有效的平台对手'
  }
  const mode = payload.match_mode || payload.matchMode || ''
  const visibility = payload.visibility || ''
  if (mode && !['practice', 'ranked'].includes(mode)) return '请选择有效的对局模式'
  if (visibility && !['private', 'public'].includes(visibility)) return '请选择有效的公开范围'
  if (mode === 'ranked' && visibility === 'private') return '排位赛必须公开展示'
  if (Number(payload.game_type || payload.gameType || 0) === 1) {
    const rulesVersion = Number(payload.snooker_rules_version || payload.snookerRulesVersion || 2)
    if (![1, 2].includes(rulesVersion)) return '不支持的斯诺克规则版本'
    if (rulesVersion === 2) {
      const format = payload.snooker_format || payload.snookerFormat || ''
      const targetWins = Number(payload.snooker_target_wins ?? payload.snookerTargetWins ?? 0)
      if (format === 'free') {
        if (targetWins !== 0) return '自由局数不能设置目标胜局'
      } else if (format === 'race_to') {
        if (!Number.isInteger(targetWins) || targetWins < 1 || targetWins > 25) return '抢N局目标必须在1至25之间'
      } else {
        const bestOfFrames = Number(payload.best_of_frames || payload.bestOfFrames || 0)
        const startingActor = Number(payload.starting_actor || payload.startingActor || 0)
        if (!Number.isInteger(bestOfFrames) || bestOfFrames <= 0 || bestOfFrames % 2 === 0) return '斯诺克总局数必须为正奇数'
        if (![1, 2].includes(startingActor)) return '请选择首局开球方'
      }
    }
  }
  return ''
}

export const normalizeStartMatchOptions = ({ match_mode, matchMode, visibility } = {}) => {
  const mode = match_mode || matchMode || 'ranked'
  const normalizedMode = mode === 'practice' || mode === 'ranked' ? mode : 'ranked'
  const normalizedVisibility = visibility || (normalizedMode === 'practice' ? 'private' : 'public')
  return {
    match_mode: normalizedMode,
    visibility: normalizedMode === 'ranked' ? 'public' : normalizedVisibility === 'public' ? 'public' : 'private'
  }
}

export const normalizePendingMatchContext = (storageKey = '', raw = '') => {
  const type = storageKey === 'pending_match_challenge'
    ? 'challenge'
    : storageKey === 'pending_match_rematch'
      ? 'rematch'
      : ''
  if (!type || !raw) return { valid: false, context: null, message: '开局信息已失效' }

  let parsed
  try {
    parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
  } catch {
    return { valid: false, context: null, message: type === 'challenge' ? '邀约开局信息已失效' : '重赛信息已失效' }
  }

  const context = {
    ...parsed,
    context_type: type,
    challenge_id: Number(parsed?.challenge_id || 0),
    opponent_id: Number(parsed?.opponent_id || 0),
    game_type: Number(parsed?.game_type || 0),
    match_mode: parsed?.match_mode === 'ranked' ? 'ranked' : 'practice',
    visibility: parsed?.visibility === 'public' ? 'public' : 'private'
  }
  if (context.game_type === 1) {
    context.snooker_rules_version = Number(parsed?.snooker_rules_version || 2)
    context.snooker_format = parsed?.snooker_format === 'race_to' ? 'race_to' : 'free'
    context.snooker_target_wins = context.snooker_format === 'race_to'
      ? Math.min(Math.max(Number(parsed?.snooker_target_wins) || 1, 1), 25)
      : 0
  }
  const valid = context.opponent_id > 0 && context.game_type > 0 && (type !== 'challenge' || context.challenge_id > 0)
  return {
    valid,
    context: valid ? context : null,
    message: valid ? '' : type === 'challenge' ? '邀约开局信息不完整' : '重赛信息不完整'
  }
}

export const validateScannedOpponentForContext = (context = {}, scannedOpponent = {}) => {
  if (context.context_type !== 'challenge') return ''
  const expectedId = Number(context.opponent_id || 0)
  const scannedId = Number(scannedOpponent.id || scannedOpponent.user_id || scannedOpponent.opponent_id || 0)
  if (expectedId > 0 && scannedId !== expectedId) return '请扫描邀约中的指定对手'
  return ''
}

export const buildStartMatchPayload = ({
  gameType,
  opponent = {},
  matchMode = 'ranked',
  visibility,
  challengeId = 0
} = {}) => {
  const options = normalizeStartMatchOptions({ matchMode, visibility })
  const payload = {
    game_type: Number(gameType) || 0,
    opponent_id: Number(opponent.id || opponent.user_id || opponent.opponent_id || 0),
    opponent_name: opponent.nickname || opponent.name || opponent.opponent_name || '对手',
    opponent_avatar: opponent.avatar || opponent.opponent_avatar || '',
    ...options
  }
  if (Number(challengeId) > 0) payload.challenge_id = Number(challengeId)
  if (payload.game_type === 1) {
    payload.snooker_rules_version = 2
    payload.snooker_format = 'free'
    payload.snooker_target_wins = 0
  }
  return payload
}
