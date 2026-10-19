import request from '@/utils/request'
import {
  buildUpdatePayload,
  normalizeConfig,
  type MemberRightsConfig,
  type PartialRecord
} from '@/utils/member-rights-config'

export { createDefaultMemberRightsConfig } from '@/utils/member-rights-config'
export type {
  MemberAchievementScores,
  MemberGrowthRules,
  MemberGrowthThresholds,
  MemberLevelMultipliers,
  MemberRankingRightsRules,
  MemberRightsConfig
} from '@/utils/member-rights-config'

export const getMemberRightsConfig = async (): Promise<MemberRightsConfig> => {
  const res = await request.get<unknown, PartialRecord>('/api/admin/member/ranking-rights-config')
  return normalizeConfig(res)
}

export const updateMemberRightsConfig = async (
  data: MemberRightsConfig
): Promise<{ code?: number; success?: boolean; message?: string; updated_at?: string; updated_by?: number }> => {
  return request.post('/api/admin/member/ranking-rights-config', buildUpdatePayload(data))
}
