import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  resolveNetworkConfig
} from '../utils/runtime-config.js'

const runtimeConfigSource = readFileSync(new URL('../utils/runtime-config.js', import.meta.url), 'utf8')

test('runtime environment uses the NODE_ENV expression recognized by the UniApp compiler', () => {
  assert.match(runtimeConfigSource, /process\.env\.NODE_ENV/)
})

test('resolveNetworkConfig uses development hosts', () => {
  assert.deepEqual(resolveNetworkConfig({ env: 'development' }), {
    httpBaseUrl: 'https://dev-api.kekemate.cn',
    wsBaseUrl: 'wss://dev-api.kekemate.cn'
  })
})

test('resolveNetworkConfig uses production host in production', () => {
  assert.deepEqual(resolveNetworkConfig({ env: 'production' }), {
    httpBaseUrl: 'https://api.zhuifen.cn',
    wsBaseUrl: 'wss://ws.zhuifen.cn'
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

