import test from 'node:test'
import assert from 'node:assert/strict'

import {
  createMemberPaymentRequest,
  formatMemberPrice,
  getMemberPayChannelOptions,
  resolveMemberCenterSummary,
  resolveMemberEntryCard,
  resolveMemberPlanCards,
  shouldTreatMemberOrderAsPaid
} from '../utils/member-center.js'

test('resolveMemberEntryCard returns unopened copy when user has never subscribed', () => {
  const card = resolveMemberEntryCard({}, new Date('2026-03-31T10:00:00+08:00'))

  assert.equal(card.statusText, '未开通')
  assert.equal(card.title, '月卡会员')
  assert.equal(card.actionText, '立即开通')
})

test('resolveMemberEntryCard returns active copy when membership is still valid', () => {
  const card = resolveMemberEntryCard({
    is_active: true,
    member_expires_at: '2026-04-30 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'))

  assert.equal(card.statusText, '会员中')
  assert.equal(card.title, '会员有效期至 2026-04-30 10:00:00')
  assert.equal(card.actionText, '立即续费')
})

test('resolveMemberCenterSummary treats member_expires_at as UTC+8 instead of local device time', () => {
  const summary = resolveMemberCenterSummary({
    is_active: true,
    member_expires_at: '2026-03-31 00:30:00'
  }, new Date('2026-03-30T16:20:00Z'))

  assert.equal(summary.statusText, '会员中')
})

test('resolveMemberCenterSummary returns expired copy after member expires', () => {
  const summary = resolveMemberCenterSummary({
    is_active: false,
    member_expires_at: '2026-03-20 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'))

  assert.equal(summary.statusText, '已到期')
  assert.equal(summary.title, '会员已到期')
  assert.match(summary.description, /2026-03-20 10:00:00/)
})

test('resolveMemberPlanCards marks the selected plan only', () => {
  const cards = resolveMemberPlanCards([
    { plan_code: 'member_monthly', plan_name: '月卡会员' },
    { plan_code: 'member_yearly', plan_name: '年卡会员' }
  ], 'member_monthly')

  assert.equal(cards[0].selected, true)
  assert.equal(cards[1].selected, false)
})

test('member center helpers expose stable payment payload helpers', () => {
  assert.deepEqual(createMemberPaymentRequest({
    planCode: 'member_monthly',
    payChannel: 'alipay'
  }), {
    plan_code: 'member_monthly',
    pay_channel: 'alipay'
  })

  assert.equal(formatMemberPrice(1900), '¥19.00')
  assert.equal(shouldTreatMemberOrderAsPaid({ success: true, status: 'paid' }), true)
  assert.equal(shouldTreatMemberOrderAsPaid({ success: true, status: 'pending' }), false)
})

test('member center exposes app payment channel options', () => {
  const options = getMemberPayChannelOptions()

  assert.equal(options.length, 2)
  assert.deepEqual(options.map(item => item.value), ['alipay', 'wechat'])
})

test('resolveMemberCenterSummary avoids payment wording in compliance mode', () => {
  const summary = resolveMemberCenterSummary({
    is_active: true,
    member_expires_at: '2026-04-30 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'), { complianceMode: true })

  assert.equal(summary.statusText, '会员权益中')
  assert.doesNotMatch(summary.title, /月卡|开通|续费/)
  assert.match(summary.description, /获赠|有效期/)
  assert.equal(summary.primaryActionText, '查看权益')
})

test('resolveMemberEntryCard avoids price wording in compliance mode', () => {
  const card = resolveMemberEntryCard({}, new Date('2026-03-31T10:00:00+08:00'), { complianceMode: true })

  assert.equal(card.statusText, '待发放')
  assert.doesNotMatch(card.title, /月卡|开通/)
  assert.equal(card.actionText, '查看权益')
  assert.equal(card.priceText, '')
})

test('resolveMemberEntryCard keeps active compliance copy focused on the entry itself', () => {
  const card = resolveMemberEntryCard({
    is_active: true,
    member_expires_at: '2026-04-30 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'), { complianceMode: true })

  assert.equal(card.statusText, '会员权益中')
  assert.equal(card.title, '会员权益已生效')
  assert.equal(card.description, '查看当前权益明细和有效期。')
  assert.equal(card.actionText, '查看权益')
  assert.equal(card.priceText, '')
})
