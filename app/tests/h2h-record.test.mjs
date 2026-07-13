import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildH2HHistoryParams,
  buildH2HCurrentMonthKey,
  buildH2HMonthDateRange,
  shiftH2HMonthKey,
  buildH2HLoadFailureAction,
  buildH2HViewModel,
  normalizeH2HRecordOptions,
  shouldApplyH2HHistoryResponse
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

test('buildH2HHistoryParams includes optional date range for calendar requests', () => {
  assert.deepEqual(
    buildH2HHistoryParams({
      opponentId: 18,
      page: 1,
      pageSize: 100,
      startDate: '2026-04-01',
      endDate: '2026-04-30'
    }),
    {
      opponent_id: 18,
      page: 1,
      page_size: 100,
      start_date: '2026-04-01',
      end_date: '2026-04-30'
    }
  )
})

test('buildH2HMonthDateRange returns the inclusive month boundary dates', () => {
  assert.deepEqual(
    buildH2HMonthDateRange('2026-04'),
    {
      startDate: '2026-04-01',
      endDate: '2026-04-30'
    }
  )
})

test('buildH2HCurrentMonthKey returns the local calendar month', () => {
  assert.equal(buildH2HCurrentMonthKey(new Date(2026, 3, 30, 10, 0, 0)), '2026-04')
})

test('shiftH2HMonthKey moves across year boundaries', () => {
  assert.equal(shiftH2HMonthKey('2026-01', -1), '2025-12')
  assert.equal(shiftH2HMonthKey('2026-12', 1), '2027-01')
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
      navigationTitle: '球友 B的交锋记录',
      subjectName: '球友 B',
      winRateLabel: '球友 B 对阿杰的胜率是'
    }
  )
})

test('buildH2HViewModel exposes a current-user opponent marker in friend detail mode', () => {
  assert.deepEqual(
    buildH2HViewModel({
      targetUserId: 52,
      targetName: '球友 B',
      opponentId: 18,
      opponentName: '我自己',
      currentUserId: 18
    }),
    {
      isTargetMode: true,
      navigationTitle: '球友 B的交锋记录',
      subjectName: '球友 B',
      winRateLabel: '球友 B 对我自己的胜率是',
      isOpponentMe: true
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

test('shouldApplyH2HHistoryResponse ignores stale filter responses', () => {
  assert.equal(shouldApplyH2HHistoryResponse({
    requestId: 4,
    latestRequestId: 5
  }), false)

  assert.equal(shouldApplyH2HHistoryResponse({
    requestId: 5,
    latestRequestId: 5
  }), true)
})

test('h2h record page handles target access failures with an explicit navigateBack flow', () => {
  const source = readFileSync(new URL('../subPages/user/h2hRecord.vue', import.meta.url), 'utf8')

  assert.match(source, /buildH2HLoadFailureAction/)
  assert.match(source, /uni\.navigateBack/)
})

test('h2h record page uses view models, exposes retry copy, and links cards to match detail', () => {
  const source = readFileSync(new URL('../subPages/user/h2hRecord.vue', import.meta.url), 'utf8')

  assert.match(source, /buildH2HHeroViewModel/)
  assert.match(source, /buildH2HCalendarViewModel/)
  assert.match(source, /shouldShowH2HSummaryCard/)
  assert.match(source, /matchDetail\?match_id=\$\{matchId\}&source=history&perspective_user_id=\$\{subjectUserId\}/)
  assert.match(source, /重新加载/)
})

test('h2h record page merges calendar and list into one month-filtered history section', () => {
  const source = readFileSync(new URL('../subPages/user/h2hRecord.vue', import.meta.url), 'utf8')
  const style = readFileSync(new URL('../subPages/user/h2hRecord.scss', import.meta.url), 'utf8')

  assert.match(source, /<text class="history-section-title">比赛历史<\/text>/)
  assert.match(source, /<view class="history-section-header">[\s\S]*<view class="history-month-row">[\s\S]*<view class="history-view-switch">/)
  assert.match(source, /historyViewMode/)
  assert.match(source, /shiftHistoryMonth/)
  assert.match(source, /pageSize = 10/)
  assert.doesNotMatch(source, />全部</)
  assert.doesNotMatch(source, />胜利</)
  assert.doesNotMatch(source, />失败</)
  assert.doesNotMatch(source, />最近5场</)
  assert.match(source, /calendarViewModel\.weekdays/)
  assert.match(source, /cell\.winCount/)
  assert.match(source, /cell\.lossCount/)
  assert.doesNotMatch(source, /<view class="h2h-calendar-card"[\s\S]*?<view class="empty-wrapper compact" v-if="!hasCalendarMatches">/)
  assert.match(style, /\.history-view-switch/)
  assert.match(style, /\.history-month-button/)
  assert.match(style, /\.calendar-result-dot[\s\S]*&\.win[\s\S]*background-color: #22c55e;/)
  assert.match(style, /\.calendar-result-dot[\s\S]*&\.lose[\s\S]*background-color: #ef4444;/)
})
