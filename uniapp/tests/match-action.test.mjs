import test from 'node:test'
import assert from 'node:assert/strict'

import { buildMatchActionPayload } from '../utils/match-action.js'

test('buildMatchActionPayload appends client_action_id and base_revision to write payloads', () => {
  const payload = buildMatchActionPayload({
    match_id: 18,
    actor: 2,
    score: 7
  }, 12, () => 'action-123')

  assert.deepEqual(payload, {
    match_id: 18,
    actor: 2,
    score: 7,
    client_action_id: 'action-123',
    base_revision: 12
  })
})

test('buildMatchActionPayload falls back to revision 0 when client has not synced yet', () => {
  const payload = buildMatchActionPayload({
    match_id: 19
  }, undefined, () => 'action-456')

  assert.equal(payload.client_action_id, 'action-456')
  assert.equal(payload.base_revision, 0)
})
