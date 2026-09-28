let ownerKey = ''
let entries = new Map()

const identityKey = ({ userId = 0, authGeneration = -1 } = {}) => (
  `${Number(userId) || 0}:${Number(authGeneration)}`
)

const ensureOwner = (identity) => {
  const nextKey = identityKey(identity)
  if (ownerKey !== nextKey) {
    ownerKey = nextKey
    entries = new Map()
  }
}

export const cacheAchievementDetail = (identity, achievement) => {
  if (!achievement?.id) return
  ensureOwner(identity)
  entries.set(String(achievement.id), { ...achievement })
}

export const getCachedAchievementDetail = (identity, achievementId) => {
  ensureOwner(identity)
  return entries.get(String(achievementId)) || null
}

export const clearAchievementDetailCache = () => {
  ownerKey = ''
  entries = new Map()
}
