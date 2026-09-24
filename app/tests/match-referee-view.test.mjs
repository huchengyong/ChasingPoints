import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveCompletionSourceLabel,
  resolveRefereeHistoryCard,
  resolveRefereeIdentityCard,
  resolveRefereeResultViewConfig
} from '../utils/match-referee-view.js'

test('referee view keeps player and referee result modes distinct', () => {
  assert.equal(resolveRefereeResultViewConfig({ viewerRole: 'player1', status: 2 }).showH2HActions, true)
  assert.equal(resolveRefereeResultViewConfig({ viewerRole: 'player2', status: 2 }).showH2HActions, true)
  const referee = resolveRefereeResultViewConfig({ viewerRole: 'referee', status: 2 })
  assert.equal(referee.pageTitle, '执裁详情')
  assert.equal(referee.showH2HActions, false)
  assert.equal(referee.showRematchButton, false)
})

test('unknown and cancelled attribution never claims referee settlement', () => {
  const unknown = resolveRefereeIdentityCard({
    refereeBound: true,
    refereeUserId: 3,
    completionSource: 'unknown',
    status: 2
  })
  assert.equal(unknown.hasReliableAttribution, false)
  assert.equal(unknown.completionLabel, '')

  const cancelled = resolveRefereeHistoryCard({
    id: 9,
    status: 3,
    completion_source: 'referee'
  })
  assert.equal(cancelled.statusText, '已取消')
  assert.equal(cancelled.completionLabel, '')
})

test('ordinary referee copy remains neutral and known completion labels stay safe', () => {
  const card = resolveRefereeIdentityCard({
    refereeBound: true,
    refereeUserId: 3,
    refereeName: '裁判丙',
    completionSource: 'referee',
    status: 2
  })
  assert.equal(card.neutralLabel, '本场裁判')
  assert.equal(card.completionLabel, '裁判结算')
  assert.equal(resolveCompletionSourceLabel('player_cancelled'), '未知')
})
