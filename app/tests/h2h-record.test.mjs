import test from 'node:test'
import assert from 'node:assert/strict'

import { buildH2HHistoryParams } from '../utils/h2h-record.js'

test('buildH2HHistoryParams prefers opponent_id when available', () => {
  assert.deepEqual(
    buildH2HHistoryParams({
      opponentId: 18,
      fallbackOpponentId: 0,
      opponentName: '球友 A',
      page: 2,
      pageSize: 20,
      result: 1
    }),
    {
      opponent_id: 18,
      page: 2,
      page_size: 20,
      result: 1
    }
  )
})

test('buildH2HHistoryParams falls back to opponent_name for anonymous opponents', () => {
  assert.deepEqual(
    buildH2HHistoryParams({
      opponentId: 0,
      fallbackOpponentId: 0,
      opponentName: '线下朋友',
      page: 1,
      pageSize: 20,
      result: 0
    }),
    {
      opponent_name: '线下朋友',
      page: 1,
      page_size: 20
    }
  )
})
