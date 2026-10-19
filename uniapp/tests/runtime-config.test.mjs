import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'

import {
  resolveNetworkConfig
} from '../utils/runtime-config.js'

const runtimeConfigSource = readFileSync(new URL('../utils/runtime-config.js', import.meta.url), 'utf8')

test('runtime environment uses the NODE_ENV expression recognized by the UniApp compiler', () => {
  assert.match(runtimeConfigSource, /process\.env\.NODE_ENV/)
})

test('resolveNetworkConfig uses development hosts', () => {
  assert.deepEqual(resolveNetworkConfig({ env: 'development' }), {
    httpBaseUrl: 'https://api-zhuifen.kekemate.cn',
    wsBaseUrl: 'wss://api-zhuifen.kekemate.cn'
  })
})

test('resolveNetworkConfig uses production host in production', () => {
  assert.deepEqual(resolveNetworkConfig({ env: 'production' }), {
    httpBaseUrl: 'https://api.zhuifen.cn',
    wsBaseUrl: 'wss://ws.zhuifen.cn'
  })
})

// 模拟小程序/App 构建产物：编译器替换 NODE_ENV，运行环境中不存在 process 全局。
const runCompiledRuntimeConfig = (nodeEnv) => {
  const compiled = runtimeConfigSource
    .replace(/process\.env\.NODE_ENV/g, JSON.stringify(nodeEnv))
    .replace(/^export /gm, '')
    .concat('\nresult = NETWORK_CONFIG\n')
  const sandbox = { result: null }
  runInNewContext(compiled, sandbox)
  // 沙箱内创建的对象原型不同，转成普通对象再比较
  return JSON.parse(JSON.stringify(sandbox.result))
}

test('production bundle without a process global still uses the production hosts', () => {
  assert.deepEqual(runCompiledRuntimeConfig('production'), {
    httpBaseUrl: 'https://api.zhuifen.cn',
    wsBaseUrl: 'wss://ws.zhuifen.cn'
  })
})

test('development bundle without a process global keeps the development hosts', () => {
  assert.deepEqual(runCompiledRuntimeConfig('development'), {
    httpBaseUrl: 'https://api-zhuifen.kekemate.cn',
    wsBaseUrl: 'wss://api-zhuifen.kekemate.cn'
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

