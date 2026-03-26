<script setup lang="ts">
import type { SiteNavLink } from '../data/site'

defineProps<{
  links: SiteNavLink[]
}>()
</script>

<template>
  <footer class="site-footer">
    <div class="site-footer__inner">
      <div>
        <h2>追分</h2>
        <p>把对局、战绩和竞技感，认真留给真正爱打球的人。</p>
      </div>

      <nav class="site-footer__nav" aria-label="页脚导航">
        <NuxtLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="site-footer__link"
        >
          {{ link.label }}
        </NuxtLink>
      </nav>
    </div>
  </footer>
</template>

<style scoped>
.site-footer {
  padding: 28px 0 48px;
  border-top: 1px solid rgba(31, 26, 16, 0.08);
}

.site-footer__inner {
  width: min(1120px, calc(100vw - 32px));
  margin: 0 auto;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.site-footer h2 {
  margin: 0;
  font-size: 1.2rem;
}

.site-footer p {
  max-width: 24rem;
  margin: 10px 0 0;
  color: rgba(31, 26, 16, 0.68);
  line-height: 1.7;
}

.site-footer__nav {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 14px 18px;
}

.site-footer__link {
  color: rgba(31, 26, 16, 0.7);
}

@media (max-width: 720px) {
  .site-footer__inner {
    flex-direction: column;
  }

  .site-footer__nav {
    justify-content: flex-start;
  }
}
</style>
