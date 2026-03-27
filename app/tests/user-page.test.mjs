import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../pages/user/index.vue', import.meta.url),
  'utf8'
)

test('user page guest hero keeps login as primary action and start PK as secondary action', () => {
  assert.match(source, /hero-btn hero-btn-primary" @click="handleGoLogin"/)
  assert.match(source, /hero-btn hero-btn-secondary" @click="handleGuestStartPK"/)
})
