import { resolveChasingScoreView } from './chasing-score.js'

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
  // 参与者详情接口已按查看者返回「我 / 对手」并带 is_player1；公开详情没有这个字段，靠 perspectiveUserId 判断。
  const isReferee = match.viewer_role === 'referee'
  const isOwnMatchPayload = typeof match.is_player1 === 'boolean' && !isReferee
  const isPerspectivePlayer = Number(perspectiveUserId) > 0 &&
    (Number(match.player1_id) === Number(perspectiveUserId) || Number(match.player2_id) === Number(perspectiveUserId))
  // 未登录、观战者或裁判看固定选手1 / 选手2，不跟随查看者换边。
  const neutralPerspective = isReferee || !(isOwnMatchPayload || isPerspectivePlayer)
  const swapScoreBreakdown = shouldSwapPerspective || (isOwnMatchPayload && match.is_player1 === false)

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
        : (match.player1_avatar || match.my_avatar || ''),
      // 会员外观跟随头像取同一侧，保证头像框与头像始终是同一个人。
      vipLevel: shouldSwapPerspective
        ? pickNumber(match.player2_vip_level, match.opponent_vip_level)
        : pickNumber(match.player1_vip_level, match.my_vip_level),
      rankLevel: shouldSwapPerspective
        ? pickNumber(match.player2_rank_level, match.opponent_rank_level)
        : pickNumber(match.player1_rank_level, match.my_rank_level)
    },
    player2Info: {
      name: shouldSwapPerspective
        ? (match.player1_name || match.my_name || '玩家2')
        : (match.player2_name || match.opponent_name || '玩家2'),
      avatar: shouldSwapPerspective
        ? (match.player1_avatar || match.my_avatar || '')
        : (match.player2_avatar || match.opponent_avatar || ''),
      vipLevel: shouldSwapPerspective
        ? pickNumber(match.player1_vip_level, match.my_vip_level)
        : pickNumber(match.player2_vip_level, match.opponent_vip_level),
      rankLevel: shouldSwapPerspective
        ? pickNumber(match.player1_rank_level, match.my_rank_level)
        : pickNumber(match.player2_rank_level, match.opponent_rank_level)
    },
    roundRecords: Array.isArray(match.rounds) ? normalizeRoundRecords(match.rounds, shouldSwapPerspective) : [],
    scoreBreakdown: pickNumber(match.game_type) === 2
      ? resolveChasingScoreView(match.score_breakdown, { swap: swapScoreBreakdown, neutral: neutralPerspective })
      : null
  }
}
