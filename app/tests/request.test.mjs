import test from 'node:test'
import assert from 'node:assert/strict'

import {
  shouldResolveBusinessResponse,
  unwrapBusinessResponse
} from '../utils/request-response.js'

test('shouldResolveBusinessResponse returns true for normal success payloads', () => {
  assert.equal(shouldResolveBusinessResponse({ success: true }), true)
  assert.equal(shouldResolveBusinessResponse({ code: 0 }), true)
})

test('shouldResolveBusinessResponse returns true for optimistic-lock conflict payloads with accepted flag', () => {
  assert.equal(shouldResolveBusinessResponse({
    success: false,
    accepted: false,
    message: 'match revision conflict',
    snapshot: {
      server_revision: 12
    }
  }), true)
})

test('shouldResolveBusinessResponse returns false for regular business failures without recovery payload', () => {
  assert.equal(shouldResolveBusinessResponse({
    success: false,
    message: '请求失败'
  }), false)
})

test('unwrapBusinessResponse preserves falsy data payloads', () => {
  assert.equal(unwrapBusinessResponse({ code: 0, data: 0 }), 0)
  assert.equal(unwrapBusinessResponse({ code: 0, data: false }), false)
  assert.equal(unwrapBusinessResponse({ code: 0, data: '' }), '')
})

test('unwrapBusinessResponse falls back to the full envelope only when data key is absent', () => {
  const response = { code: 0, success: true, message: 'ok' }

  assert.equal(unwrapBusinessResponse(response), response)
})
