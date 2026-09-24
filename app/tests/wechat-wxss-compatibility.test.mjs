import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const appSource = readFileSync(new URL('../App.vue', import.meta.url), 'utf8')

test('WeChat styles exclude the unsupported universal selector', () => {
  const wechatSource = appSource.replace(
    /\/\* #ifndef MP-WEIXIN \*\/[\s\S]*?\/\* #endif \*\//g,
    ''
  )

  assert.doesNotMatch(wechatSource, /^\s*\*,\s*$/m)
})
