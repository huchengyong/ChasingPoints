<script setup lang="ts">
import PageHero from '../components/PageHero.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { contactInfo } from '../data/contact'
import { siteConfig } from '../data/site'
import { usePageSeo } from '../composables/usePageSeo'

usePageSeo({
  title: contactInfo.title,
  description: contactInfo.description,
  path: '/contact'
})
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <PageHero
        eyebrow="沟通与合作"
        :title="contactInfo.title"
        :description="contactInfo.description"
      />

      <section class="contact-card">
        <div>
          <p class="contact-card__label">联系邮箱</p>
          <a :href="`mailto:${contactInfo.email}`">{{ contactInfo.email }}</a>
        </div>

        <div>
          <p class="contact-card__label">座机</p>
          <a :href="`tel:${contactInfo.phone}`">{{ contactInfo.phone }}</a>
        </div>

        <div>
          <p class="contact-card__label">合作说明</p>
          <p>{{ contactInfo.businessNote }}</p>
        </div>
      </section>
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell {
  width: min(960px, calc(100vw - 32px));
  margin: 0 auto;
  padding-bottom: 72px;
}

.contact-card {
  display: grid;
  gap: 18px;
  padding: 28px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.contact-card__label {
  margin: 0 0 10px;
  color: #9f6c00;
  font-weight: 600;
}

.contact-card a,
.contact-card p:last-child {
  color: rgba(31, 26, 16, 0.78);
  line-height: 1.8;
}
</style>
