<script setup lang="ts">
defineProps<{
  title: string
  action: {
    label: string
    to: string
  }
}>()
</script>

<template>
  <section class="download-panel">
    <div class="download-panel__copy">
      <p class="download-panel__eyebrow">下载引导</p>
      <h2>{{ title }}</h2>
    </div>

    <NuxtLink class="download-panel__button" :to="action.to">
      {{ action.label }}
    </NuxtLink>
  </section>
</template>

<style scoped>
.download-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 32px;
  border-radius: 28px;
  background:
    radial-gradient(circle at top right, rgba(224, 174, 18, 0.24), transparent 34%),
    linear-gradient(135deg, #1d170b, #2a2110);
  color: #fff7e1;
}

.download-panel__eyebrow {
  margin: 0 0 10px;
  color: rgba(255, 247, 225, 0.7);
  font-weight: 600;
}

.download-panel h2 {
  margin: 0;
  max-width: 34rem;
  font-size: clamp(1.8rem, 3vw, 2.6rem);
}

.download-panel__button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 148px;
  height: 52px;
  padding: 0 22px;
  border-radius: 999px;
  background: linear-gradient(135deg, #d9a617, #bb8400);
  color: #fff;
  font-weight: 700;
  white-space: nowrap;
}

@media (max-width: 720px) {
  .download-panel {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
