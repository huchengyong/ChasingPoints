import test from 'node:test'
import assert from 'node:assert/strict'

import {
  createMemberPaymentRequest,
  formatMemberPrice,
  getMemberPayChannelOptions,
  resolveMemberCenterSummary,
  resolveMemberIdentityPreview,
  resolveMemberEntryCard,
  resolveMemberPlanCards,
  shouldTreatMemberOrderAsPaid
} from '../utils/member-center.js'

test('resolveMemberCenterSummary keeps compliance copy neutral before any reward is granted', () => {
  const summary = resolveMemberCenterSummary({ is_active: false }, new Date('2026-10-19T10:00:00+08:00'), {
    complianceMode: true
  })

  assert.equal(summary.statusText, '暂无权益')
  assert.doesNotMatch(summary.description, /注册/)
})

test('resolveMemberEntryCard returns unopened copy when user has never subscribed', () => {
  const card = resolveMemberEntryCard({}, new Date('2026-03-31T10:00:00+08:00'))

  assert.equal(card.statusText, '未开通')
  assert.equal(card.title, '月卡VIP')
  assert.equal(card.actionText, '立即开通')
})

test('resolveMemberEntryCard returns active copy when membership is still valid', () => {
  const card = resolveMemberEntryCard({
    is_active: true,
    member_expires_at: '2026-04-30 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'))

  assert.equal(card.statusText, 'VIP中')
  assert.equal(card.title, 'VIP有效期至 2026-04-30 10:00:00')
  assert.equal(card.actionText, '立即续费')
})

test('resolveMemberCenterSummary treats member_expires_at as UTC+8 instead of local device time', () => {
  const summary = resolveMemberCenterSummary({
    is_active: true,
    member_expires_at: '2026-03-31 00:30:00',
    growth_level: 2
  }, new Date('2026-03-30T16:20:00Z'))

  assert.equal(summary.statusText, 'VIP中')
  assert.match(summary.description, /展示「标准」外观/)
  assert.doesNotMatch(summary.description, /倍率|110%|高光/)
})

test('resolveMemberCenterSummary returns expired copy after member expires', () => {
  const summary = resolveMemberCenterSummary({
    is_active: false,
    member_expires_at: '2026-03-20 10:00:00',
    growth_level: 3
  }, new Date('2026-03-31T10:00:00+08:00'))

  assert.equal(summary.statusText, '已到期')
  assert.equal(summary.title, 'VIP已到期')
  assert.match(summary.description, /2026-03-20 10:00:00/)
  assert.match(summary.description, /「进阶」外观/)
  assert.doesNotMatch(summary.description, /倍率|120%|高光/)
})

