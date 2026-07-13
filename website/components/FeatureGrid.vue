<script setup lang="ts">
interface FeatureItem {
  title: string
  description: string
}

defineProps<{
  id?: string
  title: string
  intro: string
  items: FeatureItem[]
}>()
</script>

<template>
  <section :id="id" class="feature-grid">
    <div class="section-heading">
      <p class="section-heading__eyebrow">核心价值</p>
      <h2 class="section-heading__title">{{ title }}</h2>
      <p class="section-heading__intro">{{ intro }}</p>
    </div>

    <div class="feature-grid__cards">
      <article
        v-for="item in items"
        :key="item.title"
        class="feature-grid__card"
      >
        <h3>{{ item.title }}</h3>
        <p>{{ item.description }}</p>
      </article>
    </div>
  </section>
</template>

<style scoped>
.feature-grid {
  padding: 48px 0;
}

.section-heading {
  max-width: 42rem;
}

.section-heading__eyebrow {
  margin: 0 0 12px;
  color: #9f6c00;
  font-weight: 600;
}

.section-heading__title {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
}

.section-heading__intro {
  margin: 16px 0 0;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
}

.feature-grid__cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
  margin-top: 28px;
}

.feature-grid__card {
  padding: 24px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(31, 26, 16, 0.08);
  box-shadow: 0 18px 40px rgba(31, 26, 16, 0.06);
}

.feature-grid__card h3 {
  margin: 0;
  font-size: 1.2rem;
}

.feature-grid__card p {
  margin: 12px 0 0;
  color: rgba(31, 26, 16, 0.7);
  line-height: 1.7;
}

@media (max-width: 960px) {
  .feature-grid__cards {
    grid-template-columns: 1fr;
  }
}
</style>
