import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  formatChallengeDayLabel,
  formatChallengeSchedule,
  formatChallengeExpiry,
  resolveChallengeQuickCard
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

test('quick card flags challenges that wait on me and links to the right page', () => {
  const serverTime = '2026-08-25 12:00:00'
  const opponentEntered = resolveChallengeQuickCard({
    id: 5,
    status: 1,
    from_user_id: 7,
    from_nickname: '阿杰',
    to_user_id: 18,
    waiting_user_id: 7,
    game_type: 3,
    match_mode: 'ranked',
    scheduled_date: '2026-08-25',
    start_hour: 16,
    end_hour: 17
  }, { userId: 18, serverTime })

  assert.equal(opponentEntered.title, '对方已进入，等你开始')
  assert.equal(opponentEntered.needsAction, true)
  assert.equal(opponentEntered.subText, '阿杰 · 中式八球 · 排位 · 今天16点—17点')
  assert.equal(opponentEntered.actionLabel, '进入')
  assert.equal(opponentEntered.url, '/subPages/match/challengeWaiting?challenge_id=5')

  const sent = resolveChallengeQuickCard({ id: 6, status: 0, from_user_id: 18, to_nickname: '球友 B', game_type: 2, match_mode: 'practice' }, { userId: 18 })
  assert.equal(sent.title, '等待对方回应')
  assert.equal(sent.needsAction, false)
  assert.equal(sent.subText, '球友 B · 九球追分 · 练习')

  const received = resolveChallengeQuickCard({ id: 7, status: 0, from_user_id: 9, from_nickname: '老王' }, { userId: 18 })
  assert.equal(received.title, '收到约球邀请')
  assert.equal(received.needsAction, true)

  const started = resolveChallengeQuickCard({ id: 8, status: 6, from_user_id: 18, match_id: 42 }, { userId: 18 })
  assert.equal(started.matchId, 42)
  assert.equal(started.challengeId, 8)
  assert.equal(started.actionLabel, '继续比赛')
  assert.equal(started.url, '/subPages/match/playing?match_id=42')

  assert.equal(resolveChallengeQuickCard(null), null)
  assert.equal(resolveChallengeQuickCard({ id: 9, status: 4 }, { userId: 18 }), null)
})
