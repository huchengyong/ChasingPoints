<script setup lang="ts">
interface FaqItem {
  question: string
  answer: string
}

defineProps<{
  items: FaqItem[]
}>()
</script>

<template>
  <section class="faq-list">
    <div class="section-heading">
      <p class="section-heading__eyebrow">常见问题</p>
      <h2 class="section-heading__title">先把最常见的问题说清楚，再决定要不要下载</h2>
    </div>

    <div class="faq-list__items">
      <details
        v-for="item in items"
        :key="item.question"
        class="faq-list__item"
      >
        <summary>{{ item.question }}</summary>
        <p>{{ item.answer }}</p>
      </details>
    </div>
  </section>
</template>

<style scoped>
.faq-list {
  padding: 48px 0 72px;
}

.section-heading__eyebrow {
  margin: 0 0 12px;
  color: var(--ui-brand-strong);
  font-weight: 600;
}

.section-heading__title {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
  max-width: 40rem;
}

.faq-list__items {
  display: grid;
  gap: 14px;
  margin-top: 28px;
}

.faq-list__item {
  padding: 20px 22px;
  border: 1px solid rgba(31, 26, 16, 0.08);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.74);
}

.faq-list__item summary {
  cursor: pointer;
  font-weight: 600;
  list-style: none;
}

.faq-list__item summary::-webkit-details-marker {
  display: none;
}

.faq-list__item p {
  margin: 14px 0 0;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
}
</style>
