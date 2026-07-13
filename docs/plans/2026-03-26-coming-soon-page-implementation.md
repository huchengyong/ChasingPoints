# Coming Soon Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a lightweight `/coming-soon` page to the `website/` project for platforms whose download entry is not yet open.

**Architecture:** Reuse the existing website shell, page hero, footer, and SEO helper so the page feels like a first-class part of the official site instead of an error page. Keep the content static and intentionally small: one short message plus buttons back to the homepage and download page.

**Tech Stack:** Nuxt 3, Vue 3, TypeScript, existing website components and Vitest.

---

### Task 1: Add the fixed coming-soon route

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website/pages/coming-soon.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website/tests/coming-soon-content.test.ts`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website/data/site.ts`

**Step 1: Write the failing test**

Create a test that verifies:

- the site nav includes `/coming-soon`
- the page copy includes `敬请期待`
- the page presents links back to `/` and `/download`

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website
npm run test -- tests/coming-soon-content.test.ts
```

Expected: FAIL because the route and related content do not exist yet.

**Step 3: Write minimal implementation**

- Add `coming-soon.vue` using existing `SiteHeader`, `PageHero`, `SiteFooter`, and `usePageSeo`.
- Keep content within one viewport and avoid adding countdowns, forms, or extra marketing sections.
- Add a nav entry or footer entry only if needed for internal consistency.

**Step 4: Run tests and build**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website
npm run test -- tests/coming-soon-content.test.ts
npm run build
```

Expected: PASS and the new route is prerendered successfully.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website/pages/coming-soon.vue /Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website/tests/coming-soon-content.test.ts /Users/wisesearch/Projects/ChasingPoints/.worktrees/website/website/data/site.ts
git commit -m "feat: add website coming soon page"
```
