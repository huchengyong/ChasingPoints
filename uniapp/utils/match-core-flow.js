import { formatDuration, resolveStatusCardContent } from './user-homepage.js'

const MATCH_MODE_TEXT = { ranked: '排位', practice: '练习' }

export const resolveMatchHomePrimaryAction = ({ currentMatch = null } = {}) => {
  if (currentMatch?.finish_state === 'pending_confirmation') {
    return { type: 'confirm', label: '处理结束确认' }
  }
  if (Number(currentMatch?.status) === 1) {
    return { type: 'continue', label: '继续对局' }
  }
  return { type: 'start', label: '发起对局' }
}

// 「我的对局」卡：双方按 player1/player2 固定顺序展示，裁判视角不标记「我」。
export const buildOngoingMatchCard = (match, currentUser = {}) => {
  const action = resolveMatchHomePrimaryAction({ currentMatch: match })
  if (!match || action.type === 'start') return null

  const isReferee = match.viewer_role === 'referee'
  const isPending = action.type === 'confirm'
  const viewerIndex = match.viewer_role === 'player2' ? 1 : 0
  const { players } = resolveStatusCardContent({
    mode: isReferee ? 'ongoing-referee' : 'ongoing-player',
    currentMatch: match,
    currentUser
  })

  return {
    id: Number(match.id) || 0,
    isReferee,
    isPending,
    statusText: isPending ? '待确认结束' : isReferee ? '执裁中' : '进行中',
    metaText: [match.game_type_name || '台球', MATCH_MODE_TEXT[match.match_mode]].filter(Boolean).join(' · '),
    durationText: formatDuration(match.duration_seconds),
    players: players.map((player, index) => ({ ...player, isMe: !isReferee && index === viewerIndex })),
    actionLabel: isPending ? '处理结束确认' : isReferee ? '进入裁判记分' : '继续对局'
  }
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
  if (context.game_type === 1) {
    context.best_of_frames = Number(match.best_of_frames) || 3
    context.starting_actor = Number(match.starting_actor) === 2 ? 2 : 1
  }
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
