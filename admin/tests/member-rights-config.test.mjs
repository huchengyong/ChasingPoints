import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildUpdatePayload,
  createDefaultMemberRightsConfig,
  normalizeConfig
} from '../src/utils/member-rights-config.ts'

// 与后端 AdminMemberRightsConfigResp 的 JSON 字段一致；数值刻意不取默认值，才能看出是否真的读到了。
const serverConfig = {
  growth_rules: {
    points_per_completed_match: 2,
    daily_cap: 8,
    level_threshold_lv2: 20,
    level_threshold_lv3: 70,
    level_threshold_lv4: 300,
    level_threshold_lv5: 900,
    expire_strategy: 'freeze_preserve'
  },
  ranking_rights_rules: {
    ordinary_user_achievement_enabled: true,
    daily_cap: 120,
    daily_positive_cap: 400,
    break_50_score: 9,
    golden_break_score: 5,
    break_and_run_score: 7,
    run_out_score: 5,
    break_100_score: 17,
    nine_on_break_score: 7,
    break_147_score: 31,
    level1_multiplier: 100,
    level2_multiplier: 111,
    level3_multiplier: 121,
    level4_multiplier: 131,
    level5_multiplier: 141
  }
}

test('normalizeConfig reads the fields the backend actually returns', () => {
  const config = normalizeConfig(serverConfig)

  assert.equal(config.growth_rules.daily_match_cap, 8)
  assert.equal(config.growth_rules.points_per_completed_match, 2)
  assert.equal(config.growth_rules.level_thresholds.lv5, 900)
  assert.equal(config.growth_rules.expiration_policy, 'freeze_preserve_resume')
  assert.equal(config.ranking_rights_rules.ordinary_user_can_gain_achievement_rank_score, true)
  assert.equal(config.ranking_rights_rules.daily_achievement_rank_score_cap, 120)
  assert.equal(config.ranking_rights_rules.daily_total_positive_rank_score_cap, 400)
  assert.equal(config.ranking_rights_rules.achievement_scores.small_gold, 5)
  assert.equal(config.ranking_rights_rules.achievement_scores.golden_break, 7)
  assert.equal(config.ranking_rights_rules.level_multipliers.lv2, 111)
})

test('saving a loaded config sends back the values it was loaded with', () => {
  const payload = buildUpdatePayload(normalizeConfig(serverConfig))

  assert.deepEqual(payload, {
    growth_rules: serverConfig.growth_rules,
    ranking_rights_rules: serverConfig.ranking_rights_rules
  })
})

test('createDefaultMemberRightsConfig keeps the documented defaults', () => {
  const config = createDefaultMemberRightsConfig()

  assert.equal(config.growth_rules.daily_match_cap, 5)
  assert.equal(config.growth_rules.expiration_policy, 'freeze_preserve_resume')
  assert.equal(config.ranking_rights_rules.daily_achievement_rank_score_cap, 200)
  assert.equal(config.ranking_rights_rules.ordinary_user_can_gain_achievement_rank_score, false)
})
