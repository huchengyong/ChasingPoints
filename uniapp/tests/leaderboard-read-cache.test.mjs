import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appRoot = new URL('..', import.meta.url)
const readAppFile = (path) => readFileSync(new URL(path, appRoot), 'utf8')

const loadPublicReadStore = (mocks) => {
  const source = readAppFile('store/publicRead.js')
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
    'getEventNewsList',
    'getEventNewsView',
    'getLeaderboard',
    'getLeaderboardSummary',
    'getNearbyVenues',
    'getPublicMatches',
    'buildNearbyVenueCacheKey',
    'buildPublicReadKey',
    `${source}; return usePublicReadStore`
  )(
    defineStore,
    mocks.getEventNewsList || (() => Promise.resolve({ success: true })),
    mocks.getEventNewsView || (() => Promise.resolve({ success: true })),
    mocks.getLeaderboard,
    mocks.getLeaderboardSummary,
    mocks.getNearbyVenues || (() => Promise.resolve({ success: true })),
    mocks.getPublicMatches || (() => Promise.resolve({ success: true, list: [] })),
    mocks.buildNearbyVenueCacheKey || (() => 'nearby'),
    mocks.buildPublicReadKey || ((prefix, params) => `${prefix}:${JSON.stringify(params)}`)
  )
}

