import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const manifestSource = readFileSync(new URL('../manifest.json', import.meta.url), 'utf8')

test('WeChat manifest declares the getLocation private API', () => {
  assert.match(manifestSource, /"requiredPrivateInfos"\s*:\s*\[[\s\S]*?"getLocation"[\s\S]*?\]/)
})

test('WeChat manifest describes why user location is requested', () => {
  assert.match(manifestSource, /"scope\.userLocation"\s*:\s*\{[\s\S]*?"desc"\s*:\s*"[^"]+"/)
})
