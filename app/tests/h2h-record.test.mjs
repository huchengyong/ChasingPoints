import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildH2HHistoryParams,
  buildH2HLoadFailureAction,
  buildH2HViewModel,
  normalizeH2HRecordOptions
} from '../utils/h2h-record.js'

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

test('buildH2HHistoryParams carries target_user_id in friend mode', () => {
  assert.deepEqual(
    buildH2HHistoryParams({
      targetUserId: 52,
      opponentId: 18,
      fallbackOpponentId: 0,
      opponentName: '球友 A',
      page: 1,
      pageSize: 20,
      result: 2
    }),
    {
      target_user_id: 52,
      opponent_id: 18,
      page: 1,
      page_size: 20,
      result: 2
    }
  )
})

test('normalizeH2HRecordOptions decodes both target user and opponent context', () => {
  assert.deepEqual(
    normalizeH2HRecordOptions({
      target_user_id: '52',
      target_name: encodeURIComponent('球友 B'),
      target_avatar: encodeURIComponent('https://img.example/b.png'),
      opponent_id: '18',
      opponent_name: encodeURIComponent('阿杰')
    }),
    {
      targetUserId: 52,
      targetName: '球友 B',
      targetAvatar: 'https://img.example/b.png',
      opponentId: 18,
      opponentName: '阿杰'
    }
  )
})

test('buildH2HViewModel keeps current self-vs-opponent copy by default', () => {
  assert.deepEqual(
    buildH2HViewModel({
      targetUserId: 0,
      targetName: '',
      opponentName: '阿杰'
    }),
    {
      isTargetMode: false,
      navigationTitle: '交锋记录 - 阿杰',
      subjectName: '你',
      winRateLabel: '你对阿杰的胜率是'
    }
  )
})

test('buildH2HViewModel switches to friend-vs-opponent copy in target mode', () => {
  assert.deepEqual(
    buildH2HViewModel({
      targetUserId: 52,
      targetName: '球友 B',
      opponentName: '阿杰'
    }),
    {
      isTargetMode: true,
      navigationTitle: '战绩详情 - 球友 B',
      subjectName: '球友 B',
      winRateLabel: '球友 B 对阿杰的胜率是'
    }
  )
})

test('buildH2HLoadFailureAction requests a toast and navigateBack for target access failures', () => {
  assert.deepEqual(
    buildH2HLoadFailureAction({
      targetUserId: 52,
      error: new Error('仅可查看好友的对方战绩')
    }),
    {
      shouldNavigateBack: true,
      toastMessage: '仅可查看好友的对方战绩'
    }
  )
})

test('buildH2HLoadFailureAction ignores generic self-mode failures', () => {
  assert.equal(
    buildH2HLoadFailureAction({
      targetUserId: 0,
      error: new Error('网络请求失败')
    }),
    null
  )
})

test('h2h record page handles target access failures with an explicit navigateBack flow', () => {
  const source = readFileSync(new URL('../subPages/user/h2hRecord.vue', import.meta.url), 'utf8')

  assert.match(source, /buildH2HLoadFailureAction/)
  assert.match(source, /uni\.navigateBack/)
})

test('h2h record page uses view models, exposes retry copy, and links cards to match detail', () => {
  const source = readFileSync(new URL('../subPages/user/h2hRecord.vue', import.meta.url), 'utf8')

  assert.match(source, /buildH2HHeroViewModel/)
  assert.match(source, /shouldShowH2HSummaryCard/)
  assert.match(source, /matchDetail\?match_id=\$\{matchId\}/)
  assert.match(source, /重新加载/)
})
