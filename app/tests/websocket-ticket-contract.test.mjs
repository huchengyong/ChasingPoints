import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const matchApiSource = readFileSync(new URL('../api/match.js', import.meta.url), 'utf8')
const userApiSource = readFileSync(new URL('../api/user.js', import.meta.url), 'utf8')
const websocketSource = readFileSync(new URL('../utils/websocket.js', import.meta.url), 'utf8')

test('app API exposes user and match WebSocket ticket facades through the request layer', () => {
  assert.match(userApiSource, /export const issueUserWSTicket = \(\) => \{[\s\S]*?post\('\/api\/user\/ws-ticket'\)/)
  assert.match(userApiSource, /issueUserWSTicket/)
  assert.match(matchApiSource, /export const issueMatchWSTicket = \(matchId\) => \{[\s\S]*?post\('\/api\/match\/ws-ticket', \{ match_id: Number\(matchId\) \}\)/)
})

test('websocket client source does not embed long-lived access tokens in query strings', () => {
  assert.doesNotMatch(websocketSource, /token=\$\{?token/)
  assert.doesNotMatch(websocketSource, /[?&]token=/)
})
