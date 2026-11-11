import test from 'node:test'
import assert from 'node:assert/strict'

import {
  resolveAchievementRecords,
  resolveMatchRankingRightsSummary
} from '../utils/member-ranking-rights.js'
import * as rankingRights from '../utils/member-ranking-rights.js'

test('resolveAchievementRecords keeps only recorded special achievements', () => {
  const records = resolveAchievementRecords({
    break_clear: 1,
    break_100: 2,
    golden_break: 0
  })

  assert.deepEqual(records.map(item => [item.label, item.count]), [
    ['炸清', 1],
    ['单杆100+', 2]
  ])
})

test('the member ranking rights card and its multiplier constants are gone, only the per-match summary remains', () => {
  assert.equal(rankingRights.resolveMemberRankingRightsCard, undefined)
  assert.equal(rankingRights.MEMBER_RANKING_LEVELS, undefined)
  assert.equal(rankingRights.MEMBER_RANKING_DAILY_CAP, undefined)
  assert.deepEqual(Object.keys(rankingRights).sort(), ['resolveAchievementRecords', 'resolveMatchRankingRightsSummary'])
})

test('resolveMatchRankingRightsSummary explains v2 matches without member wording', () => {
  const recorded = resolveMatchRankingRightsSummary({
    rank_rules_version: 2,
    achievements: { break_clear: 1 },
    my_rank_details: [{ label: '胜负结算（赛前胜率 36%）', value: 57 }]
  })
  assert.equal(recorded.badgeText, '高光不计分')
  assert.equal(recorded.countedInRanking, false)
  assert.match(recorded.description, /炸清x1/)
  assert.match(recorded.description, /排位分只看胜负与对手实力/)
  assert.doesNotMatch(recorded.description, /VIP/)

  const plain = resolveMatchRankingRightsSummary({ rank_rules_version: 2, achievements: {} })
  assert.equal(plain.badgeText, '只看胜负')
  assert.match(plain.description, /会员与普通用户规则相同/)
})

test('resolveMatchRankingRightsSummary marks practice matches as not counted in ranking', () => {
  const recorded = resolveMatchRankingRightsSummary({
    match_mode: 'practice',
    achievements: { break_clear: 1 }
  })
  assert.equal(recorded.badgeText, '练习赛')
  assert.equal(recorded.title, '练习赛不计入排位')
  assert.equal(recorded.tone, 'slate')
  assert.equal(recorded.countedInRanking, false)
  assert.equal(recorded.recordedAchievementsText, '炸清x1')
  assert.equal(recorded.description, '本场记录了 炸清x1，高光会保留在战报和成就里。')

  const plain = resolveMatchRankingRightsSummary({ match_mode: 'practice' })
  assert.equal(plain.recordedAchievementsText, '')
  assert.equal(plain.description, '练习赛的胜负和战绩不影响排位分。')

  // 排位赛规则保持原样。
  const ranked = resolveMatchRankingRightsSummary({ match_mode: 'ranked', rank_rules_version: 2, achievements: {} })
  assert.equal(ranked.badgeText, '只看胜负')
})
