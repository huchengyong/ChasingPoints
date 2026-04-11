import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveMemberCenterSummary,
  resolveMemberGrowthCard
} from '../utils/member-center.js'

test('resolveMemberGrowthCard returns active growth progress for valid members', () => {
  const card = resolveMemberGrowthCard({
    is_active: true,
    member_expires_at: '2026-05-30 10:00:00',
    growth_points: 78,
    today_growth_count: 3
  }, new Date('2026-04-11T10:00:00+08:00'))

  assert.equal(card.visible, true)
  assert.equal(card.levelLabel, 'Lv3')
  assert.equal(card.growthPointsText, '成长值 78')
  assert.equal(card.todayProgressText, '今日已计入 3/5')
  assert.equal(card.nextLevelText, '距 Lv4 还差 182 场')
  assert.equal(card.frozen, false)
})

test('resolveMemberGrowthCard returns frozen copy after membership expires', () => {
  const card = resolveMemberGrowthCard({
    is_active: false,
    member_expires_at: '2026-03-31 10:00:00',
    growth_points: 78,
    today_growth_count: 1
  }, new Date('2026-04-11T10:00:00+08:00'))

  assert.equal(card.visible, true)
  assert.equal(card.levelLabel, 'Lv3')
  assert.equal(card.statusText, '成长已冻结')
  assert.equal(card.frozen, true)
  assert.match(card.description, /续开后恢复/)
})

test('resolveMemberGrowthCard returns unopened guidance for users without growth progress', () => {
  const card = resolveMemberGrowthCard({}, new Date('2026-04-11T10:00:00+08:00'))

  assert.equal(card.visible, true)
  assert.equal(card.levelLabel, 'Lv1')
  assert.equal(card.growthPointsText, '成长值 0')
  assert.equal(card.todayProgressText, '今日已计入 0/5')
  assert.equal(card.nextLevelText, '开通会员后，完成真实对局可累计成长')
})

test('resolveMemberCenterSummary includes growth progress for active members', () => {
  const summary = resolveMemberCenterSummary({
    is_active: true,
    member_expires_at: '2026-05-30 10:00:00',
    growth_points: 78,
    today_growth_count: 3
  }, new Date('2026-04-11T10:00:00+08:00'))

  assert.equal(summary.statusText, '会员中')
  assert.match(summary.description, /Lv3/)
  assert.match(summary.description, /今日已计入 3\/5/)
})

