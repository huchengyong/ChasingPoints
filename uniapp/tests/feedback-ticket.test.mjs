import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildFeedbackTicketPayload,
  FEEDBACK_CATEGORY_OPTIONS,
  normalizeFeedbackCategory
} from '../utils/feedback-ticket.js'

test('normalizeFeedbackCategory falls back to feedback for unknown values', () => {
  assert.equal(normalizeFeedbackCategory('complaint'), 'complaint')
  assert.equal(normalizeFeedbackCategory('report'), 'report')
  assert.equal(normalizeFeedbackCategory('unknown'), 'feedback')
  assert.equal(normalizeFeedbackCategory(''), 'feedback')
})

test('buildFeedbackTicketPayload trims content and contact for app submissions', () => {
  assert.deepEqual(
    buildFeedbackTicketPayload({
      category: 'complaint',
      content: '  对处理结果有异议  ',
      contact: '  13800000000  '
    }),
    {
      source: 'app',
      category: 'complaint',
      content: '对处理结果有异议',
      contact: '13800000000'
    }
  )
})

test('feedback category options explicitly cover complaints and reports', () => {
  assert.deepEqual(
    FEEDBACK_CATEGORY_OPTIONS.map((item) => item.value),
    ['feedback', 'complaint', 'report']
  )
  assert.ok(FEEDBACK_CATEGORY_OPTIONS.some((item) => item.label.includes('投诉')))
  assert.ok(FEEDBACK_CATEGORY_OPTIONS.some((item) => item.label.includes('举报')))
})
