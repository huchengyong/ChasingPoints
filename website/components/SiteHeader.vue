<script setup lang="ts">
import type { SiteNavLink } from '../data/site'

defineProps<{
  links: SiteNavLink[]
}>()
</script>

<template>
  <header class="site-header">
    <div class="site-header__inner">
      <NuxtLink class="site-header__brand" to="/">
        <img alt="追分 Logo" class="site-header__logo" src="/brand-logo.png">
        <span>追分</span>
      </NuxtLink>

      <nav class="site-header__nav" aria-label="官网导航">
        <NuxtLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="site-header__link"
        >
          {{ link.label }}
        </NuxtLink>
      </nav>

      <NuxtLink class="site-header__cta" to="/download">
        立即下载
      </NuxtLink>
    </div>
  </header>
</template>

<style scoped>
.site-header {
  position: sticky;
  top: 0;
  z-index: 20;
  backdrop-filter: blur(18px);
  background: rgba(247, 244, 236, 0.78);
  border-bottom: 1px solid rgba(31, 26, 16, 0.08);
}

.site-header__inner {
  width: min(1120px, calc(100vw - 32px));
  margin: 0 auto;
  min-height: 76px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.site-header__brand {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  font-size: 1.05rem;
  font-weight: 700;
}

.site-header__logo {
  width: 40px;
  height: 40px;
  border-radius: 10px;
}

.site-header__nav {
  display: flex;
  align-items: center;
  gap: 20px;
  color: rgba(31, 26, 16, 0.76);
}

.site-header__link:hover,
.site-header__link:focus-visible {
  color: #1f1a10;
}

.site-header__cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 112px;
  height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  background: linear-gradient(135deg, #d9a617, #bb8400);
  color: #fff;
  font-weight: 600;
  box-shadow: 0 12px 30px rgba(187, 132, 0, 0.22);
}

@media (max-width: 860px) {
  .site-header__nav {
    display: none;
  }
}
</style>
