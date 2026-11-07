<script setup lang="ts">
interface PreviewCard {
  title: string
  value: string
  detail: string
}

defineProps<{
  eyebrow: string
  title: string
  description: string
  previewCards: PreviewCard[]
  primaryAction: {
    label: string
    to: string
  }
  secondaryAction: {
    label: string
    to: string
  }
}>()
</script>

<template>
  <section class="hero">
    <div class="hero__copy">
      <p class="hero__eyebrow">{{ eyebrow }}</p>
      <h1 class="hero__title">{{ title }}</h1>
      <p class="hero__description">{{ description }}</p>

      <div class="hero__actions">
        <NuxtLink class="hero__primary" :to="primaryAction.to">
          {{ primaryAction.label }}
        </NuxtLink>
        <a class="hero__secondary" :href="secondaryAction.to">
          {{ secondaryAction.label }}
        </a>
      </div>
    </div>

    <div class="hero__visual" aria-hidden="true">
      <div class="hero__phone-shell">
        <div class="hero__phone-top">
          <span class="hero__dot" />
          <span class="hero__dot hero__dot--small" />
        </div>
        <div class="hero__phone-body">
          <div
            v-for="card in previewCards"
            :key="card.title"
            class="hero__stat-card"
          >
            <span class="hero__stat-title">{{ card.title }}</span>
            <strong class="hero__stat-value">{{ card.value }}</strong>
            <span class="hero__stat-detail">{{ card.detail }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(280px, 0.95fr);
  gap: 40px;
  align-items: center;
  padding: 72px 0 48px;
}

.hero__eyebrow {
  width: fit-content;
  margin: 0 0 18px;
  padding: 8px 14px;
  border-radius: 999px;
  background: var(--ui-brand-tint);
  color: var(--ui-brand-strong);
  font-size: 0.92rem;
  font-weight: 600;
}

.hero__title {
  margin: 0;
  font-size: clamp(2.8rem, 6vw, 4.8rem);
  line-height: 0.98;
  letter-spacing: -0.05em;
}

.hero__description {
  max-width: 32rem;
  margin: 22px 0 0;
  color: rgba(31, 26, 16, 0.75);
  font-size: 1.08rem;
  line-height: 1.8;
}

.hero__actions {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  margin-top: 28px;
}

.hero__primary,
.hero__secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 148px;
  height: 52px;
  padding: 0 22px;
  border-radius: 999px;
  font-weight: 600;
}

.hero__primary {
  background: linear-gradient(135deg, var(--ui-brand-gradient-start), var(--ui-brand-gradient-end));
  color: #fff;
  box-shadow: var(--ui-shadow-primary);
}

.hero__secondary {
  border: 1px solid rgba(31, 26, 16, 0.1);
  background: rgba(255, 255, 255, 0.78);
}

.hero__visual {
  display: flex;
  justify-content: center;
}

.hero__phone-shell {
  width: min(100%, 360px);
  padding: 18px;
  border-radius: 32px;
  background:
    linear-gradient(180deg, rgba(35, 28, 11, 0.96), rgba(18, 15, 8, 0.98));
  box-shadow: 0 36px 60px rgba(31, 26, 16, 0.24);
}

.hero__phone-top {
  display: flex;
  justify-content: center;
  gap: 10px;
  padding-bottom: 16px;
}

.hero__dot {
  width: 54px;
  height: 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
}

.hero__dot--small {
  width: 8px;
}

.hero__phone-body {
  display: grid;
  gap: 14px;
  padding: 16px;
  border-radius: 24px;
  background:
    radial-gradient(circle at top right, rgba(224, 174, 18, 0.24), transparent 30%),
    linear-gradient(180deg, #241d0e 0%, #171208 100%);
}

.hero__stat-card {
  display: grid;
  gap: 6px;
  padding: 18px;
  border: 1px solid rgba(255, 247, 225, 0.08);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.04);
  color: var(--ui-text-on-accent);
}

.hero__stat-title,
.hero__stat-detail {
  color: rgba(255, 247, 225, 0.72);
}

.hero__stat-value {
  font-size: 1.85rem;
}

@media (max-width: 960px) {
  .hero {
    grid-template-columns: 1fr;
    padding-top: 48px;
  }
}
</style>
