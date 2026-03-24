# Login Funnel Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Increase first-time login conversion and improve visual quality by turning the welcome page and login page into one clear, brand-consistent entry funnel.

**Architecture:** Keep the existing welcome-to-login route structure, but refactor both pages around a single `手机号登录 / 注册` path, gated third-party login visibility, and inline validation states. Extract any reusable funnel rules into pure utility functions so copy, availability, and validation logic can be tested without spinning up the full uni-app runtime.

**Tech Stack:** UniApp Vue3, Pinia, scoped SCSS, Node `node:test` tests for pure utilities.

---

### Task 1: Add a testable entry-funnel rules module

**Files:**
- Create: `app/utils/entry-funnel.js`
- Create: `app/tests/entry-funnel.test.mjs`
- Reference: `app/pages/welcome/index.vue`
- Reference: `app/pages/login/login.vue`

**Step 1: Write the failing test**

Add `app/tests/entry-funnel.test.mjs` covering:

- `resolveWelcomeActions({ isHarmony })`
- `isPhoneValid(phone)`
- `isCodeValid(code)`
- `canRequestSms({ phone, isSending, countdown })`
- `canSubmitLogin({ phone, code, isAgreed, isLogging })`

Include concrete assertions such as:

```js
assert.equal(isPhoneValid('13800138000'), true)
assert.equal(isPhoneValid('123'), false)
assert.equal(isCodeValid('123456'), true)
assert.equal(canSubmitLogin({
  phone: '13800138000',
  code: '123456',
  isAgreed: true,
  isLogging: false
}), true)
assert.equal(resolveWelcomeActions({ isHarmony: false }).showHuaweiLogin, false)
```

**Step 2: Run the test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
node --test app/tests/entry-funnel.test.mjs
```

Expected: FAIL because `app/utils/entry-funnel.js` does not exist yet.

**Step 3: Write the minimal implementation**

Create `app/utils/entry-funnel.js` with small pure helpers:

- `isPhoneValid(phone)`
- `isCodeValid(code)`
- `canRequestSms({ phone, isSending, countdown })`
- `canSubmitLogin({ phone, code, isAgreed, isLogging })`
- `resolveWelcomeActions({ isHarmony })`

Keep them framework-free so both pages can reuse them.

**Step 4: Run the test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
node --test app/tests/entry-funnel.test.mjs
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/utils/entry-funnel.js /Users/wisesearch/Projects/ChasingPoints/app/tests/entry-funnel.test.mjs
git commit -m "test: add entry funnel interaction rules"
```

### Task 2: Refactor the welcome page into a single-path entry screen

**Files:**
- Modify: `app/pages/welcome/index.vue`
- Modify: `app/pages/welcome/index.scss`
- Reference: `app/utils/entry-funnel.js`

**Step 1: Update the welcome page structure**

Change the template so the page exposes:

- one primary CTA: `手机号登录 / 注册`
- one secondary CTA: `华为账号登录`
- one weak action: `先逛逛`

Remove the current duplicated “注册” and “登录” split.

**Step 2: Gate the Huawei login action by platform**

Use runtime or compile-time platform detection so the Huawei entry is only rendered when supported. Do not keep a visible action that just toasts “请在HarmonyOS中使用华为登录”.

**Step 3: Preserve agreement state across pages**

When the user agrees on the welcome page, store that state so the login page can initialize from it instead of forcing a second click.

**Step 4: Restyle the page around a bottom action panel**

Update `app/pages/welcome/index.scss` to implement:

- stronger brand hero copy
- a bottom floating action card
- clear hierarchy between primary, secondary, and tertiary actions
- brand-consistent gold accents

**Step 5: Run focused regression checks**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "注册|登录" app/pages/welcome/index.vue
```

Expected: no separate primary “注册” and “登录” CTA pair remains.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/welcome/index.vue /Users/wisesearch/Projects/ChasingPoints/app/pages/welcome/index.scss
git commit -m "refactor: simplify welcome page entry path"
```

### Task 3: Rebuild the login page around inline validation and a stronger primary path

**Files:**
- Modify: `app/pages/login/login.vue`
- Reference: `app/utils/entry-funnel.js`
- Reference: `app/api/auth.js`
- Reference: `app/store/user.js`

**Step 1: Add computed validation and disabled states**

Refactor the page to derive:

- phone validity
- code validity
- send-code enabled state
- submit enabled state
- inline error messaging

from the shared entry-funnel helpers instead of relying only on click-time toast checks.

**Step 2: Update login copy and information hierarchy**

Change the header copy to:

- title: `手机号登录 / 注册`
- subtitle: `未注册手机号验证后将自动创建账号`

Reduce decorative header weight so the form is visible earlier on small screens.

**Step 3: Move agreement handling next to the primary CTA**

Make agreement acceptance a visible precondition:

- place it directly above the primary button
- keep the button disabled until agreed
- preserve deep links to the user agreement and privacy policy

**Step 4: Gate third-party login visibility**

Only render Huawei login on supported platforms. If the button is not actionable on the current platform, it should not appear.

**Step 5: Improve success and error flow**

Adjust the page so:

- field problems use inline hints first
- API failures still use toast feedback
- successful login navigates immediately instead of waiting a fixed 1500ms

**Step 6: Run the targeted tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
node --test app/tests/entry-funnel.test.mjs app/tests/bind-phone-flow.test.mjs
```

Expected: PASS.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/login/login.vue
git commit -m "refactor: improve login page conversion flow"
```

### Task 4: Polish the shared visual system for the entry funnel

**Files:**
- Modify: `app/pages/welcome/index.scss`
- Modify: `app/pages/login/login.vue`
- Reference: `docs/plans/2026-03-24-brand-theme-design.md`

**Step 1: Align both pages to the same brand family**

Make sure both pages share:

- warm dark or warm neutral surfaces
- consistent brand gold CTA styling
- matching border, muted text, and secondary action treatments

**Step 2: Reduce visual competition on the login page**

Tune spacing and scale so:

- header does not push the form below the fold
- the send-code action reads as secondary
- the primary CTA stays dominant

**Step 3: Verify platform and state variants manually**

Check these states in the preview runtime:

- welcome page default state
- welcome page after agreement checked
- login page with invalid phone
- login page with valid phone and disabled submit
- login page during countdown
- login success path

Expected: the funnel feels consistent, readable, and faster than the original flow.

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/welcome/index.scss /Users/wisesearch/Projects/ChasingPoints/app/pages/login/login.vue
git commit -m "style: polish entry funnel visual hierarchy"
```

### Task 5: Record rollout notes and final verification

**Files:**
- Modify: `docs/plans/2026-03-24-login-funnel-refresh-design.md`
- Modify: `docs/plans/2026-03-24-login-funnel-refresh-implementation.md`

**Step 1: Run a final file audit**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
git diff --stat
node --test app/tests/entry-funnel.test.mjs app/tests/bind-phone-flow.test.mjs app/tests/home-index.test.mjs
```

Expected: relevant files only, tests PASS.

**Step 2: Add any implementation notes**

If the implementation keeps any temporary compromise, document it here, for example:

- shared agreement state stored locally for now rather than in Pinia
- Huawei login gated by runtime capability only

**Step 3: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-login-funnel-refresh-design.md /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-login-funnel-refresh-implementation.md
git commit -m "docs: capture login funnel refresh plan"
```
