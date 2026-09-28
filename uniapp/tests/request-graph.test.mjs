import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import { createRankCacheActions, createRankCacheState, normalizeUserRankInfos } from '../utils/rank-cache.js'
import { canApplySessionRecoveryResult, createSessionRecovery } from '../utils/session-recovery.js'
import { getRequestCountSnapshot, recordRequest, resetRequestCounts } from '../utils/request-metrics.js'

const appRoot = new URL('..', import.meta.url)
const readAppFile = (path) => readFileSync(new URL(path, appRoot), 'utf8')
const flush = () => new Promise((resolve) => setImmediate(resolve))

const createStoreLoader = (source, dependencies, exportName) => {
  const defineStore = (_name, definition) => () => {
    const state = definition.state()
    Object.entries(definition.actions).forEach(([name, action]) => {
      state[name] = action.bind(state)
    })
    return state
  }
  return new Function('defineStore', ...Object.keys(dependencies), `${source}; return ${exportName}`)(
    defineStore,
    ...Object.values(dependencies)
  )
}

const loadActivityStore = (mocks) => {
  const source = readAppFile('store/activity.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')
  return createStoreLoader(source, {
    getCurrentMatch: mocks.getCurrentMatch,
    getFriendRequests: mocks.getFriendRequests,
    getNotificationList: mocks.getNotificationList,
    getUnreadCount: mocks.getUnreadCount,
    useFriendRequestStore: () => ({ setPendingCount: () => {} }),
    useNotificationStore: () => ({ setUnreadCount: () => {} })
  }, 'useActivityStore')
}

const loadUserOverviewStore = (getUserOverview) => {
  const source = readAppFile('store/userOverview.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')
  return createStoreLoader(source, { getUserOverview }, 'useUserOverviewStore')
}

const loadPublicReadStore = (mocks) => {
  const source = readAppFile('store/publicRead.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')
  return createStoreLoader(source, {
    getEventNewsList: mocks.getEventNewsList,
    getEventNewsView: mocks.getEventNewsView || (() => Promise.resolve({ success: true })),
    getLeaderboard: mocks.getLeaderboard || (() => Promise.resolve({ success: true })),
    getLeaderboardSummary: mocks.getLeaderboardSummary,
    getNearbyVenues: mocks.getNearbyVenues,
    buildNearbyVenueCacheKey: () => 'nearby',
    buildPublicReadKey: (prefix, params) => `${prefix}:${JSON.stringify(params)}`
  }, 'usePublicReadStore')
}

const createRankStore = (requestRankInfos) => {
  const state = createRankCacheState()
  const actions = createRankCacheActions({ requestRankInfos, normalizeRankInfos: normalizeUserRankInfos })
  Object.entries(actions).forEach(([name, action]) => {
    state[name] = action.bind(state)
  })
  return state
}

const rankResponse = () => ({
  success: true,
  rank_infos: [1, 2, 3, 4].map((gameType) => ({
    game_type: gameType,
    rank_info: { rank_score: gameType }
  }))
})

const completedOverview = () => ({
  success: true,
  availability: { stats: true, reputation: true, member: true, favorite_venue_reward: true },
  stats: { success: true },
  reputation: { success: true },
  member_status: { success: true },
  favorite_venue_reward: { success: true }
})

const bootstrapActivity = (activity, identity) => activity.applyBootstrap(identity, {
  availability: {
    current_match: true,
    unread_count: true,
    pending_friend_request_count: true,
    latest_season_rollover: true
  }
})

const activityRequestMocks = () => ({
  getCurrentMatch: () => { recordRequest('GET', '/api/match/current'); return Promise.resolve({ success: true }) },
  getUnreadCount: () => { recordRequest('GET', '/api/notification/unread-count'); return Promise.resolve({ count: 0 }) },
  getFriendRequests: () => { recordRequest('GET', '/api/friend/requests'); return Promise.resolve({ total: 0 }) },
  getNotificationList: () => { recordRequest('GET', '/api/notification/list'); return Promise.resolve({ list: [] }) }
})

