<template>
  <div class="llm-page">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <div>
            <span>大模型配置</span>
            <div class="card-subtitle">配置后端调用大模型的接口格式、地址、API Key 与默认参数，鉴权统一使用 Authorization: Bearer。</div>
          </div>
        </div>
      </template>

      <el-form :model="form" label-width="120px" class="config-form" v-loading="loading">
        <el-form-item label="接口格式">
          <el-radio-group v-model="form.api_format">
            <el-radio-button v-for="option in API_FORMAT_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }}
            </el-radio-button>
          </el-radio-group>
          <div class="field-hint">{{ currentFormat.hint }}</div>
        </el-form-item>
        <el-form-item label="接口地址">
          <el-input v-model="form.endpoint_url" :placeholder="currentFormat.endpointExample" clearable />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input
            v-model="form.api_key"
            type="password"
            show-password
            autocomplete="new-password"
            :placeholder="apiKeyPlaceholder"
          />
          <div class="field-hint">API Key 保存后不会再回显明文；修改时重新填写即可，留空则保持不变。</div>
        </el-form-item>
        <el-form-item label="模型">
          <el-input v-model="form.model" placeholder="claude-haiku-4-5" clearable />
        </el-form-item>
        <el-form-item label="max_tokens">
          <el-input-number v-model="form.max_tokens" :min="1" :max="1000000" />
        </el-form-item>

        <div class="config-section-title">curl 预览</div>
        <pre class="curl-preview">{{ curlPreview }}</pre>

        <div class="config-actions">
          <el-button @click="fetchConfig">重载配置</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getLlmConfig, updateLlmConfig, type LlmApiFormat } from '@/api/llm'
import { buildLlmCurl } from '@/utils/llmCurl'

const DEFAULT_MAX_TOKENS = 1024

const API_FORMAT_OPTIONS: {
  value: LlmApiFormat
  label: string
  endpointExample: string
  hint: string
}[] = [
  {
    value: 'anthropic',
    label: 'Anthropic Messages',
    endpointExample: 'https://opencode.ai/inference/anthropic/v1/messages',
    hint: '请求路径 …/v1/messages，响应文本取 content[0].text。'
  },
  {
    value: 'openai',
    label: 'OpenAI 兼容',
    endpointExample: 'https://api.openai.com/v1/chat/completions',
    hint: '请求路径 …/v1/chat/completions（Chat Completions），响应文本取 choices[0].message.content。'
  }
]

const loading = ref(false)
const saving = ref(false)
const apiKeyMasked = ref('')
const form = reactive({
  api_format: 'anthropic' as LlmApiFormat,
  endpoint_url: '',
  api_key: '',
  model: '',
  max_tokens: DEFAULT_MAX_TOKENS as number | undefined
})

const currentFormat = computed(() => API_FORMAT_OPTIONS.find((option) => option.value === form.api_format) ?? API_FORMAT_OPTIONS[0])
const apiKeyPlaceholder = computed(() => (apiKeyMasked.value ? `已配置 ${apiKeyMasked.value}，留空则保持不变` : '请输入 API Key'))
const curlPreview = computed(() => buildLlmCurl(form))

// 请求层已统一弹出错误提示，这里只负责恢复页面状态。
const fetchConfig = async () => {
  loading.value = true
  try {
    const res = await getLlmConfig()
    // 未配置时后端返回空字符串，沿用默认的 Anthropic Messages
    form.api_format = res.api_format || 'anthropic'
    form.endpoint_url = res.endpoint_url
    form.api_key = ''
    form.model = res.model
    // 未配置时后端返回 0，回到示例中的默认值
    form.max_tokens = res.max_tokens > 0 ? res.max_tokens : DEFAULT_MAX_TOKENS
    apiKeyMasked.value = res.has_api_key ? res.api_key_masked : ''
  } catch {
    // 已由请求层提示
  } finally {
    loading.value = false
  }
}

const validateForm = () => {
  const endpoint = form.endpoint_url.trim()
  if (!endpoint) {
    ElMessage.error('请填写接口地址')
    return false
  }
  if (!/^https:\/\/\S+$/i.test(endpoint)) {
    ElMessage.error('接口地址必须以 https:// 开头')
    return false
  }
  if (!apiKeyMasked.value && !form.api_key.trim()) {
    ElMessage.error('请填写 API Key')
    return false
  }
  if (!form.model.trim()) {
    ElMessage.error('请填写模型名称')
    return false
  }
  if (!form.max_tokens || form.max_tokens < 1) {
    ElMessage.error('max_tokens 必须大于 0')
    return false
  }
  return true
}

const handleSave = async () => {
  if (!validateForm()) return

  saving.value = true
  try {
    const res = await updateLlmConfig({
      api_format: form.api_format,
      endpoint_url: form.endpoint_url.trim(),
      api_key: form.api_key.trim(),
      model: form.model.trim(),
      max_tokens: form.max_tokens as number
    })
    ElMessage.success(res.message || '大模型配置已保存')
    await fetchConfig()
  } catch {
    // 已由请求层提示
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<style scoped lang="scss">
.llm-page {
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

.field-hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: #6b7280;
}

.config-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
  margin-top: 4px;
}

.curl-preview {
  margin: 0;
  padding: 12px 16px;
  border-radius: 8px;
  background: #f3f4f6;
  color: #1f2937;
  font-size: 13px;
  line-height: 1.6;
  overflow-x: auto;
  white-space: pre;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
