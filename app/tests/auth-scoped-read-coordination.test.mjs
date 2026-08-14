import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appRoot = new URL('..', import.meta.url)
const readAppFile = (path) => readFileSync(new URL(path, appRoot), 'utf8')

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

const loadActivityStore = (mocks) => {
  const source = readAppFile('store/activity.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')

  const defineStore = (_name, definition) => () => {
    const state = definition.state()
    Object.entries(definition.actions).forEach(([name, action]) => {
      state[name] = action.bind(state)
    })
    return state
  }

  return new Function(
    'defineStore',
    'getCurrentMatch',
    'getFriendRequests',
    'getNotificationList',
    'getUnreadCount',
    'useFriendRequestStore',
    'useNotificationStore',
    `${source}; return useActivityStore`
  )(
    defineStore,
    mocks.getCurrentMatch,
    mocks.getFriendRequests,
    mocks.getNotificationList,
    mocks.getUnreadCount,
    () => ({ setPendingCount: () => {} }),
    () => ({ setUnreadCount: () => {} })
  )
}

const loadNotificationStore = (mocks) => {
  const source = readAppFile('store/notification.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')
  const defineStore = (_name, setup) => () => setup()
  return new Function('defineStore', 'ref', 'getUnreadCount', 'shouldFetchUnreadCount', `${source}; return useNotificationStore`)(
    defineStore,
    (value) => ({ value }),
    mocks.getUnreadCount,
    () => true
  )
}

const loadFriendRequestStore = (mocks) => {
  const source = readAppFile('store/friendRequest.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')
  const defineStore = (_name, setup) => () => setup()
  return new Function('defineStore', 'ref', 'getFriendRequests', 'shouldFetchAuthState', `${source}; return useFriendRequestStore`)(
    defineStore,
    (value) => ({ value }),
    mocks.getFriendRequests,
    () => true
  )
}

const loadUserOverviewStore = (mocks) => {
  const source = readAppFile('store/userOverview.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')

  const defineStore = (_name, definition) => () => {
    const state = definition.state()
    Object.entries(definition.actions).forEach(([name, action]) => {
      state[name] = action.bind(state)
    })
    return state
  }

  return new Function('defineStore', 'getUserOverview', `${source}; return useUserOverviewStore`)(
    defineStore,
    mocks.getUserOverview
  )
}

const loadUserDataInvalidationStore = () => {
  const source = readAppFile('store/userDataInvalidation.js')
    .replace(/^import .*$/gm, '')
    .replaceAll('export const ', 'const ')

  const defineStore = (_name, definition) => () => {
    const state = definition.state()
    Object.entries(definition.actions).forEach(([name, action]) => {
      state[name] = action.bind(state)
    })
    return state
  }

  return new Function('defineStore', `${source}; return useUserDataInvalidationStore`)(defineStore)
}

const successfulActivityResponse = (id) => ({
  getCurrentMatch: () => Promise.resolve({ success: true, match: { id } }),
  getUnreadCount: () => Promise.resolve({ count: id }),
  getFriendRequests: () => Promise.resolve({ total: id }),
  getNotificationList: () => Promise.resolve({ list: [] })
})

test('notification and friend request stores ignore old auth-generation responses', async () => {
  const originalUni = globalThis.uni
  globalThis.uni = { getStorageSync: () => 'token' }
  try {
    const unread = deferred()
    const pending = deferred()
    const notificationStore = loadNotificationStore({ getUnreadCount: () => unread.promise })()
    const friendRequestStore = loadFriendRequestStore({ getFriendRequests: () => pending.promise })()
    const oldIdentity = { userId: 7, authGeneration: 3 }
    const newIdentity = { userId: 8, authGeneration: 4 }

    const oldUnreadRequest = notificationStore.fetchUnreadCount(oldIdentity)
    const oldPendingRequest = friendRequestStore.fetchPendingCount(oldIdentity)
    notificationStore.setUnreadCount(9, newIdentity)
    friendRequestStore.setPendingCount(6, newIdentity)
    unread.resolve({ count: 1 })
    pending.resolve({ total: 1 })
    await Promise.all([oldUnreadRequest, oldPendingRequest])

    assert.equal(notificationStore.ownerUserId.value, 8)
    assert.equal(notificationStore.authGeneration.value, 4)
    assert.equal(notificationStore.unreadCount.value, 9)
    assert.equal(friendRequestStore.ownerUserId.value, 8)
    assert.equal(friendRequestStore.authGeneration.value, 4)
    assert.equal(friendRequestStore.pendingCount.value, 6)
  } finally {
    globalThis.uni = originalUni
  }
})

