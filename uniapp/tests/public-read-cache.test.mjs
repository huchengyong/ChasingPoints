import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildNearbyVenueCacheKey,
  buildPublicReadKey
} from '../utils/public-read-cache.js'

test('public read keys are stable when parameter property order changes', () => {
  assert.equal(
    buildPublicReadKey('event-news', { page: 1, game_type: 3 }),
    buildPublicReadKey('event-news', { game_type: 3, page: 1 })
  )
})

test('nearby venue keys bucket coordinates instead of retaining full precision', () => {
  const first = buildNearbyVenueCacheKey({
    latitude: 22.54311,
    longitude: 114.05781,
    radius: 5000,
    limit: 3
  })
  const second = buildNearbyVenueCacheKey({
    latitude: 22.54314,
    longitude: 114.05784,
    radius: 5000,
    limit: 3
  })

  assert.equal(first, second)
  assert.doesNotMatch(first, /22\.54311|114\.05781/)
})
