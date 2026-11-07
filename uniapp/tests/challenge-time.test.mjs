import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  parseServerTime,
  computeScheduledDate,
  availableStartHours,
  defaultChallengeSlot,
  endHourOptions
} from '../utils/challenge-time.js'

test('parseServerTime reads Beijing server time without device timezone', () => {
  const parts = parseServerTime('2026-08-25 15:20:30')
  assert.deepEqual(parts, { year: 2026, month: 8, day: 25, hour: 15, minute: 20 })
  assert.equal(parseServerTime(''), null)
  assert.equal(parseServerTime('garbage'), null)
})

test('computeScheduledDate shifts by day offset on server date', () => {
  assert.equal(computeScheduledDate('2026-08-25 15:20:00', 0), '2026-08-25')
  assert.equal(computeScheduledDate('2026-08-25 15:20:00', 1), '2026-08-26')
  assert.equal(computeScheduledDate('2026-08-31 23:00:00', 2), '2026-09-02')
  assert.equal(computeScheduledDate('', 0), '')
})

test('availableStartHours limits today remaining hours', () => {
  assert.equal(availableStartHours('2026-08-25 15:20:00', 0)[0], 15)
  assert.equal(availableStartHours('2026-08-25 15:20:00', 1).length, 24)
  assert.equal(availableStartHours('', 0).length, 24)
})

test('defaultChallengeSlot picks next full hour, rolls to tomorrow when exhausted', () => {
  // 15:20 → 下一可用整点为 16 点。
  assert.deepEqual(defaultChallengeSlot('2026-08-25 15:20:00'), { dayOffset: 0, startHour: 16, endHour: 17 })
  // 16:00 整点仍可选 16—17。
  assert.deepEqual(defaultChallengeSlot('2026-08-25 16:00:00'), { dayOffset: 0, startHour: 16, endHour: 17 })
  // 23:20 当天没有剩余区间 → 明天 0—1 点。
  assert.deepEqual(defaultChallengeSlot('2026-08-25 23:20:00'), { dayOffset: 1, startHour: 0, endHour: 1 })
})

test('endHourOptions must be later than start and cap at 24', () => {
  assert.deepEqual(endHourOptions(22), [23, 24])
  assert.equal(endHourOptions(0)[0], 1)
  assert.equal(endHourOptions(0).length, 24)
  assert.deepEqual(endHourOptions(24), [])
})
