<template>
  <div class="reputation-page">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <div>
            <span>信誉制度配置</span>
            <div class="card-subtitle">管理异常比赛识别、扣分策略、禁赛阈值与自然恢复规则。</div>
          </div>
        </div>
      </template>

      <el-form :model="config" label-width="148px" class="config-form" v-loading="configLoading">
        <div class="config-section-title">基础规则</div>
        <div class="config-grid">
          <el-form-item label="信誉满分">
            <el-input-number v-model="config.base_rules.max_score" :min="1" :max="9999" />
          </el-form-item>
          <el-form-item label="新用户初始信誉">
            <el-input-number v-model="config.base_rules.initial_score" :min="0" :max="9999" />
          </el-form-item>
          <el-form-item label="禁赛阈值">
            <el-input-number v-model="config.base_rules.ban_threshold" :min="0" :max="9999" />
          </el-form-item>
          <el-form-item label="禁赛时长(小时)">
            <el-input-number v-model="config.base_rules.ban_duration_hours" :min="0" :max="9999" />
          </el-form-item>
          <el-form-item label="信誉最低分">
            <el-input-number v-model="config.base_rules.min_score" :min="0" :max="9999" />
          </el-form-item>
        </div>

        <el-divider />

        <div class="config-section-title">自然恢复</div>
        <div class="config-grid">
          <el-form-item label="启用自然恢复">
            <el-switch v-model="config.recovery_rules.enabled" />
          </el-form-item>
          <el-form-item label="每小时恢复值">
            <el-input-number v-model="config.recovery_rules.recover_per_hour" :min="0" :max="999" />
          </el-form-item>
          <el-form-item label="恢复上限">
            <el-input-number v-model="config.recovery_rules.recover_max_score" :min="0" :max="9999" />
          </el-form-item>
        </div>

        <el-divider />

        <div class="config-section-title">时长异常规则</div>
        <div class="duration-rule-list">
          <el-card v-for="rule in config.detection_rules.duration_rules" :key="rule.game_type" shadow="never" class="duration-rule-card">
            <template #header>
              <div class="duration-rule-header">
                <span>{{ resolveGameTypeLabel(rule.game_type) }}</span>
                <el-switch v-model="rule.enabled" />
              </div>
            </template>
            <div class="config-grid">
              <el-form-item label="单局参考分钟数">
                <el-input-number v-model="rule.min_minutes_per_round" :min="1" :max="999" />
              </el-form-item>
              <el-form-item label="最少总局数">
                <el-input-number v-model="rule.min_total_rounds" :min="1" :max="999" />
              </el-form-item>
              <el-form-item label="最短总时长(分钟)">
                <el-input-number v-model="rule.min_total_duration_minutes" :min="1" :max="9999" />
              </el-form-item>
              <el-form-item label="命中扣分">
                <el-input-number v-model="rule.penalty_score" :min="0" :max="9999" />
              </el-form-item>
            </div>
          </el-card>
        </div>

        <el-divider />

        <div class="config-section-title">同对手高频规则</div>
        <div class="config-grid">
          <el-form-item label="启用高频判定">
            <el-switch v-model="config.detection_rules.same_opponent_rule.enabled" />
          </el-form-item>
          <el-form-item label="只统计同玩法">
            <el-switch v-model="config.detection_rules.same_opponent_rule.require_same_game_type" />
          </el-form-item>
          <el-form-item label="统计窗口(分钟)">
            <el-input-number v-model="config.detection_rules.same_opponent_rule.window_minutes" :min="0" :max="9999" />
          </el-form-item>
          <el-form-item label="最大允许场次">
            <el-input-number v-model="config.detection_rules.same_opponent_rule.max_matches" :min="0" :max="999" />
          </el-form-item>
          <el-form-item label="命中扣分">
            <el-input-number v-model="config.detection_rules.same_opponent_rule.penalty_score" :min="0" :max="9999" />
          </el-form-item>
          <el-form-item label="单场多规则叠加">
            <el-switch v-model="config.detection_rules.stack_penalties_per_match" />
          </el-form-item>
        </div>

        <div class="notes-block">
          <div class="config-section-title">说明</div>
          <el-alert type="info" :closable="false" title="配置只影响后续比赛，不回溯历史信誉。" />
          <el-alert type="info" :closable="false" title="禁赛只拦截发起比赛，不影响已开始比赛的收尾。" />
          <el-alert type="info" :closable="false" title="管理后台列表显示的是只读计算后的当前信誉，不会因为浏览页面而修改用户数据。" />
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
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createDefaultReputationConfig,
  getReputationConfig,
  updateReputationConfig,
  type ReputationConfig
} from '@/api/reputation'

