<template>
  <div class="member-rights-page">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <div>
            <span>会员排位权益配置</span>
            <div class="card-subtitle">独立维护会员成长 MVP 与会员排位权益 V2，旧会员奖励配置页保持不变。</div>
          </div>
          <div v-if="config.updated_at" class="update-meta">最近更新：{{ config.updated_at }}</div>
        </div>
      </template>

      <el-form
        :model="config"
        label-width="132px"
        class="config-form"
        v-loading="configLoading"
      >
        <div class="config-section-title">会员成长 MVP</div>
        <div class="config-grid">
          <el-form-item label="每场成长值">
            <el-input-number v-model="config.growth_rules.points_per_completed_match" :min="1" :max="100" />
          </el-form-item>
          <el-form-item label="每日成长上限">
            <el-input-number v-model="config.growth_rules.daily_match_cap" :min="1" :max="100" />
          </el-form-item>
          <el-form-item label="仅真实完赛生效">
            <el-switch v-model="config.growth_rules.only_completed_real_matches" />
          </el-form-item>
          <el-form-item label="到期策略">
            <el-input :model-value="expirationPolicyText" disabled />
          </el-form-item>
        </div>

        <div class="nested-section-title">等级门槛</div>
        <div class="config-grid">
          <el-form-item label="Lv1">
            <el-input-number v-model="config.growth_rules.level_thresholds.lv1" :min="0" :max="0" disabled />
          </el-form-item>
          <el-form-item label="Lv2">
            <el-input-number v-model="config.growth_rules.level_thresholds.lv2" :min="1" :max="999999" />
          </el-form-item>
          <el-form-item label="Lv3">
            <el-input-number v-model="config.growth_rules.level_thresholds.lv3" :min="1" :max="999999" />
          </el-form-item>
          <el-form-item label="Lv4">
            <el-input-number v-model="config.growth_rules.level_thresholds.lv4" :min="1" :max="999999" />
          </el-form-item>
          <el-form-item label="Lv5">
            <el-input-number v-model="config.growth_rules.level_thresholds.lv5" :min="1" :max="999999" />
          </el-form-item>
        </div>

        <el-divider />

        <div class="config-section-title">会员排位权益 V2</div>
        <div class="config-grid">
          <el-form-item label="普通用户计入特殊战绩">
            <el-switch v-model="config.ranking_rights_rules.ordinary_user_can_gain_achievement_rank_score" />
          </el-form-item>
          <el-form-item label="特殊战绩单日上限">
            <el-input-number v-model="config.ranking_rights_rules.daily_achievement_rank_score_cap" :min="1" :max="999999" />
          </el-form-item>
          <el-form-item label="取整方式">
            <el-input :model-value="roundingModeText" disabled />
          </el-form-item>
          <el-form-item label="仅胜方生效">
            <el-switch v-model="config.ranking_rights_rules.win_only" />
          </el-form-item>
          <el-form-item label="不设单场上限">
            <el-switch v-model="config.ranking_rights_rules.no_per_match_cap" />
          </el-form-item>
        </div>

        <div class="nested-section-title">特殊战绩基础分</div>
        <div class="config-grid">
          <el-form-item label="单杆 50+">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.break_50" :min="0" :max="999999" />
          </el-form-item>
          <el-form-item label="小金">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.small_gold" :min="0" :max="999999" />
          </el-form-item>
          <el-form-item label="炸清">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.break_clear" :min="0" :max="999999" />
          </el-form-item>
          <el-form-item label="接清">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.continue_clear" :min="0" :max="999999" />
          </el-form-item>
          <el-form-item label="单杆 100+">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.break_100" :min="0" :max="999999" />
          </el-form-item>
          <el-form-item label="大金">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.golden_break" :min="0" :max="999999" />
          </el-form-item>
          <el-form-item label="147">
            <el-input-number v-model="config.ranking_rights_rules.achievement_scores.break_147" :min="0" :max="999999" />
          </el-form-item>
        </div>

        <div class="nested-section-title">会员等级倍率</div>
        <div class="config-grid">
          <el-form-item label="Lv1">
            <el-input-number v-model="config.ranking_rights_rules.level_multipliers.lv1" :min="0" :max="1000" />
          </el-form-item>
          <el-form-item label="Lv2">
            <el-input-number v-model="config.ranking_rights_rules.level_multipliers.lv2" :min="0" :max="1000" />
          </el-form-item>
          <el-form-item label="Lv3">
            <el-input-number v-model="config.ranking_rights_rules.level_multipliers.lv3" :min="0" :max="1000" />
          </el-form-item>
          <el-form-item label="Lv4">
            <el-input-number v-model="config.ranking_rights_rules.level_multipliers.lv4" :min="0" :max="1000" />
          </el-form-item>
          <el-form-item label="Lv5">
            <el-input-number v-model="config.ranking_rights_rules.level_multipliers.lv5" :min="0" :max="1000" />
          </el-form-item>
        </div>

        <div class="notes-block">
          <div class="config-section-title">规则说明</div>
          <el-alert
            type="info"
            :closable="false"
            title="不回填历史排位；仅影响后续结算。"
          />
          <el-alert
            type="info"
            :closable="false"
            title="特殊战绩排位分统一向下取整。"
          />
          <el-alert
            type="info"
            :closable="false"
            title="会员特殊战绩排位分默认仅胜方可得，且只有单日上限，没有单场上限。"
          />
        </div>

        <div class="config-actions">
          <el-button @click="fetchConfig">重载配置</el-button>
          <el-button type="primary" :loading="configSaving" @click="handleSave">保存配置</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createDefaultMemberRightsConfig,
  getMemberRightsConfig,
  updateMemberRightsConfig,
  type MemberRightsConfig
} from '@/api/member-rights'

