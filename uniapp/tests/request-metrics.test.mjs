import test from 'node:test'
import assert from 'node:assert/strict'

import {
  getRequestCountSnapshot,
  normalizeRequestRoute,
  recordRequest,
  resetRequestCounts
} from '../utils/request-metrics.js'

test.beforeEach(() => {
  globalThis.__CHASING_POINTS_REQUEST_METRICS__ = true
  resetRequestCounts()
})

test.after(() => {
  delete globalThis.__CHASING_POINTS_REQUEST_METRICS__
})

test('request metrics count only method and normalized route', () => {
  recordRequest('get', '/api/match/detail?match_id=42&token=secret')
  recordRequest('POST', 'https://example.com/api/user/info?access_token=secret')
  recordRequest('GET', '/api/match/detail?match_id=99')

  assert.deepEqual(getRequestCountSnapshot(), {
    'GET /api/match/detail': 2,
    'POST /api/user/info': 1
  })
})

test('request route normalization removes origin and query string', () => {
  assert.equal(normalizeRequestRoute('https://api.example.com/api/event-news/list?page=1'), '/api/event-news/list')
  assert.equal(normalizeRequestRoute('api/user/info'), '/api/user/info')
  assert.equal(normalizeRequestRoute(''), '/')
})

test('request metrics are disabled in production without explicit debug enablement', () => {
  const originalNodeEnv = process.env.NODE_ENV
  process.env.NODE_ENV = 'production'
  delete globalThis.__CHASING_POINTS_REQUEST_METRICS__
  resetRequestCounts()

  recordRequest('GET', '/api/user/info')
  assert.deepEqual(getRequestCountSnapshot(), {})

  if (originalNodeEnv === undefined) {
    delete process.env.NODE_ENV
  } else {
    process.env.NODE_ENV = originalNodeEnv
  }
  globalThis.__CHASING_POINTS_REQUEST_METRICS__ = true
})
