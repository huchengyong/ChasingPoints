import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('notification list uses unread_count from its own response without an immediate count request', () => {
  const source = readFileSync(new URL('../pages/message/index.vue', import.meta.url), 'utf8')
  assert.match(source, /notificationStore\.setUnreadCount\(res\.unread_count, getReadIdentity\(\)\)/)
  assert.doesNotMatch(source, /notificationStore\.fetchUnreadCount\(\)/)
})
