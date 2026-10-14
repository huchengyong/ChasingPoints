import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildFinishSlideNote,
  formatSettlementClock,
  resolveRankSettlementView
} from '../utils/rank-settlement.js'

const match = (overrides = {}) => ({
  is_player1: false,
  player1_id: 1001,
  player2_id: 2002,
  my_name: '小王',
  opponent_name: '小李',
  completed_by_user_id: 1001,
  rank_settlement_status: 'pending',
  rank_settlement_due_at: '2026-08-25T16:10:00+08:00',
  can_dispute_result: true,
  ...overrides
})

test('settlement clock is shown in Beijing time whatever offset the server used', () => {
  assert.equal(formatSettlementClock('2026-08-25T16:10:00+08:00'), '16:10')
  assert.equal(formatSettlementClock('2026-08-25T08:10:00Z'), '16:10')
  assert.equal(formatSettlementClock('2026-08-25T16:05:00Z'), '00:05')
  assert.equal(formatSettlementClock(''), '')
  assert.equal(formatSettlementClock('不是时间'), '')
})

test('the player who did not finish sees the pending settlement and a dispute entry', () => {
  const view = resolveRankSettlementView(match())
  assert.equal(view.visible, true)
  assert.equal(view.hideRankChange, true)
  assert.equal(view.statusText, '16:10 生效')
  assert.equal(view.rankValueText, '待生效')
  assert.equal(view.finisherText, '小李滑动结束')
  assert.equal(view.canDispute, true)
  assert.match(view.disputeHint, /小李结束了本场/)
  assert.match(view.disputeHint, /16:10 前提出/)
  assert.match(view.disputeHint, /不计排位积分，比分与记录保留/)
})

test('the finisher sees the same settlement but never a dispute entry', () => {
  const view = resolveRankSettlementView(match({
    is_player1: true, my_name: '小李', opponent_name: '小王', can_dispute_result: false
  }))
  assert.equal(view.finisherText, '你滑动结束')
  assert.equal(view.statusText, '16:10 生效')
  assert.equal(view.canDispute, false)
  assert.equal(view.disputeHint, '')
})

test('the dispute entry follows the server flag only, never the phone clock', () => {
  assert.equal(resolveRankSettlementView(match({ can_dispute_result: false })).canDispute, false)
  assert.equal(resolveRankSettlementView(match({ can_dispute_result: undefined })).canDispute, false)
  const farPast = resolveRankSettlementView(match({ rank_settlement_due_at: '2020-01-01T00:00:00+08:00' }))
  assert.equal(farPast.canDispute, true)
})

test('a pending or voided result never shows a rank change of zero', () => {
  const pending = resolveRankSettlementView(match())
  assert.equal(pending.hideRankChange, true)
  assert.match(pending.rankNote, /16:10 生效/)

  const voided = resolveRankSettlementView(match({ rank_settlement_status: 'voided', can_dispute_result: false }))
  assert.equal(voided.hideRankChange, true)
  assert.equal(voided.statusText, '有异议 · 不计排位积分')
  assert.equal(voided.rankValueText, '不计分')
  assert.equal(voided.canDispute, false)
})

test('a settled result keeps the normal rank display and says it took effect', () => {
  const view = resolveRankSettlementView(match({ rank_settlement_status: 'settled', can_dispute_result: false }))
  assert.equal(view.visible, true)
  assert.equal(view.hideRankChange, false)
  assert.equal(view.statusText, '已生效')
  assert.equal(view.canDispute, false)
})

test('matches without a dispute window use the ordinary result page', () => {
  for (const overrides of [
    { rank_settlement_status: 'none', rank_settlement_due_at: '' },
    { rank_settlement_status: 'settled', rank_settlement_due_at: '' },
    { rank_settlement_status: undefined, rank_settlement_due_at: undefined },
    { rank_settlement_status: 'unknown-status', rank_settlement_due_at: '' }
  ]) {
    const view = resolveRankSettlementView(match(overrides))
    assert.equal(view.visible, false, JSON.stringify(overrides))
    assert.equal(view.hideRankChange, false)
    assert.equal(view.canDispute, false)
  }
  assert.equal(resolveRankSettlementView(null).visible, false)
})

test('a referee never sees a viewer-relative finisher name or dispute entry', () => {
  const view = resolveRankSettlementView(match({ viewer_role: 'referee', can_dispute_result: false }))
  assert.equal(view.canDispute, false)
  assert.equal(view.finisherText, '对方滑动结束')
})

test('slide note always says no confirmation is needed and explains ranked consequences', () => {
  assert.equal(
    buildFinishSlideNote({ challengeId: 7, matchMode: 'ranked', myScore: 5, opponentScore: 3 }),
    '松手即结束，无需对方确认。按 5 : 3 结算排位，对方10分钟内可提出异议。'
  )
  assert.equal(
    buildFinishSlideNote({ challengeId: 7, matchMode: 'ranked', myScore: 3, opponentScore: 5 }),
    '松手即结束，无需对方确认。按 3 : 5 结算排位，对方10分钟内可提出异议。'
  )
  for (const input of [
    { challengeId: 7, matchMode: 'practice', myScore: 5, opponentScore: 3 },
    { challengeId: 7, matchMode: 'ranked', myScore: 4, opponentScore: 4 },
    { challengeId: 0, matchMode: 'ranked', myScore: 5, opponentScore: 3 },
    { challengeId: 7, matchMode: 'ranked' },
    undefined
  ]) {
    assert.equal(buildFinishSlideNote(input), '松手即结束，无需对方确认。', JSON.stringify(input))
  }
})
