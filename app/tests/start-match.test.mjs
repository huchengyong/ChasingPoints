import test from 'node:test'
import assert from 'node:assert/strict'

import { validateStartMatchPayload } from '../utils/start-match.js'

test('validateStartMatchPayload rejects empty opponent id', () => {
  assert.equal(
    validateStartMatchPayload({
      game_type: 3,
      opponent_id: 0,
      opponent_name: '球友A'
    }),
    '请选择有效的平台对手'
  )
})

test('validateStartMatchPayload accepts positive opponent id', () => {
  assert.equal(
    validateStartMatchPayload({
      game_type: 3,
      opponent_id: 2001,
      opponent_name: '球友A'
    }),
    ''
  )
})
