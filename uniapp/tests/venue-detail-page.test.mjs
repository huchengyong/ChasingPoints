import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../subPages/venue/detail.vue', import.meta.url),
  'utf8'
)

test('venue detail page reads route params from onLoad instead of current page internals', () => {
  assert.match(source, /import\s*\{\s*onLoad\s*\}\s*from\s*'@dcloudio\/uni-app'/)
  assert.match(source, /onLoad\(\(options\)\s*=>\s*\{[\s\S]*venueId\.value\s*=\s*Number\.parseInt\(options\?\.(id|venue_id)/)
  assert.doesNotMatch(source, /getCurrentPages\(\)/)
  assert.doesNotMatch(source, /currentPage\.options\.id/)
})
