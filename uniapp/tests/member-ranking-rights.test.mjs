import test from 'node:test'
import assert from 'node:assert/strict'

import {
  MEMBER_RANKING_DAILY_CAP,
  resolveAchievementRecords,
  resolveMemberRankingRightsCard
} from '../utils/member-ranking-rights.js'

test('resolveMemberRankingRightsCard exposes active level multiplier for members', () => {
  const card = resolveMemberRankingRightsCard({
    is_active: true,
    member_expires_at: '2026-05-30 10:00:00',
    growth_level: 3
  }, new Date('2026-04-12T10:00:00+08:00'))

  assert.equal(card.statusText, '权益生效中')
  assert.equal(card.levelLabel, 'Lv3')
  assert.equal(card.currentPercentText, '120%')
  assert.equal(card.dailyCap, MEMBER_RANKING_DAILY_CAP)
  assert.match(card.description, /普通用户高光会保留记录但不计入排位/)
})

test('resolveMemberRankingRightsCard keeps frozen multiplier for expired members', () => {
  const card = resolveMemberRankingRightsCard({
    is_active: false,
    member_expires_at: '2026-04-01 10:00:00',
    growth_level: 4
  }, new Date('2026-04-12T10:00:00+08:00'))

  assert.equal(card.statusText, '权益已冻结')
  assert.equal(card.currentPercentText, '130%')
  assert.match(card.description, /续开后会按 Lv4 的 130% 倍率恢复结算/)
})

test('resolveAchievementRecords keeps only recorded special achievements', () => {
  const records = resolveAchievementRecords({
    break_clear: 1,
    break_100: 2,
    golden_break: 0
  })

  assert.deepEqual(records.map(item => [item.label, item.count]), [
    ['炸清', 1],
    ['单杆100+', 2]
  ])
})