const configLoading = ref(false)
const configSaving = ref(false)
const config = reactive<MemberRightsConfig>(createDefaultMemberRightsConfig())

const expirationPolicyText = computed(() => {
  return config.growth_rules.expiration_policy === 'freeze_preserve_resume'
    ? '冻结但保留，续开后恢复'
    : config.growth_rules.expiration_policy
})

const roundingModeText = computed(() => {
  return config.ranking_rights_rules.rounding_mode === 'floor' ? '向下取整' : config.ranking_rights_rules.rounding_mode
})

const assignConfig = (nextConfig: MemberRightsConfig) => {
  config.growth_rules.points_per_completed_match = nextConfig.growth_rules.points_per_completed_match
  config.growth_rules.daily_match_cap = nextConfig.growth_rules.daily_match_cap
  config.growth_rules.level_thresholds.lv1 = nextConfig.growth_rules.level_thresholds.lv1
  config.growth_rules.level_thresholds.lv2 = nextConfig.growth_rules.level_thresholds.lv2
  config.growth_rules.level_thresholds.lv3 = nextConfig.growth_rules.level_thresholds.lv3
  config.growth_rules.level_thresholds.lv4 = nextConfig.growth_rules.level_thresholds.lv4
  config.growth_rules.level_thresholds.lv5 = nextConfig.growth_rules.level_thresholds.lv5
  config.growth_rules.expiration_policy = nextConfig.growth_rules.expiration_policy
  config.growth_rules.only_completed_real_matches = nextConfig.growth_rules.only_completed_real_matches

  config.ranking_rights_rules.ordinary_user_can_gain_achievement_rank_score =
    nextConfig.ranking_rights_rules.ordinary_user_can_gain_achievement_rank_score
  config.ranking_rights_rules.daily_achievement_rank_score_cap =
    nextConfig.ranking_rights_rules.daily_achievement_rank_score_cap
  config.ranking_rights_rules.rounding_mode = nextConfig.ranking_rights_rules.rounding_mode
  config.ranking_rights_rules.win_only = nextConfig.ranking_rights_rules.win_only
  config.ranking_rights_rules.no_per_match_cap = nextConfig.ranking_rights_rules.no_per_match_cap
  config.ranking_rights_rules.achievement_scores.break_50 = nextConfig.ranking_rights_rules.achievement_scores.break_50
  config.ranking_rights_rules.achievement_scores.small_gold = nextConfig.ranking_rights_rules.achievement_scores.small_gold
  config.ranking_rights_rules.achievement_scores.break_clear = nextConfig.ranking_rights_rules.achievement_scores.break_clear
  config.ranking_rights_rules.achievement_scores.continue_clear = nextConfig.ranking_rights_rules.achievement_scores.continue_clear
  config.ranking_rights_rules.achievement_scores.break_100 = nextConfig.ranking_rights_rules.achievement_scores.break_100
  config.ranking_rights_rules.achievement_scores.golden_break = nextConfig.ranking_rights_rules.achievement_scores.golden_break
  config.ranking_rights_rules.achievement_scores.break_147 = nextConfig.ranking_rights_rules.achievement_scores.break_147
  config.ranking_rights_rules.level_multipliers.lv1 = nextConfig.ranking_rights_rules.level_multipliers.lv1
  config.ranking_rights_rules.level_multipliers.lv2 = nextConfig.ranking_rights_rules.level_multipliers.lv2
  config.ranking_rights_rules.level_multipliers.lv3 = nextConfig.ranking_rights_rules.level_multipliers.lv3
  config.ranking_rights_rules.level_multipliers.lv4 = nextConfig.ranking_rights_rules.level_multipliers.lv4
  config.ranking_rights_rules.level_multipliers.lv5 = nextConfig.ranking_rights_rules.level_multipliers.lv5
  config.updated_at = nextConfig.updated_at
  config.updated_by = nextConfig.updated_by
}

