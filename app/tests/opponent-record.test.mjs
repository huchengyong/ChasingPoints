import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildOpponentCardViewModels,
  buildOpponentH2HUrl,
  buildOpponentRecordRequestParams,
  buildOpponentStatsSummary,
  buildOpponentRecordViewModel,
  normalizeOpponentRecordOptions
} from '../utils/opponent-record.js'

test('normalizeOpponentRecordOptions decodes target user context from route params', () => {
  assert.deepEqual(
    normalizeOpponentRecordOptions({
      target_user_id: '52',
      target_name: encodeURIComponent('球友 B'),
      target_avatar: encodeURIComponent('https://img.example/b.png')
    }),
    {
      targetUserId: 52,
      targetName: '球友 B',
      targetAvatar: 'https://img.example/b.png'
    }
  )
})

test('buildOpponentRecordRequestParams carries target_user_id in friend mode', () => {
  assert.deepEqual(
    buildOpponentRecordRequestParams({
      page: 2,
      pageSize: 20,
      keyword: '阿杰',
      targetUserId: 52
    }),
    {
      page: 2,
      page_size: 20,
      keyword: '阿杰',
      target_user_id: 52
    }
  )
})

test('buildOpponentRecordViewModel keeps current copy for self mode', () => {
  assert.deepEqual(
    buildOpponentRecordViewModel({
      targetName: '',
      targetUserId: 0
    }),
    {
      isTargetMode: false,
      subjectName: '你',
      navigationTitle: '对手记录',
      loginHint: '请登录后查看对手记录',
      searchPlaceholder: '搜索对手',
      emptyText: '暂无对手记录',
      emptyHint: '快去发起一场PK吧！'
    }
  )
})

test('buildOpponentRecordViewModel switches to friend-focused copy in target mode', () => {
  assert.deepEqual(
    buildOpponentRecordViewModel({
      targetName: '球友 B',
      targetUserId: 52
    }),
    {
      isTargetMode: true,
      subjectName: '球友 B',
      navigationTitle: '对方战绩',
      loginHint: '请登录后查看对方战绩',
      searchPlaceholder: '搜索 TA 的对手',
      emptyText: '暂无对方战绩',
      emptyHint: '还没看到球友 B 的历史对手记录'
    }
  )
})

test('buildOpponentH2HUrl keeps target user context when opening friend battle detail', () => {
  assert.equal(
    buildOpponentH2HUrl({
      opponent: {
        id: 18,
        name: '阿杰'
      },
      targetUserId: 52,
      targetName: '球友 B',
      targetAvatar: 'https://img.example/b.png'
    }),
    '/subPages/user/h2hRecord?target_user_id=52&target_name=%E7%90%83%E5%8F%8B%20B&target_avatar=https%3A%2F%2Fimg.example%2Fb.png&opponent_id=18&opponent_name=%E9%98%BF%E6%9D%B0'
  )
})

test('buildOpponentStatsSummary explains how to use the list before drilling down', () => {
  assert.deepEqual(
    buildOpponentStatsSummary({
      totalOpponents: 6,
      totalWins: 14,
      targetName: '',
      targetUserId: 0
    }),
    {
      title: '你的对手档案',
      subtitle: '已经形成 6 个对手样本，累计拿下 14 场胜利'
    }
  )
})

test('buildOpponentCardViewModels converts raw records into replay-friendly cards', () => {
  const [card] = buildOpponentCardViewModels({
    subjectName: '你',
    opponents: [{
      id: 18,
      name: '阿杰',
      wins: 7,
      losses: 5,
      win_rate: 58.33,
      last_match_at: '2026-03-27 18:30:00'
    }]
  })

  assert.equal(card.relationshipText, '你略占上风')
  assert.equal(card.relationshipBadge, '领先')
  assert.equal(card.recordText, '总交锋 7 胜 5 负')
  assert.equal(card.winRateText, '58%')
})

test('buildOpponentCardViewModels marks the current user in friend opponent lists and labels win rate', () => {
  const [card] = buildOpponentCardViewModels({
    subjectName: '球友 B',
    currentUserId: 18,
    showWinRateLabel: true,
    opponents: [{
      id: 18,
      name: '我自己',
      wins: 3,
      losses: 2,
      win_rate: 60,
      last_match_at: '2026-03-27 18:30:00'
    }]
  })

  assert.equal(card.isMeOpponent, true)
  assert.equal(card.winRateText, '60% 胜率')
})

test('buildOpponentCardViewModels uses backend total_matches as the sample source of truth', () => {
  const [card] = buildOpponentCardViewModels({
    subjectName: '你',
    opponents: [{
      id: 20,
      name: '阿峰',
      total_matches: 6,
      wins: 4,
      losses: 0,
      win_rate: 100,
      last_match_at: '2026-03-27 18:30:00'
    }]
  })

  assert.equal(card.sampleText, '6 场交锋')
  assert.equal(card.relationshipText, '你明显占优')
})

test('opponent record page keeps search state on refresh and uses card view models', () => {
  const source = readFileSync(new URL('../subPages/user/opponentRecord.vue', import.meta.url), 'utf8')

  assert.match(source, /buildOpponentCardViewModels/)
  assert.match(source, /buildOpponentStatsSummary/)
  assert.match(source, /retry-btn/)
  assert.doesNotMatch(source, /searchKeyword\.value = ''/)
})
