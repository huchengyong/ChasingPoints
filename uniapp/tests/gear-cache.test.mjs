import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import { clearUserScopedRuntimeState } from '../utils/session-storage.js'
import { GEAR_CACHE_TTL_MS, createGearCacheActions, createGearCacheState } from '../utils/gear-cache.js'

const identity = (userId, authGeneration = 1) => ({ userId, authGeneration })

const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

// 用可控的请求函数与时钟构造缓存实例。
const createCache = (handler) => {
  const clock = { time: 1_000_000 }
  const calls = []
  const cache = Object.assign(createGearCacheState(), createGearCacheActions({
    requestMine: () => {
      calls.push(clock.time)
      return handler(calls.length)
    },
    now: () => clock.time
  }))
  return { cache, calls, clock }
}

const ok = (marker) => ({ success: true, items: [], defaults: [], marker })

test('first entry fetches once and repeated onShow inside the TTL reuses the cache', async () => {
  const { cache, calls, clock } = createCache(async () => ok('first'))
  const first = await cache.load(identity(1))
  clock.time += GEAR_CACHE_TTL_MS - 1
  const again = await cache.load(identity(1))
  assert.equal(first.marker, 'first')
  assert.equal(again, first)
  assert.equal(calls.length, 1)
  assert.equal(cache.peek(identity(1)), first)
})

test('an expired entry is fetched again', async () => {
  const { cache, calls, clock } = createCache(async (n) => ok(`call-${n}`))
  await cache.load(identity(1))
  clock.time += GEAR_CACHE_TTL_MS
  const refreshed = await cache.load(identity(1))
  assert.equal(refreshed.marker, 'call-2')
  assert.equal(calls.length, 2)
})

test('force bypasses the TTL', async () => {
  const { cache, calls } = createCache(async (n) => ok(`call-${n}`))
  await cache.load(identity(1))
  const forced = await cache.load(identity(1), { force: true })
  assert.equal(forced.marker, 'call-2')
  assert.equal(calls.length, 2)
})

test('concurrent reads share a single request', async () => {
  const gate = deferred()
  const { cache, calls } = createCache(() => gate.promise)
  const reads = [cache.load(identity(1)), cache.load(identity(1)), cache.load(identity(1))]
  assert.equal(calls.length, 1)
  gate.resolve(ok('shared'))
  const results = await Promise.all(reads)
  assert.ok(results.every((item) => item.marker === 'shared'))
})

test('invalidate makes the next read go back to the server', async () => {
  const { cache, calls } = createCache(async (n) => ok(`call-${n}`))
  await cache.load(identity(1))
  cache.invalidate()
  const next = await cache.load(identity(1))
  assert.equal(next.marker, 'call-2')
  assert.equal(calls.length, 2)
})

test('a write that lands while a read is in flight is not hidden by the older response', async () => {
  const gates = [deferred(), deferred()]
  const { cache, calls } = createCache((n) => gates[n - 1].promise)
  const stale = cache.load(identity(1))
  // 请求进行中发生了写操作：失效后立刻再读，必须发起新请求而不是复用旧请求
  cache.invalidate()
  const fresh = cache.load(identity(1))
  assert.equal(calls.length, 2)

  gates[1].resolve(ok('after-write'))
  assert.equal((await fresh).marker, 'after-write')
  gates[0].resolve(ok('before-write'))
  const staleResult = await stale
  assert.equal(staleResult.marker, 'after-write', 'the older response must hand back the newest result')
  assert.equal(cache.peek(identity(1)).marker, 'after-write')
})

test('an old response that returns after invalidation is shown once but never cached as fresh', async () => {
  const first = deferred()
  const { cache, calls } = createCache((n) => (n === 1 ? first.promise : Promise.resolve(ok('second'))))
  const read = cache.load(identity(1))
  cache.invalidate()
  first.resolve(ok('old'))
  assert.equal((await read).marker, 'old')
  assert.equal(cache.peek(identity(1)), null)
  assert.equal((await cache.load(identity(1))).marker, 'second')
  assert.equal(calls.length, 2)
})

test('switching accounts clears the cache and drops the late response of the old account', async () => {
  const oldAccount = deferred()
  const { cache, calls } = createCache((n) => (n === 1 ? oldAccount.promise : Promise.resolve(ok('new-account'))))
  const late = cache.load(identity(1, 1))
  const fresh = await cache.load(identity(2, 2))
  assert.equal(fresh.marker, 'new-account')
  oldAccount.resolve(ok('old-account'))
  assert.equal(await late, null, 'the old account response must be dropped')
  assert.equal(cache.peek(identity(2, 2)).marker, 'new-account')
  assert.equal(cache.peek(identity(1, 1)), null)
  assert.equal(calls.length, 2)
})

test('a new auth generation of the same user also clears the cache', async () => {
  const { cache, calls } = createCache(async (n) => ok(`call-${n}`))
  await cache.load(identity(1, 1))
  assert.equal(cache.peek(identity(1, 2)), null)
  const after = await cache.load(identity(1, 2))
  assert.equal(after.marker, 'call-2')
  assert.equal(calls.length, 2)
})

test('logged-out identities never read', async () => {
  const { cache, calls } = createCache(async () => ok('x'))
  assert.equal(await cache.load(identity(0)), null)
  assert.equal(await cache.load({}), null)
  assert.equal(calls.length, 0)
})

test('a failed request throws and leaves the cache as it was', async () => {
  const { cache } = createCache(async (n) => (n === 1 ? ok('good') : { success: false, message: '服务繁忙' }))
  await cache.load(identity(1))
  await assert.rejects(() => cache.load(identity(1), { force: true }), /服务繁忙/)
  assert.equal(cache.peek(identity(1)).marker, 'good')
  // 失败后不会卡死在进行中状态
  const { cache: other } = createCache(async () => { throw new Error('网络错误') })
  await assert.rejects(() => other.load(identity(1)), /网络错误/)
  assert.equal(other.inFlight, null)
})

test('the pinia store wires the request facade and keeps the cache in utils', () => {
  const source = readFileSync(new URL('../store/gear.js', import.meta.url), 'utf8')
  assert.match(source, /createGearCacheActions/)
  assert.match(source, /getMyGear/)
  assert.doesNotMatch(source, /uni\./)
})

test('logging out or switching accounts clears the gear cache with the other user-scoped state', async () => {
  const { cache } = createCache(async () => ok('mine'))
  await cache.load(identity(1))
  assert.ok(cache.peek(identity(1)))

  clearUserScopedRuntimeState({ gearStore: cache })
  assert.equal(cache.peek(identity(1)), null)
  assert.equal(cache.ownerUserId, 0)
  assert.equal(cache.data, null)

  const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')
  assert.match(read('store/user.js'), /gearStore: useGearStore\(\)/)
  assert.match(read('utils/session-storage.js'), /gearStore\?\.clear\(\)/)
})
