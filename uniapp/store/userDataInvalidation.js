import { defineStore } from 'pinia'

export const USER_DATA_SCOPES = [
  'rank',
  'stats',
  'h2h',
  'opponents',
  'history',
  'honor',
  'member',
  'reputation',
  'season',
  'leaderboard',
  'friend_requests',
  'notification'
]

const emptyScopeVersions = () => USER_DATA_SCOPES.reduce((versions, scope) => {
  versions[scope] = 0
  return versions
}, {})

const normalizeIdentity = ({ userId = 0, authGeneration = -1 } = {}) => ({
  userId: Number(userId) || 0,
  authGeneration: Number(authGeneration)
})

export const useUserDataInvalidationStore = defineStore('userDataInvalidation', {
  state: () => ({
    ownerUserId: 0,
    authGeneration: -1,
    competitiveRevision: 0,
    scopeVersions: emptyScopeVersions()
  }),

  actions: {
    matchesIdentity(identity) {
      const normalized = normalizeIdentity(identity)
      return this.ownerUserId === normalized.userId && this.authGeneration === normalized.authGeneration
    },

    ensureIdentity(identity) {
      const normalized = normalizeIdentity(identity)
      if (normalized.userId <= 0) {
        this.clear()
        return false
      }
      if (!this.matchesIdentity(normalized)) {
        this.ownerUserId = normalized.userId
        this.authGeneration = normalized.authGeneration
        this.competitiveRevision = 0
        this.scopeVersions = emptyScopeVersions()
      }
      return true
    },

    invalidate(identity, scopes = [], competitiveRevision = 0) {
      if (!this.ensureIdentity(identity)) return
      const revision = Number(competitiveRevision) || 0
      if (revision > this.competitiveRevision) {
        this.competitiveRevision = revision
      }
      for (const scope of scopes) {
        if (Object.prototype.hasOwnProperty.call(this.scopeVersions, scope)) {
          this.scopeVersions[scope] += 1
        }
      }
    },

    versionOf(scope) {
      return Number(this.scopeVersions[scope]) || 0
    },

    clear() {
      this.ownerUserId = 0
      this.authGeneration = -1
      this.competitiveRevision = 0
      this.scopeVersions = emptyScopeVersions()
    }
  }
})
