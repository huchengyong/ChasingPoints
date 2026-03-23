import test from 'node:test'
import assert from 'node:assert/strict'

import { formatMonthKey } from '../utils/format.js'

test('formatMonthKey uses local calendar month instead of UTC month', () => {
  const boundaryDate = new Date('2026-03-31T16:30:00.000Z')

  assert.equal(formatMonthKey(boundaryDate), '2026-04')
})
