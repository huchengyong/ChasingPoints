import test from 'node:test'
import assert from 'node:assert/strict'

import {
  calculateReconnectDelay,
  classifyWebSocketFailure,
  isCurrentConnectionGeneration,
  nextConnectionGeneration,
  shouldReconnectWebSocket
} from '../utils/websocket-reconnect.js'

test('websocket reconnect delay grows exponentially with jitter and caps at 60 seconds', () => {
  assert.equal(calculateReconnectDelay({ attempt: 1, random: () => 0.5 }), 3000)
  assert.equal(calculateReconnectDelay({ attempt: 2, random: () => 0.5 }), 6000)
  assert.equal(calculateReconnectDelay({ attempt: 6, random: () => 0.5 }), 60000)
  assert.equal(calculateReconnectDelay({ attempt: 6, random: () => 1 }), 60000)
  assert.equal(calculateReconnectDelay({ attempt: 3, random: () => 0 }), 9600)
  assert.equal(calculateReconnectDelay({ attempt: 3, random: () => 1 }), 14400)
})

test('websocket reconnect policy stops for lifecycle and auth terminal conditions only', () => {
  assert.deepEqual(shouldReconnectWebSocket({}), { allow: true, reason: 'recoverable' })
  assert.deepEqual(shouldReconnectWebSocket({ manualDisconnect: true }), { allow: false, reason: 'manual-disconnect' })
  assert.deepEqual(shouldReconnectWebSocket({ foreground: false }), { allow: false, reason: 'background' })
  assert.deepEqual(shouldReconnectWebSocket({ online: false }), { allow: false, reason: 'offline' })
  assert.deepEqual(shouldReconnectWebSocket({ sessionInvalid: true }), { allow: false, reason: 'session-invalid' })
  assert.deepEqual(shouldReconnectWebSocket({ permissionDenied: true }), { allow: false, reason: 'permission-denied' })
  assert.deepEqual(shouldReconnectWebSocket({ targetActive: false }), { allow: false, reason: 'target-inactive' })
})

test('websocket ticket errors preserve session and permission terminal boundaries', () => {
  assert.equal(classifyWebSocketFailure({ category: 'session', statusCode: 401 }), 'session-invalid')
  assert.equal(classifyWebSocketFailure({ category: 'forbidden', statusCode: 403 }), 'permission-denied')
  assert.equal(classifyWebSocketFailure({ category: 'network' }), 'recoverable')
  assert.equal(classifyWebSocketFailure({ statusCode: 503 }), 'recoverable')
  assert.deepEqual(
    shouldReconnectWebSocket({ error: { responseData: { reason: 'SESSION_INVALID' } } }),
    { allow: false, reason: 'session-invalid' }
  )
})

test('websocket connection generation rejects obsolete callbacks', () => {
  const first = nextConnectionGeneration()
  const second = nextConnectionGeneration(first)
  assert.equal(isCurrentConnectionGeneration(second, first), false)
  assert.equal(isCurrentConnectionGeneration(second, second), true)
})
