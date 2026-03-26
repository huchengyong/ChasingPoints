<script setup lang="ts">
import type { LegalDocument } from '../data/legal'

defineProps<{
  document: LegalDocument
}>()
</script>

<template>
  <article class="legal-article">
    <header class="legal-article__header">
      <h2>{{ document.title }}</h2>
      <p class="legal-article__date">更新日期：{{ document.updatedAt }}</p>
      <p class="legal-article__intro">{{ document.intro }}</p>
    </header>

    <section
      v-for="section in document.sections"
      :key="section.title"
      class="legal-article__section"
    >
      <h3>{{ section.title }}</h3>
      <p
        v-for="paragraph in section.paragraphs"
        :key="paragraph"
      >
        {{ paragraph }}
      </p>
    </section>
  </article>
</template>

<style scoped>
.legal-article {
  padding: 20px 0 72px;
}

.legal-article__header {
  padding: 28px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.legal-article__header h2 {
  margin: 0;
  font-size: clamp(2rem, 3vw, 3rem);
}

.legal-article__date,
.legal-article__intro,
.legal-article__section p {
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.9;
}

.legal-article__date {
  margin: 12px 0 0;
}

.legal-article__intro {
  margin: 18px 0 0;
}

.legal-article__section {
  margin-top: 18px;
  padding: 24px 28px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.65);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.legal-article__section h3 {
  margin: 0;
  font-size: 1.2rem;
}

.legal-article__section p {
  margin: 14px 0 0;
}
</style>
