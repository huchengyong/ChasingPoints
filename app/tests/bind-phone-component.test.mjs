import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../components/bindPhone.vue', import.meta.url),
  'utf8'
)

test('bind phone button keeps an explicit active style in dark mode', () => {
  assert.match(
    source,
    /&\.dark-mode\s*\{[\s\S]*?\.bind-btn\s*\{[\s\S]*?&\.active\s*\{/,
    'dark mode should provide an active button style override'
  )
})

test('bind phone button uses explicit centering styles for its label', () => {
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?display:\s*flex;/)
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?align-items:\s*center;/)
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?justify-content:\s*center;/)
  assert.match(source, /\.bind-btn\s*\{[\s\S]*?padding:\s*0;/)
})
