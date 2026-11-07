import test from 'node:test'
import assert from 'node:assert/strict'

import { resolveSeasonTimeline } from '../utils/season-lifecycle.js'

const season = {
  start_date: '2026-07-01',
  end_date: '2026-07-31'
}

test('season timeline keeps the full end date in the half-open window', () => {
  const noonOnEndDate = Date.parse('2026-07-31T12:00:00+08:00')
  const timeline = resolveSeasonTimeline(season, noonOnEndDate)

  assert.equal(timeline.remainDays, 1)
  assert.ok(timeline.progressPercent > 98)
  assert.ok(timeline.progressPercent < 100)
})

test('season timeline reaches zero only at the next business-day boundary', () => {
  const endExclusive = Date.parse('2026-08-01T00:00:00+08:00')
  assert.deepEqual(resolveSeasonTimeline(season, endExclusive), {
    remainDays: 0,
    progressPercent: 100
  })
})

test('season timeline honors authoritative timestamps from the configured business timezone', () => {
  const timeline = resolveSeasonTimeline({
    start_date: '2026-07-01',
    end_date: '2026-07-31',
    start_at: '2026-07-01T00:00:00-04:00',
    end_at_exclusive: '2026-08-01T00:00:00-04:00'
  }, Date.parse('2026-08-01T03:00:00Z'))

  assert.equal(timeline.remainDays, 1)
  assert.ok(timeline.progressPercent < 100)
})

test('season timeline rejects invalid date-only values', () => {
  assert.deepEqual(resolveSeasonTimeline({ start_date: '2026-07-01', end_date: 'invalid' }), {
    remainDays: 0,
    progressPercent: 0
  })
})
