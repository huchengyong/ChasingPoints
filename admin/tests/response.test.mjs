import test from 'node:test'
import assert from 'node:assert/strict'

import { unwrapBusinessData } from '../src/utils/response.ts'

test('unwrapBusinessData preserves falsy payloads', () => {
  assert.equal(unwrapBusinessData({ code: 0, data: 0 }), 0)
  assert.equal(unwrapBusinessData({ code: 0, data: false }), false)
  assert.equal(unwrapBusinessData({ code: 0, data: '' }), '')
})

test('unwrapBusinessData returns the original envelope when data field is absent', () => {
  const response = { code: 0, success: true, message: 'ok' }

  assert.equal(unwrapBusinessData(response), response)
})
