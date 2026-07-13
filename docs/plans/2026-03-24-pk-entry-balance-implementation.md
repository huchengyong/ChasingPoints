# PK Entry Balance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make PK initiation and PK acceptance equally discoverable on the user homepage by pairing `发起PK` with a new visible `出示PK码` action that opens the existing QR presentation flow.

**Architecture:** Keep the existing match start and QR-code generation logic intact, but rebalance the user homepage action group so the “scan someone else” and “let someone scan me” paths are exposed side by side. Reuse the current QR modal implementation, only adjusting copy and trigger placement so the behavior change stays low risk.

**Tech Stack:** UniApp Vue3, Composition API, scoped SCSS, existing user homepage action/state logic.

---

### Task 1: Add a testable PK action-group helper

**Files:**
- Create: `app/utils/pk-entry-actions.js`
- Create: `app/tests/pk-entry-actions.test.mjs`
- Reference: `app/pages/user/index.vue`

**Step 1: Write the failing test**

Add `app/tests/pk-entry-actions.test.mjs` with minimal assertions for a pure helper like `resolvePkEntryActions`.

Cover:

- logged-in state exposes both `发起PK` and `出示PK码`
- guest state still keeps login-first behavior for PK actions if needed
- QR modal copy resolves to the new wording

Example assertions:

```js
assert.deepEqual(resolvePkEntryActions({ isLoggedIn: true }), {
  primaryText: '发起PK',
  secondaryText: '出示PK码'
})
assert.equal(resolveQrCodeModalCopy().title, '出示PK码')
```

**Step 2: Run the test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
node --test app/tests/pk-entry-actions.test.mjs
```

Expected: FAIL because the new helper file does not exist yet.

**Step 3: Write the minimal implementation**

Create `app/utils/pk-entry-actions.js` with small pure helpers such as:

- `resolvePkEntryActions({ isLoggedIn })`
- `resolveQrCodeModalCopy()`

Keep them framework-free.

**Step 4: Run the test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
node --test app/tests/pk-entry-actions.test.mjs
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/utils/pk-entry-actions.js /Users/wisesearch/Projects/ChasingPoints/app/tests/pk-entry-actions.test.mjs
git commit -m "test: add pk entry action helpers"
```

### Task 2: Expose `出示PK码` beside the current PK CTA

**Files:**
- Modify: `app/pages/user/index.vue`
- Reference: `app/utils/pk-entry-actions.js`

**Step 1: Update the status card action structure**

Refactor the logged-in action area so the PK action group includes:

- primary action `发起PK`
- secondary action `出示PK码`

Do not remove the current `发起PK` flow.

**Step 2: Reuse the current QR modal trigger**

Wire the new `出示PK码` action to the same modal opening behavior as the current top-right QR icon trigger.

This should call the existing QR modal logic instead of creating a parallel flow.

**Step 3: Keep the legacy icon entry**

Leave the top-right QR icon in place for now so existing users do not lose their learned path.

**Step 4: Run a focused structure check**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "出示PK码|handleQrCode|handleStartPK" app/pages/user/index.vue
```

Expected: the page now exposes both the scan flow and the QR-display flow in visible action wiring.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/user/index.vue
git commit -m "refactor: expose visible pk acceptance action"
```

### Task 3: Refresh the action-group and QR modal copy

**Files:**
- Modify: `app/pages/user/index.vue`
- Modify: `app/pages/user/index.scss`
- Reference: `app/utils/pk-entry-actions.js`

**Step 1: Update QR modal text**

Change the current QR modal copy to:

- title: `出示PK码`
- primary hint: `让对手打开“发起PK”后扫描此码，即可开始匹配`

Optionally add a secondary weak hint if the layout supports it cleanly.

**Step 2: Style the new secondary button**

Update the action-group styles so:

- `发起PK` remains the dominant CTA
- `出示PK码` reads as clearly available but secondary
- the pair feels like one coordinated group rather than two unrelated buttons

**Step 3: Check layout integrity**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "qrcode-modal-title|qrcode-hint|status-btn" app/pages/user/index.vue app/pages/user/index.scss
```

Expected: the QR modal wording and action-group styling now reflect the new IA.

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/user/index.vue /Users/wisesearch/Projects/ChasingPoints/app/pages/user/index.scss
git commit -m "style: balance pk action group and qr guidance"
```

### Task 4: Run regression verification and capture rollout notes

**Files:**
- Modify: `docs/plans/2026-03-24-pk-entry-balance-design.md`
- Modify: `docs/plans/2026-03-24-pk-entry-balance-implementation.md`

**Step 1: Run targeted tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
node --test app/tests/pk-entry-actions.test.mjs app/tests/home-index.test.mjs
```

Expected: PASS.

**Step 2: Run a quick behavior audit**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
git diff --stat
rg -n "出示PK码|我的二维码|出示PK码" app/pages/user/index.vue app/pages/user/index.scss
```

Expected: only the user homepage and new helper/test files changed for this feature.

**Step 3: Record any rollout note**

If any compromise remains, document it here, for example:

- legacy QR icon intentionally retained as a transition path
- only the user homepage is updated in this iteration

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-pk-entry-balance-design.md /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-pk-entry-balance-implementation.md
git commit -m "docs: capture pk entry balance plan"
```
