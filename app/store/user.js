import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    refreshToken: '',
    userInfo: null,
    isLoggedIn: false,
    needBindPhone: false,
    expiresIn: 0
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
      this.token = data.token || data.access_token
      this.refreshToken = data.refreshToken || data.refresh_token || ''
      this.userInfo = data.user || data.user_info
      this.expiresIn = data.expiresIn || data.expires_in || 0
      this.isLoggedIn = true
      this.needBindPhone = data.needBindPhone || data.need_bind_phone || false

      // 存储 token 到本地
      uni.setStorageSync('token', this.token)
      if (this.refreshToken) {
        uni.setStorageSync('refreshToken', this.refreshToken)
      }
    },

    // 退出登录
    logout() {
      this.token = ''
      this.refreshToken = ''
      this.userInfo = null
      this.isLoggedIn = false
      this.needBindPhone = false
      this.expiresIn = 0

      // 清除本地存储
      uni.removeStorageSync('token')
      uni.removeStorageSync('refreshToken')
      uni.removeStorageSync('loginMethod')
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
      const token = uni.getStorageSync('token')
      if (token) {
        this.token = token
        this.isLoggedIn = true
        this.refreshToken = uni.getStorageSync('refreshToken') || ''
      }
    }
  },

  // 持久化配置
  persist: {
    key: 'user-store',
    paths: ['token', 'refreshToken', 'userInfo', 'isLoggedIn', 'needBindPhone']
  }
})
