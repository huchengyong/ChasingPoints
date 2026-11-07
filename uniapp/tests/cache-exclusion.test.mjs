import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const read = (path) => readFile(new URL(path, import.meta.url), 'utf8')

test('generic request transport does not cache GET responses', async () => {
  const source = await read('../utils/request.js')
  assert.doesNotMatch(source, /responseCache|cacheKey|staleWhileRevalidate|localStorage|setStorageSync/)
})

test('sensitive and realtime API facades do not import public read cache stores', async () => {
  const paths = [
    '../api/auth.js',
    '../api/member.js',
    '../api/match.js',
    '../api/user.js'
  ]
  for (const path of paths) {
    const source = await read(path)
    assert.doesNotMatch(source, /publicRead|loadStatic|loadLeaderboard|loadVenueCache/)
  }
})
