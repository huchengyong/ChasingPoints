# Website Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a standalone `website/` Nuxt 3 marketing site that introduces the Chasing Points app, supports SEO basics, and routes users into iOS and Android download flows.

**Architecture:** Add a new root-level `website/` workspace using Nuxt 3 static generation. Keep page content in shared data modules so homepage, download, FAQ, and legal pages stay maintainable. Reuse the existing app legal copy as the content baseline, but present it through web-first layouts and page-specific SEO metadata.

**Tech Stack:** Nuxt 3, Vue 3, TypeScript, static generation, npm, Vitest for lightweight content/helper tests, existing repository docs workflow.

---

### Task 1: Scaffold the standalone `website/` workspace

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/package.json`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/nuxt.config.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/tsconfig.json`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/app.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/assets/styles/main.css`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/vitest.config.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/tests/site-config.test.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/data/site.ts`

**Step 1: Write the failing test**

Create `/Users/wisesearch/Projects/ChasingPoints/website/tests/site-config.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { siteConfig } from '../data/site'

describe('siteConfig', () => {
  it('contains the core marketing routes', () => {
    expect(siteConfig.navLinks.map((item) => item.to)).toEqual([
      '/',
      '/download',
      '/privacy',
      '/agreement',
      '/contact'
    ])
  })
})
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/site-config.test.ts
```

Expected: FAIL because the `website/` workspace and `siteConfig` module do not exist yet.

**Step 3: Write minimal implementation**

- Initialize `website/package.json` with `dev`, `build`, `generate`, and `test` scripts.
- Add Nuxt and Vitest dependencies.
- Create `nuxt.config.ts` with:
  - static generation enabled
  - global CSS registration for `assets/styles/main.css`
  - base site URL placeholder for later deployment config
- Create `data/site.ts` with:
  - brand name
  - default site title template
  - canonical route list
  - footer/legal navigation items
- Create a minimal `app.vue` that renders `NuxtPage`.

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/site-config.test.ts
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/website
git commit -m "feat: scaffold website workspace"
```

### Task 2: Add shared marketing content and SEO helpers

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/data/home.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/data/download.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/data/legal.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/data/contact.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/composables/usePageSeo.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/tests/page-seo.test.ts`
- Reference: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/agreement/privacyPolicy.vue`
- Reference: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/agreement/userAgreement.vue`

**Step 1: Write the failing test**

Create `/Users/wisesearch/Projects/ChasingPoints/website/tests/page-seo.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { buildPageSeo } from '../composables/usePageSeo'

describe('buildPageSeo', () => {
  it('returns download page metadata with canonical path', () => {
    const result = buildPageSeo({
      title: '下载追分',
      description: 'iOS 与 Android 下载入口',
      path: '/download'
    })

    expect(result.title).toContain('下载追分')
    expect(result.ogUrl).toContain('/download')
  })
})
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/page-seo.test.ts
```

Expected: FAIL because `buildPageSeo` does not exist yet.

**Step 3: Write minimal implementation**

- Create `usePageSeo.ts` with a pure `buildPageSeo()` helper and a Nuxt-facing `usePageSeo()` wrapper.
- Add shared content modules:
  - `home.ts` for hero copy, features, scenarios, FAQ
  - `download.ts` for iOS/Android links, QR labels, download notes
  - `legal.ts` for privacy/agreement updated dates and body sections
  - `contact.ts` for email / WeChat / business contact placeholders
- Convert the current app legal copy into structured arrays of headings and paragraphs instead of leaving it embedded in Vue templates.
- Keep download links and contact info centralized so future edits happen in one place.

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/page-seo.test.ts
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/website/data /Users/wisesearch/Projects/ChasingPoints/website/composables /Users/wisesearch/Projects/ChasingPoints/website/tests
git commit -m "feat: add website content and seo helpers"
```

### Task 3: Build the shared layout and homepage

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/SiteHeader.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/HeroSection.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/FeatureGrid.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/ScenarioCards.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/DownloadPanel.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/FaqList.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/SiteFooter.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/pages/index.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/tests/home-content.test.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/public/brand-logo.png`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/public/og-cover.jpg`

**Step 1: Write the failing test**

Create `/Users/wisesearch/Projects/ChasingPoints/website/tests/home-content.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { homePageContent } from '../data/home'

describe('homePageContent', () => {
  it('keeps the primary CTA pointed at the download page', () => {
    expect(homePageContent.hero.primaryAction.to).toBe('/download')
  })

  it('includes exactly four product highlight items', () => {
    expect(homePageContent.highlights).toHaveLength(4)
  })
})
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/home-content.test.ts
```

Expected: FAIL until the homepage content contract is implemented.

**Step 3: Write minimal implementation**

- Create a global shell with:
  - sticky site header
  - consistent max-width container
  - footer legal navigation
- Build the homepage in this order:
  - hero
  - core value/highlight grid
  - scenario cards
  - secondary download callout
  - FAQ
- Keep the visual tone simple and trustworthy:
  - real product screenshots over abstract artwork
  - clear CTA hierarchy
  - responsive layout for mobile first, desktop refined second
- Set page SEO with homepage-specific title and description.

**Step 4: Run tests and a build**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/home-content.test.ts
npm run build
```

