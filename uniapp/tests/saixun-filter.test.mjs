import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildCurrentYearFilter,
  buildEventNewsListParams,
  buildYearBoundaryDate,
  buildYearDateRange,
  buildYearOptions
} from '../utils/saixun-filter.js'

test('buildCurrentYearFilter defaults to current natural year full range', () => {
  assert.deepEqual(buildCurrentYearFilter('2026-04-07T10:00:00+08:00'), {
    year: 2026,
    from: '2026-01-01',
    to: '2026-12-31',
    preset: 'full_year'
  })
})

test('buildYearDateRange returns full year boundaries', () => {
  assert.deepEqual(buildYearDateRange(2025), {
    from: '2025-01-01',
    to: '2025-12-31'
  })
})

test('buildEventNewsListParams prefers explicit from/to over year', () => {
  assert.deepEqual(buildEventNewsListParams({
    year: 2026,
    from: '2026-01-01',
    to: '2026-04-30'
  }, 2, 20), {
    page: 2,
    page_size: 20,
    from: '2026-01-01',
    to: '2026-04-30'
  })

  assert.deepEqual(buildEventNewsListParams({ year: 2025 }, 1, 10), {
    page: 1,
    page_size: 10,
    year: 2025
  })
})

test('buildEventNewsListParams merges optional game type and status filters', () => {
  assert.deepEqual(buildEventNewsListParams({
    year: 2026,
    gameType: 1,
    status: 2
  }, 1, 10), {
    page: 1,
    page_size: 10,
    year: 2026,
    game_type: 1,
    status: 2
  })

  assert.deepEqual(buildEventNewsListParams({ year: 2026, gameType: 0, status: -1 }, 1, 10), {
    page: 1,
    page_size: 10,
    year: 2026
  })

  assert.deepEqual(buildEventNewsListParams({ year: 2026, status: 0 }, 1, 10), {
    page: 1,
    page_size: 10,
    year: 2026,
    status: 0
  })
})

test('buildYearOptions exposes bounded year choices', () => {
  assert.deepEqual(buildYearOptions(2026, 1, new Date('2026-06-01T00:00:00Z').getTime()), [2025, 2026])
  assert.deepEqual(buildYearOptions(2024, 2, new Date('2026-06-01T00:00:00Z').getTime()), [2023, 2024, 2025, 2026])
  assert.deepEqual(buildYearOptions(2023, 2, new Date('2026-06-01T00:00:00Z').getTime()), [2023, 2024, 2025])
  assert.deepEqual(buildYearOptions(2021, 2, new Date('2026-06-01T00:00:00Z').getTime()), [2023])
})

test('buildYearBoundaryDate clamps the day into the month', () => {
  assert.equal(buildYearBoundaryDate(2026, 2, 31), '2026-02-28')
})
