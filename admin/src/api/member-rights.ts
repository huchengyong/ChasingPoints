import request from '@/utils/request'

export interface MemberGrowthThresholds {
  lv1: number
  lv2: number
  lv3: number
  lv4: number
  lv5: number
}

export interface MemberGrowthRules {
  points_per_completed_match: number
  daily_match_cap: number
  level_thresholds: MemberGrowthThresholds
  expiration_policy: 'freeze_preserve_resume'
  only_completed_real_matches: boolean
}

export interface MemberAchievementScores {
  break_50: number
  small_gold: number
  break_clear: number
  continue_clear: number
  break_100: number
  golden_break: number
  break_147: number
}

export interface MemberLevelMultipliers {
  lv1: number
  lv2: number
  lv3: number
  lv4: number
  lv5: number
}

export interface MemberRankingRightsRules {
  ordinary_user_can_gain_achievement_rank_score: boolean
  daily_achievement_rank_score_cap: number
  rounding_mode: 'floor'
  win_only: boolean
  no_per_match_cap: boolean
  achievement_scores: MemberAchievementScores
  level_multipliers: MemberLevelMultipliers
}

export interface MemberRightsConfig {
  growth_rules: MemberGrowthRules
  ranking_rights_rules: MemberRankingRightsRules
  updated_at?: string
  updated_by?: number
}

type PartialRecord = Record<string, any>

const DEFAULT_CONFIG: MemberRightsConfig = {
  growth_rules: {
    points_per_completed_match: 1,
    daily_match_cap: 5,
    level_thresholds: {
      lv1: 0,
      lv2: 10,
      lv3: 60,
      lv4: 260,
      lv5: 760
    },
    expiration_policy: 'freeze_preserve_resume',
    only_completed_real_matches: true
  },
  ranking_rights_rules: {
    ordinary_user_can_gain_achievement_rank_score: false,
    daily_achievement_rank_score_cap: 200,
    rounding_mode: 'floor',
    win_only: true,
    no_per_match_cap: true,
    achievement_scores: {
      break_50: 8,
      small_gold: 4,
      break_clear: 6,
      continue_clear: 4,
      break_100: 16,
      golden_break: 6,
      break_147: 30
    },
    level_multipliers: {
      lv1: 100,
      lv2: 110,
      lv3: 120,
      lv4: 130,
      lv5: 140
    }
  }
}

const getObject = (source: PartialRecord | undefined, keys: string[]) => {
  for (const key of keys) {
    const value = source?.[key]
    if (value && typeof value === 'object') {
      return value as PartialRecord
    }
  }
  return {}
}

const getNumber = (source: PartialRecord | undefined, keys: string[], fallback: number) => {
  for (const key of keys) {
    const value = source?.[key]
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
  }
  return fallback
}

const getBoolean = (source: PartialRecord | undefined, keys: string[], fallback: boolean) => {
  for (const key of keys) {
    const value = source?.[key]
    if (typeof value === 'boolean') {
      return value
    }
  }
  return fallback
}

const getString = <T extends string>(source: PartialRecord | undefined, keys: string[], fallback: T) => {
  for (const key of keys) {
    const value = source?.[key]
    if (typeof value === 'string' && value) {
      return value as T
    }
  }
  return fallback
}

