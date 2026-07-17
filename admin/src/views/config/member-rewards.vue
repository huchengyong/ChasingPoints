<template>
  <div class="member-rewards-page">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <span>会员奖励配置</span>
        </div>
      </template>

      <el-form :model="rewardConfig" label-width="120px" class="config-form" v-loading="configLoading">
        <div class="config-section-title">常玩球馆奖励</div>
        <div class="config-grid">
          <el-form-item label="奖励开关">
            <el-switch v-model="rewardConfig.enabled" />
          </el-form-item>
          <el-form-item label="弹窗开关">
            <el-switch v-model="rewardConfig.popup_enabled" />
          </el-form-item>
          <el-form-item label="奖励天数">
            <el-input-number v-model="rewardConfig.reward_days" :min="1" :max="365" />
          </el-form-item>
          <el-form-item label="开始时间">
            <el-date-picker
              v-model="rewardConfig.start_at"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="可选"
            />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-date-picker
              v-model="rewardConfig.end_at"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="可选"
            />
          </el-form-item>
        </div>

        <div class="config-section-title">新用户注册奖励</div>
        <div class="config-grid">
          <el-form-item label="奖励开关">
            <el-switch v-model="rewardConfig.welcome_reward_enabled" />
          </el-form-item>
          <el-form-item label="奖励天数">
            <el-input-number v-model="rewardConfig.welcome_reward_days" :min="1" :max="365" />
          </el-form-item>
        </div>

        <div class="config-actions">
          <el-button type="primary" :loading="configSaving" @click="handleSaveRewardConfig">保存配置</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getVenueRewardConfig, updateVenueRewardConfig, type VenueRewardConfig } from '@/api/venue'

const configLoading = ref(false)
const configSaving = ref(false)
const rewardConfig = reactive<VenueRewardConfig>({
  enabled: false,
  popup_enabled: false,
  reward_days: 30,
  welcome_reward_enabled: false,
  welcome_reward_days: 7,
  start_at: '',
  end_at: ''
})

const fetchRewardConfig = async () => {
  configLoading.value = true
  try {
    const res = await getVenueRewardConfig()
    if (res.success) {
      rewardConfig.enabled = !!res.enabled
      rewardConfig.popup_enabled = !!res.popup_enabled
      rewardConfig.reward_days = res.reward_days || 30
      rewardConfig.welcome_reward_enabled = res.welcome_reward_enabled ?? false
      rewardConfig.welcome_reward_days = res.welcome_reward_days || 7
      rewardConfig.start_at = res.start_at || ''
      rewardConfig.end_at = res.end_at || ''
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取奖励配置失败')
  } finally {
    configLoading.value = false
  }
}

const handleSaveRewardConfig = async () => {
  configSaving.value = true
  try {
    const res = await updateVenueRewardConfig({ ...rewardConfig })
    if (res.success) {
      ElMessage.success(res.message || '奖励配置已保存')
      fetchRewardConfig()
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '保存失败')
  } finally {
    configSaving.value = false
  }
}

onMounted(() => {
  fetchRewardConfig()
})
</script>

<style scoped lang="scss">
.member-rewards-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.config-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
