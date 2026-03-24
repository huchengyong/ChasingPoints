# Brand Theme Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align the app visual identity with the new yellow-white logo by making gold the brand color, demoting green to a semantic status color, and progressively removing hardcoded green usage from the user-facing app.

**Architecture:** Keep `app/theme.json` as the main theme token source, mirror those decisions into `app/App.vue` CSS variables and `app/store/theme.js` system UI colors, then refactor high-visibility pages away from hardcoded green values toward brand and semantic tokens. Roll out in phases so the app never sits in a half-broken mixed state longer than one task.

**Tech Stack:** UniApp Vue3, Pinia, SCSS, uni-app theme variables.

---

### Task 1: Define the new brand and semantic tokens

**Files:**
- Modify: `app/theme.json`
- Modify: `app/App.vue`
- Modify: `app/store/theme.js`
- Modify: `app/uni.scss`
- Reference: `app/pages/welcome/index.scss`

**Step 1: Snapshot the current color usage**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "#18b05b|#22c55e|#FFD700|#007aff" app
```

Expected: multiple matches across theme config, global variables, and page-level styles.

**Step 2: Update `app/theme.json` tokens**

Introduce the approved brand direction:

- set `primary` to the gold brand value for both light and dark themes
- keep green under semantic keys such as `success` or existing `iconGreen`
- update `tabActive`, `switchBgActive`, and brand-adjacent tokens to gold
- keep `danger`, blue info, and neutral tokens intact unless they clash with the new warm palette

**Step 3: Mirror the same decisions into global CSS variables**

Update `app/App.vue`:

- `--primary-color`
- `--primary-color-light`
- neutral background and text variables

Make the dark theme warm-neutral instead of green-neutral.

**Step 4: Sync system chrome colors**

Update `app/store/theme.js` so:

- navigation bar background matches the new warm surfaces
- selected tab color uses the brand gold
- inactive tab color remains readable in both themes

**Step 5: Update uni-app default theme variables**

Change `app/uni.scss`:

- `$uni-color-primary`
- keep success, warning, and error semantically distinct

**Step 6: Verify the core token layer**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "\"primary\"|tabActive|switchBgActive|welcomeButtonPrimary" app/theme.json app/App.vue app/store/theme.js app/uni.scss
```

Expected: theme entry points now consistently reflect the gold-led brand.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/theme.json /Users/wisesearch/Projects/ChasingPoints/app/App.vue /Users/wisesearch/Projects/ChasingPoints/app/store/theme.js /Users/wisesearch/Projects/ChasingPoints/app/uni.scss
git commit -m "refactor: establish gold-led brand theme tokens"
```

### Task 2: Align the welcome and login entry experience

**Files:**
- Modify: `app/pages/welcome/index.scss`
- Modify: `app/pages/login/login.vue`

**Step 1: Replace inconsistent green gradients and tags in login**

Update the login page gradients, outlines, and badge colors so:

- main action uses brand gold
- soft fills use brand tint
- any success or verified states remain green

**Step 2: Keep welcome page high-impact but same family**

Refine `app/pages/welcome/index.scss`:

- keep the bright yellow CTA if it still looks best
- make sure its hue stays within the same family as the new app brand gold
- keep text contrast strong on dark surfaces

**Step 3: Verify there is no green brand leakage in the entry flow**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "#18b05b|#22c55e|#166534" app/pages/welcome app/pages/login
```

Expected: any remaining green values are semantic-only, not primary CTA or brand gradients.

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/welcome/index.scss /Users/wisesearch/Projects/ChasingPoints/app/pages/login/login.vue
git commit -m "refactor: align welcome and login brand styling"
```

### Task 3: Refactor the highest-traffic user-facing pages

**Files:**
- Modify: `app/pages/index/index.scss`
- Modify: `app/pages/index/index.vue`
- Modify: `app/pages/user/index.scss`
- Modify: `app/pages/user/index.vue`
- Modify: `app/pages/match/index.scss`
- Modify: `app/pages/match/index.vue`
- Modify: `app/pages/ranking/index.scss`

**Step 1: Replace hardcoded brand green with theme-driven values**

On these pages, convert:

- main buttons
- selected chips
- hero cards
- spinners used as primary-page accent
- rank highlight blocks

from direct green hex values to the new gold-led theme values or CSS variables.

**Step 2: Preserve semantic green where it communicates status**

Do not recolor:

- “in progress”
- positive outcome
- online / active indicators
- success confirmations

unless the current use is clearly branding rather than status.

**Step 3: Verify the main page set is cleaned up**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "#18b05b|#22c55e" app/pages/index app/pages/user app/pages/match app/pages/ranking
```

Expected: remaining matches are deliberate semantic green only.

**Step 4: Manual smoke-check the main user flow**

Check in the uni-app preview runtime:

- welcome
- login
- home
- match list
- ranking
- user center

Expected: brand gold is the dominant accent, green reads as status, and both light/dark modes remain readable.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/index /Users/wisesearch/Projects/ChasingPoints/app/pages/user /Users/wisesearch/Projects/ChasingPoints/app/pages/match /Users/wisesearch/Projects/ChasingPoints/app/pages/ranking
git commit -m "refactor: apply new brand theme to core app pages"
```

### Task 4: Clean up shared components and high-visibility subpages

**Files:**
- Modify: `app/components/gameTypeModal.vue`
- Modify: `app/components/bindPhone.vue`
- Modify: `app/subPages/season/index.vue`
- Modify: `app/subPages/season/report.vue`
- Modify: `app/subPages/notification/index.vue`
- Modify: `app/subPages/venue/index.vue`
- Modify: `app/subPages/tournament/index.vue`

**Step 1: Move component-level primary color constants to the new brand**

Replace local `$primary-color: #18b05b;` declarations with the approved gold brand equivalent or shared variables where available.

**Step 2: Keep charts, progress, and live-state indicators semantically honest**

If a progress bar or live badge currently uses green to signal success or activity, leave it green. Only convert elements that are acting as brand accents.

**Step 3: Verify shared surfaces**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "#18b05b|#22c55e" app/components app/subPages/season app/subPages/notification app/subPages/venue app/subPages/tournament
```

Expected: remaining green usage is intentionally semantic.

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/components /Users/wisesearch/Projects/ChasingPoints/app/subPages/season /Users/wisesearch/Projects/ChasingPoints/app/subPages/notification /Users/wisesearch/Projects/ChasingPoints/app/subPages/venue /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament
git commit -m "refactor: update shared components for new brand palette"
```

### Task 5: Run a final regression sweep and leave a color audit trail

**Files:**
- Modify: `docs/plans/2026-03-24-brand-theme-design.md`
- Modify: `docs/plans/2026-03-24-brand-theme-implementation.md`

**Step 1: Run a final hardcoded color audit**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "#18b05b|#22c55e|#FFD700|#007aff" app
```

Expected: only approved, deliberate uses remain.

**Step 2: Record exceptions**

If any old values remain intentionally:

- note which files keep them
- explain whether they are semantic status colors, legacy plugin constraints, or deferred work

Add that note to the two plan docs so the theme refactor has a clear stopping point.

**Step 3: Manual visual QA**

Check both light and dark mode for:

- text contrast
- button contrast
- tab visibility
- disabled states
- warning / danger / success distinction

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-brand-theme-design.md /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-brand-theme-implementation.md
git commit -m "docs: capture brand theme rollout notes"
```
