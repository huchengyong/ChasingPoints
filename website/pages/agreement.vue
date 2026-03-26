<script setup lang="ts">
import LegalArticle from '../components/LegalArticle.vue'
import PageHero from '../components/PageHero.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { siteConfig } from '../data/site'
import { userAgreement } from '../data/legal'
import { usePageSeo } from '../composables/usePageSeo'

usePageSeo({
  title: userAgreement.title,
  description: userAgreement.intro,
  path: '/agreement'
})
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <PageHero
        eyebrow="使用条款"
        :title="userAgreement.title"
        :description="userAgreement.intro"
      />
      <LegalArticle :document="userAgreement" />
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
