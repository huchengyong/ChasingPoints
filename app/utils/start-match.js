export const validateStartMatchPayload = (payload = {}) => {
  const opponentId = Number(payload.opponent_id || payload.opponentId || 0)
  if (!Number.isFinite(opponentId) || opponentId <= 0) {
    return '请选择有效的平台对手'
  }
  return ''
}
