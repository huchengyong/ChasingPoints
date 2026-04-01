<template>
  <div class="login-container">
    <el-card class="login-box">
      <template #header>
        <div class="login-header">
          <el-icon size="32"><Trophy /></el-icon>
          <h2>追分管理后台</h2>
        </div>
      </template>

      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        @keyup.enter="handleLogin"
      >
        <el-form-item prop="email">
          <el-input
            v-model="loginForm.email"
            placeholder="邮箱"
            :prefix-icon="Message"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="密码"
            :prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="handleLogin"
            style="width: 100%"
          >
            登录
          </el-button>
        </el-form-item>

        <el-form-item v-if="!adminExists && !checking">
          <el-button
            size="large"
            @click="initDialogVisible = true"
            style="width: 100%"
          >
            首次初始化管理员
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-tips">
        <p v-if="!adminExists && !checking">首次初始化需要输入管理员邮箱、密码和部署侧配置的 `ADMIN_SETUP_TOKEN`。</p>
        <p v-if="!adminExists && !checking">初始化成功后，请使用刚创建的管理员账号登录。</p>
        <p v-if="adminExists && !checking">请使用管理员账号登录系统。</p>
      </div>
    </el-card>

    <el-dialog
      v-model="initDialogVisible"
      title="首次初始化管理员"
      width="420px"
      destroy-on-close
    >
      <el-form ref="initFormRef" :model="initForm" :rules="initRules" label-position="top">
        <el-form-item prop="email" label="管理员邮箱">
          <el-input
            v-model="initForm.email"
            placeholder="请输入管理员邮箱"
            :prefix-icon="Message"
          />
        </el-form-item>

        <el-form-item prop="password" label="管理员密码">
          <el-input
            v-model="initForm.password"
            type="password"
            placeholder="请输入8-72位密码"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item prop="setupToken" label="Setup Token">
          <el-input
            v-model="initForm.setupToken"
            type="password"
            placeholder="请输入 ADMIN_SETUP_TOKEN"
            :prefix-icon="Key"
            show-password
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="initDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="initLoading" @click="handleInitAdmin">
            确认初始化
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Key, Lock, Message } from '@element-plus/icons-vue'
import { initAdmin, login, checkAdminExists } from '@/api/auth'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()

const loginFormRef = ref()
const initFormRef = ref()
const loading = ref(false)
const initLoading = ref(false)
const initDialogVisible = ref(false)
const adminExists = ref(true) // 默认认为已存在，避免闪烁显示
const checking = ref(true)

const loginForm = reactive({
  email: '',
  password: ''
})

const initForm = reactive({
  email: '',
  password: '',
  setupToken: ''
})

const loginRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const initRules = {
  email: loginRules.email,
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, max: 72, message: '密码长度需为8-72位', trigger: 'blur' }
  ],
  setupToken: [{ required: true, message: '请输入 setup token', trigger: 'blur' }]
}

const handleLogin = async () => {
  const valid = await loginFormRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true

  try {
    const res = await login({
      email: loginForm.email.trim(),
      password: loginForm.password
    })

    userStore.setToken(res.token)
    userStore.setUserInfo(res.userInfo)
    ElMessage.success('登录成功')
    router.push('/')
  } catch (error: any) {
    ElMessage.error(error.message || '登录失败')
  } finally {
    loading.value = false
  }
}

const handleInitAdmin = async () => {
  const valid = await initFormRef.value?.validate().catch(() => false)
  if (!valid) return

  initLoading.value = true

  try {
    const res = await initAdmin({
      email: initForm.email.trim(),
      password: initForm.password,
      setupToken: initForm.setupToken.trim()
    })

    ElMessage.success(res.message || '管理员账号初始化成功')
    loginForm.email = initForm.email.trim()
    loginForm.password = initForm.password
    initForm.setupToken = ''
    initDialogVisible.value = false
    adminExists.value = true // 初始化成功后更新状态
  } catch (error: any) {
    ElMessage.error(error.message || '初始化失败')
  } finally {
    initLoading.value = false
  }
}

// 检查管理员是否存在
const checkAdmin = async () => {
  checking.value = true
  try {
    const res = await checkAdminExists()
    adminExists.value = res.exists
  } catch (error) {
    // 请求失败时默认认为已存在（安全起见）
    adminExists.value = true
  } finally {
    checking.value = false
  }
}

// 页面加载时检查
onMounted(() => {
  checkAdmin()
})
</script>

<style scoped lang="scss">
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-box {
  width: 400px;

  .login-header {
    text-align: center;

    .el-icon {
      color: #409eff;
      margin-bottom: 8px;
    }

    h2 {
      margin: 0;
      font-size: 20px;
      color: #303133;
    }
  }

  .login-tips {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #e4e7ed;
    text-align: center;

    p {
      margin: 4px 0;
      font-size: 13px;
      color: #606266;
      line-height: 1.6;
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
