import test from 'node:test'
import assert from 'node:assert/strict'

import { pickEventNewsViewPayload } from '../utils/event-news-response.js'

test('pickEventNewsViewPayload only accepts the canonical event_news field', () => {
  assert.deepEqual(pickEventNewsViewPayload({
    success: true,
    event_news: { id: 18, title: '赛事详情' },
    tournament: { id: 3, name: '赛事实体' },
    matches: [{ id: 88 }]
  }), {
    eventNews: { id: 18, title: '赛事详情' },
    tournament: { id: 3, name: '赛事实体' },
    matches: [{ id: 88 }]
  })
})

test('pickEventNewsViewPayload ignores missing event_news payloads', () => {
  assert.equal(pickEventNewsViewPayload({
    success: true,
    event: { id: 99, title: '旧字段' }
  }), null)
})
