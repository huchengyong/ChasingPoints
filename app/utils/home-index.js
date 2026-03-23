const REPORT_POST_TYPE = 1
const ENDED_TOURNAMENT_STATUS = 2
const getTournamentPriority = (status) => {
  if (status === 1) return 0
  if (status === 0) return 1
  return 2
}

const getTimestamp = (dateTime) => {
  if (!dateTime) return Number.POSITIVE_INFINITY
  const parsed = new Date(dateTime).getTime()
  return Number.isNaN(parsed) ? Number.POSITIVE_INFINITY : parsed
}

export const pickFeaturedTournament = (list = [], now = Date.now()) => {
  const currentTime = typeof now === 'string' ? new Date(now).getTime() : now
  const validList = Array.isArray(list) ? list.filter(Boolean) : []

  if (!validList.length) return null

  const candidates = validList
    .filter((item) => item.status !== ENDED_TOURNAMENT_STATUS)
    .sort((left, right) => {
      const leftPriority = getTournamentPriority(left.status)
      const rightPriority = getTournamentPriority(right.status)

      if (leftPriority !== rightPriority) {
        return leftPriority - rightPriority
      }

      const leftDiff = Math.abs(getTimestamp(left.start_time) - currentTime)
      const rightDiff = Math.abs(getTimestamp(right.start_time) - currentTime)
      return leftDiff - rightDiff
    })

  return candidates[0] || validList[0]
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
