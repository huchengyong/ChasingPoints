import test from 'node:test'
import assert from 'node:assert/strict'

import {
  inferWsBaseUrl,
  resolveNetworkConfig
} from '../utils/runtime-config.js'

test('resolveNetworkConfig uses tunnel host in development', () => {
  assert.deepEqual(resolveNetworkConfig({ env: 'development' }), {
    httpBaseUrl: 'https://api-tunnel2.kekemate.com',
    wsBaseUrl: 'wss://api-tunnel2.kekemate.com'
  })
})

test('resolveNetworkConfig uses production host in production', () => {
  assert.deepEqual(resolveNetworkConfig({ env: 'production' }), {
    httpBaseUrl: 'https://api-bm.dianzaozao.com',
    wsBaseUrl: 'wss://api-bm.dianzaozao.com'
  })
})

test('resolveNetworkConfig lets explicit overrides win', () => {
  assert.deepEqual(resolveNetworkConfig({
    env: 'production',
    override: {
      httpBaseUrl: 'https://staging.example.com',
      wsBaseUrl: 'wss://socket.example.com'
    }
  }), {
    httpBaseUrl: 'https://staging.example.com',
    wsBaseUrl: 'wss://socket.example.com'
  })
})

test('inferWsBaseUrl converts http protocols to websocket protocols', () => {
  assert.equal(inferWsBaseUrl('https://api-bm.dianzaozao.com'), 'wss://api-bm.dianzaozao.com')
  assert.equal(inferWsBaseUrl('http://localhost:8080'), 'ws://localhost:8080')
})
