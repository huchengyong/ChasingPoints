<script setup lang="ts">
import DownloadPanel from '../components/DownloadPanel.vue'
import PageHero from '../components/PageHero.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { downloadEntries, downloadPageContent } from '../data/download'
import { siteConfig } from '../data/site'
import { usePageSeo } from '../composables/usePageSeo'

usePageSeo({
  title: downloadPageContent.title,
  description: downloadPageContent.description,
  path: '/download'
})
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <PageHero
        eyebrow="下载入口"
        :title="downloadPageContent.title"
        :description="downloadPageContent.description"
      />

      <section class="download-grid">
        <article
          v-for="entry in downloadEntries"
          :key="entry.platform"
          class="download-card"
        >
          <div>
            <p class="download-card__platform">{{ entry.label }}</p>
            <h2>{{ entry.platform === 'ios' ? 'iPhone / iPad' : 'Android 手机' }}</h2>
            <p class="download-card__note">{{ entry.note }}</p>
          </div>

          <div class="download-card__actions">
            <a class="download-card__button" :href="entry.href" target="_blank" rel="noreferrer">
              立即获取
            </a>
            <a :href="entry.qrTargetUrl" target="_blank" rel="noreferrer">
              <img :alt="`${entry.label} 二维码`" :src="entry.qrImage" class="download-card__qr">
            </a>
          </div>
        </article>
      </section>

      <section class="download-notes">
        <h2>下载前先了解一下</h2>
        <ul>
          <li v-for="note in downloadPageContent.notes" :key="note">
            {{ note }}
          </li>
        </ul>
      </section>

      <DownloadPanel
        :action="{ label: '返回首页', to: '/' }"
        title="下载之后，就从下一场球开始认真记录。"
      />
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell {
  width: min(1120px, calc(100vw - 32px));
  margin: 0 auto;
}

.download-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  padding: 8px 0 32px;
}

.download-card,
.download-notes {
  padding: 28px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.download-card {
  display: flex;
  justify-content: space-between;
  gap: 18px;
}

.download-card__platform {
  margin: 0 0 10px;
  color: #9f6c00;
  font-weight: 600;
}

.download-card h2,
.download-notes h2 {
  margin: 0;
}

.download-card__note {
  margin: 14px 0 0;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
}

.download-card__actions {
  display: grid;
  gap: 14px;
  justify-items: center;
}

.download-card__button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 120px;
  height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  background: linear-gradient(135deg, #d9a617, #bb8400);
  color: #fff;
  font-weight: 700;
}

.download-card__qr {
  width: 132px;
  height: 132px;
  border-radius: 18px;
  border: 1px solid rgba(31, 26, 16, 0.1);
  background: #fff;
}

.download-notes {
  margin-bottom: 48px;
}

.download-notes ul {
  margin: 18px 0 0;
  padding-left: 20px;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.9;
}

@media (max-width: 900px) {
  .download-grid {
    grid-template-columns: 1fr;
  }

  .download-card {
    flex-direction: column;
  }
}
</style>
