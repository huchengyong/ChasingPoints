export default defineNuxtConfig({
  compatibilityDate: '2025-03-01',
  devtools: {
    enabled: false
  },
  css: ['~/assets/styles/main.css'],
  app: {
    head: {
      htmlAttrs: {
        lang: 'zh-CN'
      },
      viewport: 'width=device-width, initial-scale=1'
    }
  },
  nitro: {
    prerender: {
      crawlLinks: true,
      routes: ['/coming-soon']
    }
  },
  runtimeConfig: {
    public: {
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'https://www.zhuifen.cn'
    }
  }
})
