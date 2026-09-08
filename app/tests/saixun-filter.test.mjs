import test from 'node:test'
import assert from 'node:assert/strict'

import {
  SAIXUN_DATE_PRESETS,
  buildCurrentYearFilter,
  buildCustomDateRange,
  buildEventNewsListParams,
  buildPresetDateRange,
  buildYearBoundaryDate,
  buildYearDateRange,
  buildYearOptions,
  formatSaiXunFilterLabel
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

test('buildPresetDateRange supports quick ranges', () => {
  assert.deepEqual(buildPresetDateRange(2026, 'q1'), {
    from: '2026-01-01',
    to: '2026-03-31',
    preset: 'q1'
  })
  assert.deepEqual(buildPresetDateRange(2026, 'jan_apr'), {
    from: '2026-01-01',
    to: '2026-04-30',
    preset: 'jan_apr'
  })
})

test('formatSaiXunFilterLabel formats preset and custom ranges', () => {
  assert.equal(formatSaiXunFilterLabel({ from: '2026-01-01', to: '2026-12-31', preset: 'full_year' }), '全年')
  assert.equal(formatSaiXunFilterLabel({ from: '2026-01-01', to: '2026-03-31', preset: 'q1' }), '1月-3月')
  assert.equal(formatSaiXunFilterLabel({ from: '2026-01-01', to: '2026-04-30', preset: 'custom' }), '2026.01.01 - 2026.04.30')
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

test('buildYearOptions and presets expose expected choices', () => {
  assert.deepEqual(buildYearOptions(2026, 1, new Date('2026-06-01T00:00:00Z').getTime()), [2025, 2026])
  assert.deepEqual(buildYearOptions(2024, 2, new Date('2026-06-01T00:00:00Z').getTime()), [2022, 2023, 2024, 2025, 2026])
  assert.deepEqual(SAIXUN_DATE_PRESETS.map(item => item.key), ['full_year', 'q1', 'jan_apr', 'jul_dec', 'custom'])
})

test('buildCustomDateRange and buildYearBoundaryDate normalize date values', () => {
  assert.deepEqual(buildCustomDateRange(2026, '2026-02-01', '2026-02-28'), {
    year: 2026,
    from: '2026-02-01',
    to: '2026-02-28',
    preset: 'custom'
  })
  assert.deepEqual(buildCustomDateRange(2026, '2026-03-10', '2026-02-28'), {
    year: 2026,
    from: '2026-02-28',
    to: '2026-03-10',
    preset: 'custom'
  })
  assert.equal(buildYearBoundaryDate(2026, 2, 31), '2026-02-28')
})
