import { createPinia } from 'pinia'
import { createPersistedState } from 'pinia-plugin-persistedstate'

const pinia = createPinia()

// 使用持久化插件
pinia.use(createPersistedState({
  storage: {
    getItem: uni.getStorageSync,
    setItem: uni.setStorageSync,
  }
}))

export default pinia