test('activity Store coalesces concurrent reads into one request batch', async () => {
  const requests = []
  const pending = deferred()
  const useActivityStore = loadActivityStore({
    getCurrentMatch: () => { requests.push('match'); return pending.promise },
    getUnreadCount: () => { requests.push('unread'); return pending.promise },
    getFriendRequests: () => { requests.push('friend'); return pending.promise },
    getNotificationList: () => { requests.push('season'); return pending.promise }
  })
  const store = useActivityStore()

  const first = store.fetch({ userId: 7, authGeneration: 3 })
  const second = store.fetch({ userId: 7, authGeneration: 3 })
  assert.deepEqual(requests, ['match', 'unread', 'friend', 'season'])

  pending.resolve({ success: true, match: { id: 7 }, count: 7, total: 7, list: [] })
  await Promise.all([first, second])
  assert.equal(store.currentMatch.id, 7)
})

test('activity Store applies Bootstrap without scheduling legacy activity requests', () => {
  const calls = []
  const useActivityStore = loadActivityStore({
    getCurrentMatch: () => { calls.push('match'); return Promise.resolve({}) },
    getUnreadCount: () => { calls.push('unread'); return Promise.resolve({}) },
    getFriendRequests: () => { calls.push('friend'); return Promise.resolve({}) },
    getNotificationList: () => { calls.push('season'); return Promise.resolve({}) }
  })
  const store = useActivityStore()
  store.applyBootstrap({ userId: 7, authGeneration: 3 }, {
    current_match: { id: 99 },
    unread_count: 4,
    pending_friend_request_count: 2,
    latest_season_rollover: { id: 8 },
    availability: {
      current_match: true,
      unread_count: true,
      pending_friend_request_count: true,
      latest_season_rollover: true
    }
  })

  assert.deepEqual(calls, [])
  assert.equal(store.currentMatch.id, 99)
  assert.equal(store.unreadCount, 4)
  assert.equal(store.pendingFriendRequestCount, 2)
  assert.equal(store.latestSeasonRollover.id, 8)
  assert.equal(store.loaded, true)
  assert.equal(store.dirty, false)
})

test('activity Store releases a reserved Bootstrap waiter when the identity is cleared', async () => {
  const useActivityStore = loadActivityStore(successfulActivityResponse(7))
  const store = useActivityStore()
  const reservation = store.reserveBootstrap({ userId: 7, authGeneration: 3 })
  assert.ok(reservation)
  const waiter = store.inFlight
  store.clear()
  const result = await Promise.race([
    waiter,
    new Promise((_, reject) => setTimeout(() => reject(new Error('reserved Bootstrap waiter was not released')), 50))
  ])
  assert.deepEqual(result, store.snapshot())
  assert.equal(store.inFlight, null)
})

test('activity Store prevents an old account in-flight result from writing into the new account', async () => {
  const oldRequests = [deferred(), deferred(), deferred(), deferred()]
  const newRequests = [deferred(), deferred(), deferred(), deferred()]
  let requestIndex = 0
  const nextRequest = () => (requestIndex < 4 ? oldRequests[requestIndex++] : newRequests[requestIndex++ - 4]).promise
  const useActivityStore = loadActivityStore({
    getCurrentMatch: nextRequest,
    getUnreadCount: nextRequest,
    getFriendRequests: nextRequest,
    getNotificationList: nextRequest
  })
  const store = useActivityStore()

  const oldFetch = store.fetch({ userId: 1, authGeneration: 10 })
  const newFetch = store.fetch({ userId: 2, authGeneration: 11 })
  oldRequests[0].resolve({ success: true, match: { id: 1 } })
  oldRequests[1].resolve({ count: 1 })
  oldRequests[2].resolve({ total: 1 })
  oldRequests[3].resolve({ list: [] })
  await oldFetch

  assert.equal(store.ownerUserId, 2)
  assert.equal(store.currentMatch, null)

  newRequests[0].resolve({ success: true, match: { id: 2 } })
  newRequests[1].resolve({ count: 2 })
  newRequests[2].resolve({ total: 2 })
  newRequests[3].resolve({ list: [] })
  await newFetch
  assert.equal(store.currentMatch.id, 2)
})

test('activity Store keeps stale data when a refresh fails', async () => {
  let shouldFail = false
  const successful = successfulActivityResponse(9)
  const useActivityStore = loadActivityStore({
    getCurrentMatch: () => shouldFail ? Promise.reject(new Error('offline')) : successful.getCurrentMatch(),
    getUnreadCount: () => shouldFail ? Promise.reject(new Error('offline')) : successful.getUnreadCount(),
    getFriendRequests: () => shouldFail ? Promise.reject(new Error('offline')) : successful.getFriendRequests(),
    getNotificationList: () => shouldFail ? Promise.reject(new Error('offline')) : successful.getNotificationList()
  })
  const store = useActivityStore()

  await store.fetch({ userId: 9, authGeneration: 1 })
  store.markDirty()
  shouldFail = true
  await store.fetch({ userId: 9, authGeneration: 1 })

  assert.equal(store.currentMatch.id, 9)
  assert.equal(store.unreadCount, 9)
  assert.equal(store.pendingFriendRequestCount, 9)
  assert.equal(store.dirty, true)
})

