const pickValue = (...values) => values.find((value) => value !== undefined && value !== null && value !== '')

const pickNumber = (...values) => {
  const value = pickValue(...values)
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const normalizeRoundRecords = (rounds = [], shouldSwapPerspective = false) => rounds.map((round = {}) => {
  const winner = pickNumber(round.winner)
  const perspectiveWinner = shouldSwapPerspective && winner > 0 ? (winner === 1 ? 2 : 1) : winner

  return {
    ...round,
    player1_score: shouldSwapPerspective
      ? pickNumber(round.player2_score, round.opponent_score)
      : pickNumber(round.player1_score, round.my_score),
    player2_score: shouldSwapPerspective
      ? pickNumber(round.player1_score, round.my_score)
      : pickNumber(round.player2_score, round.opponent_score),
    winner: perspectiveWinner,
    result: perspectiveWinner === 1 ? 'win' : 'loss',
    resultText: perspectiveWinner === 1 ? '胜' : '负'
  }
})

export const shouldUsePublicMatchDetail = ({ mode = '', source = '' } = {}) => (
  mode === 'spectate' || source === 'history'
)

export const shouldShowSpectateBadge = ({ mode = '' } = {}) => mode === 'spectate'

export const normalizeMatchDetailPayload = (match = {}, { perspectiveUserId = 0 } = {}) => {
  const shouldSwapPerspective = Number(perspectiveUserId) > 0 &&
    Number(match.player2_id) === Number(perspectiveUserId) &&
    Number(match.player1_id) !== Number(perspectiveUserId)

  return {
    matchData: {
      player1_score: shouldSwapPerspective
        ? pickNumber(match.player2_score, match.opponent_score)
        : pickNumber(match.player1_score, match.my_score),
      player2_score: shouldSwapPerspective
        ? pickNumber(match.player1_score, match.my_score)
        : pickNumber(match.player2_score, match.opponent_score),
      game_type: pickNumber(match.game_type) || 3,
      status: pickNumber(match.status) || 1,
      duration_seconds: pickNumber(match.duration_seconds),
      current_round: pickNumber(match.current_round) || 1,
      total_rounds: pickNumber(match.total_rounds),
      viewer_role: match.viewer_role || '',
      referee_bound: !!match.referee_bound,
      referee_user_id: pickNumber(match.referee_user_id),
      referee_name: match.referee_name || '',
      referee_avatar: match.referee_avatar || '',
      referee_joined_at: match.referee_joined_at || '',
      referee_duration_seconds: pickNumber(match.referee_duration_seconds),
      completed_by_user_id: pickNumber(match.completed_by_user_id),
      completion_source: match.completion_source || 'unknown'
    },
    player1Info: {
      name: shouldSwapPerspective
        ? (match.player2_name || match.opponent_name || '玩家1')
        : (match.player1_name || match.my_name || '玩家1'),
      avatar: shouldSwapPerspective
        ? (match.player2_avatar || match.opponent_avatar || '')
        : (match.player1_avatar || match.my_avatar || '')
    },
    player2Info: {
      name: shouldSwapPerspective
        ? (match.player1_name || match.my_name || '玩家2')
        : (match.player2_name || match.opponent_name || '玩家2'),
      avatar: shouldSwapPerspective
        ? (match.player1_avatar || match.my_avatar || '')
        : (match.player2_avatar || match.opponent_avatar || '')
    },
    roundRecords: Array.isArray(match.rounds) ? normalizeRoundRecords(match.rounds, shouldSwapPerspective) : []
  }
}