const deferred = () => {
  let resolve
  const promise = new Promise((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

test('leaderboard cache single-flights same viewer and retains that viewer ranking', async () => {
  const pending = deferred()
  let calls = 0
  const usePublicReadStore = loadPublicReadStore({
    getLeaderboard: () => {
      calls += 1
      return pending.promise
    },
    getLeaderboardSummary: () => Promise.resolve({ success: true })
  })
  const store = usePublicReadStore()
  const options = { identity: { userId: 7, authGeneration: 3 } }

  const first = store.loadLeaderboard({ game_type: 3, page: 1, page_size: 20 }, options)
  const second = store.loadLeaderboard({ game_type: 3, page: 1, page_size: 20 }, options)
  await Promise.resolve()
  assert.equal(calls, 1)

  pending.resolve({
    success: true,
    top_three: [{ user_id: 1 }],
    my_ranking: { user_id: 7, rank: 4 },
    list: [{ user_id: 8 }],
    total: 8
  })
  const [firstResult, secondResult] = await Promise.all([first, second])
  assert.equal(firstResult.my_ranking.user_id, 7)
  assert.deepEqual(secondResult, firstResult)
})

test('leaderboard cache never shares personalized rankings across auth identities', async () => {
  let calls = 0
  const usePublicReadStore = loadPublicReadStore({
    getLeaderboard: async () => {
      calls += 1
      return { success: true, my_ranking: { user_id: calls }, top_three: [], list: [], total: 0 }
    },
    getLeaderboardSummary: () => Promise.resolve({ success: true })
  })
  const store = usePublicReadStore()
  const params = { game_type: 3, page: 1, page_size: 20 }

  const firstOwner = await store.loadLeaderboard(params, { identity: { userId: 1, authGeneration: 4 } })
  const secondOwner = await store.loadLeaderboard(params, { identity: { userId: 2, authGeneration: 5 } })
  const firstOwnerAgain = await store.loadLeaderboard(params, { identity: { userId: 1, authGeneration: 4 } })

  assert.equal(calls, 2)
  assert.equal(firstOwner.my_ranking.user_id, 1)
  assert.equal(secondOwner.my_ranking.user_id, 2)
  assert.equal(firstOwnerAgain.my_ranking.user_id, 1)
})

test('leaderboard summary has its own identity-safe single-flight cache key', async () => {
  let calls = 0
  const usePublicReadStore = loadPublicReadStore({
    getLeaderboard: () => Promise.resolve({ success: true }),
    getLeaderboardSummary: async () => {
      calls += 1
      return { success: true, top_three: [{ user_id: 1 }], my_ranking: { user_id: 7 }, version: 'v1' }
    }
  })
  const store = usePublicReadStore()
  const options = { identity: { userId: 7, authGeneration: 3 } }

  const [first, second] = await Promise.all([
    store.loadLeaderboardSummary({ game_type: 3 }, options),
    store.loadLeaderboardSummary({ game_type: 3 }, options)
  ])

  assert.equal(calls, 1)
  assert.equal(first.version, 'v1')
  assert.deepEqual(second, first)
})

test('event news detail cache single-flights by event id', async () => {
  const pending = deferred()
  let calls = 0
  const usePublicReadStore = loadPublicReadStore({
    getLeaderboard: () => Promise.resolve({ success: true }),
    getLeaderboardSummary: () => Promise.resolve({ success: true }),
    getEventNewsView: () => {
      calls += 1
      return pending.promise
    }
  })
  const store = usePublicReadStore()
  const first = store.loadEventNewsView({ event_id: 42 })
  const second = store.loadEventNewsView({ event_id: 42 })
  await Promise.resolve()
  assert.equal(calls, 1)

  pending.resolve({ success: true, event_news: { id: 42 } })
  assert.deepEqual(await first, await second)
})

test('home and ranking pages use the cached aggregate leaderboard reads', () => {
  const homeSource = readAppFile('pages/index/index.vue')
  const rankingSource = readAppFile('pages/ranking/index.vue')
  const eventListSource = readAppFile('pages/tournament/index.vue')
  const eventDetailSource = readAppFile('subPages/tournament/detail.vue')

  assert.match(homeSource, /const homeLoading = computed\(\(\) => homeState\.value\.status === ASYNC_PAGE_STATUS\.LOADING\)/)
  assert.match(homeSource, /publicReadStore\.loadLeaderboardSummary\(/)
  assert.doesNotMatch(homeSource, /getLeaderboard\(/)
  assert.match(rankingSource, /const publicReadStore = usePublicReadStore\(\)/)
  assert.match(rankingSource, /publicReadStore\.loadLeaderboard\(/)
  assert.doesNotMatch(rankingSource, /getLeaderboard\(/)
  assert.match(eventListSource, /publicReadStore\.loadEventNews\(/)
  assert.doesNotMatch(eventListSource, /getEventNewsList\(/)
  assert.match(eventDetailSource, /publicReadStore\.loadEventNewsView\(/)
  assert.doesNotMatch(eventDetailSource, /getEventNewsView\(/)
})

const withFakeNow = async (run) => {
  const realNow = Date.now
  let now = 1_000_000
  Date.now = () => now
  try {
    return await run({ advance: (ms) => { now += ms } })
  } finally {
    Date.now = realNow
  }
}

test('live match preview asks for the first small hall page and keeps it for 30 seconds', async () => {
  await withFakeNow(async ({ advance }) => {
    const requests = []
    const usePublicReadStore = loadPublicReadStore({
      getPublicMatches: async (params) => {
        requests.push(params)
        return { success: true, list: [{ id: requests.length }], total: 1 }
      }
    })
    const store = usePublicReadStore()

    const first = await store.loadLiveMatchPreview()
    advance(29_000)
    const cached = await store.loadLiveMatchPreview()
    assert.equal(requests.length, 1, 'inside the TTL the preview is served from cache')
    assert.deepEqual(cached, first)
    assert.deepEqual(requests[0], { scope: 'hall', status: 1, page: 1, page_size: 4 })

    advance(2_000)
    const refreshed = await store.loadLiveMatchPreview()
    assert.equal(requests.length, 2, 'after the TTL it reloads')
    assert.equal(refreshed.list[0].id, 2)
  })
})

test('live match preview can be forced to reload and single-flights concurrent readers', async () => {
  const pending = deferred()
  let calls = 0
  const usePublicReadStore = loadPublicReadStore({
    getPublicMatches: () => {
      calls += 1
      return pending.promise
    }
  })
  const store = usePublicReadStore()

  const first = store.loadLiveMatchPreview()
  const second = store.loadLiveMatchPreview({ force: true })
  await Promise.resolve()
  assert.equal(calls, 1)

  pending.resolve({ success: true, list: [{ id: 1 }] })
  assert.deepEqual(await first, await second)

  await store.loadLiveMatchPreview({ force: true })
  assert.equal(calls, 2)
})

test('live match preview does not cache a failure and keeps the last good list when a refresh fails', async () => {
  await withFakeNow(async ({ advance }) => {
    let mode = 'fail'
    const usePublicReadStore = loadPublicReadStore({
      getPublicMatches: async () => {
        if (mode === 'fail') throw new Error('offline')
        return { success: true, list: [{ id: 9 }] }
      }
    })
    const store = usePublicReadStore()

    await assert.rejects(store.loadLiveMatchPreview(), /offline/)
    mode = 'ok'
    const loaded = await store.loadLiveMatchPreview()
    assert.equal(loaded.list[0].id, 9, 'a failure leaves nothing behind that blocks the next read')

    mode = 'fail'
    advance(31_000)
    const stale = await store.loadLiveMatchPreview()
    assert.equal(stale.list[0].id, 9, 'a failed refresh falls back to the previous list')
  })
})

test('the live match preview holds no viewer data', () => {
  const source = readAppFile('store/publicRead.js')
  assert.match(source, /liveMatches: \{\}/)
  assert.match(source, /scope: 'hall'/)
  assert.doesNotMatch(source, /liveMatches[\s\S]{0,200}identity/)
})
