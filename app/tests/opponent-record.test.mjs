import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildOpponentH2HUrl,
  buildOpponentRecordRequestParams,
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
