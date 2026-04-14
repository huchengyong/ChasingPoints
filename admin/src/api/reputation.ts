import request from '@/utils/request'

type PartialRecord = Record<string, any>

export interface ReputationBaseRules {
  max_score: number
  initial_score: number
  ban_threshold: number
  ban_duration_hours: number
  min_score: number
}

export interface ReputationRecoveryRules {
  enabled: boolean
  recover_per_hour: number
  recover_max_score: number
}

export interface ReputationDurationRule {
  game_type: number
  enabled: boolean
  min_minutes_per_round: number
  min_total_rounds: number
  min_total_duration_minutes: number
  penalty_score: number
}

export interface ReputationSameOpponentRule {
  enabled: boolean
  window_minutes: number
  max_matches: number
  penalty_score: number
  require_same_game_type: boolean
}

export interface ReputationDetectionRules {
  duration_rules: ReputationDurationRule[]
  same_opponent_rule: ReputationSameOpponentRule
  stack_penalties_per_match: boolean
}

export interface ReputationConfig {
  base_rules: ReputationBaseRules
  recovery_rules: ReputationRecoveryRules
  detection_rules: ReputationDetectionRules
}

const DEFAULT_CONFIG: ReputationConfig = {
  base_rules: {
    max_score: 100,
    initial_score: 100,
    ban_threshold: 60,
    ban_duration_hours: 24,
    min_score: 0
  },
  recovery_rules: {
    enabled: true,
    recover_per_hour: 1,
    recover_max_score: 100
  },
  detection_rules: {
    duration_rules: [
      { game_type: 1, enabled: true, min_minutes_per_round: 15, min_total_rounds: 3, min_total_duration_minutes: 45, penalty_score: 12 },
      { game_type: 2, enabled: true, min_minutes_per_round: 3, min_total_rounds: 5, min_total_duration_minutes: 18, penalty_score: 8 },
      { game_type: 3, enabled: true, min_minutes_per_round: 2, min_total_rounds: 5, min_total_duration_minutes: 15, penalty_score: 10 },
      { game_type: 4, enabled: true, min_minutes_per_round: 4, min_total_rounds: 5, min_total_duration_minutes: 20, penalty_score: 10 }
    ],
    same_opponent_rule: {
      enabled: true,
      window_minutes: 30,
      max_matches: 4,
      penalty_score: 8,
      require_same_game_type: true
    },
    stack_penalties_per_match: false
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

const getArray = (source: PartialRecord | undefined, keys: string[]) => {
  for (const key of keys) {
    const value = source?.[key]
    if (Array.isArray(value)) {
      return value as PartialRecord[]
    }
  }
  return []
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

const normalizeDurationRules = (rawRules: PartialRecord[]) => {
  const fallback = DEFAULT_CONFIG.detection_rules.duration_rules
  const rules = (rawRules.length > 0 ? rawRules : fallback).map((rule, index) => {
    const base = fallback[index] || fallback.find((item) => item.game_type === getNumber(rule, ['game_type', 'gameType'], 0)) || fallback[0]
    return {
      game_type: getNumber(rule, ['game_type', 'gameType'], base.game_type),
      enabled: getBoolean(rule, ['enabled'], base.enabled),
      min_minutes_per_round: getNumber(rule, ['min_minutes_per_round', 'minMinutesPerRound'], base.min_minutes_per_round),
      min_total_rounds: getNumber(rule, ['min_total_rounds', 'minTotalRounds'], base.min_total_rounds),
      min_total_duration_minutes: getNumber(rule, ['min_total_duration_minutes', 'minTotalDurationMinutes'], base.min_total_duration_minutes),
      penalty_score: getNumber(rule, ['penalty_score', 'penaltyScore'], base.penalty_score)
    }
  })

  return rules.sort((left, right) => left.game_type - right.game_type)
}

const normalizeConfig = (raw?: PartialRecord): ReputationConfig => {
  const baseRules = getObject(raw, ['base_rules', 'baseRules'])
  const recoveryRules = getObject(raw, ['recovery_rules', 'recoveryRules'])
  const detectionRules = getObject(raw, ['detection_rules', 'detectionRules'])
  const sameOpponentRule = getObject(detectionRules, ['same_opponent_rule', 'sameOpponentRule'])

  return {
    base_rules: {
      max_score: getNumber(baseRules, ['max_score', 'maxScore'], DEFAULT_CONFIG.base_rules.max_score),
      initial_score: getNumber(baseRules, ['initial_score', 'initialScore'], DEFAULT_CONFIG.base_rules.initial_score),
      ban_threshold: getNumber(baseRules, ['ban_threshold', 'banThreshold'], DEFAULT_CONFIG.base_rules.ban_threshold),
      ban_duration_hours: getNumber(baseRules, ['ban_duration_hours', 'banDurationHours'], DEFAULT_CONFIG.base_rules.ban_duration_hours),
      min_score: getNumber(baseRules, ['min_score', 'minScore'], DEFAULT_CONFIG.base_rules.min_score)
    },
    recovery_rules: {
      enabled: getBoolean(recoveryRules, ['enabled'], DEFAULT_CONFIG.recovery_rules.enabled),
      recover_per_hour: getNumber(recoveryRules, ['recover_per_hour', 'recoverPerHour'], DEFAULT_CONFIG.recovery_rules.recover_per_hour),
      recover_max_score: getNumber(recoveryRules, ['recover_max_score', 'recoverMaxScore'], DEFAULT_CONFIG.recovery_rules.recover_max_score)
    },
    detection_rules: {
      duration_rules: normalizeDurationRules(getArray(detectionRules, ['duration_rules', 'durationRules'])),
      same_opponent_rule: {
        enabled: getBoolean(sameOpponentRule, ['enabled'], DEFAULT_CONFIG.detection_rules.same_opponent_rule.enabled),
        window_minutes: getNumber(sameOpponentRule, ['window_minutes', 'windowMinutes'], DEFAULT_CONFIG.detection_rules.same_opponent_rule.window_minutes),
        max_matches: getNumber(sameOpponentRule, ['max_matches', 'maxMatches'], DEFAULT_CONFIG.detection_rules.same_opponent_rule.max_matches),
        penalty_score: getNumber(sameOpponentRule, ['penalty_score', 'penaltyScore'], DEFAULT_CONFIG.detection_rules.same_opponent_rule.penalty_score),
        require_same_game_type: getBoolean(
          sameOpponentRule,
          ['require_same_game_type', 'requireSameGameType'],
          DEFAULT_CONFIG.detection_rules.same_opponent_rule.require_same_game_type
        )
      },
      stack_penalties_per_match: getBoolean(
        detectionRules,
        ['stack_penalties_per_match', 'stackPenaltiesPerMatch'],
        DEFAULT_CONFIG.detection_rules.stack_penalties_per_match
      )
    }
  }
}

export const createDefaultReputationConfig = (): ReputationConfig => normalizeConfig(DEFAULT_CONFIG as PartialRecord)

export const getReputationConfig = async (): Promise<ReputationConfig> => {
  const res = await request.get<unknown, PartialRecord>('/api/admin/reputation/config')
  return normalizeConfig(res)
}

export const updateReputationConfig = async (
  data: ReputationConfig
): Promise<{ code?: number; success?: boolean; message?: string }> => {
  return request.post('/api/admin/reputation/config', data)
}
