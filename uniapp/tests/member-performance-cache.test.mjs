import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import { createPerformanceCacheActions, createPerformanceCacheState } from '../utils/member-performance-cache.js'

const identity = (userId, authGeneration = 1) => ({ userId, authGeneration })

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

// 用可控的请求函数构造缓存实例，记录每次请求的参数。
const createCache = (handler) => {
  const calls = []
  const cache = Object.assign(createPerformanceCacheState(), createPerformanceCacheActions({
    requestPerformance: (params) => {
      calls.push(params)
      return handler(params, calls.length)
    }
  }))
  return { cache, calls }
}

const ok = (marker, extra = {}) => ({ success: true, locked: false, sample_count: 10, marker, ...extra })

test('first entry fetches once and repeated onShow with the same version reuses the cache', async () => {
  const { cache, calls } = createCache(async () => ok('first'))
  const first = await cache.load(identity(1), 3, '0:0')
  const again = await cache.load(identity(1), 3, '0:0')
  assert.equal(first.marker, 'first')
  assert.equal(again, first)
  assert.deepEqual(calls, [{ game_type: 3 }])
  assert.equal(cache.peek(identity(1), 3), first)
})

test('concurrent reads of the same key share a single request', async () => {
  const gate = deferred()
  const { cache, calls } = createCache(() => gate.promise)
  const reads = [cache.load(identity(1), 3, 'v1'), cache.load(identity(1), 3, 'v1'), cache.load(identity(1), 3, 'v1')]
  assert.equal(calls.length, 1)
  gate.resolve(ok('shared'))
  const results = await Promise.all(reads)
  assert.ok(results.every((item) => item.marker === 'shared'))
})

test('each game type is cached independently', async () => {
  const { cache, calls } = createCache(async ({ game_type }) => ok(`type-${game_type}`))
  await cache.load(identity(1), 3, 'v')
  await cache.load(identity(1), 2, 'v')
  assert.deepEqual(calls.map((item) => item.game_type), [3, 2])
  assert.equal(cache.peek(identity(1), 3).marker, 'type-3')
  assert.equal(cache.peek(identity(1), 2).marker, 'type-2')
  // 不支持的球种按默认球种读取，不会以任意值发请求。
  await cache.load(identity(1), 99, 'next-version')
  assert.equal(calls.at(-1).game_type, 3)
})

test('a changed invalidation version refetches, and the older in-flight response never overwrites the newer data', async () => {
  const gates = [deferred(), deferred()]
  const { cache } = createCache((_params, index) => gates[index - 1].promise)

  const older = cache.load(identity(1), 3, 'v1')
  const newer = cache.load(identity(1), 3, 'v2')
  // 新版本的响应先到，旧版本的迟到响应后到。
  gates[1].resolve(ok('v2'))
  assert.equal((await newer).marker, 'v2')
  gates[0].resolve(ok('v1'))
  const olderResult = await older
  assert.equal(olderResult.marker, 'v2', 'the superseded caller is handed the newest result')
  assert.equal(cache.peek(identity(1), 3).marker, 'v2')
})

test('switching accounts clears the cache and ignores a late response from the previous identity', async () => {
  const gates = [deferred(), deferred()]
  const { cache } = createCache((_params, index) => gates[index - 1].promise)

  const oldUser = cache.load(identity(1, 1), 3, 'v')
  const newUser = cache.load(identity(2, 2), 3, 'v')
  gates[1].resolve(ok('user-2'))
  assert.equal((await newUser).marker, 'user-2')
  gates[0].resolve(ok('user-1'))
  assert.equal(await oldUser, null, 'the previous identity must not receive or write data')

  assert.equal(cache.peek(identity(2, 2), 3).marker, 'user-2')
  assert.equal(cache.peek(identity(1, 1), 3), null)
  assert.equal(cache.ownerUserId, 2)
})

test('the same user after a new login (auth generation changed) starts from an empty cache', async () => {
  const { cache, calls } = createCache(async () => ok('x'))
  await cache.load(identity(1, 1), 3, 'v')
  assert.equal(cache.peek(identity(1, 2), 3), null)
  await cache.load(identity(1, 2), 3, 'v')
  assert.equal(calls.length, 2)
})

test('clear (logout) drops cached data and ignores a response that arrives afterwards', async () => {
  const gate = deferred()
  const { cache } = createCache(() => gate.promise)
  const pending = cache.load(identity(1), 3, 'v')
  cache.clear()
  gate.resolve(ok('late'))
  assert.equal(await pending, null)
  assert.equal(cache.peek(identity(1), 3), null)
  assert.deepEqual(cache.entries, {})
})

test('a failed or unsuccessful read throws, keeps older data and retries on the next read', async () => {
  let attempt = 0
  const { cache, calls } = createCache(async () => {
    attempt += 1
    if (attempt === 1) return ok('before')
    if (attempt === 2) throw new Error('network down')
    if (attempt === 3) return { success: false, message: '暂时不可用' }
    return ok('after')
  })
  await cache.load(identity(1), 3, 'v1')

  await assert.rejects(() => cache.load(identity(1), 3, 'v2'), /network down/)
  assert.equal(cache.peek(identity(1), 3).marker, 'before', 'stale data stays readable for a refresh-failed banner')

  await assert.rejects(() => cache.load(identity(1), 3, 'v2'), /暂时不可用/)
  assert.equal(cache.peek(identity(1), 3).marker, 'before')

  assert.equal((await cache.load(identity(1), 3, 'v2')).marker, 'after')
  assert.equal(calls.length, 4)
})

test('a guest identity never reads', async () => {
  const { cache, calls } = createCache(async () => ok('x'))
  assert.equal(await cache.load(identity(0, 1), 3, 'v'), null)
  assert.equal(calls.length, 0)
})

test('the store, facade and logout cleanup are wired to the cache', () => {
  const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')
  assert.match(read('store/memberPerformance.js'), /requestPerformance: getMemberPerformance/)
  assert.match(read('api/member.js'), /get\('\/api\/member\/performance', params, options\)/)
  assert.match(read('api/member.js'), /get\('\/api\/member\/match-replay', params, options\)/)
  assert.match(read('store/user.js'), /memberPerformanceStore: useMemberPerformanceStore\(\)/)
  assert.match(read('utils/session-storage.js'), /memberPerformanceStore\?\.clear\(\)/)
})
