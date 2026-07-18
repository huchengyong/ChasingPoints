export const resolveMatchHomePrimaryAction = ({ currentMatch = null } = {}) => {
  if (currentMatch?.finish_state === 'pending_confirmation') {
    return { type: 'confirm', label: '处理结束确认' }
  }
  if (Number(currentMatch?.status) === 1) {
    return { type: 'continue', label: '继续对局' }
  }
  return { type: 'start', label: '发起对局' }
}

export const resolveFinishAction = (capabilities = {}) => {
  if (capabilities.can_confirm_finish) {
    return { type: 'confirm', label: '确认结束' }
  }
  if (capabilities.can_dispute_finish) {
    return { type: 'dispute', label: '提出异议' }
  }
  if (capabilities.can_withdraw_finish) {
    return { type: 'withdraw', label: '撤回请求' }
  }
  if (capabilities.can_request_finish) {
    return { type: 'request', label: '发起结束确认' }
  }
  if (capabilities.can_finish) {
    return { type: 'finish', label: '结束本场对局' }
  }
  return { type: 'readonly', label: '等待对手处理' }
}

export const resolveViewerRoleCopy = (viewerRole = '') => {
  if (viewerRole === 'referee') {
    return { left: '选手1', right: '选手2', role: '裁判视角' }
  }
  return { left: '我方', right: '对手', role: '选手视角' }
}

export const buildRematchContext = (match = {}) => {
  const context = {
    opponent_id: Number(match.opponent_id || match.player2_id || 0),
    opponent_name: match.opponent_name || match.player2_name || '对手',
    opponent_avatar: match.opponent_avatar || match.player2_avatar || '',
    game_type: Number(match.game_type) || 3,
    match_mode: match.match_mode === 'ranked' ? 'ranked' : 'practice'
  }
  if (match.visibility) context.visibility = match.visibility === 'public' ? 'public' : 'private'
  return context
}

export const buildFinishedMatchDetailRoute = (match = {}, userId = 0) => {
  const matchId = Number(match.match_id || match.id || 0)
  if (Number(match.status) !== 2 || matchId <= 0) return ''
  const currentUserId = Number(userId) || 0
  const player1Id = Number(match.player1_id || match.user_id || 0)
  const player2Id = Number(match.player2_id || match.opponent_id || 0)
  if (currentUserId > 0 && (currentUserId === player1Id || currentUserId === player2Id)) {
    return `/subPages/match/matchDetail?match_id=${matchId}`
  }
  return `/subPages/match/matchDetail?match_id=${matchId}&mode=spectate`
}
