# WEBSITE FRONTEND GUIDE

## OVERVIEW
`website/` 是追分官网子项目，技术栈为 Nuxt 3 + Vue 3 + TypeScript。当前首期职责是承接品牌首页、下载页、隐私政策、用户协议和联系页，并提供基础 SEO 能力。

## STRUCTURE
```text
website/
├── AGENTS.md
├── README.md
├── app.vue
├── assets/styles/
├── components/
├── composables/
├── data/
├── pages/
├── public/
├── tests/
└── nuxt.config.ts
```

## CURRENT FACTS
- 官网当前使用 `Nuxt 3.21.2`。
- 页面 SEO 统一走 `composables/usePageSeo.ts`。
- 首页与次级页都依赖 `data/*.ts` 作为文案和配置来源。
- 正式域名通过 `NUXT_PUBLIC_SITE_URL` 注入；未配置时默认使用 `https://www.zhuifen.cn`。
- 下载链接、联系信息和备案号通过 `NUXT_PUBLIC_IOS_DOWNLOAD_URL`、`NUXT_PUBLIC_ANDROID_DOWNLOAD_URL`、`NUXT_PUBLIC_CONTACT_EMAIL`、`NUXT_PUBLIC_CONTACT_PHONE`、`NUXT_PUBLIC_ICP_RECORD_NUMBER` 注入。

## CONVENTIONS
- 用简体中文沟通。
- 公开页面视觉遵循仓库根级 [DESIGN.md](/Users/wisesearch/Projects/ChasingPoints/DESIGN.md)，品牌、颜色、圆角、间距与组件状态从 `assets/styles/main.css` 的语义 Token 消费。
- 不要把下载链接、联系方式或协议正文直接散写在页面组件里，优先维护 `data/*.ts`。
- 静态 SEO 资产放在 `public/`，包含 `robots.txt`、`sitemap.xml` 和分享图。
- 页面路由保持短路径：`/`、`/download`、`/privacy`、`/agreement`、`/contact`。

## COMMANDS
```bash
cd website

npm install
npm run dev
npm run build
npm run generate
npm run test
```

## REVIEW HOTSPOTS
- 路由页面：`website/pages/*.vue`
- 内容与文案：`website/data/*.ts`
- SEO 配置：`website/composables/usePageSeo.ts`
- 站点配置：`website/nuxt.config.ts`

## ANTI-PATTERNS
- 不要把官网做成 `admin/` 的子路由或复用后台布局。
- 不要直接在多个页面里重复写相同 SEO meta。
- 不要把构建产物 `.nuxt/`、`.output/`、`node_modules/` 提交进版本控制。
