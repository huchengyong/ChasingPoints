import test from 'node:test'
import assert from 'node:assert/strict'

import { formatMonthKey } from '../utils/format.js'

test('formatMonthKey uses local calendar month instead of UTC month', () => {
  const boundaryDate = new Date('2026-03-31T16:30:00.000Z')

  assert.equal(formatMonthKey(boundaryDate), '2026-04')
})

test('formatMonthKey falls back gracefully when Intl is unavailable', () => {
  const originalIntl = globalThis.Intl

  globalThis.Intl = undefined

  try {
    assert.equal(formatMonthKey(new Date(2026, 3, 1, 8, 0, 0)), '2026-04')
  } finally {
    globalThis.Intl = originalIntl
  }
})
