import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../pages/user/index.vue', import.meta.url),
  'utf8'
)
const styleSource = readFileSync(
  new URL('../pages/user/index.scss', import.meta.url),
  'utf8'
)

test('user page guest hero keeps login as primary action and start PK as secondary action', () => {
  assert.match(source, /hero-btn hero-btn-primary" @click="handleGoLogin"/)
  assert.match(source, /hero-btn hero-btn-secondary" @click="handleGuestStartPK"/)
})

test('user page only renders the status action row when there are visible actions', () => {
  assert.match(
    source,
    /<view v-if="showPrimaryStatusAction \|\| showSecondaryStatusAction" class="status-actions">/
  )
})

test('user page rank chip renders the computed highest rank label instead of hardcoded copy', () => {
  assert.match(source, /<text>{{ highestRankChipText }}<\/text>/)
  assert.doesNotMatch(source, /<text>当前段位<\/text>/)
})

test('user page status eyebrow uses readable muted text on the light status card', () => {
  assert.doesNotMatch(
    styleSource,
    /\.guest-eyebrow,\s*\.status-eyebrow\s*\{[\s\S]*color:\s*rgba\(255,\s*255,\s*255,\s*0\.74\);/
  )
  assert.match(
    styleSource,
    /\.status-eyebrow\s*\{[\s\S]*color:\s*\$text-muted-light;/
  )
})
