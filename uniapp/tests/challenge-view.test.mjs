import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  formatChallengeDayLabel,
  formatChallengeSchedule,
  formatChallengeExpiry
} from '../utils/challenge-view.js'

test('day label follows Beijing server date across midnight and years', () => {
  assert.equal(formatChallengeDayLabel('2026-08-25', '2026-08-25 21:00:00'), '今天')
  assert.equal(formatChallengeDayLabel('2026-08-25', '2026-08-26 00:30:00'), '昨天')
  assert.equal(formatChallengeDayLabel('2026-08-26', '2026-08-25 12:00:00'), '明天')
  assert.equal(formatChallengeDayLabel('2026-08-27', '2026-08-25 12:00:00'), '8月27日')
  assert.equal(formatChallengeDayLabel('2026-08-25', '2026-08-27 12:00:00'), '8月25日')
  assert.equal(formatChallengeDayLabel('2025-12-31', '2027-01-02 12:00:00'), '2025年12月31日')
})

test('schedule line composes day and hour range', () => {
  assert.equal(
    formatChallengeSchedule({ scheduled_date: '2026-08-25', start_hour: 16, end_hour: 17 }, '2026-08-25 12:00:00'),
    '今天16点—17点'
  )
  assert.equal(
    formatChallengeSchedule({ scheduled_date: '2026-08-25', start_hour: 23, end_hour: 24 }, '2026-08-26 00:30:00'),
    '昨天23点—24点'
  )
})

test('expiry copy uses next-morning 7am wording and 已失效 after expiry', () => {
  assert.equal(
    formatChallengeExpiry({ expires_at: '2026-08-26 07:00:00' }, '2026-08-25 21:00:00'),
    '明天早晨7点失效'
  )
  assert.equal(
    formatChallengeExpiry({ expires_at: '2026-08-26 07:00:00' }, '2026-08-26 00:30:00'),
    '今天早晨7点失效'
  )
  assert.equal(
    formatChallengeExpiry({ expires_at: '2026-08-26 07:00:00' }, '2026-08-26 07:00:00'),
    '已失效'
  )
})
