import { defineStore } from 'pinia'
import { useRankStore } from './rank.js'
import { useNotificationStore } from './notification.js'
import { useFriendRequestStore } from './friendRequest.js'
import { userWS } from '@/utils/websocket.js'
import {
  clearStoredSession,
  clearUserScopedRuntimeState,
  readStoredSession
} from '@/utils/session-storage.js'

const clearCurrentUserRuntimeState = () => {
  clearUserScopedRuntimeState({
    rankStore: useRankStore(),
    notificationStore: useNotificationStore(),
    friendRequestStore: useFriendRequestStore(),
    userSocket: userWS
  })
}

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    refreshToken: '',
    userInfo: null,
    isLoggedIn: false,
    needBindPhone: false,
    expiresIn: 0,
    // 仅运行时使用；登录、退出或账号合并替换会话时递增，使旧异步结果失效。
    authGeneration: 0
  }),

  getters: {
    // 获取用户ID
    userId: (state) => state.userInfo?.id || 0,

    // 获取用户昵称
    nickname: (state) => state.userInfo?.nickname || '',

    // 获取用户头像
    avatar: (state) => state.userInfo?.avatar || '',

    // 获取脱敏手机号
    phone: (state) => state.userInfo?.phone || ''
  },

  actions: {
    // 登录
    login(data) {
      const nextUser = data.user || data.user_info
      this.authGeneration += 1
      clearCurrentUserRuntimeState()
      this.token = data.token || data.access_token
      this.refreshToken = data.refreshToken || data.refresh_token || ''
      this.userInfo = nextUser
      this.expiresIn = data.expiresIn || data.expires_in || 0
      this.isLoggedIn = true
      this.needBindPhone = data.needBindPhone || data.need_bind_phone || false

      // 存储 token 到本地
      uni.setStorageSync('token', this.token)
      if (this.refreshToken) {
        uni.setStorageSync('refreshToken', this.refreshToken)
      }
      userWS.connect().catch((error) => {
        console.error('[UserStore] 用户WS连接失败:', error)
      })
    },

    // 刷新登录态，只更新令牌，不覆盖用户资料
    refreshAuth(data) {
      this.token = data.token || data.access_token
      this.refreshToken = data.refreshToken || data.refresh_token || this.refreshToken
      this.expiresIn = data.expiresIn || data.expires_in || this.expiresIn
      this.isLoggedIn = true

      uni.setStorageSync('token', this.token)
      if (this.refreshToken) {
        uni.setStorageSync('refreshToken', this.refreshToken)
      }
    },

    // 退出登录：清空运行状态与全部持久化身份键，避免下次启动恢复旧用户
    logout() {
      this.authGeneration += 1
      clearCurrentUserRuntimeState()
      this.token = ''
      this.refreshToken = ''
      this.userInfo = null
      this.isLoggedIn = false
      this.needBindPhone = false
      this.expiresIn = 0

      clearStoredSession(uni)
    },

    // 设置是否需要绑定手机号
    setNeedBindPhone(value) {
      this.needBindPhone = value
    },

    // 更新用户信息
    updateUserInfo(userInfo) {
      this.userInfo = { ...this.userInfo, ...userInfo }
    },

    // 更新昵称
    updateNickname(nickname) {
      if (this.userInfo) {
        this.userInfo.nickname = nickname
      }
    },

    // 绑定手机号成功后更新
    bindPhoneSuccess(phone) {
      if (this.userInfo) {
        this.userInfo.phone = phone
      }
      this.needBindPhone = false
    },

    // 初始化（从本地存储恢复）
    init() {
      const { token, refreshToken, hasSession } = readStoredSession(uni)
      if (hasSession) {
        this.token = token
        this.isLoggedIn = true
        this.refreshToken = refreshToken || ''
      }
    }
  },

  // 持久化配置
  persist: {
    key: 'user-store',
    paths: ['token', 'refreshToken', 'userInfo', 'isLoggedIn', 'needBindPhone', 'expiresIn']
  }
})
