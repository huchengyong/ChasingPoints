export const SUPPORTED_RANK_GAME_TYPES = [1, 2, 3, 4]

export const normalizeUserRankInfos = (response = {}) => {
  const items = Array.isArray(response.rank_infos) ? response.rank_infos : []
  return items.reduce((map, item) => {
    const gameType = Number(item?.game_type || 0)
    if (SUPPORTED_RANK_GAME_TYPES.includes(gameType) && item?.rank_info) {
      map[gameType] = item.rank_info
    }
    return map
  }, {})
}

export const hasCompleteRankInfoMap = (rankInfoMap = {}) => {
  return SUPPORTED_RANK_GAME_TYPES.every((gameType) => rankInfoMap[gameType])
}

export const createRankCacheState = () => ({
  ownerUserId: 0,
  rankInfoMap: {},
  loaded: false,
  dirty: false,
  generation: 0,
  loading: false,
  inFlight: null
})

export const createRankCacheActions = ({ requestRankInfos, normalizeRankInfos = normalizeUserRankInfos }) => ({
  ensureOwner(userId) {
    const normalizedUserId = Number(userId) || 0
    if (this.ownerUserId !== normalizedUserId) {
      this.clear()
      this.ownerUserId = normalizedUserId
    }
    return normalizedUserId
  },

  ensureFresh(userId) {
    const normalizedUserId = this.ensureOwner(userId)
    if (!normalizedUserId) return Promise.resolve(null)
    if (this.loaded && !this.dirty) return Promise.resolve(this.rankInfoMap)
    return this.refresh(normalizedUserId)
  },

  forceRefresh(userId) {
    const normalizedUserId = this.ensureOwner(userId)
    if (!normalizedUserId) return Promise.resolve(null)
    this.invalidate(normalizedUserId)
    return this.refresh(normalizedUserId)
  },

  refresh(userId) {
    const normalizedUserId = Number(userId) || 0
    if (!normalizedUserId || this.ownerUserId !== normalizedUserId) return Promise.resolve(null)
    if (this.inFlight) return this.inFlight

    this.loading = true
    const request = (async () => {
      try {
        while (this.ownerUserId === normalizedUserId) {
          const requestGeneration = this.generation
          let response
          try {
            response = await requestRankInfos()
          } catch (error) {
            if (this.ownerUserId === normalizedUserId && this.generation !== requestGeneration) continue
            throw error
          }

          if (this.ownerUserId !== normalizedUserId) return null
          if (this.generation !== requestGeneration) continue
          if (!response?.success) throw new Error(response?.message || '获取段位信息失败')

          const nextMap = normalizeRankInfos(response)
          if (!hasCompleteRankInfoMap(nextMap)) throw new Error('段位信息不完整')
          if (this.ownerUserId !== normalizedUserId || this.generation !== requestGeneration) continue

          this.rankInfoMap = nextMap
          this.loaded = true
          this.dirty = false
          return nextMap
        }
        return null
      } finally {
        if (this.inFlight === request) this.inFlight = null
        if (this.ownerUserId === normalizedUserId) this.loading = false
      }
    })()

    this.inFlight = request
    return request
  },

  invalidate(userId = this.ownerUserId) {
    if (Number(userId) !== this.ownerUserId) return
    this.generation += 1
    this.dirty = true
  },

  clear() {
    this.ownerUserId = 0
    this.rankInfoMap = {}
    this.loaded = false
    this.dirty = false
    this.generation += 1
    this.loading = false
    this.inFlight = null
  }
})

export const shouldInvalidateRankAfterSettlement = ({
  isParticipant,
  matchMode,
  status,
  result,
  player1Score,
  player2Score,
  myScore,
  opponentScore
} = {}) => {
  if (!isParticipant || matchMode !== 'ranked' || Number(status) !== 2) return false
  if (Number(result) > 0) return Number(result) !== 3

  const firstScore = Number.isFinite(player1Score) ? player1Score : myScore
  const secondScore = Number.isFinite(player2Score) ? player2Score : opponentScore
  return Number.isFinite(firstScore) && Number.isFinite(secondScore) && firstScore !== secondScore
}
