<script setup lang="ts">
import LegalArticle from '../components/LegalArticle.vue'
import PageHero from '../components/PageHero.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { privacyPolicy } from '../data/legal'
import { siteConfig } from '../data/site'
import { usePageSeo } from '../composables/usePageSeo'

usePageSeo({
  title: privacyPolicy.title,
  description: privacyPolicy.intro,
  path: '/privacy'
})
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <PageHero
        eyebrow="隐私与数据"
        :title="privacyPolicy.title"
        :description="privacyPolicy.intro"
      />
      <LegalArticle :document="privacyPolicy" />
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell {
  width: min(960px, calc(100vw - 32px));
  margin: 0 auto;
}
</style>
