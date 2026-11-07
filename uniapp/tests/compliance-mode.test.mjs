import test from 'node:test'
import assert from 'node:assert/strict'

import {
  APP_COMPLIANCE_MODE,
  canShowPaidMembershipEntry,
  canShowSocialPublishing,
  canShowUserTournamentActions,
  getSaiXunNavigationTitle,
  getSaiXunTabLabel
} from '../utils/compliance-mode.js'

test('compliance mode disables high-risk launch features by default', () => {
  assert.equal(APP_COMPLIANCE_MODE, true)
  assert.equal(canShowPaidMembershipEntry(), false)
  assert.equal(canShowSocialPublishing(), false)
  assert.equal(canShowUserTournamentActions(), false)
})

test('compliance mode exposes saixun naming helpers', () => {
  assert.equal(getSaiXunTabLabel(), '赛讯')
  assert.equal(getSaiXunNavigationTitle(), '赛讯')
})
