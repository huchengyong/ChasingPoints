import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const statsDetailSource = readFileSync(
  new URL('../subPages/user/statsDetail.vue', import.meta.url),
  'utf8'
)

test('stats detail page redirects guests straight to login instead of rendering a login-required placeholder', () => {
  assert.doesNotMatch(statsDetailSource, /login-required/)
  assert.match(statsDetailSource, /const token = uni\.getStorageSync\('token'\)/)
  assert.match(statsDetailSource, /if \(!token\) \{\s*goLogin\(\)\s*return/s)
  assert.match(statsDetailSource, /const goLogin = \(\) => \{\s*uni\.navigateTo\(\{ url: '\/pages\/login\/login' \}\)/s)
})

test('stats detail page only loads stats after passing the login guard', () => {
  assert.match(statsDetailSource, /onLoad\(\(options\) => \{[\s\S]*currentGame\.value = gameTypeKeyMap\[gameType\][\s\S]*\}\)\s*\n\s*\n\s*onShow/s)
  assert.match(statsDetailSource, /onShow\([\s\S]*if \(!token\) \{\s*goLogin\(\)\s*return[\s\S]*loadAllStats\(\)/s)
})
