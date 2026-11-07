<script setup lang="ts">
defineProps<{
  eyebrow: string
  title: string
  description: string
}>()
</script>

<template>
  <section class="page-hero">
    <p class="page-hero__eyebrow">{{ eyebrow }}</p>
    <h1 class="page-hero__title">{{ title }}</h1>
    <p class="page-hero__description">{{ description }}</p>
  </section>
</template>

<style scoped>
.page-hero {
  padding: 56px 0 28px;
}

.page-hero__eyebrow {
  width: fit-content;
  margin: 0 0 12px;
  padding: 8px 14px;
  border-radius: 999px;
  background: var(--ui-brand-tint);
  color: var(--ui-brand-strong);
  font-weight: 600;
}

.page-hero__title {
  margin: 0;
  font-size: clamp(2.5rem, 4vw, 4rem);
}

.page-hero__description {
  max-width: 42rem;
  margin: 18px 0 0;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
}
</style>
