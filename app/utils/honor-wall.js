const SUPPORTED_GAME_TYPES = new Set([1, 2, 3, 4])

const honorSourceLabels = {
  achievement: '生涯成就',
  season: '赛季荣誉',
  tournament: '赛事荣誉'
}

const challengeIconPaths = {
  season_matches_20: '/static/images/achievements/season_matches_20.png',
  season_wins_10: '/static/images/achievements/season_wins_10.png',
  season_tournament_finish_1: '/static/images/achievements/season_tournament_finish_1.png'
}

const parseData = (data) => {
  if (!data) return {}
  if (typeof data === 'object') return data
  try {
    return JSON.parse(data)
  } catch {
    return {}
  }
}

export const normalizeHonorWallGameType = (value) => {
  const gameType = Number(value || 0)
  return SUPPORTED_GAME_TYPES.has(gameType) ? gameType : 3
}

export const normalizeHonorWallOptions = (options = {}) => ({
  userId: Number(options.user_id || 0),
  gameType: normalizeHonorWallGameType(options.game_type),
  activeTab: ['career', 'season', 'history'].includes(options.tab) ? options.tab : 'career',
  historySeasonId: Number(options.history_season_id || 0)
})

export const buildHonorWallUrl = ({ userId = 0, gameType = 3, tab = 'career', historySeasonId = 0 } = {}) => {
  const params = [`game_type=${normalizeHonorWallGameType(gameType)}`, `tab=${encodeURIComponent(tab)}`]
  if (Number(userId) > 0) params.push(`user_id=${Number(userId)}`)
  if (Number(historySeasonId) > 0) params.push(`history_season_id=${Number(historySeasonId)}`)
  return `/subPages/achievement/index?${params.join('&')}`
}

export const createLatestRequestGuard = () => {
  let latestRequestId = 0
  return {
    next: () => {
      latestRequestId += 1
      return latestRequestId
    },
    isLatest: requestId => requestId === latestRequestId
  }
}

export const buildHonorWallTabs = (viewerScope = 'self') => {
  const tabs = [
    { key: 'career', label: '生涯成就' },
    { key: 'season', label: '当前赛季' },
    { key: 'history', label: '历届荣誉' }
  ]
  return viewerScope === 'friend' ? tabs.filter(item => item.key !== 'season') : tabs
}

export const buildCareerSummaryItems = (summary = {}, gameType = 3, gameTypeLabel = '') => {
  const selectedGameType = normalizeHonorWallGameType(gameType)
  const labels = { 1: '斯诺克', 2: '九球追分', 3: '中式八球', 4: '美式九球' }
  const universalTotal = Math.max(0, Number(summary.universal_total) || 0)
  const universalUnlocked = Math.max(0, Number(summary.universal_unlocked) || 0)
  const specialtyTotal = Math.max(0, Number(summary.specialty_total) || 0)
  const specialtyUnlocked = Math.max(0, Number(summary.specialty_unlocked) || 0)
  return [
    { key: 'universal', label: '通用成就', value: `${universalUnlocked}/${universalTotal}` },
    { key: 'specialty', label: `${gameTypeLabel || labels[selectedGameType]}专精`, value: `${specialtyUnlocked}/${specialtyTotal}` },
    { key: 'season', label: '赛季荣誉', value: Math.max(0, Number(summary.season_honors) || 0) },
    { key: 'tournament', label: '赛事荣誉', value: Math.max(0, Number(summary.tournament_honors) || 0) }
  ]
}

export const buildUpcomingAchievementSection = (achievements = [], gameType = 3, viewerScope = 'self') => {
  if (viewerScope === 'friend') {
    return { hidden: true, completed: false, items: [] }
  }

  const selectedGameType = normalizeHonorWallGameType(gameType)
  const applicable = (Array.isArray(achievements) ? achievements : []).filter(item => {
    const itemGameType = Number(item?.game_type || 0)
    return itemGameType === 0 || itemGameType === selectedGameType
  })
  const locked = applicable
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => !Boolean(item?.unlocked) && Number(item?.threshold || 0) > 0)
    .map(({ item, index }) => {
      const progress = Math.max(0, Number(item.progress) || 0)
      const threshold = Math.max(0, Number(item.threshold) || 0)
      const progressPercent = getProgressPercent({ progress, threshold })
      return {
        ...item,
        progress,
        threshold,
        progressPercent,
        progressText: `${progress}/${threshold}`,
        remainingText: `还差 ${Math.max(0, threshold - progress)} 达成`,
        _definitionOrder: index
      }
    })

  locked.sort((left, right) => (
    right.progressPercent - left.progressPercent ||
    left._definitionOrder - right._definitionOrder ||
    Number(left.id || 0) - Number(right.id || 0)
  ))
  return {
    hidden: false,
    completed: applicable.length > 0 && locked.length === 0,
    items: locked.slice(0, 3).map(({ _definitionOrder, ...item }) => item)
  }
}

