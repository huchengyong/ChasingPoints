import test from 'node:test'
import assert from 'node:assert/strict'

import { resolveMatchRankingRightsSummary } from '../utils/member-ranking-rights.js'

test('resolveMatchRankingRightsSummary marks highlights as recorded only when no rank detail was counted', () => {
  const summary = resolveMatchRankingRightsSummary({
    achievements: {
      break_clear: 1,
      break_100: 1
    },
    my_rank_details: [
      { label: '基础分', value: 20 }
    ]
  })

  assert.equal(summary.countedInRanking, false)
  assert.equal(summary.badgeText, '高光仅记录')
  assert.match(summary.description, /普通用户高光会保留记录，会员用户才会按等级倍率计入/)
})

test('resolveMatchRankingRightsSummary explains counted member highlights and daily cap clipping', () => {
  const summary = resolveMatchRankingRightsSummary({
    achievements: {
      break_147: 1
    },
    my_rank_details: [
      { label: '基础分', value: 20 },
      { label: '会员特殊战绩分', value: 30 },
      { label: '每日封顶', value: -6 }
    ]
  })

  assert.equal(summary.countedInRanking, true)
  assert.equal(summary.capped, true)
  assert.equal(summary.badgeText, '已计入并触发封顶')
  assert.match(summary.description, /30 分特殊战绩已计入排位/)
  assert.match(summary.description, /6 分受到会员特殊战绩单日上限影响/)
})

test('resolveMatchRankingRightsSummary falls back to base-rank explanation without achievements', () => {
  const summary = resolveMatchRankingRightsSummary({
    achievements: {},
    my_rank_details: [{ label: '基础分', value: 20 }]
  })

  assert.equal(summary.countedInRanking, false)
  assert.equal(summary.recordedAchievementsText, '')
  assert.equal(summary.title, '本场没有可计入排位的特殊战绩')
})
