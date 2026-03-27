import test from 'node:test'
import assert from 'node:assert/strict'

import {
  pickEventNewsDetailPayload,
  pickFeaturedEventNewsPayload
} from '../utils/event-news-response.js'

test('pickFeaturedEventNewsPayload only accepts the canonical event field', () => {
  assert.deepEqual(pickFeaturedEventNewsPayload({
    success: true,
    event: { id: 12, title: '焦点赛事' },
    eventNews: { id: 34, title: '旧字段' },
    event_news: { id: 56, title: '旧下划线字段' }
  }), { id: 12, title: '焦点赛事' })
})

test('pickFeaturedEventNewsPayload ignores legacy mirror-only responses', () => {
  assert.equal(pickFeaturedEventNewsPayload({
    success: true,
    eventNews: { id: 34, title: '旧字段' }
  }), null)
})

test('pickEventNewsDetailPayload only accepts the canonical event field', () => {
  assert.deepEqual(pickEventNewsDetailPayload({
    success: true,
    event: { id: 18, title: '赛事详情' },
    eventNews: { id: 99, title: '旧字段' }
  }), { id: 18, title: '赛事详情' })
})

test('pickEventNewsDetailPayload ignores legacy detail mirrors', () => {
  assert.equal(pickEventNewsDetailPayload({
    success: true,
    eventNews: { id: 99, title: '旧字段' }
  }), null)
})
