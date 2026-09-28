import { defineStore } from 'pinia'
import { getCurrentMatch } from '@/api/match.js'
import { getChallengeSummary } from '@/api/challenge.js'
import { getFriendRequests } from '@/api/friend.js'
import { getNotificationList, getUnreadCount } from '@/api/notification.js'
import { useFriendRequestStore } from './friendRequest.js'
import { useNotificationStore } from './notification.js'

export const ACTIVITY_CACHE_TTL = 30 * 1000

const emptyActivityState = () => ({
  currentMatch: null,
  currentChallenge: null,
  receivedChallengeCount: 0,
  challengeServerTime: '',
  // 约球区块可用性：读取失败时页面不得把未知状态当作「无约球」。
  challengeAvailable: true,
  // 失效标记计数：请求期间的写入/WS 标脏不得被响应覆盖。
  dirtySeq: 0,
  unreadCount: 0,
  pendingFriendRequestCount: 0,
  latestSeasonRollover: null
})

const normalizeIdentity = ({ userId = 0, authGeneration = -1 } = {}) => ({
  userId: Number(userId) || 0,
  authGeneration: Number(authGeneration)
})

const findLatestSeasonRollover = (response = {}) => {
  const list = Array.isArray(response?.list) ? response.list : []
  return list.find((item) => item?.type === 'season_rollover' && Number(item?.is_read) !== 1) || null
}

