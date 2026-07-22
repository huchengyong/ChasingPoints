/**
 * 裁判角色视图解析工具
 * 统一解析参赛/执裁记录卡片、裁判身份卡、执裁完成回执以及 completion_source 的安全文案
 */

/**
 * 解析 completion_source 为安全展示文案
 * @param {string} source - completion_source 值
 * @returns {string} 用户可读的完成方式描述
 */
export const resolveCompletionSourceLabel = (source) => {
  switch (source) {
    case 'referee':
      return '裁判结算'
    case 'player_direct':
      return '选手直接完成'
    case 'player_confirmed':
      return '双方确认完成'
    case 'unknown':
    default:
      return '未知'
  }
}

/**
 * 判断是否可以展示具体的完成方式（非 unknown）
 */
export const hasReliableCompletionAttribution = (source) => {
  return source && source !== 'unknown'
}

/**
 * 解析裁判身份卡信息
 * @param {Object} options
 * @param {boolean} options.refereeBound
 * @param {number} options.refereeUserId
 * @param {string} options.refereeName
 * @param {string} options.refereeAvatar
 * @param {string} options.refereeJoinedAt
 * @param {number} options.refereeDurationSeconds
 * @param {number} options.completedByUserId
 * @param {string} options.completionSource
 * @returns {Object} 格式化后的裁判身份信息
 */
export const resolveRefereeIdentityCard = ({
  refereeBound = false,
  refereeUserId = 0,
  refereeName = '',
  refereeAvatar = '',
  refereeJoinedAt = '',
  refereeDurationSeconds = 0,
  completedByUserId = 0,
  completionSource = '',
  status = 0
} = {}) => {
  if (!refereeBound || !refereeUserId) {
    return { hasReferee: false }
  }

  const isCompleted = status === 2
  const hasReliableAttribution = isCompleted && hasReliableCompletionAttribution(completionSource)

  return {
    hasReferee: true,
    refereeUserId,
    refereeName: refereeName || '本场裁判',
    refereeAvatar,
    refereeJoinedAt,
    refereeDurationSeconds,
    refereeDurationText: formatDuration(refereeDurationSeconds),
    completedByUserId,
    completionSource,
    completionLabel: hasReliableAttribution ? resolveCompletionSourceLabel(completionSource) : '',
    hasReliableAttribution,
    // 中立称谓 — 不使用"官方裁判"或"认证裁判"
    neutralLabel: '本场裁判'
  }
}

/**
 * 格式化秒数为可读时长（如 "12分34秒" 或 "2小时12分"）
 */
export const formatDuration = (seconds) => {
  if (!seconds || seconds <= 0) return ''
  if (seconds < 60) return `${seconds}秒`
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (mins < 60) {
    return secs > 0 ? `${mins}分${secs}秒` : `${mins}分钟`
  }
  const hours = Math.floor(mins / 60)
  const remainMins = mins % 60
  return remainMins > 0 ? `${hours}小时${remainMins}分` : `${hours}小时`
}

/**
 * 裁判结果视图配置
 * 集中定义标题、可见模块和底部操作
 * @param {Object} options
 * @param {string} options.viewerRole - player1/player2/referee
 * @param {number} options.status - 1=进行中 2=已完成 3=已取消
 * @returns {Object} 视图配置
 */
export const resolveRefereeResultViewConfig = ({
  viewerRole = 'player1',
  status = 1
} = {}) => {
  const isReferee = viewerRole === 'referee'
  const isCompleted = status === 2

  if (isReferee && isCompleted) {
    return {
      pageTitle: '执裁详情',
      showWinLossBanner: false,
      showRankChange: false,
      showRematchButton: false,
      showShareResultButton: false,
      showH2HActions: false,
      showRefereeCard: true,
      roleHint: '裁判视角，展示双方选手和执裁归因'
    }
  }

  if (isReferee) {
    return {
      pageTitle: '执裁完成',
      showWinLossBanner: false,
      showRankChange: false,
      showRematchButton: false,
      showShareResultButton: false,
      showH2HActions: false,
      showRefereeCard: true,
      roleHint: '已完赛，可查看执裁详情'
    }
  }

  // 选手视角
  return {
    pageTitle: '战报',
    showWinLossBanner: true,
    showRankChange: true,
    showRematchButton: true,
    showShareResultButton: true,
    showH2HActions: true,
    showRefereeCard: true,
    roleHint: ''
  }
}

/**
 * 裁判历史卡片数据解析
 * 非胜负颜色、中立语义
 */
export const resolveRefereeHistoryCard = (item) => {
  const isCancelled = item.status === 3
  return {
    id: item.id,
    player1Name: item.player1_name || '玩家1',
    player1Avatar: item.player1_avatar || '',
    player2Name: item.player2_name || '玩家2',
    player2Avatar: item.player2_avatar || '',
    player1Score: item.player1_score,
    player2Score: item.player2_score,
    gameTypeName: item.game_type_name || '',
    matchMode: item.match_mode || '',
    refereeJoinedAt: item.referee_joined_at || '',
    endTime: item.end_time || '',
    refereeDurationSeconds: item.referee_duration_seconds || 0,
    refereeDurationText: formatDuration(item.referee_duration_seconds),
    completedByUserId: item.completed_by_user_id || 0,
    completionSource: item.completion_source || '',
    completionLabel: isCancelled ? '' : resolveCompletionSourceLabel(item.completion_source),
    statusText: isCancelled ? '已取消' : '已完成',
    isCancelled,
    // 不使用胜负颜色
    isNeutral: true
  }
}
