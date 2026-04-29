const REPORT_POST_TYPE = 1
const LOGIN_REQUIRED_TOOL_URLS = new Set([
  '/subPages/user/statsDetail'
])
export const HOME_NEARBY_VENUE_RADIUS_METERS = 5000
export const HOME_NEARBY_VENUE_LIMIT = 3
const EVENT_NEWS_STATUS_MAP = Object.freeze({
  0: '即将开始',
  1: '进行中',
  2: '已结束',
  3: '已取消'
})

const formatTimestamp = (dateTime) => {
  if (!dateTime) return Number.POSITIVE_INFINITY
  const parsed = new Date(dateTime).getTime()
  return Number.isNaN(parsed) ? Number.POSITIVE_INFINITY : parsed
}

export const getEventNewsStatusText = (status, fallback = '赛事情报') => {
  return EVENT_NEWS_STATUS_MAP[Number(status)] || fallback
}

export const formatEventNewsTime = (dateTime, now = Date.now()) => {
  const currentTime = typeof now === 'string' ? new Date(now).getTime() : now
  const eventTime = formatTimestamp(dateTime)

  if (!Number.isFinite(eventTime) || !Number.isFinite(currentTime)) {
    return '时间待定'
  }

  const date = new Date(eventTime)
  const currentDate = new Date(currentTime)
  const today = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate()).getTime()
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  const diffDays = Math.round((target - today) / (24 * 60 * 60 * 1000))
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')

  if (diffDays === 0) return `今天 ${hh}:${mm}`
  if (diffDays === 1) return `明天 ${hh}:${mm}`
  if (diffDays === 2) return `后天 ${hh}:${mm}`

  return `${date.getMonth() + 1}月${date.getDate()}日 ${hh}:${mm}`
}

export const buildFeaturedPostTarget = (post) => {
  if (!post || post.post_type !== REPORT_POST_TYPE) {
    return {
      type: 'community',
      url: '/pages/social/index',
      ctaText: '去社区查看更多'
    }
  }

  if (post.match_id) {
    return {
      type: 'navigate',
      url: `/subPages/match/shareResult?match_id=${post.match_id}`,
      ctaText: '查看战报'
    }
  }

  return {
    type: 'community',
    url: '/pages/social/index',
    ctaText: '去社区查看更多'
  }
}

export const resolveHomeToolNavigation = ({ url = '', isLoggedIn = false, isTabPage = false } = {}) => {
  if (isTabPage) {
    return {
      type: 'tab',
      url
    }
  }

  if (!isLoggedIn && LOGIN_REQUIRED_TOOL_URLS.has(url)) {
    return {
      type: 'login',
      url: '/pages/login/login'
    }
  }

  return {
    type: 'navigate',
    url
  }
}

export const shouldShowHomeToolEdgeMask = ({ maxScrollLeft = 0, scrollLeft = 0, edgeThreshold = 4 } = {}) => {
  const normalizedMaxScrollLeft = Number(maxScrollLeft) || 0
  if (normalizedMaxScrollLeft <= edgeThreshold) {
    return false
  }

  return (Number(scrollLeft) || 0) < (normalizedMaxScrollLeft - edgeThreshold)
}

export const buildHomeNearbyVenueParams = ({ latitude = 0, longitude = 0 } = {}) => {
  const lat = Number(latitude) || 0
  const lng = Number(longitude) || 0

  if (!lat || !lng) return null

  return {
    latitude: lat,
    longitude: lng,
    radius: HOME_NEARBY_VENUE_RADIUS_METERS,
    limit: HOME_NEARBY_VENUE_LIMIT
  }
}

export const formatHomeVenueDistance = (meters = 0) => {
  const value = Number(meters) || 0
  if (value <= 0) return ''
  if (value < 1000) return `${Math.round(value)}m`
  return `${(value / 1000).toFixed(1)}km`
}

export const resolveHomeVenueEmptyAction = (rewardStatus) => {
  const rewardEnabled = rewardStatus?.enabled === true

  return {
    text: rewardEnabled ? '添加球馆领取会员' : '添加附近球馆',
    desc: rewardEnabled ? '你可以提交常玩球馆，审核通过后会员会自动到账。' : '你可以提交常去球馆，审核通过后会展示给附近球友。',
    url: '/subPages/venue/submit'
  }
}