Expected: PASS for the test and a successful Nuxt build.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/website/components /Users/wisesearch/Projects/ChasingPoints/website/pages/index.vue /Users/wisesearch/Projects/ChasingPoints/website/public /Users/wisesearch/Projects/ChasingPoints/website/tests/home-content.test.ts
git commit -m "feat: build website homepage"
```

### Task 4: Implement download, legal, and contact pages

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/pages/download.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/pages/privacy.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/pages/agreement.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/pages/contact.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/LegalArticle.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/components/PageHero.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/tests/legal-content.test.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/public/qr-ios.png`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/public/qr-android.png`

**Step 1: Write the failing test**

Create `/Users/wisesearch/Projects/ChasingPoints/website/tests/legal-content.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { privacyPolicy, userAgreement } from '../data/legal'

describe('legal content', () => {
  it('keeps updated dates for both legal documents', () => {
    expect(privacyPolicy.updatedAt).toBeTruthy()
    expect(userAgreement.updatedAt).toBeTruthy()
  })

  it('contains at least one section in each document', () => {
    expect(privacyPolicy.sections.length).toBeGreaterThan(0)
    expect(userAgreement.sections.length).toBeGreaterThan(0)
  })
})
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/legal-content.test.ts
```

Expected: FAIL until the legal content modules and page renderers are ready.

**Step 3: Write minimal implementation**

- Build `/download` with:
  - iOS direct download card
  - Android direct download card
  - QR block for desktop users
  - short trust-building notes
- Build `/privacy` and `/agreement` using a shared `LegalArticle` component that renders sectioned content from `data/legal.ts`.
- Build `/contact` with simple contact info and business collaboration copy.
- Set page-specific SEO metadata on every page.
- Keep actual iOS/Android URLs and contact details behind a single config module so production values can be filled safely.

**Step 4: Run tests and a build**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/legal-content.test.ts
npm run build
```

Expected: PASS for the test and a successful build with the new routes.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/website/pages /Users/wisesearch/Projects/ChasingPoints/website/components /Users/wisesearch/Projects/ChasingPoints/website/public /Users/wisesearch/Projects/ChasingPoints/website/tests/legal-content.test.ts
git commit -m "feat: add website secondary pages"
```

### Task 5: Finish SEO assets, verification, and repo documentation

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/public/robots.txt`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/public/sitemap.xml`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/website/nuxt.config.ts`
- Create: `/Users/wisesearch/Projects/ChasingPoints/website/README.md`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/AGENTS.md`

**Step 1: Write the failing test**

Create `/Users/wisesearch/Projects/ChasingPoints/website/tests/seo-assets.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('seo assets', () => {
  it('includes the download route in sitemap.xml', () => {
    const sitemap = readFileSync(new URL('../public/sitemap.xml', import.meta.url), 'utf8')
    expect(sitemap).toContain('/download')
  })
})
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test -- tests/seo-assets.test.ts
```

Expected: FAIL because the SEO assets do not exist yet.

**Step 3: Write minimal implementation**

- Add `robots.txt` and a static `sitemap.xml` covering:
  - `/`
  - `/download`
  - `/privacy`
  - `/agreement`
  - `/contact`
- Ensure `nuxt.config.ts` sets a final `siteUrl` value from runtime config or environment variables.
- Add `website/README.md` documenting:
  - install
  - local dev
  - build
  - where to edit download links
  - where to update legal content
- Update root `AGENTS.md` so the repository map includes `website/` and points future contributors to the new subproject.

**Step 4: Run final verification**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test
npm run build
```

Then manually verify in local preview:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run dev
```

Expected:

- all Vitest suites PASS
- Nuxt build succeeds
- homepage, download page, privacy page, agreement page, and contact page render correctly on desktop and mobile widths

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/website /Users/wisesearch/Projects/ChasingPoints/AGENTS.md
git commit -m "docs: finalize website setup and seo assets"
```

### Task 6: Release checklist before deployment

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/website/data/download.ts`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/website/data/contact.ts`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/website/public/og-cover.jpg`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/website/public/qr-ios.png`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/website/public/qr-android.png`

**Step 1: Fill production content**

Replace placeholders with:

- real iOS download URL
- real Android download URL
- final business contact info
- final OG share image
- final QR images

**Step 2: Re-run final verification**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/website
npm run test
npm run build
```

Expected: PASS.

**Step 3: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/website/data /Users/wisesearch/Projects/ChasingPoints/website/public
git commit -m "chore: fill website production content"
```
