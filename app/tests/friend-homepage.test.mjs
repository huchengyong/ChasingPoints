import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildFriendHomepageSummary,
  fetchFriendHomepageData,
  normalizeFriendHomepageOptions
} from '../utils/friend-homepage.js'

test('normalizeFriendHomepageOptions decodes route params into stable display fields', () => {
  assert.deepEqual(
    normalizeFriendHomepageOptions({
      opponent_id: '52',
      opponent_name: encodeURIComponent('球友 A'),
      opponent_avatar: encodeURIComponent('https://img.example/a.png'),
      rank_name: encodeURIComponent('超凡')
    }),
    {
      opponentId: 52,
      opponentName: '球友 A',
      opponentAvatar: 'https://img.example/a.png',
      rankName: '超凡'
    }
  )
})

test('buildFriendHomepageSummary exposes readable stat copy for matched friends', () => {
  assert.deepEqual(
    buildFriendHomepageSummary({
      stats: {
        total_matches: 12,
        my_wins: 7,
        opponent_wins: 5
      },
      lastMatchAt: '2026-03-20T12:00:00.000Z',
      formatRelativeTime: () => '3天前'
    }),
    {
      hasMatches: true,
      heroTitle: '你暂时领先',
      heroDesc: '你们已经认真交手 12 场，你暂时领先。',
      totalMatchesText: '12 场交手',
      recordText: '你 7 胜 5 负',
      recentMatchText: '最近一次交手 3天前',
      emptyTitle: '',
      emptyDesc: ''
    }
  )
})

test('buildFriendHomepageSummary falls back to a calm empty state when no matches exist', () => {
  assert.deepEqual(
    buildFriendHomepageSummary({
      stats: {
        total_matches: 0,
        my_wins: 0,
        opponent_wins: 0
      },
      lastMatchAt: '',
      formatRelativeTime: () => ''
    }),
    {
      hasMatches: false,
      heroTitle: '你们还没正式交手',
      heroDesc: '先约一场真实对局，这里就会慢慢留下你们的战绩。',
      totalMatchesText: '0 场交手',
      recordText: '你 0 胜 0 负',
      recentMatchText: '最近一次交手 暂无记录',
      emptyTitle: '你们还没有正式交手',
      emptyDesc: '先积累一场真实对局，再来看战绩和 PK 报表'
    }
  )
})

test('buildFriendHomepageSummary uses a balanced headline for tied records', () => {
  assert.deepEqual(
    buildFriendHomepageSummary({
      stats: {
        total_matches: 6,
        my_wins: 3,
        opponent_wins: 3
      },
      lastMatchAt: '2026-03-20T12:00:00.000Z',
      formatRelativeTime: () => '昨天'
    }),
    {
      hasMatches: true,
      heroTitle: '你们目前势均力敌',
      heroDesc: '你们最近打得很胶着，暂时谁也没拉开差距。',
      totalMatchesText: '6 场交手',
      recordText: '你 3 胜 3 负',
      recentMatchText: '最近一次交手 昨天',
      emptyTitle: '',
      emptyDesc: ''
    }
  )
})

test('buildFriendHomepageSummary uses a hopeful line when the friend is ahead', () => {
  assert.deepEqual(
    buildFriendHomepageSummary({
      stats: {
        total_matches: 9,
        my_wins: 4,
        opponent_wins: 5
      },
      lastMatchAt: '2026-03-20T12:00:00.000Z',
      formatRelativeTime: () => '前天'
    }),
    {
      hasMatches: true,
      heroTitle: '他暂时领先',
      heroDesc: '这位球友最近手感更好一点，你还有机会下一场追回来。',
      totalMatchesText: '9 场交手',
      recordText: '你 4 胜 5 负',
      recentMatchText: '最近一次交手 前天',
      emptyTitle: '',
      emptyDesc: ''
    }
  )
})

test('fetchFriendHomepageData requests stats and the first history page together', async () => {
  const calls = []
  const result = await fetchFriendHomepageData({
    params: { opponent_id: 18 },
    getStats: async (params) => {
      calls.push(['stats', params])
      return { stats: { total_matches: 4 } }
    },
    getHistory: async (params) => {
      calls.push(['history', params])
      return { list: [{ match_time: '2026-03-20T12:00:00.000Z' }] }
    }
  })

  assert.deepEqual(calls, [
    ['stats', { opponent_id: 18 }],
    ['history', { opponent_id: 18, page: 1, page_size: 1 }]
  ])
  assert.deepEqual(result, {
    statsRes: { stats: { total_matches: 4 } },
    historyRes: { list: [{ match_time: '2026-03-20T12:00:00.000Z' }] }
  })
})

test('fetchFriendHomepageData rejects when history request fails so callers can show an error state', async () => {
  await assert.rejects(
    fetchFriendHomepageData({
      params: { opponent_id: 18 },
      getStats: async () => ({ stats: { total_matches: 4 } }),
      getHistory: async () => {
        throw new Error('history failed')
      }
    }),
    /history failed/
  )
})

test('friend homepage template keeps a dedicated loadFailed branch instead of falling through to empty stats', () => {
  const source = readFileSync(new URL('../subPages/social/friendHomepage.vue', import.meta.url), 'utf8')

  assert.match(source, /v-else-if="loadFailed"/)
  assert.match(source, /重新加载/)
})
