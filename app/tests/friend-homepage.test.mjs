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
      heroTitle: '这位球友最近有 12 场真实对局',
      heroDesc: '可以直接往下看 TA 的对手战绩列表，PK 报表里再看双方对比。',
      totalMatchesText: '真实对局 12 场',
      recordText: '累计胜场 7 场',
      recentMatchText: '最近更新 3天前',
      emptyTitle: '',
      emptyDesc: '',
      battleSectionTitle: '对方战绩',
      battleSectionTip: '直接查看 TA 最近的过往对手',
      hiddenTitle: '对方已隐藏战绩',
      hiddenDesc: '这位好友暂时没有公开自己的战绩列表'
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
      heroTitle: '这位球友还没有公开的对局样本',
      heroDesc: '等 TA 完成更多真实对局后，这里会直接更新最新战绩列表。',
      totalMatchesText: '真实对局 0 场',
      recordText: '累计胜场 0 场',
      recentMatchText: '最近更新 暂无记录',
      emptyTitle: '暂时还没有可展示的战绩',
      emptyDesc: '等这位好友完成真实对局后，再回来看看',
      battleSectionTitle: '对方战绩',
      battleSectionTip: '直接查看 TA 最近的过往对手',
      hiddenTitle: '对方已隐藏战绩',
      hiddenDesc: '这位好友暂时没有公开自己的战绩列表'
    }
  )
})

test('buildFriendHomepageSummary keeps friend homepage copy free from my-side comparison wording', () => {
  const summary = buildFriendHomepageSummary({
    stats: {
      total_matches: 9,
      my_wins: 4,
      opponent_wins: 5
    },
    lastMatchAt: '2026-03-20T12:00:00.000Z',
    formatRelativeTime: () => '前天'
  })

  assert.doesNotMatch(summary.heroTitle, /你|我们|交手|领先|落后|势均力敌/)
  assert.doesNotMatch(summary.heroDesc, /你|我们|交手|领先|落后|追回|胶着/)
  assert.equal(summary.recordText, '累计胜场 4 场')
  assert.equal(summary.recentMatchText, '最近更新 前天')
})

test('buildFriendHomepageSummary exposes dedicated hidden state copy for friend battle list', () => {
  const summary = buildFriendHomepageSummary({
    stats: {
      total_matches: 9,
      my_wins: 4,
      opponent_wins: 5
    },
    lastMatchAt: '2026-03-20T12:00:00.000Z',
    formatRelativeTime: () => '前天'
  })

  assert.equal(summary.hiddenTitle, '对方已隐藏战绩')
  assert.equal(summary.hiddenDesc, '这位好友暂时没有公开自己的战绩列表')
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
