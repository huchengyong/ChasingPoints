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

const normalizeIdentity = (identity = {}) => {
  if (typeof identity === 'object' && identity !== null) {
    return {
      userId: Number(identity.userId) || 0,
      authGeneration: Number.isFinite(Number(identity.authGeneration))
        ? Number(identity.authGeneration)
        : -1
    }
  }
  return { userId: Number(identity) || 0, authGeneration: -1 }
}

const sameIdentity = (left, right) => (
  left.userId === right.userId && left.authGeneration === right.authGeneration
)

export const createRankCacheState = () => ({
  ownerUserId: 0,
  authGeneration: -1,
  rankInfoMap: {},
  loaded: false,
  dirty: false,
  generation: 0,
  loading: false,
  inFlight: null
})

export const createRankCacheActions = ({ requestRankInfos, normalizeRankInfos = normalizeUserRankInfos }) => ({
  ensureOwner(identity) {
    const expected = normalizeIdentity(identity)
    if (!sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected)) {
      this.clear()
      this.ownerUserId = expected.userId
      this.authGeneration = expected.authGeneration
    }
    return expected
  },

  ensureFresh(identity) {
    const expected = this.ensureOwner(identity)
    if (!expected.userId) return Promise.resolve(null)
    if (this.loaded && !this.dirty) return Promise.resolve(this.rankInfoMap)
    return this.refresh(expected)
  },

  forceRefresh(identity) {
    const expected = this.ensureOwner(identity)
    if (!expected.userId) return Promise.resolve(null)
    this.invalidate(expected)
    return this.refresh(expected)
  },

  refresh(identity) {
    const expected = normalizeIdentity(identity)
    if (!expected.userId || !sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected)) {
      return Promise.resolve(null)
    }
    if (this.inFlight) return this.inFlight

    this.loading = true
    const request = (async () => {
      try {
        while (sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected)) {
          const requestGeneration = this.generation
          let response
          try {
            response = await requestRankInfos()
          } catch (error) {
            if (sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected) && this.generation !== requestGeneration) continue
            throw error
          }

          if (!sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected)) return null
          if (this.generation !== requestGeneration) continue
          if (!response?.success) throw new Error(response?.message || '获取段位信息失败')

          const nextMap = normalizeRankInfos(response)
          if (!hasCompleteRankInfoMap(nextMap)) throw new Error('段位信息不完整')
          if (!sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected) || this.generation !== requestGeneration) continue

          this.rankInfoMap = nextMap
          this.loaded = true
          this.dirty = false
          return nextMap
        }
        return null
      } finally {
        if (this.inFlight === request) this.inFlight = null
        if (sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected)) this.loading = false
      }
    })()

    this.inFlight = request
    return request
  },

  invalidate(identity = { userId: this.ownerUserId, authGeneration: this.authGeneration }) {
    const expected = normalizeIdentity(identity)
    if (!sameIdentity({ userId: this.ownerUserId, authGeneration: this.authGeneration }, expected)) return
    this.generation += 1
    this.dirty = true
  },

  clear() {
    this.ownerUserId = 0
    this.authGeneration = -1
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
