import test from 'node:test'
import assert from 'node:assert/strict'

import { formatDateTime, formatMonthKey, formatRelativeTime } from '../utils/format.js'

const pad = (value) => String(value).padStart(2, '0')
const expectedLocal = (date) => `${date.getFullYear()}年${pad(date.getMonth() + 1)}月${pad(date.getDate())}日 ${pad(date.getHours())}:${pad(date.getMinutes())}`
const spaceSeparated = (date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`

test('formatDateTime reads both ISO offsets and space separated timestamps', () => {
  const iso = new Date('2026-11-10T20:28:53+08:00')
  assert.equal(formatDateTime('2026-11-10T20:28:53+08:00'), expectedLocal(iso))
  // iOS 小程序无法解析空格分隔的串，格式化前需归一化。
  assert.equal(formatDateTime('2026-11-10 20:28:53'), expectedLocal(new Date(2026, 10, 10, 20, 28, 53)))
})

test('formatDateTime returns an empty string for unparseable input', () => {
  assert.equal(formatDateTime('not-a-date'), '')
})

test('formatRelativeTime reads space separated timestamps and ISO strings', () => {
  const now = Date.now()

  assert.equal(formatRelativeTime(spaceSeparated(new Date(now - 2 * 60 * 60 * 1000))), '2小时前')
  assert.equal(formatRelativeTime(new Date(now - 3 * 60 * 1000).toISOString()), '3分钟前')
})

test('formatRelativeTime returns an empty string for unparseable input', () => {
  assert.equal(formatRelativeTime('not-a-date'), '')
  assert.equal(formatRelativeTime(''), '')
})

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