export const useActivityStore = defineStore('activity', {
  state: () => ({
    ownerUserId: 0,
    authGeneration: -1,
    loaded: false,
    dirty: true,
    loadedAt: 0,
    inFlight: null,
    reservationRelease: null,
    lastError: '',
    ...emptyActivityState()
  }),

  getters: {
    isFresh: (state) => !state.dirty && state.loaded && Date.now() - state.loadedAt < ACTIVITY_CACHE_TTL
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
        const releaseReservation = this.reservationRelease
        Object.assign(this, {
          ownerUserId: normalized.userId,
          authGeneration: normalized.authGeneration,
          loaded: false,
          dirty: true,
          loadedAt: 0,
          inFlight: null,
          reservationRelease: null,
          lastError: '',
          ...emptyActivityState()
        })
        releaseReservation?.()
      }
      return true
    },

    shouldFetch({ force = false, now = Date.now() } = {}) {
      return force || !this.loaded || this.dirty || now - this.loadedAt >= ACTIVITY_CACHE_TTL
    },

    async fetch(identity, { force = false, silent = true } = {}) {
      if (!this.ensureIdentity(identity)) {
        return emptyActivityState()
      }
      if (!this.shouldFetch({ force })) {
        return this.snapshot()
      }
      if (this.inFlight) {
        return this.inFlight
      }

      const startDirtySeq = this.dirtySeq
      const expectedIdentity = normalizeIdentity(identity)
      const request = Promise.allSettled([
        getCurrentMatch({ silent }),
        getChallengeSummary(),
        getUnreadCount(),
        getFriendRequests({ page: 1, page_size: 1 }),
        getNotificationList({ page: 1, page_size: 1, type: 'season_rollover' })
      ]).then((results) => {
        if (!this.matchesIdentity(expectedIdentity) || this.inFlight !== request) {
          return this.snapshot()
        }

        const [matchResult, challengeResult, unreadResult, friendRequestResult, seasonResult] = results
        const failed = results.some((result) => result.status === 'rejected')
        if (matchResult.status === 'fulfilled') {
          const response = matchResult.value || {}
          this.currentMatch = response.success && response.match ? response.match : null
        }
        if (challengeResult.status === 'fulfilled') {
          const summary = challengeResult.value || {}
          this.challengeAvailable = summary.success !== false
          this.currentChallenge = summary.success && summary.current_challenge ? summary.current_challenge : null
          this.receivedChallengeCount = Number(summary.received_pending_count) || 0
          this.challengeServerTime = summary.server_time || ''
        } else {
          this.challengeAvailable = false
        }
        if (unreadResult.status === 'fulfilled') {
          this.unreadCount = Number(unreadResult.value?.count) || 0
          useNotificationStore().setUnreadCount(this.unreadCount, expectedIdentity)
        }
        if (friendRequestResult.status === 'fulfilled') {
          this.pendingFriendRequestCount = Number(friendRequestResult.value?.total) || 0
          useFriendRequestStore().setPendingCount(this.pendingFriendRequestCount, expectedIdentity)
        }
        if (seasonResult.status === 'fulfilled') {
          this.latestSeasonRollover = findLatestSeasonRollover(seasonResult.value)
        }

        this.loaded = this.loaded || !failed
        // 请求期间发生的新标脏（写成功/WS）不得被本次响应覆盖。
        this.dirty = failed || this.dirtySeq > startDirtySeq
        this.lastError = failed ? '活动状态部分加载失败' : ''
        if (!failed) {
          this.loadedAt = Date.now()
        }
        return this.snapshot()
      })

      this.inFlight = request
      return request.finally(() => {
        if (this.inFlight === request) {
          this.inFlight = null
        }
      })
    },

    // 标脏后的补读：fetch 会直接复用 inFlight；旧批次仍在途中时先等它结束，再发起新读取。
    async refreshIfDirty(identity, { silent = true } = {}) {
      if (!this.matchesIdentity(identity) || !this.dirty) return
      if (this.inFlight) {
        await this.inFlight.catch(() => {})
      }
      if (!this.matchesIdentity(identity) || !this.dirty) return
      await this.fetch(identity, { silent })
    },

    reserveBootstrap(identity) {
      if (!this.ensureIdentity(identity) || this.inFlight) {
        return null
      }
      const expectedIdentity = normalizeIdentity(identity)
      let resolve
      const request = new Promise((resolveRequest) => {
        resolve = resolveRequest
      })
      let settled = false
      const release = () => {
        if (settled) return
        settled = true
        if (this.inFlight === request) {
          this.inFlight = null
        }
        if (this.reservationRelease === release) {
          this.reservationRelease = null
        }
        resolve(this.snapshot())
      }
      this.inFlight = request
      const startDirtySeq = this.dirtySeq
      this.reservationRelease = release
      return {
        matches: (candidate) => {
          const normalized = normalizeIdentity(candidate)
          return normalized.userId === expectedIdentity.userId && normalized.authGeneration === expectedIdentity.authGeneration
        },
        apply: (bootstrap) => {
          if (this.inFlight !== request) return
          this.applyBootstrap(identity, bootstrap, { startDirtySeq })
          release()
        },
        release
      }
    },

    applyBootstrap(identity, bootstrap = {}, { startDirtySeq = null } = {}) {
      if (!this.ensureIdentity(identity)) {
        return this.snapshot()
      }
      const availability = bootstrap?.availability || {}
      const isAvailable = (scope) => availability[scope] !== false

      if (isAvailable('current_match')) {
        this.currentMatch = bootstrap?.current_match || null
      }
      if (isAvailable('current_challenge')) {
        this.currentChallenge = bootstrap?.current_challenge || null
      }
      if (isAvailable('challenge_received_count')) {
        this.receivedChallengeCount = Number(bootstrap?.received_pending_challenge_count) || 0
      }
      const challengeScopes = ['current_challenge', 'challenge_received_count']
      if (challengeScopes.some((scope) => Object.prototype.hasOwnProperty.call(availability, scope))) {
        this.challengeAvailable = challengeScopes.every((scope) => availability[scope] !== false)
      }
      if (bootstrap?.server_time) {
        this.challengeServerTime = bootstrap.server_time
      }
      if (isAvailable('unread_count')) {
        this.setUnreadCount(bootstrap?.unread_count)
      }
      if (isAvailable('pending_friend_request_count')) {
        this.setPendingFriendRequestCount(bootstrap?.pending_friend_request_count)
      }
      if (isAvailable('latest_season_rollover')) {
        this.latestSeasonRollover = bootstrap?.latest_season_rollover || null
      }

      const requiredScopes = [
        'current_match',
        'current_challenge',
        'challenge_received_count',
        'unread_count',
        'pending_friend_request_count',
        'latest_season_rollover'
      ]
      const partial = requiredScopes.some((scope) => availability[scope] === false)
      this.loaded = true
      // 读取期间发生的新标脏不得被这份快照覆盖；调用方可传入请求开始时的序号。
      this.dirty = partial || (startDirtySeq !== null && this.dirtySeq > startDirtySeq)
      this.lastError = partial ? '活动状态部分加载失败' : ''
      this.loadedAt = Date.now()
      return this.snapshot()
    },

    markDirty() {
      if (this.ownerUserId > 0) {
        this.dirty = true
        this.dirtySeq += 1
      }
    },

    setCurrentMatch(match) {
      this.currentMatch = match || null
    },

    setUnreadCount(count, identity = { userId: this.ownerUserId, authGeneration: this.authGeneration }) {
      if (!this.matchesIdentity(identity)) return
      this.unreadCount = Math.max(0, Number(count) || 0)
      useNotificationStore().setUnreadCount(this.unreadCount, identity)
    },

    setPendingFriendRequestCount(count, identity = { userId: this.ownerUserId, authGeneration: this.authGeneration }) {
      if (!this.matchesIdentity(identity)) return
      this.pendingFriendRequestCount = Math.max(0, Number(count) || 0)
      useFriendRequestStore().setPendingCount(this.pendingFriendRequestCount, identity)
    },

    setLatestSeasonRollover(notification) {
      this.latestSeasonRollover = notification || null
    },

    snapshot() {
      return {
        currentMatch: this.currentMatch,
        currentChallenge: this.currentChallenge,
        receivedChallengeCount: this.receivedChallengeCount,
        challengeServerTime: this.challengeServerTime,
        unreadCount: this.unreadCount,
        pendingFriendRequestCount: this.pendingFriendRequestCount,
        latestSeasonRollover: this.latestSeasonRollover
      }
    },

    clear() {
      const releaseReservation = this.reservationRelease
      Object.assign(this, {
        ownerUserId: 0,
        authGeneration: -1,
        loaded: false,
        dirty: true,
        loadedAt: 0,
        inFlight: null,
        reservationRelease: null,
        lastError: '',
        ...emptyActivityState()
      })
      releaseReservation?.()
    }
  }
})
