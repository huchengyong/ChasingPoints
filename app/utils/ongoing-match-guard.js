const PLAYING_ROUTE = '/subPages/match/playing'

const normalizeMatchPayload = (match = {}, fallback = {}) => {
  const normalizedMatchId = Number(match.match_id || match.id || fallback.match_id || fallback.id || 0)

  return {
    match_id: normalizedMatchId,
    game_type: Number(match.game_type || fallback.game_type || 3),
    opponent_name: match.opponent_name || fallback.opponent_name || '对手',
    opponent_avatar: match.opponent_avatar || fallback.opponent_avatar || '',
    opponent_id: Number(match.opponent_id || fallback.opponent_id || 0),
    match_mode: match.match_mode || fallback.match_mode || 'ranked',
    visibility: match.visibility || fallback.visibility || 'public',
    challenge_id: Number(match.challenge_id || fallback.challenge_id || 0)
  }
}

export const isPlayingMatchRoute = (route = '') => route.includes(PLAYING_ROUTE)

export const resolveMatchScanAction = (scanResult = '') => {
  let payload = null
  try {
    payload = JSON.parse(scanResult)
  } catch {
    return {
      type: 'error',
      message: '无效的二维码'
    }
  }

  if (payload?.type === 'match_referee' && payload?.match_id && payload?.join_token) {
    return {
      type: 'join_referee',
      refereeJoin: {
        match_id: Number(payload.match_id),
        join_token: payload.join_token
      }
    }
  }

  if (payload?.user_id) {
    return {
      type: 'start_match',
      opponent: payload
    }
  }

  return {
    type: 'error',
    message: '无效的二维码'
  }
}

export const shouldPromptOngoingMatch = ({
  isLoggedIn = false,
  currentRoute = '',
  currentMatch = null,
  hasPromptedInForeground = false
} = {}) => {
  if (!isLoggedIn || !currentMatch) return false
  if (hasPromptedInForeground) return false
  if (!currentRoute) return false

  return !isPlayingMatchRoute(currentRoute)
}

export const buildPlayingRoute = (match = {}) => {
  const normalized = normalizeMatchPayload(match)

  return `${PLAYING_ROUTE}?match_id=${normalized.match_id}&game_type=${normalized.game_type}&opponent_name=${encodeURIComponent(normalized.opponent_name)}&opponent_avatar=${encodeURIComponent(normalized.opponent_avatar)}`
}

export const resolveStartMatchGuardAction = ({
  response = null,
  selectedGameType = 3,
  scannedOpponent = {}
} = {}) => {
  if (!response?.success) {
    return {
      type: 'error',
      message: response?.message || '创建对局失败'
    }
  }

  const scannedFallback = {
    game_type: selectedGameType,
    opponent_name: scannedOpponent.nickname || scannedOpponent.opponent_name || '对手',
    opponent_avatar: scannedOpponent.avatar || scannedOpponent.opponent_avatar || '',
    opponent_id: Number(scannedOpponent.user_id || scannedOpponent.opponent_id || 0)
  }

  if (response.action === 'created') {
    return {
      type: 'navigate',
      match: normalizeMatchPayload(response.match || { match_id: response.match_id }, scannedFallback)
    }
  }

  if (response.action === 'resume_existing') {
    return {
      type: 'resume',
      match: normalizeMatchPayload(response.ongoing_match || { match_id: response.match_id }, {
        ...scannedFallback,
        match_id: response.match_id
      })
    }
  }

  if (response.action === 'blocked' && response.block_reason === 'self_ongoing') {
    return {
      type: 'prompt_self_ongoing',
      message: response.message || '你还有未结束的对局，请先处理',
      match: normalizeMatchPayload(response.ongoing_match || { match_id: response.match_id }, {
        ...scannedFallback,
        match_id: response.match_id
      })
    }
  }

  if (response.action === 'blocked' && response.block_reason === 'opponent_ongoing') {
    return {
      type: 'toast_opponent_ongoing',
      message: response.message || '对手还有未结束的对局，暂时无法开始新的 PK'
    }
  }

  return {
    type: 'error',
    message: response.message || '创建对局失败'
  }
}
