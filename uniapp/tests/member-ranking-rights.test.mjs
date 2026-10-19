import test from 'node:test'
import assert from 'node:assert/strict'

import {
  MEMBER_RANKING_DAILY_CAP,
  resolveAchievementRecords,
  resolveMatchRankingRightsSummary,
  resolveMemberRankingRightsCard
} from '../utils/member-ranking-rights.js'

test('resolveMemberRankingRightsCard exposes active level multiplier for members', () => {
  const card = resolveMemberRankingRightsCard({
    is_active: true,
    member_expires_at: '2026-05-30 10:00:00',
    growth_level: 3
  }, new Date('2026-04-12T10:00:00+08:00'))

  assert.equal(card.statusText, '权益生效中')
  assert.equal(card.levelLabel, 'Lv3')
  assert.equal(card.currentPercentText, '120%')
  assert.equal(card.dailyCap, MEMBER_RANKING_DAILY_CAP)
  assert.match(card.description, /普通用户高光会保留记录但不计入排位/)
})

test('resolveMemberRankingRightsCard keeps frozen multiplier for expired members', () => {
  const card = resolveMemberRankingRightsCard({
    is_active: false,
    member_expires_at: '2026-04-01 10:00:00',
    growth_level: 4
  }, new Date('2026-04-12T10:00:00+08:00'))

  assert.equal(card.statusText, '权益已冻结')
  assert.equal(card.currentPercentText, '130%')
  assert.match(card.description, /续开后会按 Lv4 的 130% 倍率恢复结算/)
})

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

const V2_ACTIVE = { rulesVersion: 2, rulesEffectiveAt: '2026-11-01T00:00:00+08:00' }

test('resolveMemberRankingRightsCard retires member ranking points once rating v2 is active', () => {
  const card = resolveMemberRankingRightsCard({
    is_active: true,
    member_expires_at: '2026-12-30 10:00:00',
    growth_level: 4
  }, new Date('2026-11-02T10:00:00+08:00'), V2_ACTIVE)

  assert.equal(card.retired, true)
  assert.equal(card.sectionTitle, '排位规则')
  assert.equal(card.headline, '高光不再计入排位分')
  assert.match(card.description, /自2026年11月1日起/)
  assert.deepEqual(card.levels, [])
  assert.equal(card.currentPercentText, '')
  assert.doesNotMatch(card.ruleItems.join(' '), /倍率|200 分/)
})

test('resolveMemberRankingRightsCard keeps legacy rights and warns before the v2 start', () => {
  const card = resolveMemberRankingRightsCard({
    is_active: true,
    member_expires_at: '2026-12-30 10:00:00',
    growth_level: 2
  }, new Date('2026-10-20T10:00:00+08:00'), { rulesVersion: 1, rulesEffectiveAt: '2026-11-01T00:00:00+08:00' })

  assert.equal(card.retired, false)
  assert.equal(card.currentPercentText, '110%')
  assert.match(card.ruleItems.at(-1), /自2026年11月1日起排位规则升级，特殊战绩不再计入排位分/)
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
