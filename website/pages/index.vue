<script setup lang="ts">
import FeatureGrid from '../components/FeatureGrid.vue'
import FaqList from '../components/FaqList.vue'
import DownloadPanel from '../components/DownloadPanel.vue'
import HeroSection from '../components/HeroSection.vue'
import ScenarioCards from '../components/ScenarioCards.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { homePageContent } from '../data/home'
import { siteConfig } from '../data/site'
import { usePageSeo } from '../composables/usePageSeo'

usePageSeo({
  title: '台球竞技记录 App',
  description:
    '追分帮助台球爱好者记录对局、生成战报、沉淀战绩，并快速进入官方下载入口。',
  path: '/'
})
</script>

<template>
  <div class="page-shell">
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell__main">
      <HeroSection
        :description="homePageContent.hero.description"
        :eyebrow="homePageContent.hero.eyebrow"
        :preview-cards="homePageContent.hero.previewCards"
        :primary-action="homePageContent.hero.primaryAction"
        :secondary-action="homePageContent.hero.secondaryAction"
        :title="homePageContent.hero.title"
      />

      <FeatureGrid
        id="features"
        :intro="'把记录、战报和持续进步这三件事放在同一条路径里，让普通台球爱好者也能认真留下每一次上桌。'"
        :items="homePageContent.values"
        title="不是简单记分，而是把一场球真正留下来"
      />

      <FeatureGrid
        :intro="'首页不堆砌功能说明书，只保留最能打动首次访问者的四个亮点。'"
        :items="homePageContent.highlights"
        title="四个足够直接的理由，让你知道为什么值得装"
      />

      <ScenarioCards :items="homePageContent.scenarios" />

      <DownloadPanel
        :action="homePageContent.downloadCallout.primaryAction"
        :title="homePageContent.downloadCallout.title"
      />

      <FaqList :items="homePageContent.faq" />
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell__main {
  width: min(1120px, calc(100vw - 32px));
  margin: 0 auto;
}
</style>