test('user overview Store maps the aggregate contract scopes and response keys', async () => {
  const responses = [
    {
      success: true,
      stats: { success: true, total_matches: 4 },
      reputation: { success: true, reputation_score: 100 },
      member_status: { success: true, status: 'active' },
      favorite_venue_reward: { success: true, venue_id: 9 },
      availability: {
        stats: true,
        reputation: true,
        member: true,
        favorite_venue_reward: true
      }
    },
    {
      success: true,
      stats: { success: true, total_matches: 5 },
      reputation: { success: true, reputation_score: 99 },
      favorite_venue_reward: { success: true, venue_id: 10 },
      availability: {
        stats: true,
        reputation: true,
        member: false,
        favorite_venue_reward: true
      }
    }
  ]
  const useUserOverviewStore = loadUserOverviewStore({
    getUserOverview: () => Promise.resolve(responses.shift())
  })
  const store = useUserOverviewStore()
  const identity = { userId: 7, authGeneration: 3 }

  await store.fetch(identity)
  assert.equal(store.memberStatus.status, 'active')
  assert.equal(store.favoriteVenueRewardStatus.venue_id, 9)
  assert.equal(store.availability.memberStatus, true)
  assert.equal(store.availability.favoriteVenueRewardStatus, true)

  await store.fetch(identity, { force: true })
  assert.equal(store.memberStatus.status, 'active')
  assert.equal(store.favoriteVenueRewardStatus.venue_id, 10)
  assert.equal(store.availability.memberStatus, false)
  assert.equal(store.dirty, true)
})

test('user data invalidation scopes stay identity-bound and retain the newest competitive revision', () => {
  const useUserDataInvalidationStore = loadUserDataInvalidationStore()
  const store = useUserDataInvalidationStore()

  store.invalidate({ userId: 7, authGeneration: 3 }, ['rank', 'stats', 'h2h'], 8)
  assert.equal(store.versionOf('rank'), 1)
  assert.equal(store.versionOf('stats'), 1)
  assert.equal(store.versionOf('h2h'), 1)
  assert.equal(store.competitiveRevision, 8)

  store.invalidate({ userId: 7, authGeneration: 3 }, ['rank'], 6)
  assert.equal(store.versionOf('rank'), 2)
  assert.equal(store.competitiveRevision, 8)

  store.invalidate({ userId: 8, authGeneration: 4 }, ['season'], 2)
  assert.equal(store.ownerUserId, 8)
  assert.equal(store.versionOf('rank'), 0)
  assert.equal(store.versionOf('season'), 1)
})

test('user websocket updates exact counters and marks only declared scopes dirty', () => {
  const appSource = readAppFile('App.vue')
  const userPageSource = readAppFile('pages/user/index.vue')
  const websocketSource = readAppFile('utils/websocket.js')

  assert.match(websocketSource, /USER_DATA_UPDATED: 'user_data_updated'/)
  assert.match(appSource, /userWS\.on\(WS_MESSAGE_TYPES\.USER_DATA_UPDATED, this\.handleUserDataUpdated\)/)
  assert.match(appSource, /useUserDataInvalidationStore\(\)\.invalidate\(identity, scopes, data\.competitive_revision\)/)
  assert.match(appSource, /useActivityStore\(\)\.setUnreadCount\(data\.unread_count, identity\)/)
  assert.match(appSource, /useActivityStore\(\)\.setPendingFriendRequestCount\(data\.pending_friend_request_count, identity\)/)
  assert.doesNotMatch(userPageSource, /notificationStore\.fetchUnreadCount\(\)|friendRequestStore\.fetchPendingCount\(\)/)
})

test('client pages keep auth-scoped stores and avoid first-load or tab-switch duplicate requests', () => {
  const activitySource = readAppFile('store/activity.js')
  const overviewSource = readAppFile('store/userOverview.js')
  const rankingSource = readAppFile('pages/ranking/index.vue')
  const challengeSource = readAppFile('subPages/social/challenges.vue')

  assert.match(activitySource, /ownerUserId: 0/)
  assert.match(activitySource, /authGeneration: -1/)
  assert.match(activitySource, /if \(this\.inFlight\) \{\s*return this\.inFlight/)
  assert.match(overviewSource, /ownerUserId: 0/)
  assert.match(overviewSource, /authGeneration: -1/)
  assert.match(overviewSource, /getUserOverview\(\{ silent \}\)/)
  assert.match(overviewSource, /apply\('member', 'memberStatus', 'member_status'\)/)
  assert.match(overviewSource, /apply\('favorite_venue_reward', 'favoriteVenueRewardStatus', 'favorite_venue_reward'\)/)
  assert.doesNotMatch(overviewSource, /getUserStats\(|getUserReputation\(|getMemberStatus\(|getFavoriteVenueRewardStatus\(/)
  assert.doesNotMatch(rankingSource, /onMounted\(/)
  assert.match(rankingSource, /onShow\(\(\) => \{\s*fetchLeaderboard\(\)/)
  assert.match(rankingSource, /publicReadStore\.loadLeaderboard\(/)
  assert.match(challengeSource, /const allChallenges = ref\(\[\]\)/)
  assert.match(challengeSource, /const switchTab = \(newTab\) => \{\s*tab\.value = newTab\s*\}/)
})