const normalizeConfig = (raw?: PartialRecord): MemberRightsConfig => {
  const growthRules = getObject(raw, ['growth_rules', 'growthRules'])
  const growthThresholds = getObject(growthRules, ['level_thresholds', 'levelThresholds'])
  const rankingRightsRules = getObject(raw, ['ranking_rights_rules', 'rankingRightsRules'])
  const achievementScores = getObject(rankingRightsRules, ['achievement_scores', 'achievementScores'])
  const levelMultipliers = getObject(rankingRightsRules, ['level_multipliers', 'levelMultipliers'])

  return {
    growth_rules: {
      points_per_completed_match: getNumber(
        growthRules,
        ['points_per_completed_match', 'pointsPerCompletedMatch'],
        DEFAULT_CONFIG.growth_rules.points_per_completed_match
      ),
      daily_match_cap: getNumber(growthRules, ['daily_match_cap', 'dailyMatchCap'], DEFAULT_CONFIG.growth_rules.daily_match_cap),
      level_thresholds: {
        lv1: getNumber(growthThresholds, ['lv1'], DEFAULT_CONFIG.growth_rules.level_thresholds.lv1),
        lv2: getNumber(growthThresholds, ['lv2'], DEFAULT_CONFIG.growth_rules.level_thresholds.lv2),
        lv3: getNumber(growthThresholds, ['lv3'], DEFAULT_CONFIG.growth_rules.level_thresholds.lv3),
        lv4: getNumber(growthThresholds, ['lv4'], DEFAULT_CONFIG.growth_rules.level_thresholds.lv4),
        lv5: getNumber(growthThresholds, ['lv5'], DEFAULT_CONFIG.growth_rules.level_thresholds.lv5)
      },
      expiration_policy: getString(
        growthRules,
        ['expiration_policy', 'expirationPolicy'],
        DEFAULT_CONFIG.growth_rules.expiration_policy
      ),
      only_completed_real_matches: getBoolean(
        growthRules,
        ['only_completed_real_matches', 'onlyCompletedRealMatches'],
        DEFAULT_CONFIG.growth_rules.only_completed_real_matches
      )
    },
    ranking_rights_rules: {
      ordinary_user_can_gain_achievement_rank_score: getBoolean(
        rankingRightsRules,
        ['ordinary_user_can_gain_achievement_rank_score', 'ordinaryUserCanGainAchievementRankScore'],
        DEFAULT_CONFIG.ranking_rights_rules.ordinary_user_can_gain_achievement_rank_score
      ),
      daily_achievement_rank_score_cap: getNumber(
        rankingRightsRules,
        ['daily_achievement_rank_score_cap', 'dailyAchievementRankScoreCap'],
        DEFAULT_CONFIG.ranking_rights_rules.daily_achievement_rank_score_cap
      ),
      rounding_mode: getString(
        rankingRightsRules,
        ['rounding_mode', 'roundingMode'],
        DEFAULT_CONFIG.ranking_rights_rules.rounding_mode
      ),
      win_only: getBoolean(rankingRightsRules, ['win_only', 'winOnly'], DEFAULT_CONFIG.ranking_rights_rules.win_only),
      no_per_match_cap: getBoolean(
        rankingRightsRules,
        ['no_per_match_cap', 'noPerMatchCap'],
        DEFAULT_CONFIG.ranking_rights_rules.no_per_match_cap
      ),
      achievement_scores: {
        break_50: getNumber(achievementScores, ['break_50', 'break50'], DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.break_50),
        small_gold: getNumber(
          achievementScores,
          ['small_gold', 'smallGold'],
          DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.small_gold
        ),
        break_clear: getNumber(
          achievementScores,
          ['break_clear', 'breakClear'],
          DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.break_clear
        ),
        continue_clear: getNumber(
          achievementScores,
          ['continue_clear', 'continueClear'],
          DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.continue_clear
        ),
        break_100: getNumber(
          achievementScores,
          ['break_100', 'break100'],
          DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.break_100
        ),
        golden_break: getNumber(
          achievementScores,
          ['golden_break', 'goldenBreak'],
          DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.golden_break
        ),
        break_147: getNumber(
          achievementScores,
          ['break_147', 'break147'],
          DEFAULT_CONFIG.ranking_rights_rules.achievement_scores.break_147
        )
      },
      level_multipliers: {
        lv1: getNumber(levelMultipliers, ['lv1'], DEFAULT_CONFIG.ranking_rights_rules.level_multipliers.lv1),
        lv2: getNumber(levelMultipliers, ['lv2'], DEFAULT_CONFIG.ranking_rights_rules.level_multipliers.lv2),
        lv3: getNumber(levelMultipliers, ['lv3'], DEFAULT_CONFIG.ranking_rights_rules.level_multipliers.lv3),
        lv4: getNumber(levelMultipliers, ['lv4'], DEFAULT_CONFIG.ranking_rights_rules.level_multipliers.lv4),
        lv5: getNumber(levelMultipliers, ['lv5'], DEFAULT_CONFIG.ranking_rights_rules.level_multipliers.lv5)
      }
    },
    updated_at: getString(raw, ['updated_at', 'updatedAt'], ''),
    updated_by: getNumber(raw, ['updated_by', 'updatedBy'], 0) || undefined
  }
}

export const createDefaultMemberRightsConfig = (): MemberRightsConfig => {
  return normalizeConfig(DEFAULT_CONFIG)
}

export const getMemberRightsConfig = async (): Promise<MemberRightsConfig> => {
  const res = await request.get<unknown, PartialRecord>('/api/admin/member/ranking-rights-config')
  return normalizeConfig(res)
}

export const updateMemberRightsConfig = async (
  data: MemberRightsConfig
): Promise<{ code?: number; success?: boolean; message?: string; updated_at?: string; updated_by?: number }> => {
  return request.post('/api/admin/member/ranking-rights-config', data)
}
