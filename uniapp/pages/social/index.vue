<template>
  <!-- 旧赛讯 tabbar 路径兼容跳转：已发布二维码/分享链接/平台配置引用统一重定向到当前赛讯列表 -->
  <view class="social-compat" :class="{ 'dark-mode': isDarkMode }"></view>
</template>

<script setup>
import { onLoad } from '@dcloudio/uni-app'
import { usePageTheme } from '@/utils/page-theme.js'

const { isDarkMode } = usePageTheme()

onLoad(() => {
  uni.switchTab({ url: '/pages/tournament/index' })
})
</script>

<style scoped>
.social-compat {
  background: #F7F4EC;
}
.social-compat.dark-mode {
  background: #141109;
}
</style>