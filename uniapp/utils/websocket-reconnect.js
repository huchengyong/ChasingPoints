const DEFAULT_BASE_DELAY_MS = 3000
const DEFAULT_MAX_DELAY_MS = 60000
const DEFAULT_JITTER_RATIO = 0.2

export function classifyWebSocketFailure(error = {}) {
  const statusCode = Number(error?.statusCode || error?.status || 0)
  const category = String(error?.category || '').toLowerCase()
  const reason = String(
    error?.responseData?.reason || error?.reason || error?.code || ''
  ).toUpperCase()

  if (category === 'session' || statusCode === 401 || reason === 'SESSION_INVALID') {
    return 'session-invalid'
  }
  if (category === 'forbidden' || statusCode === 403 || reason === 'FORBIDDEN') {
    return 'permission-denied'
  }
  return 'recoverable'
}

export function calculateReconnectDelay({
  attempt,
  baseDelayMs = DEFAULT_BASE_DELAY_MS,
  maxDelayMs = DEFAULT_MAX_DELAY_MS,
  jitterRatio = DEFAULT_JITTER_RATIO,
  random = Math.random
} = {}) {
  const safeAttempt = Math.max(1, Number(attempt) || 1)
  const safeBase = Math.max(0, Number(baseDelayMs) || DEFAULT_BASE_DELAY_MS)
  const safeMax = Math.max(safeBase, Number(maxDelayMs) || DEFAULT_MAX_DELAY_MS)
  const capped = Math.min(safeMax, safeBase * (2 ** (safeAttempt - 1)))
  const safeJitter = Math.max(0, Math.min(1, Number(jitterRatio) || 0))
  const jitter = capped * safeJitter * ((Number(random()) || 0) * 2 - 1)
  return Math.max(0, Math.min(safeMax, Math.round(capped + jitter)))
}

export function shouldReconnectWebSocket({
  manualDisconnect = false,
  foreground = true,
  online = true,
  sessionInvalid = false,
  permissionDenied = false,
  targetActive = true,
  error = null
} = {}) {
  const failure = classifyWebSocketFailure(error)
  if (manualDisconnect) return { allow: false, reason: 'manual-disconnect' }
  if (!foreground) return { allow: false, reason: 'background' }
  if (!online) return { allow: false, reason: 'offline' }
  if (sessionInvalid || failure === 'session-invalid') return { allow: false, reason: 'session-invalid' }
  if (permissionDenied || failure === 'permission-denied') return { allow: false, reason: 'permission-denied' }
  if (!targetActive) return { allow: false, reason: 'target-inactive' }
  return { allow: true, reason: 'recoverable' }
}

export function nextConnectionGeneration(currentGeneration = 0) {
  return Math.max(0, Number(currentGeneration) || 0) + 1
}

export function isCurrentConnectionGeneration(currentGeneration, callbackGeneration) {
  return Number(currentGeneration) === Number(callbackGeneration)
}
