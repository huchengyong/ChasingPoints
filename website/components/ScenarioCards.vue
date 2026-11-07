<script setup lang="ts">
interface ScenarioItem {
  title: string
  description: string
}

defineProps<{
  items: ScenarioItem[]
}>()
</script>

<template>
  <section class="scenario-cards">
    <div class="section-heading">
      <p class="section-heading__eyebrow">真实场景</p>
      <h2 class="section-heading__title">不是为了记一场球，而是为了留下你的竞技轨迹</h2>
    </div>

    <div class="scenario-cards__list">
      <article
        v-for="(item, index) in items"
        :key="item.title"
        class="scenario-cards__item"
      >
        <span class="scenario-cards__index">0{{ index + 1 }}</span>
        <h3>{{ item.title }}</h3>
        <p>{{ item.description }}</p>
      </article>
    </div>
  </section>
</template>

<style scoped>
.scenario-cards {
  padding: 48px 0;
}

.section-heading__eyebrow {
  margin: 0 0 12px;
  color: var(--ui-brand-strong);
  font-weight: 600;
}

.section-heading__title {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
  max-width: 44rem;
}

.scenario-cards__list {
  display: grid;
  gap: 16px;
  margin-top: 28px;
}

.scenario-cards__item {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 20px;
  align-items: start;
  padding: 24px;
  border-radius: 24px;
  background: var(--ui-surface-accent);
  color: var(--ui-text-on-accent);
}

.scenario-cards__index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  border-radius: 20px;
  background: linear-gradient(135deg, var(--ui-brand-gradient-start), var(--ui-brand-gradient-end));
  color: #fff;
  font-size: 1.2rem;
  font-weight: 700;
}

.scenario-cards__item h3 {
  margin: 0;
  font-size: 1.25rem;
}

.scenario-cards__item p {
  margin: 12px 0 0;
  color: rgba(255, 247, 225, 0.72);
  line-height: 1.7;
}

@media (max-width: 720px) {
  .scenario-cards__item {
    grid-template-columns: 1fr;
  }
}
</style>