const fetchConfig = async () => {
  configLoading.value = true
  try {
    const res = await getMemberRightsConfig()
    assignConfig(res)
  } catch (error: any) {
    ElMessage.error(error.message || '获取会员排位权益配置失败')
  } finally {
    configLoading.value = false
  }
}

const validateConfig = () => {
  const thresholds = config.growth_rules.level_thresholds
  if (
    thresholds.lv1 !== 0 ||
    thresholds.lv2 <= thresholds.lv1 ||
    thresholds.lv3 <= thresholds.lv2 ||
    thresholds.lv4 <= thresholds.lv3 ||
    thresholds.lv5 <= thresholds.lv4
  ) {
    ElMessage.error('会员成长等级门槛必须严格递增，且 Lv1 固定为 0')
    return false
  }

  const multipliers = config.ranking_rights_rules.level_multipliers
  if (
    multipliers.lv2 < multipliers.lv1 ||
    multipliers.lv3 < multipliers.lv2 ||
    multipliers.lv4 < multipliers.lv3 ||
    multipliers.lv5 < multipliers.lv4
  ) {
    ElMessage.error('会员等级倍率必须按 Lv1 到 Lv5 非递减')
    return false
  }

  return true
}

const handleSave = async () => {
  if (!validateConfig()) return

  configSaving.value = true
  try {
    const res = await updateMemberRightsConfig(config)
    ElMessage.success(res.message || '会员排位权益配置已保存')
    await fetchConfig()
  } catch (error: any) {
    ElMessage.error(error.message || '保存会员排位权益配置失败')
  } finally {
    configSaving.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<style scoped lang="scss">
.member-rights-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.card-subtitle {
  margin-top: 6px;
  font-size: 13px;
  color: #6b7280;
}

.update-meta {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
  margin-top: 4px;
}

.nested-section-title {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
  margin-top: 4px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 16px;
}

.notes-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 960px) {
  .card-header {
    flex-direction: column;
  }

  .update-meta {
    white-space: normal;
  }

  .config-grid {
    grid-template-columns: 1fr;
  }
}
</style>