test.beforeEach(() => {
  globalThis.__CHASING_POINTS_REQUEST_METRICS__ = true
  resetRequestCounts()
})

test.after(() => {
  delete globalThis.__CHASING_POINTS_REQUEST_METRICS__
})

test('home request graph stays within four reads after Bootstrap hydrates activity', async () => {
  const useActivityStore = loadActivityStore(activityRequestMocks())
  const useUserOverviewStore = loadUserOverviewStore(() => {
    recordRequest('GET', '/api/user/overview')
    return Promise.resolve(completedOverview())
  })
  const usePublicReadStore = loadPublicReadStore({
    getLeaderboardSummary: () => {
      recordRequest('GET', '/api/public/rank/leaderboard-summary')
      return Promise.resolve({ success: true, top_three: [] })
    },
    getEventNewsList: () => {
      recordRequest('GET', '/api/event-news/list')
      return Promise.resolve({ success: true, list: [] })
    },
    getNearbyVenues: () => {
      recordRequest('GET', '/api/venue/nearby')
      return Promise.resolve({ success: true, list: [] })
    }
  })
  const identity = { userId: 7, authGeneration: 3 }
  const activity = useActivityStore()
  bootstrapActivity(activity, identity)

  await Promise.all([
    activity.fetch(identity),
    useUserOverviewStore().fetch(identity),
    usePublicReadStore().loadLeaderboardSummary({ game_type: 3 }, { identity }),
    usePublicReadStore().loadEventNews({ page: 1, page_size: 1 }),
    usePublicReadStore().loadNearbyVenues({ latitude: 22.5, longitude: 114.1, radius: 5000, limit: 3 })
  ])

  assert.deepEqual(getRequestCountSnapshot(), {
    'GET /api/user/overview': 1,
    'GET /api/public/rank/leaderboard-summary': 1,
    'GET /api/event-news/list': 1,
    'GET /api/venue/nearby': 1
  })
})

test('match and user first screens reuse Bootstrap activity within their request caps', async () => {
  const identity = { userId: 7, authGeneration: 3 }
  const useActivityStore = loadActivityStore(activityRequestMocks())
  const activity = useActivityStore()
  bootstrapActivity(activity, identity)

  recordRequest('GET', '/api/public/matches')
  await activity.fetch(identity)
  assert.deepEqual(getRequestCountSnapshot(), { 'GET /api/public/matches': 1 })

  resetRequestCounts()
  const useUserOverviewStore = loadUserOverviewStore(() => {
    recordRequest('GET', '/api/user/overview')
    return Promise.resolve(completedOverview())
  })
  const rankStore = createRankStore(() => {
    recordRequest('GET', '/api/rank/infos')
    return Promise.resolve(rankResponse())
  })
  await Promise.all([
    activity.fetch(identity),
    useUserOverviewStore().fetch(identity),
    rankStore.ensureFresh(identity.userId)
  ])
  assert.deepEqual(getRequestCountSnapshot(), {
    'GET /api/user/overview': 1,
    'GET /api/rank/infos': 1
  })
})

const appScript = () => readAppFile('App.vue')
  .match(/<script>([\s\S]*?)<\/script>/)[1]
  .replace(/^\s*import[\s\S]*?from\s+['"][^'"]+['"]\s*\n/gm, '')
  .replace('export default', 'return')

