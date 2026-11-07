export const validateStartMatchPayload = (payload = {}) => {
  const gameType = Number(payload.game_type || payload.gameType || 0)
  const inviteToken = String(payload.invite_token || payload.inviteToken || '').trim()
  const opponentId = Number(payload.opponent_id || payload.opponentId || 0)
  if (!inviteToken && (!Number.isFinite(opponentId) || opponentId <= 0)) {
    return '请选择有效的平台对手'
  }
  const mode = payload.match_mode || payload.matchMode || ''
  const visibility = payload.visibility || ''
  if (mode && !['practice', 'ranked'].includes(mode)) return '请选择有效的对局模式'
  if (visibility && !['private', 'public'].includes(visibility)) return '请选择有效的公开范围'
  if (mode === 'ranked' && visibility === 'private') return '排位赛必须公开展示'
  if (gameType === 1) {
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
  if ([3, 4].includes(gameType)) {
    const format = payload.match_format || payload.matchFormat || ''
    const targetWins = Number(payload.target_wins ?? payload.targetWins ?? 0)
    if (!format) {
      if (targetWins !== 0) return '请选择有效的赛制'
    } else if (format === 'free') {
      if (targetWins !== 0) return '自由局数不能设置目标胜局'
    } else if (format === 'race_to') {
      if (!Number.isInteger(targetWins) || targetWins < 1 || targetWins > 65) return '抢N局目标必须在1至65之间'
    } else if (format !== 'legacy') {
      return '请选择有效的赛制'
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

// 扫码确认页的对局类型选项：练习赛可私密，排位赛必须公开。
export const START_MATCH_MODE_OPTIONS = [
  { label: '练习赛（默认私密，不影响竞技权益）', matchMode: 'practice', visibility: 'private' },
  { label: '练习赛（公开展示，不影响竞技权益）', matchMode: 'practice', visibility: 'public' },
  { label: '排位赛（公开展示，需结束确认）', matchMode: 'ranked', visibility: 'public' }
]

export const buildStartMatchPayload = ({
  gameType,
  opponent = {},
  matchMode = 'ranked',
  visibility,
  inviteToken = ''
} = {}) => {
  const options = normalizeStartMatchOptions({ matchMode, visibility })
  const normalizedInviteToken = String(inviteToken || '').trim()
  const payload = {
    game_type: Number(gameType) || 0,
    ...options
  }
  if (normalizedInviteToken) {
    payload.invite_token = normalizedInviteToken
  } else {
    payload.opponent_id = Number(opponent.id || opponent.user_id || opponent.opponent_id || 0)
    payload.opponent_name = opponent.nickname || opponent.name || opponent.opponent_name || '对手'
    payload.opponent_avatar = opponent.avatar || opponent.opponent_avatar || ''
  }
  if (payload.game_type === 1) {
    payload.snooker_rules_version = 2
    payload.snooker_format = 'free'
    payload.snooker_target_wins = 0
  }
  if ([3, 4].includes(payload.game_type)) {
    payload.match_format = 'free'
    payload.target_wins = 0
  }
  return payload
}
