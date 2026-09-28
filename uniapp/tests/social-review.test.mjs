import test from 'node:test'
import assert from 'node:assert/strict'

import { buildPostReviewSuccessCopy, resolveMyPostReviewMeta } from '../utils/social-review.js'

test('resolveMyPostReviewMeta returns pending copy for pending posts', () => {
  assert.deepEqual(resolveMyPostReviewMeta(2), {
    tagText: '审核中，仅自己可见',
    tagType: 'pending',
    reasonVisible: false,
    reasonText: ''
  })
})

test('resolveMyPostReviewMeta shows reject reason for rejected posts', () => {
  assert.deepEqual(resolveMyPostReviewMeta(3, '包含辱骂内容'), {
    tagText: '审核未通过',
    tagType: 'rejected',
    reasonVisible: true,
    reasonText: '包含辱骂内容'
  })
})

test('resolveMyPostReviewMeta treats published posts as public', () => {
  assert.deepEqual(resolveMyPostReviewMeta(1), {
    tagText: '已发布',
    tagType: 'published',
    reasonVisible: false,
    reasonText: ''
  })
})

test('buildPostReviewSuccessCopy points users to my posts', () => {
  assert.deepEqual(buildPostReviewSuccessCopy(), {
    toast: '已提交审核',
    hint: '可在“我的动态”查看进度'
  })
})