test('resolveMemberPlanCards marks the selected plan only', () => {
  const cards = resolveMemberPlanCards([
    { plan_code: 'member_monthly', plan_name: '月卡VIP' },
    { plan_code: 'member_yearly', plan_name: '年卡VIP' }
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

  assert.equal(summary.statusText, 'VIP权益中')
  assert.doesNotMatch(summary.title, /月卡|开通|续费/)
  assert.match(summary.description, /获赠|有效期/)
  assert.equal(summary.primaryActionText, '查看权益')
})

test('resolveMemberEntryCard avoids price wording in compliance mode', () => {
  const card = resolveMemberEntryCard({}, new Date('2026-03-31T10:00:00+08:00'), { complianceMode: true })

  assert.equal(card.visible, false)
  assert.equal(card.statusText, '')
  assert.equal(card.title, '')
  assert.equal(card.description, '')
  assert.equal(card.actionText, '')
  assert.equal(card.eyebrow, '')
  assert.equal(card.priceText, '')
})

test('resolveMemberEntryCard keeps active compliance copy focused on the entry itself', () => {
  const card = resolveMemberEntryCard({
    is_active: true,
    member_expires_at: '2026-04-30 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'), { complianceMode: true })

  assert.equal(card.statusText, 'VIP权益中')
  assert.equal(card.title, 'VIP权益已生效')
  assert.equal(card.description, '查看当前权益明细和有效期。')
  assert.equal(card.actionText, '查看权益')
  assert.equal(card.priceText, '')
})

test('resolveMemberEntryCard shows ended compliance copy when member history exists', () => {
  const card = resolveMemberEntryCard({
    is_active: false,
    member_expires_at: '2026-03-20 10:00:00'
  }, new Date('2026-03-31T10:00:00+08:00'), { complianceMode: true })

  assert.equal(card.visible, true)
  assert.equal(card.statusText, '已结束')
  assert.equal(card.title, '获赠VIP已结束')
  assert.doesNotMatch(card.title, /月卡|开通|续费/)
  assert.equal(card.actionText, '查看权益')
  assert.equal(card.eyebrow, 'VIP权益')
  assert.equal(card.priceText, '')
})

test('resolveMemberCenterSummary no longer advertises ranking multipliers in any state', () => {
  const now = new Date('2026-11-05T10:00:00+08:00')
  const states = [
    { is_active: true, member_expires_at: '2026-12-31 00:30:00', growth_level: 2 },
    { is_active: true, member_expires_at: '2026-12-31 00:30:00', growth_level: 5 },
    { is_active: false, member_expires_at: '2026-11-02 10:00:00', growth_level: 3 },
    {}
  ]
  for (const status of states) {
    const summary = resolveMemberCenterSummary(status, now)
    assert.doesNotMatch(`${summary.title} ${summary.description}`, /高光|倍率|计入排位|\d+%/, JSON.stringify(status))
  }
  assert.match(resolveMemberCenterSummary({}, now).description, /成长等级解锁更高阶外观/)
  assert.match(resolveMemberCenterSummary(states[1], now).description, /「尊享」外观/)
})

test('resolveMemberIdentityPreview shows the current tier only for active members', () => {
  const now = new Date('2026-11-05T10:00:00+08:00')
  const active = resolveMemberIdentityPreview({ is_active: true, member_expires_at: '2026-12-31 00:30:00', growth_level: 4 }, now, { rankLevel: 5 })
  assert.equal(active.active, true)
  assert.equal(active.level, 4)
  assert.equal(active.identity.tierLabel, '进阶')
  assert.equal(active.identity.badgeText, 'VIP4')
  assert.match(active.identity.frameUrl, /frame_diamond\.png$/)
  assert.deepEqual(active.tiers.map((item) => [item.label, item.levelText, item.current, item.reached]), [
    ['标准', 'Lv1–Lv2', false, true],
    ['进阶', 'Lv3–Lv4', true, true],
    ['尊享', 'Lv5', false, false]
  ])
  assert.deepEqual(active.tiers.map((item) => item.badgeText), ['VIP1', 'VIP3', 'VIP5'])
  assert.match(active.hint, /进阶/)
  assert.match(active.hint, /排行榜、对局和好友列表/)
})

test('resolveMemberIdentityPreview hides the identity after expiry and explains it is restored on renewal', () => {
  const now = new Date('2026-11-05T10:00:00+08:00')
  const expired = resolveMemberIdentityPreview({ is_active: false, member_expires_at: '2026-11-02 10:00:00', growth_level: 3 }, now, { rankLevel: 4 })
  assert.equal(expired.active, false)
  assert.equal(expired.frozen, true)
  assert.equal(expired.identity.showBadge, false)
  assert.equal(expired.identity.frameUrl, '')
  assert.equal(expired.tiers.some((item) => item.current), false)
  assert.equal(expired.level, 3, 'growth is frozen, not lost')
  assert.match(expired.hint, /已隐藏/)
  assert.match(expired.hint, /续开/)
  assert.match(expired.hint, /「进阶」/)

  const never = resolveMemberIdentityPreview({}, now)
  assert.equal(never.frozen, false)
  assert.equal(never.identity.showBadge, false)
  assert.match(never.hint, /开通VIP后/)
})

test('resolveMemberIdentityPreview avoids purchase wording in compliance mode', () => {
  const now = new Date('2026-11-05T10:00:00+08:00')
  const texts = [
    resolveMemberIdentityPreview({}, now, { complianceMode: true }).hint,
    resolveMemberIdentityPreview({ is_active: false, member_expires_at: '2026-11-02 10:00:00', growth_level: 2 }, now, { complianceMode: true }).hint
  ]
  for (const text of texts) {
    assert.doesNotMatch(text, /开通|订阅|续开|续费|支付/)
  }
  assert.match(texts[0], /获得VIP权益后/)
})
