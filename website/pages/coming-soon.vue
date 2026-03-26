<script setup lang="ts">
import PageHero from '../components/PageHero.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { usePageSeo } from '../composables/usePageSeo'
import { comingSoonContent } from '../data/coming-soon'
import { siteConfig } from '../data/site'

usePageSeo({
  title: comingSoonContent.title,
  description: comingSoonContent.description,
  path: '/coming-soon'
})
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <section class="coming-soon-card">
        <PageHero
          eyebrow="下载状态"
          :title="comingSoonContent.title"
          :description="comingSoonContent.description"
        />

        <p class="coming-soon-card__detail">
          {{ comingSoonContent.detail }}
        </p>

        <div class="coming-soon-card__actions">
          <NuxtLink class="coming-soon-card__primary" :to="comingSoonContent.primaryAction.to">
            {{ comingSoonContent.primaryAction.label }}
          </NuxtLink>
          <NuxtLink class="coming-soon-card__secondary" :to="comingSoonContent.secondaryAction.to">
            {{ comingSoonContent.secondaryAction.label }}
          </NuxtLink>
        </div>
      </section>
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell {
  width: min(880px, calc(100vw - 32px));
  margin: 0 auto;
  min-height: calc(100vh - 152px);
  display: flex;
  align-items: center;
}

.coming-soon-card {
  width: 100%;
  padding: 12px 32px 40px;
  border-radius: 32px;
  background:
    radial-gradient(circle at top right, rgba(224, 174, 18, 0.18), transparent 28%),
    rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(31, 26, 16, 0.08);
  box-shadow: 0 24px 50px rgba(31, 26, 16, 0.08);
}

.coming-soon-card__detail {
  margin: -4px 0 0;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
  font-size: 1.02rem;
}

.coming-soon-card__actions {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  margin-top: 28px;
}

.coming-soon-card__primary,
.coming-soon-card__secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 132px;
  height: 48px;
  padding: 0 20px;
  border-radius: 999px;
  font-weight: 600;
}

.coming-soon-card__primary {
  background: linear-gradient(135deg, #d9a617, #bb8400);
  color: #fff;
}

.coming-soon-card__secondary {
  border: 1px solid rgba(31, 26, 16, 0.1);
  background: rgba(255, 255, 255, 0.82);
  color: #1f1a10;
}
</style>