export const getProgressPercent = ({ progress = 0, threshold = 0, completed = false, unlocked = false } = {}) => {
  if (completed || unlocked) return 100
  const safeThreshold = Number(threshold) || 0
  if (safeThreshold <= 0) return 0
  const percent = Math.round((Math.max(0, Number(progress) || 0) / safeThreshold) * 100)
  return Math.min(100, percent)
}

export const buildChallengeViewModel = (challenge = {}) => {
  const progress = Math.max(0, Number(challenge.progress) || 0)
  const threshold = Math.max(0, Number(challenge.threshold) || 0)
  const completed = Boolean(challenge.completed || (threshold > 0 && progress >= threshold))
  return {
    ...challenge,
    progress,
    threshold,
    completed,
    icon: challengeIconPaths[challenge.key] || '',
    progressPercent: getProgressPercent({ progress, threshold, completed }),
    progressText: completed ? '已完成' : `${progress}/${threshold}`,
    remainingText: completed ? '本赛季已达成' : `还差 ${Math.max(0, threshold - progress)} 完成`
  }
}

export const resolveCurrentSeasonState = (currentSeason) => {
  if (!currentSeason || !Number(currentSeason.season_id || 0)) {
    return {
      mode: 'intermission',
      title: '赛季间歇中',
      description: '当前没有进行中的赛季，生涯成就与历届荣誉仍会永久保留。'
    }
  }
  return {
    mode: 'active',
    title: currentSeason.season_name || '当前赛季',
    description: '挑战按当前赛季与球种独立累计，新赛季会从零开始。'
  }
}

export const buildRecentHonorViewModel = (item = {}) => ({
  ...item,
  sourceLabel: honorSourceLabels[item.source_type] || '荣誉',
  detailText: item.reward_title_name
    ? `同时获得称号「${item.reward_title_name}」`
    : (item.description || item.source_ref_name || '已永久记录')
})

export const buildHistorySeasonOptions = (honors = []) => {
  const seen = new Set()
  return (Array.isArray(honors) ? honors : []).reduce((result, item = {}) => {
    const seasonId = Number(item.source_ref_id || 0)
    if (item.source_type !== 'season' || seasonId <= 0 || seen.has(seasonId)) return result
    seen.add(seasonId)
    result.push({
      id: seasonId,
      label: item.source_ref_name || item.name || `赛季 ${seasonId}`
    })
    return result
  }, [])
}

export const findLatestUnreadSeasonRollover = (list = []) => {
  const unread = (Array.isArray(list) ? list : []).filter(item => (
    item?.type === 'season_rollover' && !Boolean(item.is_read)
  ))
  unread.sort((left, right) => {
    const timeDiff = String(right.created_at || '').localeCompare(String(left.created_at || ''))
    return timeDiff || (Number(right.id || 0) - Number(left.id || 0))
  })
  return unread[0] || null
}

export const buildSeasonRolloverModal = (notification = {}) => {
  const data = parseData(notification.data)
  const fromName = data.from_season_name || '上一赛季'
  const toName = data.to_season_name || ''
  const intermission = Boolean(data.intermission || !Number(data.to_season_id || 0))
  const lines = [
    `${fromName} 已归档，生涯成就保持不变。`,
    `赛季挑战已归档 ${Number(data.challenge_archived_count || 0)} 项，其中完成 ${Number(data.challenge_completed_count || 0)} 项。`,
    `赛季荣誉已记录 ${Number(data.season_title_count || 0)} 枚。`
  ]
  if (intermission) {
    lines.push('当前处于赛季间歇期，新赛季开启后会自动更新。')
  } else {
    lines.push(`${toName || '新赛季'} 已自动开启，赛季挑战从零开始。`)
  }
  return {
    title: intermission ? `${fromName} 已结算` : `${toName || '新赛季'} 已开启`,
    content: lines.join('\n'),
    confirmText: '知道了',
    showCancel: false
  }
}

let rolloverPresentationPromise = null

export const presentLatestSeasonRollover = ({
  getNotificationList,
  markAsRead,
  showModal,
  onRead
} = {}) => {
  if (rolloverPresentationPromise) return rolloverPresentationPromise
  if (typeof getNotificationList !== 'function' || typeof markAsRead !== 'function' || typeof showModal !== 'function') {
    return Promise.resolve(null)
  }

  rolloverPresentationPromise = (async () => {
    const response = await getNotificationList({
      page: 1,
      page_size: 10,
      type: 'season_rollover'
    })
    const notification = findLatestUnreadSeasonRollover(response?.list || response || [])
    if (!notification) return null

    const modal = buildSeasonRolloverModal(notification)
    await new Promise(resolve => {
      showModal({
        ...modal,
        complete: resolve
      })
    })
    await markAsRead({ notification_id: notification.id })
    if (typeof onRead === 'function') await onRead(notification)
    return notification
  })().finally(() => {
    rolloverPresentationPromise = null
  })

  return rolloverPresentationPromise
}
