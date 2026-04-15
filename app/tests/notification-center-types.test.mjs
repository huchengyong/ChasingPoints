import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const source = readFileSync(
  new URL('../subPages/notification/index.vue', import.meta.url),
  'utf8'
)

test('notification center maps all real backend notification types with explicit icons and styles', () => {
  assert.match(source, /challenge:\s*'🎯'/)
  assert.match(source, /tournament:\s*'🏆'/)
  assert.match(source, /friend_request:\s*'👥'/)
  assert.match(source, /follow:\s*'⭐'/)
  assert.match(source, /match_result:\s*'🏁'/)
  assert.match(source, /return map\[type\] \|\| '🔔'/)
  assert.match(source, /&\.type-follow\s*\{/)
  assert.match(source, /&\.type-match_result\s*\{/)
})
