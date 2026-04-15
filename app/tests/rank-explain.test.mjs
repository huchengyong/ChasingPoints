import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveRankExplainLoadingMode,
  shouldApplyRankExplainResponse
} from '../utils/rank-explain.js'

test('resolveRankExplainLoadingMode only uses full-page loading before first successful render', () => {
  assert.equal(resolveRankExplainLoadingMode({
    hasLoadedOnce: false,
    isFetching: true
  }), 'initial')

  assert.equal(resolveRankExplainLoadingMode({
    hasLoadedOnce: true,
    isFetching: true
  }), 'refreshing')

  assert.equal(resolveRankExplainLoadingMode({
    hasLoadedOnce: true,
    isFetching: false
  }), 'idle')
})

test('shouldApplyRankExplainResponse ignores stale tab-switch responses', () => {
  assert.equal(shouldApplyRankExplainResponse({
    requestId: 2,
    latestRequestId: 3
  }), false)

  assert.equal(shouldApplyRankExplainResponse({
    requestId: 3,
    latestRequestId: 3
  }), true)
})
