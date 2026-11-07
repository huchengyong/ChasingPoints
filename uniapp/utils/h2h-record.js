const decodeRouteValue = (value = '') => {
  if (!value) return ''

  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export const buildH2HHistoryParams = ({
  targetUserId = 0,
  opponentId = 0,
  fallbackOpponentId = 0,
  opponentName = '',
  page = 1,
  pageSize = 20,
  result = 0,
  startDate = '',
  endDate = ''
} = {}) => {
  const resolvedOpponentId = Number(opponentId) || Number(fallbackOpponentId) || 0
  const params = {
    page,
    page_size: pageSize
  }

  if (Number(targetUserId) > 0) {
    params.target_user_id = Number(targetUserId)
  }

  if (resolvedOpponentId > 0) {
    params.opponent_id = resolvedOpponentId
  } else if (opponentName) {
    params.opponent_name = opponentName
  }

  if (result > 0) {
    params.result = result
  }
  if (startDate) {
    params.start_date = startDate
  }
  if (endDate) {
    params.end_date = endDate
  }

  return params
}

export const buildH2HMonthDateRange = (monthKey = '') => {
  const matched = String(monthKey).match(/^(\d{4})-(\d{2})$/)
  if (!matched) return null

  const year = Number(matched[1])
  const month = Number(matched[2])
  if (month < 1 || month > 12) return null

  const lastDay = new Date(year, month, 0).getDate()
  return {
    startDate: `${year}-${String(month).padStart(2, '0')}-01`,
    endDate: `${year}-${String(month).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`
  }
}

export const buildH2HCurrentMonthKey = (date = new Date()) => {
  const resolved = date instanceof Date ? date : new Date(date)
  if (Number.isNaN(resolved.getTime())) return buildH2HCurrentMonthKey(new Date())

  return `${resolved.getFullYear()}-${String(resolved.getMonth() + 1).padStart(2, '0')}`
}

export const shiftH2HMonthKey = (monthKey = '', offset = 0) => {
  const matched = String(monthKey).match(/^(\d{4})-(\d{2})$/)
  if (!matched) return buildH2HCurrentMonthKey()

  const date = new Date(Number(matched[1]), Number(matched[2]) - 1 + Number(offset || 0), 1)
  return buildH2HCurrentMonthKey(date)
}

export const normalizeH2HRecordOptions = (options = {}) => ({
  targetUserId: Number(options.target_user_id || 0) || 0,
  targetName: decodeRouteValue(options.target_name || ''),
  targetAvatar: decodeRouteValue(options.target_avatar || ''),
  opponentId: Number(options.opponent_id || 0) || 0,
  opponentName: decodeRouteValue(options.opponent_name || '')
})

export const buildH2HViewModel = ({
  targetUserId = 0,
  targetName = '',
  opponentId = 0,
  currentUserId = 0,
  opponentName = ''
} = {}) => {
  const isTargetMode = Number(targetUserId) > 0
  const isOpponentMe = isTargetMode && Number(currentUserId) > 0 && Number(opponentId) === Number(currentUserId)
  const resolvedOpponentName = opponentName || '对手'

  if (!isTargetMode) {
    return {
      isTargetMode: false,
      navigationTitle: `交锋记录 - ${resolvedOpponentName}`,
      subjectName: '你',
      winRateLabel: `你对${resolvedOpponentName}的胜率是`
    }
  }

  const resolvedTargetName = targetName || '这位球友'
  return {
    isTargetMode: true,
    navigationTitle: `${resolvedTargetName}的交锋记录`,
    subjectName: resolvedTargetName,
    winRateLabel: `${resolvedTargetName} 对${resolvedOpponentName}的胜率是`,
    ...(isOpponentMe ? { isOpponentMe: true } : {})
  }
}

export const buildH2HLoadFailureAction = ({
  targetUserId = 0,
  error
} = {}) => {
  const message = error?.message || ''

  if (Number(targetUserId) > 0 && message === '仅可查看好友的对方战绩') {
    return {
      shouldNavigateBack: true,
      toastMessage: message
    }
  }

  return null
}

export const shouldApplyH2HHistoryResponse = ({
  requestId = 0,
  latestRequestId = 0
} = {}) => requestId === latestRequestId
