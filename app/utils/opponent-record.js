const decodeRouteValue = (value = '') => {
  if (!value) return ''

  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export const normalizeOpponentRecordOptions = (options = {}) => ({
  targetUserId: Number(options.target_user_id || 0) || 0,
  targetName: decodeRouteValue(options.target_name || ''),
  targetAvatar: decodeRouteValue(options.target_avatar || '')
})

export const buildOpponentRecordRequestParams = ({
  page = 1,
  pageSize = 20,
  keyword = '',
  targetUserId = 0
} = {}) => {
  const params = {
    page,
    page_size: pageSize,
    keyword
  }

  if (Number(targetUserId) > 0) {
    params.target_user_id = Number(targetUserId)
  }

  return params
}

export const buildOpponentRecordViewModel = ({
  targetName = '',
  targetUserId = 0
} = {}) => {
  const isTargetMode = Number(targetUserId) > 0

  if (!isTargetMode) {
    return {
      isTargetMode: false,
      navigationTitle: '对手记录',
      loginHint: '请登录后查看对手记录',
      searchPlaceholder: '搜索对手',
      emptyText: '暂无对手记录',
      emptyHint: '快去发起一场PK吧！'
    }
  }

  const resolvedTargetName = targetName || '这位球友'

  return {
    isTargetMode: true,
    navigationTitle: '对方战绩',
    loginHint: '请登录后查看对方战绩',
    searchPlaceholder: '搜索 TA 的对手',
    emptyText: '暂无对方战绩',
    emptyHint: `还没看到${resolvedTargetName} 的历史对手记录`
  }
}

export const buildOpponentH2HUrl = ({
  opponent = {},
  targetUserId = 0,
  targetName = '',
  targetAvatar = ''
} = {}) => {
  const query = []

  if (Number(targetUserId) > 0) {
    query.push(`target_user_id=${Number(targetUserId)}`)
    query.push(`target_name=${encodeURIComponent(targetName || '球友')}`)
    query.push(`target_avatar=${encodeURIComponent(targetAvatar || '')}`)
  }

  if (Number(opponent.id) > 0) {
    query.push(`opponent_id=${Number(opponent.id)}`)
  }

  query.push(`opponent_name=${encodeURIComponent(opponent.name || '对手')}`)

  return `/subPages/user/h2hRecord?${query.join('&')}`
}