const createAppHarness = ({ getUserBootstrap, userStore }) => {
  const activityStore = {
    inFlight: null,
    reserveBootstrap(identity) {
      const ownerKey = `${identity.userId}:${identity.authGeneration}`
      if (this.inFlight?.ownerKey === ownerKey) return null
      const request = { ownerKey }
      this.inFlight = request
      return {
        matches: (candidate) => `${candidate.userId}:${candidate.authGeneration}` === ownerKey,
        apply: () => {
          if (activityStore.inFlight === request) activityStore.inFlight = null
        },
        release: () => {
          if (activityStore.inFlight === request) activityStore.inFlight = null
        }
      }
    },
    applyBootstrap: () => {}
  }
  const definition = new Function(
    'createSessionRecovery',
    'canApplySessionRecoveryResult',
    'getUserBootstrap',
    'useUserStore',
    'useActivityStore',
    'useUserDataInvalidationStore',
    appScript()
  )(
    createSessionRecovery,
    canApplySessionRecoveryResult,
    getUserBootstrap,
    () => userStore,
    () => activityStore,
    () => ({})
  )
  const instance = {
    appIsForeground: true,
    sessionRecoveryLifecycle: 1,
    bootstrapActivityReservation: null,
    validatedAuthGeneration: -1,
    ...definition.methods,
    applyBootstrapCompetitiveRevision: () => {},
    connectUserWS: () => {},
    uploadPushTokenIfValidated: () => {},
    checkOngoingMatchReminder: () => {}
  }
  return instance
}

test('cold start and each foreground generation issue one Bootstrap, while old account results are ignored', async () => {
  const pending = []
  const updates = []
  const userStore = {
    userId: 1,
    authGeneration: 1,
    isLoggedIn: true,
    updateUserInfo: (user) => updates.push(user)
  }
  const getUserBootstrap = () => {
    recordRequest('GET', '/api/user/bootstrap')
    return new Promise((resolve) => pending.push(resolve))
  }
  const app = createAppHarness({ getUserBootstrap, userStore })

  app.restoreUserSession()
  app.restoreUserSession()
  assert.deepEqual(getRequestCountSnapshot(), { 'GET /api/user/bootstrap': 1 })

  userStore.userId = 2
  userStore.authGeneration = 2
  app.sessionRecoveryLifecycle = 2
  app.restoreUserSession()
  assert.deepEqual(getRequestCountSnapshot(), { 'GET /api/user/bootstrap': 2 })

  pending[0]({ success: true, user_info: { id: 1 } })
  await flush()
  assert.deepEqual(updates, [])

  pending[1]({ success: true, user_info: { id: 2 } })
  await flush()
  assert.deepEqual(updates, [{ id: 2 }])

  app.sessionRecoveryLifecycle = 3
  app.restoreUserSession()
  assert.deepEqual(getRequestCountSnapshot(), { 'GET /api/user/bootstrap': 3 })
  pending[2]({ success: true, user_info: { id: 2 } })
  await flush()
})

test('core pages are wired only to their capped aggregate reads', () => {
  const homeSource = readAppFile('pages/index/index.vue')
  const matchSource = readAppFile('pages/match/index.vue')
  const rankingSource = readAppFile('pages/ranking/index.vue')
  const userSource = readAppFile('pages/user/index.vue')
  const statsSource = readAppFile('subPages/user/statsDetail.vue')
  const h2hSource = readAppFile('subPages/user/h2hRecord.vue')

  assert.match(homeSource, /activityStore\.fetch\(getActivityIdentity\(\)/)
  assert.match(homeSource, /userOverviewStore\.fetch\(getActivityIdentity\(\)/)
  assert.match(homeSource, /publicReadStore\.loadLeaderboardSummary\(/)
  assert.match(matchSource, /const publicMatchesRequest = getPublicMatches\(/)
  assert.match(matchSource, /activityStore\.fetch\(/)
  assert.match(rankingSource, /onShow\(\(\) => \{\s*fetchLeaderboard\(\)/)
  assert.match(rankingSource, /publicReadStore\.loadLeaderboard\(/)
  assert.match(userSource, /activityStore\.fetch\(identity, \{ force, silent: true \}\)/)
  assert.match(userSource, /userOverviewStore\.fetch\(identity, \{ force, silent: true \}\)/)
  assert.match(userSource, /rankStore\.ensureFresh\(identity\)/)
  assert.match(statsSource, /getStatsOverview\(/)
  assert.doesNotMatch(statsSource, /getStatsByGameType\(|getRecentTrend\(|getRankScoreTrend\(/)
  assert.match(h2hSource, /getH2HOverview\(/)
  assert.doesNotMatch(h2hSource, /getH2HStats\(/)
})
