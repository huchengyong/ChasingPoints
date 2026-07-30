import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import {
  createRankCacheActions,
  createRankCacheState,
  normalizeUserRankInfos,
  shouldInvalidateRankAfterSettlement
} from '../utils/rank-cache.js'

const createResponse = (score = 0) => ({
  success: true,
  rank_infos: [1, 2, 3, 4].map((gameType) => ({
    game_type: gameType,
    rank_info: { rank_score: score + gameType }
  }))
})

const createDeferred = () => {
  let resolve
  let reject
  const promise = new Promise((onResolve, onReject) => {
    resolve = onResolve
    reject = onReject
  })
  return { promise, resolve, reject }
}

const flush = () => new Promise((resolve) => setImmediate(resolve))

const createCache = (requestRankInfos) => {
  const cache = createRankCacheState()
  const actions = createRankCacheActions({ requestRankInfos, normalizeRankInfos: normalizeUserRankInfos })
  for (const [name, action] of Object.entries(actions)) {
    cache[name] = action.bind(cache)
  }
  return cache
}

test('valid rank cache reuses the complete aggregate response without another request', async () => {
  let calls = 0
  const cache = createCache(async () => {
    calls += 1
    return createResponse(10)
  })

  const first = await cache.ensureFresh(1001)
  const second = await cache.ensureFresh(1001)

  assert.equal(calls, 1)
  assert.equal(first[3].rank_score, 13)
  assert.equal(second[3].rank_score, 13)
  assert.equal(cache.dirty, false)
})

test('an invalidation during a request waits for a follow-up response instead of applying stale data', async () => {
  const firstRequest = createDeferred()
  const secondRequest = createDeferred()
  let calls = 0
  const cache = createCache(() => {
    calls += 1
    return calls === 1 ? firstRequest.promise : secondRequest.promise
  })

  const pending = cache.ensureFresh(1002)
  cache.invalidate(1002)
  firstRequest.resolve(createResponse(10))
  await flush()

  assert.equal(calls, 2)
  secondRequest.resolve(createResponse(20))
  const result = await pending

  assert.equal(result[3].rank_score, 23)
  assert.equal(cache.rankInfoMap[3].rank_score, 23)
  assert.equal(cache.dirty, false)
})

test('switching users drops an in-flight response from the previous owner', async () => {
  const firstRequest = createDeferred()
  const cache = createCache(() => firstRequest.promise)

  const pending = cache.ensureFresh(1003)
  cache.ensureOwner(1004)
  firstRequest.resolve(createResponse(10))

  assert.equal(await pending, null)
  assert.equal(cache.ownerUserId, 1004)
  assert.deepEqual(cache.rankInfoMap, {})
  assert.equal(cache.loaded, false)
})

test('a failed refresh keeps the last successful result dirty for a later retry', async () => {
  let calls = 0
  const cache = createCache(async () => {
    calls += 1
    if (calls === 1) return createResponse(10)
    throw new Error('network unavailable')
  })

  await cache.ensureFresh(1005)
  await assert.rejects(cache.forceRefresh(1005), /network unavailable/)

  assert.equal(cache.rankInfoMap[3].rank_score, 13)
  assert.equal(cache.loaded, true)
  assert.equal(cache.dirty, true)
})

test('incomplete aggregate responses are rejected instead of becoming a valid cache', async () => {
  const cache = createCache(async () => ({
    success: true,
    rank_infos: [{ game_type: 3, rank_info: { rank_score: 20 } }]
  }))

  await assert.rejects(cache.ensureFresh(1006), /段位信息不完整/)
  assert.equal(cache.loaded, false)
  assert.equal(cache.dirty, false)
})

test('only decisive ranked participant settlements invalidate rank data', () => {
  assert.equal(shouldInvalidateRankAfterSettlement({
    isParticipant: true,
    matchMode: 'ranked',
    status: 2,
    result: 1
  }), true)
  assert.equal(shouldInvalidateRankAfterSettlement({
    isParticipant: true,
    matchMode: 'ranked',
    status: 2,
    result: 3
  }), false)
  assert.equal(shouldInvalidateRankAfterSettlement({
    isParticipant: true,
    matchMode: 'practice',
    status: 2,
    result: 1
  }), false)
  assert.equal(shouldInvalidateRankAfterSettlement({
    isParticipant: false,
    matchMode: 'ranked',
    status: 2,
    result: 1
  }), false)
})

test('user page keeps cached rank visible, enables manual refresh, and invalidates all completion paths', () => {
  const userPageSource = readFileSync(new URL('../pages/user/index.vue', import.meta.url), 'utf8')
  const playingPageSource = readFileSync(new URL('../subPages/match/playing.vue', import.meta.url), 'utf8')
  const pagesSource = readFileSync(new URL('../pages.json', import.meta.url), 'utf8')
  const appSource = readFileSync(new URL('../App.vue', import.meta.url), 'utf8')

  assert.match(userPageSource, /!hasRankCache && \(coreDataLoading \|\| rankLoading\)/)
  assert.match(userPageSource, /rankStore\.forceRefresh\(userStore\.userId\)/)
  assert.match(userPageSource, /data\?\.category === 'match_result'/)
  assert.match(pagesSource, /"path": "pages\/user\/index"[\s\S]*"enablePullDownRefresh": true/)
  assert.match(playingPageSource, /const handleSync[\s\S]*invalidateRankAfterSettlement\(data\)/)
  assert.match(playingPageSource, /const handleFinishStateUpdate[\s\S]*invalidateRankAfterSettlement\(data\)/)
  assert.match(playingPageSource, /const handleMatchEnd[\s\S]*invalidateRankAfterSettlement\(data\)/)
  assert.match(appSource, /WS_MESSAGE_TYPES\.RANK_INFO_UPDATED/)
  assert.match(appSource, /userWS\.on\(WS_MESSAGE_TYPES\.RANK_INFO_UPDATED, this\.handleRankInfoUpdated\)/)
  assert.match(appSource, /userWS\.connect\(\)/)
  assert.doesNotMatch(userPageSource, /userWS\.disconnect\(\)/)
})