const configLoading = ref(false)
const configSaving = ref(false)
const config = reactive<ReputationConfig>(createDefaultReputationConfig())

const resolveGameTypeLabel = (gameType: number) => {
  const labels: Record<number, string> = {
    1: '斯诺克',
    2: '九球追分',
    3: '中式八球',
    4: '美式九球'
  }
  return labels[gameType] || `玩法 ${gameType}`
}

const assignConfig = (nextConfig: ReputationConfig) => {
  config.base_rules.max_score = nextConfig.base_rules.max_score
  config.base_rules.initial_score = nextConfig.base_rules.initial_score
  config.base_rules.ban_threshold = nextConfig.base_rules.ban_threshold
  config.base_rules.ban_duration_hours = nextConfig.base_rules.ban_duration_hours
  config.base_rules.min_score = nextConfig.base_rules.min_score

  config.recovery_rules.enabled = nextConfig.recovery_rules.enabled
  config.recovery_rules.recover_per_hour = nextConfig.recovery_rules.recover_per_hour
  config.recovery_rules.recover_max_score = nextConfig.recovery_rules.recover_max_score

  config.detection_rules.duration_rules = nextConfig.detection_rules.duration_rules.map((rule) => ({ ...rule }))
  config.detection_rules.same_opponent_rule = { ...nextConfig.detection_rules.same_opponent_rule }
  config.detection_rules.stack_penalties_per_match = nextConfig.detection_rules.stack_penalties_per_match
}

const fetchConfig = async () => {
  configLoading.value = true
  try {
    const res = await getReputationConfig()
    assignConfig(res)
  } catch (error: any) {
    ElMessage.error(error.message || '获取信誉制度配置失败')
  } finally {
    configLoading.value = false
  }
}

const validateConfig = () => {
  if (config.base_rules.max_score <= 0) {
    ElMessage.error('信誉满分必须大于 0')
    return false
  }
  if (config.base_rules.min_score >= config.base_rules.max_score) {
    ElMessage.error('信誉最低分必须小于信誉满分')
    return false
  }
  if (config.base_rules.initial_score < config.base_rules.ban_threshold) {
    ElMessage.error('新用户初始信誉不能低于禁赛阈值')
    return false
  }
  if (config.base_rules.initial_score > config.base_rules.max_score) {
    ElMessage.error('新用户初始信誉不能高于信誉满分')
    return false
  }
  if (config.recovery_rules.enabled && config.recovery_rules.recover_per_hour <= 0) {
    ElMessage.error('启用自然恢复时，每小时恢复值必须大于 0')
    return false
  }
  if (config.recovery_rules.recover_max_score > config.base_rules.max_score) {
    ElMessage.error('恢复上限不能高于信誉满分')
    return false
  }

  const seenGameTypes = new Set<number>()
  for (const rule of config.detection_rules.duration_rules) {
    if (seenGameTypes.has(rule.game_type)) {
      ElMessage.error(`玩法 ${resolveGameTypeLabel(rule.game_type)} 重复配置`)
      return false
    }
    seenGameTypes.add(rule.game_type)

    if (rule.min_minutes_per_round <= 0 || rule.min_total_rounds <= 0 || rule.min_total_duration_minutes <= 0) {
      ElMessage.error(`${resolveGameTypeLabel(rule.game_type)} 的时长阈值必须大于 0`)
      return false
    }
    if (rule.enabled && rule.penalty_score <= 0) {
      ElMessage.error(`${resolveGameTypeLabel(rule.game_type)} 启用时，命中扣分必须大于 0`)
      return false
    }
  }

  const sameOpponentRule = config.detection_rules.same_opponent_rule
  if (sameOpponentRule.enabled) {
    if (sameOpponentRule.window_minutes <= 0) {
      ElMessage.error('启用同对手高频规则时，统计窗口必须大于 0')
      return false
    }
    if (sameOpponentRule.max_matches <= 0) {
      ElMessage.error('启用同对手高频规则时，最大允许场次必须大于 0')
      return false
    }
    if (sameOpponentRule.penalty_score <= 0) {
      ElMessage.error('启用同对手高频规则时，命中扣分必须大于 0')
      return false
    }
  }

  return true
}

const handleSave = async () => {
  if (!validateConfig()) return

  configSaving.value = true
  try {
    const res = await updateReputationConfig(config)
    ElMessage.success(res.message || '信誉制度配置已保存')
    await fetchConfig()
  } catch (error: any) {
    ElMessage.error(error.message || '保存信誉制度配置失败')
  } finally {
    configSaving.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<style scoped lang="scss">
.reputation-page {
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

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 16px;
}

.duration-rule-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.duration-rule-card {
  border-radius: 12px;
}

.duration-rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
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
  .config-grid {
    grid-template-columns: 1fr;
  }
}
</style>
