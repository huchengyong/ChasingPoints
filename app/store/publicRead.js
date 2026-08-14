import { defineStore } from 'pinia'
import { getEventNewsList, getEventNewsView } from '@/api/event-news.js'
import { getLeaderboard, getLeaderboardSummary } from '@/api/rank.js'
import { getNearbyVenues } from '@/api/venue.js'
import { buildNearbyVenueCacheKey, buildPublicReadKey } from '@/utils/public-read-cache.js'
import { readStaticReadCache, writeStaticReadCache } from '@/utils/static-read-cache.js'

const EVENT_NEWS_TTL = 5 * 60 * 1000
const LEADERBOARD_TTL = 30 * 1000
const NEARBY_VENUES_TTL = 3 * 60 * 1000
const STATIC_DATA_TTL = 24 * 60 * 60 * 1000

const isFresh = (entry, ttl, now = Date.now()) => (
  Boolean(entry?.loadedAt) && now - entry.loadedAt < ttl
)

const leaderboardOwnerKey = ({ userId = 0, authGeneration = -1 } = {}) => {
  const normalizedUserId = Number(userId) || 0
  const normalizedGeneration = Number(authGeneration)
  return normalizedUserId > 0 && Number.isFinite(normalizedGeneration)
    ? `user:${normalizedUserId}:${normalizedGeneration}`
    : 'guest'
}

const leaderboardCacheKey = (scope, params, identity) => buildPublicReadKey(scope, {
  ...params,
  viewer: leaderboardOwnerKey(identity)
})

const normalizeLeaderboardResponse = (response = {}) => ({
  success: Boolean(response.success),
  total: Number(response.total) || 0,
  top_three: Array.isArray(response.top_three) ? response.top_three : [],
  my_ranking: response.my_ranking || null,
  list: Array.isArray(response.list) ? response.list : [],
  version: String(response.version || '')
})

export const usePublicReadStore = defineStore('publicRead', {
  state: () => ({
    eventNews: {},
    eventNewsViews: {},
    leaderboard: {},
    nearbyVenues: {},
    staticData: {}
  }),

  actions: {
    async loadEntry(group, key, loader, { ttl, force = false } = {}) {
      const entries = this[group]
      const existing = entries[key]
      if (!force && isFresh(existing, ttl)) {
        return existing.value
      }
      if (existing?.inFlight) {
        return existing.inFlight
      }

      const entry = existing || { value: null, loadedAt: 0, inFlight: null }
      const request = Promise.resolve()
        .then(loader)
        .then((value) => {
          entry.value = value
          entry.loadedAt = Date.now()
          entry.lastError = ''
          return value
        })
        .catch((error) => {
          entry.lastError = error?.message || '公共数据加载失败'
          if (entry.value !== null) return entry.value
          throw error
        })
        .finally(() => {
          if (entry.inFlight === request) {
            entry.inFlight = null
          }
        })

      entry.inFlight = request
      entries[key] = entry
      return request
    },

    loadEventNews(params = {}, options = {}) {
      const key = buildPublicReadKey('event-news', params)
      return this.loadEntry('eventNews', key, () => getEventNewsList(params), {
        ttl: EVENT_NEWS_TTL,
        force: options.force
      })
    },

    loadEventNewsView(params = {}, options = {}) {
      const key = buildPublicReadKey('event-news-view', params)
      return this.loadEntry('eventNewsViews', key, () => getEventNewsView(params), {
        ttl: EVENT_NEWS_TTL,
        force: options.force
      })
    },

    loadLeaderboard(params = {}, options = {}) {
      const key = leaderboardCacheKey('leaderboard', params, options.identity)
      return this.loadEntry('leaderboard', key, async () => normalizeLeaderboardResponse(await getLeaderboard(params)), {
        ttl: LEADERBOARD_TTL,
        force: options.force
      })
    },

    loadLeaderboardSummary(params = {}, options = {}) {
      const key = leaderboardCacheKey('leaderboard-summary', params, options.identity)
      return this.loadEntry('leaderboard', key, async () => normalizeLeaderboardResponse(await getLeaderboardSummary(params)), {
        ttl: LEADERBOARD_TTL,
        force: options.force
      })
    },

    loadNearbyVenues(params = {}, options = {}) {
      const key = buildNearbyVenueCacheKey(params)
      return this.loadEntry('nearbyVenues', key, () => getNearbyVenues(params), {
        ttl: NEARBY_VENUES_TTL,
        force: options.force
      })
    },

    loadStatic(key, loader, options = {}) {
      const version = options.version || 1
      if (!options.force && !this.staticData[key]) {
        const cached = readStaticReadCache(key, version)
        if (cached) {
          this.staticData[key] = { value: cached.value, loadedAt: cached.loadedAt, inFlight: null }
        }
      }
      return this.loadEntry('staticData', key, async () => {
        const value = await loader()
        writeStaticReadCache(key, value, version)
        return value
      }, {
        ttl: options.ttl || STATIC_DATA_TTL,
        force: options.force
      })
    },

    clearLeaderboard() {
      this.leaderboard = {}
    },

    invalidate(group, key = '') {
      if (!this[group]) return
      if (key) {
        delete this[group][key]
        return
      }
      this[group] = {}
    }
  }
})
