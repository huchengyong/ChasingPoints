import { defineStore } from 'pinia'
import { getUserOverview } from '@/api/user.js'

export const USER_OVERVIEW_CACHE_TTL = 60 * 1000

const emptyOverview = () => ({
  stats: null,
  reputation: null,
  memberStatus: null,
  favoriteVenueRewardStatus: null,
  availability: {
    stats: false,
    reputation: false,
    memberStatus: false,
    favoriteVenueRewardStatus: false
  }
})

const normalizeIdentity = ({ userId = 0, authGeneration = -1 } = {}) => ({
  userId: Number(userId) || 0,
  authGeneration: Number(authGeneration)
})

export const useUserOverviewStore = defineStore('userOverview', {
  state: () => ({
    ownerUserId: 0,
    authGeneration: -1,
    loaded: false,
    dirty: true,
    loadedAt: 0,
    inFlight: null,
    lastError: '',
    ...emptyOverview()
  }),

  getters: {
    isFresh: (state) => !state.dirty && state.loaded && Date.now() - state.loadedAt < USER_OVERVIEW_CACHE_TTL
  },

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
        Object.assign(this, {
          ownerUserId: normalized.userId,
          authGeneration: normalized.authGeneration,
          loaded: false,
          dirty: true,
          loadedAt: 0,
          inFlight: null,
          lastError: '',
          ...emptyOverview()
        })
      }
      return true
    },

    shouldFetch({ force = false, now = Date.now() } = {}) {
      return force || !this.loaded || this.dirty || now - this.loadedAt >= USER_OVERVIEW_CACHE_TTL
    },

    async fetch(identity, { force = false, silent = true } = {}) {
      if (!this.ensureIdentity(identity)) {
        return this.snapshot()
      }
      if (!this.shouldFetch({ force })) {
        return this.snapshot()
      }
      if (this.inFlight) {
        return this.inFlight
      }

      const expectedIdentity = normalizeIdentity(identity)
      const request = getUserOverview({ silent }).then((response = {}) => {
        if (!this.matchesIdentity(expectedIdentity) || this.inFlight !== request) {
          return this.snapshot()
        }
        if (!response.success) {
          this.dirty = true
          this.lastError = response.message || '用户概览加载失败'
          return this.snapshot()
        }

        const availability = response.availability || {}
        const apply = (scope, stateKey, responseKey) => {
          const available = availability[scope] !== false
          this.availability[stateKey] = available
          if (available) {
            this[stateKey] = response[responseKey] || null
          }
          return !available
        }
        const partial = [
          apply('stats', 'stats', 'stats'),
          apply('reputation', 'reputation', 'reputation'),
          apply('member', 'memberStatus', 'member_status'),
          apply('favorite_venue_reward', 'favoriteVenueRewardStatus', 'favorite_venue_reward')
        ].some(Boolean)
        this.loaded = true
        this.dirty = partial
        this.lastError = partial ? '用户概览部分加载失败' : ''
        this.loadedAt = Date.now()
        return this.snapshot()
      })

      this.inFlight = request
      return request.finally(() => {
        if (this.inFlight === request) {
          this.inFlight = null
        }
      })
    },

    markDirty() {
      if (this.ownerUserId > 0) {
        this.dirty = true
      }
    },

    snapshot() {
      return {
        stats: this.stats,
        reputation: this.reputation,
        memberStatus: this.memberStatus,
        favoriteVenueRewardStatus: this.favoriteVenueRewardStatus,
        availability: { ...this.availability }
      }
    },

    clear() {
      Object.assign(this, {
        ownerUserId: 0,
        authGeneration: -1,
        loaded: false,
        dirty: true,
        loadedAt: 0,
        inFlight: null,
        lastError: '',
        ...emptyOverview()
      })
    }
  }
})
