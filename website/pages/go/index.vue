<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageHero from '../../components/PageHero.vue'
import SiteFooter from '../../components/SiteFooter.vue'
import SiteHeader from '../../components/SiteHeader.vue'
import { siteConfig } from '../../data/site'

type EntryVariant = 'home' | 'invite' | 'match'

const route = useRoute()

// 预渲染时没有查询参数，挂载后再按链接切换文案，避免水合不一致。
const variant = ref<EntryVariant>('home')

const entryCopy: Record<EntryVariant, { eyebrow: string, title: string, description: string }> = {
  home: {
    eyebrow: '微信入口',
    title: '在微信里打开追分',
    description: '扫码或长按识别小程序码，记录对局、生成战报，把每一场球留下来。'
  },
  invite: {
    eyebrow: '球友邀请',
    title: '球友邀请你打一局',
    description: '用微信打开追分小程序，登录后直接进入对局确认，不用再扫码。'
  },
  match: {
    eyebrow: '对局分享',
    title: '这场对局值得一看',
    description: '用微信打开追分小程序，查看比分、局记录与双方数据。'
  }
}

const content = computed(() => entryCopy[variant.value])

onMounted(() => {
  const { j, m } = route.query
  if (typeof j === 'string' && j) {
    variant.value = 'invite'
  } else if (typeof m === 'string' && m) {
    variant.value = 'match'
  }
})

useSeoMeta({
  title: '在微信里打开追分 | 追分官网',
  description: '微信扫码打开追分小程序，记录对局与战报。',
  robots: 'noindex, nofollow'
})
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <PageHero
        :eyebrow="content.eyebrow"
        :title="content.title"
        :description="content.description"
      />

      <section class="entry-card">
        <img class="entry-card__qr" src="/qr-miniprogram.jpg" alt="追分小程序码">

        <div class="entry-card__body">
          <p class="entry-card__hint">打开微信扫一扫，或长按识别小程序码</p>
          <p class="entry-card__note">
            小程序打开即用，不需要额外安装。鸿蒙版上架前，先用小程序记录你的对局。
          </p>
          <NuxtLink class="entry-card__link" to="/download">
            查看其他平台
          </NuxtLink>
        </div>
      </section>
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell {
  width: min(920px, calc(100vw - 32px));
  margin: 0 auto;
}

.entry-card {
  display: flex;
  align-items: center;
  gap: 32px;
  margin: 8px 0 48px;
  padding: 32px;
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.entry-card__qr {
  width: 200px;
  height: 200px;
  border-radius: 20px;
  border: 1px solid rgba(31, 26, 16, 0.1);
  background: #fff;
}

.entry-card__hint {
  margin: 0;
  font-weight: 600;
}

.entry-card__note {
  margin: 14px 0 24px;
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
}

.entry-card__link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 140px;
  height: 44px;
  padding: 0 20px;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--ui-brand-gradient-start), var(--ui-brand-gradient-end));
  color: #fff;
  font-weight: 700;
}

@media (max-width: 900px) {
  .entry-card {
    flex-direction: column;
    text-align: center;
  }

  .entry-card__qr {
    width: 160px;
    height: 160px;
  }
}
</style>
